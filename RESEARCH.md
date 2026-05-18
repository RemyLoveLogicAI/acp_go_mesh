# Agent Communication Protocol Research
## Cutting-Edge Patterns for Go Agent Mesh

**Researched:** 2026-05-18  
**Domain:** Agent-to-agent communication protocols, multi-agent mesh topology, Go implementation  
**Confidence:** HIGH (cross-verified across official specs, GitHub repos, and production SDKs)

---

## Executive Summary

Your current system is a solid, working proof-of-concept but it is roughly 18 months behind the
production state-of-the-art. The field has converged on five complementary layers that, taken
together, define what a modern agent mesh looks like in 2025:

1. **A2A (Google/Linux Foundation)** — the cross-agent coordination wire protocol.  
   Go SDK exists: `github.com/a2aproject/a2a-go/v2`
2. **MCP (Anthropic)** — the tool-and-context injection layer for LLM-facing agents.  
   Go SDK exists: `github.com/mark3labs/mcp-go`
3. **AG-UI** — the structured streaming protocol between agents and frontends.  
   Go SDK exists: `github.com/ag-ui-protocol/ag-ui/sdks/community/go`
4. **OpenAI Agents SDK / Swarm primitives** — the handoff + guardrails mental model (Python-first,
   but the patterns are language-agnostic and directly translatable to Go).
5. **IBM ACP** — now merged into A2A; its strongest surviving contribution is the
   `await`/multi-turn session pattern and the async `run_id` / `session_id` lifecycle.

**The single most impactful next step:** Replace the flat `ACPMessage{method, sender, target,
params}` struct with an A2A-compliant Task envelope (typed states, artifact streaming, Agent Card
discovery) while keeping UDS as your local transport. Everything else flows from that upgrade.

---

## What Your Current System Does Well

| Strength | Detail |
|----------|--------|
| UDS transport | Zero-copy, OS-enforced process isolation, no network stack overhead |
| JSON-RPC 2.0 base | Correct wire format choice — A2A and MCP both use it |
| Approval gate | Human-in-the-loop before shell execution — production-correct |
| PTY streaming to WebSocket | Real-time terminal output in UI — AG-UI equivalent |
| vis-network topology graph | Live mesh visualization — rare even in production systems |

---

## What the State of the Art Has That You Are Missing

| Gap | Current State | Standard Pattern | Priority |
|-----|---------------|------------------|----------|
| Task lifecycle states | No formal states; messages fire-and-forget | A2A: submitted → working → input-required → completed/failed/canceled | CRITICAL |
| Agent Card / capability discovery | Flat string array in `register` message | Structured JSON at `/.well-known/agent.json` with skills, securitySchemes, inputModes | HIGH |
| Streaming partial results | PTY stdout piped raw; no protocol-level streaming | A2A `TaskArtifactUpdateEvent` / AG-UI `TextMessageContent` delta events | HIGH |
| Multi-turn sessions | No session concept; each `user_intent` is stateless | ACP `session_id` + A2A `input-required` state for agent-initiated clarification | HIGH |
| Server-initiated LLM sampling | Manager hardcodes worker target; no LLM reasoning loop | MCP `sampling/createMessage` — server asks client to run an LLM call | HIGH |
| Capability-based routing | Manager hardcodes `Target: "worker"` | Capability registry; route by required skill, not agent name | HIGH |
| Structured handoffs | No context forwarding between agents | OpenAI handoff pattern: full message history + HandoffInputData passed on transfer | MEDIUM |
| Observability / tracing | PTY stdout only | OpenTelemetry spans on every message, OTLP export, Arize Phoenix or Jaeger | MEDIUM |
| Auth / identity | No auth between agents | A2A: OAuth2 client credentials or mTLS for service-to-service | MEDIUM |
| Progress notifications | No intermediate updates | MCP `notifications/progress` with `progressToken`; A2A `TaskStatusUpdateEvent` | MEDIUM |
| Cancellation | No cancel path | JSON-RPC `cancelled` notification; A2A `tasks/cancel` endpoint | LOW |
| Gossip / emergent routing | Hub-and-spoke only; harness is SPOF | Gossip substrate for decentralized capability propagation (O(log N) convergence) | LOW |

---

## Protocol Deep-Dives

---

### 1. IBM ACP — Agent Communication Protocol

**Status:** Merged with A2A under Linux Foundation (May 2025). ACP is winding down active
development. Its most valuable contributions have been absorbed into A2A. You should build toward
A2A, not ACP. [VERIFIED: WebSearch + GitHub i-am-bee/acp]

**What ACP contributed that A2A now has:**

**REST API surface (the `runs` pattern):**
```
POST /runs           — synchronous execution, returns run object
POST /runs/stream    — streaming execution (SSE)
GET  /runs/{run_id}  — poll async task status
GET  /agents         — list registered agents
```

**Run object schema:**
```json
{
  "run_id":    "run_abc123",
  "agent_name": "worker",
  "session_id": "sess_xyz",
  "status":    "completed",
  "output": [
    {
      "role": "agent/worker",
      "parts": [
        {
          "content":          "ls output here",
          "content_type":     "text/plain",
          "content_encoding": "plain"
        }
      ]
    }
  ],
  "error": null
}
```

