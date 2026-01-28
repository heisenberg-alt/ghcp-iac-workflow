// =============================================================================
// Module Registry Copilot Agent
// =============================================================================
// A Copilot Agent that manages approved IaC modules, validates module usage,
// and recommends modules from the organization's registry.
//
// Features:
//   - Module catalog management
//   - Version validation
//   - Module recommendations
//   - Source verification
//   - Stream results via Server-Sent Events
//
// Usage:
//   go run .
//   # Server starts on :8086
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
	Port           string
	RegistryURL    string
	AllowedSources []string
	WebhookSecret  string
	Debug          bool
}

func loadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	allowedSources := os.Getenv("ALLOWED_MODULE_SOURCES")
	if allowedSources == "" {
		allowedSources = "registry.terraform.io,github.com/yourorg"
	}

	return &Config{
		Port:           port,
		RegistryURL:    os.Getenv("MODULE_REGISTRY_URL"),
		AllowedSources: strings.Split(allowedSources, ","),
		WebhookSecret:  os.Getenv("GITHUB_WEBHOOK_SECRET"),
		Debug:          os.Getenv("DEBUG") != "",
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
// Module Types
// =============================================================================

type Module struct {
	Name          string         `json:"name"`
	Source        string         `json:"source"`
	Version       string         `json:"version"`
	Description   string         `json:"description"`
	Provider      string         `json:"provider"`
	Category      string         `json:"category"`
	Tags          []string       `json:"tags"`
	Inputs        []ModuleInput  `json:"inputs"`
	Outputs       []ModuleOutput `json:"outputs"`
	Examples      []string       `json:"examples"`
	Approved      bool           `json:"approved"`
	MinVersion    string         `json:"min_version"`
	MaxVersion    string         `json:"max_version,omitempty"`
	Deprecated    bool           `json:"deprecated"`
	ReplacedBy    string         `json:"replaced_by,omitempty"`
	Documentation string         `json:"documentation"`
}

type ModuleInput struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

type ModuleOutput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ModuleUsage struct {
	Name    string `json:"name"`
	Source  string `json:"source"`
	Version string `json:"version"`
	Line    int    `json:"line"`
}

type ModuleFinding struct {
	Module      string  `json:"module"`
	Source      string  `json:"source"`
	Version     string  `json:"version"`
	Status      string  `json:"status"` // approved, not_approved, deprecated, version_mismatch, unknown_source
	Severity    string  `json:"severity"`
	Message     string  `json:"message"`
	Remediation string  `json:"remediation"`
	Recommended *Module `json:"recommended,omitempty"`
}

// =============================================================================
// Server
// =============================================================================

type Server struct {
	config  *Config
	mux     *http.ServeMux
	catalog []Module
}

func NewServer(config *Config) *Server {
	s := &Server{
		config: config,
		mux:    http.NewServeMux(),
	}
	s.loadCatalog()
	s.setupRoutes()
	return s
}

func (s *Server) loadCatalog() {
	// Load module catalog from JSON
	data, err := os.ReadFile("data/modules.json")
	if err != nil {
		log.Printf("Warning: Could not load module catalog: %v, using defaults", err)
		s.catalog = s.getDefaultCatalog()
		return
	}

	var config struct {
		Modules []Module `json:"modules"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Warning: Could not parse module catalog: %v, using defaults", err)
		s.catalog = s.getDefaultCatalog()
		return
	}

	s.catalog = config.Modules
	log.Printf("Loaded %d modules in catalog", len(s.catalog))
}

func (s *Server) getDefaultCatalog() []Module {
	return []Module{
		{
			Name:        "storage-account",
			Source:      "registry.terraform.io/Azure/storage/azurerm",
			Version:     "3.0.0",
			Description: "Azure Storage Account with security best practices",
			Provider:    "azurerm",
			Category:    "storage",
			Tags:        []string{"storage", "azure", "approved"},
			Approved:    true,
			MinVersion:  "2.0.0",
			Inputs: []ModuleInput{
				{Name: "name", Type: "string", Description: "Storage account name", Required: true},
				{Name: "resource_group_name", Type: "string", Description: "Resource group name", Required: true},
				{Name: "location", Type: "string", Description: "Azure region", Required: true},
			},
			Outputs: []ModuleOutput{
				{Name: "id", Description: "Storage account ID"},
				{Name: "primary_blob_endpoint", Description: "Primary blob endpoint"},
			},
			Documentation: "https://registry.terraform.io/modules/Azure/storage/azurerm",
		},
		{
			Name:        "aks-cluster",
			Source:      "registry.terraform.io/Azure/aks/azurerm",
			Version:     "7.0.0",
			Description: "Azure Kubernetes Service cluster with best practices",
			Provider:    "azurerm",
			Category:    "containers",
			Tags:        []string{"kubernetes", "aks", "azure", "approved"},
			Approved:    true,
			MinVersion:  "6.0.0",
			Inputs: []ModuleInput{
				{Name: "cluster_name", Type: "string", Description: "AKS cluster name", Required: true},
				{Name: "resource_group_name", Type: "string", Description: "Resource group name", Required: true},
				{Name: "location", Type: "string", Description: "Azure region", Required: true},
				{Name: "kubernetes_version", Type: "string", Description: "Kubernetes version", Required: false, Default: "1.28"},
			},
			Documentation: "https://registry.terraform.io/modules/Azure/aks/azurerm",
		},
		{
			Name:          "key-vault",
			Source:        "registry.terraform.io/Azure/keyvault/azurerm",
			Version:       "2.1.0",
			Description:   "Azure Key Vault with security configurations",
			Provider:      "azurerm",
			Category:      "security",
			Tags:          []string{"keyvault", "secrets", "azure", "approved"},
			Approved:      true,
			MinVersion:    "2.0.0",
			Documentation: "https://registry.terraform.io/modules/Azure/keyvault/azurerm",
		},
		{
			Name:          "vnet",
			Source:        "registry.terraform.io/Azure/network/azurerm",
			Version:       "5.0.0",
			Description:   "Azure Virtual Network with subnets",
			Provider:      "azurerm",
			Category:      "networking",
			Tags:          []string{"network", "vnet", "azure", "approved"},
			Approved:      true,
			MinVersion:    "4.0.0",
			Documentation: "https://registry.terraform.io/modules/Azure/network/azurerm",
		},
		{
			Name:          "legacy-storage",
			Source:        "github.com/old-org/terraform-azure-storage",
			Version:       "1.0.0",
			Description:   "Legacy storage module - deprecated",
			Provider:      "azurerm",
			Category:      "storage",
			Approved:      false,
			Deprecated:    true,
			ReplacedBy:    "storage-account",
			Documentation: "",
		},
	}
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/agent", s.handleAgent)
	s.mux.HandleFunc("/modules", s.handleModules)
	s.mux.HandleFunc("/modules/search", s.handleSearch)
	s.mux.HandleFunc("/modules/recommend", s.handleRecommend)
	s.mux.HandleFunc("/", s.handleAgent)
}

func (s *Server) Run() error {
	addr := ":" + s.config.Port
	log.Printf("📦 Module Registry Agent starting on %s", addr)
	log.Printf("📍 Endpoints:")
	log.Printf("   POST /agent           - Agent endpoint (SSE)")
	log.Printf("   GET  /modules         - List all approved modules")
	log.Printf("   GET  /modules/search  - Search modules")
	log.Printf("   POST /modules/recommend - Get module recommendations")
	log.Printf("   GET  /health          - Health check")
	log.Printf("📦 Catalog: %d modules", len(s.catalog))
	return http.ListenAndServe(addr, s.mux)
}

// =============================================================================
// Health Check
// =============================================================================

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	approved := 0
	for _, m := range s.catalog {
		if m.Approved {
			approved++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "healthy",
		"service":          "module-registry-agent",
		"total_modules":    len(s.catalog),
		"approved_modules": approved,
	})
}

// =============================================================================
// Modules Endpoint
// =============================================================================

func (s *Server) handleModules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	approved := []Module{}
	for _, m := range s.catalog {
		if m.Approved && !m.Deprecated {
			approved = append(approved, m)
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"modules": approved,
		"count":   len(approved),
	})
}

// =============================================================================
// Search Endpoint
// =============================================================================

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")

	var results []Module
	for _, m := range s.catalog {
		if !m.Approved || m.Deprecated {
			continue
		}

		// Match query
		if query != "" {
			queryLower := strings.ToLower(query)
			if !strings.Contains(strings.ToLower(m.Name), queryLower) &&
				!strings.Contains(strings.ToLower(m.Description), queryLower) {
				continue
			}
		}

		// Match category
		if category != "" && !strings.EqualFold(m.Category, category) {
			continue
		}

		results = append(results, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

// =============================================================================
// Recommend Endpoint
// =============================================================================

func (s *Server) handleRecommend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ResourceType string `json:"resource_type"`
		UseCase      string `json:"use_case"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	recommendations := s.getRecommendations(req.ResourceType, req.UseCase)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recommendations": recommendations,
	})
}

