// =============================================================================
// Deploy Promoter Copilot Agent
// =============================================================================
// A Copilot Agent that manages environment promotion workflows, ensuring safe
// progression of infrastructure changes through dev → staging → production.
//
// Features:
//   - Environment promotion workflows
//   - Approval gates
//   - Version tracking
//   - Rollback capabilities
//   - Stream results via Server-Sent Events
//
// Usage:
//   go run .
//   # Server starts on :8088
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
	GitHubToken   string
	WebhookSecret string
	Debug         bool
}

func loadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	return &Config{
		Port:          port,
		GitHubToken:   os.Getenv("GITHUB_TOKEN"),
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
// Promotion Types
// =============================================================================

type Environment struct {
	Name             string   `json:"name"`
	Level            int      `json:"level"` // 1=dev, 2=staging, 3=prod
	RequiresApproval bool     `json:"requires_approval"`
	Approvers        []string `json:"approvers"`
}

type PromotionRequest struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	Version     string `json:"version"`
	Changes     int    `json:"changes"`
	RequestedBy string `json:"requested_by"`
}

type PromotionResult struct {
	Allowed   bool     `json:"allowed"`
	Reason    string   `json:"reason"`
	Checks    []Check  `json:"checks"`
	NextSteps []string `json:"next_steps"`
}

type Check struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type DeploymentVersion struct {
	Version     string    `json:"version"`
	Environment string    `json:"environment"`
	DeployedAt  time.Time `json:"deployed_at"`
	Status      string    `json:"status"`
	Changes     []string  `json:"changes"`
}

// =============================================================================
// Server
// =============================================================================

type Server struct {
	config       *Config
	mux          *http.ServeMux
	environments []Environment
	deployments  map[string][]DeploymentVersion
}

func NewServer(config *Config) *Server {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
		environments: []Environment{
			{Name: "dev", Level: 1, RequiresApproval: false},
			{Name: "staging", Level: 2, RequiresApproval: true, Approvers: []string{"lead"}},
			{Name: "prod", Level: 3, RequiresApproval: true, Approvers: []string{"lead", "manager"}},
		},
		deployments: make(map[string][]DeploymentVersion),
	}
	s.seedDeployments()
	s.setupRoutes()
	return s
}

func (s *Server) seedDeployments() {
	s.deployments["dev"] = []DeploymentVersion{
		{Version: "v1.3.0", Environment: "dev", DeployedAt: time.Now().Add(-1 * time.Hour), Status: "deployed"},
	}
	s.deployments["staging"] = []DeploymentVersion{
		{Version: "v1.2.0", Environment: "staging", DeployedAt: time.Now().Add(-24 * time.Hour), Status: "deployed"},
	}
	s.deployments["prod"] = []DeploymentVersion{
		{Version: "v1.1.0", Environment: "prod", DeployedAt: time.Now().Add(-72 * time.Hour), Status: "deployed"},
	}
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/agent", s.handleAgent)
	s.mux.HandleFunc("/promote", s.handlePromote)
	s.mux.HandleFunc("/status", s.handleStatus)
	s.mux.HandleFunc("/", s.handleAgent)
}

func (s *Server) Run() error {
	addr := ":" + s.config.Port
	log.Printf("🚀 Deploy Promoter Agent starting on %s", addr)
	log.Printf("📍 Endpoints:")
	log.Printf("   POST /agent   - Agent endpoint (SSE)")
	log.Printf("   POST /promote - Promote deployment")
	log.Printf("   GET  /status  - Deployment status")
	log.Printf("   GET  /health  - Health check")
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": "deploy-promoter-agent",
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.deployments)
}

func (s *Server) handlePromote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Use POST /agent for promotions",
	})
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
	s.processPromotion(r.Context(), req, sse)
}

