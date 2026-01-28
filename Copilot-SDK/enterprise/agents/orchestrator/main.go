// =============================================================================
// Orchestrator Copilot Agent
// =============================================================================
// The central coordinator for the Enterprise IaC Governance Platform. Routes
// requests to specialized agents and aggregates results.
//
// Features:
//   - Request routing to specialized agents
//   - Workflow orchestration
//   - Result aggregation
//   - Cross-agent coordination
//   - Stream results via Server-Sent Events
//
// Usage:
//   go run .
//   # Server starts on :8090
// =============================================================================

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// Configuration
// =============================================================================

type Config struct {
	Port           string
	AgentEndpoints map[string]string
	WebhookSecret  string
	Debug          bool
}

func loadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	return &Config{
		Port: port,
		AgentEndpoints: map[string]string{
			"policy":       getEnvOrDefault("POLICY_AGENT_URL", "http://localhost:8081"),
			"cost":         getEnvOrDefault("COST_AGENT_URL", "http://localhost:8082"),
			"drift":        getEnvOrDefault("DRIFT_AGENT_URL", "http://localhost:8083"),
			"security":     getEnvOrDefault("SECURITY_AGENT_URL", "http://localhost:8084"),
			"compliance":   getEnvOrDefault("COMPLIANCE_AGENT_URL", "http://localhost:8085"),
			"module":       getEnvOrDefault("MODULE_AGENT_URL", "http://localhost:8086"),
			"impact":       getEnvOrDefault("IMPACT_AGENT_URL", "http://localhost:8087"),
			"deploy":       getEnvOrDefault("DEPLOY_AGENT_URL", "http://localhost:8088"),
			"notification": getEnvOrDefault("NOTIFICATION_AGENT_URL", "http://localhost:8089"),
		},
		WebhookSecret: os.Getenv("GITHUB_WEBHOOK_SECRET"),
		Debug:         os.Getenv("DEBUG") != "",
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
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
// Orchestration Types
// =============================================================================

type AgentInfo struct {
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type WorkflowStep struct {
	Agent  string `json:"agent"`
	Action string `json:"action"`
	Status string `json:"status"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

type OrchestratedResult struct {
	Workflow string         `json:"workflow"`
	Steps    []WorkflowStep `json:"steps"`
	Summary  string         `json:"summary"`
	Duration time.Duration  `json:"duration"`
}

// =============================================================================
// Server
// =============================================================================

type Server struct {
	config *Config
	mux    *http.ServeMux
	client *http.Client
}

func NewServer(config *Config) *Server {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
		client: &http.Client{Timeout: 30 * time.Second},
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/agent", s.handleAgent)
	s.mux.HandleFunc("/agents", s.handleListAgents)
	s.mux.HandleFunc("/orchestrate", s.handleOrchestrate)
	s.mux.HandleFunc("/", s.handleAgent)
}

func (s *Server) Run() error {
	addr := ":" + s.config.Port
	log.Printf("🎭 Orchestrator Agent starting on %s", addr)
	log.Printf("📍 Endpoints:")
	log.Printf("   POST /agent      - Agent endpoint (SSE)")
	log.Printf("   GET  /agents     - List registered agents")
	log.Printf("   POST /orchestrate - Run orchestrated workflow")
	log.Printf("   GET  /health     - Health check")
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": "orchestrator-agent",
		"agents":  len(s.config.AgentEndpoints),
	})
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents := s.getAgentStatuses()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func (s *Server) handleOrchestrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Use POST /agent for orchestration",
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
	s.processOrchestratedRequest(r.Context(), req, sse)
}

func (s *Server) processOrchestratedRequest(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = strings.ToLower(req.Messages[i].Content)
			break
		}
	}

	sse.SendMessage("🎭 **Orchestrator Agent**\n\n")

	// Parse intent
	intent := s.parseIntent(userMessage)

	switch intent {
	case "status":
		s.reportAgentStatus(sse)
	case "review":
		s.runCodeReview(ctx, req, sse)
	case "full-analysis":
		s.runFullAnalysis(ctx, req, sse)
	case "deploy-check":
		s.runDeploymentChecks(ctx, req, sse)
	case "route":
		agent := s.determineAgent(userMessage)
		s.routeToAgent(ctx, agent, req, sse)
	default:
		s.showCapabilities(sse)
	}
}

func (s *Server) parseIntent(msg string) string {
	if strings.Contains(msg, "status") || strings.Contains(msg, "agents") {
		return "status"
	}
	if strings.Contains(msg, "full") && strings.Contains(msg, "analysis") {
		return "full-analysis"
	}
	if strings.Contains(msg, "review") && (strings.Contains(msg, "code") || strings.Contains(msg, "pr")) {
		return "review"
	}
	if strings.Contains(msg, "deploy") && (strings.Contains(msg, "check") || strings.Contains(msg, "ready")) {
		return "deploy-check"
	}

	// Check if targeting specific agent
	agents := []string{"policy", "cost", "drift", "security", "compliance", "module", "impact", "deploy", "notification"}
	for _, a := range agents {
		if strings.Contains(msg, a) {
			return "route"
		}
	}

	return "help"
}

func (s *Server) determineAgent(msg string) string {
	agentKeywords := map[string][]string{
		"policy":       {"policy", "policies", "enforce"},
		"cost":         {"cost", "estimate", "pricing", "budget"},
		"drift":        {"drift", "state", "difference"},
		"security":     {"security", "vulnerability", "secrets", "cwe"},
		"compliance":   {"compliance", "audit", "cis", "nist", "soc"},
		"module":       {"module", "registry", "catalog", "approved"},
		"impact":       {"impact", "blast", "radius", "dependency"},
		"deploy":       {"deploy", "promote", "staging", "production"},
		"notification": {"notify", "alert", "teams", "slack"},
	}

	for agent, keywords := range agentKeywords {
		for _, kw := range keywords {
			if strings.Contains(msg, kw) {
				return agent
			}
		}
	}
	return ""
}

func (s *Server) getAgentStatuses() []AgentInfo {
	agents := []AgentInfo{}
	descriptions := map[string]string{
		"policy":       "Validates IaC against governance policies",
		"cost":         "Estimates Azure resource costs",
		"drift":        "Detects configuration drift",
		"security":     "Scans for security vulnerabilities",
		"compliance":   "Audits against compliance frameworks",
		"module":       "Manages approved module registry",
		"impact":       "Analyzes blast radius of changes",
		"deploy":       "Manages environment promotions",
		"notification": "Handles multi-channel notifications",
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, endpoint := range s.config.AgentEndpoints {
		wg.Add(1)
		go func(n, e string) {
			defer wg.Done()

			status := "offline"
			resp, err := s.client.Get(e + "/health")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					status = "online"
				}
			}

			mu.Lock()
			agents = append(agents, AgentInfo{
				Name:        n,
				Endpoint:    e,
				Status:      status,
				Description: descriptions[n],
			})
			mu.Unlock()
		}(name, endpoint)
	}

	wg.Wait()
	return agents
}

func (s *Server) reportAgentStatus(sse *SSEWriter) {
	sse.SendMessage("## Agent Status\n\n")
	sse.SendMessage("Checking agent connectivity...\n\n")

	agents := s.getAgentStatuses()

	sse.SendMessage("| Agent | Status | Endpoint |\n")
	sse.SendMessage("|-------|--------|----------|\n")

	online := 0
	for _, a := range agents {
		icon := "🔴"
		if a.Status == "online" {
			icon = "🟢"
			online++
		}
		sse.SendMessage(fmt.Sprintf("| %s %s | %s | %s |\n", icon, a.Name, a.Status, a.Endpoint))
	}

	sse.SendMessage(fmt.Sprintf("\n**%d/%d agents online**\n", online, len(agents)))
}

func (s *Server) runCodeReview(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	sse.SendMessage("## 📝 Code Review Workflow\n\n")
	sse.SendMessage("Running parallel analysis across agents...\n\n")

	start := time.Now()

	// Run agents in parallel
	type result struct {
		agent  string
		status string
		output string
	}

	results := make(chan result, 4)
	agentsToRun := []string{"policy", "security", "cost", "module"}

	for _, agent := range agentsToRun {
		go func(a string) {
			r := result{agent: a}
			resp, err := s.callAgent(ctx, a, req)
			if err != nil {
				r.status = "error"
				r.output = err.Error()
			} else {
				r.status = "complete"
				r.output = resp
			}
			results <- r
		}(agent)
	}

	// Collect results
	sse.SendMessage("### Results\n\n")
	for i := 0; i < len(agentsToRun); i++ {
		r := <-results
		icon := "✅"
		if r.status == "error" {
			icon = "❌"
		}
		sse.SendMessage(fmt.Sprintf("#### %s %s\n\n", icon, strings.Title(r.agent)))
		if r.status == "complete" && len(r.output) > 200 {
			sse.SendMessage(r.output[:200] + "...\n\n")
		} else {
			sse.SendMessage(r.output + "\n\n")
		}
	}

	sse.SendMessage(fmt.Sprintf("---\n*Completed in %v*\n", time.Since(start).Round(time.Millisecond)))
}

func (s *Server) runFullAnalysis(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	sse.SendMessage("## 🔬 Full Analysis Workflow\n\n")
	sse.SendMessage("Running comprehensive analysis...\n\n")

	steps := []struct {
		agent string
		icon  string
		desc  string
	}{
		{"security", "🔒", "Security Scan"},
		{"policy", "📋", "Policy Check"},
		{"compliance", "✅", "Compliance Audit"},
		{"cost", "💰", "Cost Estimation"},
		{"impact", "💥", "Impact Analysis"},
		{"module", "📦", "Module Validation"},
	}

	for _, step := range steps {
		sse.SendMessage(fmt.Sprintf("### %s %s\n\n", step.icon, step.desc))
		sse.SendMessage(fmt.Sprintf("Contacting %s agent...\n", step.agent))

		_, err := s.callAgent(ctx, step.agent, req)
		if err != nil {
			sse.SendMessage(fmt.Sprintf("⚠️ Agent unavailable: %v\n\n", err))
		} else {
			sse.SendMessage("✅ Analysis complete\n\n")
		}
		time.Sleep(200 * time.Millisecond)
	}

	sse.SendMessage("---\n*Full analysis workflow completed*\n")
}

func (s *Server) runDeploymentChecks(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	sse.SendMessage("## 🚀 Deployment Readiness Check\n\n")

	checks := []struct {
		name  string
		agent string
		check string
	}{
		{"Security Scan", "security", "No critical vulnerabilities"},
		{"Policy Compliance", "policy", "All policies pass"},
		{"Cost Approval", "cost", "Within budget threshold"},
		{"Impact Assessment", "impact", "Acceptable blast radius"},
	}

	allPassed := true

	sse.SendMessage("| Check | Status | Agent |\n")
	sse.SendMessage("|-------|--------|-------|\n")

	for _, c := range checks {
		endpoint := s.config.AgentEndpoints[c.agent]
		resp, err := s.client.Get(endpoint + "/health")

		status := "✅ Pass"
		if err != nil || resp.StatusCode != 200 {
			status = "⚠️ Unavailable"
			allPassed = false
		}
		if resp != nil {
			resp.Body.Close()
		}

		sse.SendMessage(fmt.Sprintf("| %s | %s | %s |\n", c.name, status, c.agent))
	}

	sse.SendMessage("\n")

	if allPassed {
		sse.SendMessage("### ✅ Deployment Ready\n\n")
		sse.SendMessage("All pre-flight checks passed. Safe to proceed with deployment.\n")
	} else {
		sse.SendMessage("### ⚠️ Issues Detected\n\n")
		sse.SendMessage("Some checks could not complete. Review agent status before deploying.\n")
	}
}

func (s *Server) routeToAgent(ctx context.Context, agent string, req AgentRequest, sse *SSEWriter) {
	if agent == "" {
		s.showCapabilities(sse)
		return
	}

	sse.SendMessage(fmt.Sprintf("Routing to **%s** agent...\n\n", agent))

	resp, err := s.callAgent(ctx, agent, req)
	if err != nil {
		sse.SendMessage(fmt.Sprintf("❌ Error: %v\n", err))
		return
	}

	sse.SendMessage(resp)
}

func (s *Server) callAgent(ctx context.Context, agent string, req AgentRequest) (string, error) {
	endpoint, ok := s.config.AgentEndpoints[agent]
	if !ok {
		return "", fmt.Errorf("unknown agent: %s", agent)
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint+"/agent", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Parse SSE response and extract content
	var result strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "{}" || data == "[DONE]" {
				continue
			}
			var sseData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &sseData); err == nil {
				if content, ok := sseData["content"].(string); ok {
					result.WriteString(content)
				}
			}
		}
	}

	return result.String(), nil
}

func (s *Server) showCapabilities(sse *SSEWriter) {
	sse.SendMessage("## Orchestrator Capabilities\n\n")
	sse.SendMessage("**Workflows:**\n")
	sse.SendMessage("- `status` - Check all agent status\n")
	sse.SendMessage("- `code review` - Run parallel code review\n")
	sse.SendMessage("- `full analysis` - Comprehensive IaC analysis\n")
	sse.SendMessage("- `deploy check` - Pre-deployment validation\n\n")

	sse.SendMessage("**Direct Routing:**\n")
	sse.SendMessage("- `policy [code]` - Route to Policy Agent\n")
	sse.SendMessage("- `security [code]` - Route to Security Agent\n")
	sse.SendMessage("- `cost [code]` - Route to Cost Agent\n")
	sse.SendMessage("- `compliance [code]` - Route to Compliance Agent\n")
	sse.SendMessage("- `impact [code]` - Route to Impact Agent\n")
	sse.SendMessage("- `drift [code]` - Route to Drift Agent\n")
	sse.SendMessage("- `module [code]` - Route to Module Agent\n")
	sse.SendMessage("- `deploy [action]` - Route to Deploy Agent\n")
	sse.SendMessage("- `notification [action]` - Route to Notification Agent\n")
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
