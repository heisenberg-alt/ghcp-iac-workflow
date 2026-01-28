// =============================================================================
// Compliance Auditor Copilot Agent
// =============================================================================
// A Copilot Agent that audits Infrastructure as Code against regulatory
// compliance frameworks including CIS, NIST, SOC2, HIPAA, and PCI-DSS.
//
// Features:
//   - Multi-framework compliance checking
//   - Control mapping and validation
//   - Compliance report generation
//   - Stream results via Server-Sent Events
//
// Usage:
//   go run .
//   # Server starts on :8085
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
	Port              string
	EnabledFrameworks []string
	WebhookSecret     string
	Debug             bool
}

func loadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	frameworks := os.Getenv("COMPLIANCE_FRAMEWORKS")
	if frameworks == "" {
		frameworks = "CIS,NIST,SOC2"
	}

	return &Config{
		Port:              port,
		EnabledFrameworks: strings.Split(frameworks, ","),
		WebhookSecret:     os.Getenv("GITHUB_WEBHOOK_SECRET"),
		Debug:             os.Getenv("DEBUG") != "",
	}
}

// =============================================================================
// Copilot Agent Types
// =============================================================================

type AgentRequest struct {
	Messages          []Message          `json:"messages"`
	CopilotReferences []CopilotReference `json:"copilot_references,omitempty"`
	Streaming         bool               `json:"streaming,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
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
// Compliance Types
// =============================================================================

type ComplianceFramework struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Description string              `json:"description"`
	Controls    []ComplianceControl `json:"controls"`
}

type ComplianceControl struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Category      string         `json:"category"`
	Severity      string         `json:"severity"`
	ResourceTypes []string       `json:"resourceTypes"`
	Checks        []ControlCheck `json:"checks"`
	Remediation   string         `json:"remediation"`
	Documentation string         `json:"documentation"`
}

type ControlCheck struct {
	Property string      `json:"property"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value,omitempty"`
	Default  interface{} `json:"default,omitempty"`
}

type ComplianceFinding struct {
	Framework     string `json:"framework"`
	ControlID     string `json:"control_id"`
	ControlTitle  string `json:"control_title"`
	Category      string `json:"category"`
	Severity      string `json:"severity"`
	ResourceType  string `json:"resource_type"`
	ResourceName  string `json:"resource_name"`
	Status        string `json:"status"` // pass, fail, not_applicable
	Message       string `json:"message"`
	Remediation   string `json:"remediation"`
	Documentation string `json:"documentation"`
}

type Resource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Line       int                    `json:"line"`
}

// =============================================================================
// Server
// =============================================================================

type Server struct {
	config     *Config
	mux        *http.ServeMux
	frameworks []ComplianceFramework
}

func NewServer(config *Config) *Server {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
	}
	s.loadFrameworks()
	s.setupRoutes()
	return s
}

