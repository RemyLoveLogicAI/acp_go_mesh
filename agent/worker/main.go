package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"acp-mesh/pkg/a2a"
)

// ACPMessage is the legacy wire format.
type ACPMessage struct {
	JSONRPC        string                 `json:"jsonrpc"`
	ID             string                 `json:"id,omitempty"`
	Method         string                 `json:"method"`
	Sender         string                 `json:"sender"`
	Target         string                 `json:"target,omitempty"`
	RequiredSkills []string               `json:"required_skills,omitempty"` // capability-based routing
	TaskID         string                 `json:"task_id,omitempty"`
	SessionID      string                 `json:"session_id,omitempty"`
	Params         map[string]interface{} `json:"params,omitempty"`
}


// taskContext holds everything needed to resume a task after approval.
type taskContext struct {
	TaskID    string
	Command   string
	Requester string
	SessionID string
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

	// Send Registration + Agent Card (A2A discovery manifest)
	regMsg := ACPMessage{
		JSONRPC: "2.0",
		Method:  "register",
		Sender:  agentID,
		Params: map[string]interface{}{
			"capabilities": capabilities,
			"tools": []map[string]interface{}{
				{"name": "execute_shell", "description": "Execute a shell command on the host"},
			},
			"agentCard": a2a.AgentCard{
				Name:        agentID,
				Description: "Shell execution worker with approval gating",
				Version:     "1.0.0",
				Skills: []a2a.AgentSkill{
					{ID: "execute_shell", Name: "Shell Execution", Description: "Runs shell commands with user approval", Tags: []string{"shell", "execution"}},
				},
				DefaultInputModes:  []string{"text"},
				DefaultOutputModes: []string{"text"},
				LegacyCaps:         capabilities,
			},

		},
	}
	b, _ := json.Marshal(regMsg)
	conn.Write(append(b, '\n'))

	// taskContexts stores pending tasks keyed by taskID.
	// This replaces the old blocking variables so the worker never pauses.
	var taskCtxs sync.Map
	// cancelChans stores cancel signal channels keyed by taskID.
	var cancelChans sync.Map

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		var msg ACPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		switch msg.Method {
		case "mcp/tools/call":
			toolName, ok := msg.Params["tool_name"].(string)
			if ok && toolName == "execute_shell" {
				command, _ := msg.Params["command"].(string)
				taskID, _ := msg.Params["task_id"].(string)
				requester, _ := msg.Params["requester"].(string)

				fmt.Printf("\033[36m[Worker %s]\033[0m Received MCP call (task %s) from %s: %s\n", agentID, taskID, msg.Sender, command)

				// Store context so we can resume after approval
				taskCtxs.Store(taskID, taskContext{
					TaskID:    taskID,
					Command:   command,
					Requester: requester,
					SessionID: msg.SessionID,
				})
				// Create cancel channel for this task
				cancelChans.Store(taskID, make(chan struct{}))

				// Transition task to input-required (approval needed)
				// Do NOT block. The worker continues listening.
				updateMsg := ACPMessage{
					JSONRPC:   "2.0",
					Method:    "tasks/sendUpdate",
					Sender:    agentID,
					Target:    "harness",
					SessionID: msg.SessionID,
					Params: map[string]interface{}{
						"task_id": taskID,
						"status": a2a.TaskStatus{
							State:     a2a.TaskStateInputRequired,
							Message:   fmt.Sprintf("Approval required for: %s", command),
							Timestamp: a2a.NewTaskStatus(a2a.TaskStateInputRequired, "").Timestamp,
						},
					},
				}

				ub, _ := json.Marshal(updateMsg)
				conn.Write(append(ub, '\n'))

				fmt.Printf("\033[36m[Worker %s]\033[0m Task %s awaiting approval (input-required). Worker continues listening.\n", agentID, taskID)
			}

		case "tasks/cancel":
			taskID, _ := msg.Params["task_id"].(string)
			if taskID == "" {
				continue
			}
			if ch, ok := cancelChans.Load(taskID); ok {
				close(ch.(chan struct{}))
				fmt.Printf("\033[36m[Worker %s]\033[0m Cancel signal received for task %s\n", agentID, taskID)
			}

		case "approval_response":
			taskID, _ := msg.Params["task_id"].(string)
			approvalStatus, _ := msg.Params["status"].(string)

			val, ok := taskCtxs.Load(taskID)
			if !ok {
				fmt.Printf("\033[31m[Worker %s]\033[0m Received approval for unknown task %s\n", agentID, taskID)
				continue
			}
			ctx := val.(taskContext)

			if approvalStatus == "approved" {
				fmt.Printf("\033[36m[Worker %s]\033[0m Task %s approved. Executing: %s\n", agentID, taskID, ctx.Command)

				// Transition to working
				workingMsg := ACPMessage{
					JSONRPC:   "2.0",
					Method:    "tasks/sendUpdate",
					Sender:    agentID,
					Target:    "harness",
					SessionID: ctx.SessionID,
					Params: map[string]interface{}{
						"task_id": taskID,
						"status": a2a.TaskStatus{
							State:     a2a.TaskStateWorking,
							Message:   "Executing shell command",
							Timestamp: a2a.NewTaskStatus(a2a.TaskStateWorking, "").Timestamp,
						},
					},
				}
				wb, _ := json.Marshal(workingMsg)
				conn.Write(append(wb, '\n'))

				// Execute shell command with cancellation support
				execCtx, cancel := context.WithCancel(context.Background())
				defer cancel()
				cmd := exec.CommandContext(execCtx, "sh", "-c", ctx.Command)
				cancelCh, hasCancel := cancelChans.Load(taskID)

				var resStatus a2a.TaskState
				var resOutput string

				// Start command
				if err := cmd.Start(); err != nil {
					resStatus = a2a.TaskStateFailed
					resOutput = "Error starting command: " + err.Error()
				} else {
					// Watch for cancel signal in a goroutine
					if hasCancel {
						go func() {
							select {
							case <-cancelCh.(chan struct{}):
								cmd.Process.Kill()
								cancel()
							case <-execCtx.Done():
							}
						}()
					}

					// Wait for command to finish
					err := cmd.Wait()
					output, _ := cmd.CombinedOutput()
					resStatus = a2a.TaskStateCompleted
					resOutput = string(output)
					if err != nil {
						if execCtx.Err() == context.Canceled {
							resStatus = a2a.TaskStateCanceled
							resOutput += "\n(Canceled by user)"
						} else {
							resStatus = a2a.TaskStateFailed
							resOutput += "\nError: " + err.Error()
						}
					}
				}

				// Transition to completed or failed
				finalMsg := ACPMessage{
					JSONRPC:   "2.0",
					Method:    "tasks/sendUpdate",
					Sender:    agentID,
					Target:    "harness",
					SessionID: ctx.SessionID,
					Params: map[string]interface{}{
						"task_id": taskID,
						"status": a2a.TaskStatus{
							State:     resStatus,
							Message:   "Execution finished",
							Timestamp: a2a.NewTaskStatus(resStatus, "").Timestamp,
						},
						"artifact": a2a.Artifact{
							Name:  "shell_output",
							Parts: []a2a.Part{a2a.NewTextPart(resOutput)},
							Metadata: map[string]interface{}{
								"exit_ok": err == nil,
								"command": ctx.Command,
							},
						},
					},
				}
				fb, _ := json.Marshal(finalMsg)
				conn.Write(append(fb, '\n'))

				// Also send MCP response to requester
				if ctx.Requester != "" {
					mcpStatus := "success"
					if resStatus == a2a.TaskStateFailed {
						mcpStatus = "error"
					}
					resp := ACPMessage{
						JSONRPC:   "2.0",
						Method:    "mcp/tools/call/response",
						Sender:    agentID,
						Target:    ctx.Requester,
						SessionID: ctx.SessionID,
						Params: map[string]interface{}{
							"task_id": taskID,
							"status":  mcpStatus,
							"result":  resOutput,
						},
					}
					rb, _ := json.Marshal(resp)
					conn.Write(append(rb, '\n'))
				}


				fmt.Printf("\033[36m[Worker %s]\033[0m Task %s finished (%s).\n", agentID, taskID, resStatus)

			} else {
				fmt.Printf("\033[31m[Worker %s]\033[0m Task %s rejected by user.\n", agentID, taskID)

				// Transition to failed (rejected)
				rejectMsg := ACPMessage{
					JSONRPC: "2.0",
					Method:  "tasks/sendUpdate",
					Sender:  agentID,
					Target:  "harness",
					Params: map[string]interface{}{
						"task_id": taskID,
						"status": a2a.TaskStatus{
							State:     a2a.TaskStateFailed,
							Message:   "Execution rejected by user",
							Timestamp: a2a.NewTaskStatus(a2a.TaskStateFailed, "").Timestamp,
						},
					},
				}
				jb, _ := json.Marshal(rejectMsg)
				conn.Write(append(jb, '\n'))

				// MCP response to requester
				if ctx.Requester != "" {
					resp := ACPMessage{
						JSONRPC: "2.0",
						Method:  "mcp/tools/call/response",
						Sender:  agentID,
						Target:  ctx.Requester,
						Params: map[string]interface{}{
							"task_id": taskID,
							"status":  "error",
							"result":  "Execution rejected by user.",
						},
					}
					rb, _ := json.Marshal(resp)
					conn.Write(append(rb, '\n'))
				}
			}

			// Clean up
			taskCtxs.Delete(taskID)
		}
	}
}
