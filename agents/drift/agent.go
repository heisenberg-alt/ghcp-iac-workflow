// Package drift provides the Drift Detection agent.
package drift

import (
	"context"
	"fmt"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// Agent detects configuration drift in IaC resources.
type Agent struct{}

// New creates a new drift Agent.
func New() *Agent { return &Agent{} }

func (a *Agent) ID() string { return "drift" }

func (a *Agent) Metadata() protocol.AgentMetadata {
	return protocol.AgentMetadata{
		ID:          "drift",
		Name:        "Drift Detector",
		Description: "Compares IaC-declared configuration against expected state to detect drift",
		Version:     "1.0.0",
	}
}

func (a *Agent) Capabilities() protocol.AgentCapabilities {
	return protocol.AgentCapabilities{
		Formats:       []protocol.SourceFormat{protocol.FormatTerraform, protocol.FormatBicep},
		NeedsIaCInput: true,
	}
}

// Handle checks for configuration drift in parsed resources.
func (a *Agent) Handle(_ context.Context, req protocol.AgentRequest, emit protocol.Emitter) error {
	if !protocol.RequireIaC(req, emit, "drift detection") {
		return nil
	}

	emit.SendMessage("## Drift Detection\n\n")
	emit.SendMessage(fmt.Sprintf("Comparing **%d** declared resource(s) against expected state...\n\n", len(req.IaC.Resources)))

	var drifts []driftResult
	for _, res := range req.IaC.Resources {
		drifts = append(drifts, detectDrift(res)...)
	}

	if len(drifts) == 0 {
		emit.SendMessage("**No drift detected.** All resources match their declared configuration.\n")
		return nil
	}

	emit.SendMessage(fmt.Sprintf("**%d drift(s) detected**\n\n", len(drifts)))
	emit.SendMessage("| Resource | Property | Expected | Actual | Severity |\n")
	emit.SendMessage("|----------|----------|----------|--------|----------|\n")
	for _, d := range drifts {
		emit.SendMessage(fmt.Sprintf("| %s.%s | %s | %s | %s | %s |\n",
			d.ResourceType, d.ResourceName, d.Property, d.Expected, d.Actual, d.Severity))
	}

	return nil
}

type driftResult struct {
	ResourceType string
	ResourceName string
	Property     string
	Expected     string
	Actual       string
	Severity     string
}

func detectDrift(res protocol.Resource) []driftResult {
	var drifts []driftResult
	switch res.Type {
	case "azurerm_storage_account":
		if v, ok := res.Properties["min_tls_version"]; ok {
			if fmt.Sprintf("%v", v) != "TLS1_2" {
				drifts = append(drifts, driftResult{
					ResourceType: res.Type, ResourceName: res.Name,
					Property: "min_tls_version", Expected: "TLS1_2",
					Actual: fmt.Sprintf("%v", v), Severity: "high",
				})
			}
		}
		if v, ok := res.Properties["enable_https_traffic_only"]; ok {
			if v != true {
				drifts = append(drifts, driftResult{
					ResourceType: res.Type, ResourceName: res.Name,
					Property: "enable_https_traffic_only", Expected: "true",
					Actual: fmt.Sprintf("%v", v), Severity: "high",
				})
			}
		}
	case "azurerm_key_vault":
		if v, ok := res.Properties["soft_delete_enabled"]; ok {
			if v != true {
				drifts = append(drifts, driftResult{
					ResourceType: res.Type, ResourceName: res.Name,
					Property: "soft_delete_enabled", Expected: "true",
					Actual: fmt.Sprintf("%v", v), Severity: "high",
				})
			}
		}
	}
	return drifts
}
