package testkit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/analyzer"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/config"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/costestimator"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/infraops"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/server"
)

// ── Analyzer Characterization ──

func TestAnalyzer_InsecureStorage_Findings(t *testing.T) {
	a := analyzer.New(nil, false)
	code := LoadFixture("insecure-storage.tf")
	req := BuildRequest(code)

	events := CaptureSSE(func(sse *server.SSEWriter) {
		a.Analyze(context.Background(), req, sse)
	})

	combined := AllMessages(events)

	if !strings.Contains(combined, "Terraform") {
		t.Error("expected detection message to mention Terraform")
	}
	if !strings.Contains(combined, "resource(s)") {
		t.Error("expected resource count message")
	}

	expectedRules := []string{"POL-001", "POL-003", "POL-004", "SEC-002", "NIST-SC7", "NIST-SC28"}
	for _, ruleID := range expectedRules {
		if !strings.Contains(combined, ruleID) {
			t.Errorf("expected finding for rule %s", ruleID)
		}
	}

	if !strings.Contains(combined, "Blast Radius") {
		t.Error("expected blast radius section")
	}

	refs := EventsOfType(events, "copilot_references")
	if len(refs) == 0 {
		t.Error("expected copilot_references event")
	}
}

func TestAnalyzer_SecureStorage_NoHighFindings(t *testing.T) {
	a := analyzer.New(nil, false)
	code := LoadFixture("minimal-secure-storage.tf")
	req := BuildRequest(code)

	events := CaptureSSE(func(sse *server.SSEWriter) {
		a.Analyze(context.Background(), req, sse)
	})

	combined := AllMessages(events)

	if !strings.Contains(combined, "Terraform") {
		t.Error("expected detection message")
	}

	policyFails := []string{"POL-001", "POL-003", "POL-004"}
	for _, ruleID := range policyFails {
		if strings.Contains(combined, ruleID) {
			t.Errorf("secure storage should not trigger %s", ruleID)
		}
	}
}

func TestAnalyzer_InsecureBicep_Findings(t *testing.T) {
	a := analyzer.New(nil, false)
	code := LoadFixture("insecure-storage.bicep")
	req := BuildRequest(code)

	events := CaptureSSE(func(sse *server.SSEWriter) {
		a.Analyze(context.Background(), req, sse)
	})

	combined := AllMessages(events)

	if !strings.Contains(combined, "Bicep") {
		t.Error("expected detection message for Bicep")
	}

	expectedRules := []string{"POL-001", "POL-003", "POL-004"}
	for _, ruleID := range expectedRules {
		if !strings.Contains(combined, ruleID) {
			t.Errorf("Bicep insecure storage should trigger %s", ruleID)
		}
	}
}

func TestAnalyzer_MixedResources_BroadCoverage(t *testing.T) {
	a := analyzer.New(nil, false)
	code := LoadFixture("mixed-resources.tf")
	req := BuildRequest(code)

	events := CaptureSSE(func(sse *server.SSEWriter) {
		a.Analyze(context.Background(), req, sse)
	})

	combined := AllMessages(events)

	if !strings.Contains(combined, "3") || !strings.Contains(combined, "resource(s)") {
		t.Error("expected 3 resources found message")
	}
	if !strings.Contains(combined, "POL-001") {
		t.Error("expected POL-001 for insecure storage in mixed")
	}
	if !strings.Contains(combined, "POL-005") {
		t.Error("expected POL-005 for key vault soft delete")
	}
	if !strings.Contains(combined, "POL-006") {
		t.Error("expected POL-006 for key vault purge protection")
	}
	if !strings.Contains(combined, "SEC-005") {
		t.Error("expected SEC-005 for open NSG")
	}
	if !strings.Contains(combined, "Blast Radius") {
		t.Error("expected blast radius section")
	}
}

func TestAnalyzer_NoCode_ShowsUsage(t *testing.T) {
	a := analyzer.New(nil, false)
	text := LoadFixture("no-iac-input.txt")
	req := BuildPlainRequest(text)

	events := CaptureSSE(func(sse *server.SSEWriter) {
		a.Analyze(context.Background(), req, sse)
	})

	combined := AllMessages(events)
	if !strings.Contains(combined, "No IaC code detected") {
		t.Error("expected usage message when no code found")
	}
}

// ── Cost Estimator Characterization ──

func TestCostEstimator_AKSCluster(t *testing.T) {
	e := costestimator.New(nil, false, false)
	code := LoadFixture("aks-cluster.tf")
	req := BuildRequest(code)

	events := CaptureSSE(func(sse *server.SSEWriter) {
		e.Estimate(context.Background(), req, sse)
	})

	combined := AllMessages(events)

	if !strings.Contains(combined, "Terraform") {
		t.Error("expected Terraform detection")
	}
	if !strings.Contains(combined, "resource(s)") {
		t.Error("expected resource count")
	}
	if !strings.Contains(combined, "$") {
		t.Error("expected cost estimate with $ symbol")
	}

	refs := EventsOfType(events, "copilot_references")
	if len(refs) == 0 {
		t.Error("expected copilot_references event for pricing")
	}
}

