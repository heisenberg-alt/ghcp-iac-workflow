package httpx

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/server"
)

func TestSSEEmitter_SendMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	sse := server.NewSSEWriter(rec)
	if sse == nil {
		t.Fatal("NewSSEWriter returned nil")
	}
	emit := NewSSEEmitter(sse)
	emit.SendMessage("hello world")
	body := rec.Body.String()
	if !strings.Contains(body, "event: copilot_message") {
		t.Error("expected copilot_message event")
	}
	if !strings.Contains(body, "hello world") {
		t.Error("expected message content")
	}
}

func TestSSEEmitter_SendReferences(t *testing.T) {
	rec := httptest.NewRecorder()
	sse := server.NewSSEWriter(rec)
	emit := NewSSEEmitter(sse)
	emit.SendReferences([]protocol.Reference{
		{Title: "Test Ref", URL: "https://example.com"},
	})
	body := rec.Body.String()
	if !strings.Contains(body, "event: copilot_references") {
		t.Error("expected copilot_references event")
	}
	if !strings.Contains(body, "Test Ref") {
		t.Error("expected reference title")
	}
}

func TestSSEEmitter_SendDone(t *testing.T) {
	rec := httptest.NewRecorder()
	sse := server.NewSSEWriter(rec)
	emit := NewSSEEmitter(sse)
	emit.SendDone()
	body := rec.Body.String()
	if !strings.Contains(body, "event: copilot_done") {
		t.Error("expected copilot_done event")
	}
}

func TestSSEEmitter_SendError(t *testing.T) {
	rec := httptest.NewRecorder()
	sse := server.NewSSEWriter(rec)
	emit := NewSSEEmitter(sse)
	emit.SendError("something broke")
	body := rec.Body.String()
	if !strings.Contains(body, "something broke") {
		t.Error("expected error message in output")
	}
}

func TestSSEEmitter_ImplementsEmitter(t *testing.T) {
	var _ protocol.Emitter = (*SSEEmitter)(nil)
}
