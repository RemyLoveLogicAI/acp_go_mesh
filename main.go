package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"

	"acp-mesh/pkg/a2a"
	"acp-mesh/pkg/harness"
)

//go:embed ui/index.html
var uiFiles embed.FS

var (
	udsSocketPath = "/tmp/a2a-mesh.sock"
	upgrader      = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// ACPMessage is the legacy wire format. Still used for basic transport.
type ACPMessage struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Sender  string                 `json:"sender"`
	Target  string                 `json:"target,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

var (
	// Legacy state for tool routing + capability tracking
	stateMu       sync.RWMutex
	agentConns    = make(map[string]net.Conn)
	capabilities  = make(map[string][]string)
	tools         = make(map[string]string) // tool_name -> agent_id
	wsClients     = make(map[*websocket.Conn]bool)
	wsClientsMu   sync.RWMutex

	// A2A task store (Wave 1)
	taskStore = harness.NewTaskStore()
)

func broadcastWS(msg interface{}) {
	wsClientsMu.Lock()
	defer wsClientsMu.Unlock()
	b, _ := json.Marshal(msg)
	for client := range wsClients {
		client.WriteMessage(websocket.TextMessage, b)
	}
}

func broadcastState() {
	// Snapshot under stateMu, then release before calling broadcastWS
	// to avoid a deadlock between stateMu and wsClientsMu.
	stateMu.RLock()
	agentInfo := make(map[string]map[string]interface{})
	for id, caps := range capabilities {
		agentInfo[id] = map[string]interface{}{
			"capabilities": caps,
			"tools":        []string{},
		}
	}
	for toolName, agentID := range tools {
		if info, ok := agentInfo[agentID]; ok {
			toolsList := info["tools"].([]string)
			info["tools"] = append(toolsList, toolName)
		}
	}
	stateMu.RUnlock()

	broadcastWS(map[string]interface{}{
		"type":   "state_update",
		"agents": agentInfo,
	})
}

func handleUDSConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	var myAgentID string

	// Create a buffered channel for outbound messages to this agent.
	// We use a WaitGroup to guarantee the writer goroutine has fully drained
	// before we close the channel, preventing a send-on-closed-channel panic.
	outbound := make(chan []byte, 64)
	var writerWg sync.WaitGroup
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		for payload := range outbound {
			if _, err := conn.Write(payload); err != nil {
				// Drain remaining messages so the channel close is unblocked.
				for range outbound {
				}
				return
			}
		}
	}()
	// Close the channel after the read loop exits, then wait for the writer.
	defer func() {
		close(outbound)
		writerWg.Wait()
	}()

	for scanner.Scan() {
		line := scanner.Text()
		var msg ACPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		dispatchMessage(msg, line, conn, &myAgentID, outbound)
	}

	// Cleanup on disconnect
	if myAgentID != "" {
		stateMu.Lock()
		delete(agentConns, myAgentID)
		delete(capabilities, myAgentID)
		for k, v := range tools {
			if v == myAgentID {
				delete(tools, k)
			}
		}
		stateMu.Unlock()
		taskStore.UnregisterAgentConn(myAgentID)
		broadcastState()
		broadcastWS(map[string]interface{}{
			"type":    "log",
			"source":  "Harness",
			"message": fmt.Sprintf("Agent %s disconnected.", myAgentID),
		})
	}
}

// dispatchMessage routes a single parsed wire message to the correct handler.
// Extracted from handleUDSConnection to keep the read-loop small and testable.
// agentID is a pointer so the register case can persist the identity for the
// lifetime of the connection.
func dispatchMessage(msg ACPMessage, rawLine string, conn net.Conn, agentID *string, outbound chan<- []byte) {
	myAgentID := *agentID
	switch msg.Method {
	case "register":
		*agentID = msg.Sender
		myAgentID = *agentID
			var caps []string
			if rawCaps, ok := msg.Params["capabilities"].([]interface{}); ok {
				for _, c := range rawCaps {
					caps = append(caps, c.(string))
				}
			}
			stateMu.Lock()
			agentConns[myAgentID] = conn
			capabilities[myAgentID] = caps
			if rawTools, ok := msg.Params["tools"].([]interface{}); ok {
				for _, t := range rawTools {
					if tMap, ok := t.(map[string]interface{}); ok {
						if toolName, ok := tMap["name"].(string); ok {
							tools[toolName] = myAgentID
						}
					}
				}
			}
			stateMu.Unlock()

			// Register in A2A task store
			taskStore.RegisterAgentConn(myAgentID, outbound)

			// Accept agent_card (snake_case from agents) OR agentCard (camelCase)
			for _, cardKey := range []string{"agent_card", "agentCard"} {
				if rawCard, ok := msg.Params[cardKey].(map[string]interface{}); ok {
					var card a2a.AgentCard
					cb, _ := json.Marshal(rawCard)
					json.Unmarshal(cb, &card)
					taskStore.RegisterAgentCard(myAgentID, card)
					break
				}
			}

			broadcastWS(map[string]interface{}{
				"type":    "log",
				"source":  "Harness",
				"message": fmt.Sprintf("Agent %s discovered (A2A).", myAgentID),
			})
			broadcastState()

		case "mcp/tools/list":
			if msg.Target == "harness" {
				stateMu.RLock()
				var toolList []string
				for t := range tools {
					toolList = append(toolList, t)
				}
				stateMu.RUnlock()

				resp := ACPMessage{
					JSONRPC: "2.0",
					Method:  "mcp/tools/list/response",
					Sender:  "harness",
					Target:  myAgentID,
					Params: map[string]interface{}{
						"tools": toolList,
					},
				}
				b, _ := json.Marshal(resp)
				outbound <- append(b, '\n')
			}

		case "mcp/tools/call":
			if msg.Target == "harness" {
				toolName, _ := msg.Params["tool_name"].(string)
				stateMu.RLock()
				targetAgent, exists := tools[toolName]
				stateMu.RUnlock()

				broadcastWS(map[string]interface{}{
					"type": "acp_trace",
					"data": msg,
				})

				if exists {
					// A2A Task Envelope: wrap the MCP tool call in a Task
					taskID := fmt.Sprintf("task_%s_%d", myAgentID, time.Now().UnixNano())
					reqPayload, _ := json.Marshal(msg.Params)

					newTask := a2a.Task{
						ID:     taskID,
						Status: a2a.NewTaskStatus(a2a.TaskStateSubmitted, "Task submitted via MCP tool call"),
						Metadata: map[string]interface{}{
							"sender":       myAgentID,
							"targetAgent":  targetAgent,
							"tool_name":    toolName,
							"requestPayload": string(reqPayload),
						},
					}
					_, err := taskStore.CreateTask(newTask)
					if err != nil {
						log.Printf("[Harness] Failed to create task: %v", err)
						return
					}

					// Forward to target agent with task context
					forward := ACPMessage{
						JSONRPC: "2.0",
						Method:  "mcp/tools/call",
						Sender:  myAgentID,
						Target:  targetAgent,
						Params:  msg.Params,
					}
					forward.Params["task_id"] = taskID
					forward.Params["requester"] = myAgentID
					fb, _ := json.Marshal(forward)

					stateMu.RLock()
					targetConn, tExists := agentConns[targetAgent]
					stateMu.RUnlock()
					if tExists {
						_, err := targetConn.Write(append(fb, '\n'))
						if err != nil {
							log.Printf("[Harness] Failed to forward to %s: %v", targetAgent, err)
						}
					}
				} else {
					resp := ACPMessage{
						JSONRPC: "2.0",
						Method:  "mcp/tools/call/response",
						Sender:  "harness",
						Target:  msg.Sender,
						Params: map[string]interface{}{
							"status": "error",
							"result": "Tool not found: " + toolName,
						},
					}
					b, _ := json.Marshal(resp)
					outbound <- append(b, '\n')
				}
			}

		case "tasks/send":
			// A2A native task send
			var task a2a.Task
			if rawTask, ok := msg.Params["task"].(map[string]interface{}); ok {
				tb, _ := json.Marshal(rawTask)
				json.Unmarshal(tb, &task)
			}
			if task.ID == "" {
				task.ID = fmt.Sprintf("task_%s_%d", msg.Sender, time.Now().UnixNano())
			}
			if task.Status.State == "" {
				task.Status = a2a.NewTaskStatus(a2a.TaskStateSubmitted, "Task received via tasks/send")
			}
			sm, err := taskStore.CreateTask(task)
			if err != nil {
				log.Printf("[Harness] tasks/send create failed: %v", err)
				return
			}

			// If task has a target, forward it
			if msg.Target != "" && msg.Target != "harness" {
				stateMu.RLock()
				targetConn, tExists := agentConns[msg.Target]
				stateMu.RUnlock()
				if tExists {
					_, err := targetConn.Write(append([]byte(rawLine), '\n'))
					if err != nil {
						log.Printf("[Harness] Failed to forward task to %s: %v", msg.Target, err)
					}
				}
			}

			// Broadcast task creation to UI
			broadcastWS(map[string]interface{}{
				"type":   "task_update",
				"taskId": task.ID,
				"status": sm.Task().Status,
			})

		case "tasks/sendUpdate":
			// A2A task status update from an agent
			taskID, _ := msg.Params["task_id"].(string)
			if rawStatus, ok := msg.Params["status"].(map[string]interface{}); ok && taskID != "" {
				var status a2a.TaskStatus
				sb, _ := json.Marshal(rawStatus)
				json.Unmarshal(sb, &status)

				var requester string
				if sm, ok := taskStore.GetTask(taskID); ok {
					if meta, ok := sm.Task().Metadata["sender"]; ok {
						requester, _ = meta.(string)
					}
				}

				err := taskStore.TransitionTask(taskID, status.State, status.Message)
				if err != nil {
					log.Printf("[Harness] Task transition failed: %v", err)
					return
				}

				// Broadcast to UI
				broadcastWS(map[string]interface{}{
					"type":   "task_update",
					"taskId": taskID,
					"status": status,
				})

				// Forward to requester (manager) via outbound channel (not raw net.Conn)
				if requester != "" && requester != msg.Sender {
					resp := ACPMessage{
						JSONRPC: "2.0",
						Method:  "tasks/sendUpdate",
						Sender:  "harness",
						Target:  requester,
						Params: map[string]interface{}{
							"task_id": taskID,
							"status":  status,
						},
					}
					rb, _ := json.Marshal(resp)
					taskStore.SendToAgent(requester, append(rb, '\n'))
				}
			}

		case "approval_response":
			// Legacy approval response: translate into A2A task update
			taskID, _ := msg.Params["task_id"].(string)
			approvalStatus, _ := msg.Params["status"].(string)
			if taskID != "" {
				var newState a2a.TaskState
				var message string
				if approvalStatus == "approved" {
					newState = a2a.TaskStateWorking
					message = "User approved execution"
				} else {
					newState = a2a.TaskStateFailed
					message = "User rejected execution"
				}
				err := taskStore.TransitionTask(taskID, newState, message)
				if err != nil {
					log.Printf("[Harness] Approval transition failed: %v", err)
					return
				}

				// Forward approval to the worker via outbound channel
				if sm, ok := taskStore.GetTask(taskID); ok {
					meta := sm.Task().Metadata
					if targetAgent, ok := meta["targetAgent"].(string); ok && targetAgent != "" {
						if !taskStore.SendToAgent(targetAgent, append([]byte(rawLine), '\n')) {
							log.Printf("[Harness] Failed to forward approval to %s: agent not reachable", targetAgent)
						}
					}
				}

				// Broadcast to UI
				if sm, ok := taskStore.GetTask(taskID); ok {
					broadcastWS(map[string]interface{}{
						"type":   "task_update",
						"taskId": taskID,
						"status": sm.Task().Status,
					})
				}
			}

		case "discover":
			// Capability-based agent discovery (Wave 1 / Wave 2 bridge)
			var required []string
			if rawSkills, ok := msg.Params["required_skills"].([]interface{}); ok {
				for _, s := range rawSkills {
					if sv, ok := s.(string); ok {
						required = append(required, sv)
					}
				}
			}
			all := taskStore.ListAgentCards()
			var matched []map[string]interface{}
			for agentID, card := range all {
				if agentID == myAgentID {
					continue // don't discover yourself
				}
				skillIDs := make(map[string]bool)
				for _, sk := range card.Skills {
					skillIDs[sk.ID] = true
				}
				// Also check legacy capabilities
				stateMu.RLock()
				for _, cap := range capabilities[agentID] {
					skillIDs[cap] = true
				}
				stateMu.RUnlock()
				matches := len(required) == 0
				if !matches {
					for _, r := range required {
						if skillIDs[r] {
							matches = true
							break
						}
					}
				}
				if matches {
					matched = append(matched, map[string]interface{}{
						"id":   agentID,
						"name": card.Name,
						"skills": skillIDs,
					})
				}
			}
			discResp := ACPMessage{
				JSONRPC: "2.0",
				Method:  "discover_response",
				Sender:  "harness",
				Target:  myAgentID,
				Params:  map[string]interface{}{"agents": matched},
			}
			db, _ := json.Marshal(discResp)
			outbound <- append(db, '\n')

		case "execute_task":
			// Manager → Worker task delegation — create A2A task record then forward.
			target := msg.Target
			command, _ := msg.Params["command"].(string)
			taskID := fmt.Sprintf("task_%s_%d", myAgentID, time.Now().UnixNano())
			newTask := a2a.Task{
				ID:     taskID,
				Status: a2a.NewTaskStatus(a2a.TaskStateSubmitted, "Delegated via execute_task"),
				Metadata: map[string]interface{}{
					"sender":      myAgentID,
					"targetAgent": target,
					"command":     command,
				},
			}
			if _, err := taskStore.CreateTask(newTask); err != nil {
				log.Printf("[Harness] execute_task store failed: %v", err)
			}
			broadcastWS(map[string]interface{}{
				"type":    "task_update",
				"taskId":  taskID,
				"status":  newTask.Status,
				"command": command,
			})
			// Inject the harness-assigned task_id before forwarding
			if msg.Params == nil {
				msg.Params = map[string]interface{}{}
			}
			msg.Params["task_id"] = taskID
			if !taskStore.SendToAgent(target, func() []byte {
				b, _ := json.Marshal(msg)
				return append(b, '\n')
			}()) {
				log.Printf("[Harness] execute_task: target %s not reachable", target)
			}

		case "task_state_update":
			// Legacy method emitted by worker — translate to A2A transition.
			taskID, _ := msg.Params["task_id"].(string)
			stateStr, _ := msg.Params["state"].(string)
			messageStr, _ := msg.Params["message"].(string)
			if taskID != "" && stateStr != "" {
				newState := a2a.TaskState(stateStr)
				var requesterID string
				if sm, ok := taskStore.GetTask(taskID); ok {
					if v, ok := sm.Task().Metadata["sender"]; ok {
						requesterID, _ = v.(string)
					}
				}
				if err := taskStore.TransitionTask(taskID, newState, messageStr); err != nil {
					log.Printf("[Harness] task_state_update transition failed: %v", err)
				} else {
					broadcastWS(map[string]interface{}{
						"type":    "task_update",
						"taskId":  taskID,
						"state":   stateStr,
						"message": messageStr,
					})
					// Forward state change to requester (manager)
					if requesterID != "" && requesterID != myAgentID {
						upd := ACPMessage{
							JSONRPC: "2.0", Method: "task_state_update",
							Sender:  "harness", Target: requesterID,
							Params:  map[string]interface{}{"task_id": taskID, "state": stateStr, "message": messageStr},
						}
						ub, _ := json.Marshal(upd)
						taskStore.SendToAgent(requesterID, append(ub, '\n'))
					}
				}
			}

		case "approval_request":
			// Worker requests human approval — broadcast to UI instead of routing
			// to non-existent 'ui_user' agent connection.
			broadcastWS(map[string]interface{}{
				"type": "approval_request",
				"data": msg,
			})

		case "ping":
			// Heartbeat — send pong back via outbound channel (not raw conn)
			pong := ACPMessage{JSONRPC: "2.0", Method: "pong", Sender: "harness", Target: myAgentID}
			pb, _ := json.Marshal(pong)
			outbound <- append(pb, '\n')

		default:
			// Standard message routing for unrecognised methods
			broadcastWS(map[string]interface{}{
				"type": "acp_trace",
				"data": msg,
			})
			if msg.Target != "" {
				if !taskStore.SendToAgent(msg.Target, append([]byte(rawLine), '\n')) {
					log.Printf("[Harness] Route miss: target %s not connected", msg.Target)
				}
			}
		}
	}
}

func startUDSServer() {
	os.Remove(udsSocketPath)
	listener, err := net.Listen("unix", udsSocketPath)
	if err != nil {
		log.Fatalf("Failed to start UDS server: %v", err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleUDSConnection(conn)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	wsClientsMu.Lock()
	wsClients[conn] = true

	// Send current topology
	stateMu.RLock()
	agentInfo := make(map[string]map[string]interface{})
	for id, caps := range capabilities {
		agentInfo[id] = map[string]interface{}{
			"capabilities": caps,
			"tools":        []string{},
		}
	}
	for toolName, agentID := range tools {
		if info, ok := agentInfo[agentID]; ok {
			toolsList := info["tools"].([]string)
			info["tools"] = append(toolsList, toolName)
		}
	}
	stateMu.RUnlock()

	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"state_update","agents":%s}`, toJSON(agentInfo))))

	// Also send all active tasks
	wsClientsMu.Unlock()

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg ACPMessage
		if err := json.Unmarshal(p, &msg); err == nil && msg.Target != "" {
			broadcastWS(map[string]interface{}{
				"type": "acp_trace",
				"data": msg,
			})

			stateMu.RLock()
			targetConn, exists := agentConns[msg.Target]
			stateMu.RUnlock()
			if exists {
				_, err := targetConn.Write(append(p, '\n'))
				if err != nil {
					broadcastWS(map[string]interface{}{
						"type":   "log",
						"source": "Harness",
						"message": fmt.Sprintf("Failed to route message from UI: target %s not found", msg.Target),
					})
				}
			}
		}
	}

	wsClientsMu.Lock()
	delete(wsClients, conn)
	wsClientsMu.Unlock()
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func spawnAgent(agentPath, agentID, caps string) *exec.Cmd {
	cmd := exec.Command("go", "run", agentPath)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "ACP_SOCKET="+udsSocketPath)
	cmd.Env = append(cmd.Env, "AGENT_ID="+agentID)
	cmd.Env = append(cmd.Env, "CAPABILITIES="+caps)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Fatalf("Failed to start %s: %v", agentID, err)
	}

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				cleanBuf := bytes.ReplaceAll(buf[:n], []byte("\n"), []byte("\r\n"))
				broadcastWS(map[string]interface{}{
					"type": "pty_stream",
					"data": string(cleanBuf),
				})
			}
			if err != nil {
				break
			}
		}
	}()

	return cmd
}

func main() {
	go startUDSServer()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html, _ := uiFiles.ReadFile("ui/index.html")
		w.Header().Set("Content-Type", "text/html")
		w.Write(html)
	})
	http.HandleFunc("/ws", handleWebSocket)

	go func() {
		log.Println("[Harness] Command Center UI running at http://localhost:8080")
		http.ListenAndServe(":8080", nil)
	}()

	time.Sleep(500 * time.Millisecond)

	log.Println("Spawning worker agent...")
	spawnAgent("agent/worker/main.go", "worker", "shell_exec,read")

	time.Sleep(1 * time.Second)

	log.Println("Spawning manager agent...")
	spawnAgent("agent/manager/main.go", "manager", "orchestrate")

	select {}
}
