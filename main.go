package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gabrielmer/waku-name-service/wns"
	"github.com/joho/godotenv"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/waku-go-bindings/waku"
	"google.golang.org/protobuf/proto"
)

const AppName = "wns-server"
const AppVersion = "1"

const ServerPubKeyHex = "04e3cd903bec9535b87077c94b950b69a5d669875070566125f7dd7a586a4eefc137db9d6906fa123b64947433ae104afd9935bedb75ce67dc1238e95a78d29111"

func sendMessage(serverPubKeyHex string) error {

	var serverKeyInfo *payload.KeyInfo = new(payload.KeyInfo)
	var err error
	serverKeyInfo.Kind = payload.Asymmetric
	pubKey, err := wns.HexToPubKey(serverPubKeyHex)
	if err != nil {
		fmt.Printf("Failed to parse server's public key: %v\n", err)
		return errors.New("could not parse server's public key")
	}
	serverKeyInfo.PubKey = *pubKey

	var senderKeyInfo *payload.KeyInfo = new(payload.KeyInfo)
	senderKeyInfo, err = wns.GenerateKeys()
	if err != nil {
		fmt.Printf("Failed generating sender's public key: %v\n", err)
		return errors.New("could not parse sender's public key")
	}

	senderHexPubKey := wns.PubKeyToHex(&senderKeyInfo.PubKey)

	testSenderNode, err := wns.SetupWakuNode()
	if err != nil {
		fmt.Printf("Failed to start test sender node: %v\n", err)
		return errors.New("could not start test sender node")
	}
	defer testSenderNode.Stop()

	req := wns.Request{
		RequestID: "1234",
		PublicKey: senderHexPubKey,
		Service:   "ResolveWallet",
		Input:     "",
	}

	jsonBytes, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("Failed generating json bytes: %v\n", err)
		return errors.New("could not generate json bytes")
	}

	// Send test message
	message := &pb.WakuMessage{
		Payload:      []byte(string(jsonBytes)),
		ContentTopic: wns.PubKeyHexToContentTopic(serverPubKeyHex),
		Version:      proto.Uint32(1),
		Timestamp:    proto.Int64(time.Now().UnixNano()),
	}

	err = payload.EncodeWakuMessage(message, serverKeyInfo)
	if err != nil {
		fmt.Printf("Failed to encode message: %v\n", err)
		return errors.New("could not encode message")
	}

	pubsubTopic := waku.FormatWakuRelayTopic(16, 64)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	testSenderNode.RelayPublish(ctx, message, pubsubTopic)

	return nil

}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Set up wake nodes first
	wakuNode, err := wns.SetupWakuNode()
	if err != nil {
		fmt.Printf("Failed to start server node: %v\n", err)
		os.Exit(1)
	}
	defer wakuNode.Stop()

	serverKeyInfo, err := wns.FillKeysFromEnv()
	if err != nil {
		fmt.Printf("Failed fetching server keys: %v\n", err)
		os.Exit(1)
	}

	// Start server
	go wns.StartWnsServer(wakuNode, serverKeyInfo)

	err = sendMessage(ServerPubKeyHex)
	if err != nil {
		fmt.Printf("Failed sending message: %v\n", err)
		os.Exit(1)
	}

	/* serverPubKeyHex := wns.PubKeyToHex(&serverKeyInfo.PubKey)
	fmt.Println("-------------- serverPubKeyHex: ", serverPubKeyHex) */

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
