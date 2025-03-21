package wns

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/waku-go-bindings/waku"
	"github.com/waku-org/waku-go-bindings/waku/common"
	"google.golang.org/protobuf/proto"
)

const requestTimeout = 30 * time.Second

type Request struct {
	RequestID string `json:"requestId"`
	PublicKey string `json:"publicKey"`
	Service   string `json:"service"`
	Input     string `json:"input"`
}

type Response struct {
	RequestID string `json:"requestId"`
	Output    string `json:"output"`
	Status    string `json:"service"`
	Message   string `json:"message"`
}

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

	pubKeyHex := PubKeyToHex(&keyInfo.PubKey)
	contentTopic := PubKeyHexToContentTopic(pubKeyHex)

	for {
		select {
		case envelope := <-serverNode.MsgChan:
			if envelope != nil {
				// Print the payload
				fmt.Printf("Content topic: %s\n", envelope.Message().ContentTopic)
				if envelope.Message().ContentTopic == contentTopic {
					handleReceivedMessage(serverNode, envelope, keyInfo)
				}
			}
		}
	}
}

func handleReceivedMessage(serverNode *waku.WakuNode, envelope common.Envelope, keyInfo *payload.KeyInfo) {
	fmt.Printf("Received message with payload: %v\n", envelope.Message().Payload)
	payload.DecodeWakuMessage(envelope.Message(), keyInfo)
	fmt.Printf("Decoded payload: %s\n", envelope.Message().Payload)

	var req Request
	err := json.Unmarshal([]byte(envelope.Message().Payload), &req)
	if err != nil {
		fmt.Println("Invalid JSON:", err)
		return
	}

	fmt.Println("Received request")
	fmt.Printf("Request ID: %s\n", req.RequestID)
	fmt.Printf("Public Key: %s\n", req.PublicKey)
	fmt.Printf("Service: %s\n", req.Service)
	fmt.Printf("Input: %s\n", req.Input)

	var senderKeyInfo *payload.KeyInfo = new(payload.KeyInfo)

	senderKeyInfo.Kind = payload.Asymmetric
	pubKey, err := HexToPubKey(req.PublicKey)
	if err != nil {
		fmt.Printf("Failed to parse sender's public key: %v\n", err)
		return
	}
	senderKeyInfo.PubKey = *pubKey

	if req.Service == "ResolveWallet" {
		handleResolveWallet(serverNode, req, senderKeyInfo)
	}

}

func handleResolveWallet(serverNode *waku.WakuNode, req Request, senderKeyInfo *payload.KeyInfo) {

	ethAddress, err := GenerateEthereumAddress()
	if err != nil {
		fmt.Printf("Failed generating ethereum address: %v\n", err)
		return
	}

	res := Response{
		RequestID: req.RequestID,
		Output:    ethAddress,
		Status:    "200",
		Message:   "OK",
	}

	jsonBytes, err := json.Marshal(res)
	if err != nil {
		fmt.Printf("Failed generating json bytes: %v\n", err)
		return
	}

	// fmt.Println("--------------- string(jsonBytes): ", string(jsonBytes))
	//fmt.Println("--------------- []byte(string(jsonBytes)): ", []byte(string(jsonBytes)))

	// Send test message
	message := &pb.WakuMessage{
		Payload:      []byte(string(jsonBytes)),
		ContentTopic: PubKeyHexToContentTopic(req.PublicKey),
		Version:      proto.Uint32(1),
		Timestamp:    proto.Int64(time.Now().UnixNano()),
	}

	err = payload.EncodeWakuMessage(message, senderKeyInfo)
	if err != nil {
		fmt.Printf("Failed to encode message: %v\n", err)
	}

	// fmt.Println("----------- encoded bytes: ", message.Payload)

	pubsubTopic := waku.FormatWakuRelayTopic(16, 64)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fmt.Println("--------- sending response!")
	serverNode.RelayPublish(ctx, message, pubsubTopic)

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

func GenerateEthereumAddress() (string, error) {

	keyInfo, err := GenerateKeys()
	if err != nil {
		return "", err
	}

	address := crypto.PubkeyToAddress(keyInfo.PubKey)
	fmt.Println("Ethereum Address:", address.Hex())

	return address.Hex(), nil
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

func PubKeyHexToContentTopic(pubKeyHex string) string {
	key := ""
	if len(pubKeyHex) <= 16 {
		key = pubKeyHex
	} else {
		key = pubKeyHex[:16]
	}

	contentTopic := fmt.Sprintf("/wns/1/%s/proto", key)
	return contentTopic
}
