// Package impact provides the Blast Radius / Impact Analysis agent.
package impact

import (
	"context"
	"fmt"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/parser"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// Agent calculates the blast radius and risk weight of IaC resources.
type Agent struct{}

// New creates a new impact Agent.
func New() *Agent { return &Agent{} }

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
func (a *Agent) Handle(_ context.Context, req protocol.AgentRequest, emit protocol.Emitter) error {
	if req.IaC == nil || len(req.IaC.Resources) == 0 {
		emit.SendMessage("No IaC resources provided for impact analysis.\n")
		return nil
	}

	emit.SendMessage("## Blast Radius\n\n")

	total := 0
	for _, res := range req.IaC.Resources {
		weight := resourceRiskWeight(res.Type)
		total += weight
		emit.SendMessage(fmt.Sprintf("- **%s.%s** — risk weight: %d\n", parser.ShortType(res.Type), res.Name, weight))
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
	return nil
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
