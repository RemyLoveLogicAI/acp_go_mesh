// Package harness provides the A2A task broker and agent card registry.
package harness

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"acp-mesh/pkg/a2a"
)

// TaskStore holds all active tasks and provides pub/sub for status updates.
type TaskStore struct {
	mu          sync.RWMutex
	tasks       map[string]*a2a.StateMachine
	subs        map[string][]chan a2a.TaskUpdate // taskID -> subscriber channels
	agents      map[string]a2a.AgentCard         // agentID -> card
	agentConns  map[string]chan []byte         // agentID -> outbound message channel
	sessions    map[string]*a2a.Session        // sessionID -> session
	skillRR     map[string]int                   // skillID -> round-robin index
	skillsIndex map[string][]string            // skillID -> list of agentIDs
	cancelChans map[string]chan struct{}     // taskID -> cancel signal channel
}
// NewTaskStore creates an empty task store.
func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks:       make(map[string]*a2a.StateMachine),
		subs:        make(map[string][]chan a2a.TaskUpdate),
		agents:      make(map[string]a2a.AgentCard),
		agentConns:  make(map[string]chan []byte),
		sessions:    make(map[string]*a2a.Session),
		skillRR:     make(map[string]int),
		skillsIndex: make(map[string][]string),
		cancelChans: make(map[string]chan struct{}),
	}
}



// RegisterAgentCard stores an agent's discovery manifest and indexes its skills for fast O(1) discovery.
func (ts *TaskStore) RegisterAgentCard(agentID string, card a2a.AgentCard) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Clean up any existing index mapping for this agent (supporting clean re-registration)
	for skillID, list := range ts.skillsIndex {
		var newList []string
		for _, aid := range list {
			if aid != agentID {
				newList = append(newList, aid)
			}
		}
		if len(newList) == 0 {
			delete(ts.skillsIndex, skillID)
		} else {
			ts.skillsIndex[skillID] = newList
		}
	}

	ts.agents[agentID] = card

	// Index new skills
	for _, s := range card.Skills {
		ts.skillsIndex[s.ID] = append(ts.skillsIndex[s.ID], agentID)
	}
	// Index new legacy caps
	for _, c := range card.LegacyCaps {
		ts.skillsIndex[c] = append(ts.skillsIndex[c], agentID)
	}
}

// GetAgentCard returns the registered card for an agent.
func (ts *TaskStore) GetAgentCard(agentID string) (a2a.AgentCard, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	c, ok := ts.agents[agentID]
	return c, ok
}

// FindAgentBySkill resolves an agent ID that supports the requested skill ID.
// Performs a high-performance O(1) index lookup with thread-safe round-robin load balancing.
func (ts *TaskStore) FindAgentBySkill(skillID string) (string, bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	list, ok := ts.skillsIndex[skillID]
	if !ok || len(list) == 0 {
		return "", false
	}
	idx := ts.skillRR[skillID]
	if idx >= len(list) {
		idx = 0
	}
	agentID := list[idx]
	ts.skillRR[skillID] = (idx + 1) % len(list)
	return agentID, true
}



// getOrCreateSessionLocked retrieves an existing session or initializes a new one.
// Must be called with ts.mu lock held.
func (ts *TaskStore) getOrCreateSessionLocked(sessionID string) *a2a.Session {
	if s, ok := ts.sessions[sessionID]; ok {
		s.LastSeen = time.Now().UTC()
		return s
	}
	s := &a2a.Session{
		ID:        sessionID,
		CreatedAt: time.Now().UTC(),
		LastSeen:  time.Now().UTC(),
	}
	ts.sessions[sessionID] = s
	return s
}

// GetOrCreateSession retrieves an existing session or initializes a new one.
func (ts *TaskStore) GetOrCreateSession(sessionID string) *a2a.Session {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.getOrCreateSessionLocked(sessionID)
}

// AppendSessionMessage appends a message to a session's history.
func (ts *TaskStore) AppendSessionMessage(sessionID string, msg a2a.Message) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	s := ts.getOrCreateSessionLocked(sessionID)
	s.Messages = append(s.Messages, msg)
	s.LastSeen = time.Now().UTC()
}


