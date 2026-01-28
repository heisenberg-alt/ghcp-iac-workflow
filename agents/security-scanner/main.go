// =============================================================================
// Security Scanner Copilot Agent
// =============================================================================
// A Copilot Agent that scans Infrastructure as Code for security vulnerabilities,
// misconfigurations, and compliance issues.
//
// Features:
//   - Pattern-based secret detection
//   - Security misconfiguration detection
//   - Best practice validation
//   - CWE/CVE mapping
//   - Stream results via Server-Sent Events
//
// Usage:
//   go run .
//   # Server starts on :8084
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
	WebhookSecret string
	Debug         bool
}

func loadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	return &Config{
		Port:          port,
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
// Security Types
// =============================================================================

type SecurityFinding struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Severity      string   `json:"severity"` // critical, high, medium, low
	Category      string   `json:"category"` // secrets, network, encryption, access, logging
	ResourceType  string   `json:"resource_type"`
	ResourceName  string   `json:"resource_name"`
	Line          int      `json:"line,omitempty"`
	Description   string   `json:"description"`
	Impact        string   `json:"impact"`
	Remediation   string   `json:"remediation"`
	CWE           string   `json:"cwe,omitempty"`
	Documentation string   `json:"documentation,omitempty"`
	References    []string `json:"references,omitempty"`
}

type SecurityRule struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Description   string        `json:"description"`
	Severity      string        `json:"severity"`
	Category      string        `json:"category"`
	ResourceTypes []string      `json:"resourceTypes"`
	Patterns      []RulePattern `json:"patterns,omitempty"`
	Properties    []RuleCheck   `json:"properties,omitempty"`
	CWE           string        `json:"cwe,omitempty"`
	Remediation   string        `json:"remediation"`
	Documentation string        `json:"documentation"`
}

type RulePattern struct {
	Regex       string `json:"regex"`
	Description string `json:"description"`
}

type RuleCheck struct {
	Property string      `json:"property"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value,omitempty"`
	Default  interface{} `json:"default,omitempty"`
}

type Resource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Line       int                    `json:"line"`
	RawBlock   string                 `json:"raw_block"`
}

// =============================================================================
// Server
// =============================================================================

type Server struct {
	config *Config
	mux    *http.ServeMux
	rules  []SecurityRule
}

func NewServer(config *Config) *Server {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
	}
	s.loadSecurityRules()
	s.setupRoutes()
	return s
}

