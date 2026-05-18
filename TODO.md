# ACP Go Mesh — Evolution Roadmap

## Wave 1: Dual-protocol envelope + first-class input-required/approval resume ✅

### Step 1 — Audit current protocol + lifecycle coupling
- [x] Identify current wire format: ACPMessage with method routing.
- [x] Identify existing task/session state handling in harness.
- [x] Identify worker approval/input-required implementation.

### Step 2 — Add normalized A2A-like internal envelope (no breaking changes yet)
- [x] Introduce an internal “A2A envelope” struct and conversion functions:
  - ACPMessage -> internal Task/Message representation
  - internal Task updates -> ACPMessage tasks/sendUpdate payloads
- [x] Gate outgoing/ingoing behavior to preserve legacy method names.

### Step 3 — Complete task lifecycle semantics in harness
- [x] Make harness treat `input-required` as a first-class pause state:
  - when transition -> notify UI/manager
  - ensure metadata (requester/targetAgent/sessionId) is preserved
- [x] Ensure approval_response causes a deterministic transition:
  - `input-required` -> `working`
  - re-forward the original “awaited” request/intent with session continuity.

### Step 4 — Trace propagation end-to-end
- [x] Ensure traceId/spanId is consistently emitted for:
  - task created via tool call
  - tasks/sendUpdate transitions
  - approval transitions

### Step 5 — Incremental worker/manager compatibility
- [x] Update worker/manager to use new internal envelope conversion helpers (but keep methods).
- [x] Add minimal tests or at least local “smoke” scripts for multi-turn approval.

### Step 6 — Verify with end-to-end run
- [x] Manual run:
  - start harness
  - send user_intent
  - observe input-required notification
  - approve
  - verify completed output and trace continuity

---

## Wave 2: Full A2A Protocol Migration + Advanced Features

### 1. Full A2A Protocol Migration
- [ ] Replace ACPMessage with native A2AEnvelope throughout codebase
- [ ] Update UDS transport to use A2A wire format
- [ ] Update WebSocket transport to use A2A wire format
- [ ] Migrate all message handlers to A2A methods
- [ ] Remove legacy ACPMessage conversion helpers
- [ ] Update tests to use A2A format

### 2. Security Hardening
- [ ] Implement JWT-based agent authentication
- [ ] Add TLS encryption for all transports (UDS, TCP, WebSocket)
- [ ] Implement RBAC for agent capabilities
- [ ] Add message signing and verification
- [ ] Secure task metadata (secrets, credentials)
- [ ] Add audit logging for security events

### 3. Multi-Agent Coordination
- [ ] Implement agent discovery service
- [ ] Add capability matching and registration
- [ ] Implement agent health checks and heartbeat
- [ ] Add load balancing for agent selection
- [ ] Implement agent pool management
- [ ] Add agent lifecycle (spawn, terminate, restart)

### 4. Advanced Task Orchestration
- [ ] Implement task dependencies and DAG execution
- [ ] Add task scheduling and queue management
- [ ] Implement workflow execution engine
- [ ] Add task retry and failure handling policies
- [ ] Implement task cancellation and rollback
- [ ] Add task priority and deadline support

### 5. Enhanced Observability
- [ ] Add Prometheus metrics endpoint
- [ ] Implement structured logging with correlation IDs
- [ ] Add distributed tracing with Jaeger/OTel
- [ ] Implement alerting rules and notifications
- [ ] Add performance profiling endpoints
- [ ] Create dashboards for system health

### 6. Performance Optimization
- [ ] Implement connection pooling for transports
- [ ] Add message batching for efficiency
- [ ] Implement async message processing
- [ ] Add caching for frequently accessed data
- [ ] Optimize serialization (protobuf vs JSON)
- [ ] Add rate limiting and backpressure handling

