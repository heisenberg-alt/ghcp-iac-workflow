// Package analyzer provides unified IaC analysis combining policy checking,
// security scanning, compliance auditing, and impact analysis.
package analyzer

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/llm"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/parser"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/server"
)

// Severity levels for findings.
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
	SeverityInfo     = "info"
)

// Finding represents a single analysis result.
type Finding struct {
	RuleID       string `json:"rule_id"`
	Category     string `json:"category"`
	Severity     string `json:"severity"`
	Resource     string `json:"resource"`
	ResourceType string `json:"resource_type"`
	Message      string `json:"message"`
	Remediation  string `json:"remediation"`
}

// Analyzer performs unified IaC analysis.
type Analyzer struct {
	llmClient *llm.Client
	enableLLM bool
	rules     []Rule
	logger    *log.Logger
}

// New creates a new Analyzer.
func New(llmClient *llm.Client, enableLLM bool) *Analyzer {
	return &Analyzer{
		llmClient: llmClient,
		enableLLM: enableLLM,
		rules:     AllRules(),
		logger:    log.New(log.Writer(), "[analyzer] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Analyze performs a comprehensive analysis on IaC code and streams results via SSE.
func (a *Analyzer) Analyze(ctx context.Context, req server.AgentRequest, sse *server.SSEWriter) {
	userMessage := req.GetLastUserMessage()
	code := parser.ExtractCode(userMessage)
	if code == "" {
		code = req.GetCodeFromReferences()
	}

	if code == "" {
		a.showUsage(sse)
		return
	}

	iacType := parser.DetectIaCType(code)
	sse.SendMessage(fmt.Sprintf("Detected **%s** code\n\n", iacType))

	resources := parser.ParseResourcesOfType(code, iacType)
	if len(resources) == 0 {
		sse.SendMessage("No resources found to analyze.\n")
		return
	}

	sse.SendMessage(fmt.Sprintf("Found **%d** resource(s) to analyze\n\n", len(resources)))

	// Run deterministic rules
	var findings []Finding
	for _, res := range resources {
		for _, rule := range a.rules {
			if !rule.Applies(res.Type) {
				continue
			}

			// Pattern-based rules (check raw block)
			if rule.IsPatternRule() {
				if violations := rule.CheckPatterns(res.RawBlock); len(violations) > 0 {
					for _, v := range violations {
						findings = append(findings, Finding{
							RuleID:       rule.ID,
							Category:     rule.Category,
							Severity:     rule.Severity,
							Resource:     res.Name,
							ResourceType: res.Type,
							Message:      v,
							Remediation:  rule.Remediation,
						})
					}
				}
				continue
			}

			// Property-based rules
			if msg := rule.Check(res.Properties); msg != "" {
				findings = append(findings, Finding{
					RuleID:       rule.ID,
					Category:     rule.Category,
					Severity:     rule.Severity,
					Resource:     res.Name,
					ResourceType: res.Type,
					Message:      msg,
					Remediation:  rule.Remediation,
				})
			}
		}
	}

	// Stream findings
	a.streamFindings(sse, findings, resources)

	// Calculate blast radius
	a.streamBlastRadius(sse, resources)

	// LLM enhancement
	token := server.GitHubToken(ctx)
	if a.enableLLM && a.llmClient != nil && token != "" {
		sse.SendMessage("\n## AI-Enhanced Analysis\n\n")
		a.enhanceWithLLM(ctx, token, code, findings, sse)
	}

	// References
	sse.SendReferences([]server.Reference{
		{Title: "CIS Azure Benchmark", URL: "https://www.cisecurity.org/benchmark/azure"},
		{Title: "Azure Security Best Practices", URL: "https://learn.microsoft.com/en-us/azure/security/fundamentals/best-practices-and-patterns"},
	})
}

func (a *Analyzer) streamFindings(sse *server.SSEWriter, findings []Finding, resources []parser.Resource) {
	if len(findings) == 0 {
		sse.SendMessage("## Results\n\nAll checks passed! No issues found.\n")
		return
	}

	// Count by severity
	counts := map[string]int{}
	for _, f := range findings {
		counts[f.Severity]++
	}

	sse.SendMessage("## Summary\n\n")
	sse.SendMessage(fmt.Sprintf("| Severity | Count |\n|----------|-------|\n"))
	for _, sev := range []string{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo} {
		if c, ok := counts[sev]; ok {
			icon := severityIcon(sev)
			sse.SendMessage(fmt.Sprintf("| %s %s | %d |\n", icon, sev, c))
		}
	}
	sse.SendMessage("\n")

	// Group by category
	categories := map[string][]Finding{}
	for _, f := range findings {
		categories[f.Category] = append(categories[f.Category], f)
	}

	for cat, catFindings := range categories {
		sse.SendMessage(fmt.Sprintf("### %s\n\n", cat))
		sse.SendMessage("| Rule | Severity | Resource | Issue | Fix |\n")
		sse.SendMessage("|------|----------|----------|-------|-----|\n")
		for _, f := range catFindings {
			sse.SendMessage(fmt.Sprintf("| %s | %s %s | %s.%s | %s | %s |\n",
				f.RuleID, severityIcon(f.Severity), f.Severity,
				shortType(f.ResourceType), f.Resource,
				f.Message, f.Remediation))
		}
		sse.SendMessage("\n")
	}
}

func (a *Analyzer) streamBlastRadius(sse *server.SSEWriter, resources []parser.Resource) {
	sse.SendMessage("## Blast Radius\n\n")

	total := 0
	for _, res := range resources {
		weight := resourceRiskWeight(res.Type)
		total += weight
		sse.SendMessage(fmt.Sprintf("- **%s.%s** — risk weight: %d\n", shortType(res.Type), res.Name, weight))
	}

	level := "Low"
	if total > 20 {
		level = "Critical"
	} else if total > 10 {
		level = "High"
	} else if total > 5 {
		level = "Medium"
	}

	sse.SendMessage(fmt.Sprintf("\n**Total blast radius: %d (%s)**\n", total, level))
}

const analysisPrompt = `You are a senior cloud security and compliance engineer reviewing Azure IaC code.
Given the code and deterministic findings below, provide:
1. A brief executive summary (2-3 sentences)
2. Additional issues not caught by rules (configuration anti-patterns, missing best practices)
3. Architecture recommendations

Be specific. Reference actual resource names and properties. Use markdown formatting.
Keep it concise - aim for 200-300 words.`

func (a *Analyzer) enhanceWithLLM(ctx context.Context, token, code string, findings []Finding, sse *server.SSEWriter) {
	var sb strings.Builder
	sb.WriteString("## IaC Code\n```\n")
	sb.WriteString(code)
	sb.WriteString("\n```\n\n## Deterministic Findings\n")
	for _, f := range findings {
		sb.WriteString(fmt.Sprintf("- [%s] %s %s: %s\n", f.RuleID, f.Severity, f.Resource, f.Message))
	}

	messages := []llm.ChatMessage{{Role: llm.RoleUser, Content: sb.String()}}
	contentCh, errCh := a.llmClient.Stream(ctx, token, analysisPrompt, messages)

	for content := range contentCh {
		sse.SendMessage(content)
	}
	if err := <-errCh; err != nil {
		a.logger.Printf("LLM enhancement failed: %v", err)
	}
}

func (a *Analyzer) showUsage(sse *server.SSEWriter) {
	sse.SendMessage("## IaC Analyzer\n\n")
	sse.SendMessage("No IaC code detected. Paste Terraform or Bicep code to analyze.\n\n")
	sse.SendMessage("**What I check:**\n")
	sse.SendMessage("- Security: secrets, public access, encryption, TLS\n")
	sse.SendMessage("- Policy: naming, tagging, SKU compliance\n")
	sse.SendMessage("- Compliance: CIS, NIST, SOC2 benchmarks\n")
	sse.SendMessage("- Impact: blast radius, risk assessment\n\n")
	sse.SendMessage("**Example:** Paste a Terraform resource block and ask me to analyze it.\n")
}

func severityIcon(sev string) string {
	switch sev {
	case SeverityCritical:
		return "🔴"
	case SeverityHigh:
		return "🟠"
	case SeverityMedium:
		return "🟡"
	case SeverityLow:
		return "🔵"
	default:
		return "⚪"
	}
}

func shortType(t string) string {
	if i := strings.IndexByte(t, '_'); i >= 0 {
		return t[i+1:]
	}
	return t
}

func resourceRiskWeight(resType string) int {
	weights := map[string]int{
		"azurerm_kubernetes_cluster":     8,
		"azurerm_virtual_machine":        5,
		"azurerm_linux_virtual_machine":  5,
		"azurerm_mssql_server":           7,
		"azurerm_mssql_database":         6,
		"azurerm_cosmosdb_account":       7,
		"azurerm_key_vault":              6,
		"azurerm_storage_account":        4,
		"azurerm_container_registry":     4,
		"azurerm_service_plan":           3,
		"azurerm_redis_cache":            5,
		"azurerm_virtual_network":        3,
		"azurerm_subnet":                 2,
		"azurerm_network_security_group": 4,
	}
	if w, ok := weights[resType]; ok {
		return w
	}
	return 2
}