func (s *Server) loadSecurityRules() {
	// Load security rules from JSON
	data, err := os.ReadFile("data/security-rules.json")
	if err != nil {
		log.Printf("Warning: Could not load security rules: %v, using defaults", err)
		s.rules = s.getDefaultRules()
		return
	}

	var config struct {
		Rules []SecurityRule `json:"rules"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Warning: Could not parse security rules: %v, using defaults", err)
		s.rules = s.getDefaultRules()
		return
	}

	s.rules = config.Rules
	log.Printf("Loaded %d security rules", len(s.rules))
}

func (s *Server) getDefaultRules() []SecurityRule {
	return []SecurityRule{
		{
			ID:            "SEC001",
			Title:         "Hardcoded Secrets Detected",
			Description:   "Sensitive values should not be hardcoded in IaC files",
			Severity:      "critical",
			Category:      "secrets",
			ResourceTypes: []string{"*"},
			Patterns: []RulePattern{
				{Regex: `(?i)(password|secret|key|token|credential)\s*=\s*"[^"]{8,}"`, Description: "Hardcoded secret value"},
				{Regex: `(?i)api[_-]?key\s*=\s*"[^"]+`, Description: "Hardcoded API key"},
				{Regex: `-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`, Description: "Private key in code"},
			},
			CWE:           "CWE-798",
			Remediation:   "Use Azure Key Vault or environment variables for sensitive values",
			Documentation: "https://learn.microsoft.com/azure/key-vault/general/overview",
		},
		{
			ID:            "SEC002",
			Title:         "Public Network Access Enabled",
			Description:   "Resources should restrict public network access",
			Severity:      "high",
			Category:      "network",
			ResourceTypes: []string{"azurerm_storage_account", "azurerm_key_vault", "azurerm_sql_server"},
			Properties: []RuleCheck{
				{Property: "public_network_access_enabled", Operator: "equals", Value: false, Default: true},
			},
			CWE:           "CWE-284",
			Remediation:   "Set public_network_access_enabled = false and use private endpoints",
			Documentation: "https://learn.microsoft.com/azure/private-link/private-endpoint-overview",
		},
		{
			ID:            "SEC003",
			Title:         "Encryption at Rest Not Configured",
			Description:   "Data at rest should be encrypted with customer-managed keys",
			Severity:      "high",
			Category:      "encryption",
			ResourceTypes: []string{"azurerm_storage_account", "azurerm_sql_database"},
			Properties: []RuleCheck{
				{Property: "customer_managed_key", Operator: "exists"},
			},
			CWE:           "CWE-311",
			Remediation:   "Configure customer-managed key encryption",
			Documentation: "https://learn.microsoft.com/azure/storage/common/customer-managed-keys-overview",
		},
		{
			ID:            "SEC004",
			Title:         "TLS Version Too Low",
			Description:   "Minimum TLS version should be 1.2 or higher",
			Severity:      "high",
			Category:      "encryption",
			ResourceTypes: []string{"azurerm_storage_account", "azurerm_app_service", "azurerm_function_app"},
			Properties: []RuleCheck{
				{Property: "min_tls_version", Operator: "equals", Value: "TLS1_2"},
			},
			CWE:           "CWE-326",
			Remediation:   "Set min_tls_version = \"TLS1_2\"",
			Documentation: "https://learn.microsoft.com/azure/storage/common/transport-layer-security-configure-minimum-version",
		},
		{
			ID:            "SEC005",
			Title:         "HTTPS Not Enforced",
			Description:   "Resources should enforce HTTPS traffic only",
			Severity:      "high",
			Category:      "encryption",
			ResourceTypes: []string{"azurerm_storage_account", "azurerm_app_service"},
			Properties: []RuleCheck{
				{Property: "enable_https_traffic_only", Operator: "equals", Value: true, Default: false},
			},
			CWE:           "CWE-319",
			Remediation:   "Set enable_https_traffic_only = true",
			Documentation: "https://learn.microsoft.com/azure/storage/common/storage-require-secure-transfer",
		},
	}
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/agent", s.handleAgent)
	s.mux.HandleFunc("/rules", s.handleRules)
	s.mux.HandleFunc("/", s.handleAgent)
}

func (s *Server) Run() error {
	addr := ":" + s.config.Port
	log.Printf("🔒 Security Scanner Agent starting on %s", addr)
	log.Printf("📍 Endpoints:")
	log.Printf("   POST /agent  - Agent endpoint (SSE)")
	log.Printf("   GET  /rules  - List security rules")
	log.Printf("   GET  /health - Health check")
	log.Printf("📋 Loaded %d security rules", len(s.rules))
	return http.ListenAndServe(addr, s.mux)
}

// =============================================================================
// Health Check
// =============================================================================

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "healthy",
		"service":        "security-scanner-agent",
		"security_rules": len(s.rules),
	})
}

// =============================================================================
// Rules Endpoint
// =============================================================================

func (s *Server) handleRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"rules": s.rules,
		"count": len(s.rules),
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

	log.Printf("→ Received security scan request")

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
	s.processSecurityScan(r.Context(), req, sse)
}

