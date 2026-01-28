// Enterprise IaC Governance Dashboard
// Simple dashboard showing all 10 agents with real-time status
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

//go:embed static/*
var staticFS embed.FS

// Agent configuration
type Agent struct {
	Name        string `json:"name"`
	Port        int    `json:"port"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Status      string `json:"status"`
	Details     string `json:"details"`
}

// Platform status response
type PlatformStatus struct {
	Timestamp    string  `json:"timestamp"`
	TotalAgents  int     `json:"totalAgents"`
	OnlineAgents int     `json:"onlineAgents"`
	Agents       []Agent `json:"agents"`
}

// Server state
type Server struct {
	agents []Agent
	mu     sync.RWMutex
}

var defaultAgents = []Agent{
	{Name: "Policy Checker", Port: 8081, Description: "Validates IaC against organization policies", Icon: "📋"},
	{Name: "Cost Estimator", Port: 8082, Description: "Estimates Azure resource costs", Icon: "💰"},
	{Name: "Drift Detector", Port: 8083, Description: "Detects infrastructure drift from IaC state", Icon: "🔄"},
	{Name: "Security Scanner", Port: 8084, Description: "Scans for security vulnerabilities", Icon: "🔒"},
	{Name: "Compliance Auditor", Port: 8085, Description: "Audits against CIS, NIST, SOC2 frameworks", Icon: "✅"},
	{Name: "Module Registry", Port: 8086, Description: "Manages approved IaC modules", Icon: "📦"},
	{Name: "Impact Analyzer", Port: 8087, Description: "Analyzes blast radius of changes", Icon: "💥"},
	{Name: "Deploy Promoter", Port: 8088, Description: "Manages environment promotions", Icon: "🚀"},
	{Name: "Notification Manager", Port: 8089, Description: "Sends alerts via Teams/Slack/Email", Icon: "🔔"},
	{Name: "Orchestrator", Port: 8090, Description: "Central coordinator for all agents", Icon: "🎯"},
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	server := &Server{
		agents: defaultAgents,
	}

	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/status", server.handleStatus)
	mux.HandleFunc("/api/agent/", server.handleAgentProxy)
	mux.HandleFunc("/api/test", server.handleTest)

	// Serve static files
	mux.HandleFunc("/", server.handleStatic)

	fmt.Printf(`
╔══════════════════════════════════════════════════════════════════╗
║     Enterprise IaC Governance Dashboard                          ║
║                                                                  ║
║     Dashboard URL: http://localhost:%s                         ║
║                                                                  ║
║     Monitoring 10 agents on ports 8081-8090                      ║
╚══════════════════════════════════════════════════════════════════╝
`, port)

	log.Printf("Starting Enterprise Dashboard on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// For video files, serve from filesystem (not embedded)
	if strings.HasSuffix(path, ".mp4") {
		// path is like "/static/demo.mp4", we need "static/demo.mp4"
		videoPath := strings.TrimPrefix(path, "/")

		if data, err := os.ReadFile(videoPath); err == nil {
			w.Header().Set("Content-Type", "video/mp4")
			w.Header().Set("Accept-Ranges", "bytes")
			w.Write(data)
			return
		}
		http.Error(w, "Video not found", http.StatusNotFound)
		return
	}

	content, err := staticFS.ReadFile("static" + path)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Set content type
	if strings.HasSuffix(path, ".html") {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	} else if strings.HasSuffix(path, ".css") {
		w.Header().Set("Content-Type", "text/css")
	} else if strings.HasSuffix(path, ".js") {
		w.Header().Set("Content-Type", "application/javascript")
	}

	w.Write(content)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	s.mu.Lock()
	defer s.mu.Unlock()

	onlineCount := 0
	updatedAgents := make([]Agent, len(s.agents))

	for i, agent := range s.agents {
		updatedAgents[i] = agent
		status, details := checkAgentHealth(agent.Port)
		updatedAgents[i].Status = status
		updatedAgents[i].Details = details
		if status == "online" {
			onlineCount++
		}
	}

	response := PlatformStatus{
		Timestamp:    time.Now().Format(time.RFC3339),
		TotalAgents:  len(updatedAgents),
		OnlineAgents: onlineCount,
		Agents:       updatedAgents,
	}

	json.NewEncoder(w).Encode(response)
}

func checkAgentHealth(port int) (string, string) {
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://localhost:%d/health", port)

	resp, err := client.Get(url)
	if err != nil {
		return "offline", "Connection failed"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "error", fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var healthData map[string]interface{}
	if err := json.Unmarshal(body, &healthData); err == nil {
		// Extract useful details
		details := []string{}
		if rules, ok := healthData["policy_rules"].(float64); ok {
			details = append(details, fmt.Sprintf("%d rules", int(rules)))
		}
		if rules, ok := healthData["security_rules"].(float64); ok {
			details = append(details, fmt.Sprintf("%d security rules", int(rules)))
		}
		if frameworks, ok := healthData["frameworks"].([]interface{}); ok {
			details = append(details, fmt.Sprintf("%d frameworks", len(frameworks)))
		}
		if modules, ok := healthData["approved_modules"].(float64); ok {
			details = append(details, fmt.Sprintf("%d modules", int(modules)))
		}
		if agents, ok := healthData["agents"].(float64); ok {
			details = append(details, fmt.Sprintf("%d sub-agents", int(agents)))
		}
		if controls, ok := healthData["controls"].(float64); ok {
			details = append(details, fmt.Sprintf("%d controls", int(controls)))
		}
		if len(details) > 0 {
			return "online", strings.Join(details, ", ")
		}
	}

	return "online", "Healthy"
}

func (s *Server) handleAgentProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	// Extract port from path: /api/agent/8081
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	port := parts[3]

	// Forward request to agent
	agentURL := fmt.Sprintf("http://localhost:%s/agent", port)

	body, _ := io.ReadAll(r.Body)
	resp, err := http.Post(agentURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	// Stream the SSE response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	io.Copy(w, resp.Body)
}

func (s *Server) handleTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == "OPTIONS" {
		return
	}

	// Run a quick test on all agents
	results := make(map[string]interface{})

	testCode := `resource "azurerm_storage_account" "test" {
  name                     = "teststorage"
  resource_group_name      = "test-rg"
  location                 = "eastus"
  account_tier             = "Standard"
  account_replication_type = "GRS"
}`

	// Test each agent
	for _, agent := range defaultAgents {
		result := testAgent(agent.Port, testCode)
		results[agent.Name] = result
	}

	json.NewEncoder(w).Encode(results)
}

func testAgent(port int, code string) map[string]interface{} {
	client := &http.Client{Timeout: 10 * time.Second}

	reqBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": code},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("http://localhost:%d/agent", port)
	resp, err := client.Post(url, "application/json", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return map[string]interface{}{"status": "error", "message": err.Error()}
	}
	defer resp.Body.Close()

	return map[string]interface{}{"status": "ok", "code": resp.StatusCode}
}