func (s *Server) loadFrameworks() {
	// Load compliance frameworks from JSON
	data, err := os.ReadFile("data/frameworks.json")
	if err != nil {
		log.Printf("Warning: Could not load frameworks: %v, using defaults", err)
		s.frameworks = s.getDefaultFrameworks()
		return
	}

	var config struct {
		Frameworks []ComplianceFramework `json:"frameworks"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Warning: Could not parse frameworks: %v, using defaults", err)
		s.frameworks = s.getDefaultFrameworks()
		return
	}

	s.frameworks = config.Frameworks
	log.Printf("Loaded %d compliance frameworks", len(s.frameworks))
}

func (s *Server) getDefaultFrameworks() []ComplianceFramework {
	return []ComplianceFramework{
		{
			ID:          "CIS",
			Name:        "CIS Azure Foundations Benchmark",
			Version:     "2.0",
			Description: "Center for Internet Security Azure benchmark",
			Controls: []ComplianceControl{
				{
					ID:            "CIS-3.1",
					Title:         "Ensure secure transfer required is enabled",
					Description:   "Enable data encryption in transit",
					Category:      "Storage",
					Severity:      "high",
					ResourceTypes: []string{"azurerm_storage_account"},
					Checks:        []ControlCheck{{Property: "enable_https_traffic_only", Operator: "equals", Value: true, Default: false}},
					Remediation:   "Set enable_https_traffic_only = true",
					Documentation: "https://www.cisecurity.org/benchmark/azure",
				},
				{
					ID:            "CIS-3.7",
					Title:         "Ensure public access level is disabled",
					Description:   "Disable anonymous access to blob containers",
					Category:      "Storage",
					Severity:      "critical",
					ResourceTypes: []string{"azurerm_storage_account"},
					Checks:        []ControlCheck{{Property: "allow_blob_public_access", Operator: "equals", Value: false, Default: true}},
					Remediation:   "Set allow_blob_public_access = false",
					Documentation: "https://www.cisecurity.org/benchmark/azure",
				},
				{
					ID:            "CIS-8.1",
					Title:         "Ensure RBAC is enabled on AKS clusters",
					Description:   "Enable Kubernetes RBAC for access control",
					Category:      "Containers",
					Severity:      "high",
					ResourceTypes: []string{"azurerm_kubernetes_cluster"},
					Checks:        []ControlCheck{{Property: "role_based_access_control_enabled", Operator: "equals", Value: true, Default: false}},
					Remediation:   "Set role_based_access_control_enabled = true",
					Documentation: "https://www.cisecurity.org/benchmark/azure",
				},
			},
		},
		{
			ID:          "NIST",
			Name:        "NIST SP 800-53",
			Version:     "Rev 5",
			Description: "Security and Privacy Controls for Information Systems",
			Controls: []ComplianceControl{
				{
					ID:            "NIST-SC-8",
					Title:         "Transmission Confidentiality and Integrity",
					Description:   "Protect transmitted information",
					Category:      "System and Communications Protection",
					Severity:      "high",
					ResourceTypes: []string{"azurerm_storage_account", "azurerm_app_service"},
					Checks:        []ControlCheck{{Property: "enable_https_traffic_only", Operator: "equals", Value: true, Default: false}},
					Remediation:   "Enable HTTPS-only traffic",
					Documentation: "https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final",
				},
				{
					ID:            "NIST-SC-28",
					Title:         "Protection of Information at Rest",
					Description:   "Protect information at rest",
					Category:      "System and Communications Protection",
					Severity:      "high",
					ResourceTypes: []string{"azurerm_storage_account"},
					Checks:        []ControlCheck{{Property: "min_tls_version", Operator: "equals", Value: "TLS1_2"}},
					Remediation:   "Set minimum TLS version to 1.2",
					Documentation: "https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final",
				},
				{
					ID:            "NIST-AC-6",
					Title:         "Least Privilege",
					Description:   "Employ the principle of least privilege",
					Category:      "Access Control",
					Severity:      "high",
					ResourceTypes: []string{"azurerm_kubernetes_cluster"},
					Checks:        []ControlCheck{{Property: "role_based_access_control_enabled", Operator: "equals", Value: true, Default: false}},
					Remediation:   "Enable RBAC for fine-grained access control",
					Documentation: "https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final",
				},
			},
		},
		{
			ID:          "SOC2",
			Name:        "SOC 2 Type II",
			Version:     "2017",
			Description: "Trust Services Criteria",
			Controls: []ComplianceControl{
				{
					ID:            "SOC2-CC6.1",
					Title:         "Logical Access Security",
					Description:   "Implement logical access security software",
					Category:      "Common Criteria",
					Severity:      "high",
					ResourceTypes: []string{"azurerm_kubernetes_cluster", "azurerm_key_vault"},
					Checks:        []ControlCheck{{Property: "role_based_access_control_enabled", Operator: "equals", Value: true, Default: false}},
					Remediation:   "Enable RBAC for access control",
					Documentation: "https://www.aicpa.org/soc2",
				},
				{
					ID:            "SOC2-CC6.7",
					Title:         "Transmission Security",
					Description:   "Encrypt data in transit",
					Category:      "Common Criteria",
					Severity:      "high",
					ResourceTypes: []string{"azurerm_storage_account"},
					Checks:        []ControlCheck{{Property: "enable_https_traffic_only", Operator: "equals", Value: true, Default: false}},
					Remediation:   "Enable HTTPS-only traffic",
					Documentation: "https://www.aicpa.org/soc2",
				},
			},
		},
	}
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/agent", s.handleAgent)
	s.mux.HandleFunc("/frameworks", s.handleFrameworks)
	s.mux.HandleFunc("/report", s.handleReport)
	s.mux.HandleFunc("/", s.handleAgent)
}

func (s *Server) Run() error {
	addr := ":" + s.config.Port
	log.Printf("📋 Compliance Auditor Agent starting on %s", addr)
	log.Printf("📍 Endpoints:")
	log.Printf("   POST /agent      - Agent endpoint (SSE)")
	log.Printf("   GET  /frameworks - List supported frameworks")
	log.Printf("   POST /report     - Generate compliance report")
	log.Printf("   GET  /health     - Health check")
	log.Printf("📋 Enabled frameworks: %v", s.config.EnabledFrameworks)
	return http.ListenAndServe(addr, s.mux)
}

// =============================================================================
// Health Check
// =============================================================================

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "healthy",
		"service":    "compliance-auditor-agent",
		"frameworks": s.config.EnabledFrameworks,
		"controls":   s.countControls(),
	})
}

func (s *Server) countControls() int {
	count := 0
	for _, fw := range s.frameworks {
		count += len(fw.Controls)
	}
	return count
}

// =============================================================================
// Frameworks Endpoint
// =============================================================================

func (s *Server) handleFrameworks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	summary := make([]map[string]interface{}, 0)
	for _, fw := range s.frameworks {
		summary = append(summary, map[string]interface{}{
			"id":          fw.ID,
			"name":        fw.Name,
			"version":     fw.Version,
			"description": fw.Description,
			"controls":    len(fw.Controls),
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"frameworks": summary,
	})
}

// =============================================================================
// Report Endpoint
// =============================================================================

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Use POST /agent to generate compliance report",
	})
}

// =============================================================================
// Agent Handler
// =============================================================================

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("→ Received compliance audit request")

	var req AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Setup SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	sse := NewSSEWriter(w, flusher)
	s.processComplianceAudit(r.Context(), req, sse)
}

func (s *Server) processComplianceAudit(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	// Get the last user message
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = req.Messages[i].Content
			break
		}
	}

	if userMessage == "" {
		sse.SendMessage("❌ No message found. Please provide IaC code to audit.")
		return
	}

	log.Printf("Processing compliance audit: %s", truncate(userMessage, 100))

	// Send initial message
	sse.SendMessage("📋 **Compliance Auditor Agent**\n\n")
	sse.SendMessage("Auditing your Infrastructure as Code against compliance frameworks...\n\n")
	time.Sleep(300 * time.Millisecond)

	// Extract code
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
		sse.SendMessage("**How to use:**\n")
		sse.SendMessage("- Paste Terraform or Bicep code directly\n")
		sse.SendMessage("- Reference a file from your workspace\n\n")
		sse.SendMessage("**Supported Frameworks:**\n")
		for _, fw := range s.frameworks {
			sse.SendMessage(fmt.Sprintf("- 📋 %s (%s)\n", fw.Name, fw.Version))
		}
		return
	}

	// Detect IaC type
	iacType := detectIaCType(code)
	sse.SendMessage(fmt.Sprintf("📝 Detected **%s** code\n\n", iacType))
	time.Sleep(200 * time.Millisecond)

	// Parse resources
	sse.SendMessage("🔍 Parsing resources...\n")
	resources := s.parseResources(code, iacType)

	if len(resources) == 0 {
		sse.SendMessage("\n⚠️ No resources found in the code.\n")
		return
	}

	sse.SendMessage(fmt.Sprintf("   Found **%d** resource(s)\n\n", len(resources)))
	time.Sleep(200 * time.Millisecond)

	// Run compliance audits
	sse.SendMessage("📋 Running compliance audits...\n\n")

	var allFindings []ComplianceFinding

	for _, fw := range s.frameworks {
		sse.SendMessage(fmt.Sprintf("   • Checking **%s** (%d controls)...\n", fw.Name, len(fw.Controls)))
		findings := s.auditAgainstFramework(resources, fw)
		allFindings = append(allFindings, findings...)
		time.Sleep(200 * time.Millisecond)
	}

	sse.SendMessage("\n")
	time.Sleep(300 * time.Millisecond)

	// Report results
	passes := 0
	failures := 0
	for _, f := range allFindings {
		if f.Status == "pass" {
			passes++
		} else if f.Status == "fail" {
			failures++
		}
	}

	if failures == 0 {
		sse.SendMessage("✅ **All compliance checks passed!**\n\n")
		sse.SendMessage(fmt.Sprintf("Your IaC passed **%d** compliance controls.\n\n", passes))

		sse.SendMessage("**Compliance Coverage:**\n")
		for _, fw := range s.frameworks {
			sse.SendMessage(fmt.Sprintf("- ✓ %s: Compliant\n", fw.Name))
		}
	} else {
		sse.SendMessage(fmt.Sprintf("⚠️ **Found %d compliance violation(s)**\n\n", failures))

		// Summary by framework
		sse.SendMessage("**Summary by Framework:**\n")
		fwCounts := make(map[string]int)
		for _, f := range allFindings {
			if f.Status == "fail" {
				fwCounts[f.Framework]++
			}
		}
		for fw, count := range fwCounts {
			sse.SendMessage(fmt.Sprintf("- %s: %d violation(s)\n", fw, count))
		}
		sse.SendMessage("\n")

		// Group by framework
		byFramework := make(map[string][]ComplianceFinding)
		for _, f := range allFindings {
			if f.Status == "fail" {
				byFramework[f.Framework] = append(byFramework[f.Framework], f)
			}
		}

		// Details by framework
		for fw, findings := range byFramework {
			sse.SendMessage(fmt.Sprintf("### 📋 %s Violations\n\n", fw))

			for i, f := range findings {
				icon := getSeverityIcon(f.Severity)
				sse.SendMessage(fmt.Sprintf("%d. %s **[%s] %s**\n", i+1, icon, f.ControlID, f.ControlTitle))
				sse.SendMessage(fmt.Sprintf("   - Resource: `%s.%s`\n", f.ResourceType, f.ResourceName))
				sse.SendMessage(fmt.Sprintf("   - Category: %s\n", f.Category))
				sse.SendMessage(fmt.Sprintf("   - %s\n", f.Message))
				sse.SendMessage(fmt.Sprintf("   - 💡 Fix: %s\n", f.Remediation))
				if f.Documentation != "" {
					sse.SendMessage(fmt.Sprintf("   - 📖 [Documentation](%s)\n", f.Documentation))
				}
				sse.SendMessage("\n")
			}
		}

		// Send references
		refs := []Reference{}
		seen := make(map[string]bool)
		for _, f := range allFindings {
			if f.Documentation != "" && !seen[f.Documentation] {
				refs = append(refs, Reference{
					Title: f.Framework + ": " + f.ControlID,
					URL:   f.Documentation,
				})
				seen[f.Documentation] = true
			}
		}
		if len(refs) > 0 {
			sse.SendReferences(refs)
		}
	}

	// Compliance score
	total := passes + failures
	if total > 0 {
		score := float64(passes) / float64(total) * 100
		sse.SendMessage(fmt.Sprintf("\n**Compliance Score: %.1f%%** (%d/%d controls)\n", score, passes, total))
	}

	sse.SendMessage("\n---\n*Compliance audit completed*")
}

// =============================================================================
// Compliance Auditing
// =============================================================================

func (s *Server) auditAgainstFramework(resources []Resource, framework ComplianceFramework) []ComplianceFinding {
	var findings []ComplianceFinding

	for _, control := range framework.Controls {
		for _, resource := range resources {
			// Check if control applies to this resource type
			applies := false
			for _, rt := range control.ResourceTypes {
				if rt == "*" || strings.EqualFold(rt, resource.Type) {
					applies = true
					break
				}
			}

			if !applies {
				continue
			}

			// Run all checks for this control
			allPassed := true
			failMessage := ""

			for _, check := range control.Checks {
				value := getNestedProperty(resource.Properties, check.Property)

				if value == nil && check.Default != nil {
					value = check.Default
				}

				passed := false
				switch check.Operator {
				case "equals":
					passed = fmt.Sprintf("%v", value) == fmt.Sprintf("%v", check.Value)
				case "not_equals":
					passed = fmt.Sprintf("%v", value) != fmt.Sprintf("%v", check.Value)
				case "exists":
					passed = value != nil
				case "not_exists":
					passed = value == nil
				}

				if !passed {
					allPassed = false
					failMessage = fmt.Sprintf("Property '%s' does not meet compliance requirement", check.Property)
					break
				}
			}

			status := "pass"
			if !allPassed {
				status = "fail"
			}

			findings = append(findings, ComplianceFinding{
				Framework:     framework.ID,
				ControlID:     control.ID,
				ControlTitle:  control.Title,
				Category:      control.Category,
				Severity:      control.Severity,
				ResourceType:  resource.Type,
				ResourceName:  resource.Name,
				Status:        status,
				Message:       failMessage,
				Remediation:   control.Remediation,
				Documentation: control.Documentation,
			})
		}
	}

	return findings
}

// =============================================================================
// IaC Parsing
// =============================================================================

func (s *Server) parseResources(code, iacType string) []Resource {
	switch iacType {
	case "Terraform":
		return s.parseTerraform(code)
	case "Bicep":
		return s.parseBicep(code)
	default:
		return nil
	}
}

func (s *Server) parseTerraform(code string) []Resource {
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

			var props map[string]interface{}
			if blockEnd > blockStart {
				block := code[blockStart:blockEnd]
				props = parseTerraformBlock(block)
			}

			resources = append(resources, Resource{
				Type:       resourceType,
				Name:       resourceName,
				Properties: props,
				Line:       line,
			})
		}
	}

	return resources
}

func (s *Server) parseBicep(code string) []Resource {
	var resources []Resource
	resourcePattern := regexp.MustCompile(`(?m)^resource\s+(\w+)\s+'([^']+)'\s*=\s*\{`)
	matches := resourcePattern.FindAllStringSubmatchIndex(code, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			resourceName := code[match[2]:match[3]]
			resourceType := code[match[4]:match[5]]
			tfType := bicepToTerraformType(resourceType)
			line := strings.Count(code[:match[0]], "\n") + 1

			blockStart := match[1] - 1
			blockEnd := findMatchingBrace(code, blockStart)
			block := code[blockStart : blockEnd+1]
			props := parseBicepProperties(block)

			resources = append(resources, Resource{
				Type:       tfType,
				Name:       resourceName,
				Properties: props,
				Line:       line,
			})
		}
	}

	return resources
}

