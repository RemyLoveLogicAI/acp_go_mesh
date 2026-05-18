// Package a2a implements the Agent-to-Agent (A2A) protocol types.
// Aligned with the Linux Foundation A2A specification (May 2025).
// https://github.com/a2aproject/a2a-spec
package a2a

import (
	"encoding/json"
	"time"
)

// TaskState represents the lifecycle state of a task.
type TaskState string

const (
	TaskStateSubmitted     TaskState = "submitted"
	TaskStateWorking       TaskState = "working"
	TaskStateInputRequired TaskState = "input-required"
	TaskStateCompleted     TaskState = "completed"
	TaskStateFailed        TaskState = "failed"
	TaskStateCanceled      TaskState = "canceled"
)

// TaskStatus wraps the current state with a human-readable message and timestamp.
type TaskStatus struct {
	State     TaskState `json:"state"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Part is a polymorphic message part (text, data, or file).
type Part struct {
	Type string `json:"type"` // "text", "data", "file"
	Text string `json:"text,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Message is an A2A conversation message.
type Message struct {
	Role  string `json:"role"` // "user" or "agent"
	Parts []Part `json:"parts"`
}

// Session tracks conversation history across multiple tasks or interactions.
type Session struct {
	ID        string    `json:"id"`
	Messages  []Message `json:"messages,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	LastSeen  time.Time `json:"lastSeen"`
}

// Artifact is the output of a task execution.
type Artifact struct {
	Name     string                 `json:"name,omitempty"`
	Parts    []Part                 `json:"parts"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Index    int                    `json:"index,omitempty"`
}

// Task is the core A2A work unit.
type Task struct {
	ID        string     `json:"id"`
	SessionID string     `json:"sessionId,omitempty"`
	Status    TaskStatus `json:"status"`
	History   []Message  `json:"history,omitempty"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TaskUpdate is sent when a task's status changes.
type TaskUpdate struct {
	TaskID    string    `json:"taskId"`
	Status    TaskStatus `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// AgentSkill describes a single skill an agent provides.
type AgentSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// AgentCard is the discovery manifest for an agent (A2A Agent Card v1).
type AgentCard struct {
	Name              string       `json:"name"`
	Description       string       `json:"description,omitempty"`
	URL               string       `json:"url,omitempty"`
	Version           string       `json:"version"`
	Capabilities      struct {
		Streaming     bool     `json:"streaming,omitempty"`
		PushNotifications bool `json:"pushNotifications,omitempty"`
		StateTransitionHistory bool `json:"stateTransitionHistory,omitempty"`
	} `json:"capabilities"`
	Skills            []AgentSkill `json:"skills,omitempty"`
	DefaultInputModes []string     `json:"defaultInputModes,omitempty"`
	DefaultOutputModes []string    `json:"defaultOutputModes,omitempty"`
	Provider          struct {
		Organization string `json:"organization,omitempty"`
		URL          string `json:"url,omitempty"`
	} `json:"provider,omitempty"`
	LegacyCaps []string `json:"legacyCapabilities,omitempty"`
}


// A2AEnvelope wraps any A2A message for transport over the existing UDS/WS wire.
// This lets us keep the JSON-RPC-like framing while upgrading the payload to A2A semantics.
type A2AEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Sender  string          `json:"sender"`
	Target  string          `json:"target,omitempty"`
	TaskID  string          `json:"taskId,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Helper constructors
func NewTextPart(text string) Part {
	return Part{Type: "text", Text: text}
}

func NewDataPart(v interface{}) Part {
	b, _ := json.Marshal(v)
	return Part{Type: "data", Data: b}
}

func NewTaskStatus(state TaskState, message string) TaskStatus {
	return TaskStatus{
		State:     state,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}
