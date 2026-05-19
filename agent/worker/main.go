package main

// Package main implements a shell execution worker with approval gating.
import (
	"bufio"
	"bytes"
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

// A2AEnvelope is the canonical wire format for agent-to-agent communication.
type A2AEnvelope = a2a.A2AEnvelope

func getEnvelopeParams(msg A2AEnvelope) map[string]interface{} {
	params := make(map[string]interface{})
	if len(msg.Payload) == 0 {
		return params
	}
	_ = json.Unmarshal(msg.Payload, &params)
	return params
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
	regPayload, _ := json.Marshal(map[string]interface{}{
		"capabilities": capabilities,
		"tools": []map[string]interface{}{
			{"name": "execute_shell", "description": "Execute a shell command on host"},
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
	})
	regMsg := A2AEnvelope{
		JSONRPC: "2.0",
		Method:  "register",
		Sender:  agentID,
		Payload: regPayload,
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
		var env A2AEnvelope
		if err := json.Unmarshal([]byte(line), &env); err != nil {
			continue
		}
		// Extract params from envelope payload
		params := getEnvelopeParams(env)

		switch env.Method {
		case "mcp/tools/call":
			toolName, ok := params["tool_name"].(string)
			if ok && toolName == "execute_shell" {
				command, _ := params["command"].(string)
				taskID, _ := params["task_id"].(string)
				requester, _ := params["requester"].(string)

				fmt.Printf("\033[36m[Worker %s]\033[0m Received MCP call (task %s) from %s: %s\n", agentID, taskID, env.Sender, command)

				// Store context so we can resume after approval
				taskCtxs.Store(taskID, taskContext{
					TaskID:    taskID,
					Command:   command,
					Requester: requester,
					SessionID: env.SessionID,
				})
				// Create cancel channel for this task
				cancelChans.Store(taskID, make(chan struct{}))

				// Transition task to input-required (approval needed)
				// Do NOT block. The worker continues listening.
				updatePayload, _ := json.Marshal(map[string]interface{}{
					"task_id": taskID,
					"status": a2a.TaskStatus{
						State:     a2a.TaskStateInputRequired,
						Message:   fmt.Sprintf("Approval required for: %s", command),
						Timestamp: a2a.NewTaskStatus(a2a.TaskStateInputRequired, "").Timestamp,
					},
				})
				updateEnv := A2AEnvelope{
					JSONRPC:   "2.0",
					Method:    "tasks/sendUpdate",
					Sender:    agentID,
					Target:    "harness",
					TaskID:    taskID,
					SessionID: env.SessionID,
					Payload:   updatePayload,
				}

				ub, _ := json.Marshal(updateEnv)
				conn.Write(append(ub, '\n'))

				fmt.Printf("\033[36m[Worker %s]\033[0m Task %s awaiting approval (input-required). Worker continues listening.\n", agentID, taskID)
			}

		case "tasks/cancel":
			cancelParams := getEnvelopeParams(env)
			taskID, _ := cancelParams["task_id"].(string)
			if taskID == "" {
				continue
			}
			if ch, ok := cancelChans.Load(taskID); ok {
				close(ch.(chan struct{}))
				fmt.Printf("\033[36m[Worker %s]\033[0m Cancel signal received for task %s\n", agentID, taskID)
			}

		case "approval_response":
			approvalParams := getEnvelopeParams(env)
			taskID, _ := approvalParams["task_id"].(string)
			approvalStatus, _ := approvalParams["status"].(string)

			val, ok := taskCtxs.Load(taskID)
			if !ok {
				fmt.Printf("\033[31m[Worker %s]\033[0m Received approval for unknown task %s\n", agentID, taskID)
				continue
			}
			ctx := val.(taskContext)

			if approvalStatus == "approved" {
				fmt.Printf("\033[36m[Worker %s]\033[0m Task %s approved. Executing: %s\n", agentID, taskID, ctx.Command)

				// Transition to working
				workingPayload, _ := json.Marshal(map[string]interface{}{
					"task_id": taskID,
					"status": a2a.TaskStatus{
						State:     a2a.TaskStateWorking,
						Message:   "Executing shell command",
						Timestamp: a2a.NewTaskStatus(a2a.TaskStateWorking, "").Timestamp,
					},
				})
				workingEnv := A2AEnvelope{
					JSONRPC:   "2.0",
					Method:    "tasks/sendUpdate",
					Sender:    agentID,
					Target:    "harness",
					TaskID:    taskID,
					SessionID: ctx.SessionID,
					Payload:   workingPayload,
				}
				wb, _ := json.Marshal(workingEnv)
				conn.Write(append(wb, '\n'))

				// Execute shell command with cancellation support
				execCtx, cancel := context.WithCancel(context.Background())
				defer cancel()
				cmd := exec.CommandContext(execCtx, "sh", "-c", ctx.Command)
				var output bytes.Buffer
				cmd.Stdout = &output
				cmd.Stderr = &output
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
					resStatus = a2a.TaskStateCompleted
					resOutput = output.String()
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
				finalPayload, _ := json.Marshal(map[string]interface{}{
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
				})
				finalEnv := A2AEnvelope{
					JSONRPC:   "2.0",
					Method:    "tasks/sendUpdate",
					Sender:    agentID,
					Target:    "harness",
					TaskID:    taskID,
					SessionID: ctx.SessionID,
					Payload:   finalPayload,
				}
				fb, _ := json.Marshal(finalEnv)
				conn.Write(append(fb, '\n'))

				// Also send MCP response to requester
				if ctx.Requester != "" {
					mcpStatus := "success"
					if resStatus == a2a.TaskStateFailed {
						mcpStatus = "error"
					}
					mcpPayload, _ := json.Marshal(map[string]interface{}{
						"task_id": taskID,
						"status":  mcpStatus,
						"result":  resOutput,
					})
					respEnv := A2AEnvelope{
						JSONRPC:   "2.0",
						Method:    "mcp/tools/call/response",
						Sender:    agentID,
						Target:    ctx.Requester,
						TaskID:    taskID,
						SessionID: ctx.SessionID,
						Payload:   mcpPayload,
					}
					rb, _ := json.Marshal(respEnv)
					conn.Write(append(rb, '\n'))
				}

				fmt.Printf("\033[36m[Worker %s]\033[0m Task %s finished (%s).\n", agentID, taskID, resStatus)

			} else {
				fmt.Printf("\033[31m[Worker %s]\033[0m Task %s rejected by user.\n", agentID, taskID)

				// Transition to failed (rejected)
				rejectPayload, _ := json.Marshal(map[string]interface{}{
					"task_id": taskID,
					"status": a2a.TaskStatus{
						State:     a2a.TaskStateFailed,
						Message:   "Execution rejected by user",
						Timestamp: a2a.NewTaskStatus(a2a.TaskStateFailed, "").Timestamp,
					},
				})
				rejectEnv := A2AEnvelope{
					JSONRPC:   "2.0",
					Method:    "tasks/sendUpdate",
					Sender:    agentID,
					Target:    "harness",
					TaskID:    taskID,
					SessionID: ctx.SessionID,
					Payload:   rejectPayload,
				}
				jb, _ := json.Marshal(rejectEnv)
				conn.Write(append(jb, '\n'))

				// MCP response to requester
				if ctx.Requester != "" {
					mcpPayload, _ := json.Marshal(map[string]interface{}{
						"task_id": taskID,
						"status":  "error",
						"result":  "Execution rejected by user.",
					})
					respEnv := A2AEnvelope{
						JSONRPC: "2.0",
						Method:  "mcp/tools/call/response",
						Sender:  agentID,
						Target:  ctx.Requester,
						Payload: mcpPayload,
					}
					rb, _ := json.Marshal(respEnv)
					conn.Write(append(rb, '\n'))
				}
			}

			// Clean up
			taskCtxs.Delete(taskID)
		}
	}
}
