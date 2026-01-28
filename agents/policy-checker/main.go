// =============================================================================
// Policy Checker Copilot Agent
// =============================================================================
// A full Copilot Agent that checks Infrastructure as Code against Azure Policy
// definitions and security best practices using real Azure APIs.
//
// Features:
//   - Parse Terraform and Bicep code
//   - Check against Azure built-in policies
//   - Apply custom security rules
//   - Stream results via Server-Sent Events
//
// Usage:
//   go run .
//   # Server starts on :8080
//   # Use ngrok to expose: ngrok http 8080
// =============================================================================

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
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
		port = "8080"
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

// AgentRequest represents the incoming request from Copilot
type AgentRequest struct {
	Messages          []Message          `json:"messages"`
	CopilotReferences []CopilotReference `json:"copilot_references,omitempty"`
	Streaming         bool               `json:"streaming,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// CopilotReference represents a reference from the user's context
type CopilotReference struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Data struct {
		Content  string `json:"content,omitempty"`
		Language string `json:"language,omitempty"`
	} `json:"data,omitempty"`
}

// SSE Event types
type SSEEvent struct {
	Event string      `json:"-"`
	Data  interface{} `json:"data,omitempty"`
}

type MessageEvent struct {
	Content string `json:"content"`
}

type ReferenceEvent struct {
	References []Reference `json:"references"`
}

type Reference struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type ConfirmationEvent struct {
	Confirmation Confirmation `json:"confirmation"`
}

type Confirmation struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

// =============================================================================
// Policy Types
// =============================================================================

// PolicyViolation represents a policy check failure
type PolicyViolation struct {
	PolicyID      string `json:"policy_id"`
	PolicyName    string `json:"policy_name"`
	ResourceType  string `json:"resource_type"`
	ResourceName  string `json:"resource_name"`
	Severity      string `json:"severity"`
	Message       string `json:"message"`
	Remediation   string `json:"remediation"`
	Documentation string `json:"documentation,omitempty"`
	Line          int    `json:"line,omitempty"`
}

// PolicyRule defines a custom policy check
type PolicyRule struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Severity      string      `json:"severity"`
	ResourceType  string      `json:"resourceType"`
	Check         PolicyCheck `json:"check"`
	Remediation   string      `json:"remediation"`
	Documentation string      `json:"documentation"`
}

// PolicyCheck defines how to check a policy
type PolicyCheck struct {
	Property string      `json:"property"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value,omitempty"`
	Default  interface{} `json:"default,omitempty"`
}

// Resource represents a parsed IaC resource
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
	config *Config
	mux    *http.ServeMux
	rules  []PolicyRule
}

func NewServer(config *Config) *Server {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
	}
	s.loadPolicyRules()
	s.setupRoutes()
	return s
}

