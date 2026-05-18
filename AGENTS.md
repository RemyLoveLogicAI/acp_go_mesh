# AGENTS.md — LoveLogicAI Pi Mesh
# Sovereign Agent Kernel (SAK) — Pi Ecosystem Integration
# Last updated: 2026-05-18

---

## Constitutional Principles

1. **Free-First Inference**: Always route to free providers (Groq, Gemini Flash, Together AI, HuggingFace) before any paid fallback. Local Ollama is tier-0.
2. **Sovereign Local-First**: All sensitive data stays on Mini/Zo. Cloud is for compute, not custody.
3. **Multi-Provider Routing**: Orion Control Plane arbitrates across Groq (realtime), Gemini (vision/long-context), Together (code/reasoning), HF (embeddings), Ollama (local).
4. **Zero-Trust Mesh**: Every ACP message is authenticated. UDS sockets for local, NATS+TLS for remote. No plaintext over tailnet.
5. **Human-in-the-Loop**: Approval gates for destructive ops (git push to main, infra changes, PII exposure). Default deny.
6. **Deterministic Idempotency**: All automation scripts must be idempotent. Re-run safely, no side effects on second pass.
7. **Dark Cinematic Aesthetic**: All dashboards, panels, and UIs use dark themes. War-room style. Kernel-panic aesthetic.

---

## Agent Roles

| Role | Identity | Node | Responsibilities |
|---|---|---|---|
| **Orion** | Control Plane | Mini (100.101.54.8) | Provider routing, MCP tool registry, A2A task store, mesh topology broadcast |
| **Zo** | Worker Hub | Mini / Zo (100.73.154.25) | NATS broker, Playwright automation, inference worker pool |
| **Loki** | Orchestrator | Mini | Kanban daemon, task scheduling, cron orchestration |
| **Feynman** | Researcher | Mini | Literature review, alphaXiv synthesis, intelligence briefings |
| **Fusion** | Mesh Coordinator | Mini + MacBook Air | Multi-node agent orchestration, worktree sync, cross-node delegation |
| **Hermes** | Messenger | Mini | Mattermost bridge, Telegram alerts, ACP message routing |
| **Sage** | Reviewer | Mini | Code audit, security scan, quality gates |

---

## MCP Tool Registry

Tools exposed by Orion Control Plane (via `pi-mcp-adapter`):

- `orion/route_inference` — Route a prompt through the optimal free provider
- `orion/list_models` — List available models across all providers
- `orion/get_topology` — Get current mesh agent topology
- `orion/execute_task` — Delegate a task to a specific mesh node
- `orion/get_task_state` — Query A2A task state by ID
- `orion/broadcast` — Send a message to all mesh agents
- `orion/web_search` — Free-first web search via @ollama/pi-web-search
- `orion/research` — Trigger Feynman literature review

---

## Communication Protocols

| Layer | Protocol | Transport | Security |
|---|---|---|---|
| Local Agent Wire | ACP (Agent Communication Protocol) | UDS /tmp/a2a-mesh.sock | Unix permissions |
| Remote Mesh | ACP over NATS | NATS :4222 + TLS | Tailscale mesh encryption |
| UI Streaming | WebSocket | ws://localhost:8080 | Origin-restricted (localhost only) |
| Mobile Alerts | Telegram Bot API | HTTPS | Bot token auth |
| Team Coordination | Mattermost REST + WebSocket | localhost:8065 | Session token |
| Research Feed | alphaXiv API | HTTPS | API key |

---

## Node Topology

```
                    [Tailnet : ts5.zocomputer.io]
                              |
        +---------------------+---------------------+
        |                     |                     |
   [Mini M4]            [Zo Linux]          [MacBook Air]
  100.101.54.8        100.73.154.25         100.84.222.44
        |                     |                     |
   Ollama:11434         NATS:4222            Fusion Node
   Mattermost:8065      SSH:2288             Pi Worker
   Orion:8000           A2A Mesh             Git Worktrees
   Fusion Daemon        Playwright
   GlimpseUI Panel
```

---

## Task Delegation Flow

1. **Request** -> Manager agent sends `tasks/send` to Orion
2. **Route** -> Orion queries topology cache, selects target worker
3. **Forward** -> Orion sends task via `taskStore.SendToAgent()` (thread-safe channel)
4. **Execute** -> Worker processes task, streams updates via `tasks/sendUpdate`
5. **Review** -> Sage agent audits result (if configured)
6. **Approve** -> Human-in-the-loop for sensitive ops
7. **Archive** -> Task state persisted in A2A store, broadcast to UI

---

## Security Gates

- `approval_request` is broadcast to WebSocket clients for human review
- `approval_response` transitions task to `working` or `failed`
- PII detection triggers automatic redaction + human review
- Hallucination scores above threshold trigger re-verification via second provider
- All destructive git ops require explicit PR approval

---

## Free Provider Priority Ladder

1. **Ollama local** (gemma4, qwen3:30b, qwen3.5, llama3.2) — zero cost, full privacy
2. **Groq** — ultra-low latency, good for realtime/interactive
3. **Gemini Flash** — vision, long context (1M tokens), tool use
4. **Together AI** — code generation, complex reasoning (DeepSeek-R1)
5. **HuggingFace** — lightweight tasks, specialized embeddings
6. **Claude** — paid fallback only if `costPolicy: paid_allowed`

---

## Observability Signals

| Signal | Source | Channel |
|---|---|---|
| Tokens/sec | Orion | Mattermost #ops-dashboard |
| Latency p99 | Orion | Mattermost #ops-dashboard |
| Hallucination score | Sage reviewer | Mattermost #peer-coordination |
| PII detection | Content filter | Mattermost #human-override |
| Revenue | Usage tracker | Mattermost #ops-dashboard |
| Risk score | Security auditor | Mattermost #human-override |
| Agent health | Fusion | Mattermost #agent-hermes-mini |
| Task backlog | Loki kanban | Mattermost #agent-loki |

---

## Quick Commands

```bash
# Start Fusion daemon
npx fusion daemon --config .fusion/config.json

# Trigger Feynman research
feynman research "quantum error correction" --output markdown

# Open war-room panel
npx glimpseui war-room.html --title "LoveLogicAI Mesh"

# Check mesh topology
curl http://localhost:8000/topology

# Send mobile alert
python3 ~/.config/lovelogic/pi-mesh/telegram_bot.py --alert "Fusion node online"

# Post to Mattermost
python3 ~/.config/lovelogic/pi-mesh/mattermost_bridge.py --channel peer-coordination --message "Mesh state updated"
```
