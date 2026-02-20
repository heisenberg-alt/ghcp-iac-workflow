# Agent Modularization Overhaul Plan

## Framing note (current state, stated clearly)

What exists today is a strong **rule-based IaC governance automation platform** with an **agent-style interface**.
It delivers practical value (policy, compliance, security, cost, drift, and orchestration checks), but it is mostly deterministic and not yet model-reasoning driven.
This overhaul plan is intentionally aimed at closing that gap by modularizing the architecture, simplifying extension points, and preparing the system for genuinely smarter agent behavior.

## 1) Goals

This refactor targets the following outcomes:

- Increase testability through reusable modules with small, explicit interfaces.
- Reduce duplicated code across agents.
- Make adding new agents straightforward and low-risk.
- Make the runtime **Terraform + Bicep first-class**, while converging both formats into a shared internal model.
- Use **no external IaC parsing modules**; keep parsing/normalization internal and reuse existing parsing code.
- Run everything in a single process with two host modes:
  - HTTP web port (current style)
  - MCP stdio transport
- Follow **Grokking Simplicity** (separate actions/calculations/data) and **A Philosophy of Software Design** (deep modules, information hiding, simple interfaces).
- End with an **AgentHost** that hosts all agents, while `agents/` contains one module per agent.
- Add easy-to-run tests (`go test`) with extendable scenario fixtures.

---

## 2) Current State Analysis

### Observed architecture

- 10 agents currently run as independent HTTP servers on separate ports.
- Each agent is a large `main.go` with mixed responsibilities (transport + parsing + domain rules + formatting).
- Orchestrator calls other agents over HTTP, even when all services run on the same machine.

### Key duplication and complexity hotspots

- Repeated patterns in nearly every agent:
  - config loading
  - Copilot request/response structs
  - `/health` and `/agent` handlers
  - SSE writer implementation
  - code extraction and IaC type detection logic
- Approximate size concentration in agent `main.go` files: **~7,870 lines total**.
- No current `_test.go` files in this repository.

### Structural friction

- Multiple independent Go modules across agents and tools.
- Inconsistent module naming (`copilot-iac`, `copilot-iac-lab`, `your-org`).
- Current operational model depends on process-per-agent startup scripts and port wiring.
- Terraform parsing is currently duplicated and mostly regex-based per agent, with repeated IaC type detection logic.

---

## 3) Target Architecture

### 3.1 Single-process host

Create a single binary (`agent-host`) that loads all agent modules in-process.

Core responsibilities:

- Register agents
- Route requests by agent ID
- Serve health/status
- Expose one or both transports:
  - HTTP (`/agent/{id}`, `/health`, `/agents`)
  - MCP stdio (tool-style routing to the same in-process agents)

### 3.2 Agent interface

Define a small shared interface:

```go
type Agent interface {
    ID() string
    Metadata() Metadata
    Capabilities() AgentCapabilities
    Handle(ctx context.Context, req AgentRequest, emit Emitter) error
}

type AgentCapabilities struct {
    Formats           []SourceFormat
    NeedsIaCInput     bool
    NeedsRawCode      bool
    NeedsFileContents bool
}
```

- `AgentRequest` and `Emitter` become shared protocol abstractions.
- Each agent module focuses on domain logic; host and transport stay outside.
- `Capabilities()` lets AgentHost route/validate inputs before invoking handlers.

### 3.3 IaC request model (Terraform + Bicep, no external modules)

Because requests are Terraform/Bicep, agents should receive normalized parsed structures instead of each agent reparsing raw text.

Decision:

- The **primary agent input shape** is `[]Resource`.
- Keep a request envelope that supports both IaC-analysis agents and command-style agents.
- Include source format metadata so agents can keep format-specific behavior when needed.
- Include optional raw/file content because some current agents still perform text-level checks.
- Build and own parsing/normalization internally in `internal/terraform` by extracting/reusing existing logic (`detectIaCType`, `extractCode`, `parseResources`, `bicepToTerraformType`).

Proposed shared types:

```go
type SourceFormat string

const (
    FormatTerraform SourceFormat = "terraform"
    FormatBicep     SourceFormat = "bicep"
    FormatUnknown   SourceFormat = "unknown"
)

type Resource struct {
    Type       string
    Name       string
    Properties map[string]interface{}
    Line       int
    File       string
}

type SourceFile struct {
    Path    string
    Content string
}

type IaCInput struct {
    Format    SourceFormat
    RawCode   string
    Files     []SourceFile
    Resources []Resource
}

type AgentRequest struct {
    Prompt     string
    Messages   []Message
    References []Reference
    IaC        *IaCInput
    Metadata   map[string]string
}
```

Benefits:

- One parse step per request (host-level), reused by all agents.
- Single canonical `[]Resource` contract for structured analysis.
- Works for both Terraform and Bicep while keeping one agent-side interface.
- No dependency lock-in to external Terraform parsing libraries.
- Supports all current behavior patterns:
  - policy/security/compliance/cost/drift/impact use `IaC.Resources`
  - security/module-registry can still use `IaC.RawCode` / `IaC.Files` for text-level checks
  - deploy/notification can work with `Prompt` even when `IaC == nil`
