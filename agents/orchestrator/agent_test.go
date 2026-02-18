package orchestrator

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

// stubAgent implements protocol.Agent for testing.
type stubAgent struct {
	id     string
	output string
}

func (s *stubAgent) ID() string                               { return s.id }
func (s *stubAgent) Metadata() protocol.AgentMetadata         { return protocol.AgentMetadata{ID: s.id} }
func (s *stubAgent) Capabilities() protocol.AgentCapabilities { return protocol.AgentCapabilities{} }
func (s *stubAgent) Handle(_ context.Context, _ protocol.AgentRequest, emit protocol.Emitter) error {
	emit.SendMessage(s.output)
	return nil
}

func stubLookup(agents ...protocol.Agent) AgentLookup {
	m := make(map[string]protocol.Agent)
	for _, a := range agents {
		m[a.ID()] = a
	}
	return func(id string) (protocol.Agent, bool) {
		a, ok := m[id]
		return a, ok
	}
}

func TestAgent_ID(t *testing.T) {
	a := New(stubLookup())
	if a.ID() != "orchestrator" {
		t.Error("expected ID = orchestrator")
	}
}

func TestAgent_FullAnalysis(t *testing.T) {
	lookup := stubLookup(
		&stubAgent{id: "policy", output: "[policy-output]"},
		&stubAgent{id: "security", output: "[security-output]"},
		&stubAgent{id: "compliance", output: "[compliance-output]"},
		&stubAgent{id: "impact", output: "[impact-output]"},
	)
	a := New(lookup)
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "analyze this code:\n```hcl\nresource \"azurerm_storage_account\" \"t\" {}\n```"},
		},
	}
	rec := &recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	for _, expect := range []string{"[policy-output]", "[security-output]", "[compliance-output]", "[impact-output]"} {
		if !strings.Contains(combined, expect) {
			t.Errorf("missing %s in output", expect)
		}
	}
}

func TestAgent_CostIntent(t *testing.T) {
	lookup := stubLookup(
		&stubAgent{id: "cost", output: "[cost-output]"},
		&stubAgent{id: "policy", output: "[policy-output]"},
	)
	a := New(lookup)
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "estimate the cost of this infrastructure"},
		},
	}
	rec := &recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	if !strings.Contains(combined, "[cost-output]") {
		t.Error("expected cost agent to run")
	}
	if strings.Contains(combined, "[policy-output]") {
		t.Error("policy agent should not run for cost intent")
	}
}

func TestAgent_OpsIntent(t *testing.T) {
	lookup := stubLookup(
		&stubAgent{id: "deploy", output: "[deploy-output]"},
		&stubAgent{id: "drift", output: "[drift-output]"},
		&stubAgent{id: "notification", output: "[notify-output]"},
	)
	a := New(lookup)
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "deploy to staging environment"},
		},
	}
	rec := &recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	if !strings.Contains(combined, "[deploy-output]") {
		t.Error("expected deploy agent to run")
	}
}

func TestAgent_HelpIntent(t *testing.T) {
	a := New(stubLookup())
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "help me understand what you can do"},
		},
	}
	rec := &recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	if !strings.Contains(combined, "Available commands") {
		t.Error("expected help message")
	}
}

func TestAgent_MissingAgent(t *testing.T) {
	// Register only policy — security, compliance, impact are missing
	lookup := stubLookup(&stubAgent{id: "policy", output: "[policy-output]"})
	a := New(lookup)
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "analyze:\n```hcl\nresource \"x\" \"y\" {}\n```"},
		},
	}
	rec := &recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	if !strings.Contains(combined, "[policy-output]") {
		t.Error("expected policy agent to run")
	}
	if !strings.Contains(combined, "not registered") {
		t.Error("expected 'not registered' message for missing agents")
	}
}

func TestClassifyKeywords(t *testing.T) {
	tests := []struct {
		input  string
		expect Intent
	}{
		{"scan my terraform code", IntentAnalyze},
		{"estimate cost please", IntentCost},
		{"deploy to production", IntentOps},
		{"detect drift in my environment", IntentOps},
		{"agent status", IntentStatus},
		{"help me", IntentHelp},
		{"```hcl\nresource \"x\" \"y\" {}\n```", IntentAnalyze},
	}
	for _, tt := range tests {
		got := classifyKeywords(tt.input)
		if got != tt.expect {
			t.Errorf("classifyKeywords(%q) = %q, want %q", tt.input, got, tt.expect)
		}
	}
}

func TestAgent_ImplementsAgent(t *testing.T) {
	var _ protocol.Agent = (*Agent)(nil)
}
