// Package analyzer provides IaC analysis rules for policy checking,
// security scanning, and compliance auditing.
package analyzer

// Severity levels for findings.
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
	SeverityInfo     = "info"
)

// Finding represents a single analysis result.
type Finding struct {
	RuleID       string `json:"rule_id"`
	Category     string `json:"category"`
	Severity     string `json:"severity"`
	Resource     string `json:"resource"`
	ResourceType string `json:"resource_type"`
	Message      string `json:"message"`
	Remediation  string `json:"remediation"`
}

func severityIcon(sev string) string {
	switch sev {
	case SeverityCritical:
		return "🔴"
	case SeverityHigh:
		return "🟠"
	case SeverityMedium:
		return "🟡"
	case SeverityLow:
		return "🔵"
	default:
		return "⚪"
	}
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
