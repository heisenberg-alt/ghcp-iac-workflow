package analyzer

import (
	"fmt"
	"regexp"
	"strings"
)

// Rule represents a deterministic analysis rule.
type Rule struct {
	ID          string
	Category    string
	Severity    string
	Title       string
	Description string
	Remediation string

	// ResourceTypes this rule applies to (empty = all)
	ResourceTypes []string

	// Property-based check
	Property string
	Expected interface{}
	CheckFn  func(props map[string]interface{}) string

	// Pattern-based check (for raw block scanning)
	Patterns []*regexp.Regexp
}

// Applies returns true if this rule applies to the given resource type.
func (r Rule) Applies(resType string) bool {
	if len(r.ResourceTypes) == 0 {
		return true
	}
	for _, t := range r.ResourceTypes {
		if t == resType || t == "*" {
			return true
		}
	}
	return false
}

// IsPatternRule returns true if this is a regex pattern-based rule.
func (r Rule) IsPatternRule() bool {
	return len(r.Patterns) > 0
}

// Check evaluates prop-based rules against resource properties.
func (r Rule) Check(props map[string]interface{}) string {
	if r.CheckFn != nil {
		return r.CheckFn(props)
	}
	if r.Property == "" {
		return ""
	}

	val, ok := props[r.Property]
	if !ok {
		return fmt.Sprintf("%s is not set (expected: %v)", r.Property, r.Expected)
	}

	if fmt.Sprintf("%v", val) != fmt.Sprintf("%v", r.Expected) {
		return fmt.Sprintf("%s = %v (expected: %v)", r.Property, val, r.Expected)
	}
	return ""
}

// CheckPatterns evaluates regex patterns against raw code blocks.
func (r Rule) CheckPatterns(rawBlock string) []string {
	var violations []string
	for _, p := range r.Patterns {
		if matches := p.FindAllString(rawBlock, -1); len(matches) > 0 {
			violations = append(violations, fmt.Sprintf("%s: found %d occurrence(s)", r.Title, len(matches)))
		}
	}
	return violations
}

// AllRules returns all deterministic analysis rules.
func AllRules() []Rule {
	var rules []Rule
	rules = append(rules, policyRules()...)
	rules = append(rules, securityRules()...)
	rules = append(rules, complianceRules()...)
	return rules
}

func policyRules() []Rule {
	return []Rule{
		{
			ID:            "POL-001",
			Category:      "Policy",
			Severity:      SeverityHigh,
			Title:         "Storage HTTPS Required",
			Description:   "Storage account must enforce HTTPS-only traffic",
			Remediation:   "Set enable_https_traffic_only = true",
			ResourceTypes: []string{"azurerm_storage_account"},
			Property:      "enable_https_traffic_only",
			Expected:      true,
		},
		{
			ID:            "POL-002",
			Category:      "Policy",
			Severity:      SeverityHigh,
			Title:         "AKS RBAC Required",
			Description:   "AKS clusters must enable RBAC",
			Remediation:   "Set role_based_access_control_enabled = true",
			ResourceTypes: []string{"azurerm_kubernetes_cluster"},
			Property:      "role_based_access_control_enabled",
			Expected:      true,
		},
		{
			ID:            "POL-003",
			Category:      "Policy",
			Severity:      SeverityMedium,
			Title:         "Minimum TLS Version",
			Description:   "Resources must use TLS 1.2 or higher",
			Remediation:   "Set min_tls_version = \"TLS1_2\"",
			ResourceTypes: []string{"azurerm_storage_account", "azurerm_redis_cache", "azurerm_mssql_server"},
			Property:      "min_tls_version",
			Expected:      "TLS1_2",
		},
		{
			ID:            "POL-004",
			Category:      "Policy",
			Severity:      SeverityHigh,
			Title:         "No Public Blob Access",
			Description:   "Storage accounts must not allow public blob access",
			Remediation:   "Set allow_blob_public_access = false",
			ResourceTypes: []string{"azurerm_storage_account"},
			Property:      "allow_blob_public_access",
			Expected:      false,
		},
		{
			ID:            "POL-005",
			Category:      "Policy",
			Severity:      SeverityHigh,
			Title:         "Key Vault Soft Delete",
			Description:   "Key Vault must have soft delete enabled",
			Remediation:   "Set soft_delete_enabled = true",
			ResourceTypes: []string{"azurerm_key_vault"},
			Property:      "soft_delete_enabled",
			Expected:      true,
		},
		{
			ID:            "POL-006",
			Category:      "Policy",
			Severity:      SeverityMedium,
			Title:         "Key Vault Purge Protection",
			Description:   "Key Vault must have purge protection enabled",
			Remediation:   "Set purge_protection_enabled = true",
			ResourceTypes: []string{"azurerm_key_vault"},
			Property:      "purge_protection_enabled",
			Expected:      true,
		},
	}
}

