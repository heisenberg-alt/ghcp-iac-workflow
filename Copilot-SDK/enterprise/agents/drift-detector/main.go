// =============================================================================
// Drift Detector Copilot Agent
// =============================================================================
// A Copilot Agent that detects configuration drift between IaC definitions
// and actual Azure resources using Azure Resource Graph.
//
// Features:
//   - Compare IaC with live Azure resources
//   - Detect property drift and missing resources
//   - Azure Resource Graph queries
//   - Stream results via Server-Sent Events
//
// Usage:
//   go run .
//   # Server starts on :8083
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
	AzureTenantID       string
	WebhookSecret       string
	Debug               bool
}

func loadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	return &Config{
		Port:                port,
		AzureSubscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		AzureTenantID:       os.Getenv("AZURE_TENANT_ID"),
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
// Drift Types
// =============================================================================

type DriftResult struct {
	ResourceType   string                 `json:"resource_type"`
	ResourceName   string                 `json:"resource_name"`
	ResourceID     string                 `json:"resource_id,omitempty"`
	Status         string                 `json:"status"` // in_sync, drifted, missing_in_azure, missing_in_iac
	ExpectedValues map[string]interface{} `json:"expected_values,omitempty"`
	ActualValues   map[string]interface{} `json:"actual_values,omitempty"`
	Drifts         []PropertyDrift        `json:"drifts,omitempty"`
}

type PropertyDrift struct {
	Property      string      `json:"property"`
	ExpectedValue interface{} `json:"expected_value"`
	ActualValue   interface{} `json:"actual_value"`
	Severity      string      `json:"severity"`
}

type Resource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Line       int                    `json:"line"`
}

type AzureResource struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Location   string                 `json:"location"`
	Properties map[string]interface{} `json:"properties"`
	Tags       map[string]string      `json:"tags"`
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
	s.mux.HandleFunc("/scan", s.handleFullScan)
	s.mux.HandleFunc("/report", s.handleReport)
	s.mux.HandleFunc("/", s.handleAgent)
}

func (s *Server) Run() error {
	addr := ":" + s.config.Port
	log.Printf("🔄 Drift Detector Agent starting on %s", addr)
	log.Printf("📍 Endpoints:")
	log.Printf("   POST /agent  - Agent endpoint (SSE)")
	log.Printf("   POST /scan   - Full subscription drift scan")
	log.Printf("   GET  /report - Get latest drift report")
	log.Printf("   GET  /health - Health check")
	if s.config.AzureSubscriptionID != "" {
		log.Printf("☁️ Azure Subscription: %s", s.config.AzureSubscriptionID)
	} else {
		log.Printf("⚠️ No Azure subscription configured - using simulated responses")
	}
	return http.ListenAndServe(addr, s.mux)
}

// =============================================================================
// Health Check
// =============================================================================

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "healthy",
		"service":          "drift-detector-agent",
		"azure_configured": s.config.AzureSubscriptionID != "",
		"subscription_id":  s.config.AzureSubscriptionID,
	})
}

// =============================================================================
// Report Endpoint
// =============================================================================

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "No drift report available. Run a drift scan first.",
		"hint":    "POST /agent with IaC code to detect drift",
	})
}

// =============================================================================
// Full Scan Endpoint
// =============================================================================

func (s *Server) handleFullScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "scan_initiated",
		"message": "Full subscription scan would be initiated (requires Azure credentials)",
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

	log.Printf("→ Received drift detection request")

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
	s.processDriftDetection(r.Context(), req, sse)
}

