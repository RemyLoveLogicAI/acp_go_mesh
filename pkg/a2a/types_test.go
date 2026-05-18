package a2a

import (
	"encoding/json"
	"testing"
)

func TestACPMessageToTaskConversion(t *testing.T) {
	msg := ACPMessage{
		JSONRPC:   "2.0",
		Method:    "tasks/send",
		Sender:    "manager",
		TaskID:    "task_123",
		SessionID: "session_456",
		Params: map[string]interface{}{
			"task": map[string]interface{}{
				"id":      "task_123",
				"state":   "submitted",
				"message": "Test task",
			},
		},
	}

	task, ok := FromACPMessage(msg)
	if !ok {
		t.Fatal("Expected conversion to succeed")
	}
	if task.ID != "task_123" {
		t.Errorf("Expected task ID task_123, got %s", task.ID)
	}
	if task.SessionID != "session_456" {
		t.Errorf("Expected session ID session_456, got %s", task.SessionID)
	}
}

func TestTaskUpdateToACPMessageConversion(t *testing.T) {
	update := TaskUpdate{
		TaskID:    "task_123",
		Status:    NewTaskStatus(TaskStateCompleted, "Done"),
		Timestamp: NewTaskStatus(TaskStateCompleted, "").Timestamp,
		TraceID:   "trace-123",
		SpanID:    "span-456",
		Artifacts: []Artifact{
			{Name: "output", Parts: []Part{{Type: "text", Text: "result"}}},
		},
		History: []TaskStatus{NewTaskStatus(TaskStateSubmitted, "Start")},
	}

	msg := update.ToACPMessage("harness", "manager")
	if msg.Method != "tasks/sendUpdate" {
		t.Errorf("Expected method tasks/sendUpdate, got %s", msg.Method)
	}
	if msg.Sender != "harness" {
		t.Errorf("Expected sender harness, got %s", msg.Sender)
	}
	if msg.Target != "manager" {
		t.Errorf("Expected target manager, got %s", msg.Target)
	}

	taskID, ok := msg.Params["task_id"].(string)
	if !ok || taskID != "task_123" {
		t.Errorf("Expected task_id param to be task_123")
	}
}

func TestACPMessageToA2AEnvelopeConversion(t *testing.T) {
	msg := ACPMessage{
		JSONRPC:   "2.0",
		ID:        "msg_123",
		Method:    "tasks/send",
		Sender:    "manager",
		Target:    "harness",
		TaskID:    "task_123",
		SessionID: "session_456",
		Params: map[string]interface{}{
			"task": map[string]interface{}{
				"id":      "task_123",
				"state":   "submitted",
				"message": "Test task",
			},
		},
	}

	env := msg.ToA2AEnvelope()
	if env.Method != "tasks/send" {
		t.Errorf("Expected method tasks/send, got %s", env.Method)
	}
	if env.Sender != "manager" {
		t.Errorf("Expected sender manager, got %s", env.Sender)
	}
	if env.Target != "harness" {
		t.Errorf("Expected target harness, got %s", env.Target)
	}
	if env.TaskID != "task_123" {
		t.Errorf("Expected TaskID task_123, got %s", env.TaskID)
	}
	if env.Payload == nil {
		t.Error("Expected payload to be set")
	}
}

func TestA2AEnvelopeToACPMessageConversion(t *testing.T) {
	env := A2AEnvelope{
		JSONRPC: "2.0",
		ID:      "msg_123",
		Method:  "tasks/sendUpdate",
		Sender:  "harness",
		Target:  "manager",
		TaskID:  "task_123",
	}
	payload := map[string]interface{}{
		"task_id": "task_123",
		"status":  "completed",
	}
	if b, err := json.Marshal(payload); err == nil {
		env.Payload = b
	}

	msg := env.ToACPMessage()
	if msg.Method != "tasks/sendUpdate" {
		t.Errorf("Expected method tasks/sendUpdate, got %s", msg.Method)
	}
	if msg.Sender != "harness" {
		t.Errorf("Expected sender harness, got %s", msg.Sender)
	}
	if msg.Target != "manager" {
		t.Errorf("Expected target manager, got %s", msg.Target)
	}
	if msg.TaskID != "task_123" {
		t.Errorf("Expected TaskID task_123, got %s", msg.TaskID)
	}
	if msg.Params == nil {
		t.Error("Expected params to be extracted from payload")
	}
}
