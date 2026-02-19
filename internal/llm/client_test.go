package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://example.com", "gpt-4", 1000, 30*time.Second)
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.endpoint != "https://example.com" {
		t.Errorf("endpoint = %q, want https://example.com", c.endpoint)
	}
	if c.model != "gpt-4" {
		t.Errorf("model = %q, want gpt-4", c.model)
	}
	if c.maxTokens != 1000 {
		t.Errorf("maxTokens = %d, want 1000", c.maxTokens)
	}
}

func TestComplete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var req ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("expected model test-model, got %s", req.Model)
		}
		if req.Stream {
			t.Error("expected stream=false")
		}

		resp := ChatResponse{
			Choices: []struct {
				Message ChatMessage `json:"message"`
				Delta   struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message:      ChatMessage{Role: RoleAssistant, Content: "Hello from LLM"},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-model", 100, 5*time.Second)
	result, err := c.Complete(context.Background(), "test-token", "You are helpful", []ChatMessage{
		{Role: RoleUser, Content: "Hi"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello from LLM" {
		t.Errorf("result = %q, want 'Hello from LLM'", result)
	}
}

func TestComplete_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-model", 100, 5*time.Second)
	_, err := c.Complete(context.Background(), "token", "", []ChatMessage{
		{Role: RoleUser, Content: "Hi"},
	})

	if err == nil {
		t.Error("expected error for 429 response")
	}
}

func TestComplete_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatResponse{Choices: nil})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-model", 100, 5*time.Second)
	_, err := c.Complete(context.Background(), "token", "", []ChatMessage{
		{Role: RoleUser, Content: "Hi"},
	})

	if err == nil {
		t.Error("expected error for empty choices")
	}
}

func TestComplete_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		json.NewEncoder(w).Encode(ChatResponse{})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-model", 100, 5*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := c.Complete(ctx, "token", "", []ChatMessage{
		{Role: RoleUser, Content: "Hi"},
	})
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestComplete_WithSystemPrompt(t *testing.T) {
	var receivedMessages []ChatMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedMessages = req.Messages

		resp := ChatResponse{
			Choices: []struct {
				Message ChatMessage `json:"message"`
				Delta   struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			}{
				{Message: ChatMessage{Role: RoleAssistant, Content: "response"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-model", 100, 5*time.Second)
	_, err := c.Complete(context.Background(), "token", "You are a helpful assistant", []ChatMessage{
		{Role: RoleUser, Content: "Hi"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedMessages) != 2 {
		t.Errorf("expected 2 messages (system + user), got %d", len(receivedMessages))
	}
	if receivedMessages[0].Role != RoleSystem {
		t.Errorf("first message should be system, got %s", receivedMessages[0].Role)
	}
	if receivedMessages[0].Content != "You are a helpful assistant" {
		t.Errorf("system prompt not passed correctly")
	}
}

func TestStream_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)
		if !req.Stream {
			t.Error("expected stream=true")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" \"}}]}\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"World\"}}]}\n"))
		w.Write([]byte("data: [DONE]\n"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-model", 100, 5*time.Second)
	contentCh, errCh := c.Stream(context.Background(), "token", "", []ChatMessage{
		{Role: RoleUser, Content: "Hi"},
	})

	var result string
	for chunk := range contentCh {
		result += chunk
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}
	if result != "Hello World" {
		t.Errorf("streamed result = %q, want 'Hello World'", result)
	}
}

func TestStream_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-model", 100, 5*time.Second)
	contentCh, errCh := c.Stream(context.Background(), "token", "", []ChatMessage{
		{Role: RoleUser, Content: "Hi"},
	})

	// Drain content channel
	for range contentCh {
	}

	err := <-errCh
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestStream_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Send chunks slowly
		for i := 0; i < 10; i++ {
			w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"chunk\"}}]}\n"))
			w.(http.Flusher).Flush()
			time.Sleep(50 * time.Millisecond)
		}
		w.Write([]byte("data: [DONE]\n"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-model", 100, 5*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	contentCh, errCh := c.Stream(ctx, "token", "", []ChatMessage{
		{Role: RoleUser, Content: "Hi"},
	})

	// Collect what we can
	var chunks int
	for range contentCh {
		chunks++
	}

	// We should have received some chunks before timeout
	if chunks == 0 {
		t.Error("expected to receive some chunks before cancellation")
	}

	err := <-errCh
	if err == nil {
		t.Log("No error from channel, but context was cancelled")
	}
}

func TestStream_EmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Send chunk with empty content
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"\"}}]}\n"))
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"actual content\"}}]}\n"))
		w.Write([]byte("data: [DONE]\n"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-model", 100, 5*time.Second)
	contentCh, errCh := c.Stream(context.Background(), "token", "", []ChatMessage{
		{Role: RoleUser, Content: "Hi"},
	})

	var result string
	for chunk := range contentCh {
		result += chunk
	}

	if err := <-errCh; err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty content chunks should be skipped
	if result != "actual content" {
		t.Errorf("result = %q, want 'actual content'", result)
	}
}
