// Package auth provides authentication and signature verification
// for GitHub Copilot Extension requests.
package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"strings"
)

// VerifySignature validates the X-Hub-Signature-256 header against the body.
// Returns false if the secret is empty, signature is missing, or signature doesn't match.
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

// Middleware returns an HTTP middleware that verifies request signatures.
// - GET requests are always allowed (health checks, agent listing)
// - In dev mode (empty secret): logs warning but allows request
// - In prod mode: rejects requests with invalid/missing signature
func Middleware(secret string, devMode bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip GET requests (health, agents listing)
			if r.Method == http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			// Dev mode: warn but allow if no secret configured
			if secret == "" {
				if devMode {
					log.Printf("WARNING: Webhook signature verification skipped (dev mode, no secret configured)")
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, "Webhook secret not configured", http.StatusInternalServerError)
				return
			}

			// Read body for signature verification
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}
			// Restore body for downstream handlers
			r.Body = io.NopCloser(bytes.NewReader(body))

			sig := r.Header.Get("X-Hub-Signature-256")
			if !VerifySignature(body, sig, secret) {
				http.Error(w, "Invalid signature", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