- Strongly typed interfaces for unit tests.
- Easier to extend with plan metadata later without changing every agent.

### 3.4 Folder layout (target)

```text
cmd/
  agent-host/
    main.go

internal/
  host/                 # registry, router, lifecycle
  protocol/             # shared request/response/event types
  terraform/            # internal terraform parser + normalization to []Resource
  transport/
    http/               # web server adapters
    mcpstdio/           # MCP stdio adapter
  testkit/              # scenario harness and fixtures

agents/
  policy/
  security/
  cost/
  compliance/
  drift/
  module/
  impact/
  deploy/
  notification/
  orchestrator/
```

---

## 4) Design Rules (from the two books)

### Grokking Simplicity mapping

- **Actions**: network/file/env access, HTTP/MCP transport.
- **Calculations**: Terraform parsing transforms, rule evaluation, risk scoring, cost math.
- **Data**: typed Terraform module model, request/response DTOs, findings, policies, scenario fixtures.

Rule: move calculations into pure functions with deterministic tests.

### A Philosophy of Software Design mapping

- Prefer **deep modules** for host/transport/protocol with narrow APIs.
- Hide transport and streaming details behind `Emitter`.
- Avoid temporal coupling by explicit initialization and registration steps.
- Keep per-agent APIs obvious and minimal (`ID`, `Metadata`, `Handle`).

---

## 5) Refactor Strategy (incremental, low-risk)

### Phase 0 - Baseline and characterization

- Capture current behavior with characterization tests and sample inputs/outputs.
- Define compatibility baseline for:
  - event stream shape
  - health payloads
  - core command behavior used by `gh-iac`

### Phase 1 - Shared foundation

- Introduce shared protocol types and `Emitter`.
- Build `internal/host` with in-process registry/router.
- Add `internal/terraform` parser and canonical `[]Resource` normalization pipeline for Terraform and Bicep.
- Start by extracting existing per-agent parser helpers into shared package(s), then simplify.
- Add HTTP transport adapter while preserving current endpoint semantics.

### Phase 2 - First vertical migration (pilot agent)

- Migrate one agent first (recommended: policy).
- Extract pure domain package + small adapter package that consumes typed Terraform input.
- Validate parity with characterization tests.

### Phase 3 - Migrate remaining agents

- Move each agent into `agents/<name>/` module package.
- Delete duplicated transport/config/SSE logic per agent.
- Delete duplicated per-agent parsing branches by reusing shared Terraform/Bicep parser package.

### Phase 4 - Orchestrator in-process

- Refactor orchestrator to call registered agents directly (no localhost HTTP fan-out).
- Keep workflow semantics, reduce latency and operational complexity.

### Phase 5 - MCP stdio transport

- Add MCP stdio adapter to the same host.
- Ensure MCP and HTTP paths share identical execution pipeline and output mapping.

### Phase 6 - Cleanup and compatibility wrappers

- Replace process-per-agent startup with a single host entrypoint.
- Keep temporary wrappers only if needed for transition.
- Update docs/scripts/CLI expectations to single-host mode.

---

## 6) Testing Plan (`go test` first-class)

### Test layers

1. **Unit tests (pure logic)**  
   Rule evaluation, Terraform parser transforms, scoring, cost calculations.

2. **Agent contract tests**  
   Validate each `Agent.Handle` against shared request/response contract.

3. **Transport tests**  
   - HTTP handler tests via `httptest`
   - MCP stdio adapter tests with in-memory stdin/stdout harness

4. **Host integration tests**  
   Multi-agent routing and orchestrator workflows in one process.

### Initial scenario fixtures (easy to extend)

- `minimal-secure-storage.tf` -> no critical issues.
- `insecure-storage.tf` -> policy/security/compliance findings.
- `secure-storage.bicep` / `insecure-storage.bicep` -> Bicep parity for policy/security/compliance.
- `aks-cluster.tf` sample -> cost + impact path.
- `mixed-module/main.tf` + `variables.tf` -> parser handles module-level variable/resource relationships.
- `no-iac-input.txt` -> help/usage response behavior.
- `orchestrator-full-check.json` -> multi-agent workflow scenario.

### CI/dev ergonomics

- One command: `go test ./...`
- Scenario fixtures under `internal/testkit/scenarios/`.
- Add table-driven tests to grow coverage without rewriting harness code.

---

## 7) Compatibility and rollout guardrails

- Preserve `gh-iac` command surface during migration.
- Preserve core HTTP semantics (`/health`, `/agent`) via host adapter.
- Migrate agent-by-agent behind feature toggles or registration switches.
- Keep old binaries optional only during transition; remove after parity is proven.

---

## 8) Definition of Done

- Single `agent-host` process runs all agents in-process.
- Host supports both HTTP and MCP stdio.
- Agents are modular packages under `agents/`.
- Duplicated SSE/request/health plumbing removed from individual agents.
- Agents consume one shared request contract: `Prompt` + optional `IaC{Format, Resources, RawCode, Files}`.
- `go test ./...` runs a meaningful unit + integration + scenario suite.
- Adding a new agent requires:
  1. implementing the interface,
  2. registering it,
  3. adding scenario tests.
