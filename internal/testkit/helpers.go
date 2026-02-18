// Package testkit provides shared test helpers, fixture loading, and SSE capture
// utilities for characterization and integration testing.
package testkit

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/server"
)

// SSEEvent represents a parsed Server-Sent Event.
type SSEEvent struct {
	Type string // e.g. "copilot_message", "copilot_references", "copilot_done"
	Data string // raw JSON data
}

// scenariosDir returns the absolute path to the scenarios directory.
func scenariosDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "scenarios")
}

// LoadFixture reads a fixture file from the scenarios directory.
func LoadFixture(name string) string {
	data, err := os.ReadFile(filepath.Join(scenariosDir(), name))
	if err != nil {
		panic("testkit: could not load fixture " + name + ": " + err.Error())
	}
	return string(data)
}

// BuildRequest wraps IaC code in a server.AgentRequest with a user message.
func BuildRequest(code string) server.AgentRequest {
	return server.AgentRequest{
		Messages: []server.Message{
			{Role: "user", Content: "analyze this:\n```hcl\n" + code + "\n```"},
		},
	}
}

// BuildRequestWithPrompt builds a request with a custom prompt containing IaC code.
func BuildRequestWithPrompt(prompt, code string) server.AgentRequest {
	content := prompt
	if code != "" {
		content += "\n```hcl\n" + code + "\n```"
	}
	return server.AgentRequest{
		Messages: []server.Message{
			{Role: "user", Content: content},
		},
	}
}

// BuildPlainRequest builds a request with plain text (no code).
func BuildPlainRequest(text string) server.AgentRequest {
	return server.AgentRequest{
		Messages: []server.Message{
			{Role: "user", Content: text},
		},
	}
}

// CaptureSSE runs a handler function that writes SSE events, captures the raw
// output, and parses it into structured SSEEvent slices.
func CaptureSSE(fn func(sse *server.SSEWriter)) []SSEEvent {
	rec := httptest.NewRecorder()
	sse := server.NewSSEWriter(rec)
	if sse == nil {
		panic("testkit: httptest.ResponseRecorder does not support flushing")
	}
	fn(sse)
	return ParseSSE(rec.Body.String())
}

// ParseSSE parses raw SSE text into structured events.
func ParseSSE(raw string) []SSEEvent {
	var events []SSEEvent
	lines := strings.Split(raw, "\n")
	var currentEvent string
	for _, line := range lines {
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			events = append(events, SSEEvent{Type: currentEvent, Data: data})
			currentEvent = ""
		}
	}
	return events
}

// MessageContent extracts the text content from a copilot_message SSE event's JSON data.
func MessageContent(event SSEEvent) string {
	var msg struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(event.Data), &msg); err != nil {
		return ""
	}
	if len(msg.Choices) > 0 {
		return msg.Choices[0].Delta.Content
	}
	return ""
}

// AllMessages concatenates all copilot_message content from a slice of SSE events.
func AllMessages(events []SSEEvent) string {
	var sb strings.Builder
	for _, e := range events {
		if e.Type == "copilot_message" {
			sb.WriteString(MessageContent(e))
		}
	}
	return sb.String()
}

// EventsOfType filters events to only those matching the given type.
func EventsOfType(events []SSEEvent, eventType string) []SSEEvent {
	var filtered []SSEEvent
	for _, e := range events {
		if e.Type == eventType {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// ContainsMessage checks if any copilot_message event contains the given substring.
func ContainsMessage(events []SSEEvent, substr string) bool {
	return strings.Contains(AllMessages(events), substr)
}