func (s *Server) getRecommendations(resourceType, useCase string) []Module {
	var recommendations []Module

	// Map resource types to categories
	categoryMap := map[string]string{
		"storage_account":            "storage",
		"azurerm_storage_account":    "storage",
		"kubernetes_cluster":         "containers",
		"azurerm_kubernetes_cluster": "containers",
		"key_vault":                  "security",
		"azurerm_key_vault":          "security",
		"virtual_network":            "networking",
		"azurerm_virtual_network":    "networking",
	}

	category := categoryMap[strings.ToLower(resourceType)]

	for _, m := range s.catalog {
		if !m.Approved || m.Deprecated {
			continue
		}

		if category != "" && strings.EqualFold(m.Category, category) {
			recommendations = append(recommendations, m)
		}
	}

	return recommendations
}

// =============================================================================
// Agent Handler
// =============================================================================

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("→ Received module registry request")

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
	s.processModuleCheck(r.Context(), req, sse)
}

func (s *Server) processModuleCheck(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	// Get the last user message
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = req.Messages[i].Content
			break
		}
	}

	if userMessage == "" {
		sse.SendMessage("❌ No message found. Please provide IaC code to check modules.")
		return
	}

	log.Printf("Processing module check: %s", truncate(userMessage, 100))

	// Send initial message
	sse.SendMessage("📦 **Module Registry Agent**\n\n")
	sse.SendMessage("Validating module usage against the approved catalog...\n\n")
	time.Sleep(300 * time.Millisecond)

	// Check if asking for recommendations
	if strings.Contains(strings.ToLower(userMessage), "recommend") ||
		strings.Contains(strings.ToLower(userMessage), "suggest") ||
		strings.Contains(strings.ToLower(userMessage), "find module") {
		s.handleRecommendation(userMessage, sse)
		return
	}

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
		sse.SendMessage("- Paste Terraform code with module blocks\n")
		sse.SendMessage("- Ask for module recommendations\n\n")
		sse.SendMessage("**Available Commands:**\n")
		sse.SendMessage("- `Check modules in this Terraform`\n")
		sse.SendMessage("- `Recommend a storage module`\n")
		sse.SendMessage("- `Find approved AKS modules`\n\n")
		sse.SendMessage("**Approved Module Categories:**\n")
		categories := s.getCategories()
		for _, cat := range categories {
			sse.SendMessage(fmt.Sprintf("- 📦 %s\n", cat))
		}
		return
	}

	// Parse module usage
	sse.SendMessage("🔍 Scanning for module usage...\n")
	modules := s.parseModules(code)

	if len(modules) == 0 {
		sse.SendMessage("\n📦 No module blocks found in the code.\n\n")
		sse.SendMessage("**Tip:** Consider using approved modules for common patterns:\n")
		for _, m := range s.catalog {
			if m.Approved && !m.Deprecated {
				sse.SendMessage(fmt.Sprintf("- `%s` - %s\n", m.Name, m.Description))
			}
		}
		return
	}

	sse.SendMessage(fmt.Sprintf("   Found **%d** module(s)\n\n", len(modules)))
	time.Sleep(200 * time.Millisecond)

	// Validate modules
	sse.SendMessage("✅ Validating against approved catalog...\n\n")
	findings := s.validateModules(modules)

	// Report results
	approved := 0
	issues := 0
	for _, f := range findings {
		if f.Status == "approved" {
			approved++
		} else {
			issues++
		}
	}

	if issues == 0 {
		sse.SendMessage("✅ **All modules are approved!**\n\n")
		sse.SendMessage("**Validated Modules:**\n")
		for _, f := range findings {
			sse.SendMessage(fmt.Sprintf("- ✓ `%s` (v%s)\n", f.Module, f.Version))
		}
	} else {
		sse.SendMessage(fmt.Sprintf("⚠️ **Found %d module issue(s)**\n\n", issues))

		// Summary
		sse.SendMessage("**Summary:**\n")
		sse.SendMessage(fmt.Sprintf("- ✅ Approved: %d\n", approved))
		sse.SendMessage(fmt.Sprintf("- ⚠️ Issues: %d\n", issues))
		sse.SendMessage("\n")

		// Details
		sse.SendMessage("**Module Issues:**\n\n")

		for _, f := range findings {
			if f.Status == "approved" {
				continue
			}

			icon := getStatusIcon(f.Status)
			sse.SendMessage(fmt.Sprintf("### %s `%s`\n\n", icon, f.Module))
			sse.SendMessage(fmt.Sprintf("- Source: `%s`\n", f.Source))
			sse.SendMessage(fmt.Sprintf("- Version: `%s`\n", f.Version))
			sse.SendMessage(fmt.Sprintf("- Status: **%s**\n", f.Status))
			sse.SendMessage(fmt.Sprintf("- %s\n", f.Message))
			sse.SendMessage(fmt.Sprintf("- 💡 %s\n", f.Remediation))

			if f.Recommended != nil {
				sse.SendMessage(fmt.Sprintf("\n**Recommended Replacement:**\n"))
				sse.SendMessage(fmt.Sprintf("```hcl\nmodule \"%s\" {\n", f.Recommended.Name))
				sse.SendMessage(fmt.Sprintf("  source  = \"%s\"\n", f.Recommended.Source))
				sse.SendMessage(fmt.Sprintf("  version = \"%s\"\n", f.Recommended.Version))
				sse.SendMessage("}\n```\n")
			}
			sse.SendMessage("\n")
		}
	}

	// Send references
	refs := []Reference{
		{Title: "Terraform Module Registry", URL: "https://registry.terraform.io/"},
	}
	for _, m := range s.catalog {
		if m.Approved && m.Documentation != "" {
			refs = append(refs, Reference{
				Title: m.Name,
				URL:   m.Documentation,
			})
		}
	}
	sse.SendReferences(refs)

	sse.SendMessage("\n---\n*Module validation completed*")
}

