# Wave 2 Implementation Plan

## 1. Full A2A Protocol Migration

- [ ] Replace remaining legacy `ACPMessage` call sites with `a2a.A2AEnvelope` or typed A2A payloads.
- [ ] Add explicit payload decoders for task, tool, approval, cancellation, and discovery messages.
- [ ] Update UDS transport to accept both legacy and A2A envelopes during migration.
- [ ] Update WebSocket events to emit canonical A2A task and message shapes.
- [ ] Remove duplicated protocol structs from agents and scripts after all packages use `pkg/a2a`.
- [ ] Add protocol compatibility tests for legacy input and A2A-native input.

## 2. Security Hardening

- [ ] Add agent identity validation during registration.
- [ ] Sign agent-to-harness messages with per-agent shared secrets or local keypairs.
- [ ] Enforce capability authorization before forwarding tool calls.
- [ ] Redact secrets from task metadata, artifacts, logs, and WebSocket broadcasts.
- [ ] Add audit events for registration, approval, rejection, cancellation, and tool execution.
- [ ] Add configurable deny/allow policies for shell commands and filesystem access.

## 3. Multi-Agent Coordination

- [ ] Promote agent cards into a first-class registry with lifecycle state.
- [ ] Add heartbeat tracking and stale-agent eviction.
- [ ] Implement capability scoring for routing among multiple eligible agents.
- [ ] Add load-aware selection using active task counts and recent failure rates.
- [ ] Support agent pools with spawn, stop, restart, and drain operations.
- [ ] Add coordination tests for multi-worker routing and failover.

## 4. Advanced Task Orchestration

- [ ] Add task dependency metadata and DAG validation.
- [ ] Implement queued execution for tasks blocked by dependencies.
- [ ] Add retry policies with max attempts, retryable errors, and backoff.
- [ ] Add deadline and priority fields to task scheduling.
- [ ] Implement cancellation propagation across dependent tasks.
- [ ] Add rollback hooks for failed multi-step workflows.

## 5. Enhanced Observability

- [ ] Add structured logs with task ID, session ID, agent ID, trace ID, and span ID.
- [ ] Expose a Prometheus-compatible metrics endpoint.
- [ ] Track task lifecycle counters and duration histograms.
- [ ] Expand OpenTelemetry spans around routing, approvals, queueing, and execution.
- [ ] Add health endpoint details for agents, queues, transports, and task store state.
- [ ] Create dashboard-ready JSON snapshots for the command center UI.

## 6. Performance Optimization

- [ ] Replace per-message lock hotspots with narrower critical sections.
- [ ] Add bounded outbound queues per agent connection.
- [ ] Batch low-priority WebSocket state updates under load.
- [ ] Add backpressure handling for slow agents and slow UI clients.
- [ ] Benchmark JSON encoding hot paths and evaluate pooled buffers.
- [ ] Add load tests for concurrent tasks, concurrent agents, and WebSocket fanout.

## Validation Gates

- [ ] `go test ./...` passes.
- [ ] Dockerless smoke test passes with `go run main.go` and `go run scripts/smoke.go`.
- [ ] Codacy analysis passes for all edited files.
- [ ] README and TODO stay free of merge-conflict markers.
