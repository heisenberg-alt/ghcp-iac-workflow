package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/auth"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/config"
)

// mockHandler implements Handler for testing.
type mockHandler struct {
	called  bool
	lastReq AgentRequest
}

func (m *mockHandler) HandleAgent(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	m.called = true
	m.lastReq = req
	sse.SendMessage("test response")
}

func TestNewSSEWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	sse := NewSSEWriter(rr)

	if sse == nil {
		t.Fatal("NewSSEWriter should not be nil for httptest.ResponseRecorder")
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	cc := rr.Header().Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
}

func TestSSEWriter_SendMessage(t *testing.T) {
	rr := httptest.NewRecorder()
	sse := NewSSEWriter(rr)
	sse.SendMessage("hello world")

	body := rr.Body.String()
	if !strings.Contains(body, "event: copilot_message") {
		t.Error("SendMessage should write copilot_message event")
	}
	if !strings.Contains(body, "hello world") {
		t.Error("SendMessage should include the content")
	}
	if !strings.Contains(body, `"role":"assistant"`) {
		t.Error("SendMessage should include assistant role")
	}
}

func TestSSEWriter_SendDone(t *testing.T) {
	rr := httptest.NewRecorder()
	sse := NewSSEWriter(rr)
	sse.SendDone()

	body := rr.Body.String()
	if !strings.Contains(body, "event: copilot_done") {
		t.Error("SendDone should write copilot_done event")
	}
}

func TestSSEWriter_SendError(t *testing.T) {
	rr := httptest.NewRecorder()
	sse := NewSSEWriter(rr)
	sse.SendError("something failed")

	body := rr.Body.String()
	if !strings.Contains(body, "something failed") {
		t.Error("SendError should include error message")
	}
	if !strings.Contains(body, "Error") {
		t.Error("SendError should include Error label")
	}
}

func TestSSEWriter_SendReferences(t *testing.T) {
	rr := httptest.NewRecorder()
	sse := NewSSEWriter(rr)
	refs := []Reference{
		{Title: "Test Doc", URL: "https://example.com"},
	}
	sse.SendReferences(refs)

	body := rr.Body.String()
	if !strings.Contains(body, "event: copilot_references") {
		t.Error("SendReferences should write copilot_references event")
	}
	if !strings.Contains(body, "Test Doc") {
		t.Error("SendReferences should include reference title")
	}
}

func TestSSEWriter_SendConfirmation(t *testing.T) {
	rr := httptest.NewRecorder()
	sse := NewSSEWriter(rr)
	conf := Confirmation{
		Title:   "Deploy?",
		Message: "Ready to deploy to prod",
	}
	sse.SendConfirmation(conf)

	body := rr.Body.String()
	if !strings.Contains(body, "event: copilot_confirmation") {
		t.Error("SendConfirmation should write copilot_confirmation event")
	}
	if !strings.Contains(body, "Deploy?") {
		t.Error("SendConfirmation should include title")
	}
}

func TestAgentRequest_GetLastUserMessage(t *testing.T) {
	req := AgentRequest{
		Messages: []Message{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "response"},
			{Role: "user", Content: "second"},
		},
	}
	got := req.GetLastUserMessage()
	if got != "second" {
		t.Errorf("GetLastUserMessage = %q, want second", got)
	}
}

func TestAgentRequest_GetLastUserMessage_NoUser(t *testing.T) {
	req := AgentRequest{
		Messages: []Message{
			{Role: "assistant", Content: "hello"},
		},
	}
	got := req.GetLastUserMessage()
	if got != "" {
		t.Errorf("GetLastUserMessage with no user messages = %q, want empty", got)
	}
}

func TestAgentRequest_GetLastUserMessage_Empty(t *testing.T) {
	req := AgentRequest{}
	got := req.GetLastUserMessage()
	if got != "" {
		t.Errorf("GetLastUserMessage with no messages = %q, want empty", got)
	}
}

