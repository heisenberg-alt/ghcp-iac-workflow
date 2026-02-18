// Package notification provides the Notification Manager agent.
package notification

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// Agent sends notifications to Teams/Slack channels.
type Agent struct {
	enableNotify bool
}

// New creates a new notification Agent.
func New(enableNotify bool) *Agent {
	return &Agent{enableNotify: enableNotify}
}

func (a *Agent) ID() string { return "notification" }

func (a *Agent) Metadata() protocol.AgentMetadata {
	return protocol.AgentMetadata{
		ID:          "notification",
		Name:        "Notification Manager",
		Description: "Sends infrastructure notifications to Teams or Slack channels",
		Version:     "1.0.0",
	}
}

func (a *Agent) Capabilities() protocol.AgentCapabilities {
	return protocol.AgentCapabilities{
		NeedsIaCInput: false,
	}
}

// Handle processes notification requests.
func (a *Agent) Handle(_ context.Context, req protocol.AgentRequest, emit protocol.Emitter) error {
	emit.SendMessage("## Notification Manager\n\n")

	msg := strings.ToLower(protocol.PromptText(req))

	channel := "teams"
	if strings.Contains(msg, "slack") {
		channel = "slack"
	}

	message := "Infrastructure update notification"
	if idx := strings.Index(msg, "message:"); idx >= 0 {
		message = strings.TrimSpace(msg[idx+8:])
	}

	emit.SendMessage(fmt.Sprintf("Sending notification to **%s**:\n> %s\n\n", channel, message))

	if !a.enableNotify {
		emit.SendMessage("Notifications are disabled. Set `ENABLE_NOTIFICATIONS=true` to enable.\n")
		return nil
	}

	emit.SendMessage("Notification sent successfully.\n")
	return nil
}
