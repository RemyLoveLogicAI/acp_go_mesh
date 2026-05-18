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
	mu       sync.RWMutex
	tasks    map[string]*a2a.StateMachine
	subs     map[string][]chan a2a.TaskUpdate // taskID -> subscriber channels
	agents   map[string]a2a.AgentCard         // agentID -> card
	agentConns map[string]chan []byte         // agentID -> outbound message channel
	sessions   map[string]*a2a.Session        // sessionID -> session
}

// NewTaskStore creates an empty task store.
func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks:      make(map[string]*a2a.StateMachine),
		subs:       make(map[string][]chan a2a.TaskUpdate),
		agents:     make(map[string]a2a.AgentCard),
		agentConns: make(map[string]chan []byte),
		sessions:   make(map[string]*a2a.Session),
	}
}

// RegisterAgentCard stores an agent's discovery manifest.
func (ts *TaskStore) RegisterAgentCard(agentID string, card a2a.AgentCard) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.agents[agentID] = card
}

// GetAgentCard returns the registered card for an agent.
func (ts *TaskStore) GetAgentCard(agentID string) (a2a.AgentCard, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	c, ok := ts.agents[agentID]
	return c, ok
}

// FindAgentBySkill resolves an agent ID that supports the requested skill ID.
// Supports matching against A2A structured Skills and falls back to LegacyCaps.
func (ts *TaskStore) FindAgentBySkill(skillID string) (string, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	for agentID, card := range ts.agents {
		for _, s := range card.Skills {
			if s.ID == skillID {
				return agentID, true
			}
		}
		for _, legacyCap := range card.LegacyCaps {
			if legacyCap == skillID {
				return agentID, true
			}
		}
	}
	return "", false
}

// GetOrCreateSession retrieves an existing session or initializes a new one.
func (ts *TaskStore) GetOrCreateSession(sessionID string) *a2a.Session {
	ts.mu.Lock()
	defer ts.mu.Unlock()
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

// AppendSessionMessage appends a message to a session's history.
func (ts *TaskStore) AppendSessionMessage(sessionID string, msg a2a.Message) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	s, ok := ts.sessions[sessionID]
	if !ok {
		s = &a2a.Session{
			ID:        sessionID,
			CreatedAt: time.Now().UTC(),
			LastSeen:  time.Now().UTC(),
		}
		ts.sessions[sessionID] = s
	}
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

// RegisterAgentConn registers an outbound message channel for an agent.
func (ts *TaskStore) RegisterAgentConn(agentID string, ch chan []byte) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.agentConns[agentID] = ch
}

// UnregisterAgentConn removes an agent's outbound channel.
func (ts *TaskStore) UnregisterAgentConn(agentID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.agentConns, agentID)
	delete(ts.agents, agentID)
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
	arr := ts.subs[taskID]
	for i, c := range arr {
		if c == ch {
			ts.subs[taskID] = append(arr[:i], arr[i+1:]...)
			close(c)
			break
		}
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