func (s *Server) processPromotion(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = strings.ToLower(req.Messages[i].Content)
			break
		}
	}

	sse.SendMessage("🚀 **Deploy Promoter Agent**\n\n")

	// Determine action
	if strings.Contains(userMessage, "status") {
		s.reportStatus(sse)
		return
	}

	if strings.Contains(userMessage, "rollback") {
		s.handleRollback(userMessage, sse)
		return
	}

	// Handle promotion
	source, target := s.parsePromotion(userMessage)
	if source == "" || target == "" {
		s.showHelp(sse)
		return
	}

	sse.SendMessage(fmt.Sprintf("Processing promotion: **%s** → **%s**\n\n", source, target))
	time.Sleep(300 * time.Millisecond)

	result := s.evaluatePromotion(source, target)
	s.reportPromotion(result, source, target, sse)
}

func (s *Server) parsePromotion(msg string) (string, string) {
	patterns := []struct {
		re     *regexp.Regexp
		source int
		target int
	}{
		{regexp.MustCompile(`promote\s+(\w+)\s+to\s+(\w+)`), 1, 2},
		{regexp.MustCompile(`(\w+)\s*->\s*(\w+)`), 1, 2},
		{regexp.MustCompile(`from\s+(\w+)\s+to\s+(\w+)`), 1, 2},
	}

	for _, p := range patterns {
		if m := p.re.FindStringSubmatch(msg); len(m) > 2 {
			return s.normalizeEnv(m[p.source]), s.normalizeEnv(m[p.target])
		}
	}

	// Implicit promotions
	if strings.Contains(msg, "staging") && strings.Contains(msg, "promote") {
		return "dev", "staging"
	}
	if strings.Contains(msg, "prod") && strings.Contains(msg, "promote") {
		return "staging", "prod"
	}

	return "", ""
}

func (s *Server) normalizeEnv(env string) string {
	env = strings.ToLower(strings.TrimSpace(env))
	switch env {
	case "development", "dev":
		return "dev"
	case "stage", "staging", "stg":
		return "staging"
	case "production", "prod", "prd":
		return "prod"
	}
	return env
}

func (s *Server) evaluatePromotion(source, target string) PromotionResult {
	result := PromotionResult{
		Allowed: true,
		Checks:  []Check{},
	}

	// Check 1: Source exists and has deployment
	srcDeploys := s.deployments[source]
	if len(srcDeploys) == 0 {
		result.Checks = append(result.Checks, Check{
			Name:    "Source Deployment",
			Passed:  false,
			Message: fmt.Sprintf("No deployments found in %s", source),
		})
		result.Allowed = false
	} else {
		result.Checks = append(result.Checks, Check{
			Name:    "Source Deployment",
			Passed:  true,
			Message: fmt.Sprintf("Version %s available", srcDeploys[len(srcDeploys)-1].Version),
		})
	}

	// Check 2: Environment order
	srcEnv := s.getEnvironment(source)
	tgtEnv := s.getEnvironment(target)
	if srcEnv.Level >= tgtEnv.Level {
		result.Checks = append(result.Checks, Check{
			Name:    "Environment Order",
			Passed:  false,
			Message: "Can only promote to higher environments",
		})
		result.Allowed = false
	} else {
		result.Checks = append(result.Checks, Check{
			Name:    "Environment Order",
			Passed:  true,
			Message: fmt.Sprintf("%s (L%d) → %s (L%d)", source, srcEnv.Level, target, tgtEnv.Level),
		})
	}

	// Check 3: Approval requirements
	if tgtEnv.RequiresApproval {
		result.Checks = append(result.Checks, Check{
			Name:    "Approval Required",
			Passed:  true,
			Message: fmt.Sprintf("Requires approval from: %s", strings.Join(tgtEnv.Approvers, ", ")),
		})
		result.NextSteps = append(result.NextSteps, "Request approval from required approvers")
	}

	// Check 4: Skip check (no skipping environments)
	if tgtEnv.Level-srcEnv.Level > 1 {
		result.Checks = append(result.Checks, Check{
			Name:    "Sequential Promotion",
			Passed:  false,
			Message: "Cannot skip environments. Promote through staging first.",
		})
		result.Allowed = false
	}

	if result.Allowed {
		result.Reason = "Promotion approved"
		result.NextSteps = append(result.NextSteps, "Run terraform plan in target environment")
		result.NextSteps = append(result.NextSteps, "Review changes and apply")
	} else {
		result.Reason = "Promotion blocked"
	}

	return result
}

