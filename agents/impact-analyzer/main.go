// =============================================================================
// Impact Analyzer Copilot Agent
// =============================================================================
// A Copilot Agent that analyzes the blast radius and dependency impact of
// infrastructure changes before deployment.
//
// Features:
//   - Dependency graph analysis
//   - Blast radius calculation
//   - Risk assessment
//   - Change impact scoring
//   - Stream results via Server-Sent Events
//
// Usage:
//   go run .
//   # Server starts on :8087
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
	Port                string
	AzureSubscriptionID string
	WebhookSecret       string
	Debug               bool
}

func loadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8087"
	}

	return &Config{
		Port:                port,
		AzureSubscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		WebhookSecret:       os.Getenv("GITHUB_WEBHOOK_SECRET"),
		Debug:               os.Getenv("DEBUG") != "",
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

type Reference struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// =============================================================================
// Impact Types
// =============================================================================

type Resource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	References []string               `json:"references"`
	Line       int                    `json:"line"`
}

type ImpactResult struct {
	Resource       string   `json:"resource"`
	ChangeType     string   `json:"change_type"` // create, update, delete, replace
	RiskLevel      string   `json:"risk_level"`  // low, medium, high, critical
	BlastRadius    int      `json:"blast_radius"`
	AffectedBy     []string `json:"affected_by"`
	Affects        []string `json:"affects"`
	Considerations []string `json:"considerations"`
	Downtime       bool     `json:"downtime"`
	DataLoss       bool     `json:"data_loss"`
}

type AnalysisSummary struct {
	TotalResources int    `json:"total_resources"`
	HighRisk       int    `json:"high_risk"`
	MediumRisk     int    `json:"medium_risk"`
	LowRisk        int    `json:"low_risk"`
	MaxBlastRadius int    `json:"max_blast_radius"`
	OverallRisk    string `json:"overall_risk"`
	Recommendation string `json:"recommendation"`
}

// =============================================================================
// Server
// =============================================================================

type Server struct {
	config *Config
	mux    *http.ServeMux
}

func NewServer(config *Config) *Server {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/agent", s.handleAgent)
	s.mux.HandleFunc("/analyze", s.handleAnalyze)
	s.mux.HandleFunc("/", s.handleAgent)
}

func (s *Server) Run() error {
	addr := ":" + s.config.Port
	log.Printf("💥 Impact Analyzer Agent starting on %s", addr)
	log.Printf("📍 Endpoints:")
	log.Printf("   POST /agent   - Agent endpoint (SSE)")
	log.Printf("   POST /analyze - Analyze impact")
	log.Printf("   GET  /health  - Health check")
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": "impact-analyzer-agent",
	})
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Use POST /agent for analysis",
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
	s.processImpactAnalysis(r.Context(), req, sse)
}

