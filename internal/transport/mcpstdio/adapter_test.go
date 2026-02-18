package mcpstdio

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ghcp-iac/ghcp-iac-workflow/internal/host"
	"github.com/ghcp-iac/ghcp-iac-workflow/internal/protocol"
)

type stubAgent struct {
	id     string
	desc   string
	output string
}

func (s *stubAgent) ID() string { return s.id }
func (s *stubAgent) Metadata() protocol.AgentMetadata {
	return protocol.AgentMetadata{ID: s.id, Description: s.desc}
}
func (s *stubAgent) Capabilities() protocol.AgentCapabilities { return protocol.AgentCapabilities{} }
func (s *stubAgent) Handle(_ context.Context, _ protocol.AgentRequest, emit protocol.Emitter) error {
	emit.SendMessage(s.output)
	return nil
}

func setupAdapter(agents ...protocol.Agent) (*Adapter, *bytes.Buffer, *bytes.Buffer) {
	reg := host.NewRegistry()
	for _, a := range agents {
		reg.Register(a)
	}
	disp := host.NewDispatcher(reg)
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	adapter := NewAdapter(reg, disp, in, out)
	return adapter, in, out
}

func TestInitialize(t *testing.T) {
	adapter, in, out := setupAdapter()
	in.WriteString("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\"}\n")
	adapter.Run(context.Background())

	var resp JSONRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be a map")
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("unexpected protocol version: %v", result["protocolVersion"])
	}
}

func TestToolsList(t *testing.T) {
	adapter, in, out := setupAdapter(
		&stubAgent{id: "policy", desc: "Policy checks"},
		&stubAgent{id: "cost", desc: "Cost estimation"},
	)
	in.WriteString("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}\n")
	adapter.Run(context.Background())

	var resp JSONRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result map")
	}
	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatal("expected tools array")
	}
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

func TestToolsCall(t *testing.T) {
	adapter, in, out := setupAdapter(
		&stubAgent{id: "policy", output: "Policy findings here"},
	)
	req := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"policy","arguments":{"prompt":"analyze this"}}}`
	in.WriteString(req + "\n")
	adapter.Run(context.Background())

	var resp JSONRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result map")
	}
	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		t.Fatal("expected content array")
	}
	item := content[0].(map[string]interface{})
	text := item["text"].(string)
	if !strings.Contains(text, "Policy findings") {
		t.Errorf("unexpected output: %s", text)
	}
}

func TestToolsCall_UnknownAgent(t *testing.T) {
	adapter, in, out := setupAdapter()
	req := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"unknown","arguments":{"prompt":"test"}}}`
	in.WriteString(req + "\n")
	adapter.Run(context.Background())

	var resp JSONRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !strings.Contains(resp.Error.Message, "not found") {
		t.Errorf("expected not found error, got: %s", resp.Error.Message)
	}
}

func TestUnknownMethod(t *testing.T) {
	adapter, in, out := setupAdapter()
	in.WriteString("{\"jsonrpc\":\"2.0\",\"id\":5,\"method\":\"unknown/method\"}\n")
	adapter.Run(context.Background())

	var resp JSONRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected code -32601, got %d", resp.Error.Code)
	}
}

func TestMultipleRequests(t *testing.T) {
	adapter, in, out := setupAdapter(
		&stubAgent{id: "policy", output: "policy-result"},
	)
	in.WriteString("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\"}\n")
	in.WriteString("{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}\n")
	in.WriteString("{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"policy\",\"arguments\":{\"prompt\":\"test\"}}}\n")
	adapter.Run(context.Background())

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 response lines, got %d: %v", len(lines), lines)
	}
	var resp JSONRPCResponse
	json.Unmarshal([]byte(lines[2]), &resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error in tools/call: %s", resp.Error.Message)
	}
}

func TestStdioEmitter(t *testing.T) {
	e := &StdioEmitter{}
	e.SendMessage("hello ")
	e.SendMessage("world")
	e.SendError("oops")
	e.SendDone()
	got := e.Content()
	if !strings.Contains(got, "hello world") {
		t.Errorf("expected hello world in output, got: %s", got)
	}
	if !strings.Contains(got, "Error: oops") {
		t.Errorf("expected Error: oops in output, got: %s", got)
	}
}
