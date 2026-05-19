package a2a

import (
	"encoding/json"
	"testing"
)

func TestA2AEnvelopeRoundTrip(t *testing.T) {
	payload := map[string]interface{}{
		"task": map[string]interface{}{
			"id":      "task_123",
			"state":   "submitted",
			"message": "Test task",
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	env := A2AEnvelope{
		JSONRPC:   "2.0",
		ID:        "msg_123",
		Method:    "tasks/send",
		Sender:    "manager",
		Target:    "harness",
		TaskID:    "task_123",
		SessionID: "session_456",
		Payload:   payloadBytes,
	}

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

	// Verify payload can be decoded
	var decoded map[string]interface{}
	if err := json.Unmarshal(env.Payload, &decoded); err != nil {
		t.Errorf("Failed to unmarshal payload: %v", err)
	}
	taskRaw, ok := decoded["task"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected task in payload")
	}
	if taskRaw["id"] != "task_123" {
		t.Errorf("Expected task id task_123, got %v", taskRaw["id"])
	}
}

func TestA2AEnvelopeEmptyPayload(t *testing.T) {
	env := A2AEnvelope{
		JSONRPC: "2.0",
		Method:  "register",
		Sender:  "agent1",
	}

	if env.Payload != nil {
		t.Error("Expected nil payload")
	}
}

func TestNewTaskStatus(t *testing.T) {
	status := NewTaskStatus(TaskStateCompleted, "Done")
	if status.State != TaskStateCompleted {
		t.Errorf("Expected state completed, got %s", status.State)
	}
	if status.Message != "Done" {
		t.Errorf("Expected message Done, got %s", status.Message)
	}
	if status.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestNewTextPart(t *testing.T) {
	part := NewTextPart("hello")
	if part.Type != "text" {
		t.Errorf("Expected type text, got %s", part.Type)
	}
	if part.Text != "hello" {
		t.Errorf("Expected text hello, got %s", part.Text)
	}
}

func TestNewDataPart(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	part := NewDataPart(data)
	if part.Type != "data" {
		t.Errorf("Expected type data, got %s", part.Type)
	}
	if len(part.Data) == 0 {
		t.Error("Expected non-empty data")
	}
}