// GetSession returns a session by ID.
func (ts *TaskStore) GetSession(sessionID string) (*a2a.Session, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	s, ok := ts.sessions[sessionID]
	return s, ok
}

// ListSessions returns a copy of all active sessions.
func (ts *TaskStore) ListSessions() map[string]a2a.Session {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	out := make(map[string]a2a.Session, len(ts.sessions))
	for k, v := range ts.sessions {
		if v != nil {
			out[k] = *v
		}
	}
	return out
}

// GetSessionRecentUpdates returns the last N messages across all active sessions, formatted as UI update payloads.
func (ts *TaskStore) GetSessionRecentUpdates(limit int) []map[string]interface{} {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	var updates []map[string]interface{}
	for _, sess := range ts.sessions {
		if sess == nil {
			continue
		}
		msgs := sess.Messages
		start := 0
		if len(msgs) > limit {
			start = len(msgs) - limit
		}
		for i := start; i < len(msgs); i++ {
			msg := msgs[i]
			updates = append(updates, map[string]interface{}{
				"type":       "session_update",
				"session_id": sess.ID,
				"message":    msg,
			})
		}
	}
	return updates
}




// ListAgentCards returns all registered agent cards.
func (ts *TaskStore) ListAgentCards() map[string]a2a.AgentCard {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	out := make(map[string]a2a.AgentCard, len(ts.agents))
	for k, v := range ts.agents {
		out[k] = v
	}
	return out
}

// ListTasks returns all active task state machines.
func (ts *TaskStore) ListTasks() map[string]*a2a.StateMachine {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	out := make(map[string]*a2a.StateMachine, len(ts.tasks))
	for k, v := range ts.tasks {
		out[k] = v
	}
	return out
}

// RegisterAgentConn registers an outbound message channel for an agent.
func (ts *TaskStore) RegisterAgentConn(agentID string, ch chan []byte) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.agentConns[agentID] = ch
}

// UnregisterAgent removes an agent's UDS connection and dynamic advertised capabilities from the registry.
func (ts *TaskStore) UnregisterAgent(agentID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.agentConns, agentID)
	delete(ts.agents, agentID)

	// Clean up skillsIndex mapping for this agent
	for skillID, list := range ts.skillsIndex {
		var newList []string
		for _, aid := range list {
			if aid != agentID {
				newList = append(newList, aid)
			}
		}
		if len(newList) == 0 {
			delete(ts.skillsIndex, skillID)
		} else {
			ts.skillsIndex[skillID] = newList
		}
	}
}



// SendToAgent delivers a raw JSON line to an agent's outbound channel.
func (ts *TaskStore) SendToAgent(agentID string, payload []byte) bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	ch, ok := ts.agentConns[agentID]
	if !ok {
		return false
	}
	select {
	case ch <- payload:
		return true
	default:
		return false
	}
}

// CreateTask initializes a new task and its state machine.
func (ts *TaskStore) CreateTask(task a2a.Task) (*a2a.StateMachine, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if _, exists := ts.tasks[task.ID]; exists {
		return nil, fmt.Errorf("task %s already exists", task.ID)
	}
	if task.Status.State == "" {
		task.Status = a2a.NewTaskStatus(a2a.TaskStateSubmitted, "Task created")
	}
	sm := a2a.NewStateMachine(task)
	ts.tasks[task.ID] = sm
	return sm, nil
}

// GetTask returns the state machine for a task.
func (ts *TaskStore) GetTask(taskID string) (*a2a.StateMachine, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	sm, ok := ts.tasks[taskID]
	return sm, ok
}

