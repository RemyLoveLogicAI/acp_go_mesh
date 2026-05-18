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
)

//go:embed ui/index.html
var uiFiles embed.FS

var (
	acpSocketPath = "/tmp/acp-mesh.sock"
	upgrader      = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

type ACPMessage struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Sender  string                 `json:"sender"`
	Target  string                 `json:"target,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

type HarnessState struct {
	mu           sync.Mutex
	agents       map[string]net.Conn
	capabilities map[string][]string
	wsClients    map[*websocket.Conn]bool
}

var state = &HarnessState{
	agents:       make(map[string]net.Conn),
	capabilities: make(map[string][]string),
	wsClients:    make(map[*websocket.Conn]bool),
}

func broadcastWS(msg interface{}) {
	state.mu.Lock()
	defer state.mu.Unlock()
	b, _ := json.Marshal(msg)
	for client := range state.wsClients {
		client.WriteMessage(websocket.TextMessage, b)
	}
}

func broadcastState() {
	state.mu.Lock()
	defer state.mu.Unlock()
	msg := map[string]interface{}{
		"type":   "state_update",
		"agents": state.capabilities,
	}
	b, _ := json.Marshal(msg)
	for client := range state.wsClients {
		client.WriteMessage(websocket.TextMessage, b)
	}
}

func handleUDSConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	var myAgentID string

	for scanner.Scan() {
		line := scanner.Text()
		var msg ACPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if msg.Method == "register" {
			myAgentID = msg.Sender
			var caps []string
			if rawCaps, ok := msg.Params["capabilities"].([]interface{}); ok {
				for _, c := range rawCaps {
					caps = append(caps, c.(string))
				}
			}
			state.mu.Lock()
			state.agents[myAgentID] = conn
			state.capabilities[myAgentID] = caps
			state.mu.Unlock()
			
			broadcastWS(map[string]interface{}{
				"type":    "log",
				"source":  "Harness",
				"message": fmt.Sprintf("Agent %s discovered.", myAgentID),
			})
			broadcastState()
		} else {
			broadcastWS(map[string]interface{}{
				"type": "acp_trace",
				"data": msg,
			})

			state.mu.Lock()
			targetConn, exists := state.agents[msg.Target]
			state.mu.Unlock()

			if exists {
				targetConn.Write(append([]byte(line), '\n'))
			}
		}
	}

	if myAgentID != "" {
		state.mu.Lock()
		delete(state.agents, myAgentID)
		delete(state.capabilities, myAgentID)
		state.mu.Unlock()
		broadcastState()
		broadcastWS(map[string]interface{}{
			"type":    "log",
			"source":  "Harness",
			"message": fmt.Sprintf("Agent %s disconnected.", myAgentID),
		})
	}
}

func startUDSServer() {
	os.Remove(acpSocketPath)
	listener, err := net.Listen("unix", acpSocketPath)
	if err != nil {
		log.Fatalf("Failed to start UDS server: %v", err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil { continue }
		go handleUDSConnection(conn)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil { return }
	defer conn.Close()

	state.mu.Lock()
	state.wsClients[conn] = true
	capsCopy := make(map[string][]string)
	for k, v := range state.capabilities { capsCopy[k] = v }
	state.mu.Unlock()

	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"state_update","agents":%s}`, toJSON(capsCopy))))

	for {
		_, p, err := conn.ReadMessage()
		if err != nil { break }
		
		// Process user injections
		var msg ACPMessage
		if err := json.Unmarshal(p, &msg); err == nil && msg.Target != "" {
			broadcastWS(map[string]interface{}{
				"type": "acp_trace",
				"data": msg,
			})
			// Route to target agent
			state.mu.Lock()
			targetConn, exists := state.agents[msg.Target]
			state.mu.Unlock()
			if exists {
				targetConn.Write(append(p, '\n'))
			} else {
				broadcastWS(map[string]interface{}{
					"type": "log", "source": "Harness", "message": fmt.Sprintf("Failed to route message from UI: target %s not found", msg.Target),
				})
			}
		}
	}

	state.mu.Lock()
	delete(state.wsClients, conn)
	state.mu.Unlock()
}

func toJSON(v interface{}) string { b, _ := json.Marshal(v); return string(b) }

func spawnAgent(agentPath, agentID, caps string) *exec.Cmd {
	cmd := exec.Command("go", "run", agentPath)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "ACP_SOCKET="+acpSocketPath)
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
			if err != nil { break }
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
