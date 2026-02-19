package notification

import (
	"context"
	"strings"
	"testing"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol/prototest"
)

func TestAgent_ID(t *testing.T) {
	if New(false).ID() != "notification" {
		t.Error("expected ID = notification")
	}
}

func TestAgent_NotifyDisabled(t *testing.T) {
	a := New(false)
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "notify teams message: deployment complete"},
		},
	}
	rec := &prototest.Recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.Messages, "")
	if !strings.Contains(combined, "disabled") {
		t.Error("expected disabled message")
	}
	if !strings.Contains(combined, "teams") {
		t.Error("expected teams channel")
	}
}

func TestAgent_NotifySlack(t *testing.T) {
	a := New(false)
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "notify slack message: test"},
		},
	}
	rec := &prototest.Recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.Messages, "")
	if !strings.Contains(combined, "slack") {
		t.Error("expected slack channel")
	}
}

func TestAgent_ImplementsAgent(t *testing.T) {
	var _ protocol.Agent = (*Agent)(nil)
}
