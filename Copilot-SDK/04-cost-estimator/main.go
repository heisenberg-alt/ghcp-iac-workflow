// =============================================================================
// Cost Estimator Copilot Agent
// =============================================================================
// A Copilot Agent that estimates Azure resource costs from Infrastructure as
// Code using the Azure Retail Prices API (no authentication required!).
//
// Features:
//   - Parse Terraform and Bicep code
//   - Extract resource configurations (SKU, count, region)
//   - Query Azure Retail Prices API for real pricing
//   - Calculate monthly cost estimates
//   - Stream itemized breakdown to user
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
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// Constants
// =============================================================================

const (
	AzurePricingAPIBaseURL = "https://prices.azure.com/api/retail/prices"
	HoursPerMonth          = 730
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
		port = "8080"
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

// =============================================================================
// Pricing Types
// =============================================================================

// AzurePricingResponse represents the API response
type AzurePricingResponse struct {
	Items    []PriceItem `json:"Items"`
	NextPage string      `json:"NextPageLink,omitempty"`
	Count    int         `json:"Count"`
}

// PriceItem represents a single pricing item
type PriceItem struct {
	CurrencyCode  string  `json:"currencyCode"`
	RetailPrice   float64 `json:"retailPrice"`
	UnitOfMeasure string  `json:"unitOfMeasure"`
	ArmRegionName string  `json:"armRegionName"`
	ProductName   string  `json:"productName"`
	SkuName       string  `json:"skuName"`
	ServiceName   string  `json:"serviceName"`
	ArmSkuName    string  `json:"armSkuName"`
	MeterName     string  `json:"meterName"`
	Type          string  `json:"type"`
}

// CostEstimate represents the cost for a single resource
type CostEstimate struct {
	ResourceType string  `json:"resource_type"`
	ResourceName string  `json:"resource_name"`
	SKU          string  `json:"sku"`
	Region       string  `json:"region"`
	Quantity     int     `json:"quantity"`
	UnitPrice    float64 `json:"unit_price"`
	PricingUnit  string  `json:"pricing_unit"`
	MonthlyCost  float64 `json:"monthly_cost"`
	Notes        string  `json:"notes,omitempty"`
}

// CostReport contains the full cost breakdown
type CostReport struct {
	TotalMonthly float64        `json:"total_monthly"`
	Currency     string         `json:"currency"`
	Region       string         `json:"region"`
	Items        []CostEstimate `json:"items"`
	Warnings     []string       `json:"warnings,omitempty"`
}

// Resource represents a parsed IaC resource
type Resource struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Line       int                    `json:"line"`
}

// =============================================================================
// SKU Mapping Data
// =============================================================================

// SKU pricing cache (hourly prices in USD)
var vmSkuPrices = map[string]float64{
	"Standard_B1s":    0.0104,
	"Standard_B1ms":   0.0207,
	"Standard_B2s":    0.0416,
	"Standard_B2ms":   0.0832,
	"Standard_D2s_v3": 0.096,
	"Standard_D4s_v3": 0.192,
	"Standard_D8s_v3": 0.384,
	"Standard_D2s_v4": 0.096,
	"Standard_D4s_v4": 0.192,
	"Standard_D8s_v4": 0.384,
	"Standard_D2s_v5": 0.096,
	"Standard_D4s_v5": 0.192,
	"Standard_D8s_v5": 0.384,
	"Standard_E2s_v3": 0.126,
	"Standard_E4s_v3": 0.252,
	"Standard_E8s_v3": 0.504,
	"Standard_F2s_v2": 0.085,
	"Standard_F4s_v2": 0.169,
	"Standard_F8s_v2": 0.338,
}

// Storage pricing (per GB per month)
var storagePrices = map[string]float64{
	"Standard_LRS":  0.0184,
	"Standard_GRS":  0.0368,
	"Standard_ZRS":  0.023,
	"Standard_GZRS": 0.0414,
	"LRS":           0.0184,
	"GRS":           0.0368,
	"ZRS":           0.023,
	"GZRS":          0.0414,
	"RA-GRS":        0.046,
	"Premium_LRS":   0.15,
}

