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

	var availableTools []string
	var pendingUserIntent string

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		var msg ACPMessage
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			
			if msg.Method == "mcp/tools/list/response" {
				if tools, ok := msg.Params["tools"].([]interface{}); ok {
					availableTools = nil
					for _, t := range tools {
						availableTools = append(availableTools, t.(string))
					}
					fmt.Printf("\033[35m[Manager %s]\033[0m Mesh has tools available: %v\n", agentID, availableTools)
					
					if pendingUserIntent != "" {
						hasExecute := false
						for _, t := range availableTools {
							if t == "execute_shell" {
								hasExecute = true
							}
						}
						
						if hasExecute {
							execMsg := ACPMessage{
								JSONRPC: "2.0",
								Method:  "mcp/tools/call",
								Sender:  agentID,
								Target:  "harness",
								Params: map[string]interface{}{
									"tool_name": "execute_shell",
									"command":   pendingUserIntent,
								},
							}
							eb, _ := json.Marshal(execMsg)
							fmt.Printf("\033[35m[Manager %s]\033[0m Delegating intent to mesh via execute_shell tool...\n", agentID)
							conn.Write(append(eb, '\n'))
						} else {
							fmt.Printf("\033[31m[Manager %s]\033[0m Cannot fulfill intent. Tool 'execute_shell' not found in mesh.\n", agentID)
						}
						pendingUserIntent = ""
					}
				}
			} else if msg.Method == "user_intent" {
				command, ok := msg.Params["command"].(string)
				if ok {
					fmt.Printf("\033[35m[Manager %s]\033[0m Received user intent: %s\n", agentID, command)
					
					pendingUserIntent = command
					listMsg := ACPMessage{
						JSONRPC: "2.0",
						Method:  "mcp/tools/list",
						Sender:  agentID,
						Target:  "harness",
					}
					lb, _ := json.Marshal(listMsg)
					fmt.Printf("\033[35m[Manager %s]\033[0m Querying Mesh for available MCP tools...\n", agentID)
					conn.Write(append(lb, '\n'))
				}
			} else if msg.Method == "mcp/tools/call/response" {
				result, ok := msg.Params["result"].(string)
				status, _ := msg.Params["status"].(string)
				if ok {
					if status == "success" {
						fmt.Printf("\033[35m[Manager %s]\033[0m Tool call completed successfully.\nResult:\n%s\n", agentID, result)
					} else {
						fmt.Printf("\033[31m[Manager %s]\033[0m Tool call failed.\nError:\n%s\n", agentID, result)
					}
				}
			}
		}
	}
}
