// Package parser provides shared Terraform and Bicep parsing for IaC analysis.
// It extracts resource definitions, properties, and structure from IaC code.
package parser

import "strings"

// IaCType represents the type of Infrastructure as Code.
type IaCType string

const (
	Terraform IaCType = "Terraform"
	Bicep     IaCType = "Bicep"
	Unknown   IaCType = "Unknown"
)

// ShortType strips the provider prefix from a resource type.
// e.g. "azurerm_storage_account" → "storage_account"
func ShortType(t string) string {
	if i := strings.IndexByte(t, '_'); i >= 0 {
		return t[i+1:]
	}
	return t
}
