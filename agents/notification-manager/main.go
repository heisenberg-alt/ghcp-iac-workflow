// =============================================================================
// Notification Manager Copilot Agent
// =============================================================================
// A Copilot Agent that handles multi-channel notifications for IaC events,
// including deployments, policy violations, and drift detection.
//
// Features:
//   - Multi-channel notifications (Teams, Slack, Email, Webhooks)
//   - Notification templates
//   - Event routing
//   - Notification history
//   - Stream results via Server-Sent Events
//
// Usage:
//   go run .
//   # Server starts on :8089
// =============================================================================

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// =============================================================================
// Configuration
// =============================================================================

type Config struct {
	Port          string
	TeamsWebhook  string
	SlackWebhook  string
	SMTPServer    string
	WebhookSecret string
	Debug         bool
}

func loadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8089"
	}

	return &Config{
		Port:          port,
		TeamsWebhook:  os.Getenv("TEAMS_WEBHOOK_URL"),
		SlackWebhook:  os.Getenv("SLACK_WEBHOOK_URL"),
		SMTPServer:    os.Getenv("SMTP_SERVER"),
		WebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),
		Debug:         os.Getenv("DEBUG") != "",
	}
}

// =============================================================================
// Copilot Agent Types
// =============================================================================

type AgentRequest struct {
	Messages          []Message          `json:"messages"`
	CopilotReferences []CopilotReference `json:"copilot_references,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CopilotReference struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Data struct {
		Content  string `json:"content,omitempty"`
		Language string `json:"language,omitempty"`
	} `json:"data,omitempty"`
}

// =============================================================================
// Notification Types
// =============================================================================

type Channel struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // teams, slack, email, webhook
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
}

type NotificationEvent struct {
	Type        string                 `json:"type"`     // deployment, drift, policy, security, cost
	Severity    string                 `json:"severity"` // info, warning, error, critical
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Resource    string                 `json:"resource"`
	Environment string                 `json:"environment"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data"`
}

type NotificationResult struct {
	Channel   string    `json:"channel"`
	Status    string    `json:"status"` // sent, failed, skipped
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
}

type NotificationHistory struct {
	Event   NotificationEvent    `json:"event"`
	Results []NotificationResult `json:"results"`
	SentAt  time.Time            `json:"sent_at"`
}

type RoutingRule struct {
	EventType string   `json:"event_type"`
	Severity  string   `json:"severity"`
	Channels  []string `json:"channels"`
}

// =============================================================================
// Server
// =============================================================================

type Server struct {
	config   *Config
	mux      *http.ServeMux
	channels []Channel
	rules    []RoutingRule
	history  []NotificationHistory
}

func NewServer(config *Config) *Server {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
		channels: []Channel{
			{Name: "teams-alerts", Type: "teams", Enabled: config.TeamsWebhook != "", Endpoint: config.TeamsWebhook},
			{Name: "slack-devops", Type: "slack", Enabled: config.SlackWebhook != "", Endpoint: config.SlackWebhook},
			{Name: "email-admins", Type: "email", Enabled: config.SMTPServer != "", Endpoint: config.SMTPServer},
			{Name: "webhook-audit", Type: "webhook", Enabled: true, Endpoint: "https://audit.example.com/events"},
		},
		rules: []RoutingRule{
			{EventType: "deployment", Severity: "info", Channels: []string{"slack-devops"}},
			{EventType: "deployment", Severity: "error", Channels: []string{"teams-alerts", "slack-devops", "email-admins"}},
			{EventType: "drift", Severity: "*", Channels: []string{"teams-alerts", "slack-devops"}},
			{EventType: "policy", Severity: "warning", Channels: []string{"slack-devops"}},
			{EventType: "policy", Severity: "error", Channels: []string{"teams-alerts", "email-admins"}},
			{EventType: "security", Severity: "*", Channels: []string{"teams-alerts", "email-admins", "webhook-audit"}},
			{EventType: "cost", Severity: "warning", Channels: []string{"slack-devops"}},
		},
		history: []NotificationHistory{},
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/agent", s.handleAgent)
	s.mux.HandleFunc("/notify", s.handleNotify)
	s.mux.HandleFunc("/channels", s.handleChannels)
	s.mux.HandleFunc("/history", s.handleHistory)
	s.mux.HandleFunc("/", s.handleAgent)
}

func (s *Server) Run() error {
	addr := ":" + s.config.Port
	log.Printf("📢 Notification Manager Agent starting on %s", addr)
	log.Printf("📍 Endpoints:")
	log.Printf("   POST /agent    - Agent endpoint (SSE)")
	log.Printf("   POST /notify   - Send notification")
	log.Printf("   GET  /channels - List channels")
	log.Printf("   GET  /history  - Notification history")
	log.Printf("   GET  /health   - Health check")
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": "notification-manager-agent",
	})
}

