package deploy

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
	if New().ID() != "deploy" {
		t.Error("expected ID = deploy")
	}
}

func TestAgent_DeployToStaging(t *testing.T) {
	a := New()
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "deploy to staging"},
		},
	}
	rec := &recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	if !strings.Contains(combined, "staging") {
		t.Error("expected staging in output")
	}
	if !strings.Contains(combined, "promoted") || !strings.Contains(combined, "Successfully") {
		t.Error("expected successful promotion message")
	}
}

func TestAgent_DeployToProdRequiresApproval(t *testing.T) {
	a := New()
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "deploy to production"},
		},
	}
	rec := &recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	if !strings.Contains(combined, "manual approval") {
		t.Error("expected manual approval message for prod")
	}
}

func TestAgent_Status(t *testing.T) {
	a := New()
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "environment status"},
		},
	}
	rec := &recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	if !strings.Contains(combined, "dev") || !strings.Contains(combined, "staging") || !strings.Contains(combined, "prod") {
		t.Error("expected all environments in status")
	}
}

func TestAgent_ImplementsAgent(t *testing.T) {
	var _ protocol.Agent = (*Agent)(nil)
}
