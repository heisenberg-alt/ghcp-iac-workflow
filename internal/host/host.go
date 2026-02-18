// Package host provides the agent registry and request dispatcher.
package host

import (
	"context"
	"fmt"
	"sync"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/parser"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// Registry stores registered agents and provides lookup.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]protocol.Agent
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{agents: make(map[string]protocol.Agent)}
}

// Register adds an agent to the registry.
func (r *Registry) Register(agent protocol.Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[agent.ID()] = agent
}

// Get returns an agent by ID.
func (r *Registry) Get(id string) (protocol.Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.agents[id]
	return a, ok
}

// List returns metadata for all registered agents.
func (r *Registry) List() []protocol.AgentMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	metas := make([]protocol.AgentMetadata, 0, len(r.agents))
	for _, a := range r.agents {
		metas = append(metas, a.Metadata())
	}
	return metas
}

// Dispatcher routes requests to agents via the registry.
type Dispatcher struct {
	registry  *Registry
	defaultID string
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(registry *Registry) *Dispatcher {
	return &Dispatcher{registry: registry}
}

// SetDefault sets the default agent ID used when no specific ID is provided.
func (d *Dispatcher) SetDefault(id string) {
	d.defaultID = id
}

// Dispatch looks up the agent by ID and calls its Handle method.
// If agentID is empty, the default agent is used.
func (d *Dispatcher) Dispatch(ctx context.Context, agentID string, req protocol.AgentRequest, emit protocol.Emitter) error {
	if agentID == "" {
		agentID = d.defaultID
	}
	if agentID == "" {
		return fmt.Errorf("no agent ID specified and no default configured")
	}
	agent, ok := d.registry.Get(agentID)
	if !ok {
		return fmt.Errorf("agent %q not found", agentID)
	}
	return agent.Handle(ctx, req, emit)
}

// ParseAndEnrich extracts IaC code from the request, detects the format,
// parses resources, and populates req.IaC.
func ParseAndEnrich(req *protocol.AgentRequest) {
	raw := req.Prompt
	if raw == "" {
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == "user" && req.Messages[i].Content != "" {
				raw = req.Messages[i].Content
				break
			}
		}
	}

	code := parser.ExtractCode(raw)
	if code == "" {
		return
	}

	iacType := parser.DetectIaCType(code)
	resources := parser.ParseResourcesOfType(code, iacType)

	protoResources := make([]protocol.Resource, len(resources))
	for i, r := range resources {
		protoResources[i] = protocol.Resource{
			Type:       r.Type,
			Name:       r.Name,
			Properties: r.Properties,
			Line:       r.Line,
			RawBlock:   r.RawBlock,
		}
	}

	format := protocol.FormatUnknown
	switch iacType {
	case "Terraform":
		format = protocol.FormatTerraform
	case "Bicep":
		format = protocol.FormatBicep
	}

	req.IaC = &protocol.IaCInput{
		Format:    format,
		RawCode:   code,
		Resources: protoResources,
	}
}
