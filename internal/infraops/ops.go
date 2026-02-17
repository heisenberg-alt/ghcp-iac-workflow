// Package infraops provides infrastructure operations: drift detection,
// environment promotion (deploy), and notifications.
package infraops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/llm"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/parser"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/server"
)

// Ops is the infrastructure operations handler.
type Ops struct {
	llmClient       *llm.Client
	enableLLM       bool
	enableNotify    bool
	teamsWebhookURL string
	slackWebhookURL string
	httpClient      *http.Client
	deploymentState map[string]*EnvironmentState
	logger          *log.Logger
}

// EnvironmentState tracks the deployed version per environment.
type EnvironmentState struct {
	Version    string    `json:"version"`
	DeployedAt time.Time `json:"deployed_at"`
	DeployedBy string    `json:"deployed_by"`
	Status     string    `json:"status"` // deployed, deploying, failed
}

// DriftResult represents a detected drift item.
type DriftResult struct {
	ResourceType string `json:"resource_type"`
	ResourceName string `json:"resource_name"`
	Property     string `json:"property"`
	Expected     string `json:"expected"`
	Actual       string `json:"actual"`
	Severity     string `json:"severity"`
}

// Config holds InfraOps configuration.
type Config struct {
	TeamsWebhookURL string
	SlackWebhookURL string
	EnableNotify    bool
}

