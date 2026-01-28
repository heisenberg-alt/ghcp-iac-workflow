// IaC Governance GUI - Web-based dashboard with Copilot integration
// Connects to Policy Agent (8081), Cost Estimator (8082), and Orchestrator (8090)
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed frontend/dist/*
var frontendFS embed.FS

// Configuration
type Config struct {
	Port            string
	PolicyAgentURL  string
	CostAgentURL    string
	OrchestratorURL string
	EnableCORS      bool
}

// Server holds the GUI server state
type Server struct {
	config     *Config
	mux        *http.ServeMux
	upgrader   websocket.Upgrader
	clients    map[*websocket.Conn]bool
	clientsMux sync.RWMutex
	broadcast  chan WSMessage
}

// WebSocket message types
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// API Request/Response types
type AnalyzeRequest struct {
	Code     string   `json:"code"`
	FileType string   `json:"fileType"` // terraform or bicep
	Checks   []string `json:"checks"`   // policy, cost, security, compliance
}

type AnalyzeResponse struct {
	RequestID string                 `json:"requestId"`
	Status    string                 `json:"status"`
	Results   map[string]interface{} `json:"results"`
	Resources []Resource             `json:"resources"`
	Graph     *ResourceGraph         `json:"graph"`
}

type Resource struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Line       int                    `json:"line"`
}

type ResourceGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Links []GraphLink `json:"links"`
}

type GraphNode struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Status   string `json:"status"` // ok, warning, error
}

type GraphLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // depends_on, reference
}

// Agent client for connecting to backend agents
type AgentClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

type AgentRequest struct {
	Messages []AgentMessage `json:"messages"`
}

type AgentMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Dashboard status response
type DashboardStatus struct {
	Agents      map[string]AgentStatus `json:"agents"`
	LastUpdated string                 `json:"lastUpdated"`
}

type AgentStatus struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Status  string `json:"status"`  // online, offline, error
	Latency int64  `json:"latency"` // ms
}

func main() {
	config := &Config{
		Port:            getEnv("PORT", "3000"),
		PolicyAgentURL:  getEnv("POLICY_AGENT_URL", "http://localhost:8081"),
		CostAgentURL:    getEnv("COST_AGENT_URL", "http://localhost:8082"),
		OrchestratorURL: getEnv("ORCHESTRATOR_URL", "http://localhost:8090"),
		EnableCORS:      getEnv("ENABLE_CORS", "true") == "true",
	}

	server := NewServer(config)

	log.Printf("🎨 IaC Governance GUI starting on port %s", config.Port)
	log.Printf("   Policy Agent: %s", config.PolicyAgentURL)
	log.Printf("   Cost Agent: %s", config.CostAgentURL)
	log.Printf("   Orchestrator: %s", config.OrchestratorURL)
	log.Printf("   Open http://localhost:%s in your browser", config.Port)

	if err := http.ListenAndServe(":"+config.Port, server.mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func NewServer(config *Config) *Server {
	s := &Server{
		config:    config,
		mux:       http.NewServeMux(),
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan WSMessage, 100),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}

	s.setupRoutes()
	go s.handleBroadcast()

	return s
}

func (s *Server) setupRoutes() {
	// API endpoints
	s.mux.HandleFunc("/api/health", s.withCORS(s.handleHealth))
	s.mux.HandleFunc("/api/status", s.withCORS(s.handleStatus))
	s.mux.HandleFunc("/api/analyze", s.withCORS(s.handleAnalyze))
	s.mux.HandleFunc("/api/copilot", s.withCORS(s.handleCopilot))

	// WebSocket for real-time updates
	s.mux.HandleFunc("/ws", s.handleWebSocket)

	// Serve frontend static files
	s.mux.HandleFunc("/", s.handleFrontend)
}

func (s *Server) withCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.config.EnableCORS {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		handler(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := DashboardStatus{
		Agents:      make(map[string]AgentStatus),
		LastUpdated: time.Now().Format(time.RFC3339),
	}

	// Check each agent in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex

	agents := map[string]string{
		"policy":       s.config.PolicyAgentURL,
		"cost":         s.config.CostAgentURL,
		"orchestrator": s.config.OrchestratorURL,
	}

	for name, url := range agents {
		wg.Add(1)
		go func(name, url string) {
			defer wg.Done()
			agentStatus := checkAgentHealth(name, url)
			mu.Lock()
			status.Agents[name] = agentStatus
			mu.Unlock()
		}(name, url)
	}

	wg.Wait()
	json.NewEncoder(w).Encode(status)
}

func checkAgentHealth(name, baseURL string) AgentStatus {
	status := AgentStatus{
		Name:   name,
		URL:    baseURL,
		Status: "offline",
	}

	start := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		status.Status = "error"
		return status
	}
	defer resp.Body.Close()

	status.Latency = time.Since(start).Milliseconds()

	if resp.StatusCode == http.StatusOK {
		status.Status = "online"
	} else {
		status.Status = "error"
	}

	return status
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())

	// Parse resources from code
	resources := parseIaCCode(req.Code, req.FileType)
	graph := buildResourceGraph(resources)

	response := AnalyzeResponse{
		RequestID: requestID,
		Status:    "processing",
		Results:   make(map[string]interface{}),
		Resources: resources,
		Graph:     graph,
	}

	// Broadcast initial status
	s.broadcast <- WSMessage{
		Type:    "analysis_started",
		Payload: map[string]string{"requestId": requestID},
	}

	// Run requested checks in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, check := range req.Checks {
		wg.Add(1)
		go func(checkType string) {
			defer wg.Done()

			var result interface{}
			var err error

			switch checkType {
			case "policy":
				result, err = s.runPolicyCheck(req.Code)
			case "cost":
				result, err = s.runCostEstimate(req.Code)
			}

			mu.Lock()
			if err != nil {
				response.Results[checkType] = map[string]interface{}{
					"status": "error",
					"error":  err.Error(),
				}
			} else {
				response.Results[checkType] = result
			}
			mu.Unlock()

			// Broadcast progress
			s.broadcast <- WSMessage{
				Type: "check_completed",
				Payload: map[string]interface{}{
					"requestId": requestID,
					"check":     checkType,
					"result":    result,
				},
			}
		}(check)
	}

	wg.Wait()
	response.Status = "completed"

	// Broadcast completion
	s.broadcast <- WSMessage{
		Type:    "analysis_completed",
		Payload: response,
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) runPolicyCheck(code string) (interface{}, error) {
	client := &AgentClient{
		BaseURL:    s.config.PolicyAgentURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}

	return client.SendToAgent(code, "Check this infrastructure code for policy violations")
}

func (s *Server) runCostEstimate(code string) (interface{}, error) {
	client := &AgentClient{
		BaseURL:    s.config.CostAgentURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}

	return client.SendToAgent(code, "Estimate the monthly cost for this infrastructure")
}

func (c *AgentClient) SendToAgent(code, prompt string) (interface{}, error) {
	req := AgentRequest{
		Messages: []AgentMessage{
			{Role: "user", Content: fmt.Sprintf("%s\n\n```\n%s\n```", prompt, code)},
		},
	}

	body, _ := json.Marshal(req)

	resp, err := c.HTTPClient.Post(c.BaseURL+"/agent", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read SSE response and extract content
	content := ""
	data, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			jsonStr := strings.TrimPrefix(line, "data: ")
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &event); err == nil {
				if c, ok := event["content"].(string); ok {
					content += c
				}
			}
		}
	}

	return map[string]interface{}{
		"status":  "completed",
		"content": content,
	}, nil
}

func (s *Server) handleCopilot(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Setup SSE for streaming response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	if s.config.EnableCORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	var req struct {
		Message string `json:"message"`
		Context string `json:"context"` // IaC code context
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendSSEError(w, flusher, "Invalid request")
		return
	}

	// Determine which agent to use based on message content
	agentURL := s.config.OrchestratorURL
	if strings.Contains(strings.ToLower(req.Message), "cost") || strings.Contains(strings.ToLower(req.Message), "price") {
		agentURL = s.config.CostAgentURL
	} else if strings.Contains(strings.ToLower(req.Message), "policy") || strings.Contains(strings.ToLower(req.Message), "compliance") {
		agentURL = s.config.PolicyAgentURL
	}

	// Build message with context
	fullMessage := req.Message
	if req.Context != "" {
		fullMessage = fmt.Sprintf("%s\n\nCode:\n```\n%s\n```", req.Message, req.Context)
	}

	// Forward to agent and stream response
	agentReq := AgentRequest{
		Messages: []AgentMessage{
			{Role: "user", Content: fullMessage},
		},
	}

	body, _ := json.Marshal(agentReq)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(agentURL+"/agent", "application/json", strings.NewReader(string(body)))
	if err != nil {
		sendSSEError(w, flusher, fmt.Sprintf("Agent error: %v", err))
		return
	}
	defer resp.Body.Close()

	// Stream the response
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			flusher.Flush()
		}
		if err != nil {
			break
		}
	}
}

func sendSSEError(w http.ResponseWriter, flusher http.Flusher, message string) {
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", message)
	flusher.Flush()
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	s.clientsMux.Lock()
	s.clients[conn] = true
	s.clientsMux.Unlock()

	log.Printf("WebSocket client connected (total: %d)", len(s.clients))

	// Send initial status
	status, _ := json.Marshal(WSMessage{
		Type:    "connected",
		Payload: map[string]string{"status": "connected"},
	})
	conn.WriteMessage(websocket.TextMessage, status)

	// Handle incoming messages
	go func() {
		defer func() {
			s.clientsMux.Lock()
			delete(s.clients, conn)
			s.clientsMux.Unlock()
			conn.Close()
			log.Printf("WebSocket client disconnected (total: %d)", len(s.clients))
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var msg WSMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}

			// Handle client messages
			switch msg.Type {
			case "ping":
				conn.WriteJSON(WSMessage{Type: "pong"})
			}
		}
	}()
}

func (s *Server) handleBroadcast() {
	for msg := range s.broadcast {
		s.clientsMux.RLock()
		for client := range s.clients {
			if err := client.WriteJSON(msg); err != nil {
				log.Printf("WebSocket send error: %v", err)
			}
		}
		s.clientsMux.RUnlock()
	}
}

func (s *Server) handleFrontend(w http.ResponseWriter, r *http.Request) {
	// Try to serve from embedded filesystem
	subFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		// Fallback: serve development index.html
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(developmentHTML))
		return
	}

	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	// Check if file exists
	if _, err := fs.Stat(subFS, strings.TrimPrefix(path, "/")); err != nil {
		// SPA fallback: serve index.html for all routes
		path = "/index.html"
	}

	http.FileServer(http.FS(subFS)).ServeHTTP(w, r)
}

// IaC Parsing functions
func parseIaCCode(code, fileType string) []Resource {
	if fileType == "" {
		fileType = detectFileType(code)
	}

	var resources []Resource

	if fileType == "terraform" {
		resources = parseTerraform(code)
	} else if fileType == "bicep" {
		resources = parseBicep(code)
	}

	return resources
}

func detectFileType(code string) string {
	if strings.Contains(code, "resource \"azurerm_") || strings.Contains(code, "terraform {") {
		return "terraform"
	}
	if strings.Contains(code, "param ") && strings.Contains(code, "@description") {
		return "bicep"
	}
	return "terraform"
}

func parseTerraform(code string) []Resource {
	var resources []Resource
	lines := strings.Split(code, "\n")

	var currentResource *Resource
	braceCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect resource block start
		if strings.HasPrefix(trimmed, "resource \"") {
			parts := strings.Split(trimmed, "\"")
			if len(parts) >= 4 {
				currentResource = &Resource{
					Type:       parts[1],
					Name:       parts[3],
					Properties: make(map[string]interface{}),
					Line:       i + 1,
				}
				currentResource.ID = fmt.Sprintf("%s.%s", parts[1], parts[3])
			}
			braceCount = 1
			continue
		}

		if currentResource != nil {
			braceCount += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")

			// Parse simple properties
			if strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, "#") {
				parts := strings.SplitN(trimmed, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					value = strings.Trim(value, "\"")
					currentResource.Properties[key] = value
				}
			}

			if braceCount == 0 {
				resources = append(resources, *currentResource)
				currentResource = nil
			}
		}
	}

	return resources
}

func parseBicep(code string) []Resource {
	var resources []Resource
	lines := strings.Split(code, "\n")

	var currentResource *Resource
	braceCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect resource block: resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01'
		if strings.HasPrefix(trimmed, "resource ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				name := parts[1]
				typeStr := strings.Trim(parts[2], "'")
				// Extract resource type from API version string
				if idx := strings.Index(typeStr, "@"); idx > 0 {
					typeStr = typeStr[:idx]
				}
				currentResource = &Resource{
					Type:       typeStr,
					Name:       name,
					Properties: make(map[string]interface{}),
					Line:       i + 1,
				}
				currentResource.ID = fmt.Sprintf("%s.%s", typeStr, name)
			}
			if strings.Contains(trimmed, "{") {
				braceCount = 1
			}
			continue
		}

		if currentResource != nil {
			braceCount += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")

			// Parse properties
			if strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "//") {
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					value = strings.Trim(value, "'\"")
					currentResource.Properties[key] = value
				}
			}

			if braceCount == 0 {
				resources = append(resources, *currentResource)
				currentResource = nil
			}
		}
	}

	return resources
}

func buildResourceGraph(resources []Resource) *ResourceGraph {
	graph := &ResourceGraph{
		Nodes: make([]GraphNode, 0),
		Links: make([]GraphLink, 0),
	}

	// Create nodes
	for _, res := range resources {
		category := categorizeResource(res.Type)
		graph.Nodes = append(graph.Nodes, GraphNode{
			ID:       res.ID,
			Type:     res.Type,
			Name:     res.Name,
			Category: category,
			Status:   "ok",
		})
	}

	// Find dependencies
	for _, res := range resources {
		for _, otherRes := range resources {
			if res.ID == otherRes.ID {
				continue
			}

			// Check for references in properties
			for _, value := range res.Properties {
				if str, ok := value.(string); ok {
					if strings.Contains(str, otherRes.Name) {
						graph.Links = append(graph.Links, GraphLink{
							Source: res.ID,
							Target: otherRes.ID,
							Type:   "reference",
						})
					}
				}
			}
		}
	}

	return graph
}

func categorizeResource(resourceType string) string {
	rt := strings.ToLower(resourceType)

	switch {
	case strings.Contains(rt, "network") || strings.Contains(rt, "vnet") || strings.Contains(rt, "subnet"):
		return "network"
	case strings.Contains(rt, "storage"):
		return "storage"
	case strings.Contains(rt, "vm") || strings.Contains(rt, "compute"):
		return "compute"
	case strings.Contains(rt, "database") || strings.Contains(rt, "sql") || strings.Contains(rt, "cosmos"):
		return "database"
	case strings.Contains(rt, "keyvault") || strings.Contains(rt, "security"):
		return "security"
	case strings.Contains(rt, "app") || strings.Contains(rt, "function") || strings.Contains(rt, "container"):
		return "application"
	default:
		return "other"
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Development HTML served when frontend is not built
const developmentHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>IaC Governance GUI - Development</title>
    <style>
        body { font-family: system-ui; max-width: 800px; margin: 50px auto; padding: 20px; }
        .status { padding: 20px; background: #f0f0f0; border-radius: 8px; margin: 20px 0; }
        code { background: #e0e0e0; padding: 2px 6px; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>🎨 IaC Governance GUI</h1>
    <div class="status">
        <h3>Development Mode</h3>
        <p>The React frontend is not built yet. Run these commands:</p>
        <pre><code>cd frontend
npm install
npm run build</code></pre>
        <p>Or for development with hot reload:</p>
        <pre><code>cd frontend
npm install
npm run dev</code></pre>
    </div>
    <h3>API Endpoints Available:</h3>
    <ul>
        <li><code>GET /api/health</code> - Health check</li>
        <li><code>GET /api/status</code> - Agent status</li>
        <li><code>POST /api/analyze</code> - Analyze IaC code</li>
        <li><code>POST /api/copilot</code> - Copilot chat (SSE)</li>
        <li><code>WS /ws</code> - WebSocket for real-time updates</li>
    </ul>
</body>
</html>`
