// Package auth provides authentication and signature verification
// for GitHub Copilot Extension requests.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// VerifySignature validates the X-Hub-Signature-256 header against the body.
func VerifySignature(body []byte, signature, secret string) bool {
	if secret == "" || signature == "" {
		return false
	}

	// Signature format: sha256=hex_digest
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	sigHex := strings.TrimPrefix(signature, "sha256=")

	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)

	return hmac.Equal(sigBytes, expected)
}

// SignPayload generates a signature for testing purposes.
func SignPayload(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