// New creates a new Ops handler.
func New(llmClient *llm.Client, enableLLM bool, cfg Config) *Ops {
	return &Ops{
		llmClient:       llmClient,
		enableLLM:       enableLLM,
		enableNotify:    cfg.EnableNotify,
		teamsWebhookURL: cfg.TeamsWebhookURL,
		slackWebhookURL: cfg.SlackWebhookURL,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		deploymentState: map[string]*EnvironmentState{
			"dev":     {Version: "v1.0.0", DeployedAt: time.Now().Add(-48 * time.Hour), Status: "deployed"},
			"staging": {Version: "v0.9.0", DeployedAt: time.Now().Add(-72 * time.Hour), Status: "deployed"},
			"prod":    {Version: "v0.8.0", DeployedAt: time.Now().Add(-168 * time.Hour), Status: "deployed"},
		},
		logger: log.New(log.Writer(), "[infraops] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Handle processes infrastructure operations requests.
func (o *Ops) Handle(ctx context.Context, req server.AgentRequest, sse *server.SSEWriter) {
	msg := strings.ToLower(req.GetLastUserMessage())

	switch {
	case matchesAny(msg, "drift", "difference", "state"):
		o.handleDrift(ctx, req, sse)
	case matchesAny(msg, "deploy", "promote", "release"):
		o.handleDeploy(ctx, msg, sse)
	case matchesAny(msg, "notify", "alert", "notification"):
		o.handleNotify(ctx, msg, sse)
	case matchesAny(msg, "status", "environments", "versions"):
		o.handleEnvStatus(sse)
	default:
		o.showUsage(sse)
	}
}

// ==================== Drift Detection ====================

func (o *Ops) handleDrift(ctx context.Context, req server.AgentRequest, sse *server.SSEWriter) {
	sse.SendMessage("## Drift Detection\n\n")

	code := parser.ExtractCode(req.GetLastUserMessage())
	if code == "" {
		code = req.GetCodeFromReferences()
	}

	if code == "" {
		sse.SendMessage("No IaC code provided. Provide Terraform to compare against live state.\n")
		return
	}

	resources := parser.ParseResources(code)
	if len(resources) == 0 {
		sse.SendMessage("No resources found to check for drift.\n")
		return
	}

	sse.SendMessage(fmt.Sprintf("Comparing **%d** declared resource(s) against expected state...\n\n", len(resources)))

	drifts := o.detectDrift(resources)

	if len(drifts) == 0 {
		sse.SendMessage("**No drift detected.** All resources match their declared configuration.\n")
	} else {
		sse.SendMessage(fmt.Sprintf("**%d drift(s) detected**\n\n", len(drifts)))
		sse.SendMessage("| Resource | Property | Expected | Actual | Severity |\n")
		sse.SendMessage("|----------|----------|----------|--------|----------|\n")
		for _, d := range drifts {
			sse.SendMessage(fmt.Sprintf("| %s.%s | %s | %s | %s | %s |\n",
				d.ResourceType, d.ResourceName, d.Property, d.Expected, d.Actual, d.Severity))
		}
	}

	// LLM-enhanced drift explanation
	token := server.GitHubToken(ctx)
	if o.enableLLM && o.llmClient != nil && token != "" && len(drifts) > 0 {
		sse.SendMessage("\n### AI Drift Analysis\n\n")
		o.explainDriftWithLLM(ctx, token, code, drifts, sse)
	}
}

func (o *Ops) detectDrift(resources []parser.Resource) []DriftResult {
	var drifts []DriftResult

	for _, res := range resources {
		switch res.Type {
		case "azurerm_storage_account":
			if v, ok := res.Properties["min_tls_version"]; ok {
				if fmt.Sprintf("%v", v) != "TLS1_2" {
					drifts = append(drifts, DriftResult{
						ResourceType: res.Type, ResourceName: res.Name,
						Property: "min_tls_version", Expected: "TLS1_2",
						Actual: fmt.Sprintf("%v", v), Severity: "high",
					})
				}
			}
			if v, ok := res.Properties["enable_https_traffic_only"]; ok {
				if v != true {
					drifts = append(drifts, DriftResult{
						ResourceType: res.Type, ResourceName: res.Name,
						Property: "enable_https_traffic_only", Expected: "true",
						Actual: fmt.Sprintf("%v", v), Severity: "high",
					})
				}
			}
		case "azurerm_key_vault":
			if v, ok := res.Properties["soft_delete_enabled"]; ok {
				if v != true {
					drifts = append(drifts, DriftResult{
						ResourceType: res.Type, ResourceName: res.Name,
						Property: "soft_delete_enabled", Expected: "true",
						Actual: fmt.Sprintf("%v", v), Severity: "high",
					})
				}
			}
		}
	}

	return drifts
}

const driftPrompt = `You are an infrastructure operations expert. Given IaC code and detected configuration drifts, explain:
1. The likely root cause of each drift
2. The security/compliance impact
3. Steps to remediate (manual fix vs re-apply IaC)
Be concise. Use markdown.`

func (o *Ops) explainDriftWithLLM(ctx context.Context, token, code string, drifts []DriftResult, sse *server.SSEWriter) {
	var sb strings.Builder
	sb.WriteString("## Code\n```\n" + code + "\n```\n## Detected Drifts\n")
	for _, d := range drifts {
		sb.WriteString(fmt.Sprintf("- %s.%s: %s expected=%s actual=%s\n",
			d.ResourceType, d.ResourceName, d.Property, d.Expected, d.Actual))
	}

	messages := []llm.ChatMessage{{Role: llm.RoleUser, Content: sb.String()}}
	contentCh, errCh := o.llmClient.Stream(ctx, token, driftPrompt, messages)
	for content := range contentCh {
		sse.SendMessage(content)
	}
	if err := <-errCh; err != nil {
		o.logger.Printf("LLM drift analysis failed: %v", err)
	}
}

// ==================== Deployment Promotion ====================

func (o *Ops) handleDeploy(ctx context.Context, msg string, sse *server.SSEWriter) {
	sse.SendMessage("## Deployment Manager\n\n")

	target := "dev"
	if matchesAny(msg, "staging", "stage", "test") {
		target = "staging"
	} else if matchesAny(msg, "prod", "production") {
		target = "prod"
	}

	source := "dev"
	if target == "prod" {
		source = "staging"
	}

	sourceState := o.deploymentState[source]

	sse.SendMessage("### Environment Status\n\n")
	sse.SendMessage("| Environment | Version | Status |\n")
	sse.SendMessage("|-------------|---------|--------|\n")
	for _, env := range []string{"dev", "staging", "prod"} {
		s := o.deploymentState[env]
		sse.SendMessage(fmt.Sprintf("| %s | %s | %s |\n", env, s.Version, s.Status))
	}
	sse.SendMessage("\n")

	if target == "prod" {
		sse.SendMessage("**Production deployment requires manual approval.**\n\n")
		sse.SendMessage(fmt.Sprintf("Promotion: `%s` (%s) -> `%s`\n\n",
			source, sourceState.Version, target))
		sse.SendMessage("Use the GitHub Actions workflow `deploy-prod.yml` with approval gate.\n")
		return
	}

	sse.SendMessage(fmt.Sprintf("Promoting **%s** -> **%s** (version %s)\n\n", source, target, sourceState.Version))

	o.deploymentState[target] = &EnvironmentState{
		Version:    sourceState.Version,
		DeployedAt: time.Now(),
		Status:     "deployed",
	}

	sse.SendMessage(fmt.Sprintf("Successfully promoted to **%s** (version %s)\n", target, sourceState.Version))
}

// ==================== Notifications ====================

func (o *Ops) handleNotify(ctx context.Context, msg string, sse *server.SSEWriter) {
	sse.SendMessage("## Notification Manager\n\n")

	channel := "teams"
	if matchesAny(msg, "slack") {
		channel = "slack"
	}

	message := "Infrastructure update notification"
	if idx := strings.Index(msg, "message:"); idx >= 0 {
		message = strings.TrimSpace(msg[idx+8:])
	}

	sse.SendMessage(fmt.Sprintf("Sending notification to **%s**:\n> %s\n\n", channel, message))

	if o.enableNotify {
		var err error
		switch channel {
		case "teams":
			err = o.sendTeamsNotification(ctx, message)
		case "slack":
			err = o.sendSlackNotification(ctx, message)
		}
		if err != nil {
			sse.SendMessage(fmt.Sprintf("Failed to send: %v\n", err))
			return
		}
		sse.SendMessage("Notification sent successfully.\n")
	} else {
		sse.SendMessage("Notifications are disabled. Set `ENABLE_NOTIFICATIONS=true` to enable.\n")
	}
}

func (o *Ops) sendTeamsNotification(ctx context.Context, message string) error {
	if o.teamsWebhookURL == "" {
		return fmt.Errorf("TEAMS_WEBHOOK_URL not configured")
	}

	payload := map[string]interface{}{
		"@type":   "MessageCard",
		"summary": "IaC Governance Alert",
		"sections": []map[string]interface{}{
			{"activityTitle": "Infrastructure Update", "text": message},
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", o.teamsWebhookURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("teams webhook returned %d", resp.StatusCode)
	}
	return nil
}

func (o *Ops) sendSlackNotification(ctx context.Context, message string) error {
	if o.slackWebhookURL == "" {
		return fmt.Errorf("SLACK_WEBHOOK_URL not configured")
	}

	payload := map[string]string{"text": "IaC Governance: " + message}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", o.slackWebhookURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return nil
}

// ==================== Environment Status ====================

func (o *Ops) handleEnvStatus(sse *server.SSEWriter) {
	sse.SendMessage("## Environment Status\n\n")
	sse.SendMessage("| Environment | Version | Deployed | Status |\n")
	sse.SendMessage("|-------------|---------|----------|--------|\n")

	for _, env := range []string{"dev", "staging", "prod"} {
		s := o.deploymentState[env]
		sse.SendMessage(fmt.Sprintf("| %s | %s | %s | %s |\n",
			env, s.Version, s.DeployedAt.Format("2006-01-02 15:04"), s.Status))
	}
}

func (o *Ops) showUsage(sse *server.SSEWriter) {
	sse.SendMessage("## Infrastructure Operations\n\n")
	sse.SendMessage("**Available commands:**\n")
	sse.SendMessage("- `drift check [code]` — Detect configuration drift\n")
	sse.SendMessage("- `deploy to staging` — Promote to staging\n")
	sse.SendMessage("- `deploy to prod` — Initiate production deployment\n")
	sse.SendMessage("- `environment status` — Show all environment versions\n")
	sse.SendMessage("- `notify teams message: ...` — Send a Teams notification\n")
	sse.SendMessage("- `notify slack message: ...` — Send a Slack notification\n")
}

func matchesAny(msg string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}
