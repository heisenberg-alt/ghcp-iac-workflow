// gh-iac - GitHub CLI Extension for IaC Governance
// Integrates with GitHub Copilot for infrastructure-as-code validation
//
// Installation:
//
//	gh extension install ./gh-iac
//
// Usage:
//
//	gh iac help
//	gh iac status
//	gh iac policy <code>
//	gh iac cost <code>
//	gh copilot suggest --target shell "check my terraform" | gh iac
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	Version = "1.0.0"

	// ANSI colors
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
)

// Command represents a CLI command
type Command struct {
	Name        string
	Aliases     []string
	Port        int
	Description string
	Usage       string
}

var commands = []Command{
	{Name: "help", Aliases: []string{"h", "-h", "--help"}, Description: "Show help information"},
	{Name: "version", Aliases: []string{"v", "-v", "--version"}, Description: "Show version"},
	{Name: "status", Aliases: []string{"s"}, Description: "Show all agent status"},
	{Name: "check", Aliases: []string{"full", "governance"}, Port: 8090, Description: "Run full governance check", Usage: "gh iac check <code>"},
	{Name: "policy", Aliases: []string{"p"}, Port: 8081, Description: "Check against policies", Usage: "gh iac policy <code>"},
	{Name: "cost", Aliases: []string{"pricing", "c"}, Port: 8082, Description: "Estimate costs", Usage: "gh iac cost <code>"},
	{Name: "drift", Aliases: []string{"d"}, Port: 8083, Description: "Detect drift", Usage: "gh iac drift [resource_group]"},
	{Name: "security", Aliases: []string{"scan", "sec"}, Port: 8084, Description: "Security scan", Usage: "gh iac security <code>"},
	{Name: "compliance", Aliases: []string{"audit"}, Port: 8085, Description: "Compliance audit", Usage: "gh iac compliance <code>"},
	{Name: "modules", Aliases: []string{"registry"}, Port: 8086, Description: "Module registry", Usage: "gh iac modules [search]"},
	{Name: "impact", Aliases: []string{"blast"}, Port: 8087, Description: "Impact analysis", Usage: "gh iac impact <description>"},
	{Name: "deploy", Aliases: []string{"promote"}, Port: 8088, Description: "Deployment promotion", Usage: "gh iac deploy [status|promote]"},
	{Name: "notify", Aliases: []string{"alerts"}, Port: 8089, Description: "Notifications", Usage: "gh iac notify [channels|test]"},
}

func main() {
	// Check for piped input
	pipedInput := ""
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		input, _ := io.ReadAll(os.Stdin)
		pipedInput = strings.TrimSpace(string(input))
	}

	if len(os.Args) < 2 {
		if pipedInput != "" {
			// Piped input without command - run full check
			runAgent(8090, "check", pipedInput)
			return
		}
		showHelp()
		return
	}

	cmd := strings.ToLower(os.Args[1])
	args := ""
	if len(os.Args) > 2 {
		args = strings.Join(os.Args[2:], " ")
	}

	// Combine with piped input
	if pipedInput != "" && args == "" {
		args = pipedInput
	} else if pipedInput != "" && args != "" {
		args = args + "\n" + pipedInput
	}

	executeCommand(cmd, args)
}

func executeCommand(cmd string, args string) {
	// Find command
	var found *Command
	for i := range commands {
		if commands[i].Name == cmd {
			found = &commands[i]
			break
		}
		for _, alias := range commands[i].Aliases {
			if alias == cmd {
				found = &commands[i]
				break
			}
		}
		if found != nil {
			break
		}
	}

	if found == nil {
		fmt.Fprintf(os.Stderr, "%sUnknown command: %s%s\n", Red, cmd, Reset)
		fmt.Fprintf(os.Stderr, "Run 'gh iac help' for usage\n")
		os.Exit(1)
	}

	switch found.Name {
	case "help":
		showHelp()
	case "version":
		showVersion()
	case "status":
		showStatus()
	default:
		if found.Port > 0 {
			runAgent(found.Port, found.Name, args)
		}
	}
}

func showHelp() {
	fmt.Printf(`%s%sgh-iac%s - GitHub CLI Extension for IaC Governance%s

%sUSAGE%s
  gh iac <command> [arguments]
  cat main.tf | gh iac policy
  gh copilot suggest "check terraform" | gh iac

%sCOMMANDS%s
`, Bold, Cyan, Reset, Reset, Bold, Reset, Bold, Reset)

	// Governance commands
	fmt.Printf("\n  %sGovernance:%s\n", Yellow, Reset)
	printCmd("check", "full, governance", "Run full governance check (all agents)")
	printCmd("policy", "p", "Check IaC against organization policies")
	printCmd("cost", "pricing, c", "Estimate Azure resource costs")
	printCmd("security", "scan, sec", "Scan for security vulnerabilities")
	printCmd("compliance", "audit", "Audit against CIS, NIST, SOC2")

	// Operations commands
	fmt.Printf("\n  %sOperations:%s\n", Yellow, Reset)
	printCmd("drift", "d", "Detect infrastructure drift")
	printCmd("impact", "blast", "Analyze blast radius")
	printCmd("deploy", "promote", "Manage deployments")
	printCmd("modules", "registry", "Search approved modules")
	printCmd("notify", "alerts", "Manage notifications")

	// Utility commands
	fmt.Printf("\n  %sUtility:%s\n", Yellow, Reset)
	printCmd("status", "s", "Show all agent status")
	printCmd("help", "h", "Show this help")
	printCmd("version", "v", "Show version")

	fmt.Printf(`
%sEXAMPLES%s
  # Check policies
  gh iac policy 'resource "azurerm_storage_account" "sa" { name = "test" }'

  # Estimate costs from file
  cat main.tf | gh iac cost

  # Security scan
  gh iac security < infrastructure/main.tf

  # Full governance check
  gh iac check 'resource "azurerm_key_vault" "kv" { }'

  # Use with GitHub Copilot
  gh copilot suggest "create secure storage account" | gh iac security

%sLEARN MORE%s
  https://github.com/your-org/gh-iac

`, Bold, Reset, Bold, Reset)
}