**Multi-turn with `session_id`:**
Sessions maintain state across multiple `POST /runs` calls. The client passes the same `session_id`
to continue a conversation thread. The `await` state pauses an agent mid-task and waits for the
client to resume with additional input — equivalent to A2A's `input-required` state.

**ACP `await` → A2A `input-required` mapping:**
```
ACP:  run.status == "await"        → send POST /runs with same session_id + new input
A2A:  task.status == "input-required" → send tasks/sendSubscribe with new message
```

**Streaming format (SSE over HTTP):**
```
data: {"type": "message.delta", "part": {"content": "partial output...", "content_type": "text/plain"}}
data: {"type": "message.completed", "run_id": "run_abc"}
```

**Key ACP concept to carry forward:** `session_id` as a first-class routing key in your harness.
Your harness should maintain a `sessions` map alongside `agents`, enabling multi-turn conversation
state that survives individual message exchanges.

---

### 2. Anthropic MCP — Model Context Protocol

**Spec version:** 2025-11-25 (latest)  
**Source:** [modelcontextprotocol.io/specification/2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25)  
[VERIFIED: Official spec site]

MCP is not an agent-to-agent protocol. It is a **tool-and-context injection protocol** — the
standard way for an LLM-backed agent to discover and call tools, access files/resources, and
request additional LLM inference. Your agents should speak MCP to expose their capabilities to
LLM orchestrators.

**Go SDK:** `github.com/mark3labs/mcp-go` — implements spec 2025-11-25 with backward compatibility
to 2024-11-05. [VERIFIED: pkg.go.dev + GitHub releases]

#### Capability Negotiation (initialization handshake)

```json
// Client → Server: initialize
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-11-25",
    "capabilities": {
      "sampling": {},
      "roots": { "listChanged": true }
    },
    "clientInfo": { "name": "harness", "version": "1.0.0" }
  }
}

// Server → Client: initialize response
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-11-25",
    "capabilities": {
      "tools": { "listChanged": true },
      "logging": {}
    },
    "serverInfo": { "name": "worker-agent", "version": "1.0.0" }
  }
}
```

#### Tools — the core primitive

```json
// tools/list response
{
  "tools": [
    {
      "name": "shell_exec",
      "description": "Execute a shell command and return stdout/stderr",
      "inputSchema": {
        "type": "object",
        "properties": {
          "command": { "type": "string", "description": "Shell command to run" },
          "timeout_seconds": { "type": "number", "default": 30 }
        },
        "required": ["command"]
      },
      "annotations": {
        "readOnlyHint":  false,
        "destructiveHint": true,
        "requiresHumanApproval": true
      }
    }
  ]
}
```

**Tool annotations (new in 2025-11-25)** are structured metadata the host uses to decide whether to
auto-approve or gate tool calls. Your existing approval gate maps directly to
`requiresHumanApproval: true`.

#### Sampling — server-initiated LLM requests (the biggest missed feature)

Sampling lets your **worker agent ask the LLM a question** without the LLM being in the loop at the
transport level. The server (worker) sends a `sampling/createMessage` to the client (harness), the
harness calls the actual LLM, and returns the response.

```json
// Worker (Server) → Harness (Client): sampling/createMessage
{
  "jsonrpc": "2.0",
  "id": 42,
  "method": "sampling/createMessage",
  "params": {
    "messages": [
      {
        "role": "user",
        "content": {
          "type": "text",
          "text": "The command 'rm -rf /tmp/work' failed with permission denied. How should I retry?"
        }
      }
    ],
    "maxTokens": 512,
    "tools": [
      {
        "name": "shell_exec",
        "description": "...",
        "inputSchema": { ... }
      }
    ],
    "toolChoice": "auto"
  }
}

// Harness (Client) → Worker (Server): response
{
  "jsonrpc": "2.0",
  "id": 42,
  "result": {
    "role": "assistant",
    "content": {
      "type": "text",
      "text": "Try running with sudo: sudo rm -rf /tmp/work"
    }
  }
}
```

**New in 2025-11-25:** `tools` and `toolChoice` fields in `sampling/createMessage`. The server can
now initiate multi-step agentic loops entirely through the sampling channel — the worker can run
its own reasoning loop using the client's LLM tokens.

#### Roots — filesystem scoping

```json
// Server requests what directories it may access
{ "jsonrpc": "2.0", "id": 1, "method": "roots/list" }

// Client responds with allowed paths
{
  "result": {
    "roots": [
      { "uri": "file:///home/user/projects/mesh", "name": "Mesh Project" }
    ]
  }
}
```

#### Progress notifications

```json
{
  "jsonrpc": "2.0",
  "method": "notifications/progress",
  "params": {
    "progressToken": "task-abc-123",
    "progress": 60,
    "total": 100,
    "message": "Compiling step 3 of 5..."
  }
}
```

#### Cancellation

```json
{
  "jsonrpc": "2.0",
  "method": "notifications/cancelled",
  "params": {
    "requestId": "long-running-op-id",
    "reason": "User cancelled"
  }
}
```

#### Go implementation with mark3labs/mcp-go

