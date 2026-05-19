package main

// Package main implements the ACP Go Mesh harness with UDS/Wire transport,
// task lifecycle management, OpenTelemetry tracing, and WebSocket UI.
import (
	"bufio"
	"bytes"
	"context"
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"acp-mesh/pkg/a2a"
	"acp-mesh/pkg/harness"
)

//go:embed ui/index.html
var uiFiles embed.FS

var (
	udsSocketPath = "/tmp/a2a-mesh.sock"
	upgrader      = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return false
			}
			return origin == "http://localhost:8080" || origin == "http://127.0.0.1:8080"
		},
	}
)

// getEnvelopeParams extracts params from A2AEnvelope Payload
func getEnvelopeParams(env a2a.A2AEnvelope) map[string]interface{} {
	params := make(map[string]interface{})
	if len(env.Payload) == 0 {
		return params
	}
	_ = json.Unmarshal(env.Payload, &params)
	return params
}

// MessagePresenter handles transformation of raw wire messages into displayable UI text representations.
type MessagePresenter struct{}

// Present formats an ACPMessage into its UI display representation (role and text content).
func (mp MessagePresenter) Present(msg a2a.ACPMessage) (role string, textContent string) {
	role = "agent"
	if msg.Sender == "ui_user" || msg.Method == "user_intent" {
		role = "user"
	}

	if msg.Method == "user_intent" {
		if cmd, ok := msg.Params["command"].(string); ok {
			textContent = cmd
		}
	} else if msg.Method == "mcp/tools/call" {
		if toolName, ok := msg.Params["tool_name"].(string); ok {
			textContent = fmt.Sprintf("Calls tool %s", toolName)
		}
	} else if msg.Method == "mcp/tools/call/response" {
		if result, ok := msg.Params["result"].(string); ok {
			textContent = result
		}
	} else {
		textContent = fmt.Sprintf("Message Method: %s", msg.Method)
	}
	return role, textContent
}

var (
	// Legacy state for tool routing + capability tracking
	stateMu       sync.RWMutex
	agentConns    = make(map[string]net.Conn)
	capabilities  = make(map[string][]string)
	tools         = make(map[string]string) // tool_name -> agent_id
	wsClients     = make(map[*websocket.Conn]bool)
	wsClientsMu   sync.RWMutex
	topologyCache = make(map[string]map[string]interface{})

	// A2A task store (Wave 1)
	taskStore = harness.NewTaskStore()

	// OpenTelemetry tracer and active span tracking
	tracer        = otel.Tracer("acp-mesh")
	activeSpans   = make(map[string]trace.Span)
	activeSpansMu sync.Mutex
)

func broadcastWS(msg interface{}) {
	b, _ := json.Marshal(msg)
	wsClientsMu.Lock()
	defer wsClientsMu.Unlock()
	for client := range wsClients {
		client.WriteMessage(websocket.TextMessage, b)
	}
}

func broadcastState() {
	stateMu.RLock()
	agentInfo := make(map[string]map[string]interface{}, len(topologyCache))
	for k, v := range topologyCache {
		agentInfo[k] = v
	}
	stateMu.RUnlock()

	broadcastWS(map[string]interface{}{
		"type":   "state_update",
		"agents": agentInfo,
	})
}

func cleanupAgent(myAgentID string) {
	if myAgentID == "" {
		return
	}
	stateMu.Lock()
	delete(agentConns, myAgentID)
	delete(capabilities, myAgentID)
	for k, v := range tools {
		if v == myAgentID {
			delete(tools, k)
		}
	}
	delete(topologyCache, myAgentID)
	stateMu.Unlock()

	taskStore.UnregisterAgent(myAgentID)
	broadcastState()
	broadcastWS(map[string]interface{}{
		"type":    "log",
		"source":  "Harness",
		"message": fmt.Sprintf("Agent %s disconnected.", myAgentID),
	})
}

func handleUDSConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	var myAgentID string

	defer func() {
		cleanupAgent(myAgentID)
	}()

	// Buffered channel for outbound messages to this agent.
	// WaitGroup guarantees the writer goroutine finishes before we close the channel,
	// preventing send-on-closed-channel panics from late handler writes.
	outbound := make(chan []byte, 64)
	var writerWg sync.WaitGroup
	writerWg.Add(1)
	go func() {
		defer writerWg.Done()
		for payload := range outbound {
			if _, err := conn.Write(payload); err != nil {
				conn.Close()
				for range outbound {
					// Discard remaining messages to unblock senders.
				}
				return
			}
		}
	}()
	defer func() {
		close(outbound)
		writerWg.Wait()
	}()

	for scanner.Scan() {
		line := scanner.Text()
		var env a2a.A2AEnvelope
		if err := json.Unmarshal([]byte(line), &env); err != nil {
			continue
		}
		dispatchMessage(env, line, conn, &myAgentID, outbound)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[Harness] Scanner error: %v", err)
	}
}

func recordSessionMessage(msg a2a.ACPMessage) {
	if msg.SessionID == "" {
		return
	}
	role, textContent := MessagePresenter{}.Present(msg)
	if textContent != "" {
		a2aMsg := a2a.Message{
			Role:  role,
			Parts: []a2a.Part{{Type: "text", Text: textContent}},
		}
		taskStore.AppendSessionMessage(msg.SessionID, a2aMsg)
		broadcastWS(map[string]interface{}{
			"type":       "session_update",
			"session_id": msg.SessionID,
			"message":    a2aMsg,
		})
	}
}

func dispatchMessage(env a2a.A2AEnvelope, rawLine string, conn net.Conn, agentID *string, outbound chan []byte) {
	myAgentID := *agentID
	// Convert to ACPMessage for now for backward compatibility
	msg := env.ToACPMessage()
	recordSessionMessage(msg)
	switch msg.Method {
	case "register":
		handleRegister(msg, conn, agentID, outbound)
	case "mcp/tools/list":
		handleMCPToolsList(msg, myAgentID, outbound)
	case "mcp/tools/call":
		handleMCPToolsCall(msg, myAgentID, outbound)
	case "tasks/send":
		handleTasksSend(msg, rawLine)
	case "tasks/sendUpdate":
		handleTasksSendUpdate(msg)
	case "tasks/cancel":
		handleTasksCancel(msg, outbound)
	case "discover":
		handleDiscover(msg, myAgentID, outbound)
	case "approval_response":
		handleApprovalResponse(msg, rawLine)
	default:
		handleDefaultRoute(msg, rawLine)
	}
}

func handleRegister(msg a2a.ACPMessage, conn net.Conn, agentID *string, outbound chan []byte) {
	*agentID = msg.Sender
	myAgentID := *agentID

	var caps []string
	if rawCaps, ok := msg.Params["capabilities"].([]interface{}); ok {
		for _, c := range rawCaps {
			caps = append(caps, c.(string))
		}
	}

	stateMu.Lock()
	agentConns[myAgentID] = conn
	capabilities[myAgentID] = caps
	var cachedTools []string
	if rawTools, ok := msg.Params["tools"].([]interface{}); ok {
		for _, t := range rawTools {
			if tMap, ok := t.(map[string]interface{}); ok {
				if toolName, ok := tMap["name"].(string); ok {
					tools[toolName] = myAgentID
					cachedTools = append(cachedTools, toolName)
				}
			}
		}
	}
	topologyCache[myAgentID] = map[string]interface{}{
		"capabilities": caps,
		"tools":        cachedTools,
	}
	stateMu.Unlock()

	taskStore.RegisterAgentConn(myAgentID, outbound)

	if rawCard, ok := msg.Params["agentCard"].(map[string]interface{}); ok {
		var card a2a.AgentCard
		cb, _ := json.Marshal(rawCard)
		json.Unmarshal(cb, &card)
		taskStore.RegisterAgentCard(myAgentID, card)
	}

	broadcastWS(map[string]interface{}{
		"type":    "log",
		"source":  "Harness",
		"message": fmt.Sprintf("Agent %s discovered (A2A).", myAgentID),
	})
	broadcastState()
}

