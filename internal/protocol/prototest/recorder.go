// Package prototest provides shared test utilities for the protocol package.
package prototest

import "github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"

// Recorder is a test double for protocol.Emitter that records all messages.
type Recorder struct {
	Messages []string
}

func (r *Recorder) SendMessage(content string)               { r.Messages = append(r.Messages, content) }
func (r *Recorder) SendReferences(_ []protocol.Reference)    {}
func (r *Recorder) SendConfirmation(_ protocol.Confirmation) {}
func (r *Recorder) SendError(msg string)                     { r.Messages = append(r.Messages, msg) }
func (r *Recorder) SendDone()                                {}
