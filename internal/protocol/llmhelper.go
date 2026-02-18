package protocol

import (
	"context"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/llm"
)

// StreamLLM is a shared helper that streams LLM output through an Emitter.
// It eliminates the duplicated streaming pattern across agents.
func StreamLLM(ctx context.Context, client *llm.Client, token, systemPrompt string, messages []llm.ChatMessage, emit Emitter) error {
	contentCh, errCh := client.Stream(ctx, token, systemPrompt, messages)
	for content := range contentCh {
		emit.SendMessage(content)
	}
	return <-errCh
}
