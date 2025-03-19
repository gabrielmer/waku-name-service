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

// var ContentTopic = fmt.Sprintf("/%s/%s/foo/plain", AppName, AppVersion)

func main() {

	// Keep running until interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// Set up signal handling
	go func() {
		sig := <-quit
		fmt.Printf("\nReceived signal: %v\n", sig)
		fmt.Println("Server shutting down...")

		// Exit with a zero status code
		os.Exit(0)
	}()

	wakuNode, err := wns.SetupWakuNode()
	if err != nil {
		fmt.Printf("Failed to start server node: %v\n", err)
	}
	defer wakuNode.Stop()

	// Start server
	go wns.StartWnsServer(wakuNode)

	testSenderNode, err := wns.SetupWakuNode()
	if err != nil {
		fmt.Printf("Failed to start test sender node: %v\n", err)
	}
	defer testSenderNode.Stop()

	message := &pb.WakuMessage{
		Payload:      []byte("hellooo"),
		ContentTopic: "test-content-topic",
		Version:      proto.Uint32(0),
		Timestamp:    proto.Int64(time.Now().UnixNano()),
	}
	// send message
	pubsubTopic := waku.FormatWakuRelayTopic(16, 64)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()
	testSenderNode.RelayPublish(ctx2, message, pubsubTopic)

	// Block until something else causes an exit
	select {}
}
