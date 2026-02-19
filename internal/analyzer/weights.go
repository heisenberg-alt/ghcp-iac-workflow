package analyzer

// ResourceRiskWeight returns the risk score for a resource type.
// Higher scores indicate resources with greater blast radius when modified.
func ResourceRiskWeight(resType string) int {
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