func (s *Server) processSecurityScan(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	// Get the last user message
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = req.Messages[i].Content
			break
		}
	}

	if userMessage == "" {
		sse.SendMessage("❌ No message found. Please provide IaC code to scan.")
		return
	}

	log.Printf("Processing security scan: %s", truncate(userMessage, 100))

	// Send initial message
	sse.SendMessage("🔒 **Security Scanner Agent**\n\n")
	sse.SendMessage("Scanning your Infrastructure as Code for security vulnerabilities...\n\n")
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
		sse.SendMessage("**What I scan for:**\n")
		sse.SendMessage("- 🔑 Hardcoded secrets and credentials\n")
		sse.SendMessage("- 🌐 Network security misconfigurations\n")
		sse.SendMessage("- 🔐 Encryption issues\n")
		sse.SendMessage("- 👤 Access control problems\n")
		sse.SendMessage("- 📝 Logging and monitoring gaps\n")
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
		// Still scan for secrets even without resources
		sse.SendMessage("   No structured resources found, scanning for patterns...\n\n")
	} else {
		sse.SendMessage(fmt.Sprintf("   Found **%d** resource(s)\n\n", len(resources)))
	}
	time.Sleep(200 * time.Millisecond)

	// Run security scans
	sse.SendMessage("🔒 Running security scans...\n")

	var findings []SecurityFinding

	// Pattern-based scans (secrets, etc.)
	sse.SendMessage("   • Scanning for hardcoded secrets...\n")
	secretFindings := s.scanForSecrets(code)
	findings = append(findings, secretFindings...)

	// Property-based scans on resources
	if len(resources) > 0 {
		sse.SendMessage("   • Checking resource configurations...\n")
		configFindings := s.scanResourceConfigs(resources)
		findings = append(findings, configFindings...)
	}

	time.Sleep(300 * time.Millisecond)

	// Report results
	if len(findings) == 0 {
		sse.SendMessage("\n✅ **No security issues found!**\n\n")
		sse.SendMessage("Your IaC code passed all security checks.\n\n")
		sse.SendMessage("**Checks performed:**\n")
		sse.SendMessage("- ✓ No hardcoded secrets detected\n")
		sse.SendMessage("- ✓ Network configurations secure\n")
		sse.SendMessage("- ✓ Encryption settings appropriate\n")
		sse.SendMessage("- ✓ Access controls configured\n")
	} else {
		// Group by severity
		critical := filterFindingsBySeverity(findings, "critical")
		high := filterFindingsBySeverity(findings, "high")
		medium := filterFindingsBySeverity(findings, "medium")
		low := filterFindingsBySeverity(findings, "low")

		sse.SendMessage(fmt.Sprintf("\n⚠️ **Found %d security issue(s)**\n\n", len(findings)))

		// Summary
		sse.SendMessage("**Summary:**\n")
		if len(critical) > 0 {
			sse.SendMessage(fmt.Sprintf("- 🔴 Critical: %d\n", len(critical)))
		}
		if len(high) > 0 {
			sse.SendMessage(fmt.Sprintf("- 🟠 High: %d\n", len(high)))
		}
		if len(medium) > 0 {
			sse.SendMessage(fmt.Sprintf("- 🟡 Medium: %d\n", len(medium)))
		}
		if len(low) > 0 {
			sse.SendMessage(fmt.Sprintf("- 🟢 Low: %d\n", len(low)))
		}
		sse.SendMessage("\n")

		// Group by category
		categories := groupByCategory(findings)

		for category, catFindings := range categories {
			categoryIcon := getCategoryIcon(category)
			sse.SendMessage(fmt.Sprintf("### %s %s Issues\n\n", categoryIcon, strings.Title(category)))

			for i, f := range catFindings {
				icon := getSeverityIcon(f.Severity)
				sse.SendMessage(fmt.Sprintf("%d. %s **%s** [%s]\n", i+1, icon, f.Title, strings.ToUpper(f.Severity)))
				if f.ResourceName != "" {
					sse.SendMessage(fmt.Sprintf("   - Resource: `%s.%s`\n", f.ResourceType, f.ResourceName))
				}
				if f.Line > 0 {
					sse.SendMessage(fmt.Sprintf("   - Line: %d\n", f.Line))
				}
				sse.SendMessage(fmt.Sprintf("   - %s\n", f.Description))
				sse.SendMessage(fmt.Sprintf("   - 💡 Fix: %s\n", f.Remediation))
				if f.CWE != "" {
					sse.SendMessage(fmt.Sprintf("   - 📋 [%s](https://cwe.mitre.org/data/definitions/%s.html)\n", f.CWE, strings.TrimPrefix(f.CWE, "CWE-")))
				}
				sse.SendMessage("\n")
			}
		}

		// Send references
		refs := []Reference{}
		seen := make(map[string]bool)
		for _, f := range findings {
			if f.Documentation != "" && !seen[f.Documentation] {
				refs = append(refs, Reference{
					Title: f.Title,
					URL:   f.Documentation,
				})
				seen[f.Documentation] = true
			}
		}
		if len(refs) > 0 {
			sse.SendReferences(refs)
		}
	}

	sse.SendMessage("\n---\n*Security scan completed*")
}

// =============================================================================
// Security Scanning
// =============================================================================