```go
import (
    "context"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

func NewWorkerMCPServer() *server.MCPServer {
    s := server.NewMCPServer(
        "worker-agent",
        "1.0.0",
        server.WithToolCapabilities(true),
        server.WithRecovery(),
    )

    shellTool := mcp.NewTool("shell_exec",
        mcp.WithDescription("Execute a shell command"),
        mcp.WithString("command",
            mcp.Required(),
            mcp.Description("Shell command to run"),
        ),
    )

    s.AddTool(shellTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        command, _ := req.RequireString("command")
        // ... execute with approval gate
        return mcp.NewToolResultText(output), nil
    })

    return s
}

// Expose over UDS instead of stdio:
// server.ServeStdio(s) → wrap with net.Listen("unix", socketPath)
```

**Transport note for your system:** `mark3labs/mcp-go` uses stdio by default. For UDS, you wrap
the server's `ServeStdio` pattern by passing the UDS connection as the reader/writer. The JSON-RPC
framing is identical.

---

### 3. Google A2A — Agent2Agent Protocol

**Status:** Open standard, donated to Linux Foundation June 2025, 150+ organizations  
**Spec:** [a2a-protocol.org/latest/specification](https://a2a-protocol.org/latest/specification/)  
**Go SDK:** `github.com/a2aproject/a2a-go/v2` — requires Go 1.24.4+  
[VERIFIED: Official spec site + GitHub repo]

A2A is the protocol your harness should implement as its primary routing layer. It defines exactly
the concepts your current system needs: typed tasks, capability-based discovery, streaming
artifacts, and enterprise auth.

#### Agent Card — the discovery contract

Serve at `/.well-known/agent.json` on each agent's HTTP endpoint (or embed in registration message
for UDS-local agents):

```json
{
  "name": "worker-agent",
  "description": "Shell execution and file reading agent",
  "version": "1.0.0",
  "url": "http://127.0.0.1:9001",
  "provider": {
    "organization": "acp-mesh",
    "url": "https://github.com/you/acp_go_mesh"
  },
  "capabilities": {
    "streaming":          true,
    "pushNotifications":  false,
    "stateTransitionHistory": true
  },
  "defaultInputModes":  ["text/plain", "application/json"],
  "defaultOutputModes": ["text/plain", "application/json"],
  "securitySchemes": {
    "internalToken": {
      "type": "http",
      "scheme": "bearer"
    }
  },
  "security": [{ "internalToken": [] }],
  "skills": [
    {
      "id":          "shell_exec",
      "name":        "Shell Command Execution",
      "description": "Execute arbitrary shell commands with human approval gate",
      "tags":        ["shell", "execution", "system"],
      "inputModes":  ["text/plain"],
      "outputModes": ["text/plain"],
      "examples":    ["ls -la /tmp", "cat /etc/hosts"]
    },
    {
      "id":          "file_read",
      "name":        "File Reading",
      "description": "Read file contents from allowed paths",
      "tags":        ["file", "read"],
      "inputModes":  ["text/plain"],
      "outputModes": ["text/plain"]
    }
  ]
}
```

#### Task lifecycle — typed state machine

```
submitted → working → input-required → working (resumed)
                   ↘ completed
                   ↘ failed
                   ↘ canceled
                   ↘ rejected
```

**Task envelope (replaces your flat `ACPMessage`):**

```json
{
  "jsonrpc": "2.0",
  "id": "req-001",
  "method": "message/send",
  "params": {
    "message": {
      "messageId": "msg-abc123",
      "role": "user",
      "parts": [
        {
          "kind": "text",
          "text": "List files in /tmp"
        }
      ],
      "metadata": {
        "sessionId": "sess-xyz"
      }
    }
  }
}
```

**Task response:**

```json
{
  "jsonrpc": "2.0",
  "id": "req-001",
  "result": {
    "id":     "task-001",
    "status": {
      "state":    "working",
      "message":  { "role": "agent", "parts": [{"kind": "text", "text": "Executing..."}] },
      "timestamp": "2026-05-18T01:48:00Z"
    },
    "artifacts": []
  }
}
```

#### Streaming — SSE TaskArtifactUpdateEvent

```
POST /tasks/sendSubscribe  → starts task and returns SSE stream

data: {"task": {"id": "task-001", "status": {"state": "working"}}}

data: {"artifact": {"taskId": "task-001", "index": 0, "append": false,
       "artifact": {"artifactId": "out-1", "parts": [{"kind": "text", "text": "file1.txt\n"}]}}}

data: {"artifact": {"taskId": "task-001", "index": 1, "append": true,
       "artifact": {"artifactId": "out-1", "parts": [{"kind": "text", "text": "file2.txt\n"}], "lastChunk": true}}}

data: {"taskStatus": {"taskId": "task-001", "status": {"state": "completed"}, "final": true}}
```

**`append: true` + `lastChunk: true`** is the chunked artifact pattern — agents emit incremental
output without buffering the full result.

#### Auth — OAuth2 client credentials (service-to-service)

```
Agent A requests token from auth server:
  POST /oauth/token
  grant_type=client_credentials&client_id=manager&client_secret=...

Agent A calls Agent B with Bearer token:
  Authorization: Bearer <token>
```

