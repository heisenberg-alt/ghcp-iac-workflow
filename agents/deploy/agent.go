// Package deploy provides the Deployment Manager agent.
package deploy

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// EnvironmentState tracks the deployed version per environment.
type EnvironmentState struct {
	Version    string    `json:"version"`
	DeployedAt time.Time `json:"deployed_at"`
	Status     string    `json:"status"`
}

// Agent manages environment promotions and deployments.
type Agent struct {
	mu    sync.Mutex
	state map[string]*EnvironmentState
}

// New creates a new deploy Agent with default environment state.
func New() *Agent {
	return &Agent{
		state: map[string]*EnvironmentState{
			"dev":     {Version: "v1.0.0", DeployedAt: time.Now().Add(-48 * time.Hour), Status: "deployed"},
			"staging": {Version: "v0.9.0", DeployedAt: time.Now().Add(-72 * time.Hour), Status: "deployed"},
			"prod":    {Version: "v0.8.0", DeployedAt: time.Now().Add(-168 * time.Hour), Status: "deployed"},
		},
	}
}

func (a *Agent) ID() string { return "deploy" }

func (a *Agent) Metadata() protocol.AgentMetadata {
	return protocol.AgentMetadata{
		ID:          "deploy",
		Name:        "Deployment Manager",
		Description: "Manages environment promotions (dev -> staging -> prod)",
		Version:     "1.0.0",
	}
}

func (a *Agent) Capabilities() protocol.AgentCapabilities {
	return protocol.AgentCapabilities{
		NeedsIaCInput: false,
	}
}

// Handle processes deployment/promotion requests based on prompt keywords.
func (a *Agent) Handle(_ context.Context, req protocol.AgentRequest, emit protocol.Emitter) error {
	msg := strings.ToLower(protocol.PromptText(req))

	if protocol.MatchesAny(msg, "status", "environments", "versions") {
		a.handleStatus(emit)
		return nil
	}

	a.handleDeploy(msg, emit)
	return nil
}

func (a *Agent) handleDeploy(msg string, emit protocol.Emitter) {
	emit.SendMessage("## Deployment Manager\n\n")

	target := "dev"
	if protocol.MatchesAny(msg, "staging", "stage", "test") {
		target = "staging"
	} else if protocol.MatchesAny(msg, "prod", "production") {
		target = "prod"
	}

	source := "dev"
	if target == "prod" {
		source = "staging"
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	sourceState := a.state[source]

	emit.SendMessage("### Environment Status\n\n")
	emit.SendMessage("| Environment | Version | Status |\n")
	emit.SendMessage("|-------------|---------|--------|\n")
	for _, env := range []string{"dev", "staging", "prod"} {
		s := a.state[env]
		emit.SendMessage(fmt.Sprintf("| %s | %s | %s |\n", env, s.Version, s.Status))
	}
	emit.SendMessage("\n")

	if target == "prod" {
		emit.SendMessage("**Production deployment requires manual approval.**\n\n")
		emit.SendMessage(fmt.Sprintf("Promotion: `%s` (%s) -> `%s`\n\n", source, sourceState.Version, target))
		emit.SendMessage("Use the GitHub Actions workflow `deploy-prod.yml` with approval gate.\n")
		return
	}

	emit.SendMessage(fmt.Sprintf("Promoting **%s** -> **%s** (version %s)\n\n", source, target, sourceState.Version))
	a.state[target] = &EnvironmentState{
		Version:    sourceState.Version,
		DeployedAt: time.Now(),
		Status:     "deployed",
	}
	emit.SendMessage(fmt.Sprintf("Successfully promoted to **%s** (version %s)\n", target, sourceState.Version))
}

func (a *Agent) handleStatus(emit protocol.Emitter) {
	emit.SendMessage("## Environment Status\n\n")
	emit.SendMessage("| Environment | Version | Deployed | Status |\n")
	emit.SendMessage("|-------------|---------|----------|--------|\n")

	a.mu.Lock()
	defer a.mu.Unlock()

	for _, env := range []string{"dev", "staging", "prod"} {
		s := a.state[env]
		emit.SendMessage(fmt.Sprintf("| %s | %s | %s | %s |\n",
			env, s.Version, s.DeployedAt.Format("2006-01-02 15:04"), s.Status))
	}
}
