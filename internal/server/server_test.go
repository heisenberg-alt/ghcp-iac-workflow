package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewSSEWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	sse := NewSSEWriter(rr)

	if sse == nil {
		t.Fatal("NewSSEWriter should not be nil for httptest.ResponseRecorder")
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	cc := rr.Header().Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
}

func TestSSEWriter_SendMessage(t *testing.T) {
	rr := httptest.NewRecorder()
	sse := NewSSEWriter(rr)
	sse.SendMessage("hello world")

	body := rr.Body.String()
	if !strings.Contains(body, "event: copilot_message") {
		t.Error("SendMessage should write copilot_message event")
	}
	if !strings.Contains(body, "hello world") {
		t.Error("SendMessage should include the content")
	}
	if !strings.Contains(body, `"role":"assistant"`) {
		t.Error("SendMessage should include assistant role")
	}
}

func TestSSEWriter_SendDone(t *testing.T) {
	rr := httptest.NewRecorder()
	sse := NewSSEWriter(rr)
	sse.SendDone()

	body := rr.Body.String()
	if !strings.Contains(body, "event: copilot_done") {
		t.Error("SendDone should write copilot_done event")
	}
}

func TestSSEWriter_SendError(t *testing.T) {
	rr := httptest.NewRecorder()
	sse := NewSSEWriter(rr)
	sse.SendError("something failed")

	body := rr.Body.String()
	if !strings.Contains(body, "something failed") {
		t.Error("SendError should include error message")
	}
	if !strings.Contains(body, "Error") {
		t.Error("SendError should include Error label")
	}
}

func TestSSEWriter_SendReferences(t *testing.T) {
	rr := httptest.NewRecorder()
	sse := NewSSEWriter(rr)
	refs := []Reference{
		{Title: "Test Doc", URL: "https://example.com"},
	}
	sse.SendReferences(refs)

	body := rr.Body.String()
	if !strings.Contains(body, "event: copilot_references") {
		t.Error("SendReferences should write copilot_references event")
	}
	if !strings.Contains(body, "Test Doc") {
		t.Error("SendReferences should include reference title")
	}
}

func TestSSEWriter_SendConfirmation(t *testing.T) {
	rr := httptest.NewRecorder()
	sse := NewSSEWriter(rr)
	conf := Confirmation{
		Title:   "Deploy?",
		Message: "Ready to deploy to prod",
	}
	sse.SendConfirmation(conf)

	body := rr.Body.String()
	if !strings.Contains(body, "event: copilot_confirmation") {
		t.Error("SendConfirmation should write copilot_confirmation event")
	}
	if !strings.Contains(body, "Deploy?") {
		t.Error("SendConfirmation should include title")
	}
}

func TestAgentRequest_GetLastUserMessage(t *testing.T) {
	req := AgentRequest{
		Messages: []Message{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "response"},
			{Role: "user", Content: "second"},
		},
	}
	got := req.GetLastUserMessage()
	if got != "second" {
		t.Errorf("GetLastUserMessage = %q, want second", got)
	}
}

func TestAgentRequest_GetLastUserMessage_NoUser(t *testing.T) {
	req := AgentRequest{
		Messages: []Message{
			{Role: "assistant", Content: "hello"},
		},
	}
	got := req.GetLastUserMessage()
	if got != "" {
		t.Errorf("GetLastUserMessage with no user messages = %q, want empty", got)
	}
}

func TestAgentRequest_GetLastUserMessage_Empty(t *testing.T) {
	req := AgentRequest{}
	got := req.GetLastUserMessage()
	if got != "" {
		t.Errorf("GetLastUserMessage with no messages = %q, want empty", got)
	}
}

func TestAgentRequest_GetCodeFromReferences(t *testing.T) {
	req := AgentRequest{
		CopilotReferences: []CopilotReference{
			{
				Type: "file",
				ID:   "main.tf",
				Data: struct {
					Content  string `json:"content,omitempty"`
					Language string `json:"language,omitempty"`
				}{
					Content:  `resource "azurerm_storage_account" "ex" {}`,
					Language: "terraform",
				},
			},
		},
	}
	got := req.GetCodeFromReferences()
	if !strings.Contains(got, "azurerm_storage_account") {
		t.Errorf("GetCodeFromReferences = %q, should contain terraform code", got)
	}
}
