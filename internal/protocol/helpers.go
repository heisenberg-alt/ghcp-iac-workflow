package protocol

import (
	"fmt"
	"strings"
)

// PromptText extracts the user prompt from an AgentRequest.
// It checks the Prompt field first, then falls back to the last user message.
func PromptText(req AgentRequest) string {
	if req.Prompt != "" {
		return req.Prompt
	}
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" && req.Messages[i].Content != "" {
			return req.Messages[i].Content
		}
	}
	return ""
}

// MatchesAny returns true if msg contains any of the keywords.
func MatchesAny(msg string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}

// RequireIaC checks for IaC input and emits a message if missing.
// Returns true if IaC is present and has resources.
func RequireIaC(req AgentRequest, emit Emitter, domain string) bool {
	if req.IaC == nil || len(req.IaC.Resources) == 0 {
		emit.SendMessage(fmt.Sprintf("No IaC resources provided for %s analysis.\n", domain))
		return false
	}
	return true
}

// Finding represents a rule violation found during analysis.
type Finding struct {
	RuleID       string
	Category     string
	Severity     string
	Resource     string
	ResourceType string
	Message      string
	Remediation  string
}