func handleMCPToolsList(msg a2a.ACPMessage, myAgentID string, outbound chan []byte) {
	if msg.Target != "harness" {
		return
	}
	stateMu.RLock()
	var toolList []string
	for t := range tools {
		toolList = append(toolList, t)
	}
	stateMu.RUnlock()

	resp := a2a.ACPMessage{
		JSONRPC: "2.0",
		Method:  "mcp/tools/list/response",
		Sender:  "harness",
		Target:  myAgentID,
		Params:  map[string]interface{}{"tools": toolList},
	}
	b, _ := json.Marshal(resp)
	outbound <- append(b, '\n')
}

func handleMCPToolsCall(msg a2a.ACPMessage, myAgentID string, outbound chan []byte) {
	if msg.Target != "harness" {
		return
	}
	toolName, _ := msg.Params["tool_name"].(string)
	targetAgent, exists := taskStore.FindAgentBySkill(toolName)
	if !exists {
		stateMu.RLock()
		targetAgent, exists = tools[toolName]
		stateMu.RUnlock()
	}

	broadcastWS(map[string]interface{}{
		"type": "acp_trace",
		"data": msg,
	})

	if exists {
		taskID := fmt.Sprintf("task_%s_%d", myAgentID, time.Now().UnixNano())
		reqPayload, _ := json.Marshal(msg.Params)

		// Start OpenTelemetry span for task lifecycle
		_, span := tracer.Start(context.Background(), "task-lifecycle")
		spanCtx := span.SpanContext()

		newTask := a2a.Task{
			ID:        taskID,
			SessionID: msg.SessionID,
			Status: a2a.NewTaskStatus(a2a.TaskStateSubmitted,
				"Task submitted via MCP tool call"),
			Metadata: map[string]interface{}{
				"sender":         myAgentID,
				"targetAgent":    targetAgent,
				"tool_name":      toolName,
				"requestPayload": string(reqPayload),
				"trace_id":       spanCtx.TraceID().String(),
				"span_id":        spanCtx.SpanID().String(),
				"session_id":     msg.SessionID,
			},
		}
		_, err := taskStore.CreateTask(newTask)
		if err != nil {
			log.Printf("[Harness] Failed to create task: %v", err)
			span.End()
			return
		}

		activeSpansMu.Lock()
		activeSpans[taskID] = span
		activeSpansMu.Unlock()

		forward := a2a.ACPMessage{
			JSONRPC:   "2.0",
			Method:    "mcp/tools/call",
			Sender:    myAgentID,
			Target:    targetAgent,
			SessionID: msg.SessionID,
			Params:    msg.Params,
		}
		forward.Params["task_id"] = taskID
		forward.Params["requester"] = myAgentID
		fb, _ := json.Marshal(forward)

		if !taskStore.SendToAgent(targetAgent, append(fb, '\n')) {
			log.Printf("[Harness] Failed to forward tool call to %s: agent not reachable", targetAgent)
		}
	} else {
		resp := a2a.ACPMessage{
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

func handleTasksSend(msg a2a.ACPMessage, rawLine string) {
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
	if task.SessionID == "" && msg.SessionID != "" {
		task.SessionID = msg.SessionID
	}
	sm, err := taskStore.CreateTask(task)
	if err != nil {
		log.Printf("[Harness] tasks/send create failed: %v", err)
		return
	}

	if msg.Target != "" && msg.Target != "harness" {
		if !taskStore.SendToAgent(msg.Target, append([]byte(rawLine), '\n')) {
			log.Printf("[Harness] Failed to forward task to %s: agent not reachable", msg.Target)
		}
	}

	broadcastWS(map[string]interface{}{
		"type":   "task_update",
		"taskId": task.ID,
		"status": sm.Task().Status,
	})
}

func handleTasksSendUpdate(msg a2a.ACPMessage) {
	taskID, _ := msg.Params["task_id"].(string)
	if taskID == "" {
		taskID, _ = msg.Params["taskId"].(string)
	}
	rawStatus, ok := msg.Params["status"].(map[string]interface{})
	if !ok || taskID == "" {
		return
	}
	var status a2a.TaskStatus
	sb, _ := json.Marshal(rawStatus)
	json.Unmarshal(sb, &status)

	// Validate that the sender is authorized to update this task
	sm, ok := taskStore.GetTask(taskID)
	if !ok {
		log.Printf("[Harness] tasks/sendUpdate failed: Task %s not found", taskID)
		return
	}
	meta := sm.Task().Metadata
	requester, _ := meta["sender"].(string)
	targetAgent, _ := meta["targetAgent"].(string)

	if msg.Sender != requester && msg.Sender != targetAgent {
		log.Printf("[Harness] Unauthorized tasks/sendUpdate from %s for task %s (expected %s or %s)",
			msg.Sender, taskID, requester, targetAgent)
		return
	}

	if rawArtifact, ok := msg.Params["artifact"].(map[string]interface{}); ok {
		var artifact a2a.Artifact
		ab, _ := json.Marshal(rawArtifact)
		if err := json.Unmarshal(ab, &artifact); err == nil && len(artifact.Parts) > 0 {
			sm.SetArtifact(artifact)
		}
	}

	task := sm.Task()
	updatePayload := map[string]interface{}{
		"type":      "task_update",
		"taskId":    taskID,
		"status":    status,
		"traceId":   "",
		"spanId":    "",
		"artifacts": task.Artifacts,
		"history":   sm.History(),
	}
	if meta := task.Metadata; meta != nil {
		if tid, ok := meta["trace_id"].(string); ok {
			updatePayload["traceId"] = tid
		}
		if sid, ok := meta["span_id"].(string); ok {
			updatePayload["spanId"] = sid
		}
	}

	err := taskStore.TransitionTask(taskID, status.State, status.Message)
	if err != nil {
		log.Printf("[Harness] Task transition failed: %v", err)
		return
	}

	// Broadcast input-required notification to UI/manager for approval gating
	if status.State == a2a.TaskStateInputRequired {
		broadcastWS(map[string]interface{}{
			"type":    "approval_request",
			"task_id": taskID,
			"message": status.Message,
			"traceId": updatePayload["traceId"],
			"spanId":  updatePayload["spanId"],
		})
	}

	// End OpenTelemetry span if task reached terminal state
	if status.State == a2a.TaskStateCompleted || status.State == a2a.TaskStateFailed || status.State == a2a.TaskStateCanceled {
		activeSpansMu.Lock()
		if span, ok := activeSpans[taskID]; ok {
			span.End()
			delete(activeSpans, taskID)
		}
		activeSpansMu.Unlock()
	}

	broadcastWS(updatePayload)

	if requester != "" && requester != msg.Sender {
		resp := a2a.ACPMessage{
			JSONRPC: "2.0",
			Method:  "tasks/sendUpdate",
			Sender:  "harness",
			Target:  requester,
			Params: map[string]interface{}{
				"task_id":   taskID,
				"status":    status,
				"trace_id":  updatePayload["traceId"],
				"span_id":   updatePayload["spanId"],
				"artifacts": task.Artifacts,
				"history":   sm.History(),
			},
		}
		rb, _ := json.Marshal(resp)
		if !taskStore.SendToAgent(requester, append(rb, '\n')) {
			log.Printf("[Harness] Failed to forward task update to requester %s", requester)
		}
	}
}

func handleTasksCancel(msg a2a.ACPMessage, outbound chan []byte) {
	taskID, _ := msg.Params["task_id"].(string)
	if taskID == "" {
		return
	}
	if !taskStore.CancelTask(taskID) {
		resp := a2a.ACPMessage{
			JSONRPC: "2.0",
			Method:  "tasks/cancel/response",
			Sender:  "harness",
			Target:  msg.Sender,
			Params: map[string]interface{}{
				"status": "error",
				"result": "Task not found or already terminal",
			},
		}
		b, _ := json.Marshal(resp)
		outbound <- append(b, '\n')
		return
	}
	// Forward cancel to the executing agent
	sm, ok := taskStore.GetTask(taskID)
	if ok {
		targetAgent, _ := sm.Task().Metadata["targetAgent"].(string)
		if targetAgent != "" && targetAgent != msg.Sender {
			fwd := a2a.ACPMessage{
				JSONRPC: "2.0",
				Method:  "tasks/cancel",
				Sender:  "harness",
				Target:  targetAgent,
				Params:  map[string]interface{}{"task_id": taskID},
			}
			fb, _ := json.Marshal(fwd)
			taskStore.SendToAgent(targetAgent, append(fb, '\n'))
		}
	}
	// Notify UI
	broadcastWS(map[string]interface{}{
		"type":      "task_update",
		"taskId":    taskID,
		"status":    a2a.NewTaskStatus(a2a.TaskStateCanceling, "Cancellation requested"),
		"traceId":   "",
		"spanId":    "",
		"history":   []a2a.TaskStatus{},
		"artifacts": []a2a.Artifact{},
	})
	resp := a2a.ACPMessage{
		JSONRPC: "2.0",
		Method:  "tasks/cancel/response",
		Sender:  "harness",
		Target:  msg.Sender,
		Params: map[string]interface{}{
			"status": "success",
			"result": "Cancellation signaled",
		},
	}
	b, _ := json.Marshal(resp)
	outbound <- append(b, '\n')
}

func handleDiscover(msg a2a.ACPMessage, myAgentID string, outbound chan []byte) {
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
			continue
		}
		skillIDs := make(map[string]bool)
		for _, sk := range card.Skills {
			skillIDs[sk.ID] = true
		}
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
				"id":     agentID,
				"name":   card.Name,
				"skills": skillIDs,
			})
		}
	}

	discResp := a2a.ACPMessage{
		JSONRPC: "2.0",
		Method:  "discover_response",
		Sender:  "harness",
		Target:  myAgentID,
		Params:  map[string]interface{}{"agents": matched},
	}
	db, _ := json.Marshal(discResp)
	outbound <- append(db, '\n')
}