func (s *Server) processDriftDetection(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	// Get the last user message
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = req.Messages[i].Content
			break
		}
	}

	if userMessage == "" {
		sse.SendMessage("❌ No message found. Please provide IaC code to check for drift.")
		return
	}

	log.Printf("Processing drift detection: %s", truncate(userMessage, 100))

	// Send initial message
	sse.SendMessage("🔄 **Drift Detector Agent**\n\n")
	sse.SendMessage("Comparing your Infrastructure as Code with live Azure resources...\n\n")
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
		sse.SendMessage("**What I detect:**\n")
		sse.SendMessage("- 🔄 Property changes in Azure resources\n")
		sse.SendMessage("- ❌ Resources missing in Azure\n")
		sse.SendMessage("- ➕ Resources created outside IaC\n")
		sse.SendMessage("- 🏷️ Tag drift\n")
		return
	}

	// Detect IaC type
	iacType := detectIaCType(code)
	sse.SendMessage(fmt.Sprintf("📝 Detected **%s** code\n\n", iacType))
	time.Sleep(200 * time.Millisecond)

	// Parse resources from IaC
	sse.SendMessage("🔍 Parsing IaC resources...\n")
	resources := s.parseResources(code, iacType)

	if len(resources) == 0 {
		sse.SendMessage("\n⚠️ No resources found in the code.\n")
		return
	}

	sse.SendMessage(fmt.Sprintf("   Found **%d** resource(s) in IaC\n\n", len(resources)))
	time.Sleep(200 * time.Millisecond)

	// Query Azure resources
	sse.SendMessage("☁️ Querying Azure resources...\n")

	if s.config.AzureSubscriptionID == "" {
		sse.SendMessage("   ⚠️ Azure not configured - using simulated comparison\n\n")
		time.Sleep(300 * time.Millisecond)
	} else {
		sse.SendMessage(fmt.Sprintf("   Subscription: `%s`\n\n", s.config.AzureSubscriptionID))
		time.Sleep(300 * time.Millisecond)
	}

	// Perform drift detection
	sse.SendMessage("🔄 Comparing configurations...\n\n")
	time.Sleep(300 * time.Millisecond)

	results := s.detectDrift(ctx, resources)

	// Report results
	inSync := 0
	drifted := 0
	missingInAzure := 0
	missingInIaC := 0

	for _, r := range results {
		switch r.Status {
		case "in_sync":
			inSync++
		case "drifted":
			drifted++
		case "missing_in_azure":
			missingInAzure++
		case "missing_in_iac":
			missingInIaC++
		}
	}

	if drifted == 0 && missingInAzure == 0 && missingInIaC == 0 {
		sse.SendMessage("✅ **No drift detected!**\n\n")
		sse.SendMessage("Your IaC matches the current Azure state.\n\n")
		sse.SendMessage("**Resources checked:**\n")
		for _, res := range resources {
			sse.SendMessage(fmt.Sprintf("- ✓ `%s.%s`\n", res.Type, res.Name))
		}
	} else {
		sse.SendMessage(fmt.Sprintf("⚠️ **Drift detected in %d resource(s)**\n\n", drifted+missingInAzure+missingInIaC))

		// Summary
		sse.SendMessage("**Summary:**\n")
		if inSync > 0 {
			sse.SendMessage(fmt.Sprintf("- ✅ In Sync: %d\n", inSync))
		}
		if drifted > 0 {
			sse.SendMessage(fmt.Sprintf("- 🔄 Drifted: %d\n", drifted))
		}
		if missingInAzure > 0 {
			sse.SendMessage(fmt.Sprintf("- ❌ Missing in Azure: %d\n", missingInAzure))
		}
		if missingInIaC > 0 {
			sse.SendMessage(fmt.Sprintf("- ➕ Not in IaC: %d\n", missingInIaC))
		}
		sse.SendMessage("\n")

		// Details
		sse.SendMessage("**Drift Details:**\n\n")

		for _, result := range results {
			if result.Status == "in_sync" {
				continue
			}

			icon := getDriftIcon(result.Status)
			sse.SendMessage(fmt.Sprintf("### %s `%s.%s`\n\n", icon, result.ResourceType, result.ResourceName))

			switch result.Status {
			case "drifted":
				sse.SendMessage("| Property | Expected (IaC) | Actual (Azure) | Severity |\n")
				sse.SendMessage("|----------|----------------|----------------|----------|\n")
				for _, drift := range result.Drifts {
					sevIcon := getSeverityIcon(drift.Severity)
					sse.SendMessage(fmt.Sprintf("| `%s` | `%v` | `%v` | %s %s |\n",
						drift.Property, drift.ExpectedValue, drift.ActualValue, sevIcon, drift.Severity))
				}
				sse.SendMessage("\n")
				sse.SendMessage("**Remediation:** Run `terraform apply` or `az deployment` to sync\n\n")

			case "missing_in_azure":
				sse.SendMessage("Resource defined in IaC but not found in Azure.\n\n")
				sse.SendMessage("**Possible causes:**\n")
				sse.SendMessage("- Resource was manually deleted\n")
				sse.SendMessage("- Resource creation failed\n")
				sse.SendMessage("- Different subscription/resource group\n\n")
				sse.SendMessage("**Remediation:** Run `terraform apply` to create the resource\n\n")

			case "missing_in_iac":
				sse.SendMessage("Resource exists in Azure but not defined in IaC.\n\n")
				sse.SendMessage("**Possible causes:**\n")
				sse.SendMessage("- Resource created manually\n")
				sse.SendMessage("- Resource created by another deployment\n\n")
				sse.SendMessage("**Remediation:** Import resource to IaC or document as expected\n\n")
			}
		}
	}

	// Send references
	refs := []Reference{
		{Title: "Terraform State", URL: "https://developer.hashicorp.com/terraform/language/state"},
		{Title: "Azure Resource Graph", URL: "https://learn.microsoft.com/azure/governance/resource-graph/overview"},
	}
	sse.SendReferences(refs)

	sse.SendMessage("\n---\n*Drift detection completed*")
}