func TestAgentRequest_GetCodeFromReferences(t *testing.T) {
	req := AgentRequest{
		CopilotReferences: []CopilotReference{
			{
				Type: "file",
				ID:   "main.tf",
				Data: struct {
					Content  string `json:"content,omitempty"`
					Language string `json:"language,omitempty"`
				}{
					Content:  `resource "azurerm_storage_account" "ex" {}`,
					Language: "terraform",
				},
			},
		},
	}
	got := req.GetCodeFromReferences()
	if !strings.Contains(got, "azurerm_storage_account") {
		t.Errorf("GetCodeFromReferences = %q, should contain terraform code", got)
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	// Test WithGitHubToken and GitHubToken
	ctx = WithGitHubToken(ctx, "test-token")
	if got := GitHubToken(ctx); got != "test-token" {
		t.Errorf("GitHubToken = %q, want test-token", got)
	}

	// Test empty context
	if got := GitHubToken(context.Background()); got != "" {
		t.Errorf("GitHubToken on empty ctx = %q, want empty", got)
	}
	if got := RequestID(context.Background()); got != "" {
		t.Errorf("RequestID on empty ctx = %q, want empty", got)
	}
}

func TestServer_Health(t *testing.T) {
	cfg := &config.Config{
		Port:        "0",
		Environment: "dev",
		ModelName:   "gpt-4.1-mini",
		EnableLLM:   true,
	}
	handler := &mockHandler{}
	srv := New(cfg, handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Health status = %d, want 200", rr.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}
	if body["status"] != "healthy" {
		t.Errorf("Health status field = %v, want healthy", body["status"])
	}
	if body["service"] != "ghcp-iac" {
		t.Errorf("Health service = %v, want ghcp-iac", body["service"])
	}
}

func TestServer_AgentEndpoint_POST(t *testing.T) {
	cfg := &config.Config{
		Port:        "0",
		Environment: "dev",
		ModelName:   "gpt-4.1-mini",
	}
	handler := &mockHandler{}
	srv := New(cfg, handler)

	body := `{"messages":[{"role":"user","content":"analyze this code"}]}`
	req := httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(body))
	req.Header.Set("X-GitHub-Token", "test-token")
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Agent POST status = %d, want 200", rr.Code)
	}

	if !handler.called {
		t.Error("Handler should have been called")
	}
	if handler.lastReq.GetLastUserMessage() != "analyze this code" {
		t.Errorf("Handler got message = %q", handler.lastReq.GetLastUserMessage())
	}

	respBody := rr.Body.String()
	if !strings.Contains(respBody, "copilot_done") {
		t.Error("Response should contain copilot_done event")
	}
}

func TestServer_AgentEndpoint_GET(t *testing.T) {
	cfg := &config.Config{
		Port:        "0",
		Environment: "dev",
		ModelName:   "gpt-4.1-mini",
	}
	handler := &mockHandler{}
	srv := New(cfg, handler)

	req := httptest.NewRequest(http.MethodGet, "/agent", nil)
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("Agent GET status = %d, want 405", rr.Code)
	}
}

func TestServer_Middleware_RequestID(t *testing.T) {
	cfg := &config.Config{
		Port:        "0",
		Environment: "dev",
		ModelName:   "gpt-4.1-mini",
	}

	var capturedReqID string
	handler := &mockHandler{}
	srv := New(cfg, handler)

	// Override handler to capture request ID
	originalHandleAgent := srv.handler
	srv.handler = handlerFunc(func(ctx context.Context, req AgentRequest, sse *SSEWriter) {
		capturedReqID = RequestID(ctx)
		originalHandleAgent.HandleAgent(ctx, req, sse)
	})

	body := `{"messages":[{"role":"user","content":"test"}]}`
	req := httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(body))
	req.Header.Set("X-Request-ID", "custom-req-id")
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)

	if capturedReqID != "custom-req-id" {
		t.Errorf("RequestID = %q, want custom-req-id", capturedReqID)
	}
}

// handlerFunc adapts a function to the Handler interface.
type handlerFunc func(ctx context.Context, req AgentRequest, sse *SSEWriter)

func (f handlerFunc) HandleAgent(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	f(ctx, req, sse)
}

func TestServer_Middleware_TokenExtraction(t *testing.T) {
	cfg := &config.Config{
		Port:        "0",
		Environment: "dev",
		ModelName:   "gpt-4.1-mini",
	}

	var capturedToken string
	srv := New(cfg, handlerFunc(func(ctx context.Context, req AgentRequest, sse *SSEWriter) {
		capturedToken = GitHubToken(ctx)
		sse.SendMessage("ok")
	}))

	body := `{"messages":[{"role":"user","content":"test"}]}`
	req := httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(body))
	req.Header.Set("X-GitHub-Token", "ghu_abc123")
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)

	if capturedToken != "ghu_abc123" {
		t.Errorf("Token = %q, want ghu_abc123", capturedToken)
	}
}

func TestServer_SignatureVerification(t *testing.T) {
	secret := "test-webhook-secret"
	cfg := &config.Config{
		Port:          "0",
		Environment:   "prod",
		WebhookSecret: secret,
		ModelName:     "gpt-4.1",
	}
	handler := &mockHandler{}
	srv := New(cfg, handler)

	body := `{"messages":[{"role":"user","content":"test"}]}`

	// Valid signature
	validSig := auth.SignPayload([]byte(body), secret)

	req := httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", validSig)
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Valid signature: status = %d, want 200", rr.Code)
	}

	// Invalid signature
	req = httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256=0000000000000000000000000000000000000000000000000000000000000000")
	rr = httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Invalid signature: status = %d, want 401", rr.Code)
	}

	// Missing signature
	req = httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(body))
	rr = httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Missing signature: status = %d, want 401", rr.Code)
	}
}
