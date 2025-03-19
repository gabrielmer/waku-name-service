package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gabrielmer/waku-name-service/wns"
)

const AppName = "wns-server"
const AppVersion = "1"

// var ContentTopic = fmt.Sprintf("/%s/%s/foo/plain", AppName, AppVersion)

func main() {

	wakuNode, err := wns.SetupServerNode()
	if err != nil {
		fmt.Printf("Failed to start server node: %v\n", err)
	}

	// Start server
	go wns.StartWnsServer(wakuNode)

	// Keep running until interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Server shutting down...")
}