func (s *Server) getEnvironment(name string) Environment {
	for _, e := range s.environments {
		if e.Name == name {
			return e
		}
	}
	return Environment{Name: name, Level: 0}
}

func (s *Server) reportPromotion(result PromotionResult, source, target string, sse *SSEWriter) {
	icon := "✅"
	if !result.Allowed {
		icon = "❌"
	}

	sse.SendMessage(fmt.Sprintf("## %s Promotion: %s\n\n", icon, result.Reason))

	// Checks table
	sse.SendMessage("### Pre-flight Checks\n\n")
	sse.SendMessage("| Check | Status | Details |\n")
	sse.SendMessage("|-------|--------|--------|\n")
	for _, c := range result.Checks {
		status := "✅"
		if !c.Passed {
			status = "❌"
		}
		sse.SendMessage(fmt.Sprintf("| %s | %s | %s |\n", c.Name, status, c.Message))
	}
	sse.SendMessage("\n")

	// Next steps
	if len(result.NextSteps) > 0 {
		sse.SendMessage("### Next Steps\n\n")
		for i, step := range result.NextSteps {
			sse.SendMessage(fmt.Sprintf("%d. %s\n", i+1, step))
		}
	}

	sse.SendMessage("\n---\n*Deploy Promoter Agent*")
}

func (s *Server) reportStatus(sse *SSEWriter) {
	sse.SendMessage("## Deployment Status\n\n")

	sse.SendMessage("| Environment | Version | Deployed | Status |\n")
	sse.SendMessage("|-------------|---------|----------|--------|\n")

	for _, env := range []string{"dev", "staging", "prod"} {
		deploys := s.deployments[env]
		if len(deploys) > 0 {
			d := deploys[len(deploys)-1]
			ago := time.Since(d.DeployedAt).Round(time.Hour)
			sse.SendMessage(fmt.Sprintf("| %s | %s | %v ago | %s |\n", env, d.Version, ago, d.Status))
		} else {
			sse.SendMessage(fmt.Sprintf("| %s | - | - | no deployment |\n", env))
		}
	}

	sse.SendMessage("\n### Promotion Path\n\n")
	sse.SendMessage("```\n")
	sse.SendMessage("dev (v1.3.0) → staging (v1.2.0) → prod (v1.1.0)\n")
	sse.SendMessage("```\n")
	sse.SendMessage("\n*Use 'promote dev to staging' to advance versions*")
}

func (s *Server) handleRollback(msg string, sse *SSEWriter) {
	sse.SendMessage("## 🔙 Rollback Request\n\n")
	sse.SendMessage("**Rollback Process:**\n\n")
	sse.SendMessage("1. Identify the target version\n")
	sse.SendMessage("2. Verify rollback version exists\n")
	sse.SendMessage("3. Run terraform plan with previous state\n")
	sse.SendMessage("4. Apply after approval\n\n")
	sse.SendMessage("⚠️ Rollbacks may cause service disruption.\n")
}

func (s *Server) showHelp(sse *SSEWriter) {
	sse.SendMessage("## Deploy Promoter Help\n\n")
	sse.SendMessage("**Commands:**\n")
	sse.SendMessage("- `promote dev to staging` - Promote from dev\n")
	sse.SendMessage("- `promote staging to prod` - Promote to production\n")
	sse.SendMessage("- `status` - Show deployment status\n")
	sse.SendMessage("- `rollback prod` - Initiate rollback\n\n")
	sse.SendMessage("**Promotion Rules:**\n")
	sse.SendMessage("- Must promote sequentially (dev→staging→prod)\n")
	sse.SendMessage("- Staging/prod require approvals\n")
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

func main() {
	config := loadConfig()
	server := NewServer(config)
	if err := server.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