func (s *Server) processImpactAnalysis(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = req.Messages[i].Content
			break
		}
	}

	sse.SendMessage("💥 **Impact Analyzer Agent**\n\n")
	sse.SendMessage("Analyzing blast radius and dependency impact...\n\n")
	time.Sleep(300 * time.Millisecond)

	code := extractCode(userMessage)
	if code == "" {
		for _, ref := range req.CopilotReferences {
			if ref.Data.Content != "" {
				code = ref.Data.Content
				break
			}
		}
	}

	if code == "" {
		sse.SendMessage("ℹ️ No IaC code detected.\n\n")
		sse.SendMessage("**Impact Analysis helps you understand:**\n")
		sse.SendMessage("- 💥 Blast radius of changes\n")
		sse.SendMessage("- 🔗 Resource dependencies\n")
		sse.SendMessage("- ⚠️ Risk assessment\n")
		sse.SendMessage("- ⏱️ Potential downtime\n")
		return
	}

	iacType := detectIaCType(code)
	sse.SendMessage(fmt.Sprintf("📝 Detected **%s** code\n\n", iacType))

	resources := s.parseResources(code, iacType)
	if len(resources) == 0 {
		sse.SendMessage("⚠️ No resources found.\n")
		return
	}

	sse.SendMessage(fmt.Sprintf("🔍 Found **%d** resource(s)\n\n", len(resources)))
	sse.SendMessage("📊 Building dependency graph...\n")
	time.Sleep(300 * time.Millisecond)

	// Analyze impact
	impacts := s.analyzeImpact(resources)
	summary := s.calculateSummary(impacts)

	// Report
	sse.SendMessage("\n## Impact Analysis Results\n\n")

	// Summary
	sse.SendMessage("### Summary\n\n")
	sse.SendMessage(fmt.Sprintf("| Metric | Value |\n"))
	sse.SendMessage(fmt.Sprintf("|--------|-------|\n"))
	sse.SendMessage(fmt.Sprintf("| Total Resources | %d |\n", summary.TotalResources))
	sse.SendMessage(fmt.Sprintf("| High Risk | %d |\n", summary.HighRisk))
	sse.SendMessage(fmt.Sprintf("| Medium Risk | %d |\n", summary.MediumRisk))
	sse.SendMessage(fmt.Sprintf("| Low Risk | %d |\n", summary.LowRisk))
	sse.SendMessage(fmt.Sprintf("| Max Blast Radius | %d resources |\n", summary.MaxBlastRadius))
	sse.SendMessage(fmt.Sprintf("| **Overall Risk** | **%s** |\n", summary.OverallRisk))
	sse.SendMessage("\n")

	// Risk icon
	riskIcon := "🟢"
	if summary.OverallRisk == "high" || summary.OverallRisk == "critical" {
		riskIcon = "🔴"
	} else if summary.OverallRisk == "medium" {
		riskIcon = "🟡"
	}
	sse.SendMessage(fmt.Sprintf("%s **Recommendation:** %s\n\n", riskIcon, summary.Recommendation))

	// Details
	sse.SendMessage("### Resource Impact Details\n\n")
	for _, impact := range impacts {
		icon := getRiskIcon(impact.RiskLevel)
		sse.SendMessage(fmt.Sprintf("#### %s `%s`\n\n", icon, impact.Resource))
		sse.SendMessage(fmt.Sprintf("- **Risk Level:** %s\n", impact.RiskLevel))
		sse.SendMessage(fmt.Sprintf("- **Blast Radius:** %d resource(s)\n", impact.BlastRadius))

		if impact.Downtime {
			sse.SendMessage("- ⚠️ **May cause downtime**\n")
		}
		if impact.DataLoss {
			sse.SendMessage("- ⚠️ **Risk of data loss**\n")
		}

		if len(impact.Affects) > 0 {
			sse.SendMessage(fmt.Sprintf("- **Affects:** %s\n", strings.Join(impact.Affects, ", ")))
		}

		if len(impact.Considerations) > 0 {
			sse.SendMessage("- **Considerations:**\n")
			for _, c := range impact.Considerations {
				sse.SendMessage(fmt.Sprintf("  - %s\n", c))
			}
		}
		sse.SendMessage("\n")
	}

	sse.SendMessage("---\n*Impact analysis completed*")
}

func (s *Server) parseResources(code, iacType string) []Resource {
	var resources []Resource
	resourcePattern := regexp.MustCompile(`(?m)^resource\s+"([^"]+)"\s+"([^"]+)"\s*\{`)
	matches := resourcePattern.FindAllStringSubmatchIndex(code, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			resourceType := code[match[2]:match[3]]
			resourceName := code[match[4]:match[5]]
			line := strings.Count(code[:match[0]], "\n") + 1

			blockStart := match[1]
			blockEnd := findMatchingBrace(code, blockStart)
			block := code[blockStart:blockEnd]

			// Find references
			refs := findReferences(block)

			resources = append(resources, Resource{
				Type:       resourceType,
				Name:       resourceName,
				Properties: parseTerraformBlock(block),
				References: refs,
				Line:       line,
			})
		}
	}
	return resources
}

func findReferences(block string) []string {
	var refs []string
	refPattern := regexp.MustCompile(`(\w+)\.(\w+)\.(\w+)`)
	matches := refPattern.FindAllStringSubmatch(block, -1)
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) >= 3 {
			ref := m[1] + "." + m[2]
			if !seen[ref] && m[1] != "var" && m[1] != "local" {
				refs = append(refs, ref)
				seen[ref] = true
			}
		}
	}
	return refs
}

func (s *Server) analyzeImpact(resources []Resource) []ImpactResult {
	var results []ImpactResult

	// Build dependency map
	dependencyMap := make(map[string][]string)
	for _, r := range resources {
		key := r.Type + "." + r.Name
		dependencyMap[key] = r.References
	}

	for _, r := range resources {
		key := r.Type + "." + r.Name
		impact := ImpactResult{
			Resource:   key,
			ChangeType: "update",
		}

		// Calculate blast radius
		affected := s.calculateBlastRadius(key, dependencyMap, make(map[string]bool))
		impact.BlastRadius = len(affected)
		impact.Affects = affected
		impact.AffectedBy = dependencyMap[key]

		// Assess risk based on resource type
		impact.RiskLevel, impact.Considerations, impact.Downtime, impact.DataLoss = s.assessRisk(r)

		results = append(results, impact)
	}

	return results
}