// =============================================================================
// Helpers
// =============================================================================

func parseTerraformBlock(block string) map[string]interface{} {
	props := make(map[string]interface{})
	kvPattern := regexp.MustCompile(`(?m)^\s*([a-z_]+)\s*=\s*(.+)$`)

	matches := kvPattern.FindAllStringSubmatch(block, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])

			if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
				props[key] = strings.Trim(value, "\"")
			} else if value == "true" {
				props[key] = true
			} else if value == "false" {
				props[key] = false
			} else {
				props[key] = value
			}
		}
	}

	return props
}

func parseBicepProperties(block string) map[string]interface{} {
	props := make(map[string]interface{})

	bicepToTfProps := map[string]string{
		"supportsHttpsTrafficOnly": "enable_https_traffic_only",
		"minimumTlsVersion":        "min_tls_version",
		"allowBlobPublicAccess":    "allow_blob_public_access",
	}

	kvPattern := regexp.MustCompile(`(?m)^\s*(\w+):\s*(.+)$`)
	matches := kvPattern.FindAllStringSubmatch(block, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])

			if tfKey, ok := bicepToTfProps[key]; ok {
				key = tfKey
			}

			value = strings.Trim(value, "'\"")
			if value == "true" {
				props[key] = true
			} else if value == "false" {
				props[key] = false
			} else {
				props[key] = value
			}
		}
	}

	return props
}

