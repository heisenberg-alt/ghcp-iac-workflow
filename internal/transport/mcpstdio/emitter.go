package mcpstdio

import (
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// StdioEmitter collects agent output into a string buffer for MCP responses.
type StdioEmitter struct {
	buf strings.Builder
}

func (e *StdioEmitter) SendMessage(content string)               { e.buf.WriteString(content) }
func (e *StdioEmitter) SendReferences(_ []protocol.Reference)    {}
func (e *StdioEmitter) SendConfirmation(_ protocol.Confirmation) {}
func (e *StdioEmitter) SendError(msg string)                     { e.buf.WriteString("Error: " + msg + "\n") }
func (e *StdioEmitter) SendDone()                                {}

// Content returns the collected output.
func (e *StdioEmitter) Content() string { return e.buf.String() }
