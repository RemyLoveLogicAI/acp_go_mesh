package a2a

import (
	"fmt"
	"sync"
)

// StateMachine tracks a single task's lifecycle and validates transitions.
type StateMachine struct {
	mu          sync.RWMutex
	task        Task
	transitions map[TaskState][]TaskState // valid next states from current
	history     []TaskStatus
}

// DefaultTransitions defines the valid A2A state transitions.
var DefaultTransitions = map[TaskState][]TaskState{
	TaskStateSubmitted:     {TaskStateWorking, TaskStateInputRequired, TaskStateCanceled},
	TaskStateWorking:       {TaskStateInputRequired, TaskStateCompleted, TaskStateFailed, TaskStateCanceled},
	TaskStateInputRequired: {TaskStateWorking, TaskStateCanceled},
	TaskStateCompleted:     {},
	TaskStateFailed:        {},
	TaskStateCanceled:      {},
}

// NewStateMachine creates a state machine for a task starting at Submitted.
func NewStateMachine(task Task) *StateMachine {
	if task.Status.State == "" {
		task.Status = NewTaskStatus(TaskStateSubmitted, "Task created")
	}
	return &StateMachine{
		task:        task,
		transitions: DefaultTransitions,
		history:     []TaskStatus{},
	}
}

// Task returns a copy of the current task state.
func (sm *StateMachine) Task() Task {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.task
}

// CurrentState returns the current task state.
func (sm *StateMachine) CurrentState() TaskState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.task.Status.State
}

// Transition attempts to move the task to a new state.
// Returns an error if the transition is invalid.
func (sm *StateMachine) Transition(newState TaskState, message string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	current := sm.task.Status.State
	validNext, ok := sm.transitions[current]
	if !ok {
		return fmt.Errorf("invalid current state %q", current)
	}

	allowed := false
	for _, s := range validNext {
		if s == newState {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("invalid transition %q -> %q", current, newState)
	}

	if message == "" {
		message = fmt.Sprintf("Transitioned from %s to %s", current, newState)
	}

	// Record the previous state in history before updating
	sm.history = append(sm.history, sm.task.Status)
	sm.task.Status = NewTaskStatus(newState, message)
	return nil
}

// History returns a copy of the task's state transition history.
func (sm *StateMachine) History() []TaskStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make([]TaskStatus, len(sm.history))
	copy(out, sm.history)
	return out
}

// CanTransition checks if a transition is allowed without executing it.
func (sm *StateMachine) CanTransition(newState TaskState) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	validNext, ok := sm.transitions[sm.task.Status.State]
	if !ok {
		return false
	}
	for _, s := range validNext {
		if s == newState {
			return true
		}
	}
	return false
}

// IsTerminal returns true if the task has reached a terminal state.
func (sm *StateMachine) IsTerminal() bool {
	s := sm.CurrentState()
	return s == TaskStateCompleted || s == TaskStateFailed || s == TaskStateCanceled
}

// SetArtifact appends an artifact to the task.
func (sm *StateMachine) SetArtifact(a Artifact) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.task.Artifacts = append(sm.task.Artifacts, a)
}

// SetMetadata stores a key-value pair in task metadata.
func (sm *StateMachine) SetMetadata(key string, value interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.task.Metadata == nil {
		sm.task.Metadata = make(map[string]interface{})
	}
	sm.task.Metadata[key] = value
}