func bicepToTerraformType(bicepType string) string {
	typeMap := map[string]string{
		"Microsoft.Storage/storageAccounts":          "azurerm_storage_account",
		"Microsoft.KeyVault/vaults":                  "azurerm_key_vault",
		"Microsoft.ContainerService/managedClusters": "azurerm_kubernetes_cluster",
	}

	parts := strings.Split(bicepType, "@")
	baseType := parts[0]

	if tfType, ok := typeMap[baseType]; ok {
		return tfType
	}
	return baseType
}

func getNestedProperty(props map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = props

	for _, part := range parts {
		if current == nil {
			return nil
		}
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
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
	if strings.Contains(code, "resource \"") || strings.Contains(code, "terraform {") {
		return "Terraform"
	}
	if strings.Contains(code, "resource ") && strings.Contains(code, "@") {
		return "Bicep"
	}
	return "Unknown"
}

func extractCode(message string) string {
	codeBlockPattern := regexp.MustCompile("```(?:terraform|bicep|hcl)?\\s*\\n([\\s\\S]*?)\\n```")
	if matches := codeBlockPattern.FindStringSubmatch(message); len(matches) > 1 {
		return matches[1]
	}
	if strings.Contains(message, "resource ") {
		return message
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func getSeverityIcon(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

// =============================================================================
// SSE Writer
// =============================================================================

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

func (s *SSEWriter) SendReferences(refs []Reference) {
	data := map[string]interface{}{"references": refs}
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(s.w, "event: copilot_references\ndata: %s\n\n", jsonData)
	s.flusher.Flush()
}

func (s *SSEWriter) SendDone() {
	fmt.Fprintf(s.w, "event: copilot_done\ndata: {}\n\n")
	s.flusher.Flush()
}

// =============================================================================
// Main
// =============================================================================

func main() {
	config := loadConfig()
	server := NewServer(config)

	if err := server.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