func (s *Server) calculateBlastRadius(resource string, deps map[string][]string, visited map[string]bool) []string {
	if visited[resource] {
		return nil
	}
	visited[resource] = true

	var affected []string
	for key, refs := range deps {
		if key == resource {
			continue
		}
		for _, ref := range refs {
			if strings.Contains(ref, strings.Split(resource, ".")[1]) {
				affected = append(affected, key)
				affected = append(affected, s.calculateBlastRadius(key, deps, visited)...)
			}
		}
	}
	return affected
}

func (s *Server) assessRisk(r Resource) (string, []string, bool, bool) {
	risk := "low"
	var considerations []string
	downtime := false
	dataLoss := false

	switch r.Type {
	case "azurerm_storage_account":
		risk = "high"
		dataLoss = true
		considerations = append(considerations, "Storage account changes may affect dependent applications")
		considerations = append(considerations, "Ensure data is backed up before modifications")
	case "azurerm_kubernetes_cluster":
		risk = "critical"
		downtime = true
		considerations = append(considerations, "AKS changes may cause cluster downtime")
		considerations = append(considerations, "Consider using node pools for rolling updates")
	case "azurerm_sql_server", "azurerm_sql_database":
		risk = "high"
		downtime = true
		dataLoss = true
		considerations = append(considerations, "Database changes may cause application downtime")
		considerations = append(considerations, "Ensure recent backups exist")
	case "azurerm_virtual_network", "azurerm_subnet":
		risk = "high"
		downtime = true
		considerations = append(considerations, "Network changes affect all connected resources")
		considerations = append(considerations, "Plan for network connectivity disruption")
	case "azurerm_key_vault":
		risk = "high"
		considerations = append(considerations, "Key Vault changes affect secret-dependent applications")
	default:
		risk = "medium"
	}

	return risk, considerations, downtime, dataLoss
}

func (s *Server) calculateSummary(impacts []ImpactResult) AnalysisSummary {
	summary := AnalysisSummary{
		TotalResources: len(impacts),
	}

	for _, i := range impacts {
		switch i.RiskLevel {
		case "critical", "high":
			summary.HighRisk++
		case "medium":
			summary.MediumRisk++
		default:
			summary.LowRisk++
		}
		if i.BlastRadius > summary.MaxBlastRadius {
			summary.MaxBlastRadius = i.BlastRadius
		}
	}

	if summary.HighRisk > 0 {
		summary.OverallRisk = "high"
		summary.Recommendation = "Review high-risk changes carefully. Consider staging deployment."
	} else if summary.MediumRisk > 0 {
		summary.OverallRisk = "medium"
		summary.Recommendation = "Proceed with caution. Monitor deployment closely."
	} else {
		summary.OverallRisk = "low"
		summary.Recommendation = "Safe to proceed with standard deployment process."
	}

	return summary
}

// Helpers
func parseTerraformBlock(block string) map[string]interface{} {
	props := make(map[string]interface{})
	kvPattern := regexp.MustCompile(`(?m)^\s*([a-z_]+)\s*=\s*(.+)$`)
	matches := kvPattern.FindAllStringSubmatch(block, -1)
	for _, m := range matches {
		if len(m) >= 3 {
			key := strings.TrimSpace(m[1])
			value := strings.TrimSpace(m[2])
			props[key] = strings.Trim(value, "\"")
		}
	}
	return props
}

func findMatchingBrace(code string, start int) int {
	depth := 0
	for i := start; i < len(code); i++ {
		switch code[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(code)
}

func detectIaCType(code string) string {
	if strings.Contains(code, "resource \"") {
		return "Terraform"
	}
	if strings.Contains(code, "resource ") && strings.Contains(code, "@") {
		return "Bicep"
	}
	return "Unknown"
}

func extractCode(message string) string {
	pattern := regexp.MustCompile("```(?:terraform|bicep|hcl)?\\s*\\n([\\s\\S]*?)\\n```")
	if matches := pattern.FindStringSubmatch(message); len(matches) > 1 {
		return matches[1]
	}
	if strings.Contains(message, "resource ") {
		return message
	}
	return ""
}

func getRiskIcon(risk string) string {
	switch risk {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	default:
		return "🟢"
	}
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
