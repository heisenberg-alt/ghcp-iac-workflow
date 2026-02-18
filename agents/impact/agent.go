// Package impact provides the Blast Radius / Impact Analysis agent.
package impact

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/llm"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/parser"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// Agent calculates the blast radius and risk weight of IaC resources.
type Agent struct {
	llmClient *llm.Client
	enableLLM bool
}

// New creates a new impact Agent.
func New(opts ...Option) *Agent {
	a := &Agent{}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Option configures an impact Agent.
type Option func(*Agent)

// WithLLM enables LLM-enhanced impact analysis.
func WithLLM(client *llm.Client) Option {
	return func(a *Agent) {
		a.llmClient = client
		a.enableLLM = client != nil
	}
}

func (a *Agent) ID() string { return "impact" }

func (a *Agent) Metadata() protocol.AgentMetadata {
	return protocol.AgentMetadata{
		ID:          "impact",
		Name:        "Impact Analyzer",
		Description: "Calculates blast radius and risk-weighted impact scores for IaC resources",
		Version:     "1.0.0",
	}
}

func (a *Agent) Capabilities() protocol.AgentCapabilities {
	return protocol.AgentCapabilities{
		Formats:       []protocol.SourceFormat{protocol.FormatTerraform, protocol.FormatBicep},
		NeedsIaCInput: true,
	}
}

// Handle computes blast radius for parsed resources.
func (a *Agent) Handle(ctx context.Context, req protocol.AgentRequest, emit protocol.Emitter) error {
	if req.IaC == nil || len(req.IaC.Resources) == 0 {
		emit.SendMessage("No IaC resources provided for impact analysis.\n")
		return nil
	}

	emit.SendMessage("## Blast Radius\n\n")

	total := 0
	var summary strings.Builder
	for _, res := range req.IaC.Resources {
		weight := resourceRiskWeight(res.Type)
		total += weight
		line := fmt.Sprintf("- **%s.%s** — risk weight: %d\n", parser.ShortType(res.Type), res.Name, weight)
		emit.SendMessage(line)
		summary.WriteString(line)
	}

	level := "Low"
	if total > 20 {
		level = "Critical"
	} else if total > 10 {
		level = "High"
	} else if total > 5 {
		level = "Medium"
	}

	emit.SendMessage(fmt.Sprintf("\n**Total blast radius: %d (%s)**\n", total, level))

	// LLM-enhanced blast radius explanation
	if a.enableLLM && a.llmClient != nil && req.Token != "" {
		a.enhanceWithLLM(ctx, req, summary.String(), total, level, emit)
	}

	return nil
}

const impactPrompt = `You are a senior cloud architect assessing infrastructure change risk. Given the IaC code and blast radius analysis below, provide:
1. A risk assessment explaining what could go wrong if these resources are modified or deleted
2. Dependency chain analysis — which resources depend on others
3. Rollback strategy recommendations

Be specific. Reference actual resource names. Use markdown. Keep it under 200 words.`

func (a *Agent) enhanceWithLLM(ctx context.Context, req protocol.AgentRequest, summary string, total int, level string, emit protocol.Emitter) {
	var sb strings.Builder
	sb.WriteString("## IaC Code\n```\n")
	if req.IaC != nil {
		sb.WriteString(req.IaC.RawCode)
	}
	sb.WriteString("\n```\n\n## Blast Radius\n")
	sb.WriteString(summary)
	sb.WriteString(fmt.Sprintf("\nTotal: %d (%s)\n", total, level))

	emit.SendMessage("\n#### AI Impact Assessment\n\n")
	messages := []llm.ChatMessage{{Role: llm.RoleUser, Content: sb.String()}}
	contentCh, errCh := a.llmClient.Stream(ctx, req.Token, impactPrompt, messages)
	for content := range contentCh {
		emit.SendMessage(content)
	}
	if err := <-errCh; err != nil {
		emit.SendMessage(fmt.Sprintf("\n_LLM enhancement unavailable: %v_\n", err))
	}
	emit.SendMessage("\n\n")
}

func resourceRiskWeight(resType string) int {
	weights := map[string]int{
		"azurerm_kubernetes_cluster":     8,
		"azurerm_virtual_machine":        5,
		"azurerm_linux_virtual_machine":  5,
		"azurerm_mssql_server":           7,
		"azurerm_mssql_database":         6,
		"azurerm_cosmosdb_account":       7,
		"azurerm_key_vault":              6,
		"azurerm_storage_account":        4,
		"azurerm_container_registry":     4,
		"azurerm_service_plan":           3,
		"azurerm_redis_cache":            5,
		"azurerm_virtual_network":        3,
		"azurerm_subnet":                 2,
		"azurerm_network_security_group": 4,
	}
	if w, ok := weights[resType]; ok {
		return w
	}
	return 2
}
