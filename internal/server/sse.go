package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// SSEWriter writes Server-Sent Events in the Copilot Extension protocol format.
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter creates a new SSE writer from an HTTP response writer.
// Returns nil if the response writer does not support flushing.
func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	return &SSEWriter{w: w, flusher: flusher}
}

// SendMessage sends a copilot_message event with content.
func (s *SSEWriter) SendMessage(content string) {
	s.sendEvent("copilot_message", map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]string{
					"role":    "assistant",
					"content": content,
				},
			},
		},
	})
}

// SendReferences sends a copilot_references event with links.
func (s *SSEWriter) SendReferences(refs []protocol.Reference) {
	s.sendEvent("copilot_references", refs)
}

// SendConfirmation sends a copilot_confirmation event.
func (s *SSEWriter) SendConfirmation(conf protocol.Confirmation) {
	s.sendEvent("copilot_confirmation", conf)
}

// SendError sends an error message.
func (s *SSEWriter) SendError(msg string) {
	s.SendMessage(fmt.Sprintf("❌ **Error:** %s\n", msg))
}

// SendDone sends the copilot_done event marking end of stream.
func (s *SSEWriter) SendDone() {
	fmt.Fprintf(s.w, "event: copilot_done\ndata: {}\n\n")
	s.flusher.Flush()
}

func (s *SSEWriter) sendEvent(event string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("SSE marshal error for event %s: %v", event, err)
		return
	}
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, jsonData)
	s.flusher.Flush()
}
