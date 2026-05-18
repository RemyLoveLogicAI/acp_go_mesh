package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
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
		agentID = "manager"
	}
	socketPath := os.Getenv("ACP_SOCKET")
	if socketPath == "" {
		log.Fatalf("[Agent %s] ACP_SOCKET not set", agentID)
	}

	rawCaps := os.Getenv("CAPABILITIES")
	var capabilities []string
	for _, c := range strings.Split(rawCaps, ",") {
		if c != "" {
			capabilities = append(capabilities, c)
		}
	}

	fmt.Printf("\033[35m[Manager %s]\033[0m Booting up with capabilities: %v\n", agentID, capabilities)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		log.Fatalf("[Manager %s] Failed to connect to harness: %v", agentID, err)
	}
	defer conn.Close()

	fmt.Printf("\033[35m[Manager %s]\033[0m Connected to Harness at %s\n", agentID, socketPath)

	// Send Registration / Discovery Manifest
	regMsg := ACPMessage{
		JSONRPC: "2.0",
		Method:  "register",
		Sender:  agentID,
		Params: map[string]interface{}{
			"capabilities": capabilities,
		},
	}
	b, _ := json.Marshal(regMsg)
	conn.Write(append(b, '\n'))

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		var msg ACPMessage
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			
			if msg.Method == "user_intent" {
				command, ok := msg.Params["command"].(string)
				if ok {
					fmt.Printf("\033[35m[Manager %s]\033[0m Received user intent: %s\n", agentID, command)
					
					// Delegate to worker
					execMsg := ACPMessage{
						JSONRPC: "2.0",
						Method:  "execute_task",
						Sender:  agentID,
						Target:  "worker", // Hardcoded for now, could dynamically discover
						Params: map[string]interface{}{
							"command": command,
						},
					}
					eb, _ := json.Marshal(execMsg)
					fmt.Printf("\033[35m[Manager %s]\033[0m Delegating task to worker...\n", agentID)
					conn.Write(append(eb, '\n'))
				}
			} else if msg.Method == "task_result" {
				result, ok := msg.Params["result"].(string)
				status, _ := msg.Params["status"].(string)
				if ok {
					if status == "success" {
						fmt.Printf("\033[35m[Manager %s]\033[0m Task completed successfully.\nResult:\n%s\n", agentID, result)
					} else {
						fmt.Printf("\033[31m[Manager %s]\033[0m Task failed.\nError:\n%s\n", agentID, result)
					}
				}
			}
		}
	}
}