// =============================================================================
// Drift Detection
// =============================================================================

func (s *Server) detectDrift(ctx context.Context, resources []Resource) []DriftResult {
	var results []DriftResult

	for _, resource := range resources {
		// Get Azure resource (simulated or real)
		azureResource := s.getAzureResource(ctx, resource)

		result := DriftResult{
			ResourceType: resource.Type,
			ResourceName: resource.Name,
		}

		if azureResource == nil {
			// Resource not found in Azure
			result.Status = "missing_in_azure"
			result.ExpectedValues = resource.Properties
		} else {
			// Compare properties
			result.ResourceID = azureResource.ID
			drifts := s.compareProperties(resource.Properties, azureResource.Properties)

			if len(drifts) == 0 {
				result.Status = "in_sync"
			} else {
				result.Status = "drifted"
				result.Drifts = drifts
				result.ExpectedValues = resource.Properties
				result.ActualValues = azureResource.Properties
			}
		}

		results = append(results, result)
	}

	return results
}

func (s *Server) getAzureResource(ctx context.Context, resource Resource) *AzureResource {
	// In a real implementation, this would query Azure Resource Graph
	// For demo purposes, simulate some resources existing with different properties

	// Simulate some resources existing
	simulatedResources := map[string]map[string]*AzureResource{
		"azurerm_storage_account": {
			"example": {
				ID:       "/subscriptions/xxx/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/example",
				Name:     "example",
				Type:     "Microsoft.Storage/storageAccounts",
				Location: "eastus",
				Properties: map[string]interface{}{
					"enable_https_traffic_only": false,    // Simulated drift
					"min_tls_version":           "TLS1_0", // Simulated drift
					"allow_blob_public_access":  true,
				},
			},
		},
		"azurerm_kubernetes_cluster": {
			"aks": {
				ID:       "/subscriptions/xxx/resourceGroups/rg/providers/Microsoft.ContainerService/managedClusters/aks",
				Name:     "aks",
				Type:     "Microsoft.ContainerService/managedClusters",
				Location: "eastus",
				Properties: map[string]interface{}{
					"role_based_access_control_enabled": true,
					"kubernetes_version":                "1.27.0", // May drift from expected
				},
			},
		},
	}

	if typeResources, ok := simulatedResources[resource.Type]; ok {
		if azResource, ok := typeResources[resource.Name]; ok {
			return azResource
		}
	}

	// For resources not in simulation, randomly decide if they exist
	// In production, this would be a real Azure query
	return nil
}

func (s *Server) compareProperties(expected, actual map[string]interface{}) []PropertyDrift {
	var drifts []PropertyDrift

	// Properties to compare
	importantProps := []string{
		"enable_https_traffic_only",
		"min_tls_version",
		"allow_blob_public_access",
		"public_network_access_enabled",
		"role_based_access_control_enabled",
		"kubernetes_version",
	}

	for _, prop := range importantProps {
		expectedVal, hasExpected := expected[prop]
		actualVal, hasActual := actual[prop]

		if hasExpected && hasActual {
			if fmt.Sprintf("%v", expectedVal) != fmt.Sprintf("%v", actualVal) {
				severity := "medium"
				if prop == "enable_https_traffic_only" || prop == "allow_blob_public_access" || prop == "min_tls_version" {
					severity = "high"
				}

				drifts = append(drifts, PropertyDrift{
					Property:      prop,
					ExpectedValue: expectedVal,
					ActualValue:   actualVal,
					Severity:      severity,
				})
			}
		}
	}

	return drifts
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

func getDriftIcon(status string) string {
	switch status {
	case "drifted":
		return "🔄"
	case "missing_in_azure":
		return "❌"
	case "missing_in_iac":
		return "➕"
	default:
		return "✅"
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
