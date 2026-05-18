# ACP Go Mesh 🚀

> The local-first control plane for fleets of AI agents

[![Go Version](https://img.shields.io/badge/Go-1.26.1+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-Enabled-blue.svg)](https://opentelemetry.io/)

ACP Go Mesh turns a pile of disconnected coding agents, browser agents, CLIs, MCP servers, and local workers into one observable, zero-trust mesh. It is built in Go for teams that want agent orchestration they can inspect, self-host, secure, and extend without giving up custody of their workflows.

> If Kubernetes was the missing control plane for containers, ACP Go Mesh aims to be the missing control plane for local and distributed AI agents.

## ⭐ Star This If You Believe

- **Agents should coordinate** through protocols, not fragile prompt chains.
- **Sensitive work should stay local-first** by default.
- **Every tool call should be observable, cancellable, and auditable.**
- **Human approval gates should be built into the mesh**, not bolted on later.

## ✨ Why ACP Go Mesh?

- **🎯 Protocol-native**: A2A-style task lifecycle, MCP-style tools, and AG-UI-style interaction events
- **⚡ Local-first transport**: Unix Domain Sockets for local agents, with a path to NATS+TLS for remote mesh links
- **🔒 Security-first orchestration**: Human-in-the-loop approval gates, capability scoping, session isolation, and zero-trust design
- **📊 Observable by default**: OpenTelemetry traces, real-time task status, and live topology visibility
- **🧠 Skill-based routing**: Route by capability instead of hardcoded agent names
- **🔄 Deterministic task lifecycle**: Submitted, working, input-required, completed, failed, or canceled
- **🎛️ Operator command center**: Web UI for task streaming, topology visualization, and session management

## 🧲 What Makes It Star-Worthy?

| Problem | ACP Go Mesh Approach |
| --- | --- |
| Too many agent CLIs with no shared control plane | One harness coordinates workers, managers, tools, and UI streams |
| Tool calls are opaque and risky | Approval gates and traceable task transitions |
| Local agents cannot discover each other | Agent cards, capability registry, and skill-based routing |
| Multi-agent demos collapse in production | Formal state machine, cancellation, sessions, and observability |
| Cloud orchestration leaks sensitive context | Local-first UDS transport with remote mesh support as an explicit choice |

## 🚀 Quick Start

```bash
# Clone the repository
git clone https://github.com/yourusername/acp_go_mesh.git
cd acp_go_mesh

# Run the harness (starts UDS server, web UI, and spawns default agents)
go run main.go

# Open the command center UI
open http://localhost:8080
```

The harness will automatically spawn:

- **Worker Agent**: Shell execution and file operations
- **Manager Agent**: Task orchestration and routing

## ⚡ 60-Second Demo
<<<<<<< /Users/lovelogic/Documents/01-Active-Projects/acp_go_mesh/README.md
<<<<<<< /Users/lovelogic/Documents/01-Active-Projects/acp_go_mesh/README.md
<<<<<<< /Users/lovelogic/Documents/01-Active-Projects/acp_go_mesh/README.md
<<<<<<< /Users/lovelogic/Documents/01-Active-Projects/acp_go_mesh/README.md

```bash
go run main.go
```

Then open `http://localhost:8080` and send a task through the manager. The harness registers agents, routes by capability, tracks task state, streams updates to the UI, and records telemetry for the execution path.

Expected flow:

```text
user intent → manager → capability match → worker → task update stream → completed artifact
```

### Dockerless Smoke Test Workaround

If Docker Desktop is unavailable or its socket is missing, skip Docker entirely. The harness and smoke test run as local Go processes:

```bash
go run main.go
```

In a second terminal:

```bash
go run scripts/smoke.go
```

The smoke test expects the harness health endpoint at `http://localhost:8080/health`. If port `8080` is already occupied, stop the existing harness process before starting a new one.
=======
=======
>>>>>>> /Users/lovelogic/.windsurf/worktrees/acp_go_mesh/acp_go_mesh-a420476a/README.md
=======
>>>>>>> /Users/lovelogic/.windsurf/worktrees/acp_go_mesh/acp_go_mesh-a420476a/README.md

```bash
go run main.go
```

Then open `http://localhost:8080` and send a task through the manager. The harness registers agents, routes by capability, tracks task state, streams updates to the UI, and records telemetry for the execution path.

Expected flow:

```text
user intent → manager → capability match → worker → task update stream → completed artifact
```
<<<<<<< /Users/lovelogic/Documents/01-Active-Projects/acp_go_mesh/README.md
<<<<<<< /Users/lovelogic/Documents/01-Active-Projects/acp_go_mesh/README.md
>>>>>>> /Users/lovelogic/.windsurf/worktrees/acp_go_mesh/acp_go_mesh-a420476a/README.md
=======
>>>>>>> /Users/lovelogic/.windsurf/worktrees/acp_go_mesh/acp_go_mesh-a420476a/README.md
=======
>>>>>>> /Users/lovelogic/.windsurf/worktrees/acp_go_mesh/acp_go_mesh-a420476a/README.md
=======

```bash
go run main.go
```

Then open `http://localhost:8080` and send a task through the manager. The harness registers agents, routes by capability, tracks task state, streams updates to the UI, and records telemetry for the execution path.

Expected flow:

```text
user intent → manager → capability match → worker → task update stream → completed artifact
```
>>>>>>> /Users/lovelogic/.windsurf/worktrees/acp_go_mesh/acp_go_mesh-a420476a/README.md

## 🧭 Use Cases

- **Local agent operating system** for Claude Code, Codex, Gemini CLI, browser agents, shell workers, and custom tools
- **Secure enterprise agent runtime** with human approval gates and audit trails
- **AI lab control plane** for reproducible multi-agent experiments
- **Edge orchestration layer** for private agent swarms running on laptops, homelabs, or Raspberry Pi clusters
- **Protocol playground** for A2A, MCP, AG-UI, and future agent interoperability standards

## 🏗️ Architecture

```text
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   Web UI    │◄────────►│   Harness   │◄────────►│  Agents     │
│  :8080      │  WS/HTTP │  (UDS)      │  UDS     │  (Go procs) │
└─────────────┘         └─────────────┘         └─────────────┘
     │                        │                        │
     │                        │                        │
     ▼                        ▼                        ▼
 Real-time             Task Store              Agent Discovery
 PTY streaming        State Machine           Capability Registry
 Session mgmt         OpenTelemetry           MCP Tool Registry
```

### Core Components

- **Harness**: Central coordination fabric managing agent lifecycle, task routing, and state persistence
- **Task Store**: Thread-safe task lifecycle management with state machine validation
- **Agent Registry**: Capability-based discovery and routing
- **WebSocket Gateway**: Real-time UI streaming with session synchronization
- **PTY Bridge**: Live terminal output streaming from agent processes

## 📋 Protocol Support

### A2A (Agent-to-Agent)

- ✅ Task lifecycle state machine
- ✅ Agent Card discovery (`/.well-known/agent.json`)
- ✅ Capability-based routing
- ✅ Session continuity
- ✅ Artifact streaming

### MCP (Model Context Protocol)

- ✅ Tool registration and discovery
- ✅ Tool call execution with approval gates
- ✅ Server-initiated LLM sampling
- ✅ Progress notifications
- ✅ Cancellation support

### AG-UI (Agent-User Interaction)

- ✅ Structured event streaming (SSE)
- ✅ Tool call visualization
- ✅ State synchronization
- ✅ Activity indicators

## 🔧 Usage Examples

### Basic Agent Communication

```go
// Agent registers with capabilities
msg := ACPMessage{
    JSONRPC: "2.0",
    Method:  "register",
    Sender:  "worker",
    Params: map[string]interface{}{
        "capabilities": []string{"shell_exec", "file_read"},
        "tools": []map[string]interface{}{
            {"name": "shell_exec", "description": "Execute shell commands"},
        },
    },
}
```

### Task Creation with State Tracking

```go
task := a2a.Task{
    ID:        "task_001",
    SessionID: "session_123",
    Status: a2a.NewTaskStatus(a2a.TaskStateSubmitted, "Task created"),
    Metadata: map[string]interface{}{
        "trace_id": otelSpan.SpanContext().TraceID().String(),
    },
}
taskStore.CreateTask(task)
```

### Capability-Based Routing

```go
// Route by skill instead of hardcoded agent name
msg := ACPMessage{
    Method:         "user_intent",
    RequiredSkills: []string{"shell_exec"}, // Auto-routes to capable agent
    Params: map[string]interface{}{
        "command": "ls -la /tmp",
    },
}
```

## 🛡️ Security Features

- **Human-in-the-Loop**: Destructive operations require explicit approval
- **Capability Scoping**: Agents only access tools they're registered for
- **Session Isolation**: Multi-turn conversations with proper context boundaries
- **Zero-Trust Mesh**: Every message authenticated, no plaintext over tailnet
- **PII Detection**: Automatic redaction and human review for sensitive data

## 📊 Observability

### OpenTelemetry Tracing

Every task execution creates a distributed trace with:

- Task lifecycle transitions
- Agent-to-agent routing decisions
- Tool call execution
- Approval gate decisions

### Real-time Metrics

- Tokens/sec per provider
- Latency p99 across the mesh
- Task backlog by agent
- Agent health and availability

## 🗺️ Roadmap

### Wave 1 (Current)

- ✅ A2A task state machine
- ✅ MCP tool integration
- ✅ Capability-based routing
- ✅ OpenTelemetry tracing
- ✅ Session management

### Wave 2 (Next)

- ⏳ Gossip protocol for emergent routing
- ⏳ AG-UI structured streaming
- ⏳ OAuth2/mTLS authentication
- ⏳ Multi-node mesh coordination

### Wave 3 (Future)

- ⏳ Swarm pattern for parallel execution
- ⏳ Hierarchical delegation
- ⏳ Advanced guardrails pipeline
- ⏳ Custom tool annotations

## 🤝 Contributing

The fastest way to help is to pick one of the starter issues:

- **Good first issue**: docs, examples, UI polish, protocol test fixtures
- **Help wanted**: remote mesh transport, security hardening, AG-UI event coverage
- **Research needed**: protocol compatibility matrices, routing algorithms, agent safety patterns

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for the full contributor guide.

## 🌍 Community Flywheel

If this project helps you, the best contributions are:

1. Star the repo so more agent builders discover it.
2. Share a screenshot of your mesh topology or command center.
3. Open an issue with your agent stack and what you want ACP Go Mesh to coordinate.
4. Contribute one adapter, one protocol test, or one real-world example.

## 📚 Documentation

- [Architecture Deep Dive](ARCHITECTURE.md) - System design and protocol details
- [Research Notes](RESEARCH.md) - Protocol analysis and implementation decisions
- [Agent Configuration](AGENTS.md) - Agent roles and capabilities
- [Roadmap](ROADMAP.md) - Implementation timeline and future plans

## 🙏 Acknowledgments

Built on top of excellent open-source standards:

- [A2A Protocol](https://a2a-protocol.org/) - Linux Foundation
- [MCP](https://modelcontextprotocol.io/) - Anthropic
- [AG-UI](https://docs.ag-ui.com/) - Open standard
- [OpenTelemetry](https://opentelemetry.io/) - CNCF

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

## 🔗 Links

- [Documentation](docs/)
- [Examples](examples/)
- [Issues](https://github.com/yourusername/acp_go_mesh/issues)
- [Discussions](https://github.com/yourusername/acp_go_mesh/discussions)

---

**Built for the developers turning isolated AI agents into coordinated systems.**
