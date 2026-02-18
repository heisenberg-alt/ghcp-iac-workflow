package drift

import (
	"context"
	"strings"
	"testing"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/host"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol/prototest"
)


func TestAgent_ID(t *testing.T) {
	if New().ID() != "drift" {
		t.Error("expected ID = drift")
	}
}

func TestAgent_DriftDetected(t *testing.T) {
	a := New()
	tfCode := `resource "azurerm_storage_account" "bad" {
  name                       = "badstorage"
  resource_group_name        = "rg"
  location                   = "eastus"
  account_tier               = "Standard"
  account_replication_type   = "LRS"
  enable_https_traffic_only  = false
  min_tls_version            = "TLS1_0"
}`
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "check drift:\n```hcl\n" + tfCode + "\n```"},
		},
	}
	host.ParseAndEnrich(&req)
	rec := &prototest.Recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.Messages, "")
	if !strings.Contains(combined, "drift") {
		t.Error("expected drift detection output")
	}
	if !strings.Contains(combined, "min_tls_version") {
		t.Error("expected TLS drift")
	}
}

func TestAgent_NoDrift(t *testing.T) {
	a := New()
	tfCode := `resource "azurerm_storage_account" "good" {
  name                       = "goodstorage"
  resource_group_name        = "rg"
  location                   = "eastus"
  account_tier               = "Standard"
  account_replication_type   = "LRS"
  enable_https_traffic_only  = true
  min_tls_version            = "TLS1_2"
}`
	req := protocol.AgentRequest{
		Messages: []protocol.Message{
			{Role: "user", Content: "check drift:\n```hcl\n" + tfCode + "\n```"},
		},
	}
	host.ParseAndEnrich(&req)
	rec := &prototest.Recorder{}
	err := a.Handle(context.Background(), req, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(rec.Messages, "")
	if !strings.Contains(combined, "No drift") {
		t.Error("expected no drift message")
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
