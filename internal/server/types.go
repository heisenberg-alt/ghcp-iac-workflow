package server

// AgentRequest represents the incoming request from GitHub Copilot.
type AgentRequest struct {
	Messages          []Message          `json:"messages"`
	CopilotReferences []CopilotReference `json:"copilot_references,omitempty"`
}

// Message represents a chat message in the Copilot conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// CopilotReference represents a reference from the user's editor context.
type CopilotReference struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Data struct {
		Content  string `json:"content,omitempty"`
		Language string `json:"language,omitempty"`
	} `json:"data,omitempty"`
}

// Reference represents a link reference sent back to the user.
type Reference struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// Confirmation represents a confirmation dialog sent to the user.
type Confirmation struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// GetLastUserMessage extracts the most recent user message from the request.
func (r *AgentRequest) GetLastUserMessage() string {
	for i := len(r.Messages) - 1; i >= 0; i-- {
		if r.Messages[i].Role == "user" {
			return r.Messages[i].Content
		}
	}
	return ""
}

// GetCodeFromReferences extracts code content from copilot references.
func (r *AgentRequest) GetCodeFromReferences() string {
	for _, ref := range r.CopilotReferences {
		if ref.Data.Content != "" {
			return ref.Data.Content
		}
	}
	return ""
}
