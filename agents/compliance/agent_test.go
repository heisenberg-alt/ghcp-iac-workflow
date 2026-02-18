package compliance

import (
	"context"
	"strings"
	"testing"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/host"
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
	a := New()
	if a.ID() != "compliance" {
		t.Errorf("ID = %q, want compliance", a.ID())
	}
}

func TestAgent_StorageNoNetworkRules(t *testing.T) {
	a := New()
	tfCode := `resource "azurerm_storage_account" "open" {
  name                     = "openstorage"
  resource_group_name      = "rg"
  location                 = "eastus"
  account_tier             = "Standard"
  account_replication_type = "LRS"
}`
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "analyze:\n```hcl\n" + tfCode + "\n```"},
		},
	}
	host.ParseAndEnrich(&req)
	rec := &recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	if !strings.Contains(combined, "NIST-SC7") {
		t.Error("expected NIST-SC7 finding for missing network rules")
	}
	if !strings.Contains(combined, "NIST-SC28") {
		t.Error("expected NIST-SC28 finding for missing infrastructure encryption")
	}
}

func TestAgent_NoIaC(t *testing.T) {
	a := New()
	rec := &recorder{}
	err := a.Handle(context.Background(), protocol.AgentRequest{}, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	if !strings.Contains(combined, "No IaC") {
		t.Error("expected no-IaC message")
	}
}

func TestAgent_ImplementsAgent(t *testing.T) {
	var _ protocol.Agent = (*Agent)(nil)
}
