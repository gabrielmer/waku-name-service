package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gabrielmer/waku-name-service/wns"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/waku-go-bindings/waku"
	"google.golang.org/protobuf/proto"
)

const AppName = "wns-server"
const AppVersion = "1"

func main() {
	// Set up wake nodes first
	wakuNode, err := wns.SetupWakuNode()
	if err != nil {
		fmt.Printf("Failed to start server node: %v\n", err)
		os.Exit(1)
	}
	defer wakuNode.Stop()

	// Start server
	go wns.StartWnsServer(wakuNode)

	testSenderNode, err := wns.SetupWakuNode()
	if err != nil {
		fmt.Printf("Failed to start test sender node: %v\n", err)
		os.Exit(1)
	}
	defer testSenderNode.Stop()

	// Send test message
	message := &pb.WakuMessage{
		Payload:      []byte("hellooo"),
		ContentTopic: "test-content-topic",
		Version:      proto.Uint32(0),
		Timestamp:    proto.Int64(time.Now().UnixNano()),
	}
	pubsubTopic := waku.FormatWakuRelayTopic(16, 64)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	testSenderNode.RelayPublish(ctx, message, pubsubTopic)

	// Set up signal handling after initializing everything
	fmt.Println("Server running. Press Ctrl+C to shutdown gracefully...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Wait for signal
	<-c

	// Perform cleanup
	fmt.Println("\nServer shutting down...")
	fmt.Println("Cleanup complete. Exiting.")
}
