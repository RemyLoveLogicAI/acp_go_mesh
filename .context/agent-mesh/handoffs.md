# Agent Mesh Handoffs

| ID | Owner | Objective | Context Sources | Write Scope | Required Output | Status |
| --- | --- | --- | --- | --- | --- | --- |
| mesh-001 | Sage | Review public-facing repo assets for trust, clarity, and security claims | `README.md`, `STAR_THIS_REPO.md`, `.github/` | `.context/agent-mesh/outbox/mesh-001-review.md` | Findings with severity and recommended fixes | Proposed |
| mesh-002 | Feynman | Build protocol compatibility matrix for A2A, MCP, and AG-UI claims | `README.md`, `RESEARCH.md`, `ARCHITECTURE.md` | `.context/agent-mesh/outbox/mesh-002-protocol-matrix.md` | Evidence-backed matrix with gaps | Proposed |
| mesh-003 | Loki | Convert roadmap into contributor-ready milestone issues | `ROADMAP.md`, `.github/ISSUE_TEMPLATE/feature_request.yml` | `.context/agent-mesh/outbox/mesh-003-issue-plan.md` | Prioritized issue plan with labels | Proposed |
| mesh-004 | Fusion | Design safe worktree sync and cross-agent handoff workflow | `AGENTS.md`, `.context/agent-mesh/*` | `.context/agent-mesh/outbox/mesh-004-sync-plan.md` | Sync plan with rollback controls | Proposed |
| mesh-005 | Manager Agent | Define a demo task flow from user intent to worker completion | `main.go`, `agent/manager/`, `agent/worker/`, `pkg/harness/` | `.context/agent-mesh/outbox/mesh-005-demo-flow.md` | Demo script and expected state transitions | Proposed |

## Handoff Rules

- A proposed handoff is not active until an owner accepts it or a human explicitly assigns it.
- Each owner writes only to the declared write scope.
- If required context is missing, the owner must stop and report the blocker.
- If a task requires credentials, package installation, background daemons, network exposure, or destructive operations, the owner must stop for human approval.
