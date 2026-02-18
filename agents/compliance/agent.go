// Package compliance provides the Compliance Checker agent.
package compliance

import (
	"context"
	"fmt"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/analyzer"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/parser"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// Agent performs compliance checks on IaC resources.
type Agent struct {
	rules []analyzer.Rule
}

// New creates a new compliance Agent.
func New() *Agent {
	return &Agent{
		rules: analyzer.RulesByCategory("Compliance"),
	}
}

func (a *Agent) ID() string { return "compliance" }

func (a *Agent) Metadata() protocol.AgentMetadata {
	return protocol.AgentMetadata{
		ID:          "compliance",
		Name:        "Compliance Checker",
		Description: "Validates IaC against compliance frameworks (NIST, SOC2, CIS)",
		Version:     "1.0.0",
	}
}

func (a *Agent) Capabilities() protocol.AgentCapabilities {
	return protocol.AgentCapabilities{
		Formats:       []protocol.SourceFormat{protocol.FormatTerraform, protocol.FormatBicep},
		NeedsIaCInput: true,
	}
}

// Handle runs compliance rules against parsed IaC resources.
func (a *Agent) Handle(_ context.Context, req protocol.AgentRequest, emit protocol.Emitter) error {
	if req.IaC == nil || len(req.IaC.Resources) == 0 {
		emit.SendMessage("No IaC resources provided for compliance analysis.\n")
		return nil
	}

	var findings []finding
	for _, res := range req.IaC.Resources {
		for _, rule := range a.rules {
			if !rule.Applies(res.Type) {
				continue
			}
			if msg := rule.Check(res.Properties); msg != "" {
				findings = append(findings, finding{
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
		emit.SendMessage("### Compliance Analysis\n\nAll compliance checks passed.\n")
		return nil
	}

	emit.SendMessage("### Compliance Analysis\n\n")
	emit.SendMessage("| Rule | Severity | Resource | Issue | Fix |\n")
	emit.SendMessage("|------|----------|----------|-------|-----|\n")
	for _, f := range findings {
		emit.SendMessage(fmt.Sprintf("| %s | %s | %s.%s | %s | %s |\n",
			f.RuleID, f.Severity, parser.ShortType(f.ResourceType), f.Resource,
			f.Message, f.Remediation))
	}
	emit.SendMessage("\n")

	return nil
}

type finding struct {
	RuleID       string
	Severity     string
	Resource     string
	ResourceType string
	Message      string
	Remediation  string
}
