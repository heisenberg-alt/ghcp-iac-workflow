package cost

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
	if a.ID() != "cost" {
		t.Errorf("ID = %q, want cost", a.ID())
	}
}

func TestAgent_StorageCost(t *testing.T) {
	a := New()
	tfCode := `resource "azurerm_storage_account" "main" {
  name                     = "mainstorage"
  resource_group_name      = "rg"
  location                 = "eastus"
  account_tier             = "Standard"
  account_replication_type = "LRS"
}`
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "estimate cost:\n```hcl\n" + tfCode + "\n```"},
		},
	}
	host.ParseAndEnrich(&req)
	rec := &recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	if !strings.Contains(combined, "$") {
		t.Error("expected cost output with dollar sign")
	}
	if !strings.Contains(combined, "storage_account.main") {
		t.Error("expected resource name in output")
	}
}

func TestAgent_AKSCost(t *testing.T) {
	a := New()
	tfCode := `resource "azurerm_kubernetes_cluster" "aks" {
  name                = "myaks"
  location            = "eastus"
  resource_group_name = "rg"
  dns_prefix          = "myaks"
  default_node_pool {
    name       = "default"
    node_count = 3
    vm_size    = "Standard_D2s_v3"
  }
}`
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "estimate cost:\n```hcl\n" + tfCode + "\n```"},
		},
	}
	host.ParseAndEnrich(&req)
	rec := &recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.messages, "")
	if !strings.Contains(combined, "Monthly Cost") {
		t.Error("expected monthly cost header")
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