func (s *Server) handleChannels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.channels)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.history)
}

func (s *Server) handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event NotificationEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	results := s.sendNotification(event)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	sse := NewSSEWriter(w, flusher)
	s.processNotificationRequest(r.Context(), req, sse)
}

func (s *Server) processNotificationRequest(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = strings.ToLower(req.Messages[i].Content)
			break
		}
	}

	sse.SendMessage("📢 **Notification Manager Agent**\n\n")

	// Parse command
	if strings.Contains(userMessage, "channels") || strings.Contains(userMessage, "list") {
		s.listChannels(sse)
		return
	}

	if strings.Contains(userMessage, "history") || strings.Contains(userMessage, "recent") {
		s.showHistory(sse)
		return
	}

	if strings.Contains(userMessage, "rules") || strings.Contains(userMessage, "routing") {
		s.showRules(sse)
		return
	}

	if strings.Contains(userMessage, "test") {
		s.testNotification(userMessage, sse)
		return
	}

	if strings.Contains(userMessage, "send") || strings.Contains(userMessage, "notify") {
		s.parseAndSend(userMessage, sse)
		return
	}

	s.showHelp(sse)
}

func (s *Server) listChannels(sse *SSEWriter) {
	sse.SendMessage("## Notification Channels\n\n")

	sse.SendMessage("| Channel | Type | Status |\n")
	sse.SendMessage("|---------|------|--------|\n")

	for _, ch := range s.channels {
		status := "✅ Enabled"
		if !ch.Enabled {
			status = "⚪ Disabled"
		}
		sse.SendMessage(fmt.Sprintf("| %s | %s | %s |\n", ch.Name, ch.Type, status))
	}

	sse.SendMessage("\n### Channel Types\n\n")
	sse.SendMessage("- 📱 **Teams:** Microsoft Teams webhooks\n")
	sse.SendMessage("- 💬 **Slack:** Slack incoming webhooks\n")
	sse.SendMessage("- 📧 **Email:** SMTP notifications\n")
	sse.SendMessage("- 🔗 **Webhook:** Custom HTTP endpoints\n")
}

func (s *Server) showRules(sse *SSEWriter) {
	sse.SendMessage("## Routing Rules\n\n")

	sse.SendMessage("| Event Type | Severity | Channels |\n")
	sse.SendMessage("|------------|----------|----------|\n")

	for _, rule := range s.rules {
		sev := rule.Severity
		if sev == "*" {
			sev = "all"
		}
		sse.SendMessage(fmt.Sprintf("| %s | %s | %s |\n",
			rule.EventType, sev, strings.Join(rule.Channels, ", ")))
	}

	sse.SendMessage("\n### Event Types\n\n")
	sse.SendMessage("- 🚀 `deployment` - Deployment events\n")
	sse.SendMessage("- 🔄 `drift` - Configuration drift\n")
	sse.SendMessage("- 📋 `policy` - Policy violations\n")
	sse.SendMessage("- 🔒 `security` - Security findings\n")
	sse.SendMessage("- 💰 `cost` - Cost alerts\n")
}

func (s *Server) showHistory(sse *SSEWriter) {
	sse.SendMessage("## Recent Notifications\n\n")

	if len(s.history) == 0 {
		sse.SendMessage("No notifications sent yet.\n")
		return
	}

	sse.SendMessage("| Time | Event | Channels | Status |\n")
	sse.SendMessage("|------|-------|----------|--------|\n")

	for _, h := range s.history {
		channels := []string{}
		for _, r := range h.Results {
			channels = append(channels, r.Channel)
		}
		sse.SendMessage(fmt.Sprintf("| %s | %s | %s | %d sent |\n",
			h.SentAt.Format("15:04"),
			h.Event.Type,
			strings.Join(channels, ", "),
			len(h.Results)))
	}
}

func (s *Server) testNotification(msg string, sse *SSEWriter) {
	sse.SendMessage("## 🧪 Test Notification\n\n")

	// Determine channel
	channel := "slack-devops"
	if strings.Contains(msg, "teams") {
		channel = "teams-alerts"
	} else if strings.Contains(msg, "email") {
		channel = "email-admins"
	}

	event := NotificationEvent{
		Type:        "deployment",
		Severity:    "info",
		Title:       "Test Notification",
		Message:     "This is a test notification from the Notification Manager Agent",
		Resource:    "test-resource",
		Environment: "dev",
		Timestamp:   time.Now(),
	}

	sse.SendMessage(fmt.Sprintf("Sending test to **%s**...\n\n", channel))
	time.Sleep(500 * time.Millisecond)

	sse.SendMessage("### Notification Preview\n\n")
	sse.SendMessage("```json\n")
	preview, _ := json.MarshalIndent(event, "", "  ")
	sse.SendMessage(string(preview))
	sse.SendMessage("\n```\n\n")

	sse.SendMessage(fmt.Sprintf("✅ Test notification queued for **%s**\n", channel))
}