// TransitionTask attempts a state transition on a task.
// If successful, it broadcasts the update to all subscribers.
func (ts *TaskStore) TransitionTask(taskID string, newState a2a.TaskState, message string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	sm, ok := ts.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}
	if err := sm.Transition(newState, message); err != nil {
		return err
	}

	update := a2a.TaskUpdate{
		TaskID:    taskID,
		Status:    sm.Task().Status,
		Timestamp: time.Now().UTC(),
		Artifacts: sm.Task().Artifacts,
		History:   sm.History(),
	}
	// Populate trace context from metadata if present
	if meta := sm.Task().Metadata; meta != nil {
		if tid, ok := meta["trace_id"].(string); ok {
			update.TraceID = tid
		}
		if sid, ok := meta["span_id"].(string); ok {
			update.SpanID = sid
		}
	}
	// Broadcast to subscribers
	for _, ch := range ts.subs[taskID] {
		select {
		case ch <- update:
		default:
		}
	}
	return nil
}

// Subscribe returns a channel that receives updates for a given task.
func (ts *TaskStore) Subscribe(taskID string) <-chan a2a.TaskUpdate {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ch := make(chan a2a.TaskUpdate, 16)
	ts.subs[taskID] = append(ts.subs[taskID], ch)
	return ch
}

// Unsubscribe removes a channel from a task's subscriber list.
func (ts *TaskStore) Unsubscribe(taskID string, ch <-chan a2a.TaskUpdate) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	arr, exists := ts.subs[taskID]
	if !exists || len(arr) == 0 {
		return
	}
	foundIdx := -1
	for i, c := range arr {
		if c == ch {
			foundIdx = i
			break
		}
	}
	if foundIdx != -1 {
		newList := make([]chan a2a.TaskUpdate, 0, len(arr)-1)
		for i, c := range arr {
			if i != foundIdx {
				newList = append(newList, c)
			}
		}
		if len(newList) == 0 {
			delete(ts.subs, taskID)
		} else {
			ts.subs[taskID] = newList
		}
		close(arr[foundIdx])
	}
}


// BroadcastTaskUpdate sends a task update to all subscribers and the original sender/requester.
func (ts *TaskStore) BroadcastTaskUpdate(update a2a.TaskUpdate, senderAgentID, requesterAgentID string) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	// Notify subscribers
	for _, ch := range ts.subs[update.TaskID] {
		select {
		case ch <- update:
		default:
		}
	}

	// Also notify the requester agent if they're not the sender
	if requesterAgentID != "" && requesterAgentID != senderAgentID {
		if ch, ok := ts.agentConns[requesterAgentID]; ok {
			env := a2a.A2AEnvelope{
				JSONRPC: "2.0",
				Method:  "tasks/sendUpdate",
				Sender:  "harness",
				Target:  requesterAgentID,
				TaskID:  update.TaskID,
			}
			payload, _ := json.Marshal(map[string]interface{}{
				"status": update.Status,
			})
			env.Payload = payload
			b, _ := json.Marshal(env)
			select {
			case ch <- append(b, '\n'):
			default:
			}
		}
	}
}

// IsTaskTerminal reports whether a task has reached a terminal state.
func (ts *TaskStore) IsTaskTerminal(taskID string) bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	sm, ok := ts.tasks[taskID]
	if !ok {
		return false
	}
	return sm.IsTerminal()
}

// RegisterCancel creates a cancel channel for a task and returns it.
func (ts *TaskStore) RegisterCancel(taskID string) chan struct{} {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ch := make(chan struct{})
	ts.cancelChans[taskID] = ch
	return ch
}

// GetCancel returns the cancel channel for a task if it exists.
func (ts *TaskStore) GetCancel(taskID string) (chan struct{}, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	ch, ok := ts.cancelChans[taskID]
	return ch, ok
}

// CancelTask transitions a task to canceling and signals the cancel channel.
func (ts *TaskStore) CancelTask(taskID string) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	sm, ok := ts.tasks[taskID]
	if !ok || sm.IsTerminal() {
		return false
	}
	if err := sm.Transition(a2a.TaskStateCanceling, "Cancellation requested"); err != nil {
		return false
	}
	if ch, ok := ts.cancelChans[taskID]; ok {
		close(ch)
		delete(ts.cancelChans, taskID)
	}
	return true
}
