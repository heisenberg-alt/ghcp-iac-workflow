// Package orchestrator provides a placeholder Orchestrator agent.
// The full implementation comes in Phase 4.
package orchestrator

import (
	"context"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// Agent will coordinate multi-agent workflows. Placeholder for now.
type Agent struct{}

// New creates a new orchestrator Agent.
func New() *Agent { return &Agent{} }

func (a *Agent) ID() string { return "orchestrator" }

func (a *Agent) Metadata() protocol.AgentMetadata {
	return protocol.AgentMetadata{
		ID:          "orchestrator",
		Name:        "Orchestrator",
		Description: "Coordinates multi-agent analysis workflows (full implementation in Phase 4)",
		Version:     "0.1.0",
	}
}

func (a *Agent) Capabilities() protocol.AgentCapabilities {
	return protocol.AgentCapabilities{
		Formats:       []protocol.SourceFormat{protocol.FormatTerraform, protocol.FormatBicep},
		NeedsIaCInput: true,
	}
}

// Handle is a placeholder that will be replaced in Phase 4.
func (a *Agent) Handle(_ context.Context, _ protocol.AgentRequest, emit protocol.Emitter) error {
	emit.SendMessage("## Orchestrator\n\n")
	emit.SendMessage("Multi-agent orchestration is not yet wired. Individual agents are available:\n")
	emit.SendMessage("- `policy` — Policy analysis\n")
	emit.SendMessage("- `security` — Security scanning\n")
	emit.SendMessage("- `compliance` — Compliance checking\n")
	emit.SendMessage("- `cost` — Cost estimation\n")
	emit.SendMessage("- `drift` — Drift detection\n")
	emit.SendMessage("- `deploy` — Deployment management\n")
	emit.SendMessage("- `notification` — Notifications\n")
	emit.SendMessage("- `impact` — Blast radius analysis\n")
	emit.SendMessage("- `module` — Module validation\n")
	return nil
}
