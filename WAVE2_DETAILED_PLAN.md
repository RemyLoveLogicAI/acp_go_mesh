# Wave 2 Implementation Plan — Detailed Breakdown

**Project:** ACP Go Mesh - Sovereign Agent Kernel
**Phase:** Wave 2 - Full A2A Protocol Migration + Advanced Features
**Start Date:** 2026-05-18
**Priority:** Production-Ready Multi-Agent System

---

## Executive Summary

Wave 1 successfully established the foundation:
- ✅ Dual-protocol support (A2AEnvelope + ACPMessage)
- ✅ Task lifecycle with state machine
- ✅ input-required approval gating
- ✅ Trace propagation
- ✅ Basic multi-agent routing

Wave 2 transforms this into a production-grade, secure, scalable mesh:
- Native A2A protocol throughout
- Security hardening (JWT, TLS, RBAC)
- Advanced orchestration (DAG, workflows, dependencies)
- Production observability (Prometheus, structured logging, distributed tracing)
- Performance optimization (pooling, batching, caching)
- Enhanced multi-agent coordination

---

## Current Architecture Analysis

### Strengths
1. Clean separation between `pkg/a2a` types and `pkg/harness` business logic
2. Thread-safe TaskStore with pub/sub pattern
3. Working UDS + WebSocket transports
4. Skill-based agent discovery with O(1) lookups
5. Session continuity across tasks
6. OpenTelemetry span tracking

### Migration Points
1. **Still using ACPMessage internally** - conversion overhead
2. **No authentication** - any agent can send any message
3. **No encryption** - all UDS/WS traffic is plaintext
4. **No RBAC** - agents have unrestricted capabilities
5. **Basic metrics** - no Prometheus, structured logs, or dashboards
6. **No advanced orchestration** - single-task execution only
7. **No connection pooling** - each agent has dedicated channel
8. **No message batching** - every update is sent individually

---

## Wave 2.1: Full A2A Protocol Migration

**Goal:** Remove all ACPMessage usage, use A2AEnvelope as the native wire format.

### Phase 2.1.1: Core Type Migration (2-3 hours)
**Files to modify:**
- `pkg/a2a/types.go`
- `pkg/harness/taskstore.go`
- `main.go`
- `agent/worker/main.go`
- `agent/manager/main.go`

**Tasks:**
1. ✅ Review current A2AEnvelope structure
2. [ ] Add A2A-native method constants:
   ```go
   const (
       MethodTasksSend       = "tasks/send"
       MethodTasksUpdate     = "tasks/sendUpdate"
       MethodTasksCancel     = "tasks/cancel"
       MethodRegister        = "register"
       MethodDiscover        = "discover"
       MethodApprovalRequest = "approval_request"
       MethodApprovalResponse = "approval_response"
   )
   ```
3. [ ] Remove `ToACPMessage()` and `ToA2AEnvelope()` conversion methods
4. [ ] Update all message handlers to use `A2AEnvelope` directly
5. [ ] Replace `msg.Params` with typed payload parsing:
   ```go
   type TasksSendPayload struct {
       Task Task `json:"task"`
   }
   ```
6. [ ] Update all `json.Marshal(msg)` calls to use A2AEnvelope
7. [ ] Remove deprecated `ACPMessage` type entirely

**Validation:**
- [ ] All agents can register and discover
- [ ] Task lifecycle works end-to-end
- [ ] Approval flow functions correctly
- [ ] WebSocket UI receives properly formatted messages
- [ ] No compilation warnings about ACPMessage

### Phase 2.1.2: Transport Layer Update (1-2 hours)
**Files to modify:**
- `main.go` (UDS server)
- `main.go` (WebSocket handler)

**Tasks:**
1. [ ] Update UDS scanner to parse A2AEnvelope directly
2. [ ] Remove ACPMessage→A2AEnvelope conversion in `dispatchMessage`
3. [ ] Update WebSocket handler to send/receive A2AEnvelope
4. [ ] Add envelope validation:
   ```go
   func ValidateEnvelope(env A2AEnvelope) error {
       if env.JSONRPC != "2.0" { return errors.New("invalid JSONRPC version") }
       if env.Method == "" { return errors.New("method required") }
       if env.Sender == "" { return errors.New("sender required") }
       return nil
   }
   ```

