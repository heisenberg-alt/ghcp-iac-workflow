// Package router provides LLM-powered intent classification and request routing.
// It determines which analyzer should handle a given user message by first trying
// LLM classification, then falling back to keyword matching.
package router

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/llm"
)

// codeBlockRe is pre-compiled to avoid per-request regex compilation.
var codeBlockRe = regexp.MustCompile("(?s)```.*?```")

// Intent represents the classified intent of a user message.
type Intent string

const (
	IntentAnalyze Intent = "analyze"
	IntentCost    Intent = "cost"
	IntentOps     Intent = "ops"
	IntentStatus  Intent = "status"
	IntentHelp    Intent = "help"
)

const classifyPrompt = `You are an intent classifier for an IaC governance tool. Classify the user's message into exactly one intent.

Intents:
- "analyze": Security scanning, policy checking, compliance auditing, code review of Terraform/Bicep
- "cost": Cost estimation, pricing, budget analysis
- "ops": Deployment, drift detection, environment promotion, notifications
- "status": Agent status, version info, health check
- "help": Usage help, capabilities, what can you do

Respond with ONLY a JSON object: {"intent": "<intent>"}
Do not include any other text.`

// Router classifies user intents and routes to the appropriate handler.
type Router struct {
	llmClient *llm.Client
	enableLLM bool
	logger    *log.Logger
}

// New creates a new Router.
func New(llmClient *llm.Client, enableLLM bool) *Router {
	return &Router{
		llmClient: llmClient,
		enableLLM: enableLLM,
		logger:    log.New(log.Writer(), "[router] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Classify determines the intent of a user message.
func (r *Router) Classify(ctx context.Context, token, message string) Intent {
	// Try LLM classification first
	if r.enableLLM && r.llmClient != nil && token != "" {
		if intent := r.classifyWithLLM(ctx, token, message); intent != "" {
			r.logger.Printf("LLM classified intent: %s", intent)
			return intent
		}
	}

	// Fallback to keyword matching
	intent := r.classifyKeywords(message)
	r.logger.Printf("Keyword classified intent: %s", intent)
	return intent
}

func (r *Router) classifyWithLLM(ctx context.Context, token, message string) Intent {
	messages := []llm.ChatMessage{
		{Role: llm.RoleUser, Content: message},
	}

	result, err := r.llmClient.Complete(ctx, token, classifyPrompt, messages)
	if err != nil {
		r.logger.Printf("LLM classification failed: %v", err)
		return ""
	}

	// Parse JSON response
	var resp struct {
		Intent string `json:"intent"`
	}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		// Try to extract from markdown code block
		result = strings.TrimSpace(result)
		result = strings.TrimPrefix(result, "```json")
		result = strings.TrimPrefix(result, "```")
		result = strings.TrimSuffix(result, "```")
		result = strings.TrimSpace(result)
		if err := json.Unmarshal([]byte(result), &resp); err != nil {
			r.logger.Printf("Failed to parse LLM response: %v (raw: %s)", err, result)
			return ""
		}
	}

	switch Intent(resp.Intent) {
	case IntentAnalyze, IntentCost, IntentOps, IntentStatus, IntentHelp:
		return Intent(resp.Intent)
	default:
		r.logger.Printf("Unknown LLM intent: %s", resp.Intent)
		return ""
	}
}

func (r *Router) classifyKeywords(message string) Intent {
	msg := strings.ToLower(message)

	// Strip code blocks before keyword matching to avoid false matches
	// from identifiers like "min_tls_version" triggering ops keywords.
	msgNoCode := codeBlockRe.ReplaceAllString(msg, "")

	// Score each intent by counting keyword matches.
	type intentScore struct {
		intent Intent
		score  int
	}

	categories := []struct {
		intent   Intent
		keywords []string
	}{
		{IntentAnalyze, []string{"scan", "audit", "review", "analyze", "security", "policy", "compliance", "vulnerability", "cis", "nist", "soc"}},
		{IntentCost, []string{"cost", "price", "pricing", "estimate", "budget", "expensive", "cheap", "spending", "money", "dollar"}},
		{IntentOps, []string{"deploy", "promote", "drift", "release", "rollback", "environment", "staging", "production", "notify", "notification", "alert"}},
		{IntentStatus, []string{"status", "health", "running", "uptime"}},
		{IntentHelp, []string{"help", "how to", "what can", "usage", "guide", "docs", "documentation", "capabilities"}},
	}

	best := intentScore{IntentHelp, 0}
	for _, cat := range categories {
		score := 0
		for _, w := range cat.keywords {
			if strings.Contains(msgNoCode, w) {
				score++
			}
		}
		if score > best.score {
			best = intentScore{cat.intent, score}
		}
	}

	// IaC language names can appear anywhere (including inside code blocks)
	// and boost the analyze intent.
	iacWords := []string{"terraform", "bicep"}
	for _, w := range iacWords {
		if strings.Contains(msg, w) {
			best = intentScore{IntentAnalyze, best.score + 2}
		}
	}

	if best.score > 0 {
		return best.intent
	}

	// Default: if message contains code blocks, analyze; otherwise help
	if strings.Contains(msg, "```") || strings.Contains(msg, "resource ") {
		return IntentAnalyze
	}

	return IntentHelp
}
