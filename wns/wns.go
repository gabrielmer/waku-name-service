package wns

import (
	"fmt"
	"log"
	"time"

	"github.com/waku-org/waku-go-bindings/waku"
	"github.com/waku-org/waku-go-bindings/waku/common"
)

func SetupServerNode() (*waku.WakuNode, error) {

	const requestTimeout = 30 * time.Second
	// Configure server node
	serverNodeWakuConfig := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   9020,
		TcpPort:         60020,
	}

	// Create and start server node
	serverNode, err := waku.NewWakuNode(&serverNodeWakuConfig, "serverNode")
	if err != nil {
		fmt.Printf("Failed to create server node: %v\n", err)
		return nil, err
	}
	if err := serverNode.Start(); err != nil {
		fmt.Printf("Failed to start server node: %v\n", err)
		return nil, err
	}
	return serverNode, nil
}

func StartWnsServer(serverNode *waku.WakuNode) {
	log.Println("Server started, listening for messages...")

	for {
		select {
		case envelope := <-serverNode.MsgChan:
			if envelope != nil {
				// Print the payload
				fmt.Printf("Received message with payload: %s\n", envelope.Message().Payload)
				fmt.Printf("Content topic: %s\n", envelope.Message().ContentTopic)

				// You can add more processing here if needed
			}
		}
	}
}
