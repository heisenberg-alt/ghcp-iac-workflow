package policy

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
	if a.ID() != "policy" {
		t.Errorf("ID = %q, want policy", a.ID())
	}
}

func TestAgent_Capabilities(t *testing.T) {
	a := New()
	caps := a.Capabilities()
	if !caps.NeedsIaCInput {
		t.Error("expected NeedsIaCInput = true")
	}
	if len(caps.Formats) != 2 {
		t.Errorf("expected 2 formats, got %d", len(caps.Formats))
	}
}

func TestAgent_InsecureStorage(t *testing.T) {
	a := New()
	tfCode := "resource \"azurerm_storage_account\" \"insecure\" {\n" +
		"  name                          = \"insecurestorage\"\n" +
		"  resource_group_name           = \"rg\"\n" +
		"  location                      = \"eastus\"\n" +
		"  account_tier                  = \"Standard\"\n" +
		"  account_replication_type      = \"LRS\"\n" +
		"  enable_https_traffic_only     = false\n" +
		"  min_tls_version               = \"TLS1_0\"\n" +
		"  allow_blob_public_access      = true\n" +
		"}"
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
	expectedRules := []string{"POL-001", "POL-003", "POL-004"}
	for _, ruleID := range expectedRules {
		if !strings.Contains(combined, ruleID) {
			t.Errorf("expected finding for %s", ruleID)
		}
	}
}

func TestAgent_SecureStorage(t *testing.T) {
	a := New()
	tfCode := "resource \"azurerm_storage_account\" \"secure\" {\n" +
		"  name                          = \"securestorage\"\n" +
		"  resource_group_name           = \"rg\"\n" +
		"  location                      = \"eastus\"\n" +
		"  account_tier                  = \"Standard\"\n" +
		"  account_replication_type      = \"LRS\"\n" +
		"  enable_https_traffic_only     = true\n" +
		"  min_tls_version               = \"TLS1_2\"\n" +
		"  allow_blob_public_access      = false\n" +
		"}"
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
		t.Error("secure storage should pass all policy checks")
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
