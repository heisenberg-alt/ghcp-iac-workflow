// Package protocol defines shared types for the agent-host architecture.
// It has no dependencies on other internal packages.
package protocol

// SourceFormat identifies the IaC language.
type SourceFormat string

const (
	FormatTerraform SourceFormat = "terraform"
	FormatBicep     SourceFormat = "bicep"
	FormatUnknown   SourceFormat = "unknown"
)

// Resource represents a parsed IaC resource.
type Resource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Line       int                    `json:"line"`
	RawBlock   string                 `json:"raw_block"`
}

// IaCInput holds parsed IaC data attached to a request by the host.
type IaCInput struct {
	Format    SourceFormat `json:"format"`
	RawCode   string       `json:"raw_code"`
	Resources []Resource   `json:"resources"`
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Reference is a link sent to the user.
type Reference struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// Confirmation is a confirmation prompt sent to the user.
type Confirmation struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// AgentRequest is the request passed to an Agent's Handle method.
type AgentRequest struct {
	Prompt     string            `json:"prompt"`
	Messages   []Message         `json:"messages"`
	References []Reference       `json:"references"`
	IaC        *IaCInput         `json:"iac,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// AgentMetadata describes an agent for discovery/listing.
type AgentMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// AgentCapabilities declares what inputs an agent needs.
type AgentCapabilities struct {
	Formats           []SourceFormat `json:"formats"`
	NeedsIaCInput     bool           `json:"needs_iac_input"`
	NeedsRawCode      bool           `json:"needs_raw_code"`
	NeedsFileContents bool           `json:"needs_file_contents"`
}
