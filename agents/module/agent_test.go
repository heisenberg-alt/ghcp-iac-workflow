package module

import (
	"context"
	"strings"
	"testing"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

type recorder struct {
	messages []string
}

func (r *recorder) SendMessage(content string)               { r.messages = append(r.messages, content) }
func (r *recorder) SendReferences(_ []protocol.Reference)    {}
func (r *recorder) SendConfirmation(_ protocol.Confirmation) {}
func (r *recorder) SendError(msg string)                     { r.messages = append(r.messages, msg) }
func (r *recorder) SendDone()                                {}

func TestAgent_ID(t *testing.T) {
	if New().ID() != "module" {
		t.Error("expected ID = module")
	}
}

func TestAgent_Placeholder(t *testing.T) {
	a := New()
	rec := &recorder{}
	err := a.Handle(context.Background(), protocol.AgentRequest{}, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	if !strings.Contains(combined, "not yet implemented") {
		t.Error("expected placeholder message")
	}
}

func TestAgent_ImplementsAgent(t *testing.T) {
	var _ protocol.Agent = (*Agent)(nil)
}
