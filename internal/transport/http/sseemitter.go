// Package httpx provides the HTTP transport layer for the agent-host architecture.
package httpx

import (
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/server"
)

// SSEEmitter adapts a server.SSEWriter to the protocol.Emitter interface.
type SSEEmitter struct {
	sse *server.SSEWriter
}

// NewSSEEmitter wraps an existing SSEWriter.
func NewSSEEmitter(sse *server.SSEWriter) *SSEEmitter {
	return &SSEEmitter{sse: sse}
}

// SendMessage sends a copilot_message event.
func (e *SSEEmitter) SendMessage(content string) {
	e.sse.SendMessage(content)
}

// SendReferences sends a copilot_references event.
func (e *SSEEmitter) SendReferences(refs []protocol.Reference) {
	srvRefs := make([]server.Reference, len(refs))
	for i, r := range refs {
		srvRefs[i] = server.Reference{Title: r.Title, URL: r.URL}
	}
	e.sse.SendReferences(srvRefs)
}

// SendConfirmation sends a copilot_confirmation event.
func (e *SSEEmitter) SendConfirmation(conf protocol.Confirmation) {
	e.sse.SendConfirmation(server.Confirmation{Title: conf.Title, Message: conf.Message})
}

// SendError sends an error message.
func (e *SSEEmitter) SendError(msg string) {
	e.sse.SendError(msg)
}

// SendDone sends the copilot_done event.
func (e *SSEEmitter) SendDone() {
	e.sse.SendDone()
}