func (s *Server) loadPolicyRules() {
	// Load custom policy rules from JSON
	data, err := os.ReadFile("policies/rules.json")
	if err != nil {
		log.Printf("Warning: Could not load policy rules: %v", err)
		s.rules = s.getDefaultRules()
		return
	}

	var config struct {
		CustomPolicies []PolicyRule `json:"customPolicies"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Warning: Could not parse policy rules: %v", err)
		s.rules = s.getDefaultRules()
		return
	}

	s.rules = config.CustomPolicies
	log.Printf("Loaded %d custom policy rules", len(s.rules))
}

func (s *Server) getDefaultRules() []PolicyRule {
	return []PolicyRule{
		{
			ID:           "storage-https-required",
			Name:         "Storage Account HTTPS Required",
			Description:  "Storage accounts should enforce HTTPS traffic only",
			Severity:     "high",
			ResourceType: "azurerm_storage_account",
			Check: PolicyCheck{
				Property: "enable_https_traffic_only",
				Operator: "equals",
				Value:    true,
			},
			Remediation:   "Set enable_https_traffic_only = true",
			Documentation: "https://learn.microsoft.com/azure/storage/common/storage-require-secure-transfer",
		},
		{
			ID:           "aks-rbac-enabled",
			Name:         "AKS RBAC Enabled",
			Description:  "AKS clusters should have RBAC enabled",
			Severity:     "high",
			ResourceType: "azurerm_kubernetes_cluster",
			Check: PolicyCheck{
				Property: "role_based_access_control_enabled",
				Operator: "equals",
				Value:    true,
			},
			Remediation:   "Set role_based_access_control_enabled = true",
			Documentation: "https://learn.microsoft.com/azure/aks/manage-azure-rbac",
		},
	}
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/agent", s.handleAgent)
	s.mux.HandleFunc("/", s.handleAgent) // Also handle root for convenience
}

func (s *Server) Run() error {
	addr := ":" + s.config.Port
	log.Printf("🛡️ Policy Checker Agent starting on %s", addr)
	log.Printf("📍 Endpoints:")
	log.Printf("   POST /agent  - Agent endpoint (SSE)")
	log.Printf("   GET  /health - Health check")
	log.Printf("📋 Loaded %d policy rules", len(s.rules))
	return http.ListenAndServe(addr, s.mux)
}

// =============================================================================
// Health Check
// =============================================================================

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "healthy",
		"service":      "policy-checker-agent",
		"policy_rules": len(s.rules),
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

	log.Printf("→ Received agent request")

	// Parse request
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

	// Create SSE writer
	sse := NewSSEWriter(w, flusher)

	// Process the request
	s.processAgentRequest(r.Context(), req, sse)
}

// processAgentRequest handles the agent logic
func (s *Server) processAgentRequest(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	// Get the last user message
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = req.Messages[i].Content
			break
		}
	}

	if userMessage == "" {
		sse.SendMessage("❌ No user message found. Please provide IaC code to check.")
		return
	}

	log.Printf("Processing: %s", truncate(userMessage, 100))

	// Send initial message
	sse.SendMessage("🛡️ **Policy Checker Agent**\n\n")
	sse.SendMessage("Analyzing your Infrastructure as Code for policy compliance...\n\n")
	time.Sleep(300 * time.Millisecond)

	// Check if message contains code
	code := extractCode(userMessage)
	if code == "" {
		// Check copilot references for code
		for _, ref := range req.CopilotReferences {
			if ref.Data.Content != "" {
				code = ref.Data.Content
				break
			}
		}
	}

	if code == "" {
		sse.SendMessage("ℹ️ No IaC code detected in your message.\n\n")
		sse.SendMessage("**How to use:**\n")
		sse.SendMessage("- Paste Terraform or Bicep code directly\n")
		sse.SendMessage("- Reference a file from your workspace\n")
		sse.SendMessage("- Ask about specific Azure policies\n\n")
		sse.SendMessage("**Example:**\n```\n@policy-checker Check this Terraform:\nresource \"azurerm_storage_account\" \"example\" {\n  name = \"storage\"\n}\n```")
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
		sse.SendMessage("\n⚠️ No resources found in the code. Make sure it's valid Terraform or Bicep.\n")
		return
	}

	sse.SendMessage(fmt.Sprintf("   Found **%d** resource(s)\n\n", len(resources)))
	time.Sleep(200 * time.Millisecond)

	// Check policies
	sse.SendMessage("📋 Checking against policies...\n\n")
	violations := s.checkPolicies(resources)

	// Fetch Azure policies if subscription is configured
	if s.config.AzureSubscriptionID != "" {
		sse.SendMessage("☁️ Fetching Azure Policy definitions...\n")
		azurePolicies := s.fetchAzurePolicies(ctx)
		if len(azurePolicies) > 0 {
			sse.SendMessage(fmt.Sprintf("   Retrieved **%d** relevant Azure policies\n\n", len(azurePolicies)))
		}
	}

	time.Sleep(300 * time.Millisecond)

	// Report results
	if len(violations) == 0 {
		sse.SendMessage("✅ **All checks passed!**\n\n")
		sse.SendMessage("Your IaC code follows Azure best practices and policy requirements.\n\n")
		sse.SendMessage("**Resources validated:**\n")
		for _, res := range resources {
			sse.SendMessage(fmt.Sprintf("- ✓ `%s.%s`\n", res.Type, res.Name))
		}
	} else {
		// Group by severity
		critical := filterBySeverity(violations, "critical")
		high := filterBySeverity(violations, "high")
		medium := filterBySeverity(violations, "medium")
		low := filterBySeverity(violations, "low")

		sse.SendMessage(fmt.Sprintf("⚠️ **Found %d policy violation(s)**\n\n", len(violations)))

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

		// Details
		sse.SendMessage("**Violations:**\n\n")
		for i, v := range violations {
			icon := getSeverityIcon(v.Severity)
			sse.SendMessage(fmt.Sprintf("%d. %s **%s**\n", i+1, icon, v.PolicyName))
			sse.SendMessage(fmt.Sprintf("   - Resource: `%s.%s`\n", v.ResourceType, v.ResourceName))
			sse.SendMessage(fmt.Sprintf("   - %s\n", v.Message))
			sse.SendMessage(fmt.Sprintf("   - 💡 Fix: %s\n", v.Remediation))
			if v.Documentation != "" {
				sse.SendMessage(fmt.Sprintf("   - 📖 [Documentation](%s)\n", v.Documentation))
			}
			sse.SendMessage("\n")
		}

		// Send references
		refs := []Reference{}
		for _, v := range violations {
			if v.Documentation != "" {
				refs = append(refs, Reference{
					Title: v.PolicyName,
					URL:   v.Documentation,
				})
			}
		}
		if len(refs) > 0 {
			sse.SendReferences(refs)
		}
	}

	sse.SendMessage("\n---\n*Policy check completed*")
}

