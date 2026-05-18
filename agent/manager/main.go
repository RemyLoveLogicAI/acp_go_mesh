package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"acp-mesh/pkg/a2a"
)

// ACPMessage is the legacy wire format.
type ACPMessage struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Sender  string                 `json:"sender"`
	Target  string                 `json:"target,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// taskTracker holds the manager's view of in-flight tasks.
type taskTracker struct {
	mu     sync.RWMutex
	tasks  map[string]a2a.TaskStatus
}

func newTaskTracker() *taskTracker {
	return &taskTracker{tasks: make(map[string]a2a.TaskStatus)}
}

func (tt *taskTracker) Update(taskID string, status a2a.TaskStatus) {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	tt.tasks[taskID] = status
}

func (tt *taskTracker) Get(taskID string) (a2a.TaskStatus, bool) {
	tt.mu.RLock()
	defer tt.mu.RUnlock()
	s, ok := tt.tasks[taskID]
	return s, ok
}

func main() {
	agentID := os.Getenv("AGENT_ID")
	if agentID == "" {
		agentID = "manager"
	}
	socketPath := os.Getenv("ACP_SOCKET")
	if socketPath == "" {
		log.Fatalf("[Manager %s] ACP_SOCKET not set", agentID)
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

	// Send Registration + Agent Card (A2A discovery manifest)
	regMsg := ACPMessage{
		JSONRPC: "2.0",
		Method:  "register",
		Sender:  agentID,
		Params: map[string]interface{}{
			"capabilities": capabilities,
			"agentCard": a2a.AgentCard{
				Name:        agentID,
				Description: "Orchestrator that delegates user intent to mesh workers",
				Version:     "1.0.0",
				Skills: []a2a.AgentSkill{
					{ID: "orchestrate", Name: "Mesh Orchestration", Description: "Routes user commands to capable workers", Tags: []string{"coordination", "routing"}},
				},
				DefaultInputModes:  []string{"text"},
				DefaultOutputModes: []string{"text"},
			},
		},
	}
	b, _ := json.Marshal(regMsg)
	conn.Write(append(b, '\n'))

	tracker := newTaskTracker()

	var availableTools []string
	var pendingUserIntent string

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		var msg ACPMessage
		if err := json.Unmarshal([]byte(line), &msg); err == nil {

			switch msg.Method {
			case "mcp/tools/list/response":
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

			case "user_intent":
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

			case "mcp/tools/call/response":
				result, ok := msg.Params["result"].(string)
				status, _ := msg.Params["status"].(string)
				if ok {
					if status == "success" {
						fmt.Printf("\033[35m[Manager %s]\033[0m Tool call completed successfully.\nResult:\n%s\n", agentID, result)
					} else {
						fmt.Printf("\033[31m[Manager %s]\033[0m Tool call failed.\nError:\n%s\n", agentID, result)
					}
				}

			case "tasks/sendUpdate":
				// A2A task status update from harness (forwarded from worker)
				if rawStatus, ok := msg.Params["status"].(map[string]interface{}); ok {
					var status a2a.TaskStatus
					sb, _ := json.Marshal(rawStatus)
					json.Unmarshal(sb, &status)

					taskID, _ := msg.Params["task_id"].(string)
					if taskID == "" {
						taskID, _ = msg.Params["taskId"].(string)
					}

					if taskID != "" {
						tracker.Update(taskID, status)
						fmt.Printf("\033[35m[Manager %s]\033[0m Task %s state: \033[1m%s\033[0m (%s)\n",
							agentID, taskID, status.State, status.Message)

						if status.State == a2a.TaskStateInputRequired {
							fmt.Printf("\033[33m[Manager %s]\033[0m Worker is waiting for user approval on task %s\n", agentID, taskID)
						}
					}
				}

			default:
				// Other messages (e.g. raw agent-to-agent) — log for visibility
				if msg.Method != "" && msg.Method != "register" {
					// skip noisy ones
				}
			}
		}
	}
}
