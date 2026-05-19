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
	TaskStateCanceling     TaskState = "canceling"
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
	Type string          `json:"type"` // "text", "data", "file"
	Text string          `json:"text,omitempty"`
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
	ID          string                 `json:"id"`
	SessionID   string                 `json:"sessionId,omitempty"`
	Status      TaskStatus             `json:"status"`
	History     []Message              `json:"history,omitempty"`
	Artifacts   []Artifact             `json:"artifacts,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CancelToken string                 `json:"cancelToken,omitempty"`
}

// TaskUpdate is sent when a task's status changes.
type TaskUpdate struct {
	TaskID    string       `json:"taskId"`
	Status    TaskStatus   `json:"status"`
	Timestamp time.Time    `json:"timestamp"`
	TraceID   string       `json:"traceId,omitempty"`
	SpanID    string       `json:"spanId,omitempty"`
	Artifacts []Artifact   `json:"artifacts,omitempty"`
	History   []TaskStatus `json:"history,omitempty"`
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
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	URL          string `json:"url,omitempty"`
	Version      string `json:"version"`
	Capabilities struct {
		Streaming              bool `json:"streaming,omitempty"`
		PushNotifications      bool `json:"pushNotifications,omitempty"`
		StateTransitionHistory bool `json:"stateTransitionHistory,omitempty"`
	} `json:"capabilities"`
	Skills             []AgentSkill `json:"skills,omitempty"`
	DefaultInputModes  []string     `json:"defaultInputModes,omitempty"`
	DefaultOutputModes []string     `json:"defaultOutputModes,omitempty"`
	Provider           struct {
		Organization string `json:"organization,omitempty"`
		URL          string `json:"url,omitempty"`
	} `json:"provider,omitempty"`
	LegacyCaps []string `json:"legacyCapabilities,omitempty"`
}

// ACPMessage is the legacy wire format used for transport.
// Deprecated: Use A2AEnvelope instead.
type ACPMessage struct {
	JSONRPC        string                 `json:"jsonrpc"`
	ID             string                 `json:"id,omitempty"`
	Method         string                 `json:"method"`
	Sender         string                 `json:"sender"`
	Target         string                 `json:"target,omitempty"`
	RequiredSkills []string               `json:"required_skills,omitempty"`
	TaskID         string                 `json:"task_id,omitempty"`
	SessionID      string                 `json:"session_id,omitempty"`
	Params         map[string]interface{} `json:"params,omitempty"`
}

// A2A Method constants for type-safe messaging
const (
	MethodRegister         = "register"
	MethodDiscover         = "discover"
	MethodDiscoverResponse = "discover_response"
	MethodTasksSend        = "tasks/send"
	MethodTasksUpdate      = "tasks/sendUpdate"
	MethodTasksCancel      = "tasks/cancel"
	MethodTasksCancelResp  = "tasks/cancel/response"
	MethodMCPToolsList     = "mcp/tools/list"
	MethodMCPToolsListResp = "mcp/tools/list/response"
	MethodMCPToolsCall     = "mcp/tools/call"
	MethodMCPToolsCallResp = "mcp/tools/call/response"
	MethodApprovalRequest  = "approval_request"
	MethodApprovalResponse = "approval_response"
	MethodAgentHeartbeat   = "agent/heartbeat"
)

// A2AEnvelope wraps any A2A message for transport over the existing UDS/WS wire.
// This is the canonical wire format for agent-to-agent communication.
type A2AEnvelope struct {
	JSONRPC   string          `json:"jsonrpc"`
	ID        string          `json:"id,omitempty"`
	Method    string          `json:"method"`
	Sender    string          `json:"sender"`
	Target    string          `json:"target,omitempty"`
	TaskID    string          `json:"taskId,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// Typed payload structures for A2A methods

// RegisterPayload is the payload for agent registration
type RegisterPayload struct {
	Capabilities []string            `json:"capabilities,omitempty"`
	Tools        []map[string]string `json:"tools,omitempty"`
	AgentCard    *AgentCard          `json:"agentCard,omitempty"`
}

// TasksSendPayload is the payload for tasks/send
type TasksSendPayload struct {
	Task Task `json:"task"`
}

// TasksUpdatePayload is the payload for tasks/sendUpdate
type TasksUpdatePayload struct {
	TaskID    string       `json:"task_id,omitempty"`
	TaskIDV2  string       `json:"taskId,omitempty"` // Alternative field name
	Status    TaskStatus   `json:"status"`
	TraceID   string       `json:"trace_id,omitempty"`
	SpanID    string       `json:"span_id,omitempty"`
	Artifact  *Artifact    `json:"artifact,omitempty"`
	Artifacts []Artifact   `json:"artifacts,omitempty"`
	History   []TaskStatus `json:"history,omitempty"`
}

// TasksCancelPayload is the payload for tasks/cancel
type TasksCancelPayload struct {
	TaskID string `json:"task_id"`
}

// MCPToolsCallPayload is the payload for mcp/tools/call
type MCPToolsCallPayload struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Command   string                 `json:"command,omitempty"` // Legacy field
	TaskID    string                 `json:"task_id,omitempty"`
	Requester string                 `json:"requester,omitempty"`
}

