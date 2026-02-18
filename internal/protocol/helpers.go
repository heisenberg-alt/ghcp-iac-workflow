package protocol

import "strings"

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

// Finding represents a rule violation found during analysis.
type Finding struct {
	RuleID       string
	Severity     string
	Resource     string
	ResourceType string
	Message      string
	Remediation  string
}
