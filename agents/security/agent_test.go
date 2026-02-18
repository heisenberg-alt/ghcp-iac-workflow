package security

import (
	"context"
	"strings"
	"testing"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/host"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol/prototest"
)


func TestAgent_ID(t *testing.T) {
	a := New()
	if a.ID() != "security" {
		t.Errorf("ID = %q, want security", a.ID())
	}
}

func TestAgent_Capabilities(t *testing.T) {
	a := New()
	caps := a.Capabilities()
	if !caps.NeedsIaCInput {
		t.Error("expected NeedsIaCInput = true")
	}
	if !caps.NeedsRawCode {
		t.Error("expected NeedsRawCode = true")
	}
}

func TestAgent_NSGViolation(t *testing.T) {
	a := New()
	tfCode := `resource "azurerm_network_security_group" "open" {
  name                = "open-nsg"
  location            = "eastus"
  resource_group_name = "rg"

  security_rule {
    name                       = "AllowAll"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "*"
    destination_address_prefix = "*"
  }
}`
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "analyze:\n```hcl\n" + tfCode + "\n```"},
		},
	}
	host.ParseAndEnrich(&req)
	rec := &prototest.Recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.Messages, "")
	if !strings.Contains(combined, "SEC-005") {
		t.Error("expected SEC-005 for overly permissive NSG")
	}
}

func TestAgent_SecureStorage(t *testing.T) {
	a := New()
	tfCode := `resource "azurerm_storage_account" "secure" {
  name                     = "securestorage"
  resource_group_name      = "rg"
  location                 = "eastus"
  account_tier             = "Standard"
  account_replication_type = "LRS"
  customer_managed_key     = {}
}`
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "analyze:\n```hcl\n" + tfCode + "\n```"},
		},
	}
	host.ParseAndEnrich(&req)
	rec := &prototest.Recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.Messages, "")
	if !strings.Contains(combined, "passed") {
		t.Error("expected all security checks to pass for secure storage")
	}
}

func TestAgent_NoIaC(t *testing.T) {
	a := New()
	rec := &prototest.Recorder{}
	err := a.Handle(context.Background(), protocol.AgentRequest{}, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.Messages, "")
	if !strings.Contains(combined, "No IaC") {
		t.Error("expected no-IaC message")
	}
}

func TestAgent_ImplementsAgent(t *testing.T) {
	var _ protocol.Agent = (*Agent)(nil)
}
