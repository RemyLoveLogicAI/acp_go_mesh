# Agent Mesh Registry

This registry is the canonical coordination map for ACP Go Mesh work. It defines which agent surfaces exist, what each owns, and how handoffs should be scoped.

## Mesh Inventory

| Agent or Surface | Role | Context source | Write scope | Handoff method | Risk |
| --- | --- | --- | --- | --- | --- |
| Orion | Control plane and provider router | `AGENTS.md`, future runtime topology | Routing metadata, task state, topology events | A2A task packet or `orion/execute_task` | High: central routing mistakes affect every agent |
| Zo | Worker hub and remote execution surface | `AGENTS.md`, node topology | Worker outputs, NATS-backed execution artifacts | Routed task packet with declared capability | High: remote execution and broker access |
| Loki | Orchestrator and scheduler | Roadmap, issue queue, `.context/agent-mesh/state.md` | Task plans, handoff rows, scheduling notes | `.context/agent-mesh/inbox/` packet | Medium: stale task state can misroute work |
| Feynman | Research and synthesis agent | `RESEARCH.md`, linked protocol docs | Research briefs and evidence summaries | Research packet with source requirements | Medium: unsupported claims can corrupt strategy |
| Fusion | Mesh coordinator and worktree sync | `AGENTS.md`, `.context/agent-mesh/*` | Coordination docs, handoff packets, sync plans | Registry + state + ledger update | High: cross-worktree sync can overwrite changes |
| Hermes | Messenger and alert bridge | `AGENTS.md`, future notification config | Notification summaries only | Ledger event or alert packet | Medium: notification channels may expose sensitive context |
| Sage | Reviewer and quality gate | Code diff, Codacy output, security notes | Review reports and verification notes | Review packet with read-only scope | Medium: false confidence if review is shallow |
| Manager Agent | In-repo task orchestrator | `agent/manager/`, `main.go`, `pkg/harness/` | Runtime task routing code | A2A-style task request | Medium: routing bugs block execution |
| Worker Agent | In-repo execution worker | `agent/worker/`, `main.go` | Worker execution code and artifacts | Capability-routed task | High: tool execution must remain scoped |
| GitHub Community Surface | Contributor intake | `.github/ISSUE_TEMPLATE/`, PR template | Issues, PRs, contributor reports | GitHub issue or PR | Low: public input is untrusted |

## Coordination Principles

- Use `.context/agent-mesh/state.md` as the current source of truth for active objectives.
- Use `.context/agent-mesh/handoffs.md` for assignments, owners, expected outputs, and status.
- Use `.context/agent-mesh/ledger.md` for auditable actions, verification, and rollback notes.
- Store task packets in `.context/agent-mesh/inbox/` and completed reports in `.context/agent-mesh/outbox/`.
- Never place secrets, tokens, private URLs, or sensitive personal data in this bus.
- Every handoff must declare read scope, write scope, tools allowed, acceptance criteria, stop conditions, and rollback notes.
