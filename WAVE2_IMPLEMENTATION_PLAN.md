# Wave 2: Full A2A Protocol Migration + Advanced Features
**Implementation Plan v1.0**
**Started: 2026-05-18**

## Overview
This document tracks the implementation of 36 sub-tasks across 6 major feature categories for the ACP Go Mesh evolution.

---

## 1. Full A2A Protocol Migration (6 sub-tasks)

### 1.1 Replace ACPMessage with native A2AEnvelope throughout codebase
- **Status:** 🟡 In Progress
- **Files:** `main.go`, `pkg/a2a/types.go`, `pkg/harness/taskstore.go`
- **Dependencies:** None
- **Deliverable:** All internal message passing uses A2AEnvelope
- **Tests:** Unit tests for message conversion

### 1.2 Update UDS transport to use A2A wire format
- **Status:** 📋 Planned
- **Files:** `main.go` (handleUDSConnection)
- **Dependencies:** 1.1
- **Deliverable:** UDS socket sends/receives pure A2A messages
- **Tests:** Integration test for UDS A2A transport

### 1.3 Update WebSocket transport to use A2A wire format
- **Status:** 📋 Planned
- **Files:** `main.go` (handleWebSocket, handleWSIncoming)
- **Dependencies:** 1.1
- **Deliverable:** WebSocket uses A2A envelope format
- **Tests:** WebSocket client test with A2A messages

### 1.4 Migrate all message handlers to A2A methods
- **Status:** 📋 Planned
- **Files:** `main.go` (all dispatch handlers)
- **Dependencies:** 1.2, 1.3
- **Deliverable:** All handlers work with A2A types natively
- **Tests:** Handler unit tests

### 1.5 Remove legacy ACPMessage conversion helpers
- **Status:** 📋 Planned
- **Files:** `pkg/a2a/types.go`
- **Dependencies:** 1.4
- **Deliverable:** Clean A2A-only codebase
- **Tests:** Verify no compilation errors

### 1.6 Update tests to use A2A format
- **Status:** 📋 Planned
- **Files:** `pkg/a2a/types_test.go`, new test files
- **Dependencies:** 1.5
- **Deliverable:** Complete test coverage for A2A protocol
- **Tests:** Full integration test suite

---

## 2. Security Hardening (6 sub-tasks)

### 2.1 Implement JWT-based agent authentication
- **Status:** 📋 Planned
- **Files:** `pkg/auth/jwt.go` (new), `main.go`
- **Dependencies:** None
- **Deliverable:** JWT tokens for agent identity
- **Tests:** Auth token generation and validation

### 2.2 Add TLS encryption for all transports
- **Status:** 📋 Planned
- **Files:** `pkg/transport/tls.go` (new), `main.go`
- **Dependencies:** 2.1
- **Deliverable:** TLS for UDS, TCP, WebSocket
- **Tests:** TLS handshake verification

### 2.3 Implement RBAC for agent capabilities
- **Status:** 📋 Planned
- **Files:** `pkg/auth/rbac.go` (new), `pkg/harness/taskstore.go`
- **Dependencies:** 2.1
- **Deliverable:** Role-based access control
- **Tests:** Permission enforcement tests

### 2.4 Add message signing and verification
- **Status:** 📋 Planned
- **Files:** `pkg/crypto/signing.go` (new)
- **Dependencies:** 2.1
- **Deliverable:** Cryptographic message signatures
- **Tests:** Sign/verify roundtrip tests

### 2.5 Secure task metadata (secrets, credentials)
- **Status:** 📋 Planned
- **Files:** `pkg/secrets/vault.go` (new), `pkg/harness/taskstore.go`
- **Dependencies:** 2.2, 2.4
- **Deliverable:** Encrypted sensitive metadata
- **Tests:** Secret storage/retrieval tests

### 2.6 Add audit logging for security events
- **Status:** 📋 Planned
- **Files:** `pkg/audit/logger.go` (new)
- **Dependencies:** 2.3
- **Deliverable:** Comprehensive security audit trail
- **Tests:** Audit log capture verification

---

## 3. Multi-Agent Coordination (6 sub-tasks)