func securityRules() []Rule {
	return []Rule{
		{
			ID:            "SEC-001",
			Category:      "Security",
			Severity:      SeverityCritical,
			Title:         "Hardcoded Secrets",
			Description:   "Code contains potential hardcoded credentials",
			Remediation:   "Use Key Vault references or environment variables",
			ResourceTypes: []string{"*"},
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(password|secret|key)\s*=\s*"[^"]{8,}"`),
				regexp.MustCompile(`(?i)api[_-]?key\s*=\s*"[^"]{8,}"`),
				regexp.MustCompile(`(?i)connection[_-]?string\s*=\s*"[^"]+"`),
			},
		},
		{
			ID:            "SEC-002",
			Category:      "Security",
			Severity:      SeverityHigh,
			Title:         "Public Network Access",
			Description:   "Resource allows public network access",
			Remediation:   "Set public_network_access_enabled = false or configure network rules",
			ResourceTypes: []string{"azurerm_storage_account", "azurerm_key_vault", "azurerm_mssql_server", "azurerm_cosmosdb_account"},
			CheckFn: func(props map[string]interface{}) string {
				if v, ok := props["public_network_access_enabled"]; ok && v == true {
					return "Public network access is enabled"
				}
				return ""
			},
		},
		{
			ID:            "SEC-003",
			Category:      "Security",
			Severity:      SeverityHigh,
			Title:         "HTTPS Not Enforced",
			Description:   "HTTPS traffic is not enforced",
			Remediation:   "Set enable_https_traffic_only = true",
			ResourceTypes: []string{"azurerm_storage_account"},
			CheckFn: func(props map[string]interface{}) string {
				if v, ok := props["enable_https_traffic_only"]; ok && v == false {
					return "HTTPS traffic is not enforced"
				}
				return ""
			},
		},
		{
			ID:            "SEC-004",
			Category:      "Security",
			Severity:      SeverityMedium,
			Title:         "Encryption at Rest",
			Description:   "Customer-managed encryption key not configured",
			Remediation:   "Configure customer_managed_key block",
			ResourceTypes: []string{"azurerm_storage_account", "azurerm_mssql_database"},
			CheckFn: func(props map[string]interface{}) string {
				if _, ok := props["customer_managed_key"]; !ok {
					return "No customer-managed encryption key configured"
				}
				return ""
			},
		},
		{
			ID:            "SEC-005",
			Category:      "Security",
			Severity:      SeverityHigh,
			Title:         "Overly Permissive NSG",
			Description:   "Network Security Group allows unrestricted access",
			Remediation:   "Restrict source_address_prefix to specific IPs/ranges",
			ResourceTypes: []string{"azurerm_network_security_group"},
			Patterns: []*regexp.Regexp{
				regexp.MustCompile(`source_address_prefix\s*=\s*"\*"`),
				regexp.MustCompile(`destination_port_range\s*=\s*"\*"`),
			},
		},
	}
}

func complianceRules() []Rule {
	return []Rule{
		{
			ID:            "CIS-4.1",
			Category:      "Compliance",
			Severity:      SeverityHigh,
			Title:         "CIS: Storage HTTPS",
			Description:   "CIS Azure 4.1 - Ensure storage account HTTPS transfer",
			Remediation:   "Set enable_https_traffic_only = true",
			ResourceTypes: []string{"azurerm_storage_account"},
			Property:      "enable_https_traffic_only",
			Expected:      true,
		},
		{
			ID:            "CIS-8.1",
			Category:      "Compliance",
			Severity:      SeverityMedium,
			Title:         "CIS: Key Vault Expiry",
			Description:   "CIS Azure 8.1 - Ensure Key Vault is recoverable",
			Remediation:   "Set soft_delete_enabled = true",
			ResourceTypes: []string{"azurerm_key_vault"},
			Property:      "soft_delete_enabled",
			Expected:      true,
		},
		{
			ID:            "NIST-SC7",
			Category:      "Compliance",
			Severity:      SeverityHigh,
			Title:         "NIST SC-7: Boundary Protection",
			Description:   "Network boundaries must have proper controls",
			Remediation:   "Configure network_rules with default_action = \"Deny\"",
			ResourceTypes: []string{"azurerm_storage_account"},
			CheckFn: func(props map[string]interface{}) string {
				rules, ok := props["network_rules"].(map[string]interface{})
				if !ok {
					return "No network rules configured (NIST SC-7)"
				}
				if action, ok := rules["default_action"].(string); ok {
					if strings.ToLower(action) != "deny" {
						return fmt.Sprintf("Network rules default_action = %s (should be Deny)", action)
					}
				}
				return ""
			},
		},
		{
			ID:            "NIST-SC28",
			Category:      "Compliance",
			Severity:      SeverityMedium,
			Title:         "NIST SC-28: Protection at Rest",
			Description:   "Data at rest must be encrypted",
			Remediation:   "Enable infrastructure encryption",
			ResourceTypes: []string{"azurerm_storage_account"},
			CheckFn: func(props map[string]interface{}) string {
				if v, ok := props["infrastructure_encryption_enabled"]; !ok || v != true {
					return "Infrastructure encryption not enabled"
				}
				return ""
			},
		},
		{
			ID:            "SOC2-CC6.1",
			Category:      "Compliance",
			Severity:      SeverityHigh,
			Title:         "SOC2 CC6.1: Logical Access",
			Description:   "Logical access must be restricted",
			Remediation:   "Disable public blob access",
			ResourceTypes: []string{"azurerm_storage_account"},
			Property:      "allow_blob_public_access",
			Expected:      false,
		},
		{
			ID:            "SOC2-CC6.6",
			Category:      "Compliance",
			Severity:      SeverityMedium,
			Title:         "SOC2 CC6.6: Encryption in Transit",
			Description:   "Data in transit must be encrypted",
			Remediation:   "Set min_tls_version = TLS1_2",
			ResourceTypes: []string{"azurerm_storage_account", "azurerm_redis_cache"},
			Property:      "min_tls_version",
			Expected:      "TLS1_2",
		},
	}
}
