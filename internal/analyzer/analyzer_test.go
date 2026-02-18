package analyzer

import (
	"testing"
)

func TestAllRules_Count(t *testing.T) {
	rules := AllRules()
	if len(rules) != 12 {
		t.Errorf("AllRules() returned %d rules, want 12", len(rules))
	}
}

func TestAllRules_UniqueIDs(t *testing.T) {
	rules := AllRules()
	seen := make(map[string]bool)
	for _, r := range rules {
		if seen[r.ID] {
			t.Errorf("Duplicate rule ID: %s", r.ID)
		}
		seen[r.ID] = true
	}
}

func TestRule_Applies(t *testing.T) {
	rule := Rule{
		ResourceTypes: []string{"azurerm_storage_account", "azurerm_key_vault"},
	}
	if !rule.Applies("azurerm_storage_account") {
		t.Error("Rule should apply to azurerm_storage_account")
	}
	if !rule.Applies("azurerm_key_vault") {
		t.Error("Rule should apply to azurerm_key_vault")
	}
	if rule.Applies("azurerm_virtual_machine") {
		t.Error("Rule should not apply to azurerm_virtual_machine")
	}
}

func TestRule_Applies_Wildcard(t *testing.T) {
	rule := Rule{ResourceTypes: []string{"*"}}
	if !rule.Applies("azurerm_anything") {
		t.Error("Wildcard should match everything")
	}
}

func TestRule_Applies_Empty(t *testing.T) {
	rule := Rule{}
	if !rule.Applies("azurerm_anything") {
		t.Error("Empty ResourceTypes should match everything")
	}
}

func TestRule_Check_PropertyMatch(t *testing.T) {
	rule := Rule{
		Property: "enable_https_traffic_only",
		Expected: true,
	}
	msg := rule.Check(map[string]interface{}{"enable_https_traffic_only": true})
	if msg != "" {
		t.Errorf("Check should pass when property matches, got: %q", msg)
	}
	msg = rule.Check(map[string]interface{}{"enable_https_traffic_only": false})
	if msg == "" {
		t.Error("Check should fail when property does not match")
	}
	msg = rule.Check(map[string]interface{}{})
	if msg == "" {
		t.Error("Check should fail when property is missing")
	}
}

func TestRule_Check_CustomFn(t *testing.T) {
	rule := Rule{
		CheckFn: func(props map[string]interface{}) string {
			if v, ok := props["public_network_access_enabled"]; ok && v == true {
				return "Public access enabled"
			}
			return ""
		},
	}
	msg := rule.Check(map[string]interface{}{"public_network_access_enabled": true})
	if msg == "" {
		t.Error("CheckFn should return violation")
	}
	msg = rule.Check(map[string]interface{}{"public_network_access_enabled": false})
	if msg != "" {
		t.Errorf("CheckFn should pass, got: %q", msg)
	}
}

func TestRule_CheckPatterns(t *testing.T) {
	rules := securityRules()
	var sec001 Rule
	for _, r := range rules {
		if r.ID == "SEC-001" {
			sec001 = r
			break
		}
	}
	if !sec001.IsPatternRule() {
		t.Fatal("SEC-001 should be a pattern rule")
	}
	violations := sec001.CheckPatterns(`resource "azurerm_mssql_server" "ex" {
  administrator_login_password = "SuperSecretPassword123!"
}`)
	if len(violations) == 0 {
		t.Error("SEC-001 should detect hardcoded password")
	}
	violations = sec001.CheckPatterns(`resource "azurerm_storage_account" "ex" {
  name = "test"
}`)
	if len(violations) != 0 {
		t.Errorf("SEC-001 should not flag short values, got %d violations", len(violations))
	}
}

func TestRule_CheckPatterns_NSG(t *testing.T) {
	rules := securityRules()
	var sec005 Rule
	for _, r := range rules {
		if r.ID == "SEC-005" {
			sec005 = r
			break
		}
	}
	violations := sec005.CheckPatterns(`resource "azurerm_network_security_group" "ex" {
  security_rule {
    source_address_prefix = "*"
    destination_port_range = "*"
  }
}`)
	if len(violations) < 2 {
		t.Errorf("SEC-005 should detect both wildcard patterns, got %d violations", len(violations))
	}
}

func TestPolicyRules_StorageHTTPS(t *testing.T) {
	rules := policyRules()
	var pol001 Rule
	for _, r := range rules {
		if r.ID == "POL-001" {
			pol001 = r
			break
		}
	}
	if !pol001.Applies("azurerm_storage_account") {
		t.Error("POL-001 should apply to storage accounts")
	}
	if pol001.Applies("azurerm_key_vault") {
		t.Error("POL-001 should not apply to key vaults")
	}
	msg := pol001.Check(map[string]interface{}{"enable_https_traffic_only": false})
	if msg == "" {
		t.Error("POL-001 should fail when HTTPS is not enforced")
	}
	msg = pol001.Check(map[string]interface{}{"enable_https_traffic_only": true})
	if msg != "" {
		t.Errorf("POL-001 should pass when HTTPS is enforced, got: %q", msg)
	}
}

func TestComplianceRules_NIST_SC7(t *testing.T) {
	rules := complianceRules()
	var nistSC7 Rule
	for _, r := range rules {
		if r.ID == "NIST-SC7" {
			nistSC7 = r
			break
		}
	}
	msg := nistSC7.Check(map[string]interface{}{})
	if msg == "" {
		t.Error("NIST-SC7 should fail when no network rules")
	}
	msg = nistSC7.Check(map[string]interface{}{
		"network_rules": map[string]interface{}{
			"default_action": "Allow",
		},
	})
	if msg == "" {
		t.Error("NIST-SC7 should fail when default_action is Allow")
	}
	msg = nistSC7.Check(map[string]interface{}{
		"network_rules": map[string]interface{}{
			"default_action": "Deny",
		},
	})
	if msg != "" {
		t.Errorf("NIST-SC7 should pass when default_action is Deny, got: %q", msg)
	}
}

func TestSeverityConstants(t *testing.T) {
	if SeverityCritical != "critical" {
		t.Error("SeverityCritical mismatch")
	}
	if SeverityHigh != "high" {
		t.Error("SeverityHigh mismatch")
	}
	if SeverityMedium != "medium" {
		t.Error("SeverityMedium mismatch")
	}
	if SeverityLow != "low" {
		t.Error("SeverityLow mismatch")
	}
}

func TestResourceRiskWeight(t *testing.T) {
	tests := []struct {
		resType string
		minWt   int
	}{
		{"azurerm_kubernetes_cluster", 5},
		{"azurerm_key_vault", 4},
		{"azurerm_storage_account", 3},
		{"azurerm_unknown_type", 1},
	}
	for _, tt := range tests {
		got := resourceRiskWeight(tt.resType)
		if got < tt.minWt {
			t.Errorf("resourceRiskWeight(%q) = %d, want >= %d", tt.resType, got, tt.minWt)
		}
	}
}

func TestSeverityIcon(t *testing.T) {
	got := severityIcon(SeverityCritical)
	if got == "" {
		t.Error("severityIcon should return non-empty for critical")
	}
}
