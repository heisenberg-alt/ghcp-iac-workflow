package parser

import (
	"strings"
	"testing"
)

func TestDetectIaCType_Terraform(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"resource block", `resource "azurerm_storage_account" "example" { }`},
		{"variable block", `variable "name" { type = string }`},
		{"provider block", `provider "azurerm" { features {} }`},
		{"terraform block", `terraform { required_providers { } }`},
		{"module block", `module "network" { source = "./modules" }`},
		{"data block", `data "azurerm_resource_group" "existing" { name = "rg" }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectIaCType(tt.code)
			if got != Terraform {
				t.Errorf("DetectIaCType(%q) = %v, want Terraform", tt.code, got)
			}
		})
	}
}

func TestDetectIaCType_Bicep(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"resource", `resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' = {`},
		{"param", `param location string`},
		{"targetScope", `targetScope = 'subscription'`},
		{"module", `module vnet './modules/vnet.bicep'`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectIaCType(tt.code)
			if got != Bicep {
				t.Errorf("DetectIaCType(%q) = %v, want Bicep", tt.code, got)
			}
		})
	}
}

func TestDetectIaCType_Unknown(t *testing.T) {
	tests := []string{
		"hello world",
		"just some plain text",
		"",
	}
	for _, code := range tests {
		got := DetectIaCType(code)
		if got != Unknown {
			t.Errorf("DetectIaCType(%q) = %v, want Unknown", code, got)
		}
	}
}

func TestExtractCode_FencedBlock(t *testing.T) {
	msg := "Check this:\n```terraform\nresource \"azurerm_storage_account\" \"ex\" {\n  name = \"test\"\n}\n```"
	got := ExtractCode(msg)
	if !strings.Contains(got, "azurerm_storage_account") {
		t.Errorf("ExtractCode should extract fenced code block, got: %q", got)
	}
}

func TestExtractCode_MultipleFencedBlocks(t *testing.T) {
	msg := "```hcl\nresource \"a\" \"b\" {}\n```\nand\n```terraform\nresource \"c\" \"d\" {}\n```"
	got := ExtractCode(msg)
	if !strings.Contains(got, "resource \"a\"") || !strings.Contains(got, "resource \"c\"") {
		t.Errorf("ExtractCode should extract all fenced blocks, got: %q", got)
	}
}

func TestExtractCode_InlineCode(t *testing.T) {
	msg := "Use `enable_https_traffic_only = true` in your config"
	got := ExtractCode(msg)
	if !strings.Contains(got, "enable_https_traffic_only") {
		t.Errorf("ExtractCode should extract inline code, got: %q", got)
	}
}

func TestExtractCode_RawCode(t *testing.T) {
	code := `resource "azurerm_storage_account" "ex" {
  name = "test"
}`
	got := ExtractCode(code)
	if got == "" {
		t.Error("ExtractCode should return raw IaC code when no delimiters present")
	}
}

func TestExtractCode_NoCode(t *testing.T) {
	got := ExtractCode("just plain text with no code")
	if got != "" {
		t.Errorf("ExtractCode should return empty for plain text, got: %q", got)
	}
}

func TestParseTerraform_SingleResource(t *testing.T) {
	code := `resource "azurerm_storage_account" "example" {
  name                     = "storagetest"
  resource_group_name      = "rg-test"
  location                 = "eastus"
  account_tier             = "Standard"
  account_replication_type = "LRS"
  enable_https_traffic_only = true
}`
	resources := ParseTerraform(code)
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}

	r := resources[0]
	if r.Type != "azurerm_storage_account" {
		t.Errorf("Type = %q, want azurerm_storage_account", r.Type)
	}
	if r.Name != "example" {
		t.Errorf("Name = %q, want example", r.Name)
	}
	if r.Line != 1 {
		t.Errorf("Line = %d, want 1", r.Line)
	}

	if v, ok := r.Properties["name"]; !ok || v != "storagetest" {
		t.Errorf("Properties[name] = %v, want storagetest", v)
	}
	if v, ok := r.Properties["enable_https_traffic_only"]; !ok || v != true {
		t.Errorf("Properties[enable_https_traffic_only] = %v, want true", v)
	}
}

func TestParseTerraform_NestedBlock(t *testing.T) {
	code := `resource "azurerm_storage_account" "ex" {
  name = "test"
  network_rules {
    default_action = "Deny"
    bypass         = "AzureServices"
  }
}`
	resources := ParseTerraform(code)
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}

	r := resources[0]
	nested, ok := r.Properties["network_rules"].(map[string]interface{})
	if !ok {
		t.Fatal("network_rules should be a nested map")
	}
	if v := nested["default_action"]; v != "Deny" {
		t.Errorf("network_rules.default_action = %v, want Deny", v)
	}
}

