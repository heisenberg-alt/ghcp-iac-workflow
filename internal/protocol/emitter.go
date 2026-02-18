package protocol

// Emitter is the output interface that agents use to send data to the user.
// Transport layers (SSE, MCP stdio, etc.) provide concrete implementations.
type Emitter interface {
	SendMessage(content string)
	SendReferences(refs []Reference)
	SendConfirmation(conf Confirmation)
	SendError(msg string)
	SendDone()
}