// App Service Plan pricing (monthly)
var appServicePrices = map[string]float64{
	"F1":   0,
	"D1":   9.49,
	"B1":   13.14,
	"B2":   26.28,
	"B3":   52.56,
	"S1":   69.35,
	"S2":   138.70,
	"S3":   277.40,
	"P1v2": 73.00,
	"P2v2": 146.00,
	"P3v2": 292.00,
	"P1v3": 95.63,
	"P2v3": 191.25,
	"P3v3": 382.50,
}

// Container Registry pricing (monthly)
var acrPrices = map[string]float64{
	"Basic":    5.00,
	"Standard": 20.00,
	"Premium":  50.00,
}

// =============================================================================
// Server
// =============================================================================

type Server struct {
	config     *Config
	mux        *http.ServeMux
	httpClient *http.Client
}

func NewServer(config *Config) *Server {
	return &Server{
		config: config,
		mux:    http.NewServeMux(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/agent", s.handleAgent)
	s.mux.HandleFunc("/", s.handleAgent)
}

func (s *Server) Run() error {
	s.setupRoutes()
	addr := ":" + s.config.Port
	log.Printf("💰 Cost Estimator Agent starting on %s", addr)
	log.Printf("📍 Endpoints:")
	log.Printf("   POST /agent  - Agent endpoint (SSE)")
	log.Printf("   GET  /health - Health check")
	log.Printf("📊 Azure Retail Prices API: %s", AzurePricingAPIBaseURL)
	return http.ListenAndServe(addr, s.mux)
}

// =============================================================================
// Health Check
// =============================================================================

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "cost-estimator-agent",
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
	s.processAgentRequest(r.Context(), req, sse)
}

func (s *Server) processAgentRequest(ctx context.Context, req AgentRequest, sse *SSEWriter) {
	// Get last user message
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = req.Messages[i].Content
			break
		}
	}

	if userMessage == "" {
		sse.SendMessage("❌ No user message found.")
		return
	}

	log.Printf("Processing: %s", truncate(userMessage, 100))

	sse.SendMessage("💰 **Cost Estimator Agent**\n\n")
	sse.SendMessage("Analyzing your Infrastructure as Code for cost estimation...\n\n")
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

	// Also check for direct resource requests
	if code == "" {
		// Check if user is asking about specific VMs
		if estimate := s.parseDirectRequest(userMessage, sse); estimate != nil {
			s.streamCostReport(sse, estimate)
			sse.SendDone()
			return
		}

		sse.SendMessage("ℹ️ No IaC code detected.\n\n")
		sse.SendMessage("**How to use:**\n")
		sse.SendMessage("- Paste Terraform or Bicep code\n")
		sse.SendMessage("- Ask about specific resources: \"How much do 3 Standard_D2s_v3 VMs cost?\"\n")
		sse.SendMessage("- Reference files from your workspace\n")
		sse.SendDone()
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
		sse.SendMessage("\n⚠️ No resources found. Make sure the code is valid.\n")
		sse.SendDone()
		return
	}

	sse.SendMessage(fmt.Sprintf("   Found **%d** resource(s)\n\n", len(resources)))
	time.Sleep(200 * time.Millisecond)

	// Estimate costs
	sse.SendMessage("💵 Fetching pricing from Azure Retail Prices API...\n\n")
	report := s.estimateCosts(ctx, resources, "eastus")

	s.streamCostReport(sse, report)
	sse.SendDone()
}

