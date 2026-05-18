package a2a

import (
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
