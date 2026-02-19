// Package policy provides the Policy Analyzer agent.
package policy

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/analyzer"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/llm"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/parser"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// Agent performs policy analysis on IaC resources.
type Agent struct {
	rules     []analyzer.Rule
	llmClient *llm.Client
	enableLLM bool
}

// New creates a new policy Agent.
func New(opts ...Option) *Agent {
	a := &Agent{
		rules: analyzer.RulesByCategory("Policy"),
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Option configures a policy Agent.
type Option func(*Agent)

// WithLLM enables LLM-enhanced analysis.
func WithLLM(client *llm.Client) Option {
	return func(a *Agent) {
		a.llmClient = client
		a.enableLLM = client != nil
	}
}

func (a *Agent) ID() string { return "policy" }

func (a *Agent) Metadata() protocol.AgentMetadata {
	return protocol.AgentMetadata{
		ID:          "policy",
		Name:        "Policy Analyzer",
		Description: "Checks IaC resources against organizational policies (naming, tagging, SKU, encryption, access controls)",
		Version:     "1.0.0",
	}
}

func (a *Agent) Capabilities() protocol.AgentCapabilities {
	return protocol.AgentCapabilities{
		Formats:       []protocol.SourceFormat{protocol.FormatTerraform, protocol.FormatBicep},
		NeedsIaCInput: true,
	}
}

// Handle runs policy rules against parsed IaC resources.
func (a *Agent) Handle(ctx context.Context, req protocol.AgentRequest, emit protocol.Emitter) error {
	if !protocol.RequireIaC(req, emit, "policy") {
		return nil
	}

	var findings []protocol.Finding
	for _, res := range req.IaC.Resources {
		for _, rule := range a.rules {
			if !rule.Applies(res.Type) {
				continue
			}
			if msg := rule.Check(res.Properties); msg != "" {
				findings = append(findings, protocol.Finding{
					RuleID:       rule.ID,
					Severity:     rule.Severity,
					Resource:     res.Name,
					ResourceType: res.Type,
					Message:      msg,
					Remediation:  rule.Remediation,
				})
			}
		}
	}

	if len(findings) == 0 {
		emit.SendMessage("### Policy Analysis\n\nAll policy checks passed.\n")
	} else {
		emit.SendMessage("### Policy Analysis\n\n")
		emit.SendMessage("| Rule | Severity | Resource | Issue | Fix |\n")
		emit.SendMessage("|------|----------|----------|-------|-----|\n")
		for _, f := range findings {
			emit.SendMessage(fmt.Sprintf("| %s | %s | %s.%s | %s | %s |\n",
				f.RuleID, f.Severity, parser.ShortType(f.ResourceType), f.Resource,
				f.Message, f.Remediation))
		}
		emit.SendMessage("\n")
	}

	// LLM-enhanced summary
	if a.enableLLM && a.llmClient != nil && req.Token != "" {
		a.enhanceWithLLM(ctx, req, findings, emit)
	}

	return nil
}

const policyPrompt = `You are a senior cloud policy engineer. Given the IaC code and deterministic policy findings below, provide:
1. A 2-3 sentence summary of the policy posture
2. Any additional policy concerns not caught by rules (naming conventions, tagging gaps, organizational standards)
3. Prioritized remediation steps

Be specific. Reference actual resource names. Use markdown. Keep it under 200 words.`

func (a *Agent) enhanceWithLLM(ctx context.Context, req protocol.AgentRequest, findings []protocol.Finding, emit protocol.Emitter) {
	var sb strings.Builder
	sb.WriteString("## IaC Code\n```\n")
	if req.IaC != nil {
		sb.WriteString(req.IaC.RawCode)
	}
	sb.WriteString("\n```\n\n## Policy Findings\n")
	if len(findings) == 0 {
		sb.WriteString("No violations found.\n")
	} else {
		for _, f := range findings {
			sb.WriteString(fmt.Sprintf("- [%s] %s %s: %s\n", f.RuleID, f.Severity, f.Resource, f.Message))
		}
	}

	emit.SendMessage("\n#### AI Policy Insights\n\n")
	messages := []llm.ChatMessage{{Role: llm.RoleUser, Content: sb.String()}}
	contentCh, errCh := a.llmClient.Stream(ctx, req.Token, policyPrompt, messages)
	for content := range contentCh {
		emit.SendMessage(content)
	}
	if err := <-errCh; err != nil {
		emit.SendMessage(fmt.Sprintf("\n_LLM enhancement unavailable: %v_\n", err))
	}
	emit.SendMessage("\n\n")
}