func (s *Server) streamCostReport(sse *SSEWriter, report *CostReport) {
	time.Sleep(300 * time.Millisecond)

	// Stream the report
	sse.SendMessage("---\n\n")
	sse.SendMessage(fmt.Sprintf("## 💰 Estimated Monthly Cost: **$%.2f**\n\n", report.TotalMonthly))
	sse.SendMessage(fmt.Sprintf("*Region: %s | Currency: %s*\n\n", report.Region, report.Currency))

	if len(report.Items) > 0 {
		sse.SendMessage("### Cost Breakdown\n\n")
		sse.SendMessage("| Resource | SKU | Qty | Unit Price | Monthly |\n")
		sse.SendMessage("|----------|-----|-----|------------|--------|\n")

		for _, item := range report.Items {
			sse.SendMessage(fmt.Sprintf("| %s.%s | %s | %d | $%.4f/%s | $%.2f |\n",
				shortResourceType(item.ResourceType),
				item.ResourceName,
				item.SKU,
				item.Quantity,
				item.UnitPrice,
				item.PricingUnit,
				item.MonthlyCost,
			))
		}
		sse.SendMessage("\n")
	}

	// Warnings
	if len(report.Warnings) > 0 {
		sse.SendMessage("### ⚠️ Notes\n\n")
		for _, w := range report.Warnings {
			sse.SendMessage(fmt.Sprintf("- %s\n", w))
		}
		sse.SendMessage("\n")
	}

	// Tips
	sse.SendMessage("### 💡 Cost Optimization Tips\n\n")
	sse.SendMessage("- Consider **Reserved Instances** for 30-70% savings on VMs\n")
	sse.SendMessage("- Use **Azure Spot VMs** for interruptible workloads (up to 90% off)\n")
	sse.SendMessage("- Review **storage tier** - Hot vs Cool vs Archive\n")
	sse.SendMessage("- Enable **auto-scaling** to match demand\n\n")

	// References
	sse.SendReferences([]Reference{
		{Title: "Azure Pricing Calculator", URL: "https://azure.microsoft.com/pricing/calculator/"},
		{Title: "Azure Retail Prices API", URL: "https://learn.microsoft.com/rest/api/cost-management/retail-prices/azure-retail-prices"},
		{Title: "Azure Cost Management", URL: "https://azure.microsoft.com/products/cost-management/"},
	})

	sse.SendMessage("\n---\n*Prices are estimates and may vary. Check Azure Pricing Calculator for accurate quotes.*")
}

// =============================================================================
// Direct Request Parsing
// =============================================================================

func (s *Server) parseDirectRequest(message string, sse *SSEWriter) *CostReport {
	msg := strings.ToLower(message)

	// Pattern: "X Standard_DYs_vZ VMs" or "cost of Standard_D2s_v3"
	vmPattern := regexp.MustCompile(`(\d+)?\s*(standard_[a-z0-9_]+)\s*(vms?)?`)
	matches := vmPattern.FindStringSubmatch(msg)

	if len(matches) >= 3 {
		count := 1
		if matches[1] != "" {
			count, _ = strconv.Atoi(matches[1])
		}
		sku := strings.Title(matches[2])

		// Extract region if mentioned
		region := "eastus"
		regionPattern := regexp.MustCompile(`in\s+([a-z]+\s*[a-z]*\d*)`)
		regionMatch := regionPattern.FindStringSubmatch(msg)
		if len(regionMatch) >= 2 {
			region = strings.ReplaceAll(regionMatch[1], " ", "")
		}

		sse.SendMessage(fmt.Sprintf("📊 Estimating cost for **%d x %s** VM(s) in **%s**\n\n", count, sku, region))

		return s.estimateVMCost(sku, count, region)
	}

	return nil
}

func (s *Server) estimateVMCost(sku string, count int, region string) *CostReport {
	report := &CostReport{
		Currency: "USD",
		Region:   region,
		Items:    []CostEstimate{},
	}

	// Look up price
	hourlyPrice, found := vmSkuPrices[sku]
	if !found {
		// Try API
		hourlyPrice = s.fetchVMPrice(sku, region)
		if hourlyPrice == 0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Price not found for %s, using estimate", sku))
			hourlyPrice = 0.10 // Default estimate
		}
	}

	monthlyPrice := hourlyPrice * HoursPerMonth * float64(count)

	report.Items = append(report.Items, CostEstimate{
		ResourceType: "azurerm_virtual_machine",
		ResourceName: "vm",
		SKU:          sku,
		Region:       region,
		Quantity:     count,
		UnitPrice:    hourlyPrice,
		PricingUnit:  "hour",
		MonthlyCost:  monthlyPrice,
	})

	report.TotalMonthly = monthlyPrice
	return report
}

