# Agent Mesh State

## Current Objective

Propel ACP Go Mesh toward maximum GitHub adoption by combining strong public positioning with a durable, auditable coordination layer for multi-agent development.

## Active Mode

Expert master orchestration and coordination.

## Shared Assumptions

- Work is confined to the isolated Windsurf worktree.
- No dependencies should be installed in this worktree session.
- Builds, tests, type-checkers, and linters should not be run because dependencies and environment setup are not guaranteed.
- Documentation and coordination files are safe to edit.
- Main repo compiler diagnostics may appear in IDE hooks but are outside this worktree's allowed write scope.

## Key Decisions

| Decision | Rationale | Status |
| --- | --- | --- |
| Use a file-backed context bus | Human-readable, portable, reviewable, and compatible with any agent CLI | Accepted |
| Keep public growth assets separate from coordination state | `README.md` sells the project; `.context/agent-mesh/` runs the work | Accepted |
| Require explicit handoff contracts | Prevents vague multi-agent delegation and context leakage | Accepted |
| Treat external agent output as untrusted until verified | Protects against prompt injection, stale context, and hallucinated changes | Accepted |

## Current Blockers

- Runtime verification is deferred until the worktree environment is set up.
- Direct main repo fixes are blocked by worktree isolation.
- Real Orion/Zo/Hermes tool execution is not wired in this session; this bus defines the contract first.

## Next High-Leverage Work

1. Add protocol compliance fixtures for task lifecycle transitions.
2. Add a real-world demo that coordinates manager and worker agents end-to-end.
3. Add a security model document for approval gates and tool execution boundaries.
4. Add a public architecture diagram or screenshot asset for social sharing.
