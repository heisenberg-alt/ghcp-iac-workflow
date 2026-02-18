// Package orchestrator coordinates multi-agent analysis workflows.
package orchestrator

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// codeBlockRe strips code blocks before keyword matching.
var codeBlockRe = regexp.MustCompile("(?s)```.*?```")

// Intent represents a classified user intent.
type Intent string

const (
	IntentAnalyze Intent = "analyze"
	IntentCost    Intent = "cost"
	IntentOps     Intent = "ops"
	IntentStatus  Intent = "status"
	IntentHelp    Intent = "help"
)

// AgentLookup returns a registered agent by ID.
type AgentLookup func(id string) (protocol.Agent, bool)

// Agent coordinates multi-agent workflows using the registry.
type Agent struct {
	lookup AgentLookup
}

// New creates a new orchestrator Agent that looks up agents via the provided function.
func New(lookup AgentLookup) *Agent {
	return &Agent{lookup: lookup}
}

func (a *Agent) ID() string { return "orchestrator" }

func (a *Agent) Metadata() protocol.AgentMetadata {
	return protocol.AgentMetadata{
		ID:          "orchestrator",
		Name:        "Orchestrator",
		Description: "Coordinates multi-agent analysis workflows based on intent classification",
		Version:     "1.0.0",
	}
}

func (a *Agent) Capabilities() protocol.AgentCapabilities {
	return protocol.AgentCapabilities{
		Formats:       []protocol.SourceFormat{protocol.FormatTerraform, protocol.FormatBicep},
		NeedsIaCInput: true,
	}
}

// Handle classifies intent, selects agents, and runs them in sequence.
func (a *Agent) Handle(ctx context.Context, req protocol.AgentRequest, emit protocol.Emitter) error {
	prompt := protocol.PromptText(req)
	intent := classifyKeywords(prompt)
	agentIDs := agentsForIntent(intent)

	if len(agentIDs) == 0 {
		a.handleHelp(emit)
		return nil
	}

	for _, id := range agentIDs {
		agent, ok := a.lookup(id)
		if !ok {
			emit.SendMessage(fmt.Sprintf("Agent `%s` is not registered.\n\n", id))
			continue
		}
		if err := agent.Handle(ctx, req, emit); err != nil {
			emit.SendMessage(fmt.Sprintf("Agent `%s` failed: %v\n\n", id, err))
		}
	}
	return nil
}

func (a *Agent) handleHelp(emit protocol.Emitter) {
	emit.SendMessage("## IaC Governance Agent\n\n")
	emit.SendMessage("Available commands:\n\n")
	emit.SendMessage("- **Analyze** — `analyze`, `scan`, `review`, `audit` — Runs policy, security, compliance, and impact analysis\n")
	emit.SendMessage("- **Cost** — `cost`, `estimate`, `pricing` — Estimates monthly Azure costs\n")
	emit.SendMessage("- **Ops** — `deploy`, `drift`, `notify` — Infrastructure operations\n")
	emit.SendMessage("- **Status** — `status`, `health` — Agent health check\n\n")
	emit.SendMessage("Include Terraform or Bicep code in a fenced block for analysis.\n")
}

// agentsForIntent maps an intent to the ordered list of agent IDs to invoke.
func agentsForIntent(intent Intent) []string {
	switch intent {
	case IntentAnalyze:
		return []string{"policy", "security", "compliance", "impact"}
	case IntentCost:
		return []string{"cost"}
	case IntentOps:
		return []string{"deploy", "drift", "notification"}
	case IntentStatus:
		return nil // handled as help for now
	case IntentHelp:
		return nil
	default:
		return nil
	}
}

// classifyKeywords determines intent from prompt keywords.
func classifyKeywords(message string) Intent {
	msg := strings.ToLower(message)
	msgNoCode := codeBlockRe.ReplaceAllString(msg, "")

	type scored struct {
		intent Intent
		score  int
	}

	categories := []struct {
		intent   Intent
		keywords []string
	}{
		{IntentAnalyze, []string{"scan", "audit", "review", "analyze", "security", "policy", "compliance", "vulnerability", "check", "full"}},
		{IntentCost, []string{"cost", "price", "pricing", "estimate", "budget", "expensive", "spending"}},
		{IntentOps, []string{"deploy", "promote", "drift", "release", "rollback", "environment", "staging", "production", "notify", "notification"}},
		{IntentStatus, []string{"status", "health", "running", "uptime"}},
		{IntentHelp, []string{"help", "how to", "what can", "usage", "guide", "capabilities"}},
	}

	best := scored{IntentHelp, 0}
	for _, cat := range categories {
		score := 0
		for _, w := range cat.keywords {
			if strings.Contains(msgNoCode, w) {
				score++
			}
		}
		if score > best.score {
			best = scored{cat.intent, score}
		}
	}

	if strings.Contains(msg, "terraform") || strings.Contains(msg, "bicep") {
		best = scored{IntentAnalyze, best.score + 2}
	}

	if best.score > 0 {
		return best.intent
	}

	if strings.Contains(msg, "```") || strings.Contains(msg, "resource ") {
		return IntentAnalyze
	}

	return IntentHelp
}