### 3.1 Implement agent discovery service
- **Status:** 📋 Planned
- **Files:** `pkg/discovery/service.go` (new)
- **Dependencies:** None
- **Deliverable:** Dynamic agent registry
- **Tests:** Discovery registration/lookup tests

### 3.2 Add capability matching and registration
- **Status:** 📋 Planned
- **Files:** `pkg/discovery/matcher.go` (new), `main.go`
- **Dependencies:** 3.1
- **Deliverable:** Skill-based agent selection
- **Tests:** Capability matching algorithms

### 3.3 Implement agent health checks and heartbeat
- **Status:** 📋 Planned
- **Files:** `pkg/health/checker.go` (new)
- **Dependencies:** 3.1
- **Deliverable:** Health monitoring system
- **Tests:** Heartbeat timeout detection

### 3.4 Add load balancing for agent selection
- **Status:** 📋 Planned
- **Files:** `pkg/loadbalancer/selector.go` (new)
- **Dependencies:** 3.2, 3.3
- **Deliverable:** Round-robin/weighted agent selection
- **Tests:** Load distribution verification

### 3.5 Implement agent pool management
- **Status:** 📋 Planned
- **Files:** `pkg/pool/manager.go` (new)
- **Dependencies:** 3.4
- **Deliverable:** Dynamic agent scaling
- **Tests:** Pool size adjustment tests

### 3.6 Add agent lifecycle (spawn, terminate, restart)
- **Status:** 📋 Planned
- **Files:** `pkg/lifecycle/manager.go` (new), `main.go`
- **Dependencies:** 3.5
- **Deliverable:** Full agent lifecycle management
- **Tests:** Spawn/restart/terminate tests

---

## 4. Advanced Task Orchestration (6 sub-tasks)

### 4.1 Implement task dependencies and DAG execution
- **Status:** 📋 Planned
- **Files:** `pkg/orchestration/dag.go` (new)
- **Dependencies:** None
- **Deliverable:** Dependency graph execution
- **Tests:** DAG validation and execution

### 4.2 Add task scheduling and queue management
- **Status:** 📋 Planned
- **Files:** `pkg/scheduler/queue.go` (new)
- **Dependencies:** 4.1
- **Deliverable:** Priority queue with scheduling
- **Tests:** Queue ordering verification

### 4.3 Implement workflow execution engine
- **Status:** 📋 Planned
- **Files:** `pkg/workflow/engine.go` (new)
- **Dependencies:** 4.1, 4.2
- **Deliverable:** Multi-step workflow execution
- **Tests:** Workflow state machine tests

### 4.4 Add task retry and failure handling policies
- **Status:** 📋 Planned
- **Files:** `pkg/retry/policy.go` (new), `pkg/harness/taskstore.go`
- **Dependencies:** 4.3
- **Deliverable:** Configurable retry strategies
- **Tests:** Exponential backoff tests

### 4.5 Implement task cancellation and rollback
- **Status:** 📋 Planned
- **Files:** `pkg/orchestration/rollback.go` (new)
- **Dependencies:** 4.3
- **Deliverable:** Graceful task cancellation
- **Tests:** Rollback compensation tests

### 4.6 Add task priority and deadline support
- **Status:** 📋 Planned
- **Files:** `pkg/scheduler/priority.go` (new)
- **Dependencies:** 4.2
- **Deliverable:** Priority-based scheduling
- **Tests:** Deadline enforcement tests

---

## 5. Enhanced Observability (6 sub-tasks)

### 5.1 Add Prometheus metrics endpoint
- **Status:** 📋 Planned
- **Files:** `pkg/metrics/prometheus.go` (new), `main.go`
- **Dependencies:** None
- **Deliverable:** /metrics endpoint with key metrics
- **Tests:** Metrics scraping verification

### 5.2 Implement structured logging with correlation IDs
- **Status:** 📋 Planned
- **Files:** `pkg/logging/structured.go` (new)
- **Dependencies:** None
- **Deliverable:** JSON logs with trace correlation
- **Tests:** Log parsing and correlation tests