func getApprovalState(msg a2a.ACPMessage) (a2a.TaskState, string) {
	status, _ := msg.Params["status"].(string)
	if status == "approved" {
		return a2a.TaskStateWorking, "User approved execution"
	}
	return a2a.TaskStateFailed, "User rejected execution"
}

func forwardApproval(taskID, rawLine string) {
	sm, ok := taskStore.GetTask(taskID)
	if !ok {
		return
	}
	targetAgent, ok := sm.Task().Metadata["targetAgent"].(string)
	if !ok || targetAgent == "" {
		return
	}
	if !taskStore.SendToAgent(targetAgent, append([]byte(rawLine), '\n')) {
		log.Printf("[Harness] Failed to forward approval to %s: agent not reachable", targetAgent)
	}
}

func broadcastTaskToUI(taskID string) {
	if sm, ok := taskStore.GetTask(taskID); ok {
		task := sm.Task()
		updatePayload := map[string]interface{}{
			"type":      "task_update",
			"taskId":    taskID,
			"status":    task.Status,
			"traceId":   "",
			"spanId":    "",
			"artifacts": task.Artifacts,
			"history":   sm.History(),
		}
		if meta := task.Metadata; meta != nil {
			if tid, ok := meta["trace_id"].(string); ok {
				updatePayload["traceId"] = tid
			}
			if sid, ok := meta["span_id"].(string); ok {
				updatePayload["spanId"] = sid
			}
		}
		broadcastWS(updatePayload)
	}
}