**Validation:**
- [ ] Wire protocol is pure A2AEnvelope (no legacy cruft)
- [ ] WebSocket clients send/receive native A2A
- [ ] Message tracing shows clean A2A messages

### Phase 2.1.3: Test Suite Update (1 hour)
**Files to modify:**
- `pkg/a2a/types_test.go`
- New: `pkg/harness/taskstore_test.go`

**Tasks:**
1. [ ] Remove ACPMessage conversion tests
2. [ ] Add A2AEnvelope validation tests
3. [ ] Add end-to-end task lifecycle tests
4. [ ] Add concurrent task handling tests
5. [ ] Add approval flow tests

---

## Wave 2.2: Security Hardening

**Goal:** Implement zero-trust security model with authentication, encryption, and RBAC.

### Phase 2.2.1: JWT Agent Authentication (3-4 hours)
**New files:**
- `pkg/security/jwt.go`
- `pkg/security/jwt_test.go`

**Tasks:**
1. [ ] Implement JWT token generation:
   ```go
   type AgentClaims struct {
       AgentID      string   `json:"agent_id"`
       Capabilities []string `json:"capabilities"`
       Skills       []string `json:"skills"`
       jwt.StandardClaims
   }
   ```
2. [ ] Add token signing/verification with RS256
3. [ ] Generate agent keypairs on registration
4. [ ] Store public keys in TaskStore
5. [ ] Add `Authorization: Bearer <token>` to A2AEnvelope metadata
6. [ ] Verify token on every incoming message in harness
7. [ ] Reject unauthenticated messages

**Files to modify:**
- `pkg/harness/taskstore.go` - add public key storage
- `main.go` - add token verification middleware
- `agent/worker/main.go` - add token to outgoing messages

**Validation:**
- [ ] Agents can authenticate with valid tokens
- [ ] Invalid tokens are rejected
- [ ] Expired tokens are rejected
- [ ] Token validation doesn't impact performance (<1ms overhead)

### Phase 2.2.2: TLS Encryption (2 hours)
**New files:**
- `pkg/transport/tls.go`
- `certs/` directory with generated certs

