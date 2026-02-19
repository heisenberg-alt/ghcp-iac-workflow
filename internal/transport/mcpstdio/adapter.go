// Package mcpstdio provides an MCP (Model Context Protocol) stdio transport
// that maps JSON-RPC messages on stdin/stdout to the agent-host pipeline.
package mcpstdio

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/host"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

// JSONRPCRequest represents an incoming JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolCallParams represents parameters for a tools/call request.
type ToolCallParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// ToolInfo represents an MCP tool listing entry.
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Adapter bridges MCP stdio to the agent-host pipeline.
type Adapter struct {
	registry   *host.Registry
	dispatcher *host.Dispatcher
	reader     io.Reader
	writer     io.Writer
}

// NewAdapter creates a new MCP stdio Adapter.
func NewAdapter(registry *host.Registry, dispatcher *host.Dispatcher, r io.Reader, w io.Writer) *Adapter {
	return &Adapter{
		registry:   registry,
		dispatcher: dispatcher,
		reader:     r,
		writer:     w,
	}
}

// Run reads JSON-RPC requests from the reader and writes responses to the writer.
// It blocks until the reader is exhausted or the context is cancelled.
func (a *Adapter) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(a.reader)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			a.writeError(nil, -32700, "Parse error")
			continue
		}

		a.handleRequest(ctx, &req)
	}
	return scanner.Err()
}

func (a *Adapter) handleRequest(ctx context.Context, req *JSONRPCRequest) {
	switch req.Method {
	case "tools/list":
		a.handleToolsList(req)
	case "tools/call":
		a.handleToolsCall(ctx, req)
	case "initialize":
		a.handleInitialize(req)
	default:
		a.writeError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

func (a *Adapter) handleInitialize(req *JSONRPCRequest) {
	a.writeResult(req.ID, map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "ghcp-iac-workflow",
			"version": "1.0.0",
		},
	})
}

func (a *Adapter) handleToolsList(req *JSONRPCRequest) {
	metas := a.registry.List()
	tools := make([]ToolInfo, 0, len(metas))
	for _, m := range metas {
		tools = append(tools, ToolInfo{
			Name:        m.ID,
			Description: m.Description,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "The user prompt or IaC code to analyze",
					},
				},
				"required": []string{"prompt"},
			},
		})
	}
	a.writeResult(req.ID, map[string]interface{}{"tools": tools})
}

func (a *Adapter) handleToolsCall(ctx context.Context, req *JSONRPCRequest) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		a.writeError(req.ID, -32602, "Invalid params")
		return
	}

	prompt := params.Arguments["prompt"]

	agentReq := protocol.AgentRequest{
		Prompt: prompt,
		Messages: []protocol.Message{
			{Role: "user", Content: prompt},
		},
	}
	host.ParseAndEnrich(&agentReq)

	emit := &StdioEmitter{}
	err := a.dispatcher.Dispatch(ctx, params.Name, agentReq, emit)
	if err != nil {
		a.writeError(req.ID, -32000, err.Error())
		return
	}

	a.writeResult(req.ID, map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": emit.Content(),
			},
		},
	})
}

func (a *Adapter) writeResult(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("MCP marshal error for result: %v", err)
		return
	}
	fmt.Fprintf(a.writer, "%s\n", data)
}

func (a *Adapter) writeError(id interface{}, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("MCP marshal error for error response: %v", err)
		return
	}
	fmt.Fprintf(a.writer, "%s\n", data)
}