func TestCostEstimator_NoCode_ShowsUsage(t *testing.T) {
	e := costestimator.New(nil, false, false)
	req := BuildPlainRequest("how much does azure cost?")

	events := CaptureSSE(func(sse *server.SSEWriter) {
		e.Estimate(context.Background(), req, sse)
	})

	combined := AllMessages(events)
	if !strings.Contains(combined, "No IaC code") {
		t.Error("expected usage message for cost estimator")
	}
}

// ── InfraOps Characterization ──

func TestInfraOps_Drift_WithCode(t *testing.T) {
	ops := infraops.New(nil, false, infraops.Config{})
	code := LoadFixture("insecure-storage.tf")
	req := BuildRequestWithPrompt("check for drift", code)

	events := CaptureSSE(func(sse *server.SSEWriter) {
		ops.Handle(context.Background(), req, sse)
	})

	combined := AllMessages(events)
	if !strings.Contains(combined, "Drift Detection") {
		t.Error("expected drift detection heading")
	}
}

func TestInfraOps_Deploy(t *testing.T) {
	ops := infraops.New(nil, false, infraops.Config{})
	req := BuildPlainRequest("deploy to dev")

	events := CaptureSSE(func(sse *server.SSEWriter) {
		ops.Handle(context.Background(), req, sse)
	})

	combined := AllMessages(events)
	if !strings.Contains(combined, "dev") {
		t.Error("expected deploy response mentioning dev")
	}
}

func TestInfraOps_Status(t *testing.T) {
	ops := infraops.New(nil, false, infraops.Config{})
	req := BuildPlainRequest("show environment status")

	events := CaptureSSE(func(sse *server.SSEWriter) {
		ops.Handle(context.Background(), req, sse)
	})

	combined := AllMessages(events)
	if !strings.Contains(combined, "Environment") {
		t.Error("expected environment status")
	}
}

func TestInfraOps_Usage(t *testing.T) {
	ops := infraops.New(nil, false, infraops.Config{})
	req := BuildPlainRequest("hello")

	events := CaptureSSE(func(sse *server.SSEWriter) {
		ops.Handle(context.Background(), req, sse)
	})

	combined := AllMessages(events)
	if !strings.Contains(combined, "Infrastructure Operations") {
		t.Error("expected ops usage message")
	}
}

// ── Health Endpoint Characterization ──

func TestHealth_ResponseShape(t *testing.T) {
	cfg := &config.Config{
		Port:        "0",
		Environment: "dev",
		ModelName:   "gpt-4.1-mini",
		EnableLLM:   true,
	}
	handler := handlerFunc(func(_ context.Context, _ server.AgentRequest, _ *server.SSEWriter) {})
	srv := server.New(cfg, handler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", rr.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode health: %v", err)
	}

	requiredKeys := []string{"status", "service", "environment", "model", "llm_enabled"}
	for _, key := range requiredKeys {
		if _, ok := body[key]; !ok {
			t.Errorf("health response missing key %q", key)
		}
	}
	if body["status"] != "healthy" {
		t.Errorf("status = %v, want healthy", body["status"])
	}
	if body["service"] != "ghcp-iac" {
		t.Errorf("service = %v, want ghcp-iac", body["service"])
	}
}

// ── Full SSE Flow Characterization ──

func TestFullFlow_AnalyzeSSEShape(t *testing.T) {
	a := analyzer.New(nil, false)
	code := LoadFixture("insecure-storage.tf")
	req := BuildRequest(code)

	events := CaptureSSE(func(sse *server.SSEWriter) {
		a.Analyze(context.Background(), req, sse)
	})

	msgs := EventsOfType(events, "copilot_message")
	if len(msgs) == 0 {
		t.Fatal("expected copilot_message events")
	}

	for i, e := range msgs {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(e.Data), &parsed); err != nil {
			t.Errorf("copilot_message[%d] is not valid JSON: %v", i, err)
		}
		if _, ok := parsed["choices"]; !ok {
			t.Errorf("copilot_message[%d] missing choices", i)
		}
	}

	refs := EventsOfType(events, "copilot_references")
	if len(refs) == 0 {
		t.Error("expected copilot_references event")
	}

	for _, r := range refs {
		var parsed []interface{}
		if err := json.Unmarshal([]byte(r.Data), &parsed); err != nil {
			t.Errorf("copilot_references is not valid JSON array: %v", err)
		}
	}
}

// handlerFunc adapts a function to the server.Handler interface.
type handlerFunc func(ctx context.Context, req server.AgentRequest, sse *server.SSEWriter)

func (f handlerFunc) HandleAgent(ctx context.Context, req server.AgentRequest, sse *server.SSEWriter) {
	f(ctx, req, sse)
}