func (s *Server) handleRecommendation(message string, sse *SSEWriter) {
	// Extract what type of module they want
	categories := []string{"storage", "container", "kubernetes", "aks", "network", "security", "keyvault"}

	var matchedCategory string
	messageLower := strings.ToLower(message)
	for _, cat := range categories {
		if strings.Contains(messageLower, cat) {
			matchedCategory = cat
			break
		}
	}

	sse.SendMessage("**Recommended Modules:**\n\n")

	found := false
	for _, m := range s.catalog {
		if !m.Approved || m.Deprecated {
			continue
		}

		if matchedCategory != "" {
			if !strings.Contains(strings.ToLower(m.Category), matchedCategory) &&
				!containsAny(m.Tags, matchedCategory) {
				continue
			}
		}

		found = true
		sse.SendMessage(fmt.Sprintf("### 📦 %s\n\n", m.Name))
		sse.SendMessage(fmt.Sprintf("**Description:** %s\n\n", m.Description))
		sse.SendMessage(fmt.Sprintf("**Source:** `%s`\n", m.Source))
		sse.SendMessage(fmt.Sprintf("**Latest Version:** `%s`\n", m.Version))
		sse.SendMessage(fmt.Sprintf("**Category:** %s\n\n", m.Category))

		sse.SendMessage("**Example Usage:**\n```hcl\n")
		sse.SendMessage(fmt.Sprintf("module \"%s\" {\n", m.Name))
		sse.SendMessage(fmt.Sprintf("  source  = \"%s\"\n", m.Source))
		sse.SendMessage(fmt.Sprintf("  version = \"%s\"\n\n", m.Version))
		for _, input := range m.Inputs {
			if input.Required {
				sse.SendMessage(fmt.Sprintf("  %s = \"<value>\"  # %s\n", input.Name, input.Description))
			}
		}
		sse.SendMessage("}\n```\n\n")
	}

	if !found {
		sse.SendMessage("No matching modules found. Available categories:\n")
		for _, cat := range s.getCategories() {
			sse.SendMessage(fmt.Sprintf("- %s\n", cat))
		}
	}
}

