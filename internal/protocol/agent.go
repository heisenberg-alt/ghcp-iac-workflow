package protocol

import "context"

// Agent is the interface that all domain agents implement.
type Agent interface {
	ID() string
	Metadata() AgentMetadata
	Capabilities() AgentCapabilities
	Handle(ctx context.Context, req AgentRequest, emit Emitter) error
}
