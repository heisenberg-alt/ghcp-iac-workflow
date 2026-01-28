// IaC Governance CLI - Terminal interface for Enterprise IaC Platform
// Usage: iac-cli [command] [args...]
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
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
)

type Command struct {
	Name        string
	Aliases     []string
	Port        int
	Description string
	Usage       string
	Example     string
}

var commands = []Command{
	{Name: "help", Aliases: []string{"h", "?"}, Port: 0, Description: "Show all available commands", Usage: "help"},
	{Name: "status", Aliases: []string{"s"}, Port: 0, Description: "Show all agent status", Usage: "status"},
	{Name: "check", Aliases: []string{"full", "orchestrate"}, Port: 8090, Description: "Run full governance check (all agents)", Usage: "check <code>", Example: "check resource \"azurerm_storage_account\" \"sa\" { }"},
	{Name: "policy", Aliases: []string{"p"}, Port: 8081, Description: "Check IaC against organization policies", Usage: "policy <code>", Example: "policy resource \"azurerm_key_vault\" \"kv\" { }"},
	{Name: "cost", Aliases: []string{"pricing", "c"}, Port: 8082, Description: "Estimate Azure resource costs", Usage: "cost <code>", Example: "cost resource \"azurerm_storage_account\" \"sa\" { account_tier = \"Premium\" }"},
	{Name: "drift", Aliases: []string{"d"}, Port: 8083, Description: "Detect infrastructure drift", Usage: "drift [resource_group]", Example: "drift my-resource-group"},
	{Name: "security", Aliases: []string{"scan", "sec"}, Port: 8084, Description: "Scan for security vulnerabilities", Usage: "security <code>", Example: "security resource \"azurerm_storage_account\" \"sa\" { enable_https_traffic_only = false }"},
	{Name: "compliance", Aliases: []string{"audit", "comp"}, Port: 8085, Description: "Audit against CIS, NIST, SOC2", Usage: "compliance [framework] <code>", Example: "compliance CIS resource \"azurerm_key_vault\" \"kv\" { }"},
	{Name: "modules", Aliases: []string{"registry", "mod"}, Port: 8086, Description: "Search approved IaC modules", Usage: "modules [search_term]", Example: "modules storage"},
	{Name: "impact", Aliases: []string{"blast", "i"}, Port: 8087, Description: "Analyze blast radius of changes", Usage: "impact <description>", Example: "impact delete resource group prod-rg"},
	{Name: "deploy", Aliases: []string{"promote", "dep"}, Port: 8088, Description: "Manage environment promotions", Usage: "deploy [status|promote <app> <env>]", Example: "deploy promote my-app staging"},
	{Name: "notify", Aliases: []string{"alerts", "n"}, Port: 8089, Description: "Manage notifications", Usage: "notify [channels|test <channel>]", Example: "notify test teams"},
}

func main() {
	if len(os.Args) < 2 {
		// Interactive mode
		runInteractive()
		return
	}

	// Command mode
	cmd := strings.ToLower(os.Args[1])
	args := ""
	if len(os.Args) > 2 {
		args = strings.Join(os.Args[2:], " ")
	}

	executeCommand(cmd, args)
}

func runInteractive() {
	printBanner()
	fmt.Printf("%sType 'help' for available commands, 'exit' to quit%s\n\n", colorCyan, colorReset)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s❯%s ", colorGreen, colorReset)
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" || input == "q" {
			fmt.Printf("%sGoodbye!%s\n", colorCyan, colorReset)
			break
		}

		// Parse command and args
		parts := strings.SplitN(input, " ", 2)
		cmd := strings.ToLower(parts[0])
		args := ""
		if len(parts) > 1 {
			args = parts[1]
		}

		// Remove leading slash if present
		cmd = strings.TrimPrefix(cmd, "/")

		executeCommand(cmd, args)
		fmt.Println()
	}
}