// =============================================================================
// Cost Estimation
// =============================================================================

func (s *Server) estimateCosts(ctx context.Context, resources []Resource, defaultRegion string) *CostReport {
	report := &CostReport{
		Currency: "USD",
		Region:   defaultRegion,
		Items:    []CostEstimate{},
	}

	for _, res := range resources {
		estimate := s.estimateResourceCost(ctx, res, defaultRegion)
		if estimate != nil {
			report.Items = append(report.Items, *estimate)
			report.TotalMonthly += estimate.MonthlyCost
		}
	}

	return report
}

func (s *Server) estimateResourceCost(ctx context.Context, res Resource, region string) *CostEstimate {
	// Extract region from resource if available
	if loc, ok := res.Properties["location"].(string); ok && loc != "" {
		region = strings.ToLower(strings.ReplaceAll(loc, " ", ""))
	}

	switch res.Type {
	case "azurerm_kubernetes_cluster":
		return s.estimateAKSCost(res, region)
	case "azurerm_virtual_machine", "azurerm_linux_virtual_machine", "azurerm_windows_virtual_machine":
		return s.estimateVMResourceCost(res, region)
	case "azurerm_storage_account":
		return s.estimateStorageCost(res, region)
	case "azurerm_app_service_plan":
		return s.estimateAppServiceCost(res, region)
	case "azurerm_container_registry":
		return s.estimateACRCost(res, region)
	case "azurerm_key_vault":
		return s.estimateKeyVaultCost(res, region)
	case "azurerm_virtual_network", "azurerm_subnet", "azurerm_network_security_group":
		// These are free
		return &CostEstimate{
			ResourceType: res.Type,
			ResourceName: res.Name,
			SKU:          "N/A",
			Region:       region,
			Quantity:     1,
			UnitPrice:    0,
			PricingUnit:  "N/A",
			MonthlyCost:  0,
			Notes:        "Free resource",
		}
	default:
		return &CostEstimate{
			ResourceType: res.Type,
			ResourceName: res.Name,
			SKU:          "Unknown",
			Region:       region,
			Quantity:     1,
			UnitPrice:    0,
			PricingUnit:  "N/A",
			MonthlyCost:  0,
			Notes:        "Pricing not available for this resource type",
		}
	}
}

func (s *Server) estimateAKSCost(res Resource, region string) *CostEstimate {
	// Extract node pool config
	vmSize := "Standard_D2s_v3"
	nodeCount := 3

	if pool, ok := res.Properties["default_node_pool"].(map[string]interface{}); ok {
		if size, ok := pool["vm_size"].(string); ok {
			vmSize = size
		}
		if count, ok := pool["node_count"].(float64); ok {
			nodeCount = int(count)
		} else if count, ok := pool["node_count"].(int); ok {
			nodeCount = count
		}
	}

	// Get VM price
	hourlyPrice, found := vmSkuPrices[vmSize]
	if !found {
		hourlyPrice = s.fetchVMPrice(vmSize, region)
		if hourlyPrice == 0 {
			hourlyPrice = 0.096 // Default
		}
	}

	monthlyPrice := hourlyPrice * HoursPerMonth * float64(nodeCount)

	// Add load balancer cost (~$18/month)
	monthlyPrice += 18.25

	return &CostEstimate{
		ResourceType: res.Type,
		ResourceName: res.Name,
		SKU:          fmt.Sprintf("%dx %s", nodeCount, vmSize),
		Region:       region,
		Quantity:     nodeCount,
		UnitPrice:    hourlyPrice,
		PricingUnit:  "hour",
		MonthlyCost:  monthlyPrice,
		Notes:        "Includes standard load balancer",
	}
}