// MCPToolsCallResponse is the response payload for mcp/tools/call
type MCPToolsCallResponse struct {
	TaskID string `json:"task_id,omitempty"`
	Status string `json:"status"` // "success", "error"
	Result string `json:"result"`
}

// DiscoverPayload is the payload for discover requests
type DiscoverPayload struct {
	RequiredSkills []string `json:"required_skills,omitempty"`
}

// DiscoverResponse is the payload for discover_response
type DiscoverResponse struct {
	Agents []map[string]interface{} `json:"agents"`
}

// ApprovalResponsePayload is the payload for approval_response
type ApprovalResponsePayload struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"` // "approved", "rejected"
}

// ToA2AEnvelope converts legacy ACPMessage to A2AEnvelope format.
func (msg ACPMessage) ToA2AEnvelope() A2AEnvelope {
	env := A2AEnvelope{
		JSONRPC:   msg.JSONRPC,
		ID:        msg.ID,
		Method:    msg.Method,
		Sender:    msg.Sender,
		Target:    msg.Target,
		TaskID:    msg.TaskID,
		SessionID: msg.SessionID,
	}
	if len(msg.Params) > 0 {
		if b, err := json.Marshal(msg.Params); err == nil {
			env.Payload = b
		}
	}
	return env
}

// ToACPMessage converts A2AEnvelope to legacy ACPMessage format.
func (env A2AEnvelope) ToACPMessage() ACPMessage {
	msg := ACPMessage{
		JSONRPC:   env.JSONRPC,
		ID:        env.ID,
		Method:    env.Method,
		Sender:    env.Sender,
		Target:    env.Target,
		TaskID:    env.TaskID,
		SessionID: env.SessionID,
	}
	if len(env.Payload) > 0 {
		var params map[string]interface{}
		if err := json.Unmarshal(env.Payload, &params); err == nil {
			msg.Params = params
		}
	}
	return msg
}

// ToACPMessage converts an A2A task update to legacy ACPMessage format.
func (tu TaskUpdate) ToACPMessage(sender, target string) ACPMessage {
	return ACPMessage{
		JSONRPC: "2.0",
		Method:  "tasks/sendUpdate",
		Sender:  sender,
		Target:  target,
		Params: map[string]interface{}{
			"task_id":   tu.TaskID,
			"status":    tu.Status,
			"trace_id":  tu.TraceID,
			"span_id":   tu.SpanID,
			"artifacts": tu.Artifacts,
			"history":   tu.History,
		},
	}
}

// FromACPMessage converts legacy ACPMessage to A2A Task if applicable.
func FromACPMessage(msg ACPMessage) (Task, bool) {
	if msg.Method == "tasks/send" {
		var task Task
		if rawTask, ok := msg.Params["task"].(map[string]interface{}); ok {
			tb, _ := json.Marshal(rawTask)
			json.Unmarshal(tb, &task)
		}
		if task.ID == "" {
			task.ID = msg.TaskID
		}
		if task.SessionID == "" {
			task.SessionID = msg.SessionID
		}
		return task, true
	}
	return Task{}, false
}

// Payload decoder methods for A2AEnvelope

// DecodeRegisterPayload extracts RegisterPayload from the envelope
func (env A2AEnvelope) DecodeRegisterPayload() (*RegisterPayload, error) {
	var payload RegisterPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// DecodeTasksSendPayload extracts TasksSendPayload from the envelope
func (env A2AEnvelope) DecodeTasksSendPayload() (*TasksSendPayload, error) {
	var payload TasksSendPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// DecodeTasksUpdatePayload extracts TasksUpdatePayload from the envelope
func (env A2AEnvelope) DecodeTasksUpdatePayload() (*TasksUpdatePayload, error) {
	var payload TasksUpdatePayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return nil, err
	}
	// Handle both task_id and taskId field names
	if payload.TaskID == "" && payload.TaskIDV2 != "" {
		payload.TaskID = payload.TaskIDV2
	}
	return &payload, nil
}

// DecodeTasksCancelPayload extracts TasksCancelPayload from the envelope
func (env A2AEnvelope) DecodeTasksCancelPayload() (*TasksCancelPayload, error) {
	var payload TasksCancelPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// DecodeMCPToolsCallPayload extracts MCPToolsCallPayload from the envelope
func (env A2AEnvelope) DecodeMCPToolsCallPayload() (*MCPToolsCallPayload, error) {
	var payload MCPToolsCallPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// DecodeDiscoverPayload extracts DiscoverPayload from the envelope
func (env A2AEnvelope) DecodeDiscoverPayload() (*DiscoverPayload, error) {
	var payload DiscoverPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// DecodeApprovalResponsePayload extracts ApprovalResponsePayload from the envelope
func (env A2AEnvelope) DecodeApprovalResponsePayload() (*ApprovalResponsePayload, error) {
	var payload ApprovalResponsePayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// NewA2AEnvelope creates a new A2AEnvelope with the given method and payload
func NewA2AEnvelope(method, sender, target string, payload interface{}) (A2AEnvelope, error) {
	env := A2AEnvelope{
		JSONRPC: "2.0",
		Method:  method,
		Sender:  sender,
		Target:  target,
	}
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return env, err
		}
		env.Payload = b
	}
	return env, nil
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
