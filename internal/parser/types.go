// Package parser provides shared Terraform and Bicep parsing for IaC analysis.
// It extracts resource definitions, properties, and structure from IaC code.
package parser

// IaCType represents the type of Infrastructure as Code.
type IaCType string

const (
	Terraform IaCType = "Terraform"
	Bicep     IaCType = "Bicep"
	Unknown   IaCType = "Unknown"
)

// Resource represents a parsed IaC resource.
type Resource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Line       int                    `json:"line"`
	RawBlock   string                 `json:"raw_block,omitempty"`
}
