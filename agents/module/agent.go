// Package module provides a placeholder Module Validator agent.
package module

import (
	"context"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// Agent validates Terraform/Bicep module sources and structure.
type Agent struct{}

// New creates a new module Agent.
func New() *Agent { return &Agent{} }

func (a *Agent) ID() string { return "module" }

func (a *Agent) Metadata() protocol.AgentMetadata {
	return protocol.AgentMetadata{
		ID:          "module",
		Name:        "Module Validator",
		Description: "Validates module sources, registry references, and module structure (placeholder)",
		Version:     "0.1.0",
	}
}

func (a *Agent) Capabilities() protocol.AgentCapabilities {
	return protocol.AgentCapabilities{
		Formats:       []protocol.SourceFormat{protocol.FormatTerraform, protocol.FormatBicep},
		NeedsIaCInput: true,
	}
}

// Handle is a placeholder that reports module validation is not yet implemented.
func (a *Agent) Handle(_ context.Context, _ protocol.AgentRequest, emit protocol.Emitter) error {
	emit.SendMessage("## Module Validator\n\n")
	emit.SendMessage("Module validation is not yet implemented. Future capabilities:\n")
	emit.SendMessage("- Validate module sources against approved registries\n")
	emit.SendMessage("- Check module version pinning\n")
	emit.SendMessage("- Scan module contents for policy violations\n")
	return nil
}
