package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifySignature_Valid(t *testing.T) {
	body := []byte(`{"test": true}`)
	secret := "my-secret-key"
	sig := SignPayload(body, secret)

	if !VerifySignature(body, sig, secret) {
		t.Error("VerifySignature should return true for valid signature")
	}
}

func TestVerifySignature_Invalid(t *testing.T) {
	body := []byte(`{"test": true}`)
	secret := "my-secret-key"

	if VerifySignature(body, "sha256=0000000000000000000000000000000000000000000000000000000000000000", secret) {
		t.Error("VerifySignature should return false for wrong signature")
	}
}

func TestVerifySignature_EmptySecret(t *testing.T) {
	body := []byte(`{"test": true}`)
	sig := "sha256=abc123"

	if VerifySignature(body, sig, "") {
		t.Error("VerifySignature should return false when secret is empty")
	}
}

func TestVerifySignature_EmptySignature(t *testing.T) {
	body := []byte(`{"test": true}`)

	if VerifySignature(body, "", "secret") {
		t.Error("VerifySignature should return false when signature is empty")
	}
}

func TestVerifySignature_BadPrefix(t *testing.T) {
	body := []byte(`{"test": true}`)

	if VerifySignature(body, "md5=abc123", "secret") {
		t.Error("VerifySignature should return false for non-sha256 prefix")
	}
}

func TestVerifySignature_BadHex(t *testing.T) {
	body := []byte(`{"test": true}`)

	if VerifySignature(body, "sha256=not-valid-hex!", "secret") {
		t.Error("VerifySignature should return false for invalid hex")
	}
}

func TestSignPayload(t *testing.T) {
	body := []byte("hello")
	secret := "key"
	sig := SignPayload(body, secret)

	if !strings.HasPrefix(sig, "sha256=") {
		t.Errorf("SignPayload should start with sha256=, got %q", sig)
	}
	if len(sig) != 7+64 { // "sha256=" + 64 hex chars
		t.Errorf("SignPayload length = %d, want %d", len(sig), 7+64)
	}
}

func TestSignPayload_Roundtrip(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"test"}]}`)
	secret := "webhook-secret-123"
	sig := SignPayload(body, secret)

	if !VerifySignature(body, sig, secret) {
		t.Error("SignPayload -> VerifySignature roundtrip should succeed")
	}
}

func TestExtractToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/agent", nil)
	req.Header.Set("X-GitHub-Token", "ghu_test123")

	token := ExtractToken(req)
	if token != "ghu_test123" {
		t.Errorf("ExtractToken = %q, want ghu_test123", token)
	}
}

func TestExtractToken_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/agent", nil)

	token := ExtractToken(req)
	if token != "" {
		t.Errorf("ExtractToken = %q, want empty string", token)
	}
}

func TestMiddleware_NoSecret(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mw := Middleware("")
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(`{"test":true}`))
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Middleware with empty secret should pass through, got %d", rr.Code)
	}
}

func TestMiddleware_ValidSignature(t *testing.T) {
	secret := "test-secret"
	body := `{"test":true}`

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify body is still readable
		b, _ := io.ReadAll(r.Body)
		if string(b) != body {
			t.Errorf("Body should be preserved, got %q", string(b))
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := Middleware(secret)
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", SignPayload([]byte(body), secret))
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Middleware with valid signature should pass, got %d", rr.Code)
	}
}

func TestMiddleware_InvalidSignature(t *testing.T) {
	secret := "test-secret"
	body := `{"test":true}`

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with invalid signature")
	})

	mw := Middleware(secret)
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256=0000000000000000000000000000000000000000000000000000000000000000")
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Middleware with invalid signature should return 401, got %d", rr.Code)
	}
}

func TestMiddleware_MissingSignature(t *testing.T) {
	secret := "test-secret"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called without signature")
	})

	mw := Middleware(secret)
	wrapped := mw(handler)

	req := httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Middleware with missing signature should return 401, got %d", rr.Code)
	}
}
