// Command server is the main entry point for the GitHub Copilot Extension
// for IaC governance. It wires together the config, LLM client, router,
// and all analyzers into a single HTTP server.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/analyzer"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/config"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/costestimator"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/infraops"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/llm"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/router"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/server"
)

// Build-time variables set via -ldflags.
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	cfg := config.Load()

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	log.Printf("Starting ghcp-iac server version=%s commit=%s env=%s", version, commit, cfg.Environment)

	// LLM client
	var llmClient *llm.Client
	if cfg.EnableLLM {
		llmClient = llm.NewClient(cfg.ModelEndpoint, cfg.ModelName, cfg.ModelMaxTokens, cfg.ModelTimeout)
		log.Printf("LLM enabled: model=%s endpoint=%s", cfg.ModelName, cfg.ModelEndpoint)
	} else {
		log.Println("LLM disabled — using deterministic rules only")
	}

	// Build handler
	handler := &agentHandler{
		analyzer:  analyzer.New(llmClient, cfg.EnableLLM),
		estimator: costestimator.New(llmClient, cfg.EnableLLM, cfg.EnableCostAPI),
		ops: infraops.New(llmClient, cfg.EnableLLM, infraops.Config{
			TeamsWebhookURL: cfg.TeamsWebhookURL,
			SlackWebhookURL: cfg.SlackWebhookURL,
			EnableNotify:    cfg.EnableNotifications,
		}),
		router: router.New(llmClient, cfg.EnableLLM),
		config: cfg,
	}

	// Start server
	srv := server.New(cfg, handler)
	log.Printf("Listening on :%s", cfg.Port)
	if err := srv.Run(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

type agentHandler struct {
	analyzer  *analyzer.Analyzer
	estimator *costestimator.Estimator
	ops       *infraops.Ops
	router    *router.Router
	config    *config.Config
}

// HandleAgent implements server.Handler.
func (h *agentHandler) HandleAgent(ctx context.Context, req server.AgentRequest, sse *server.SSEWriter) {
	// Route to appropriate handler
	token := server.GitHubToken(ctx)
	intent := h.router.Classify(ctx, token, req.GetLastUserMessage())

	log.Printf("request_id=%s intent=%s message_preview=%q",
		server.RequestID(ctx), intent,
		truncate(req.GetLastUserMessage(), 80))

	switch intent {
	case router.IntentAnalyze:
		h.analyzer.Analyze(ctx, req, sse)
	case router.IntentCost:
		h.estimator.Estimate(ctx, req, sse)
	case router.IntentOps:
		h.ops.Handle(ctx, req, sse)
	case router.IntentStatus:
		h.handleStatus(sse)
	case router.IntentHelp:
		h.handleHelp(sse)
	default:
		h.handleHelp(sse)
	}
}

func (h *agentHandler) handleStatus(sse *server.SSEWriter) {
	sse.SendMessage("## IaC Governance Agent Status\n\n")
	sse.SendMessage(fmt.Sprintf("**Version:** %s\n", version))
	sse.SendMessage(fmt.Sprintf("**Commit:** %s\n", commit))
	sse.SendMessage(fmt.Sprintf("**Build:** %s\n", buildTime))
	sse.SendMessage(fmt.Sprintf("**Environment:** %s\n\n", h.config.Environment))

	features := []struct {
		name    string
		enabled bool
	}{
		{"LLM Enhancement", h.config.EnableLLM},
		{"Azure Cost API", h.config.EnableCostAPI},
		{"Notifications", h.config.EnableNotifications},
	}

	sse.SendMessage("| Feature | Status |\n|---------|--------|\n")
	for _, f := range features {
		status := "enabled"
		if !f.enabled {
			status = "disabled"
		}
		sse.SendMessage(fmt.Sprintf("| %s | %s |\n", f.name, status))
	}
}

func (h *agentHandler) handleHelp(sse *server.SSEWriter) {
	sse.SendMessage("# IaC Governance Agent\n\n")
	sse.SendMessage("I help you manage Azure infrastructure as code.\n\n")

	sse.SendMessage("## Capabilities\n\n")
	sse.SendMessage("### Analyze\n")
	sse.SendMessage("Paste Terraform or Bicep code for security, policy, and compliance analysis.\n")
	sse.SendMessage("*Examples: \"check this terraform\", \"scan for security issues\", \"audit compliance\"*\n\n")

	sse.SendMessage("### Cost Estimate\n")
	sse.SendMessage("Get monthly cost breakdown for your infrastructure.\n")
	sse.SendMessage("*Examples: \"estimate cost\", \"how much will this cost\", \"pricing\"*\n\n")

	sse.SendMessage("### Infrastructure Ops\n")
	sse.SendMessage("Manage deployments, detect drift, send notifications.\n")
	sse.SendMessage("*Examples: \"deploy to staging\", \"check drift\", \"environment status\"*\n\n")

	sse.SendMessage("### Status\n")
	sse.SendMessage("Check agent version and feature status.\n")
	sse.SendMessage("*Example: \"status\"*\n\n")

	hostname, _ := os.Hostname()
	sse.SendMessage(fmt.Sprintf("---\n*ghcp-iac v%s | %s | %s*\n", version, h.config.Environment, hostname))
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