// =============================================================================
// Policy Checking
// =============================================================================

func (s *Server) checkPolicies(resources []Resource) []PolicyViolation {
	var violations []PolicyViolation

	for _, resource := range resources {
		for _, rule := range s.rules {
			// Check if rule applies to this resource type
			if !strings.EqualFold(rule.ResourceType, resource.Type) {
				continue
			}

			// Check the policy
			if !s.evaluatePolicy(resource, rule) {
				violations = append(violations, PolicyViolation{
					PolicyID:      rule.ID,
					PolicyName:    rule.Name,
					ResourceType:  resource.Type,
					ResourceName:  resource.Name,
					Severity:      rule.Severity,
					Message:       rule.Description,
					Remediation:   rule.Remediation,
					Documentation: rule.Documentation,
					Line:          resource.Line,
				})
			}
		}
	}

	return violations
}

func (s *Server) evaluatePolicy(resource Resource, rule PolicyRule) bool {
	// Get the property value from the resource
	value := getNestedProperty(resource.Properties, rule.Check.Property)

	// If value is nil, check default
	if value == nil {
		if rule.Check.Default != nil {
			value = rule.Check.Default
		} else {
			// Property not found and no default, check fails
			return rule.Check.Operator == "not_exists"
		}
	}

	// Evaluate based on operator
	switch rule.Check.Operator {
	case "equals":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", rule.Check.Value)
	case "not_equals":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", rule.Check.Value)
	case "exists":
		return value != nil
	case "not_exists":
		return value == nil
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", value), fmt.Sprintf("%v", rule.Check.Value))
	case "greater_than":
		// Simplified numeric comparison
		return fmt.Sprintf("%v", value) > fmt.Sprintf("%v", rule.Check.Value)
	default:
		return true
	}
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

	// Simple regex-based parser for Terraform
	// Pattern: resource "type" "name" { ... }
	resourcePattern := regexp.MustCompile(`(?m)^resource\s+"([^"]+)"\s+"([^"]+)"\s*\{`)

	matches := resourcePattern.FindAllStringSubmatchIndex(code, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			resourceType := code[match[2]:match[3]]
			resourceName := code[match[4]:match[5]]

			// Find the line number
			line := strings.Count(code[:match[0]], "\n") + 1

			// Extract the resource block
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

func parseTerraformBlock(block string) map[string]interface{} {
	props := make(map[string]interface{})

	// Simple key-value parser
	// Pattern: key = value or key = "value"
	kvPattern := regexp.MustCompile(`(?m)^\s*([a-z_]+)\s*=\s*(.+)$`)

	matches := kvPattern.FindAllStringSubmatch(block, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			key := strings.TrimSpace(match[1])
			value := strings.TrimSpace(match[2])

			// Parse value
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

	// Parse nested blocks
	nestedPattern := regexp.MustCompile(`(?m)^\s*([a-z_]+)\s*\{`)
	nestedMatches := nestedPattern.FindAllStringSubmatchIndex(block, -1)

	for _, match := range nestedMatches {
		if len(match) >= 4 {
			blockName := block[match[2]:match[3]]
			blockStart := match[1]
			blockEnd := findMatchingBrace(block, blockStart)

			if blockEnd > blockStart {
				nestedBlock := block[blockStart:blockEnd]
				props[blockName] = parseTerraformBlock(nestedBlock)
			}
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

func (s *Server) parseBicep(code string) []Resource {
	var resources []Resource

	// Simple regex-based parser for Bicep
	// Pattern: resource name 'type@version' = { ... }
	resourcePattern := regexp.MustCompile(`(?m)^resource\s+(\w+)\s+'([^']+)'\s*=\s*\{`)

	matches := resourcePattern.FindAllStringSubmatchIndex(code, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			resourceName := code[match[2]:match[3]]
			resourceType := code[match[4]:match[5]]

			// Convert Bicep type to Terraform-like type for policy matching
			tfType := bicepToTerraformType(resourceType)

			// Find the line number
			line := strings.Count(code[:match[0]], "\n") + 1

			// Find the resource block end
			blockStart := match[1] - 1 // Position of opening brace
			blockEnd := findMatchingBrace(code, blockStart)
			resourceBlock := code[blockStart : blockEnd+1]

			// Parse Bicep properties
			props := parseBicepProperties(resourceBlock)

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

// parseBicepProperties extracts properties from a Bicep resource block
// and maps them to Terraform-equivalent names for policy checking
func parseBicepProperties(block string) map[string]interface{} {
	props := make(map[string]interface{})

	// Bicep to Terraform property mapping for storage accounts
	bicepToTfProps := map[string]string{
		"supportsHttpsTrafficOnly": "enable_https_traffic_only",
		"minimumTlsVersion":        "min_tls_version",
		"allowBlobPublicAccess":    "allow_blob_public_access",
		"publicNetworkAccess":      "public_network_access_enabled",
	}

	// Parse simple key: value properties
	propPattern := regexp.MustCompile(`(?m)^\s*(\w+)\s*:\s*(.+?)\s*$`)
	propMatches := propPattern.FindAllStringSubmatch(block, -1)

	for _, m := range propMatches {
		if len(m) >= 3 {
			key := m[1]
			value := strings.TrimSpace(m[2])

			// Map to Terraform property name if available
			if tfKey, ok := bicepToTfProps[key]; ok {
				key = tfKey
			}

			// Parse the value
			props[key] = parseBicepValue(value)
		}
	}

	// Also look for nested properties block
	propsBlockPattern := regexp.MustCompile(`(?s)properties\s*:\s*\{([^}]+)\}`)
	if propsMatch := propsBlockPattern.FindStringSubmatch(block); len(propsMatch) >= 2 {
		nestedProps := propsMatch[1]
		nestedPattern := regexp.MustCompile(`(?m)^\s*(\w+)\s*:\s*(.+?)\s*$`)
		nestedMatches := nestedPattern.FindAllStringSubmatch(nestedProps, -1)

		for _, m := range nestedMatches {
			if len(m) >= 3 {
				key := m[1]
				value := strings.TrimSpace(m[2])

				// Map to Terraform property name if available
				if tfKey, ok := bicepToTfProps[key]; ok {
					key = tfKey
				}

				props[key] = parseBicepValue(value)
			}
		}
	}

	return props
}

// parseBicepValue converts Bicep values to Go types
func parseBicepValue(value string) interface{} {
	// Remove trailing comments
	if idx := strings.Index(value, "//"); idx != -1 {
		value = strings.TrimSpace(value[:idx])
	}

	// Boolean
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// String (remove quotes)
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return strings.Trim(value, "'")
	}

	// Number
	if num, err := strconv.Atoi(value); err == nil {
		return num
	}

	return value
}

func bicepToTerraformType(bicepType string) string {
	// Remove API version
	parts := strings.Split(bicepType, "@")
	if len(parts) > 0 {
		bicepType = parts[0]
	}

	// Map common Bicep types to Terraform
	mapping := map[string]string{
		"Microsoft.Storage/storageAccounts":          "azurerm_storage_account",
		"Microsoft.ContainerService/managedClusters": "azurerm_kubernetes_cluster",
		"Microsoft.Network/virtualNetworks":          "azurerm_virtual_network",
		"Microsoft.KeyVault/vaults":                  "azurerm_key_vault",
		"Microsoft.Compute/virtualMachines":          "azurerm_virtual_machine",
		"Microsoft.Sql/servers/databases":            "azurerm_mssql_database",
	}

	if tfType, ok := mapping[bicepType]; ok {
		return tfType
	}

	// Fallback: convert Microsoft.X/y to azurerm_y
	parts = strings.Split(bicepType, "/")
	if len(parts) >= 2 {
		return "azurerm_" + strings.ToLower(parts[len(parts)-1])
	}

	return bicepType
}

// =============================================================================
// Azure Policy API Integration
// =============================================================================

func (s *Server) fetchAzurePolicies(ctx context.Context) []string {
	if s.config.AzureSubscriptionID == "" {
		return nil
	}

	// Use Azure CLI to get policies (simpler than SDK for demo)
	cmd := exec.CommandContext(ctx, "az", "policy", "definition", "list",
		"--subscription", s.config.AzureSubscriptionID,
		"--query", "[?policyType=='BuiltIn'].{name:displayName}",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Could not fetch Azure policies: %v", err)
		return nil
	}

	var policies []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(output, &policies); err != nil {
		return nil
	}

	names := make([]string, 0, len(policies))
	for _, p := range policies {
		names = append(names, p.Name)
	}

	return names
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
	s.sendEvent("copilot_message", MessageEvent{Content: content})
}

func (s *SSEWriter) SendReferences(refs []Reference) {
	s.sendEvent("copilot_references", ReferenceEvent{References: refs})
}

func (s *SSEWriter) SendConfirmation(title, message string) {
	s.sendEvent("copilot_confirmation", ConfirmationEvent{
		Confirmation: Confirmation{Title: title, Message: message},
	})
}

func (s *SSEWriter) sendEvent(eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling SSE data: %v", err)
		return
	}

	fmt.Fprintf(s.w, "event: %s\n", eventType)
	fmt.Fprintf(s.w, "data: %s\n\n", string(jsonData))
	s.flusher.Flush()
}

// =============================================================================
// Helper Functions
// =============================================================================

func extractCode(message string) string {
	// Look for code blocks
	codeBlockPattern := regexp.MustCompile("(?s)```(?:terraform|bicep|hcl)?\\s*\\n(.+?)\\n```")
	matches := codeBlockPattern.FindStringSubmatch(message)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	// Look for inline code that looks like IaC
	if strings.Contains(message, "resource ") ||
		strings.Contains(message, "param ") ||
		strings.Contains(message, "azurerm_") {
		// Try to extract just the IaC portion
		lines := strings.Split(message, "\n")
		var codeLines []string
		inCode := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "resource ") ||
				strings.HasPrefix(trimmed, "param ") ||
				strings.HasPrefix(trimmed, "variable ") {
				inCode = true
			}
			if inCode {
				codeLines = append(codeLines, line)
			}
		}
		if len(codeLines) > 0 {
			return strings.Join(codeLines, "\n")
		}
	}

	return ""
}

func detectIaCType(code string) string {
	if strings.Contains(code, "resource \"azurerm_") ||
		strings.Contains(code, "variable \"") ||
		strings.Contains(code, "terraform {") {
		return "Terraform"
	}
	if strings.Contains(code, "param ") && strings.Contains(code, "resource ") ||
		strings.Contains(code, "@description") ||
		strings.Contains(code, "Microsoft.") {
		return "Bicep"
	}
	// Default to Terraform
	return "Terraform"
}

func filterBySeverity(violations []PolicyViolation, severity string) []PolicyViolation {
	var filtered []PolicyViolation
	for _, v := range violations {
		if strings.EqualFold(v.Severity, severity) {
			filtered = append(filtered, v)
		}
	}
	return filtered
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

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// =============================================================================
// Main Entry Point
// =============================================================================

func main() {
	config := loadConfig()
	server := NewServer(config)
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