func TestParseTerraform_MultipleResources(t *testing.T) {
	code := `resource "azurerm_resource_group" "rg" {
  name     = "test-rg"
  location = "eastus"
}

resource "azurerm_storage_account" "sa" {
  name = "testsa"
}`
	resources := ParseTerraform(code)
	if len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(resources))
	}
	if resources[0].Type != "azurerm_resource_group" {
		t.Errorf("first resource Type = %q", resources[0].Type)
	}
	if resources[1].Type != "azurerm_storage_account" {
		t.Errorf("second resource Type = %q", resources[1].Type)
	}
}

func TestParseTerraform_BooleanAndNumbers(t *testing.T) {
	code := `resource "azurerm_storage_account" "ex" {
  enable_https = true
  public_access = false
  count = 3
}`
	resources := ParseTerraform(code)
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	r := resources[0]
	if v := r.Properties["enable_https"]; v != true {
		t.Errorf("enable_https = %v (%T), want true", v, v)
	}
	if v := r.Properties["public_access"]; v != false {
		t.Errorf("public_access = %v (%T), want false", v, v)
	}
	if v := r.Properties["count"]; v != 3 {
		t.Errorf("count = %v (%T), want 3", v, v)
	}
}

func TestParseBicep_SingleResource(t *testing.T) {
	code := `resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'mystorageaccount'
  location: 'eastus'
  kind: 'StorageV2'
  properties: {
    supportsHttpsTrafficOnly: true
    minimumTlsVersion: 'TLS1_2'
  }
}`
	resources := ParseBicep(code)
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}

	r := resources[0]
	if r.Type != "azurerm_storage_account" {
		t.Errorf("Type = %q, want azurerm_storage_account", r.Type)
	}
	if r.Name != "storageAccount" {
		t.Errorf("Name = %q, want storageAccount", r.Name)
	}

	// Properties should be flattened and mapped
	if v, ok := r.Properties["enable_https_traffic_only"]; !ok || v != true {
		t.Errorf("Properties[enable_https_traffic_only] = %v, want true", v)
	}
	if v, ok := r.Properties["min_tls_version"]; !ok || v != "TLS1_2" {
		t.Errorf("Properties[min_tls_version] = %v, want TLS1_2", v)
	}
}

func TestParseBicep_UnknownResourceType(t *testing.T) {
	code := `resource custom 'Microsoft.Custom/something@2023-01-01' = {
  name: 'test'
}`
	resources := ParseBicep(code)
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if resources[0].Type != "Microsoft.Custom/something" {
		t.Errorf("Type = %q, want Microsoft.Custom/something (no mapping)", resources[0].Type)
	}
}

func TestGetNestedProperty(t *testing.T) {
	props := map[string]interface{}{
		"name": "test",
		"network_rules": map[string]interface{}{
			"default_action": "Deny",
			"bypass":         "AzureServices",
		},
	}

	val, ok := GetNestedProperty(props, "network_rules.default_action")
	if !ok || val != "Deny" {
		t.Errorf("GetNestedProperty(network_rules.default_action) = (%v, %v), want (Deny, true)", val, ok)
	}

	val, ok = GetNestedProperty(props, "name")
	if !ok || val != "test" {
		t.Errorf("GetNestedProperty(name) = (%v, %v), want (test, true)", val, ok)
	}

	_, ok = GetNestedProperty(props, "missing.key")
	if ok {
		t.Error("GetNestedProperty(missing.key) should return false")
	}

	_, ok = GetNestedProperty(props, "name.nested")
	if ok {
		t.Error("GetNestedProperty(name.nested) should return false for non-map value")
	}
}

func TestFindMatchingBrace(t *testing.T) {
	tests := []struct {
		code  string
		start int
		want  int
	}{
		{"{}", 0, 1},
		{"{ { } }", 0, 6},
		{"{ a { b } c }", 0, 12},
		{"no brace", 0, -1},
	}
	for _, tt := range tests {
		got := findMatchingBrace(tt.code, tt.start)
		if got != tt.want {
			t.Errorf("findMatchingBrace(%q, %d) = %d, want %d", tt.code, tt.start, got, tt.want)
		}
	}
}

func TestParseResources_AutoDetect(t *testing.T) {
	tfCode := `resource "azurerm_storage_account" "ex" { name = "test" }`
	resources := ParseResources(tfCode)
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource from TF auto-detect, got %d", len(resources))
	}
	if resources[0].Type != "azurerm_storage_account" {
		t.Errorf("Type = %q", resources[0].Type)
	}
}

func TestIaCType_String(t *testing.T) {
	if Terraform.String() != "Terraform" {
		t.Errorf("Terraform.String() = %q", Terraform.String())
	}
	if Bicep.String() != "Bicep" {
		t.Errorf("Bicep.String() = %q", Bicep.String())
	}
}

func TestResource_String(t *testing.T) {
	r := Resource{Type: "azurerm_storage_account", Name: "ex"}
	got := r.String()
	if got != "azurerm_storage_account.ex" {
		t.Errorf("Resource.String() = %q, want azurerm_storage_account.ex", got)
	}
}