func handleApprovalResponse(msg a2a.ACPMessage, rawLine string) {
	taskID, _ := msg.Params["task_id"].(string)
	if taskID == "" {
		return
	}
	newState, message := getApprovalState(msg)
	if err := taskStore.TransitionTask(taskID, newState, message); err != nil {
		log.Printf("[Harness] Approval transition failed: %v", err)
		return
	}

	// End OpenTelemetry span if task reached terminal state
	if newState == a2a.TaskStateFailed || newState == a2a.TaskStateCanceled {
		activeSpansMu.Lock()
		if span, ok := activeSpans[taskID]; ok {
			span.End()
			delete(activeSpans, taskID)
		}
		activeSpansMu.Unlock()
	}

	forwardApproval(taskID, rawLine)
	broadcastTaskToUI(taskID)
}

func handleDefaultRoute(msg a2a.ACPMessage, rawLine string) {
	broadcastWS(map[string]interface{}{
		"type": "acp_trace",
		"data": msg,
	})

	target := msg.Target
	if target == "" && len(msg.RequiredSkills) > 0 {
		for _, skill := range msg.RequiredSkills {
			if matchedAgent, found := taskStore.FindAgentBySkill(skill); found {
				target = matchedAgent
				break
			}
		}
	}

	if target != "" {
		routedLine := rawLine
		if msg.Target == "" {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(rawLine), &m); err == nil {
				m["target"] = target
				if b, err := json.Marshal(m); err == nil {
					routedLine = string(b)
				}
			}
		}
		if !taskStore.SendToAgent(target, append([]byte(routedLine), '\n')) {
			log.Printf("[Harness] Route error to %s: agent not reachable", target)
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

func getWSStateUpdate() map[string]interface{} {
	stateMu.RLock()
	defer stateMu.RUnlock()
	agentInfo := make(map[string]map[string]interface{}, len(topologyCache))
	for k, v := range topologyCache {
		agentInfo[k] = v
	}
	return map[string]interface{}{
		"type":   "state_update",
		"agents": agentInfo,
	}
}

func handleWSIncoming(conn *websocket.Conn) {
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var env a2a.A2AEnvelope
		if err := json.Unmarshal(p, &env); err == nil && env.Target != "" {
			// Convert to ACPMessage for backward compatibility during migration
			msg := env.ToACPMessage()
			broadcastWS(map[string]interface{}{
				"type": "acp_trace",
				"data": msg,
			})
			if msg.Target == "harness" {
				if msg.Method == "approval_response" && msg.Sender == "ui_user" {
					go handleApprovalResponse(msg, string(p))
				}
			} else {
				if !taskStore.SendToAgent(msg.Target, append(p, '\n')) {
					broadcastWS(map[string]interface{}{
						"type":    "log",
						"source":  "Harness",
						"message": fmt.Sprintf("Failed to route message from UI: target %s not found", msg.Target),
					})
				}
			}
		}
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
	wsClientsMu.Unlock()

	stateUpdate := getWSStateUpdate()
	b, _ := json.Marshal(stateUpdate)
	conn.WriteMessage(websocket.TextMessage, b)

	// Synchronize the last 50 messages of active sessions to the client on connection
	recentUpdates := taskStore.GetSessionRecentUpdates(50)
	for _, update := range recentUpdates {
		bm, err := json.Marshal(update)
		if err == nil {
			conn.WriteMessage(websocket.TextMessage, bm)
		}
	}

	handleWSIncoming(conn)

	wsClientsMu.Lock()
	delete(wsClients, conn)
	wsClientsMu.Unlock()
}

func toJSON(_ interface{}) string {
	return "{}"
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

var startTime = time.Now()

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	stateMu.RLock()
	agentInfo := make(map[string]map[string]interface{}, len(topologyCache))
	for k, v := range topologyCache {
		agentInfo[k] = v
	}
	stateMu.RUnlock()

	tasks := taskStore.ListTasks()
	sessions := taskStore.ListSessions()

	resp := map[string]interface{}{
		"status":       "healthy",
		"uptime":       time.Since(startTime).String(),
		"agents":       agentInfo,
		"taskCount":    len(tasks),
		"sessionCount": len(sessions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	go startUDSServer()

	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		html, _ := uiFiles.ReadFile("ui/index.html")
		w.Header().Set("Content-Type", "text/html")
		w.Write(html)
	})
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/health", handleHealth)

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