// =============================================================================
// Module Validation
// =============================================================================

func (s *Server) parseModules(code string) []ModuleUsage {
	var modules []ModuleUsage

	// Pattern: module "name" { source = "..." version = "..." }
	modulePattern := regexp.MustCompile(`(?m)module\s+"([^"]+)"\s*\{`)
	matches := modulePattern.FindAllStringSubmatchIndex(code, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			moduleName := code[match[2]:match[3]]
			line := strings.Count(code[:match[0]], "\n") + 1

			// Find the module block
			blockStart := match[1]
			blockEnd := findMatchingBrace(code, blockStart)
			block := code[blockStart:blockEnd]

			// Extract source
			sourcePattern := regexp.MustCompile(`source\s*=\s*"([^"]+)"`)
			sourceMatch := sourcePattern.FindStringSubmatch(block)
			source := ""
			if len(sourceMatch) > 1 {
				source = sourceMatch[1]
			}

			// Extract version
			versionPattern := regexp.MustCompile(`version\s*=\s*"([^"]+)"`)
			versionMatch := versionPattern.FindStringSubmatch(block)
			version := "unspecified"
			if len(versionMatch) > 1 {
				version = versionMatch[1]
			}

			modules = append(modules, ModuleUsage{
				Name:    moduleName,
				Source:  source,
				Version: version,
				Line:    line,
			})
		}
	}

	return modules
}

