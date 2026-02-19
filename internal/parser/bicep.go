package parser

import (
	"regexp"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

var bicepResourceRe = regexp.MustCompile(`resource\s+(\w+)\s+'([^']+)'\s*=\s*\{`)

// bicepToTFType maps Bicep resource types to Terraform type names.
var bicepToTFType = map[string]string{
	"Microsoft.Storage/storageAccounts":          "azurerm_storage_account",
	"Microsoft.KeyVault/vaults":                  "azurerm_key_vault",
	"Microsoft.Network/virtualNetworks":          "azurerm_virtual_network",
	"Microsoft.Network/networkSecurityGroups":    "azurerm_network_security_group",
	"Microsoft.ContainerService/managedClusters": "azurerm_kubernetes_cluster",
	"Microsoft.ContainerRegistry/registries":     "azurerm_container_registry",
	"Microsoft.Web/serverfarms":                  "azurerm_service_plan",
	"Microsoft.Web/sites":                        "azurerm_app_service",
	"Microsoft.Compute/virtualMachines":          "azurerm_virtual_machine",
	"Microsoft.Sql/servers":                      "azurerm_mssql_server",
	"Microsoft.Sql/servers/databases":            "azurerm_mssql_database",
	"Microsoft.Cache/redis":                      "azurerm_redis_cache",
	"Microsoft.DocumentDB/databaseAccounts":      "azurerm_cosmosdb_account",
}

// bicepToTFProperty maps Bicep property names to Terraform property names.
var bicepToTFProperty = map[string]string{
	"supportsHttpsTrafficOnly":     "enable_https_traffic_only",
	"minimumTlsVersion":            "min_tls_version",
	"allowBlobPublicAccess":        "allow_blob_public_access",
	"enableSoftDelete":             "soft_delete_enabled",
	"enablePurgeProtection":        "purge_protection_enabled",
	"enableRbac":                   "role_based_access_control_enabled",
	"publicNetworkAccess":          "public_network_access_enabled",
	"networkAcls":                  "network_rules",
	"defaultAction":                "default_action",
	"enabledForDeployment":         "enabled_for_deployment",
	"enabledForDiskEncryption":     "enabled_for_disk_encryption",
	"enabledForTemplateDeployment": "enabled_for_template_deployment",
	"keySource":                    "key_source",
	"skuName":                      "sku_name",
}

// ParseBicep extracts resources from Bicep code.
func ParseBicep(code string) []protocol.Resource {
	var resources []protocol.Resource
	matches := bicepResourceRe.FindAllStringSubmatchIndex(code, -1)

	for _, loc := range matches {
		resName := code[loc[2]:loc[3]]
		bicepType := code[loc[4]:loc[5]]

		// Strip version from type: Microsoft.Storage/storageAccounts@2023-01-01
		cleanType := bicepType
		if atIdx := strings.Index(cleanType, "@"); atIdx > 0 {
			cleanType = cleanType[:atIdx]
		}

		// Map to Terraform type name
		tfType, ok := bicepToTFType[cleanType]
		if !ok {
			tfType = cleanType
		}

		braceStart := strings.LastIndex(code[loc[0]:loc[1]], "{")
		if braceStart < 0 {
			continue
		}
		braceStart += loc[0]
		braceEnd := findMatchingBrace(code, braceStart)
		if braceEnd < 0 {
			continue
		}

		block := code[braceStart+1 : braceEnd]
		lineNum := strings.Count(code[:loc[0]], "\n") + 1

		props := parseBicepBlock(block)

		resources = append(resources, protocol.Resource{
			Type:       tfType,
			Name:       resName,
			Properties: props,
			Line:       lineNum,
			RawBlock:   code[loc[0] : braceEnd+1],
		})
	}

	return resources
}

// parseBicepBlock parses Bicep block content into a properties map.
// It flattens the nested "properties:" block to match Terraform structure.
func parseBicepBlock(block string) map[string]interface{} {
	raw := parseBicepBlockRaw(block)

	// Flatten properties block
	if propsBlock, ok := raw["properties"].(map[string]interface{}); ok {
		for k, v := range propsBlock {
			tfKey := k
			if mapped, ok := bicepToTFProperty[k]; ok {
				tfKey = mapped
			}
			raw[tfKey] = v
		}
		delete(raw, "properties")
	}

	// Map top-level property names
	result := make(map[string]interface{})
	for k, v := range raw {
		tfKey := k
		if mapped, ok := bicepToTFProperty[k]; ok {
			tfKey = mapped
		}
		result[tfKey] = v
	}

	return result
}

func parseBicepBlockRaw(block string) map[string]interface{} {
	props := make(map[string]interface{})
	lines := strings.Split(block, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// key: value
		if colonIdx := strings.Index(line, ":"); colonIdx > 0 {
			key := strings.TrimSpace(line[:colonIdx])
			val := strings.TrimSpace(line[colonIdx+1:])

			// Nested block: key: {
			if val == "{" || strings.HasSuffix(val, "{") {
				depth := 1
				var nested []string
				for i++; i < len(lines) && depth > 0; i++ {
					l := strings.TrimSpace(lines[i])
					depth += strings.Count(l, "{") - strings.Count(l, "}")
					if depth > 0 {
						nested = append(nested, lines[i])
					}
				}
				i--
				props[key] = parseBicepBlockRaw(strings.Join(nested, "\n"))
				continue
			}

			props[key] = parseBicepValue(val)
		}
	}

	return props
}

func parseBicepValue(val string) interface{} {
	// Remove trailing comma
	val = strings.TrimRight(val, ",")
	val = strings.TrimSpace(val)

	// Quoted string
	if strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'") {
		return strings.Trim(val, "'")
	}

	switch strings.ToLower(val) {
	case "true":
		return true
	case "false":
		return false
	}

	return val
}
