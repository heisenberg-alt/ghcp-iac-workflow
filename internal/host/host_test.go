package host

import (
	"context"
	"strings"
	"testing"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

type stubAgent struct {
	id   string
	meta protocol.AgentMetadata
	caps protocol.AgentCapabilities
	out  string
}

func (s *stubAgent) ID() string                               { return s.id }
func (s *stubAgent) Metadata() protocol.AgentMetadata         { return s.meta }
func (s *stubAgent) Capabilities() protocol.AgentCapabilities { return s.caps }
func (s *stubAgent) Handle(_ context.Context, _ protocol.AgentRequest, emit protocol.Emitter) error {
	emit.SendMessage(s.out)
	return nil
}

type recorder struct {
	messages      []string
	references    [][]protocol.Reference
	confirmations []protocol.Confirmation
	errors        []string
	doneCount     int
}

func (r *recorder) SendMessage(content string) { r.messages = append(r.messages, content) }
func (r *recorder) SendReferences(refs []protocol.Reference) {
	r.references = append(r.references, refs)
}
func (r *recorder) SendConfirmation(conf protocol.Confirmation) {
	r.confirmations = append(r.confirmations, conf)
}
func (r *recorder) SendError(msg string) { r.errors = append(r.errors, msg) }
func (r *recorder) SendDone()            { r.doneCount++ }

func TestRegistry_RegisterGetList(t *testing.T) {
	reg := NewRegistry()
	a1 := &stubAgent{id: "policy", meta: protocol.AgentMetadata{ID: "policy", Name: "Policy Analyzer"}}
	a2 := &stubAgent{id: "cost", meta: protocol.AgentMetadata{ID: "cost", Name: "Cost Estimator"}}
	reg.Register(a1)
	reg.Register(a2)

	got, ok := reg.Get("policy")
	if !ok || got.ID() != "policy" {
		t.Error("expected to find policy agent")
	}
	_, ok = reg.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent agent to not be found")
	}
	metas := reg.List()
	if len(metas) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(metas))
	}
}

func TestDispatcher_Dispatch(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubAgent{id: "test", out: "hello from test"})
	d := NewDispatcher(reg)
	rec := &recorder{}
	err := d.Dispatch(context.Background(), "test", protocol.AgentRequest{}, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rec.messages) != 1 || rec.messages[0] != "hello from test" {
		t.Errorf("unexpected messages: %v", rec.messages)
	}
}

func TestDispatcher_NotFound(t *testing.T) {
	reg := NewRegistry()
	d := NewDispatcher(reg)
	err := d.Dispatch(context.Background(), "missing", protocol.AgentRequest{}, &recorder{})
	if err == nil {
		t.Fatal("expected error for missing agent")
	}
}

func TestParseAndEnrich_Terraform(t *testing.T) {
	tfCode := "resource \"azurerm_storage_account\" \"test\" {\n" +
		"  name                      = \"teststorage\"\n" +
		"  resource_group_name       = \"rg\"\n" +
		"  location                  = \"eastus\"\n" +
		"  account_tier              = \"Standard\"\n" +
		"  account_replication_type  = \"LRS\"\n" +
		"}"
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "analyze:\n```hcl\n" + tfCode + "\n```"},
		},
	}
	ParseAndEnrich(&req)
	if req.IaC == nil {
		t.Fatal("expected IaC to be populated")
	}
	if req.IaC.Format != protocol.FormatTerraform {
		t.Errorf("format = %s, want terraform", req.IaC.Format)
	}
	if len(req.IaC.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(req.IaC.Resources))
	}
	if req.IaC.Resources[0].Type != "azurerm_storage_account" {
		t.Errorf("resource type = %s", req.IaC.Resources[0].Type)
	}
}

func TestParseAndEnrich_NoCode(t *testing.T) {
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "hello world"},
		},
	}
	ParseAndEnrich(&req)
	if req.IaC != nil {
		t.Error("expected IaC to be nil when no code present")
	}
}

func TestParseAndEnrich_UsesPromptFirst(t *testing.T) {
	tfCode := "resource \"azurerm_storage_account\" \"p\" { name = \"from-prompt\" }"
	req := protocol.AgentRequest{
		Prompt: "check:\n```hcl\n" + tfCode + "\n```",
		Messages: []protocol.Message{
			{Role: "user", Content: "no code here"},
		},
	}
	ParseAndEnrich(&req)
	if req.IaC == nil {
		t.Fatal("expected IaC from prompt")
	}
	if !strings.Contains(req.IaC.RawCode, "from-prompt") {
		t.Error("expected code from prompt field, not messages")
	}
}