func (s *Server) validateModules(modules []ModuleUsage) []ModuleFinding {
	var findings []ModuleFinding

	for _, usage := range modules {
		finding := ModuleFinding{
			Module:  usage.Name,
			Source:  usage.Source,
			Version: usage.Version,
		}

		// Check if source is from allowed sources
		sourceAllowed := false
		for _, allowed := range s.config.AllowedSources {
			if strings.Contains(usage.Source, allowed) {
				sourceAllowed = true
				break
			}
		}

		if !sourceAllowed {
			finding.Status = "unknown_source"
			finding.Severity = "high"
			finding.Message = "Module source is not from an approved registry"
			finding.Remediation = "Use modules from approved sources: " + strings.Join(s.config.AllowedSources, ", ")
			finding.Recommended = s.findRecommendation(usage.Name)
			findings = append(findings, finding)
			continue
		}

		// Check against catalog
		catalogModule := s.findInCatalog(usage.Source)

		if catalogModule == nil {
			finding.Status = "not_approved"
			finding.Severity = "medium"
			finding.Message = "Module is not in the approved catalog"
			finding.Remediation = "Request module approval or use an approved alternative"
			finding.Recommended = s.findRecommendation(usage.Name)
		} else if catalogModule.Deprecated {
			finding.Status = "deprecated"
			finding.Severity = "high"
			finding.Message = "Module is deprecated"
			finding.Remediation = fmt.Sprintf("Migrate to: %s", catalogModule.ReplacedBy)
			finding.Recommended = s.findByName(catalogModule.ReplacedBy)
		} else if !catalogModule.Approved {
			finding.Status = "not_approved"
			finding.Severity = "medium"
			finding.Message = "Module is not approved for production use"
			finding.Remediation = "Request approval or use an approved alternative"
		} else if usage.Version != "unspecified" && catalogModule.MinVersion != "" {
			if compareVersions(usage.Version, catalogModule.MinVersion) < 0 {
				finding.Status = "version_mismatch"
				finding.Severity = "medium"
				finding.Message = fmt.Sprintf("Version %s is below minimum required version %s", usage.Version, catalogModule.MinVersion)
				finding.Remediation = fmt.Sprintf("Upgrade to version >= %s", catalogModule.MinVersion)
			} else {
				finding.Status = "approved"
				finding.Severity = "none"
				finding.Message = "Module is approved"
			}
		} else {
			finding.Status = "approved"
			finding.Severity = "none"
			finding.Message = "Module is approved"
		}

		findings = append(findings, finding)
	}

	return findings
}

