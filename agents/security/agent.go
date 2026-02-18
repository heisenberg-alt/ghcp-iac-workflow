// Package security provides the Security Scanner agent.
package security

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/analyzer"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/llm"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/parser"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// Agent performs security analysis on IaC resources.
type Agent struct {
	rules     []analyzer.Rule
	llmClient *llm.Client
	enableLLM bool
}

// New creates a new security Agent.
func New(opts ...Option) *Agent {
	a := &Agent{
		rules: analyzer.RulesByCategory("Security"),
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Option configures a security Agent.
type Option func(*Agent)

// WithLLM enables LLM-enhanced analysis.
func WithLLM(client *llm.Client) Option {
	return func(a *Agent) {
		a.llmClient = client
		a.enableLLM = client != nil
	}
}

func (a *Agent) ID() string { return "security" }

func (a *Agent) Metadata() protocol.AgentMetadata {
	return protocol.AgentMetadata{
		ID:          "security",
		Name:        "Security Scanner",
		Description: "Scans IaC for security vulnerabilities including hardcoded secrets, public network access, encryption, and NSG rules",
		Version:     "1.0.0",
	}
}

func (a *Agent) Capabilities() protocol.AgentCapabilities {
	return protocol.AgentCapabilities{
		Formats:       []protocol.SourceFormat{protocol.FormatTerraform, protocol.FormatBicep},
		NeedsIaCInput: true,
		NeedsRawCode:  true,
	}
}

// Handle runs security rules against parsed IaC resources.
func (a *Agent) Handle(ctx context.Context, req protocol.AgentRequest, emit protocol.Emitter) error {
	if req.IaC == nil || len(req.IaC.Resources) == 0 {
		emit.SendMessage("No IaC resources provided for security analysis.\n")
		return nil
	}

	var findings []protocol.Finding
	for _, res := range req.IaC.Resources {
		for _, rule := range a.rules {
			if !rule.Applies(res.Type) {
				continue
			}
			// Pattern-based rules scan raw blocks
			if rule.IsPatternRule() {
				if violations := rule.CheckPatterns(res.RawBlock); len(violations) > 0 {
					for _, v := range violations {
						findings = append(findings, protocol.Finding{
							RuleID:       rule.ID,
							Severity:     rule.Severity,
							Resource:     res.Name,
							ResourceType: res.Type,
							Message:      v,
							Remediation:  rule.Remediation,
						})
					}
				}
				continue
			}
			// Property-based rules
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
		emit.SendMessage("### Security Analysis\n\nAll security checks passed.\n")
	} else {
		emit.SendMessage("### Security Analysis\n\n")
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

const securityPrompt = `You are a senior cloud security engineer. Given the IaC code and deterministic security findings below, provide:
1. A 2-3 sentence security posture assessment
2. Additional security risks not caught by rules (OWASP, CIS benchmarks, zero-trust gaps)
3. Prioritized remediation with specific Terraform/Bicep fixes

Be specific. Reference actual resource names and properties. Use markdown. Keep it under 250 words.`

func (a *Agent) enhanceWithLLM(ctx context.Context, req protocol.AgentRequest, findings []protocol.Finding, emit protocol.Emitter) {
	var sb strings.Builder
	sb.WriteString("## IaC Code\n```\n")
	if req.IaC != nil {
		sb.WriteString(req.IaC.RawCode)
	}
	sb.WriteString("\n```\n\n## Security Findings\n")
	if len(findings) == 0 {
		sb.WriteString("No vulnerabilities found.\n")
	} else {
		for _, f := range findings {
			sb.WriteString(fmt.Sprintf("- [%s] %s %s: %s\n", f.RuleID, f.Severity, f.Resource, f.Message))
		}
	}

	emit.SendMessage("\n#### AI Security Insights\n\n")
	messages := []llm.ChatMessage{{Role: llm.RoleUser, Content: sb.String()}}
	contentCh, errCh := a.llmClient.Stream(ctx, req.Token, securityPrompt, messages)
	for content := range contentCh {
		emit.SendMessage(content)
	}
	if err := <-errCh; err != nil {
		emit.SendMessage(fmt.Sprintf("\n_LLM enhancement unavailable: %v_\n", err))
	}
	emit.SendMessage("\n\n")
}