func printCmd(name, aliases, desc string) {
	fmt.Printf("    %s%-12s%s %s%-16s%s %s\n", Green, name, Reset, Dim, aliases, Reset, desc)
}

func showVersion() {
	fmt.Printf("gh-iac version %s\n", Version)
	fmt.Printf("GitHub CLI Extension for IaC Governance\n")
	fmt.Printf("Powered by GitHub Copilot SDK\n")
}

func showStatus() {
	fmt.Printf("\n%s%sIaC Governance Platform Status%s\n\n", Bold, Cyan, Reset)

	agents := []struct {
		Name string
		Port int
		Icon string
		Cmd  string
	}{
		{"Policy Checker", 8081, "📋", "policy"},
		{"Cost Estimator", 8082, "💰", "cost"},
		{"Drift Detector", 8083, "🔄", "drift"},
		{"Security Scanner", 8084, "🔒", "security"},
		{"Compliance Auditor", 8085, "✅", "compliance"},
		{"Module Registry", 8086, "📦", "modules"},
		{"Impact Analyzer", 8087, "💥", "impact"},
		{"Deploy Promoter", 8088, "🚀", "deploy"},
		{"Notification Manager", 8089, "🔔", "notify"},
		{"Orchestrator", 8090, "🎯", "check"},
	}

	online := 0
	for _, agent := range agents {
		status, details := checkHealth(agent.Port)
		if status == "online" {
			online++
			fmt.Printf("  %s●%s %s %-22s %s:%-4d%s  %sgh iac %s%s",
				Green, Reset, agent.Icon, agent.Name, Dim, agent.Port, Reset, Cyan, agent.Cmd, Reset)
			if details != "" && details != "Healthy" {
				fmt.Printf("  %s%s%s", Yellow, details, Reset)
			}
			fmt.Println()
		} else {
			fmt.Printf("  %s●%s %s %-22s %s:%-4d%s  %s%s%s\n",
				Red, Reset, agent.Icon, agent.Name, Dim, agent.Port, Reset, Red, details, Reset)
		}
	}

	fmt.Printf("\n%s%d/%d agents online%s\n\n", Bold, online, len(agents), Reset)
}

func checkHealth(port int) (string, string) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		return "offline", "Connection failed"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "error", fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var health map[string]interface{}
	if err := json.Unmarshal(body, &health); err == nil {
		details := []string{}
		if v, ok := health["policy_rules"].(float64); ok {
			details = append(details, fmt.Sprintf("%d rules", int(v)))
		}
		if v, ok := health["security_rules"].(float64); ok {
			details = append(details, fmt.Sprintf("%d security rules", int(v)))
		}
		if v, ok := health["frameworks"].([]interface{}); ok {
			details = append(details, fmt.Sprintf("%d frameworks", len(v)))
		}
		if v, ok := health["approved_modules"].(float64); ok {
			details = append(details, fmt.Sprintf("%d modules", int(v)))
		}
		if v, ok := health["agents"].(float64); ok {
			details = append(details, fmt.Sprintf("%d agents", int(v)))
		}
		if len(details) > 0 {
			return "online", strings.Join(details, ", ")
		}
	}
	return "online", "Healthy"
}

func runAgent(port int, cmdName string, content string) {
	if content == "" {
		content = fmt.Sprintf("Show %s information and capabilities", cmdName)
	}

	// Show what we're doing
	fmt.Fprintf(os.Stderr, "%s→ Sending to %s agent (port %d)%s\n\n", Dim, cmdName, port, Reset)

	client := &http.Client{Timeout: 60 * time.Second}

	reqBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": content},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	resp, err := client.Post(
		fmt.Sprintf("http://localhost:%d/agent", port),
		"application/json",
		strings.NewReader(string(bodyBytes)),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %s%s\n", Red, err.Error(), Reset)
		fmt.Fprintf(os.Stderr, "Make sure agents are running. Check with: gh iac status\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Stream SSE response (Copilot SDK format)
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)

		// Handle event type line
		if strings.HasPrefix(line, "event: ") {
			eventType := line[7:]
			if eventType == "copilot_done" {
				break
			}
			continue
		}

		// Handle data line
		if strings.HasPrefix(line, "data: ") {
			data := line[6:]
			if data == "[DONE]" || data == "{}" {
				break
			}

			var sseData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &sseData); err == nil {
				// Copilot SDK format: {"content":"..."}
				if content, ok := sseData["content"].(string); ok {
					fmt.Print(content)
				}
			}
		}
	}
	fmt.Println()
}