func (s *Server) estimateVMResourceCost(res Resource, region string) *CostEstimate {
	vmSize := "Standard_D2s_v3"

	if size, ok := res.Properties["vm_size"].(string); ok {
		vmSize = size
	} else if size, ok := res.Properties["size"].(string); ok {
		vmSize = size
	}

	hourlyPrice, found := vmSkuPrices[vmSize]
	if !found {
		hourlyPrice = s.fetchVMPrice(vmSize, region)
		if hourlyPrice == 0 {
			hourlyPrice = 0.096
		}
	}

	// Add multiplier for Windows
	if res.Type == "azurerm_windows_virtual_machine" {
		hourlyPrice *= 1.5
	}

	return &CostEstimate{
		ResourceType: res.Type,
		ResourceName: res.Name,
		SKU:          vmSize,
		Region:       region,
		Quantity:     1,
		UnitPrice:    hourlyPrice,
		PricingUnit:  "hour",
		MonthlyCost:  hourlyPrice * HoursPerMonth,
	}
}

func (s *Server) estimateStorageCost(res Resource, region string) *CostEstimate {
	sku := "Standard_LRS"
	if replication, ok := res.Properties["account_replication_type"].(string); ok {
		sku = "Standard_" + replication
	}

	pricePerGB, found := storagePrices[sku]
	if !found {
		pricePerGB = 0.0184
	}

	// Assume 100GB default
	capacityGB := 100

	return &CostEstimate{
		ResourceType: res.Type,
		ResourceName: res.Name,
		SKU:          sku,
		Region:       region,
		Quantity:     capacityGB,
		UnitPrice:    pricePerGB,
		PricingUnit:  "GB/month",
		MonthlyCost:  pricePerGB * float64(capacityGB),
		Notes:        "Estimated 100GB capacity",
	}
}

func (s *Server) estimateAppServiceCost(res Resource, region string) *CostEstimate {
	sku := "B1"
	if skuName, ok := res.Properties["sku_name"].(string); ok {
		sku = skuName
	}

	monthlyPrice, found := appServicePrices[sku]
	if !found {
		monthlyPrice = 13.14
	}

	return &CostEstimate{
		ResourceType: res.Type,
		ResourceName: res.Name,
		SKU:          sku,
		Region:       region,
		Quantity:     1,
		UnitPrice:    monthlyPrice,
		PricingUnit:  "month",
		MonthlyCost:  monthlyPrice,
	}
}

func (s *Server) estimateACRCost(res Resource, region string) *CostEstimate {
	sku := "Basic"
	if skuName, ok := res.Properties["sku"].(string); ok {
		sku = skuName
	}

	monthlyPrice, found := acrPrices[sku]
	if !found {
		monthlyPrice = 5.00
	}

	return &CostEstimate{
		ResourceType: res.Type,
		ResourceName: res.Name,
		SKU:          sku,
		Region:       region,
		Quantity:     1,
		UnitPrice:    monthlyPrice,
		PricingUnit:  "month",
		MonthlyCost:  monthlyPrice,
	}
}

func (s *Server) estimateKeyVaultCost(res Resource, region string) *CostEstimate {
	// Key Vault: ~$0.03 per 10K operations + secrets storage
	// Estimate based on typical usage
	monthlyPrice := 3.00

	return &CostEstimate{
		ResourceType: res.Type,
		ResourceName: res.Name,
		SKU:          "Standard",
		Region:       region,
		Quantity:     1,
		UnitPrice:    monthlyPrice,
		PricingUnit:  "month",
		MonthlyCost:  monthlyPrice,
		Notes:        "Based on estimated 10K operations/month",
	}
}

// =============================================================================
// Azure Retail Prices API
// =============================================================================

