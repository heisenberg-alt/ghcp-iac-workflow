// Package orchestrator coordinates multi-agent analysis workflows.
package orchestrator

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/llm"
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
	IntentHelp    Intent = "help"
)

// AgentLookup returns a registered agent by ID.
type AgentLookup func(id string) (protocol.Agent, bool)

// Agent coordinates multi-agent workflows using the registry.
type Agent struct {
	lookup    AgentLookup
	llmClient *llm.Client
	enableLLM bool
}

// New creates a new orchestrator Agent that looks up agents via the provided function.
func New(lookup AgentLookup, opts ...Option) *Agent {
	a := &Agent{lookup: lookup}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Option configures an orchestrator Agent.
type Option func(*Agent)

// WithLLM enables LLM-enhanced orchestration (executive summaries).
func WithLLM(client *llm.Client) Option {
	return func(a *Agent) {
		a.llmClient = client
		a.enableLLM = client != nil
	}
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

	// Tee emitter to capture output for executive summary
	tee := &teeEmitter{inner: emit}

	for _, id := range agentIDs {
		// Check context before invoking each agent
		select {
		case <-ctx.Done():
			emit.SendMessage(fmt.Sprintf("\n_Analysis interrupted: %v_\n", ctx.Err()))
			return ctx.Err()
		default:
		}

		agent, ok := a.lookup(id)
		if !ok {
			msg := fmt.Sprintf("Agent `%s` is not registered.\n\n", id)
			emit.SendMessage(msg)
			tee.captured.WriteString(msg)
			continue
		}
		if err := agent.Handle(ctx, req, tee); err != nil {
			msg := fmt.Sprintf("Agent `%s` failed: %v\n\n", id, err)
			emit.SendMessage(msg)
			tee.captured.WriteString(msg)
		}
	}

	// LLM executive summary after all agents complete
	if a.enableLLM && a.llmClient != nil && req.Token != "" && intent == IntentAnalyze {
		a.executiveSummary(ctx, req, tee.captured.String(), emit)
	}

	return nil
}

// teeEmitter forwards all messages to the inner emitter while capturing text.
type teeEmitter struct {
	inner    protocol.Emitter
	captured strings.Builder
}

func (t *teeEmitter) SendMessage(content string) {
	t.inner.SendMessage(content)
	t.captured.WriteString(content)
}
func (t *teeEmitter) SendReferences(refs []protocol.Reference)    { t.inner.SendReferences(refs) }
func (t *teeEmitter) SendConfirmation(conf protocol.Confirmation) { t.inner.SendConfirmation(conf) }
func (t *teeEmitter) SendError(msg string)                        { t.inner.SendError(msg) }
func (t *teeEmitter) SendDone()                                   { t.inner.SendDone() }

const executivePrompt = `You are a senior cloud architect reviewing a comprehensive IaC governance report. Given the combined output from policy, security, compliance, and impact analysis agents below, provide a concise executive summary:
1. Overall risk rating (Critical/High/Medium/Low) with justification
2. Top 3 issues that need immediate attention
3. A recommended action plan (3-5 bullet points)

Be decisive. Use markdown. Keep it under 150 words.`

func (a *Agent) executiveSummary(ctx context.Context, req protocol.AgentRequest, agentOutput string, emit protocol.Emitter) {
	var sb strings.Builder
	sb.WriteString("## Agent Analysis Output\n\n")
	// Truncate to avoid exceeding token limits
	if len(agentOutput) > 4000 {
		sb.WriteString(agentOutput[:4000])
		sb.WriteString("\n... (truncated)\n")
	} else {
		sb.WriteString(agentOutput)
	}

	emit.SendMessage("\n---\n\n## Executive Summary\n\n")
	messages := []llm.ChatMessage{{Role: llm.RoleUser, Content: sb.String()}}
	contentCh, errCh := a.llmClient.Stream(ctx, req.Token, executivePrompt, messages)
	for content := range contentCh {
		emit.SendMessage(content)
	}
	if err := <-errCh; err != nil {
		emit.SendMessage(fmt.Sprintf("\n_Executive summary unavailable: %v_\n", err))
	}
	emit.SendMessage("\n\n")
}

func (a *Agent) handleHelp(emit protocol.Emitter) {
	emit.SendMessage("## IaC Governance Agent\n\n")
	emit.SendMessage("Available commands:\n\n")
	emit.SendMessage("- **Analyze** — `analyze`, `scan`, `review`, `audit` — Runs policy, security, compliance, and impact analysis\n")
	emit.SendMessage("- **Cost** — `cost`, `estimate`, `pricing` — Estimates monthly Azure costs\n")
	emit.SendMessage("- **Ops** — `deploy`, `drift`, `notify` — Infrastructure operations\n\n")
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
		{IntentHelp, []string{"help", "how to", "what can", "usage", "guide", "capabilities", "status", "health"}},
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
