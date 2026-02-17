package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/config"
	"github.com/google/uuid"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	tokenKey     contextKey = "github_token"
)

// RequestID returns the request ID from the context.
func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// GitHubToken returns the GitHub token from the context.
func GitHubToken(ctx context.Context) string {
	if t, ok := ctx.Value(tokenKey).(string); ok {
		return t
	}
	return ""
}

// WithGitHubToken adds a GitHub token to the context.
func WithGitHubToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// Handler is the interface that agent handlers implement.
type Handler interface {
	HandleAgent(ctx context.Context, req AgentRequest, sse *SSEWriter)
}

// Server is the main HTTP server that hosts the Copilot Extension.
type Server struct {
	config  *config.Config
	mux     *http.ServeMux
	handler Handler
	logger  *log.Logger
}

// New creates a new Server.
func New(cfg *config.Config, handler Handler) *Server {
	s := &Server{
		config:  cfg,
		mux:     http.NewServeMux(),
		handler: handler,
		logger:  log.New(os.Stdout, "[server] ", log.LstdFlags|log.Lmsgprefix),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/agent", s.withMiddleware(s.handleAgent))
	s.mux.HandleFunc("/", s.withMiddleware(s.handleAgent))
}

// Run starts the server with graceful shutdown.
func (s *Server) Run() error {
	addr := ":" + s.config.Port
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second, // long for SSE streaming
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		s.logger.Printf("Starting on %s (env=%s, model=%s)", addr, s.config.Environment, s.config.ModelName)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatalf("Server error: %v", err)
		}
	}()

	<-done
	s.logger.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "healthy",
		"service":     "ghcp-iac",
		"environment": s.config.Environment,
		"model":       s.config.ModelName,
		"llm_enabled": s.config.EnableLLM,
	})
}

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	sse := NewSSEWriter(w)
	if sse == nil {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	s.handler.HandleAgent(r.Context(), req, sse)
	sse.SendDone()
}

// withMiddleware wraps a handler with request ID, token extraction, signature verification, logging, and recovery.
func (s *Server) withMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Panic recovery
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Printf("PANIC: %v", rec)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()

		// Request ID
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)

		// Webhook signature verification (skipped in dev when no secret is set)
		if s.config.WebhookSecret != "" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			sig := r.Header.Get("X-Hub-Signature-256")
			if !verifySignature(body, sig, s.config.WebhookSecret) {
				s.logger.Printf("Rejected request [%s]: invalid signature", reqID)
				http.Error(w, "Invalid signature", http.StatusUnauthorized)
				return
			}
		}

		// Extract GitHub token
		token := r.Header.Get("X-GitHub-Token")
		if token != "" {
			ctx = WithGitHubToken(ctx, token)
		}

		// Logging
		start := time.Now()
		s.logger.Printf("-> %s %s [%s]", r.Method, r.URL.Path, reqID)
		next(w, r.WithContext(ctx))
		s.logger.Printf("<- %s %s [%s] %v", r.Method, r.URL.Path, reqID, time.Since(start))
	}
}

// verifySignature validates the X-Hub-Signature-256 header against the request body.
func verifySignature(body []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(sigBytes, mac.Sum(nil))
}