func executeCommand(cmd string, args string) {
	// Find command
	var foundCmd *Command
	for i := range commands {
		if commands[i].Name == cmd {
			foundCmd = &commands[i]
			break
		}
		for _, alias := range commands[i].Aliases {
			if alias == cmd {
				foundCmd = &commands[i]
				break
			}
		}
		if foundCmd != nil {
			break
		}
	}

	if foundCmd == nil {
		fmt.Printf("%sUnknown command: %s%s\n", colorRed, cmd, colorReset)
		fmt.Printf("Type 'help' for available commands\n")
		return
	}

	// Handle special commands
	switch foundCmd.Name {
	case "help":
		showHelp()
	case "status":
		showStatus()
	default:
		sendToAgent(foundCmd.Port, foundCmd.Name, args)
	}
}

func showHelp() {
	fmt.Printf("\n%s%s╔══════════════════════════════════════════════════════════════════╗%s\n", colorBold, colorBlue, colorReset)
	fmt.Printf("%s%s║          🏢 IaC GOVERNANCE CLI - COMMAND REFERENCE              ║%s\n", colorBold, colorBlue, colorReset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════════════════════╝%s\n\n", colorBold, colorBlue, colorReset)

	fmt.Printf("%s%s🎯 GOVERNANCE COMMANDS%s\n", colorBold, colorPurple, colorReset)
	fmt.Printf("─────────────────────────────────────────────────────────────────────\n")
	printCmd("check", "full, orchestrate", "Run full governance check (all agents)")
	printCmd("policy", "p", "Check IaC against organization policies")
	printCmd("cost", "pricing, c", "Estimate Azure resource costs")
	printCmd("security", "scan, sec", "Scan for security vulnerabilities")
	printCmd("compliance", "audit, comp", "Audit against CIS, NIST, SOC2")

	fmt.Printf("\n%s%s🔧 OPERATIONS COMMANDS%s\n", colorBold, colorPurple, colorReset)
	fmt.Printf("─────────────────────────────────────────────────────────────────────\n")
	printCmd("drift", "d", "Detect infrastructure drift")
	printCmd("impact", "blast, i", "Analyze blast radius of changes")
	printCmd("deploy", "promote, dep", "Manage environment promotions")
	printCmd("modules", "registry, mod", "Search approved IaC modules")
	printCmd("notify", "alerts, n", "Manage notifications")

	fmt.Printf("\n%s%s⚙️  UTILITY COMMANDS%s\n", colorBold, colorPurple, colorReset)
	fmt.Printf("─────────────────────────────────────────────────────────────────────\n")
	printCmd("status", "s", "Show all agent status")
	printCmd("help", "h, ?", "Show this help message")
	printCmd("exit", "quit, q", "Exit interactive mode")

	fmt.Printf("\n%s%s💡 EXAMPLES%s\n", colorBold, colorPurple, colorReset)
	fmt.Printf("─────────────────────────────────────────────────────────────────────\n")
	fmt.Printf("  %siac-cli cost%s resource \"azurerm_storage_account\" \"sa\" { account_tier = \"Premium\" }\n", colorGreen, colorReset)
	fmt.Printf("  %siac-cli security%s resource \"azurerm_storage_account\" \"sa\" { enable_https = false }\n", colorGreen, colorReset)
	fmt.Printf("  %siac-cli check%s < main.tf\n", colorGreen, colorReset)
	fmt.Printf("  cat main.tf | %siac-cli policy%s\n", colorGreen, colorReset)
	fmt.Printf("\n")
}

func printCmd(name, aliases, desc string) {
	fmt.Printf("  %s%-12s%s %s(aliases: %s)%s\n", colorGreen, name, colorReset, colorYellow, aliases, colorReset)
	fmt.Printf("               %s%s%s\n", colorWhite, desc, colorReset)
}

func showStatus() {
	fmt.Printf("\n%sChecking agent status...%s\n\n", colorCyan, colorReset)

	agents := []struct {
		Name string
		Port int
		Icon string
	}{
		{"Policy Checker", 8081, "📋"},
		{"Cost Estimator", 8082, "💰"},
		{"Drift Detector", 8083, "🔄"},
		{"Security Scanner", 8084, "🔒"},
		{"Compliance Auditor", 8085, "✅"},
		{"Module Registry", 8086, "📦"},
		{"Impact Analyzer", 8087, "💥"},
		{"Deploy Promoter", 8088, "🚀"},
		{"Notification Manager", 8089, "🔔"},
		{"Orchestrator", 8090, "🎯"},
	}

	online := 0
	for _, agent := range agents {
		status, details := checkHealth(agent.Port)
		if status == "online" {
			online++
			fmt.Printf("  %s🟢%s %s %-22s %s:%d%s", colorGreen, colorReset, agent.Icon, agent.Name, colorCyan, agent.Port, colorReset)
			if details != "" && details != "Healthy" {
				fmt.Printf(" %s(%s)%s", colorYellow, details, colorReset)
			}
			fmt.Println()
		} else {
			fmt.Printf("  %s🔴%s %s %-22s %s:%d%s %s(%s)%s\n", colorRed, colorReset, agent.Icon, agent.Name, colorCyan, agent.Port, colorReset, colorRed, details, colorReset)
		}
	}

	fmt.Printf("\n%s%sTotal: %d/%d agents online%s\n", colorBold, colorGreen, online, len(agents), colorReset)
}

func checkHealth(port int) (string, string) {
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://localhost:%d/health", port)

	resp, err := client.Get(url)
	if err != nil {
		return "offline", "Connection failed"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "error", fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var healthData map[string]interface{}
	if err := json.Unmarshal(body, &healthData); err == nil {
		details := []string{}
		if rules, ok := healthData["policy_rules"].(float64); ok {
			details = append(details, fmt.Sprintf("%d rules", int(rules)))
		}
		if rules, ok := healthData["security_rules"].(float64); ok {
			details = append(details, fmt.Sprintf("%d security rules", int(rules)))
		}
		if frameworks, ok := healthData["frameworks"].([]interface{}); ok {
			details = append(details, fmt.Sprintf("%d frameworks", len(frameworks)))
		}
		if modules, ok := healthData["approved_modules"].(float64); ok {
			details = append(details, fmt.Sprintf("%d modules", int(modules)))
		}
		if agents, ok := healthData["agents"].(float64); ok {
			details = append(details, fmt.Sprintf("%d sub-agents", int(agents)))
		}
		if len(details) > 0 {
			return "online", strings.Join(details, ", ")
		}
	}

	return "online", "Healthy"
}

func sendToAgent(port int, cmdName string, content string) {
	// Check for piped input if no args provided
	if content == "" {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Data is being piped
			input, _ := io.ReadAll(os.Stdin)
			content = string(input)
		}
	}

	if content == "" {
		content = fmt.Sprintf("Show %s information", cmdName)
	}

	fmt.Printf("%sSending to %s agent (port %d)...%s\n\n", colorCyan, cmdName, port, colorReset)

	client := &http.Client{Timeout: 30 * time.Second}

	reqBody := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": content},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("http://localhost:%d/agent", port)
	resp, err := client.Post(url, "application/json", strings.NewReader(string(bodyBytes)))
	if err != nil {
		fmt.Printf("%sError: %s%s\n", colorRed, err.Error(), colorReset)
		return
	}
	defer resp.Body.Close()

	// Read SSE response
	reader := bufio.NewReader(resp.Body)
	var fullResponse strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			data := line[6:]
			if data == "[DONE]" {
				break
			}

			var sseData map[string]interface{}
			if err := json.Unmarshal([]byte(data), &sseData); err == nil {
				if choices, ok := sseData["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
						if delta, ok := choice["delta"].(map[string]interface{}); ok {
							if content, ok := delta["content"].(string); ok {
								fullResponse.WriteString(content)
								fmt.Print(content)
							}
						}
					}
				}
			}
		}
	}

	if fullResponse.Len() == 0 {
		// If no SSE format, read raw response
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("%s", string(body))
	}

	fmt.Println()
}

func printBanner() {
	fmt.Printf("%s%s", colorBold, colorBlue)
	fmt.Println(`
╔══════════════════════════════════════════════════════════════════╗
║                                                                  ║
║     🏢 Enterprise IaC Governance CLI                             ║
║                                                                  ║
║     Policy • Cost • Security • Compliance • Drift • Deploy       ║
║                                                                  ║
╚══════════════════════════════════════════════════════════════════╝`)
	fmt.Printf("%s\n", colorReset)
}
