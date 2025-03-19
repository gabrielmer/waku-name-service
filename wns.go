package wns

import (
	"fmt"
	"time"

	"github.com/waku-org/waku-go-bindings/waku"
	"github.com/waku-org/waku-go-bindings/waku/common"
)

func createWnsServer() (*waku.WakuNode, error) {

	const requestTimeout = 30 * time.Second
	// Configure dialer node
	serverNodeWakuConfig := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: false,
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   9020,
		TcpPort:         60020,
	}

	// Create and start dialer node
	serverNode, err := waku.NewWakuNode(&serverNodeWakuConfig, "dialerNode")
	if err != nil {
		fmt.Printf("Failed to create dialer node: %v\n", err)
		return nil, err
	}
	if err := serverNode.Start(); err != nil {
		fmt.Printf("Failed to start dialer node: %v\n", err)
		return nil, err
	}
	return serverNode, nil
}
