package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type HybridWebSocket struct {
	role      string
	serverURL string
	upgrader  websocket.Upgrader
	clients   map[string]*websocket.Conn
	conn      *websocket.Conn
	data      map[string]interface{}
	mu        sync.RWMutex
	ackStatus map[string]bool
}

func NewHybridWebSocket(role, serverURL string) *HybridWebSocket {
	return &HybridWebSocket{
		role:      role,
		serverURL: serverURL,
		upgrader:  websocket.Upgrader{},
		clients:   make(map[string]*websocket.Conn),
		data:      make(map[string]interface{}),
		ackStatus: make(map[string]bool),
	}
}

func (ws *HybridWebSocket) Start(port string) {
	if ws.role == "server" {
		http.HandleFunc("/ws", ws.handleConnections)
		log.Printf("Server running on :%s", port)
		log.Fatal(http.ListenAndServe(":"+port, nil))
	} else if ws.role == "client" {
		ws.connectToServer()
	}
}

func (ws *HybridWebSocket) handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	clientID := r.URL.Query().Get("clientId")
	ws.clients[clientID] = conn
	ws.ackStatus[clientID] = false
	log.Printf("Client %s connected", clientID)

	for {
		var message map[string]interface{}
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Printf("Client %s disconnected: %v", clientID, err)
			delete(ws.clients, clientID)
			delete(ws.ackStatus, clientID)
			break
		}

		if message["type"] == "ack" && message["clientID"] == clientID {
			ws.mu.Lock()
			ws.ackStatus[clientID] = true
			ws.mu.Unlock()
			log.Printf("ACK received from client %s", clientID)
		}
	}
}

func (ws *HybridWebSocket) broadcastData(data map[string]interface{}) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	for clientID, conn := range ws.clients {
		// Tambahkan clientID ke payload data
		dataWithClientID := map[string]interface{}{
			"data":     data,
			"clientID": clientID,
		}

		err := conn.WriteJSON(dataWithClientID)
		if err != nil {
			log.Printf("Failed to send data to client %s: %v", clientID, err)
			conn.Close()
			delete(ws.clients, clientID)
			continue
		}

		// Wait for ACK
		go func(clientID string) {
			time.Sleep(5 * time.Second) // Timeout 5 detik
			ws.mu.RLock()
			if !ws.ackStatus[clientID] {
				log.Printf("No ACK received from client %s within timeout", clientID)
			}
			ws.mu.RUnlock()
		}(clientID)
	}
}

func (ws *HybridWebSocket) connectToServer() {
	conn, _, err := websocket.DefaultDialer.Dial(ws.serverURL, nil)
	if err != nil {
		log.Printf("Failed to connect to server: %v. Retrying in 5 seconds...", err)
		time.Sleep(5 * time.Second)
		return
	}
	ws.conn = conn
	log.Println("Connected to server")

	for {
		var data map[string]interface{}
		err := conn.ReadJSON(&data)
		if err != nil {
			log.Printf("Disconnected from server: %v. Reconnecting...", err)
			break
		}
		ws.updateLocalVariable(data)

		//send ACK
		ack := map[string]interface{}{
			"type":     "ack",
			"clientID": data["clientID"],
		}
		err = conn.WriteJSON(ack)
		if err != nil {
			log.Printf("Failed to send ACK to server: %v", err)
		}
	}
}

func (ws *HybridWebSocket) updateLocalVariable(data map[string]interface{}) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.data = data
	log.Printf("Local data updated: %v", data)
}

func (ws *HybridWebSocket) getData() map[string]interface{} {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.data
}

func fetchDataFromDB() map[string]interface{} {
	return map[string]interface{}{
		"roomID":   "asadadawra",
		"rateCode": "213141",
	}
}

func runServer(ws *HybridWebSocket) {
	for {
		data := fetchDataFromDB()
		ws.broadcastData(data)
		time.Sleep(10 * time.Second)
	}
}

func main() {
	role := flag.String("role", "server", "Role of this pod: 'server' or 'client'")
	port := flag.String("port", "9999", "Port for server")
	serverURL := flag.String("serverURL", "ws://localhost:8080/ws", "WebSocket server URL (for clients)")
	flag.Parse()
	str, _ := os.Hostname()
	nw := time.Now()
	str = str + fmt.Sprintf("%d", nw.Unix())
	log.Println(str)
	fullUrl := *serverURL + "?clientId=" + str

	ws := NewHybridWebSocket(*role, fullUrl)

	if *role == "server" {
		go runServer(ws)
	}

	ws.Start(*port)
}