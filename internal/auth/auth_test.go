package auth

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifySignature_Valid(t *testing.T) {
	body := []byte("hello world")
	secret := "test-secret"
	signature := SignPayload(body, secret)

	if !VerifySignature(body, signature, secret) {
		t.Error("VerifySignature should return true for valid signature")
	}
}

func TestVerifySignature_Invalid(t *testing.T) {
	body := []byte("hello world")
	secret := "test-secret"

	tests := []struct {
		name      string
		signature string
		secret    string
	}{
		{"wrong signature", "sha256=invalid", secret},
		{"empty secret", "sha256=abc123", ""},
		{"empty signature", "", secret},
		{"no sha256 prefix", "abc123", secret},
		{"wrong secret", SignPayload(body, "wrong-secret"), secret},
		{"invalid hex", "sha256=not-hex-!@#", secret},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if VerifySignature(body, tt.signature, tt.secret) {
				t.Error("VerifySignature should return false")
			}
		})
	}
}

func TestSignPayload(t *testing.T) {
	body := []byte("test body")
	secret := "test-secret"

	sig := SignPayload(body, secret)

	if sig == "" {
		t.Error("SignPayload should return non-empty signature")
	}
	if len(sig) < 10 {
		t.Error("SignPayload should return a valid signature")
	}
	if sig[:7] != "sha256=" {
		t.Errorf("SignPayload should return sha256= prefix, got %s", sig[:7])
	}

	// Verify the signature is valid
	if !VerifySignature(body, sig, secret) {
		t.Error("SignPayload should produce valid signature")
	}
}

func TestMiddleware_SkipsGET(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	wrapped := Middleware("secret", false)(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GET should be allowed without signature, got %d", rr.Code)
	}
}

func TestMiddleware_ValidSignature(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	secret := "test-secret"
	wrapped := Middleware(secret, false)(handler)

	body := []byte(`{"test": true}`)
	req := httptest.NewRequest(http.MethodPost, "/agent", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", SignPayload(body, secret))
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Valid signature should pass, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestMiddleware_InvalidSignature(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := Middleware("secret", false)(handler)

	body := []byte(`{"test": true}`)
	req := httptest.NewRequest(http.MethodPost, "/agent", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", "sha256=invalid")
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Invalid signature should return 401, got %d", rr.Code)
	}
}

func TestMiddleware_MissingSignature(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := Middleware("secret", false)(handler)

	body := []byte(`{"test": true}`)
	req := httptest.NewRequest(http.MethodPost, "/agent", bytes.NewReader(body))
	// No X-Hub-Signature-256 header
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Missing signature should return 401, got %d", rr.Code)
	}
}

func TestMiddleware_DevModeNoSecret(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	// Dev mode with empty secret
	wrapped := Middleware("", true)(handler)

	body := []byte(`{"test": true}`)
	req := httptest.NewRequest(http.MethodPost, "/agent", bytes.NewReader(body))
	// No signature
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Dev mode with no secret should allow request, got %d", rr.Code)
	}
}

func TestMiddleware_ProdModeNoSecret(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// Prod mode (devMode=false) with empty secret
	wrapped := Middleware("", false)(handler)

	body := []byte(`{"test": true}`)
	req := httptest.NewRequest(http.MethodPost, "/agent", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Prod mode with no secret should return 500, got %d", rr.Code)
	}
}

func TestMiddleware_BodyPreserved(t *testing.T) {
	var receivedBody []byte
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	secret := "test-secret"
	wrapped := Middleware(secret, false)(handler)

	body := []byte(`{"message": "hello world"}`)
	req := httptest.NewRequest(http.MethodPost, "/agent", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", SignPayload(body, secret))
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Request should succeed, got %d", rr.Code)
	}
	if !bytes.Equal(receivedBody, body) {
		t.Errorf("Body should be preserved, got %s", receivedBody)
	}
}