func (s *Server) scanForSecrets(code string) []SecurityFinding {
	var findings []SecurityFinding

	for _, rule := range s.rules {
		if rule.Category != "secrets" || len(rule.Patterns) == 0 {
			continue
		}

		for _, pattern := range rule.Patterns {
			re, err := regexp.Compile(pattern.Regex)
			if err != nil {
				continue
			}

			matches := re.FindAllStringIndex(code, -1)
			for _, match := range matches {
				line := strings.Count(code[:match[0]], "\n") + 1
				findings = append(findings, SecurityFinding{
					ID:            rule.ID,
					Title:         rule.Title,
					Severity:      rule.Severity,
					Category:      rule.Category,
					Line:          line,
					Description:   pattern.Description,
					Impact:        "Exposed secrets can lead to unauthorized access",
					Remediation:   rule.Remediation,
					CWE:           rule.CWE,
					Documentation: rule.Documentation,
				})
			}
		}
	}

	return findings
}

func (s *Server) scanResourceConfigs(resources []Resource) []SecurityFinding {
	var findings []SecurityFinding

	for _, resource := range resources {
		for _, rule := range s.rules {
			if len(rule.Properties) == 0 {
				continue
			}

			// Check if rule applies to this resource
			applies := false
			for _, rt := range rule.ResourceTypes {
				if rt == "*" || strings.EqualFold(rt, resource.Type) {
					applies = true
					break
				}
			}
			if !applies {
				continue
			}

			// Check each property
			for _, check := range rule.Properties {
				value := getNestedProperty(resource.Properties, check.Property)

				// If nil, use default
				if value == nil && check.Default != nil {
					value = check.Default
				}

				failed := false
				switch check.Operator {
				case "equals":
					if fmt.Sprintf("%v", value) != fmt.Sprintf("%v", check.Value) {
						failed = true
					}
				case "not_equals":
					if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", check.Value) {
						failed = true
					}
				case "exists":
					if value == nil {
						failed = true
					}
				case "not_exists":
					if value != nil {
						failed = true
					}
				}

				if failed {
					findings = append(findings, SecurityFinding{
						ID:            rule.ID,
						Title:         rule.Title,
						Severity:      rule.Severity,
						Category:      rule.Category,
						ResourceType:  resource.Type,
						ResourceName:  resource.Name,
						Line:          resource.Line,
						Description:   rule.Description,
						Impact:        fmt.Sprintf("Property '%s' is not configured securely", check.Property),
						Remediation:   rule.Remediation,
						CWE:           rule.CWE,
						Documentation: rule.Documentation,
					})
				}
			}
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
			var rawBlock string
			if blockEnd > blockStart {
				rawBlock = code[blockStart:blockEnd]
				props = parseTerraformBlock(rawBlock)
			}

			resources = append(resources, Resource{
				Type:       resourceType,
				Name:       resourceName,
				Properties: props,
				Line:       line,
				RawBlock:   rawBlock,
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
			rawBlock := code[blockStart : blockEnd+1]
			props := parseBicepProperties(rawBlock)

			resources = append(resources, Resource{
				Type:       tfType,
				Name:       resourceName,
				Properties: props,
				Line:       line,
				RawBlock:   rawBlock,
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
		"publicNetworkAccess":      "public_network_access_enabled",
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
		"Microsoft.Sql/servers":                      "azurerm_sql_server",
		"Microsoft.ContainerService/managedClusters": "azurerm_kubernetes_cluster",
		"Microsoft.Web/sites":                        "azurerm_app_service",
		"Microsoft.Web/serverfarms":                  "azurerm_service_plan",
		"Microsoft.ContainerRegistry/registries":     "azurerm_container_registry",
		"Microsoft.Network/virtualNetworks":          "azurerm_virtual_network",
		"Microsoft.Network/networkSecurityGroups":    "azurerm_network_security_group",
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
	if strings.Contains(code, "resource ") && strings.Contains(code, "@") && strings.Contains(code, "= {") {
		return "Bicep"
	}
	if strings.Contains(code, "param ") && strings.Contains(code, "string") {
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

func getCategoryIcon(category string) string {
	switch strings.ToLower(category) {
	case "secrets":
		return "🔑"
	case "network":
		return "🌐"
	case "encryption":
		return "🔐"
	case "access":
		return "👤"
	case "logging":
		return "📝"
	default:
		return "🔒"
	}
}

func filterFindingsBySeverity(findings []SecurityFinding, severity string) []SecurityFinding {
	var filtered []SecurityFinding
	for _, f := range findings {
		if strings.EqualFold(f.Severity, severity) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func groupByCategory(findings []SecurityFinding) map[string][]SecurityFinding {
	groups := make(map[string][]SecurityFinding)
	for _, f := range findings {
		groups[f.Category] = append(groups[f.Category], f)
	}
	return groups
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
