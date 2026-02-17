// Package costestimator provides Azure cost estimation from IaC code.
// It uses the Azure Retail Prices API (no auth required) and enhances
// results with LLM-powered optimization recommendations.
package costestimator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/llm"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/parser"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/server"
)

const (
	// AzurePricingAPI is the public Azure Retail Prices endpoint.
	AzurePricingAPI = "https://prices.azure.com/api/retail/prices"
	// HoursPerMonth is the standard billing hours per month.
	HoursPerMonth = 730
)

// CostEstimate represents the cost for a single resource.
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

// CostReport contains the full cost breakdown.
type CostReport struct {
	TotalMonthly float64        `json:"total_monthly"`
	Currency     string         `json:"currency"`
	Region       string         `json:"region"`
	Items        []CostEstimate `json:"items"`
	Warnings     []string       `json:"warnings,omitempty"`
}

// AzurePricingResponse from the retail prices API.
type AzurePricingResponse struct {
	Items []PriceItem `json:"Items"`
	Count int         `json:"Count"`
}

// PriceItem is a single price item from the Azure API.
type PriceItem struct {
	RetailPrice   float64 `json:"retailPrice"`
	UnitOfMeasure string  `json:"unitOfMeasure"`
	ArmRegionName string  `json:"armRegionName"`
	ProductName   string  `json:"productName"`
	SkuName       string  `json:"skuName"`
	ServiceName   string  `json:"serviceName"`
	ArmSkuName    string  `json:"armSkuName"`
	MeterName     string  `json:"meterName"`
}

// Estimator calculates Azure resource costs from IaC code.
type Estimator struct {
	llmClient  *llm.Client
	enableLLM  bool
	enableAPI  bool
	httpClient *http.Client
	logger     *log.Logger
}

