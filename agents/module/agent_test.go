package module

import (
	"context"
	"strings"
	"testing"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol/prototest"
)


func TestAgent_ID(t *testing.T) {
	if New().ID() != "module" {
		t.Error("expected ID = module")
	}
}

func TestAgent_Placeholder(t *testing.T) {
	a := New()
	rec := &prototest.Recorder{}
	err := a.Handle(context.Background(), protocol.AgentRequest{}, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.Messages, "")
	if !strings.Contains(combined, "not yet implemented") {
		t.Error("expected placeholder message")
	}
}

func TestAgent_ImplementsAgent(t *testing.T) {
	var _ protocol.Agent = (*Agent)(nil)
}
