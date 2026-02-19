package server

import "github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"

// AgentRequest represents the incoming request from GitHub Copilot.
// Used for JSON decoding at the HTTP boundary.
type AgentRequest struct {
	Messages []protocol.Message `json:"messages"`
}