**Tasks:**
1. [ ] Generate self-signed CA for local mesh
2. [ ] Generate server cert for harness
3. [ ] Generate client certs for each agent
4. [ ] Wrap UDS with TLS (mtls over UDS)
5. [ ] Wrap WebSocket with TLS (wss://)
6. [ ] Add cert rotation mechanism

**Files to modify:**
- `main.go` - enable TLS for UDS and WebSocket
- `scripts/bootstrap_node.sh` - generate certs on node setup

**Validation:**
- [ ] All connections use TLS
- [ ] Certificate validation works
- [ ] No performance degradation (<5ms TLS handshake)

### Phase 2.2.3: RBAC Implementation (3-4 hours)
**New files:**
- `pkg/security/rbac.go`
- `pkg/security/rbac_test.go`
- `config/policies.yaml`

**Tasks:**
1. [ ] Define permission model:
   ```go
   type Permission struct {
       Resource string // "tasks", "agents", "tools"
       Action   string // "read", "write", "execute"
   }
   type Role struct {
       Name        string
       Permissions []Permission
   }
   ```
2. [ ] Implement policy engine (OPA-style)
3. [ ] Add role assignment to agent registration
4. [ ] Enforce permissions on every operation
5. [ ] Add audit logging for RBAC decisions

**Files to modify:**
- `pkg/harness/taskstore.go` - add role storage
- `main.go` - add RBAC middleware for all handlers

**Example policies:**
```yaml
roles:
  - name: worker
    permissions:
      - resource: tasks
        action: execute
      - resource: tools
        action: call
  - name: manager
    permissions:
      - resource: tasks
        action: "*"
      - resource: agents
        action: read
  - name: admin
    permissions:
      - resource: "*"
        action: "*"
```

**Validation:**
- [ ] Workers can execute tasks but not manage agents
- [ ] Managers can orchestrate but not execute shell commands
- [ ] Admin can perform all operations
- [ ] Unauthorized operations are blocked and logged

---

## Wave 2.3: Multi-Agent Coordination

**Goal:** Advanced agent discovery, health checks, load balancing, and lifecycle management.

### Phase 2.3.1: Agent Discovery Service (2-3 hours)
**New files:**
- `pkg/discovery/registry.go`
- `pkg/discovery/registry_test.go`

**Tasks:**
1. [ ] Extract agent discovery from TaskStore into dedicated service
2. [ ] Add agent metadata enrichment (hostname, IP, resources)
3. [ ] Implement capability queries:
   ```go
   func (r *Registry) FindAgents(query CapabilityQuery) []AgentCard
   ```
4. [ ] Add agent versioning and compatibility checks
5. [ ] Implement agent groups/pools for workload distribution

**Files to modify:**
- `pkg/harness/taskstore.go` - delegate discovery to Registry
- `main.go` - use Registry for agent routing

**Validation:**
- [ ] Agents can discover peers by skill/capability
- [ ] Query performance <10ms for 100+ agents
- [ ] Agent metadata is accurately maintained

### Phase 2.3.2: Health Checks & Heartbeat (2 hours)
**New files:**
- `pkg/health/monitor.go`

**Tasks:**
1. [ ] Add heartbeat message (`agent/heartbeat`)
2. [ ] Implement health check loop in harness:
   ```go
   func (h *HealthMonitor) CheckHealth(agentID string) AgentHealth
   ```
3. [ ] Mark agents as unhealthy after missed heartbeats
4. [ ] Auto-remove stale agents from registry
5. [ ] Broadcast health status changes to UI

**Files to modify:**
- `main.go` - add health monitor goroutine
- `agent/worker/main.go` - send heartbeats every 5s

**Validation:**
- [ ] Healthy agents are monitored
- [ ] Unhealthy agents are detected within 15s
- [ ] Stale agents are removed from routing

### Phase 2.3.3: Load Balancing & Agent Pools (3 hours)
**New files:**
- `pkg/routing/loadbalancer.go`

**Tasks:**
1. [ ] Enhance round-robin with weighted routing
2. [ ] Add least-connections routing
3. [ ] Add resource-aware routing (CPU, memory)
4. [ ] Implement agent pools (e.g., "high-memory", "gpu-enabled")
5. [ ] Add routing metrics (task distribution, latency per agent)

**Files to modify:**
- `pkg/harness/taskstore.go` - use LoadBalancer for skill routing

**Validation:**
- [ ] Tasks are evenly distributed across agents
- [ ] High-load agents receive fewer tasks
- [ ] Pool-specific routing works (e.g., GPU tasks → GPU agents)

### Phase 2.3.4: Agent Lifecycle Management (2-3 hours)
**New files:**
- `pkg/lifecycle/manager.go`

**Tasks:**
1. [ ] Implement agent spawn API:
   ```go
   func (m *Manager) SpawnAgent(config AgentConfig) (string, error)
   ```
2. [ ] Add graceful shutdown protocol
3. [ ] Implement agent restart on failure
4. [ ] Add agent scaling policies (min/max instances)
5. [ ] Support agent upgrades (blue-green deployment)

**Files to modify:**
- `main.go` - replace hardcoded spawnAgent with lifecycle API

**Validation:**
- [ ] Agents can be spawned dynamically
- [ ] Failed agents are automatically restarted
- [ ] Agents shut down gracefully (finish pending tasks)

---

## Wave 2.4: Advanced Task Orchestration

**Goal:** Support complex workflows with dependencies, DAGs, scheduling, and retry policies.

### Phase 2.4.1: Task Dependencies & DAG (4-5 hours)
**New files:**
- `pkg/orchestration/dag.go`
- `pkg/orchestration/dag_test.go`

**Tasks:**
1. [ ] Extend Task model with dependencies:
   ```go
   type Task struct {
       ...
       Dependencies []string `json:"dependencies"` // task IDs
       DependencyPolicy string `json:"dependencyPolicy"` // "all", "any"
   }
   ```
2. [ ] Implement DAG builder and validator (detect cycles)
3. [ ] Add dependency resolution engine
4. [ ] Execute tasks in dependency order
5. [ ] Handle partial failures (continue/abort)

**Files to modify:**
- `pkg/a2a/types.go` - add dependency fields
- `pkg/harness/taskstore.go` - check deps before task execution

**Example DAG:**
```go
workflow := &DAG{
    Tasks: []Task{
        {ID: "fetch_data", ...},
        {ID: "process", Dependencies: ["fetch_data"]},
        {ID: "save_results", Dependencies: ["process"]},
    },
}
```

**Validation:**
- [ ] DAG is validated before execution
- [ ] Tasks execute in correct order
- [ ] Dependency failures block dependent tasks
- [ ] Partial workflows complete successfully

### Phase 2.4.2: Task Scheduling & Queues (3 hours)
**New files:**
- `pkg/orchestration/scheduler.go`
- `pkg/orchestration/queue.go`

**Tasks:**
1. [ ] Implement priority queue for tasks
2. [ ] Add scheduling policies:
   - FIFO (first-in-first-out)
   - Priority-based
   - Deadline-based (EDF - earliest deadline first)
   - Fair-share (per-agent quota)
3. [ ] Add task backlog monitoring
4. [ ] Implement task preemption for critical tasks

**Files to modify:**
- `pkg/harness/taskstore.go` - use scheduler for task dispatch

**Validation:**
- [ ] High-priority tasks execute first
- [ ] Deadline-bound tasks meet deadlines
- [ ] Fair-share prevents agent starvation

### Phase 2.4.3: Workflow Engine (4-5 hours)
**New files:**
- `pkg/orchestration/workflow.go`
- `pkg/orchestration/workflow_test.go`

**Tasks:**
1. [ ] Define workflow DSL:
   ```yaml
   workflow:
     name: data_pipeline
     steps:
       - id: extract
         agent: worker
         tool: execute_shell
       - id: transform
         agent: processor
         depends_on: extract
       - id: load
         agent: db_writer
         depends_on: transform
   ```
2. [ ] Implement workflow parser
3. [ ] Add parallel execution support (fan-out/fan-in)
4. [ ] Support conditional branches (if/else)
5. [ ] Add loops (while, for-each)
6. [ ] Implement workflow templates

**Validation:**
- [ ] Complex workflows execute correctly
- [ ] Parallel branches work
- [ ] Conditionals route properly
- [ ] Workflows can be saved and reused

### Phase 2.4.4: Retry & Failure Handling (2-3 hours)
**New files:**
- `pkg/orchestration/retry.go`

**Tasks:**
1. [ ] Add retry policy to tasks:
   ```go
   type RetryPolicy struct {
       MaxAttempts int
       BackoffType string // "exponential", "linear", "fixed"
       BaseDelay   time.Duration
   }
   ```
2. [ ] Implement exponential backoff
3. [ ] Add circuit breaker pattern
4. [ ] Support compensating transactions (rollback)
5. [ ] Add dead-letter queue for failed tasks

**Files to modify:**
- `pkg/a2a/types.go` - add retry policy fields
- `pkg/harness/taskstore.go` - implement retry logic

**Validation:**
- [ ] Transient failures are retried
- [ ] Exponential backoff reduces load
- [ ] Circuit breaker prevents cascade failures
- [ ] Persistent failures go to DLQ

---

## Wave 2.5: Enhanced Observability

**Goal:** Production-grade monitoring with Prometheus metrics, structured logging, and distributed tracing.

### Phase 2.5.1: Prometheus Metrics (3 hours)
**New files:**
- `pkg/observability/metrics.go`
- `pkg/observability/metrics_test.go`

**Tasks:**
1. [ ] Add Prometheus client library
2. [ ] Define key metrics:
   ```go
   var (
       TasksCreated = prometheus.NewCounterVec(...)
       TaskDuration = prometheus.NewHistogramVec(...)
       ActiveAgents = prometheus.NewGauge(...)
       MessagesSent = prometheus.NewCounterVec(...)
   )
   ```
3. [ ] Instrument harness and agents
4. [ ] Expose `/metrics` endpoint
5. [ ] Create Prometheus scrape config

**Metrics to track:**
- Task throughput (tasks/sec)
- Task latency (p50, p95, p99)
- Active agents
- Messages sent/received
- Error rates
- Agent health status
- Queue backlog

**Files to modify:**
- `main.go` - add metrics endpoint
- `pkg/harness/taskstore.go` - instrument operations

**Validation:**
- [ ] Metrics endpoint returns valid Prometheus format
- [ ] Metrics update in real-time
- [ ] No performance impact (<0.1ms per metric)

### Phase 2.5.2: Structured Logging (2-3 hours)
**New files:**
- `pkg/observability/logger.go`

**Tasks:**
1. [ ] Replace `log.Printf` with structured logger (zerolog or zap)
2. [ ] Add log levels (DEBUG, INFO, WARN, ERROR)
3. [ ] Add context propagation:
   ```go
   log.Info().
       Str("agent_id", agentID).
       Str("task_id", taskID).
       Str("trace_id", traceID).
       Msg("Task started")
   ```
4. [ ] Output JSON logs for machine parsing
5. [ ] Add log sampling for high-volume logs

**Files to modify:**
- All files using `log.Printf` or `fmt.Printf`

**Validation:**
- [ ] Logs are JSON formatted
- [ ] Trace IDs appear in all related logs
- [ ] Log volume is manageable (sampling works)

### Phase 2.5.3: Distributed Tracing (2 hours)
**New files:**
- `pkg/observability/tracing.go`

**Tasks:**
1. [ ] Complete OpenTelemetry integration
2. [ ] Add span creation for all operations:
   ```go
   ctx, span := tracer.Start(ctx, "task-execution")
   defer span.End()
   ```
3. [ ] Propagate trace context in A2AEnvelope metadata
4. [ ] Export spans to Jaeger
5. [ ] Create trace visualization

**Files to modify:**
- `main.go` - configure OTel exporter
- All message handlers - add span instrumentation

**Validation:**
- [ ] End-to-end traces visible in Jaeger
- [ ] Trace context propagates across agents
- [ ] Span durations are accurate

### Phase 2.5.4: Dashboards (3 hours)
**New files:**
- `config/grafana/dashboard.json`
- `config/prometheus/alerts.yml`

**Tasks:**
1. [ ] Create Grafana dashboard with key metrics:
   - Task throughput over time
   - Task latency percentiles
   - Active agents gauge
   - Error rate over time
   - Queue backlog
2. [ ] Add alerting rules:
   ```yaml
   - alert: HighErrorRate
     expr: rate(tasks_failed[5m]) > 0.1
     for: 5m
   ```
3. [ ] Create war-room panel (dark theme, cinematic aesthetic)

**Validation:**
- [ ] Dashboard shows real-time metrics
- [ ] Alerts fire correctly
- [ ] War-room aesthetic matches project vision

---

## Wave 2.6: Performance Optimization

**Goal:** Optimize for high throughput and low latency at scale.

### Phase 2.6.1: Connection Pooling (2-3 hours)
**New files:**
- `pkg/transport/pool.go`

**Tasks:**
1. [ ] Implement connection pool for UDS:
   ```go
   type ConnectionPool struct {
       conns chan net.Conn
       size  int
   }
   ```
2. [ ] Add pool size configuration
3. [ ] Implement connection health checks
4. [ ] Add pool metrics (utilization, wait time)

**Files to modify:**
- `main.go` - use pools instead of dedicated channels

**Validation:**
- [ ] Connection reuse works correctly
- [ ] No connection leaks
- [ ] Performance improves (30-50% higher throughput)

### Phase 2.6.2: Message Batching (2 hours)
**New files:**
- `pkg/transport/batcher.go`

**Tasks:**
1. [ ] Implement message batching:
   ```go
   type Batcher struct {
       buffer   []A2AEnvelope
       maxSize  int
       maxDelay time.Duration
   }
   ```
2. [ ] Batch outbound messages (flush on size or timeout)
3. [ ] Add batch compression (gzip for large payloads)

**Files to modify:**
- `pkg/harness/taskstore.go` - batch updates to agents

**Validation:**
- [ ] Message batches are sent efficiently
- [ ] Latency doesn't increase significantly (<50ms delay)
- [ ] Network traffic reduced (40-60% fewer syscalls)

### Phase 2.6.3: Async Processing (3 hours)
**New files:**
- `pkg/async/worker_pool.go`

**Tasks:**
1. [ ] Implement worker pool for async operations:
   ```go
   type WorkerPool struct {
       workers  int
       taskChan chan func()
   }
   ```
2. [ ] Process messages asynchronously
3. [ ] Add bounded queues to prevent overload
4. [ ] Implement backpressure signaling

**Files to modify:**
- `main.go` - use worker pool for message handling

**Validation:**
- [ ] Non-blocking message processing
- [ ] Backpressure prevents OOM
- [ ] Throughput increases (2-3x)

### Phase 2.6.4: Caching (2 hours)
**New files:**
- `pkg/cache/lru.go`

**Tasks:**
1. [ ] Implement LRU cache for agent cards
2. [ ] Cache skill→agent mappings
3. [ ] Add cache invalidation on agent updates
4. [ ] Add cache metrics (hit rate, size)

**Files to modify:**
- `pkg/harness/taskstore.go` - add caching layer

**Validation:**
- [ ] Cache hit rate >80% for steady-state traffic
- [ ] Cache invalidation works correctly
- [ ] Lookup latency reduced (10-50x faster)

### Phase 2.6.5: Serialization Optimization (2-3 hours)
**New files:**
- `pkg/serial/msgpack.go`

**Tasks:**
1. [ ] Add MessagePack support as alternative to JSON
2. [ ] Benchmark JSON vs MessagePack
3. [ ] Make serialization format configurable
4. [ ] Add content negotiation for wire format

**Files to modify:**
- `pkg/a2a/types.go` - add msgpack tags
- Transport layers - support both formats

**Validation:**
- [ ] MessagePack is 30-50% faster than JSON
- [ ] Both formats work correctly
- [ ] No data loss in conversion

---

## Implementation Order & Timeline

### Week 1: Core Migration & Security Foundation
- Day 1-2: Wave 2.1 (Full A2A migration)
- Day 3-4: Wave 2.2.1 (JWT authentication)
- Day 5: Wave 2.2.2 (TLS encryption)

### Week 2: Security & Multi-Agent
- Day 1-2: Wave 2.2.3 (RBAC)
- Day 3: Wave 2.3.1 (Discovery service)
- Day 4: Wave 2.3.2 (Health checks)
- Day 5: Wave 2.3.3 (Load balancing)

### Week 3: Orchestration
- Day 1-2: Wave 2.4.1 (DAG dependencies)
- Day 3: Wave 2.4.2 (Scheduling)
- Day 4-5: Wave 2.4.3 (Workflow engine)

### Week 4: Observability & Performance
- Day 1-2: Wave 2.5.1-2 (Prometheus + structured logging)
- Day 3: Wave 2.5.3-4 (Tracing + dashboards)
- Day 4-5: Wave 2.6 (Performance optimization)

**Total Estimated Time:** 4 weeks (80-100 hours)

---

## Success Criteria

### Technical Milestones
- [ ] 100% A2A native (zero ACPMessage usage)
- [ ] All connections authenticated and encrypted
- [ ] RBAC enforced on all operations
- [ ] Workflows with 10+ tasks execute correctly
- [ ] 1000+ tasks/sec throughput
- [ ] P99 latency <100ms
- [ ] Prometheus metrics exposed
- [ ] Distributed traces visible in Jaeger
- [ ] Zero downtime agent restarts

### Production Readiness
- [ ] All components have >80% test coverage
- [ ] Documentation updated
- [ ] Docker Compose for local dev environment
- [ ] Kubernetes manifests for production deployment
- [ ] CI/CD pipeline configured
- [ ] Load testing validates performance targets
- [ ] Security audit completed

---

## Risk Mitigation

### High-Risk Areas
1. **Breaking changes during A2A migration**
   - Mitigation: Feature flags, gradual rollout, comprehensive testing
2. **Performance regression from security layers**
   - Mitigation: Benchmark before/after, optimize hot paths
3. **Complexity explosion in orchestration**
   - Mitigation: Incremental features, keep simple cases simple

### Rollback Strategy
- Maintain backward compatibility during Wave 2.1
- Use feature flags for new capabilities
- Git tags for each milestone
- Quick rollback procedure documented

---

## Next Steps

**Immediate action:** Start Wave 2.1.1 (Core Type Migration)

1. Review this plan with team
2. Set up development environment
3. Create feature branch: `wave2-migration`
4. Begin refactoring pkg/a2a/types.go
5. Run test suite after each change
6. Commit frequently with descriptive messages

**Questions to resolve:**
- Which serialization format for production? (JSON vs MessagePack)
- Observability backend? (Prometheus + Grafana vs DataDog)
- Workflow DSL format? (YAML vs JSON vs protobuf)
- Certificate management? (cert-manager vs manual)