func (s *Server) findInCatalog(source string) *Module {
	for _, m := range s.catalog {
		if strings.Contains(source, m.Source) || strings.Contains(m.Source, source) {
			return &m
		}
	}
	return nil
}

func (s *Server) findByName(name string) *Module {
	for _, m := range s.catalog {
		if strings.EqualFold(m.Name, name) {
			return &m
		}
	}
	return nil
}

func (s *Server) findRecommendation(name string) *Module {
	// Try to find a similar approved module
	nameLower := strings.ToLower(name)

	for _, m := range s.catalog {
		if !m.Approved || m.Deprecated {
			continue
		}

		// Check if names are similar
		if strings.Contains(strings.ToLower(m.Name), nameLower) ||
			strings.Contains(nameLower, strings.ToLower(m.Name)) {
			return &m
		}

		// Check tags
		for _, tag := range m.Tags {
			if strings.Contains(nameLower, tag) {
				return &m
			}
		}
	}

	return nil
}

func (s *Server) getCategories() []string {
	catMap := make(map[string]bool)
	for _, m := range s.catalog {
		if m.Approved && !m.Deprecated && m.Category != "" {
			catMap[m.Category] = true
		}
	}

	cats := make([]string, 0, len(catMap))
	for cat := range catMap {
		cats = append(cats, cat)
	}
	return cats
}

// =============================================================================
// Helpers
// =============================================================================

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

func extractCode(message string) string {
	codeBlockPattern := regexp.MustCompile("```(?:terraform|bicep|hcl)?\\s*\\n([\\s\\S]*?)\\n```")
	if matches := codeBlockPattern.FindStringSubmatch(message); len(matches) > 1 {
		return matches[1]
	}
	if strings.Contains(message, "module ") {
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

func getStatusIcon(status string) string {
	switch status {
	case "approved":
		return "✅"
	case "not_approved":
		return "⚠️"
	case "deprecated":
		return "🚫"
	case "version_mismatch":
		return "📌"
	case "unknown_source":
		return "❓"
	default:
		return "❔"
	}
}

func compareVersions(v1, v2 string) int {
	// Simple version comparison
	return strings.Compare(v1, v2)
}

func containsAny(slice []string, substr string) bool {
	for _, s := range slice {
		if strings.Contains(strings.ToLower(s), substr) {
			return true
		}
	}
	return false
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