For your local mesh, a shared HMAC secret or short-lived JWT signed by the harness is sufficient.
Full OAuth2 can be added later when agents cross process/host boundaries.

#### Go SDK usage

```go
import "github.com/a2aproject/a2a-go/v2"

// Serve an agent
requestHandler := a2asrv.NewHandler(myExecutor, options...)
jsonrpcHandler := a2asrv.NewJSONRPCHandler(requestHandler)
// Mount jsonrpcHandler on your UDS or HTTP listener

// Consume an agent  
agentCard, _ := agentcard.DefaultResolver.Resolve(ctx)
client := a2aclient.NewClient(agentCard)
response, _ := client.SendMessage(ctx, &a2a.SendMessageRequest{...})
```

---

### 4. Agent Mesh Topology Patterns

**Sources:** [gurusup.com/agent-orchestration-patterns](https://gurusup.com/blog/agent-orchestration-patterns),
[arxiv.org/html/2508.01531v1](https://arxiv.org/html/2508.01531v1), AWS Strands Agents blog
[VERIFIED: Multiple independent sources]

#### The five canonical topologies

| Pattern | Control | Failure Mode | Best For | Your Current State |
|---------|---------|--------------|----------|--------------------|
| **Hub-and-Spoke** (Orchestrator-Worker) | Centralized | Hub is SPOF | Simple delegation, small N | This is what you have |
| **Mesh** | Peer-to-peer | N(N-1)/2 connections at scale | 3-8 tightly coupled specialists | Next step |
| **Swarm** | Emergent | No single point of failure | Large N, homogeneous workers | Future |
| **Hierarchical** | Tree delegation | Subtree failure | Enterprise org charts | Future |
| **Pipeline** | Sequential | Stage failure | ETL, multi-step transforms | Use case specific |

#### Mesh topology — the right next step for your system

Mesh means agents can send messages **directly to each other** through the harness, not only through
the manager. The harness becomes a **routing fabric** rather than a bottleneck orchestrator.

```
Current:  UI → Manager → Worker → Manager → UI  (hub-and-spoke, manager = bottleneck)

Target:   UI ─────────────────────────────────→ any agent by capability
               ↗ Manager → Worker-A ↘
                         → Worker-B ↗ → Manager (result aggregation)
                         → Worker-C ↗
```

**Capability-based routing replaces hardcoded targets:**

```go
// Current (hardcoded)
Target: "worker"

// Mesh pattern (capability-based)
type RoutingRequest struct {
    RequiredCapabilities []string `json:"required_capabilities"`
    PreferredAgent       string   `json:"preferred_agent,omitempty"`
    LoadBalance          bool     `json:"load_balance"`
}

// Harness selects best agent:
func (h *HarnessState) RouteByCapability(required []string) (string, net.Conn, error) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    for agentID, caps := range h.capabilities {
        if hasAll(caps, required) {
            return agentID, h.agents[agentID], nil
        }
    }
    return "", nil, ErrNoCapableAgent
}
```

#### Gossip protocol — the emergent coordination layer

[CITED: arxiv.org/html/2508.01531v1]

Gossip enables decentralized capability propagation without a central registry. Key properties:
- **Convergence:** Fan-out=3, 25,000-agent system converges in 15 rounds (~seconds)
- **Scale:** O(log N) — scales to arbitrarily large agent pools
- **Resilience:** No SPOF — any agent can serve as a relay

**Gossip message format for capability propagation:**

```json
{
  "type":      "capability_gossip",
  "origin":    "worker-3",
  "ttl":       5,
  "version":   1716992880,
  "payload": {
    "agent_id":       "worker-3",
    "capabilities":   ["shell_exec", "file_read", "docker_run"],
    "load":           0.42,
    "available":      true
  }
}
```

Each agent forwards to K random peers, decrementing `ttl`. The harness builds a probabilistic view
of the entire mesh without every agent connecting to a central registry.

**When to add gossip:** When you have 5+ worker agents and want automatic load balancing without
modifying the harness for each new agent type.

#### Swarm pattern — parallel task execution

```
Manager receives: "Process these 100 log files"

Manager broadcasts task chunks to N workers:
  worker-1: files 1-25
  worker-2: files 26-50
  worker-3: files 51-75
  worker-4: files 76-100

Workers post results to shared "inbox" channel.
Manager aggregates when all complete (or timeout).
```

**Implementation in Go:** Use a `sync.WaitGroup` + result channel per swarm invocation. The
harness maintains a `SwarmSession` mapping a session ID to expected worker count and result
accumulator.

---

### 5. OpenAI Agents SDK — Handoff Patterns

**Source:** [openai.github.io/openai-agents-python](https://openai.github.io/openai-agents-python/)  
**Note:** SDK is Python-first; TypeScript support added March 2025. No official Go SDK.  
The **patterns** are language-agnostic and directly implementable in Go. [VERIFIED: Official docs]

#### The two core primitives

**1. Routines** — an agent is `(system_prompt, tool_set)`. That's the entire abstraction.

**2. Handoffs** — a tool that returns another agent identifier instead of a string result.
The orchestrator detects the agent return type and transfers the conversation.

#### Handoff in Go terms

```go
type Agent struct {
    Name         string
    Instructions string
    Tools        []Tool
    HandoffDests []string  // agent names this agent can hand off to
}

type HandoffResult struct {
    TargetAgent   string
    InputHistory  []Message  // full conversation history forwarded
    InputFilter   func([]Message) []Message  // optional history pruning
}

// The orchestration loop:
func RunAgentLoop(current *Agent, messages []Message) ([]Message, error) {
    for {
        result := current.ProcessMessages(messages)

        if result.IsHandoff() {
            // Apply input filter if specified
            filtered := result.Handoff.InputFilter(messages)
            // Lookup new agent
            next := registry[result.Handoff.TargetAgent]
            // Broadcast: "Transferred to [AgentName]. Adopt persona immediately."
            messages = append(filtered, handoffNotice(next.Name))
            current = next
            continue
        }

        if result.IsTerminal() {
            return messages, nil
        }

        messages = append(messages, result.NewMessages...)
    }
}
```

#### HandoffInputData — what gets passed on transfer

```go
type HandoffInputData struct {
    InputHistory     []Message  // pre-run conversation
    PreHandoffItems  []Message  // items generated before handoff invocation
    NewItems         []Message  // items from current turn including handoff call
    InputItems       []Message  // optional filtered items for next agent
}
```

**Critical pattern:** The conversation `messages` array IS the context. No separate context object.
Agent B gets exactly what Agent A produced, optionally filtered. This maps to your current system
where you could forward the full ACP trace log as context on each agent-to-agent routing.

#### Guardrails — the approval gate generalized

Your current system has one approval gate (human approves shell commands). The Agents SDK
generalizes this to a guardrail pipeline:

```go
type GuardrailResult struct {
    Passed  bool
    Reason  string
    Tripwire bool  // if true, abort the entire run
}

type InputGuardrail  func(ctx context.Context, input string) GuardrailResult
type OutputGuardrail func(ctx context.Context, output string) GuardrailResult

// Apply before tool execution:
for _, guard := range agent.InputGuardrails {
    result := guard(ctx, toolCallArgs)
    if result.Tripwire {
        return ErrGuardrailTripped
    }
    if !result.Passed {
        return ErrGuardrailFailed
    }
}
```

**For your mesh:** Add `policy_check` as an intermediate agent that every `execute_task` message
passes through. The policy agent can be hot-swapped without touching the worker.

#### Tracing — end-to-end observability

The OpenAI SDK traces every: LLM generation, tool call, handoff, guardrail check, custom event.
Trace format is OpenTelemetry-compatible.

```go
// Using OpenTelemetry in Go:
import "go.opentelemetry.io/otel"

tracer := otel.Tracer("acp-mesh")

ctx, span := tracer.Start(ctx, "agent.execute_task",
    trace.WithAttributes(
        attribute.String("agent.id", agentID),
        attribute.String("task.method", msg.Method),
        attribute.String("task.id", taskID),
    ),
)
defer span.End()

// Record handoff:
span.AddEvent("handoff", trace.WithAttributes(
    attribute.String("from", senderAgent),
    attribute.String("to", targetAgent),
))
```

---

### 6. AG-UI — Agent-User Interaction Protocol (Bonus: Replaces Your Raw PTY Stream)

**Status:** Open standard, Go SDK available  
**Source:** [docs.ag-ui.com](https://docs.ag-ui.com) / [github.com/ag-ui-protocol/ag-ui](https://github.com/ag-ui-protocol/ag-ui)  
**Go SDK:** `github.com/ag-ui-protocol/ag-ui/sdks/community/go`  
[VERIFIED: Official GitHub + SDK docs]

Your current UI receives raw PTY bytes from agents, which it renders in an xterm.js terminal.
AG-UI is the **structured version** of exactly that stream. It gives your frontend typed events
instead of raw bytes, enabling proper tool call visualization, thinking indicators, and
state synchronization.

#### Full event taxonomy

| Category | Events | Purpose |
|----------|--------|---------|
| Lifecycle | `RunStarted`, `RunFinished`, `RunError`, `StepStarted`, `StepFinished` | Bracket agent runs |
| Text | `TextMessageStart`, `TextMessageContent` (delta), `TextMessageEnd` | Stream generated text |
| Tool | `ToolCallStart`, `ToolCallArgs`, `ToolCallEnd`, `ToolCallResult` | Show tool execution in UI |
| State | `StateSnapshot`, `StateDelta` (JSON Patch RFC 6902), `MessagesSnapshot` | Sync agent state to UI |
| Activity | `ActivitySnapshot`, `ActivityDelta` | In-progress status indicators |
| Reasoning | `ReasoningStart`, `ReasoningContent`, `ReasoningEnd` | Show LLM thinking steps |
| Special | `Raw`, `Custom` | Extension points |

#### Wire format (SSE)

```
event: RunStarted
data: {"type":"RUN_STARTED","timestamp":1716992880,"runId":"run-001","threadId":"sess-xyz"}

event: TextMessageStart  
data: {"type":"TEXT_MESSAGE_START","messageId":"msg-001","role":"assistant"}

event: TextMessageContent
data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"msg-001","delta":"Executing "}

event: TextMessageContent
data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"msg-001","delta":"ls -la /tmp..."}

event: ToolCallStart
data: {"type":"TOOL_CALL_START","toolCallId":"tc-001","toolCallName":"shell_exec","parentMessageId":"msg-001"}

event: ToolCallArgs
data: {"type":"TOOL_CALL_ARGS","toolCallId":"tc-001","delta":"{\"command\":\"ls -la /tmp\"}"}

event: ToolCallEnd
data: {"type":"TOOL_CALL_END","toolCallId":"tc-001"}

event: TextMessageEnd
data: {"type":"TEXT_MESSAGE_END","messageId":"msg-001"}

event: RunFinished
data: {"type":"RUN_FINISHED","runId":"run-001"}
```

#### Go server-side emission

```go
import (
    "github.com/ag-ui-protocol/ag-ui/sdks/community/go/encoding"
    "github.com/ag-ui-protocol/ag-ui/sdks/community/go/core/events"
)

func streamAgentResponse(w http.ResponseWriter, runID string, output chan string) {
    enc := encoding.NewSSEWriter(w)

    enc.Write(events.RunStartedEvent{RunID: runID})
    enc.Write(events.TextMessageStartEvent{MessageID: "msg-1", Role: "assistant"})

    for chunk := range output {
        enc.Write(events.TextMessageContentEvent{
            MessageID: "msg-1",
            Delta:     chunk,
        })
    }

    enc.Write(events.TextMessageEndEvent{MessageID: "msg-1"})
    enc.Write(events.RunFinishedEvent{RunID: runID})
}
```

**Migration path from your PTY stream:** Keep xterm.js for raw PTY output. Add an AG-UI stream
endpoint alongside it. Route structured agent responses (manager decisions, task plans) through
AG-UI; keep raw PTY for shell command stdout. The two coexist.

---

## Architectural Responsibility Map

| Capability | Current Tier | Standard Tier | Gap |
|------------|-------------|---------------|-----|
| Agent registration / discovery | Harness (flat map) | Harness + Agent Card JSON | Add structured Agent Card |
| Task routing | Harness (name-based) | Harness (capability-based) | Add capability registry |
| Task lifecycle states | None | Harness tracks per-task state machine | Add state machine |
| Streaming agent output | PTY → WS | AG-UI SSE events → WS | Add AG-UI layer |
| LLM reasoning per agent | None | MCP sampling (agent ↔ harness ↔ LLM) | Add sampling channel |
| Human approval gate | Worker (correct) | Worker with MCP `requiresHumanApproval` hint | Add annotation |
| Multi-turn sessions | None | Harness maintains session map | Add session tracking |
| Observability | None | OTEL spans on every message | Add OTel |
| Auth between agents | None | Bearer token from harness-issued JWTs | Add token issuance |

---

## Prioritized Implementation Roadmap

### Wave 1 — Core Protocol Upgrade (Highest ROI, least disruption)

**1.1 Structured Task Envelope**  
Replace `ACPMessage{method, sender, target, params}` with an A2A-compatible task struct:

```go
type TaskMessage struct {
    JSONRPC   string        `json:"jsonrpc"`
    ID        string        `json:"id"`
    Method    string        `json:"method"`
    SessionID string        `json:"session_id,omitempty"`
    Task      *TaskEnvelope `json:"task,omitempty"`
    Message   *AgentMessage `json:"message,omitempty"`
}

type AgentMessage struct {
    MessageID string        `json:"message_id"`
    Role      string        `json:"role"`
    Parts     []MessagePart `json:"parts"`
}

type MessagePart struct {
    Kind string `json:"kind"`  // "text", "data", "file"
    Text string `json:"text,omitempty"`
    Data any    `json:"data,omitempty"`
}
```

**1.2 Typed Task State Machine in Harness**

```go
type TaskState string
const (
    TaskSubmitted     TaskState = "submitted"
    TaskWorking       TaskState = "working"
    TaskInputRequired TaskState = "input-required"
    TaskCompleted     TaskState = "completed"
    TaskFailed        TaskState = "failed"
    TaskCanceled      TaskState = "canceled"
)

type Task struct {
    ID        string
    SessionID string
    State     TaskState
    CreatedAt time.Time
    UpdatedAt time.Time
    Artifacts []Artifact
}
```

**1.3 Agent Card on Registration**

Replace the flat `register` message's `capabilities []string` with a full Agent Card JSON embedded
in `params`. Store and serve it.

---

### Wave 2 — Capability-Based Routing + Session Tracking

**2.1 Capability registry with skill matching**

```go
type AgentSkill struct {
    ID           string   `json:"id"`
    Name         string   `json:"name"`
    Tags         []string `json:"tags"`
    InputModes   []string `json:"input_modes"`
    OutputModes  []string `json:"output_modes"`
}

type HarnessState struct {
    mu           sync.RWMutex
    agents       map[string]net.Conn
    agentCards   map[string]AgentCard    // structured cards
    skills       map[string][]AgentSkill // skill-id → agents
    sessions     map[string]*Session     // session-id → state
    tasks        map[string]*Task        // task-id → state
    wsClients    map[*websocket.Conn]bool
}

func (h *HarnessState) RouteBySkill(skillID string) (string, error) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    agents := h.skills[skillID]
    if len(agents) == 0 {
        return "", fmt.Errorf("no agent with skill %q", skillID)
    }
    // Round-robin or load-based selection
    return agents[0].ID, nil
}
```

**2.2 Session continuity**

```go
type Session struct {
    ID        string
    Messages  []AgentMessage
    CreatedAt time.Time
    LastSeen  time.Time
}
```

---

### Wave 3 — MCP Tool Protocol + Sampling

**3.1** Add `mark3labs/mcp-go` as worker-agent's tool exposure layer over UDS.  
**3.2** Add sampling handler in harness: when worker sends `sampling/createMessage`, harness
proxies to configured LLM (Claude/GPT/local) and returns result.  
**3.3** Add `notifications/progress` emission from worker during long shell commands.

---

### Wave 4 — AG-UI Streaming + OTel

**4.1** Add AG-UI SSE endpoint at `/agent-stream` alongside existing `/ws`.  
**4.2** Route structured agent messages (plans, decisions) through AG-UI events.  
**4.3** Keep raw PTY stream for terminal output.  
**4.4** Add OpenTelemetry spans on: message receipt, routing decision, task state transitions,
approval gate decisions.

---

### Wave 5 — Gossip + Swarm (Future)

**5.1** Add gossip capability propagation using `weaveworks/mesh` or custom implementation.  
**5.2** Add swarm dispatch: manager can fan-out subtasks to N workers and aggregate results.  
**5.3** Add A2A `push notifications` (webhook delivery of task status to external subscribers).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead |
|---------|-------------|-------------|
| MCP server in Go | Custom JSON-RPC tool registry | `github.com/mark3labs/mcp-go` |
| A2A protocol types and client/server | Custom task envelope structs | `github.com/a2aproject/a2a-go/v2` |
| AG-UI event emission and SSE | Custom SSE encoder + event structs | `github.com/ag-ui-protocol/ag-ui/sdks/community/go` |
| OTel tracing | Custom logging spans | `go.opentelemetry.io/otel` |
| Gossip cluster membership | Custom broadcast loops | `github.com/weaveworks/mesh` (Go, battle-tested in Weave) |
| JWT issuance for inter-agent auth | Custom HMAC scheme | `github.com/golang-jwt/jwt/v5` |

---

## Common Pitfalls

### Pitfall 1: Treating A2A/ACP as drop-in replacements for your existing wire format

**What goes wrong:** Rewriting `ACPMessage` to exactly match A2A JSON breaks all existing agents
simultaneously.  
**Why it happens:** Protocol upgrades feel like all-or-nothing.  
**How to avoid:** Add A2A fields **alongside** existing fields in a transition struct. The harness
accepts both formats and normalizes internally. Old agents keep working; new agents use A2A format.

### Pitfall 2: Adding MCP without understanding the transport mismatch

**What goes wrong:** `mark3labs/mcp-go` defaults to stdio (stdin/stdout pipes). If you run your
MCP server the standard way, it won't integrate with your UDS harness.  
**Why it happens:** MCP's stdio transport assumes it's a subprocess launched by the host.  
**How to avoid:** Pass your UDS `net.Conn` as the `io.ReadWriter` for the MCP server. The
JSON-RPC framing is identical regardless of transport.

### Pitfall 3: Storing task state only in harness memory

**What goes wrong:** Harness restart loses all in-flight tasks and sessions.  
**Why it happens:** The current system has no persistence.  
**How to avoid:** For Wave 1-2, use a simple `sync.Map` with a shutdown snapshot to a JSON file.
For production, use SQLite (single binary, zero deps in Go with `modernc.org/sqlite`).

### Pitfall 4: Conflating A2A and MCP roles

**What goes wrong:** Using MCP for agent-to-agent routing, or using A2A for tool exposure to LLMs.  
**Why it happens:** Both use JSON-RPC 2.0 over similar transports.  
**Correct mapping:**
- **MCP** = agent ↔ LLM tool interface (vertical — model to tools)
- **A2A** = agent ↔ agent coordination (horizontal — peer delegation)
- **AG-UI** = agent ↔ frontend streaming (user-facing layer)

### Pitfall 5: Capability strings without schema

**What goes wrong:** `capabilities: ["shell_exec", "read"]` tells the harness nothing about
input format, output format, or cost/risk level.  
**Why it happens:** String arrays are the simplest thing that works.  
**How to avoid:** Migrate to Agent Card `skills[]` objects with `inputModes`, `outputModes`,
`tags`, and `examples`. These enable semantic matching, not just string equality.

### Pitfall 6: Forgetting the `input-required` state

**What goes wrong:** Worker blocks waiting for approval but harness has no way to tell the manager
"task is paused, waiting for human" — manager times out or retries.  
**Why it happens:** Approval gate is currently fire-and-forget from the manager's perspective.  
**How to avoid:** When worker sends `approval_request`, harness must transition task state to
`input-required` and notify manager. Manager polls or subscribes to task state updates.

---

## Standard Stack for Go Agent Mesh (2025)

```bash
go get github.com/a2aproject/a2a-go/v2
go get github.com/mark3labs/mcp-go
go get github.com/ag-ui-protocol/ag-ui/sdks/community/go
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/otel/sdk/trace
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
go get github.com/golang-jwt/jwt/v5
```

**Keep:** `github.com/gorilla/websocket`, `github.com/creack/pty`  
**Keep UDS transport** — it is faster than HTTP for local inter-process communication and not in
conflict with any of the above protocols. Use HTTP/A2A for cross-host agent communication only.

---

## Protocol Comparison Matrix

| Dimension | MCP | A2A | ACP (legacy) | OpenAI Agents SDK | AG-UI |
|-----------|-----|-----|--------------|-------------------|-------|
| Primary role | Tool/context injection | Agent coordination | Agent communication | Orchestration patterns | Frontend streaming |
| Transport | stdio / HTTP SSE | HTTP / JSON-RPC | HTTP REST | In-process | HTTP SSE / WS |
| Go SDK | mark3labs/mcp-go | a2aproject/a2a-go | (deprecated) | None (patterns only) | ag-ui-protocol/ag-ui |
| Task states | None (request/response) | Full lifecycle | run_id + status | None (stateless) | Run lifecycle events |
| Streaming | Progress notifications | SSE artifacts | SSE deltas | Not defined | Full event taxonomy |
| Discovery | Tool list endpoint | Agent Card JSON | Agent manifest | None | None |
| Auth | None in spec | OAuth2/mTLS/API key | Capability tokens | Not defined | Bearer token |
| Multi-turn | Stateful connections | input-required state | session_id + await | Message history array | Thread/run model |
| Status | Active (2025-11-25) | Active (LF, v0.3+) | Merged into A2A | Active (March 2025) | Active (2025) |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | ACP is fully merged into A2A and not receiving new features | ACP section | If ACP continues independently, could miss ACP-specific features |
| A2 | `mark3labs/mcp-go` supports UDS transport via io.ReadWriter wrapping | MCP Go section | May require additional adapter code |
| A3 | A2A Go SDK `a2a-go/v2` is production-stable | A2A section | May have API instability; verify with `go get` + check open issues |
| A4 | AG-UI Go SDK is community-maintained, not official | AG-UI section | May lag behind the TypeScript reference implementation |
| A5 | `weaveworks/mesh` is still maintained for gossip | Wave 5 | Repo may be archived; check before adopting |

---

## Sources

### Primary (HIGH confidence — official specifications)
- [A2A Protocol Specification (latest)](https://a2a-protocol.org/latest/specification/) — task lifecycle, Agent Card, SSE format, auth
- [MCP Specification 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25) — roots, sampling, tools, transport
- [MCP Client Spec (roots)](https://modelcontextprotocol.io/specification/2025-11-25/client) — roots protocol messages
- [A2A Go SDK](https://github.com/a2aproject/a2a-go) — package structure, transport bindings
- [MCP Go SDK — mark3labs/mcp-go](https://mcp-go.dev/getting-started/) — server setup, tool registration
- [AG-UI Protocol Spec](https://docs.ag-ui.com/concepts/events) — full event taxonomy
- [AG-UI Go SDK](https://github.com/ag-ui-protocol/ag-ui/blob/main/docs/sdk/go/overview.mdx) — Go packages and SSE client

### Secondary (MEDIUM confidence — verified against official sources)
- [WorkOS: IBM ACP Technical Overview](https://workos.com/blog/ibm-agent-communication-protocol-acp) — ACP REST API, run schema, streaming
- [OpenAI Agents SDK Handoffs](https://openai.github.io/openai-agents-python/handoffs/) — HandoffInputData schema, input_filter patterns
- [IBM ACP GitHub](https://github.com/i-am-bee/acp) — endpoint paths, message structure
- [A2A Agent Skills Tutorial](https://a2a-protocol.org/latest/tutorials/python/3-agent-skills-and-card/) — Agent Card JSON example
- [Portkey MCP Message Reference](https://portkey.ai/blog/mcp-message-types-complete-json-rpc-reference-guide/) — JSON-RPC schemas
- [Gossip Protocols for Agent Coordination](https://arxiv.org/html/2508.01531v1) — convergence properties, O(log N) scaling
- [Go Multi-Agent Best Practices](https://dasroot.net/posts/2026/03/best-practices-multi-agent-workflows-go/) — goroutine patterns, gRPC integration

### Tertiary (LOW confidence — single source or training knowledge)
- Agent mesh topology characterization (3-8 agent mesh vs. swarm thresholds) — cross-referenced but not from a single authoritative source
- `weaveworks/mesh` gossip library fitness for this use case — needs project health verification

---

## Metadata

**Research date:** 2026-05-18  
**Valid until:** 2026-08-18 (A2A and MCP are fast-moving; re-verify spec versions before Wave 3+)  
**Confidence breakdown:**
- Protocol specifications (A2A, MCP): HIGH — verified against official spec sites
- Go SDK availability and APIs: HIGH — verified against pkg.go.dev and GitHub
- AG-UI Go SDK maturity: MEDIUM — community SDK, TypeScript is reference
- Gossip/swarm patterns: MEDIUM — research-backed, limited Go production case studies
- IBM ACP current status: MEDIUM — merge with A2A confirmed but transition timeline unclear
