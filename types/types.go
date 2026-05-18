// Package types defines the shared A2A-compatible wire types used by the
// harness, manager, and worker. All new fields are optional so agents that
// don't yet emit them continue to interoperate.
package types

import "time"

// ─── Agent Card ──────────────────────────────────────────────────────────────

// AgentCard is an A2A-compatible capability advertisement.
// Agents embed this in their register message under params["agent_card"].
type AgentCard struct {
	Name               string       `json:"name"`
	Description        string       `json:"description,omitempty"`
	Version            string       `json:"version,omitempty"`
	DefaultInputModes  []string     `json:"defaultInputModes,omitempty"`
	DefaultOutputModes []string     `json:"defaultOutputModes,omitempty"`
	Skills             []AgentSkill `json:"skills,omitempty"`
	// LegacyCaps bridges the original flat string-array capability field.
	// The harness auto-promotes these to Skills if no structured skills are present.
	LegacyCaps []string `json:"legacy_capabilities,omitempty"`
}

// SkillIDs returns indexable skill identifiers.
// Falls back to LegacyCaps if no structured skills are defined.
func (c AgentCard) SkillIDs() []string {
	if len(c.Skills) > 0 {
		ids := make([]string, len(c.Skills))
		for i, s := range c.Skills {
			ids[i] = s.ID
		}
		return ids
	}
	out := make([]string, len(c.LegacyCaps))
	copy(out, c.LegacyCaps)
	return out
}

// AgentSkill is an A2A-compatible skill descriptor.
type AgentSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	InputModes  []string `json:"inputModes,omitempty"`
	OutputModes []string `json:"outputModes,omitempty"`
}

// ─── Task Lifecycle ───────────────────────────────────────────────────────────

// TaskState is an A2A-aligned task lifecycle state.
type TaskState string

const (
	TaskSubmitted     TaskState = "submitted"
	TaskWorking       TaskState = "working"
	TaskInputRequired TaskState = "input-required" // blocked on human approval
	TaskCompleted     TaskState = "completed"
	TaskFailed        TaskState = "failed"
	TaskCanceled      TaskState = "canceled"
	TaskRejected      TaskState = "rejected"
)

// Terminal returns true if no further transitions are expected.
func (s TaskState) Terminal() bool {
	switch s {
	case TaskCompleted, TaskFailed, TaskCanceled, TaskRejected:
		return true
	}
	return false
}

// Task is the harness-owned canonical record of an in-flight work unit.
type Task struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id,omitempty"`
	State     TaskState `json:"state"`
	Command   string    `json:"command,omitempty"`
	Requester string    `json:"requester,omitempty"` // agent that sent execute_task
	Worker    string    `json:"worker,omitempty"`    // agent assigned to execute
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Result    string    `json:"result,omitempty"`
}

// ─── Wire Format ─────────────────────────────────────────────────────────────

// ACPMessage is the extended wire format — backward-compatible with the
// original {jsonrpc, method, sender, target, params} shape.
// New top-level fields are omitempty so old agents never see them.
type ACPMessage struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Method  string `json:"method"`
	Sender  string `json:"sender"`
	Target  string `json:"target,omitempty"`

	// Wave 1 additions — all optional, backward-compatible.
	RequiredSkills []string  `json:"required_skills,omitempty"` // capability-based routing
	TaskID         string    `json:"task_id,omitempty"`
	SessionID      string    `json:"session_id,omitempty"`
	TaskStateVal   TaskState `json:"task_state,omitempty"`

	Params map[string]interface{} `json:"params,omitempty"`
}
