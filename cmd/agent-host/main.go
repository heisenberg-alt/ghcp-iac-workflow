// Command agent-host is the new entry point for the IaC governance agent.
// It registers all agents, sets up the orchestrator as the default handler,
// and supports both HTTP (SSE) and MCP stdio transports.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ghcp-iac/ghcp-iac-workflow/agents/compliance"
	"github.com/ghcp-iac/ghcp-iac-workflow/agents/cost"
	"github.com/ghcp-iac/ghcp-iac-workflow/agents/deploy"
	"github.com/ghcp-iac/ghcp-iac-workflow/agents/drift"
	"github.com/ghcp-iac/ghcp-iac-workflow/agents/impact"
	"github.com/ghcp-iac/ghcp-iac-workflow/agents/module"
	"github.com/ghcp-iac/ghcp-iac-workflow/agents/notification"
	"github.com/ghcp-iac/ghcp-iac-workflow/agents/orchestrator"
	"github.com/ghcp-iac/ghcp-iac-workflow/agents/policy"
	"github.com/ghcp-iac/ghcp-iac-workflow/agents/security"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/auth"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/config"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/host"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/llm"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/server"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/transport/mcpstdio"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	transport := flag.String("transport", "http", "Transport mode: http or stdio")
	flag.Parse()

	cfg := config.Load()

	// Create LLM client if enabled
	var llmClient *llm.Client
	if cfg.EnableLLM {
		llmClient = llm.NewClient(cfg.ModelEndpoint, cfg.ModelName, cfg.ModelMaxTokens, cfg.ModelTimeout)
		log.Printf("LLM enabled: model=%s endpoint=%s", cfg.ModelName, cfg.ModelEndpoint)
	}

	// Build registry
	registry := host.NewRegistry()

	registry.Register(policy.New(policy.WithLLM(llmClient)))
	registry.Register(security.New(security.WithLLM(llmClient)))
	registry.Register(compliance.New(compliance.WithLLM(llmClient)))
	registry.Register(cost.New(cost.WithLLM(llmClient)))
	registry.Register(drift.New())
	registry.Register(deploy.New())
	registry.Register(notification.New(cfg.EnableNotifications))
	registry.Register(impact.New(impact.WithLLM(llmClient)))
	registry.Register(module.New())

	// Orchestrator uses registry lookup
	orch := orchestrator.New(func(id string) (protocol.Agent, bool) {
		return registry.Get(id)
	}, orchestrator.WithLLM(llmClient))
	registry.Register(orch)

	dispatcher := host.NewDispatcher(registry)
	dispatcher.SetDefault("orchestrator")

	log.Printf("Registered %d agents, transport=%s", len(registry.List()), *transport)

	switch *transport {
	case "stdio":
		runStdio(registry, dispatcher)
	default:
		runHTTP(cfg, registry, dispatcher)
	}
}

func runHTTP(cfg *config.Config, registry *host.Registry, dispatcher *host.Dispatcher) {
	mux := http.NewServeMux()

	// Agent endpoint — uses orchestrator as default
	mux.HandleFunc("POST /agent", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxBodySize)
		var req server.AgentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		sse := server.NewSSEWriter(w)
		if sse == nil {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		agentReq := protocol.AgentRequest{
			Messages: make([]protocol.Message, len(req.Messages)),
			Token:    r.Header.Get("X-GitHub-Token"),
		}
		for i, m := range req.Messages {
			agentReq.Messages[i] = protocol.Message{Role: m.Role, Content: m.Content}
		}
		host.ParseAndEnrich(&agentReq)

		// Add timeout for agent dispatch
		ctx, cancel := context.WithTimeout(r.Context(), cfg.AgentTimeout)
		defer cancel()

		if err := dispatcher.Dispatch(ctx, "", agentReq, sse); err != nil {
			sse.SendError(err.Error())
		}
		sse.SendDone()
	})

	// Specific agent endpoint
	mux.HandleFunc("POST /agent/{id}", func(w http.ResponseWriter, r *http.Request) {
		agentID := r.PathValue("id")
		r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxBodySize)

		var req server.AgentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		sse := server.NewSSEWriter(w)
		if sse == nil {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		agentReq := protocol.AgentRequest{
			Messages: make([]protocol.Message, len(req.Messages)),
			Token:    r.Header.Get("X-GitHub-Token"),
		}
		for i, m := range req.Messages {
			agentReq.Messages[i] = protocol.Message{Role: m.Role, Content: m.Content}
		}
		host.ParseAndEnrich(&agentReq)

		// Add timeout for agent dispatch
		ctx, cancel := context.WithTimeout(r.Context(), cfg.AgentTimeout)
		defer cancel()

		if err := dispatcher.Dispatch(ctx, agentID, agentReq, sse); err != nil {
			sse.SendError(err.Error())
		}
		sse.SendDone()
	})

	// Agent listing
	mux.HandleFunc("GET /agents", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(registry.List())
	})

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "ok",
			"service":     "ghcp-iac-agent-host",
			"version":     version,
			"environment": cfg.Environment,
			"agents":      len(registry.List()),
		})
	})

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	// Wrap with signature verification middleware
	var handler http.Handler = mux
	handler = auth.Middleware(cfg.WebhookSecret, cfg.IsDev())(handler)

	// Configure server with timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("agent-host listening on :%s (version=%s commit=%s)", port, version, commit)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func runStdio(registry *host.Registry, dispatcher *host.Dispatcher) {
	log.SetOutput(os.Stderr) // Keep logs on stderr, stdout is for MCP
	log.Println("Starting MCP stdio transport")
	adapter := mcpstdio.NewAdapter(registry, dispatcher, os.Stdin, os.Stdout)
	if err := adapter.Run(context.Background()); err != nil {
		log.Fatalf("MCP stdio error: %v", err)
	}
}
