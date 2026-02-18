// Package cost provides the Cost Estimator agent.
package cost

import (
	"context"
	"fmt"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/parser"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// Agent estimates monthly Azure costs for IaC resources.
type Agent struct{}

// New creates a new cost Agent.
func New() *Agent { return &Agent{} }

func (a *Agent) ID() string { return "cost" }

func (a *Agent) Metadata() protocol.AgentMetadata {
	return protocol.AgentMetadata{
		ID:          "cost",
		Name:        "Cost Estimator",
		Description: "Estimates monthly Azure costs for declared IaC resources using static pricing tables",
		Version:     "1.0.0",
	}
}

func (a *Agent) Capabilities() protocol.AgentCapabilities {
	return protocol.AgentCapabilities{
		Formats:       []protocol.SourceFormat{protocol.FormatTerraform, protocol.FormatBicep},
		NeedsIaCInput: true,
	}
}

// Handle estimates costs for parsed IaC resources.
func (a *Agent) Handle(_ context.Context, req protocol.AgentRequest, emit protocol.Emitter) error {
	if req.IaC == nil || len(req.IaC.Resources) == 0 {
		emit.SendMessage("No IaC resources provided for cost estimation.\n")
		return nil
	}

	var total float64
	type item struct {
		Name    string
		SKU     string
		Monthly float64
	}
	var items []item

	for _, res := range req.IaC.Resources {
		est := estimateResource(res)
		items = append(items, item{
			Name:    parser.ShortType(res.Type) + "." + res.Name,
			SKU:     est.sku,
			Monthly: est.monthly,
		})
		total += est.monthly
	}

	emit.SendMessage(fmt.Sprintf("## Estimated Monthly Cost: **$%.2f**\n\n", total))
	emit.SendMessage("| Resource | SKU | Monthly |\n|----------|-----|---------|\n")
	for _, it := range items {
		emit.SendMessage(fmt.Sprintf("| %s | %s | $%.2f |\n", it.Name, it.SKU, it.Monthly))
	}
	emit.SendMessage("\n")

	return nil
}

type estimate struct {
	sku     string
	monthly float64
}

func estimateResource(res protocol.Resource) estimate {
	region := "eastus"
	if loc, ok := res.Properties["location"].(string); ok && loc != "" {
		region = loc
	}
	_ = region // region for future API lookups

	switch res.Type {
	case "azurerm_kubernetes_cluster":
		return estimateAKS(res)
	case "azurerm_virtual_machine", "azurerm_linux_virtual_machine", "azurerm_windows_virtual_machine":
		return estimateVM(res)
	case "azurerm_storage_account":
		return estimateStorage(res)
	case "azurerm_app_service_plan", "azurerm_service_plan":
		return estimateAppService(res)
	case "azurerm_container_registry":
		return estimateACR(res)
	case "azurerm_key_vault":
		return estimate{sku: "Standard", monthly: 3.00}
	case "azurerm_virtual_network", "azurerm_subnet", "azurerm_network_security_group":
		return estimate{sku: "N/A", monthly: 0}
	default:
		return estimate{sku: "Unknown", monthly: 0}
	}
}

const hoursPerMonth = 730

func estimateAKS(res protocol.Resource) estimate {
	vmSize := "Standard_D2s_v3"
	nodeCount := 3
	if pool, ok := res.Properties["default_node_pool"].(map[string]interface{}); ok {
		if s, ok := pool["vm_size"].(string); ok {
			vmSize = s
		}
		if c, ok := pool["node_count"].(int); ok {
			nodeCount = c
		}
	}
	hourly := vmPrice(vmSize)
	monthly := hourly*hoursPerMonth*float64(nodeCount) + 18.25
	return estimate{
		sku:     fmt.Sprintf("%dx %s", nodeCount, vmSize),
		monthly: monthly,
	}
}

func estimateVM(res protocol.Resource) estimate {
	vmSize := "Standard_D2s_v3"
	if s, ok := res.Properties["vm_size"].(string); ok {
		vmSize = s
	} else if s, ok := res.Properties["size"].(string); ok {
		vmSize = s
	}
	hourly := vmPrice(vmSize)
	if res.Type == "azurerm_windows_virtual_machine" {
		hourly *= 1.5
	}
	return estimate{sku: vmSize, monthly: hourly * hoursPerMonth}
}

func estimateStorage(res protocol.Resource) estimate {
	sku := "Standard_LRS"
	if rep, ok := res.Properties["account_replication_type"].(string); ok {
		sku = "Standard_" + rep
	}
	pricePerGB := storagePrices[sku]
	if pricePerGB == 0 {
		pricePerGB = 0.0184
	}
	return estimate{sku: sku, monthly: pricePerGB * 100}
}

func estimateAppService(res protocol.Resource) estimate {
	sku := "B1"
	if s, ok := res.Properties["sku_name"].(string); ok {
		sku = s
	}
	monthly := appServicePrices[sku]
	if monthly == 0 {
		monthly = 13.14
	}
	return estimate{sku: sku, monthly: monthly}
}

func estimateACR(res protocol.Resource) estimate {
	sku := "Basic"
	if s, ok := res.Properties["sku"].(string); ok {
		sku = s
	}
	monthly := acrPrices[sku]
	if monthly == 0 {
		monthly = 5.00
	}
	return estimate{sku: sku, monthly: monthly}
}

func vmPrice(sku string) float64 {
	if p, ok := vmSkuPrices[sku]; ok {
		return p
	}
	return 0.096
}

var vmSkuPrices = map[string]float64{
	"Standard_B1s": 0.0104, "Standard_B1ms": 0.0207,
	"Standard_B2s": 0.0416, "Standard_B2ms": 0.0832,
	"Standard_D2s_v3": 0.096, "Standard_D4s_v3": 0.192, "Standard_D8s_v3": 0.384,
	"Standard_D2s_v4": 0.096, "Standard_D4s_v4": 0.192, "Standard_D8s_v4": 0.384,
	"Standard_D2s_v5": 0.096, "Standard_D4s_v5": 0.192, "Standard_D8s_v5": 0.384,
	"Standard_E2s_v3": 0.126, "Standard_E4s_v3": 0.252, "Standard_E8s_v3": 0.504,
	"Standard_F2s_v2": 0.085, "Standard_F4s_v2": 0.169, "Standard_F8s_v2": 0.338,
}

var storagePrices = map[string]float64{
	"Standard_LRS": 0.0184, "Standard_GRS": 0.0368, "Standard_ZRS": 0.023,
	"Standard_GZRS": 0.0414, "Premium_LRS": 0.15, "Standard_RA-GRS": 0.046,
}

var appServicePrices = map[string]float64{
	"F1": 0, "D1": 9.49, "B1": 13.14, "B2": 26.28, "B3": 52.56,
	"S1": 69.35, "S2": 138.70, "S3": 277.40,
	"P1v2": 73.00, "P2v2": 146.00, "P3v2": 292.00,
	"P1v3": 95.63, "P2v3": 191.25, "P3v3": 382.50,
}

var acrPrices = map[string]float64{
	"Basic": 5.00, "Standard": 20.00, "Premium": 50.00,
}
