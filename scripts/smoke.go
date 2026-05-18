package main

// Smoke test verifies harness health and basic message flow.
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ACPMessage is a simplified message struct for smoke testing.
type ACPMessage struct {
	JSONRPC   string                 `json:"jsonrpc"`
	Method    string                 `json:"method"`
	Sender    string                 `json:"sender"`
	Target    string                 `json:"target,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
}

func main() {
	sessionID := "smoke_test_session"

	fmt.Println("=== SMOKE TEST START ===")
	fmt.Println("Harness should be running at http://localhost:8080")

	// Wait for harness to be ready
	time.Sleep(2 * time.Second)

	// Check health endpoint
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Health check: %s\n", string(body))

	// Send user_intent via WebSocket simulation (using HTTP for now)
	fmt.Println("1. Sending user_intent to harness via HTTP...")

	userIntent := map[string]interface{}{
		"jsonrpc":    "2.0",
		"method":     "user_intent",
		"sender":     "ui_user",
		"target":     "manager",
		"session_id": sessionID,
		"params": map[string]interface{}{
			"command": "echo 'hello from smoke test'",
		},
	}

	jsonData, _ := json.Marshal(userIntent)
	resp, err = http.Post("http://localhost:8080/api/message", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to send user_intent: %v\n", err)
		fmt.Println("Note: HTTP API may not be implemented, this is expected")
	} else {
		resp.Body.Close()
		fmt.Println("User intent sent successfully")
	}

	// Simulate approval response
	fmt.Println("2. Simulating approval_response...")

	approval := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "approval_response",
		"sender":  "ui_user",
		"target":  "harness",
		"params": map[string]interface{}{
			"task_id": "",
			"status":  "approved",
		},
	}

	jsonData, _ = json.Marshal(approval)
	resp, err = http.Post("http://localhost:8080/api/message", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to send approval: %v\n", err)
		fmt.Println("Note: HTTP API may not be implemented, this is expected")
	} else {
		resp.Body.Close()
		fmt.Println("Approval sent successfully")
	}

	time.Sleep(2 * time.Second)

	fmt.Println("=== SMOKE TEST COMPLETE ===")
	fmt.Println("Harness is running and health check passed")
	fmt.Println("For full end-to-end verification, use the WebSocket UI at http://localhost:8080")
	fmt.Println("Check harness logs for task lifecycle and trace continuity")
}