func (s *Server) parseAndSend(msg string, sse *SSEWriter) {
	sse.SendMessage("## 📤 Send Notification\n\n")

	// Extract event type
	eventType := "deployment"
	for _, t := range []string{"security", "drift", "policy", "cost"} {
		if strings.Contains(msg, t) {
			eventType = t
			break
		}
	}

	// Extract severity
	severity := "info"
	for _, s := range []string{"critical", "error", "warning"} {
		if strings.Contains(msg, s) {
			severity = s
			break
		}
	}

	// Build event
	event := NotificationEvent{
		Type:      eventType,
		Severity:  severity,
		Title:     fmt.Sprintf("%s Notification", strings.Title(eventType)),
		Message:   "Notification triggered via Copilot Agent",
		Timestamp: time.Now(),
	}

	sse.SendMessage(fmt.Sprintf("**Event Type:** %s\n", eventType))
	sse.SendMessage(fmt.Sprintf("**Severity:** %s\n\n", severity))

	// Find matching rules
	channels := s.findChannels(event)
	if len(channels) == 0 {
		sse.SendMessage("⚠️ No routing rules match this event.\n")
		return
	}

	sse.SendMessage(fmt.Sprintf("**Routing to:** %s\n\n", strings.Join(channels, ", ")))

	// Simulate sending
	results := s.sendNotification(event)

	sse.SendMessage("### Results\n\n")
	for _, r := range results {
		icon := "✅"
		if r.Status == "failed" {
			icon = "❌"
		} else if r.Status == "skipped" {
			icon = "⚪"
		}
		sse.SendMessage(fmt.Sprintf("- %s **%s:** %s\n", icon, r.Channel, r.Status))
	}
}

func (s *Server) findChannels(event NotificationEvent) []string {
	channelSet := make(map[string]bool)

	for _, rule := range s.rules {
		if rule.EventType == event.Type {
			if rule.Severity == "*" || rule.Severity == event.Severity {
				for _, ch := range rule.Channels {
					channelSet[ch] = true
				}
			}
		}
	}

	var channels []string
	for ch := range channelSet {
		channels = append(channels, ch)
	}
	return channels
}

func (s *Server) sendNotification(event NotificationEvent) []NotificationResult {
	channels := s.findChannels(event)
	var results []NotificationResult

	for _, chName := range channels {
		var ch *Channel
		for i := range s.channels {
			if s.channels[i].Name == chName {
				ch = &s.channels[i]
				break
			}
		}

		result := NotificationResult{
			Channel:   chName,
			Timestamp: time.Now(),
		}

		if ch == nil || !ch.Enabled {
			result.Status = "skipped"
			result.Error = "Channel not configured"
		} else {
			// Simulate sending (in real impl, would call webhook)
			result.Status = "sent"
		}

		results = append(results, result)
	}

	// Record history
	s.history = append(s.history, NotificationHistory{
		Event:   event,
		Results: results,
		SentAt:  time.Now(),
	})

	return results
}

func (s *Server) showHelp(sse *SSEWriter) {
	sse.SendMessage("## Notification Manager Help\n\n")
	sse.SendMessage("**Commands:**\n")
	sse.SendMessage("- `channels` - List notification channels\n")
	sse.SendMessage("- `rules` - Show routing rules\n")
	sse.SendMessage("- `history` - Recent notifications\n")
	sse.SendMessage("- `test [channel]` - Send test notification\n")
	sse.SendMessage("- `send [type] [severity]` - Send notification\n\n")
	sse.SendMessage("**Examples:**\n")
	sse.SendMessage("- `test teams` - Test Teams notification\n")
	sse.SendMessage("- `send security critical` - Send security alert\n")
}

type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func NewSSEWriter(w http.ResponseWriter, f http.Flusher) *SSEWriter {
	return &SSEWriter{w: w, flusher: f}
}

func (s *SSEWriter) SendMessage(content string) {
	data := map[string]string{"content": content}
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(s.w, "event: copilot_message\ndata: %s\n\n", jsonData)
	s.flusher.Flush()
}

func (s *SSEWriter) SendDone() {
	fmt.Fprintf(s.w, "event: copilot_done\ndata: {}\n\n")
	s.flusher.Flush()
}

// Suppress unused warning
var _ = regexp.MustCompile

func main() {
	config := loadConfig()
	server := NewServer(config)
	if err := server.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