// New creates a new Estimator.
func New(llmClient *llm.Client, enableLLM, enableAPI bool) *Estimator {
	return &Estimator{
		llmClient:  llmClient,
		enableLLM:  enableLLM,
		enableAPI:  enableAPI,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     log.New(log.Writer(), "[cost] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Estimate runs cost estimation on IaC code and streams results.
func (e *Estimator) Estimate(ctx context.Context, req server.AgentRequest, sse *server.SSEWriter) {
	userMessage := req.GetLastUserMessage()
	code := parser.ExtractCode(userMessage)
	if code == "" {
		code = req.GetCodeFromReferences()
	}

	if code == "" {
		e.showUsage(sse)
		return
	}

	iacType := parser.DetectIaCType(code)
	sse.SendMessage(fmt.Sprintf("Detected **%s** code\n\n", iacType))

	resources := parser.ParseResourcesOfType(code, iacType)
	if len(resources) == 0 {
		sse.SendMessage("No resources found to estimate costs for.\n")
		return
	}

	sse.SendMessage(fmt.Sprintf("Found **%d** resource(s)\n\n", len(resources)))
	sse.SendMessage("Calculating costs...\n\n")

	report := e.estimateCosts(ctx, resources, "eastus")
	e.streamReport(sse, report)

	// LLM optimization suggestions
	token := server.GitHubToken(ctx)
	if e.enableLLM && e.llmClient != nil && token != "" {
		sse.SendMessage("\n## AI Cost Optimization\n\n")
		e.optimizeWithLLM(ctx, token, code, report, sse)
	}

	sse.SendReferences([]server.Reference{
		{Title: "Azure Pricing Calculator", URL: "https://azure.microsoft.com/pricing/calculator/"},
		{Title: "Azure Cost Management", URL: "https://azure.microsoft.com/products/cost-management/"},
	})

	sse.SendMessage("\n---\n*Prices are estimates. Check Azure Pricing Calculator for exact quotes.*\n")
}

func (e *Estimator) estimateCosts(ctx context.Context, resources []parser.Resource, defaultRegion string) *CostReport {
	report := &CostReport{Currency: "USD", Region: defaultRegion}

	for _, res := range resources {
		if estimate := e.estimateResource(ctx, res, defaultRegion); estimate != nil {
			report.Items = append(report.Items, *estimate)
			report.TotalMonthly += estimate.MonthlyCost
		}
	}

	return report
}

func (e *Estimator) estimateResource(ctx context.Context, res parser.Resource, region string) *CostEstimate {
	if loc, ok := res.Properties["location"].(string); ok && loc != "" {
		region = strings.ToLower(strings.ReplaceAll(loc, " ", ""))
	}

	switch res.Type {
	case "azurerm_kubernetes_cluster":
		return e.estimateAKS(ctx, res, region)
	case "azurerm_virtual_machine", "azurerm_linux_virtual_machine", "azurerm_windows_virtual_machine":
		return e.estimateVM(ctx, res, region)
	case "azurerm_storage_account":
		return e.estimateStorage(res, region)
	case "azurerm_app_service_plan", "azurerm_service_plan":
		return e.estimateAppService(res, region)
	case "azurerm_container_registry":
		return e.estimateACR(res, region)
	case "azurerm_key_vault":
		return &CostEstimate{
			ResourceType: res.Type, ResourceName: res.Name, SKU: "Standard",
			Region: region, Quantity: 1, UnitPrice: 3.00, PricingUnit: "month",
			MonthlyCost: 3.00, Notes: "Based on ~10K operations/month",
		}
	case "azurerm_virtual_network", "azurerm_subnet", "azurerm_network_security_group":
		return &CostEstimate{
			ResourceType: res.Type, ResourceName: res.Name, SKU: "N/A",
			Region: region, Quantity: 1, MonthlyCost: 0, Notes: "Free resource",
		}
	default:
		return &CostEstimate{
			ResourceType: res.Type, ResourceName: res.Name, SKU: "Unknown",
			Region: region, Quantity: 1, MonthlyCost: 0, Notes: "Pricing not available",
		}
	}
}

func (e *Estimator) estimateAKS(ctx context.Context, res parser.Resource, region string) *CostEstimate {
	vmSize := "Standard_D2s_v3"
	nodeCount := 3

	if pool, ok := res.Properties["default_node_pool"].(map[string]interface{}); ok {
		if size, ok := pool["vm_size"].(string); ok {
			vmSize = size
		}
		if count, ok := pool["node_count"].(int); ok {
			nodeCount = count
		}
	}

	hourly := e.lookupVMPrice(ctx, vmSize, region)
	monthly := hourly * HoursPerMonth * float64(nodeCount)
	monthly += 18.25 // standard load balancer

	return &CostEstimate{
		ResourceType: res.Type, ResourceName: res.Name,
		SKU: fmt.Sprintf("%dx %s", nodeCount, vmSize), Region: region,
		Quantity: nodeCount, UnitPrice: hourly, PricingUnit: "hour/node",
		MonthlyCost: monthly, Notes: "Includes standard load balancer ($18.25/mo)",
	}
}

func (e *Estimator) estimateVM(ctx context.Context, res parser.Resource, region string) *CostEstimate {
	vmSize := "Standard_D2s_v3"
	if size, ok := res.Properties["vm_size"].(string); ok {
		vmSize = size
	} else if size, ok := res.Properties["size"].(string); ok {
		vmSize = size
	}

	hourly := e.lookupVMPrice(ctx, vmSize, region)
	if res.Type == "azurerm_windows_virtual_machine" {
		hourly *= 1.5
	}

	return &CostEstimate{
		ResourceType: res.Type, ResourceName: res.Name, SKU: vmSize,
		Region: region, Quantity: 1, UnitPrice: hourly, PricingUnit: "hour",
		MonthlyCost: hourly * HoursPerMonth,
	}
}

func (e *Estimator) estimateStorage(res parser.Resource, region string) *CostEstimate {
	sku := "Standard_LRS"
	if rep, ok := res.Properties["account_replication_type"].(string); ok {
		sku = "Standard_" + rep
	}
	pricePerGB := storagePrices[sku]
	if pricePerGB == 0 {
		pricePerGB = 0.0184
	}
	capacityGB := 100

	return &CostEstimate{
		ResourceType: res.Type, ResourceName: res.Name, SKU: sku,
		Region: region, Quantity: capacityGB, UnitPrice: pricePerGB,
		PricingUnit: "GB/month", MonthlyCost: pricePerGB * float64(capacityGB),
		Notes: "Estimated 100 GB capacity",
	}
}

func (e *Estimator) estimateAppService(res parser.Resource, region string) *CostEstimate {
	sku := "B1"
	if s, ok := res.Properties["sku_name"].(string); ok {
		sku = s
	}
	monthly := appServicePrices[sku]
	if monthly == 0 {
		monthly = 13.14
	}

	return &CostEstimate{
		ResourceType: res.Type, ResourceName: res.Name, SKU: sku,
		Region: region, Quantity: 1, UnitPrice: monthly,
		PricingUnit: "month", MonthlyCost: monthly,
	}
}

func (e *Estimator) estimateACR(res parser.Resource, region string) *CostEstimate {
	sku := "Basic"
	if s, ok := res.Properties["sku"].(string); ok {
		sku = s
	}
	monthly := acrPrices[sku]
	if monthly == 0 {
		monthly = 5.00
	}

	return &CostEstimate{
		ResourceType: res.Type, ResourceName: res.Name, SKU: sku,
		Region: region, Quantity: 1, UnitPrice: monthly,
		PricingUnit: "month", MonthlyCost: monthly,
	}
}

// lookupVMPrice tries the static table first, then the Azure Retail Prices API.
func (e *Estimator) lookupVMPrice(ctx context.Context, sku, region string) float64 {
	if price, ok := vmSkuPrices[sku]; ok {
		return price
	}
	if e.enableAPI {
		if price := e.fetchVMPrice(ctx, sku, region); price > 0 {
			return price
		}
	}
	return 0.096 // default fallback
}

func (e *Estimator) fetchVMPrice(ctx context.Context, sku, region string) float64 {
	filter := fmt.Sprintf(
		"serviceName eq 'Virtual Machines' and armSkuName eq '%s' and armRegionName eq '%s' and priceType eq 'Consumption'",
		sku, region,
	)
	apiURL := fmt.Sprintf("%s?$filter=%s", AzurePricingAPI, url.QueryEscape(filter))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return 0
	}
	resp, err := e.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return 0
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result AzurePricingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return 0
	}

	for _, item := range result.Items {
		if strings.Contains(strings.ToLower(item.ProductName), "linux") ||
			!strings.Contains(strings.ToLower(item.ProductName), "windows") {
			return item.RetailPrice
		}
	}
	if len(result.Items) > 0 {
		return result.Items[0].RetailPrice
	}
	return 0
}

func (e *Estimator) streamReport(sse *server.SSEWriter, report *CostReport) {
	sse.SendMessage(fmt.Sprintf("## Estimated Monthly Cost: **$%.2f**\n\n", report.TotalMonthly))
	sse.SendMessage(fmt.Sprintf("*Region: %s | Currency: %s*\n\n", report.Region, report.Currency))

	if len(report.Items) > 0 {
		sse.SendMessage("| Resource | SKU | Monthly |\n|----------|-----|---------|\n")
		for _, item := range report.Items {
			name := shortType(item.ResourceType) + "." + item.ResourceName
			sse.SendMessage(fmt.Sprintf("| %s | %s | $%.2f |\n", name, item.SKU, item.MonthlyCost))
		}
		sse.SendMessage("\n")
	}

	if len(report.Warnings) > 0 {
		sse.SendMessage("### Notes\n\n")
		for _, w := range report.Warnings {
			sse.SendMessage(fmt.Sprintf("- %s\n", w))
		}
	}
}

const costOptimizePrompt = `You are an Azure cloud cost optimization expert.
Given the IaC code and cost breakdown, provide 3-5 specific, actionable cost optimization recommendations.
Focus on: Reserved Instances, right-sizing, storage tier selection, spot VMs, auto-scaling.
Be concise. Use markdown bullet points.`

func (e *Estimator) optimizeWithLLM(ctx context.Context, token, code string, report *CostReport, sse *server.SSEWriter) {
	var sb strings.Builder
	sb.WriteString("## IaC Code\n```\n")
	sb.WriteString(code)
	sb.WriteString("\n```\n\n## Cost Breakdown\n")
	sb.WriteString(fmt.Sprintf("Total: $%.2f/month\n", report.TotalMonthly))
	for _, item := range report.Items {
		sb.WriteString(fmt.Sprintf("- %s (%s): $%.2f/month\n", item.ResourceName, item.SKU, item.MonthlyCost))
	}

	messages := []llm.ChatMessage{{Role: llm.RoleUser, Content: sb.String()}}
	contentCh, errCh := e.llmClient.Stream(ctx, token, costOptimizePrompt, messages)
	for content := range contentCh {
		sse.SendMessage(content)
	}
	if err := <-errCh; err != nil {
		e.logger.Printf("LLM optimization failed: %v", err)
	}
}

func (e *Estimator) showUsage(sse *server.SSEWriter) {
	sse.SendMessage("## Cost Estimator\n\n")
	sse.SendMessage("No IaC code detected. Provide Terraform or Bicep code to estimate costs.\n\n")
	sse.SendMessage("**What I estimate:**\n")
	sse.SendMessage("- Virtual Machines & AKS clusters\n")
	sse.SendMessage("- Storage Accounts\n")
	sse.SendMessage("- App Service Plans\n")
	sse.SendMessage("- Container Registries, Key Vaults\n\n")
}

func shortType(t string) string {
	if i := strings.IndexByte(t, '_'); i >= 0 {
		return t[i+1:]
	}
	return t
}

// ==================== Price Tables ====================

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

var storagePrices = map[string]float64{
	"Standard_LRS":    0.0184,
	"Standard_GRS":    0.0368,
	"Standard_ZRS":    0.023,
	"Standard_GZRS":   0.0414,
	"Premium_LRS":     0.15,
	"Standard_RA-GRS": 0.046,
}

var appServicePrices = map[string]float64{
	"F1": 0, "D1": 9.49, "B1": 13.14, "B2": 26.28, "B3": 52.56,
	"S1": 69.35, "S2": 138.70, "S3": 277.40,
	"P1v2": 73.00, "P2v2": 146.00, "P3v2": 292.00,
	"P1v3": 95.63, "P2v3": 191.25, "P3v3": 382.50,
}

var acrPrices = map[string]float64{
	"Basic": 5.00, "Standard": 20.00, "Premium": 50.00,
}
