package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
)

type ACPMessage struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Sender  string                 `json:"sender"`
	Target  string                 `json:"target,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

func main() {
	agentID := os.Getenv("AGENT_ID")
	if agentID == "" {
		agentID = "worker"
	}
	socketPath := os.Getenv("ACP_SOCKET")
	if socketPath == "" {
		log.Fatalf("[Worker %s] ACP_SOCKET not set", agentID)
	}

	rawCaps := os.Getenv("CAPABILITIES")
	var capabilities []string
	for _, c := range strings.Split(rawCaps, ",") {
		if c != "" {
			capabilities = append(capabilities, c)
		}
	}

	fmt.Printf("\033[36m[Worker %s]\033[0m Booting up with capabilities: %v\n", agentID, capabilities)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Fatalf("[Worker %s] Failed to connect to harness: %v", agentID, err)
	}
	defer conn.Close()

	fmt.Printf("\033[36m[Worker %s]\033[0m Connected to Harness at %s\n", agentID, socketPath)

	// Send Registration / Discovery Manifest
	regMsg := ACPMessage{
		JSONRPC: "2.0",
		Method:  "register",
		Sender:  agentID,
		Params: map[string]interface{}{
			"capabilities": capabilities,
			"tools": []map[string]interface{}{
				{"name": "execute_shell", "description": "Execute a shell command on the host"},
			},
		},
	}
	b, _ := json.Marshal(regMsg)
	conn.Write(append(b, '\n'))

	var pendingTaskID string
	var pendingCommand string
	var pendingRequester string

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		var msg ACPMessage
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			
			if msg.Method == "mcp/tools/call" {
				toolName, ok := msg.Params["tool_name"].(string)
				if ok && toolName == "execute_shell" {
					command, _ := msg.Params["command"].(string)
					fmt.Printf("\033[36m[Worker %s]\033[0m Received MCP call from %s for %s: %s\n", agentID, msg.Sender, toolName, command)
					
					// Store pending command
					pendingCommand = command
					pendingRequester = msg.Sender
					// Just a simple ID for now
					pendingTaskID = fmt.Sprintf("task_%d", os.Getpid())

					// Ask UI for approval
					apprMsg := ACPMessage{
						JSONRPC: "2.0",
						Method:  "approval_request",
						Sender:  agentID,
						Target:  "ui_user",
						Params: map[string]interface{}{
							"task_id": pendingTaskID,
							"command": command,
						},
					}
					ab, _ := json.Marshal(apprMsg)
					fmt.Printf("\033[36m[Worker %s]\033[0m Requesting user approval for: %s\n", agentID, command)
					conn.Write(append(ab, '\n'))
				}
			} else if msg.Method == "approval_response" {
				taskID, _ := msg.Params["task_id"].(string)
				status, _ := msg.Params["status"].(string)

				if taskID == pendingTaskID {
					if status == "approved" {
						fmt.Printf("\033[36m[Worker %s]\033[0m Command approved. Executing: %s\n", agentID, pendingCommand)
						
						// Execute shell command
						cmd := exec.Command("sh", "-c", pendingCommand)
						output, err := cmd.CombinedOutput()
						
						resStatus := "success"
						resOutput := string(output)
						if err != nil {
							resStatus = "error"
							resOutput += "\nError: " + err.Error()
						}

						resMsg := ACPMessage{
							JSONRPC: "2.0",
							Method:  "mcp/tools/call/response",
							Sender:  agentID,
							Target:  pendingRequester,
							Params: map[string]interface{}{
								"task_id": pendingTaskID,
								"status": resStatus,
								"result": resOutput,
							},
						}
						rb, _ := json.Marshal(resMsg)
						conn.Write(append(rb, '\n'))

					} else {
						fmt.Printf("\033[31m[Worker %s]\033[0m Command rejected by user.\n", agentID)
						resMsg := ACPMessage{
							JSONRPC: "2.0",
							Method:  "mcp/tools/call/response",
							Sender:  agentID,
							Target:  pendingRequester,
							Params: map[string]interface{}{
								"task_id": pendingTaskID,
								"status": "error",
								"result": "Execution rejected by user.",
							},
						}
						rb, _ := json.Marshal(resMsg)
						conn.Write(append(rb, '\n'))
					}

					// Clear pending
					pendingTaskID = ""
					pendingCommand = ""
					pendingRequester = ""
				}
			}
		}
	}
}
