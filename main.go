package wns

import (
	"context"
	"fmt"
	"time"

	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/waku-go-bindings/waku"
	"github.com/waku-org/waku-go-bindings/waku/common"
	"go.uber.org/zap"
)

const AppName = "wns-server"
const AppVersion = "1"

var ContentTopic = fmt.Sprintf("/%s/%s/foo/plain", AppName, AppVersion)

func main() {
	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		return
	}
	defer logger.Sync()

	const requestTimeout = 30 * time.Second
	// Configure dialer node
	dialerNodeWakuConfig := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   9020,
		TcpPort:         60020,
	}

	// Create and start dialer node
	dialerNode, err := waku.NewWakuNode(&dialerNodeWakuConfig, "dialerNode")
	if err != nil {
		fmt.Printf("Failed to create dialer node: %v\n", err)
		return
	}
	if err := dialerNode.Start(); err != nil {
		fmt.Printf("Failed to start dialer node: %v\n", err)
		return
	}
	defer dialerNode.Stop()
	time.Sleep(1 * time.Second)

	// Configure receiver node
	receiverNodeWakuConfig := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   9021,
		TcpPort:         60021,
	}

	// Create and start receiver node
	receiverNode, err := waku.NewWakuNode(&receiverNodeWakuConfig, "receiverNode")
	if err != nil {
		fmt.Printf("Failed to create receiver node: %v\n", err)
		return
	}
	if err := receiverNode.Start(); err != nil {
		fmt.Printf("Failed to start receiver node: %v\n", err)
		return
	}
	defer receiverNode.Stop()
	time.Sleep(1 * time.Second)

	// Get receiver node's multiaddress
	receiverMultiaddr, err := receiverNode.ListenAddresses()
	if err != nil {
		fmt.Printf("Failed to get receiver node addresses: %v\n", err)
		return
	}

	// Check initial peer counts
	dialerPeerCount, err := dialerNode.GetNumConnectedPeers()
	if err != nil {
		fmt.Printf("Failed to get dialer peer count: %v\n", err)
		return
	}
	fmt.Printf("Dialer initial peer count: %d\n", dialerPeerCount)

	receiverPeerCount, err := receiverNode.GetNumConnectedPeers()
	if err != nil {
		fmt.Printf("Failed to get receiver peer count: %v\n", err)
		return
	}
	fmt.Printf("Receiver initial peer count: %d\n", receiverPeerCount)

	// Dial peer
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	if err := dialerNode.Connect(ctx, receiverMultiaddr[0]); err != nil {
		fmt.Printf("Failed to dial peer: %v\n", err)
		return
	}
	time.Sleep(1 * time.Second)

	// Check final peer counts
	dialerPeerCount, err = dialerNode.GetNumConnectedPeers()
	if err != nil {
		fmt.Printf("Failed to get dialer peer count: %v\n", err)
		return
	}
	fmt.Printf("Dialer final peer count: %d\n", dialerPeerCount)

	receiverPeerCount, err = receiverNode.GetNumConnectedPeers()
	if err != nil {
		fmt.Printf("Failed to get receiver peer count: %v\n", err)
		return
	}
	fmt.Printf("Receiver final peer count: %d\n", receiverPeerCount)

	go func() {
		for envelope := range receiverNode.MsgChan {
			if envelope.Message().ContentTopic == ContentTopic {
				fmt.Printf("Received message: %s\n", string(envelope.Message().Payload))
			}
		}
	}()

	for i := 0; i < 10; i++ {
		payload := fmt.Sprintf("Message %d", i)
		ctx := context.Background()
		msg := &pb.WakuMessage{
			Payload:      []byte(payload),
			ContentTopic: ContentTopic,
		}
		dialerNode.RelayPublish(ctx, msg, "/waku/2/rs/16/64")
		time.Sleep(1 * time.Second)
	}

	time.Sleep(3 * time.Second)
}
