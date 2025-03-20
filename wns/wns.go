package wns

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/waku-go-bindings/waku"
	"github.com/waku-org/waku-go-bindings/waku/common"
)

const requestTimeout = 30 * time.Second

func SetupWakuNode() (*waku.WakuNode, error) {

	tcpPort, udpPort, err := waku.GetFreePortIfNeeded(0, 0)
	if err != nil {
		fmt.Printf("Failed getting free ports: %v\n", err)
		return nil, err
	}

	// Configure server node
	serverNodeWakuConfig := common.WakuConfig{
		Relay:           true,
		LogLevel:        "DEBUG",
		Discv5Discovery: true,
		DnsDiscoveryUrl: "enrtree://AMOJVZX4V6EXP7NTJPMAYJYST2QP6AJXYW76IU6VGJS7UVSNDYZG4@boot.prod.status.nodes.status.im",
		ClusterID:       16,
		Shards:          []uint16{64},
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
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

	options := func(b *backoff.ExponentialBackOff) {
		b.MaxElapsedTime = 30 * time.Second
	}

	// Sanity check, not great, but it's probably helpful
	err = waku.RetryWithBackOff(func() error {
		numConnected, err := serverNode.GetNumConnectedPeers()
		if err != nil {
			return err
		}
		// Wait for it to discover peers
		if numConnected > 2 {
			return nil
		}
		return errors.New("could not discover enough peers")
	}, options)

	if err != nil {
		fmt.Printf("Failed to setup server node: %v\n", err)
		return nil, err
	}

	return serverNode, nil
}

func StartWnsServer(serverNode *waku.WakuNode, keyInfo *payload.KeyInfo) {
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

func GenerateKeys() (*payload.KeyInfo, error) {
	var err error
	var keyInfo *payload.KeyInfo = new(payload.KeyInfo)
	keyInfo.PrivKey, err = crypto.GenerateKey()

	if err != nil {
		fmt.Printf("Failed generating keys: %v\n", err)
		return nil, err
	}

	keyInfo.Kind = payload.Asymmetric
	keyInfo.PubKey = keyInfo.PrivKey.PublicKey

	return keyInfo, nil
}

func FillKeysFromEnv() (*payload.KeyInfo, error) {

	var err error
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		return nil, errors.New("PRIVATE_KEY env variable is empty")
	}

	var keyInfo *payload.KeyInfo = new(payload.KeyInfo)

	keyInfo.PrivKey, err = crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		fmt.Printf("Failed converting hex to private key: %v\n", err)
		os.Exit(1)
	}

	keyInfo.Kind = payload.Asymmetric
	keyInfo.PubKey = keyInfo.PrivKey.PublicKey

	return keyInfo, nil

}

func PubKeyToHex(pubKey *ecdsa.PublicKey) string {
	pubKeyBytes := crypto.FromECDSAPub(pubKey)
	pubKeyHex := hex.EncodeToString(pubKeyBytes)
	return pubKeyHex
}

func HexToPubKey(pubKeyHex string) (*ecdsa.PublicKey, error) {

	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		fmt.Printf("Failed converting hex to public key: %v\n", err)
		return nil, err
	}
	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		fmt.Printf("Failed re-converting hex to public key: %v\n", err)
		os.Exit(1)
	}

	return pubKey, nil
}