### 5.3 Add distributed tracing with Jaeger/OTel
- **Status:** 🟡 Partial (basic OTel exists)
- **Files:** `pkg/tracing/otel.go` (new), `main.go`
- **Dependencies:** 5.2
- **Deliverable:** Complete trace propagation
- **Tests:** Span hierarchy validation

### 5.4 Implement alerting rules and notifications
- **Status:** 📋 Planned
- **Files:** `pkg/alerting/rules.go` (new)
- **Dependencies:** 5.1
- **Deliverable:** Alert manager integration
- **Tests:** Alert trigger verification

### 5.5 Add performance profiling endpoints
- **Status:** 📋 Planned
- **Files:** `pkg/profiling/pprof.go` (new)
- **Dependencies:** None
- **Deliverable:** /debug/pprof endpoints
- **Tests:** Profile capture verification

### 5.6 Create dashboards for system health
- **Status:** 📋 Planned
- **Files:** `dashboards/grafana/*.json` (new)
- **Dependencies:** 5.1, 5.3
- **Deliverable:** Grafana dashboard configs
- **Tests:** Dashboard rendering verification

---

## 6. Performance Optimization (6 sub-tasks)

### 6.1 Implement connection pooling for transports
- **Status:** 📋 Planned
- **Files:** `pkg/transport/pool.go` (new)
- **Dependencies:** None
- **Deliverable:** Reusable connection pools
- **Tests:** Connection reuse verification

### 6.2 Add message batching for efficiency
- **Status:** 📋 Planned
- **Files:** `pkg/transport/batch.go` (new)
- **Dependencies:** 6.1
- **Deliverable:** Batched message sending
- **Tests:** Batch size optimization tests

### 6.3 Implement async message processing
- **Status:** 🟡 Partial (goroutines exist)
- **Files:** `pkg/async/processor.go` (new)
- **Dependencies:** None
- **Deliverable:** Worker pool for message handling
- **Tests:** Throughput benchmarks

### 6.4 Add caching for frequently accessed data
- **Status:** 📋 Planned
- **Files:** `pkg/cache/memory.go` (new)
- **Dependencies:** None
- **Deliverable:** LRU cache for agent data
- **Tests:** Cache hit/miss tests

### 6.5 Optimize serialization (protobuf vs JSON)
- **Status:** 📋 Planned
- **Files:** `pkg/a2a/proto/*.proto` (new)
- **Dependencies:** 1.5
- **Deliverable:** Protobuf message encoding
- **Tests:** Serialization benchmarks

### 6.6 Add rate limiting and backpressure handling
- **Status:** 📋 Planned
- **Files:** `pkg/ratelimit/limiter.go` (new)
- **Dependencies:** 6.2
- **Deliverable:** Token bucket rate limiter
- **Tests:** Rate limit enforcement tests

---

## Implementation Strategy

### Phase 1: Foundation (Week 1-2)
- Complete Full A2A Protocol Migration (1.1-1.6)
- Basic Security (2.1, 2.2)
- Enhanced Observability foundation (5.1, 5.2, 5.3)

### Phase 2: Coordination (Week 3-4)
- Multi-Agent Coordination (3.1-3.6)
- Security hardening (2.3-2.6)

### Phase 3: Advanced Features (Week 5-6)
- Task Orchestration (4.1-4.6)
- Performance Optimization (6.1-6.6)
- Complete Observability (5.4-5.6)

### Phase 4: Testing & Refinement (Week 7-8)
- Integration testing
- Load testing
- Documentation
- Production readiness

---

## Progress Tracking

**Total Sub-tasks:** 36
**Completed:** 0 ✅
**In Progress:** 1 🟡
**Planned:** 35 📋

**Overall Progress:** 2.8% (1/36 started)

---

## Notes

- Wave 1 (dual-protocol support) is complete and stable
- All new features must maintain backward compatibility during migration
- Each sub-task should include comprehensive tests
- Documentation updates required for each category
- Performance benchmarks before/after optimization

---

## Related Documents
- [TODO.md](TODO.md) - Overall roadmap
- [AGENTS.md](AGENTS.md) - Agent architecture
- [DOCKER_DIAGNOSIS.md](DOCKER_DIAGNOSIS.md) - Infrastructure notes
