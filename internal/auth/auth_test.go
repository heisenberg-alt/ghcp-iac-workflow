package auth

import (
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
