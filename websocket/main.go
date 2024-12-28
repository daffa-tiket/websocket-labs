package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/tiket/TIX-HOTEL-UTILITIES-GO/leaderelection"
	"github.com/tiket/TIX-HOTEL-UTILITIES-GO/logs"
	"github.com/tiket/TIX-HOTEL-UTILITIES-GO/metrics"
	"github.com/tiket/TIX-HOTEL-UTILITIES-GO/websocket"
)

func main() {
	roles := flag.String("role", "server", "Role of this pod: 'server' or 'client'")
	port := flag.String("port", "9999", "Port for server")
	serverURL := flag.String("serverURL", "ws://localhost:8080/ws", "WebSocket server URL (for clients)")
	flag.Parse()
	logger, _ := logs.New(&logs.Option{})
	statsD, err := metrics.NewMonitor(
		"127.0.0.1",
		"8125",
		"websocket",
	)
	fmt.Println(err)
	ws := websocket.NewHybridWebSocket(websocket.WebSocketConfig{
		ServerUrl: *serverURL,
		Logger: logger,
		UseLeaderElection: false,
		LeaderElection: leaderelection.Config{},
		IsSendMetric: true,
		StatsDMonitor: statsD,
	})

	role := websocket.ClientRole
	if *roles == "server" {
		role = websocket.ServerRole
	}
	ws.Runner(role, *port)

	if *roles == "server" {
		for {
			fmt.Println("Broadcasting data to clients")
			data := map[string]interface{}{
				"roomID": "asadadawra",
			}
			ws.Broadcast(data)
			time.Sleep(5 * time.Second)
		}
	}
	select {}
}