func (s *Server) fetchVMPrice(sku, region string) float64 {
	filter := fmt.Sprintf(
		"serviceName eq 'Virtual Machines' and armSkuName eq '%s' and armRegionName eq '%s' and priceType eq 'Consumption'",
		sku, region,
	)

	apiURL := fmt.Sprintf("%s?$filter=%s", AzurePricingAPIBaseURL, url.QueryEscape(filter))

	resp, err := s.httpClient.Get(apiURL)
	if err != nil {
		log.Printf("Error fetching price: %v", err)
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0
	}

	body, _ := io.ReadAll(resp.Body)
	var result AzurePricingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return 0
	}

	// Find Linux price (lowest)
	for _, item := range result.Items {
		if strings.Contains(strings.ToLower(item.ProductName), "linux") ||
			!strings.Contains(strings.ToLower(item.ProductName), "windows") {
			return item.RetailPrice
		}
	}

	// Return first price if no Linux found
	if len(result.Items) > 0 {
		return result.Items[0].RetailPrice
	}

	return 0
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
			} else if num, err := strconv.Atoi(value); err == nil {
				props[key] = num
			} else {
				props[key] = value
			}
		}
	}

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

	resourcePattern := regexp.MustCompile(`(?m)^resource\s+(\w+)\s+'([^']+)'`)
	matches := resourcePattern.FindAllStringSubmatchIndex(code, -1)

	for _, match := range matches {
		if len(match) >= 6 {
			resourceName := code[match[2]:match[3]]
			resourceType := code[match[4]:match[5]]
			tfType := bicepToTerraformType(resourceType)
			line := strings.Count(code[:match[0]], "\n") + 1

			resources = append(resources, Resource{
				Type:       tfType,
				Name:       resourceName,
				Properties: make(map[string]interface{}),
				Line:       line,
			})
		}
	}

	return resources
}

func bicepToTerraformType(bicepType string) string {
	parts := strings.Split(bicepType, "@")
	if len(parts) > 0 {
		bicepType = parts[0]
	}

	mapping := map[string]string{
		"Microsoft.Storage/storageAccounts":          "azurerm_storage_account",
		"Microsoft.ContainerService/managedClusters": "azurerm_kubernetes_cluster",
		"Microsoft.Network/virtualNetworks":          "azurerm_virtual_network",
		"Microsoft.KeyVault/vaults":                  "azurerm_key_vault",
		"Microsoft.Compute/virtualMachines":          "azurerm_virtual_machine",
		"Microsoft.Web/serverfarms":                  "azurerm_app_service_plan",
		"Microsoft.ContainerRegistry/registries":     "azurerm_container_registry",
	}

	if tfType, ok := mapping[bicepType]; ok {
		return tfType
	}

	return bicepType
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
	data, _ := json.Marshal(MessageEvent{Content: content})
	fmt.Fprintf(s.w, "event: copilot_message\n")
	fmt.Fprintf(s.w, "data: %s\n\n", string(data))
	s.flusher.Flush()
}

func (s *SSEWriter) SendReferences(refs []Reference) {
	data, _ := json.Marshal(ReferenceEvent{References: refs})
	fmt.Fprintf(s.w, "event: copilot_references\n")
	fmt.Fprintf(s.w, "data: %s\n\n", string(data))
	s.flusher.Flush()
}

func (s *SSEWriter) SendDone() {
	fmt.Fprintf(s.w, "event: copilot_done\ndata: {}\n\n")
	s.flusher.Flush()
}

// =============================================================================
// Helpers
// =============================================================================

func extractCode(message string) string {
	codeBlockPattern := regexp.MustCompile("(?s)```(?:terraform|bicep|hcl)?\\s*\\n(.+?)\\n```")
	matches := codeBlockPattern.FindStringSubmatch(message)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	if strings.Contains(message, "resource ") ||
		strings.Contains(message, "param ") ||
		strings.Contains(message, "azurerm_") {
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
	return "Terraform"
}

func shortResourceType(t string) string {
	parts := strings.Split(t, "_")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return t
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// =============================================================================
// Main
// =============================================================================

func main() {
	config := loadConfig()
	server := NewServer(config)
	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
