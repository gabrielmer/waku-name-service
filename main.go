package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gabrielmer/waku-name-service/wns"
	"github.com/joho/godotenv"
	"github.com/waku-org/go-waku/waku/v2/payload"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/waku-go-bindings/waku"
	"google.golang.org/protobuf/proto"
)

const AppName = "wns-server"
const AppVersion = "1"

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Now read environment variables as usual
	privateKeyHex := os.Getenv("PRIVATE_KEY")

	// Set up wake nodes first
	wakuNode, err := wns.SetupWakuNode()
	if err != nil {
		fmt.Printf("Failed to start server node: %v\n", err)
		os.Exit(1)
	}
	defer wakuNode.Stop()

	serverKeyInfo, err := wns.GenerateKeys()
	if err != nil {
		fmt.Printf("Failed generating server keys: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("------- privateKeyHex: ", privateKeyHex)
	privKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		fmt.Printf("Failed re-converting hex to private key: %v\n", err)
		os.Exit(1)
	}

	pubBytes := crypto.FromECDSAPub(&privKey.PublicKey)
	pubHex := hex.EncodeToString(pubBytes)
	fmt.Println("------ hex public key: ", pubHex)
	pubKey2, err := crypto.UnmarshalPubkey(pubBytes)
	if err != nil {
		fmt.Printf("Failed re-converting hex to public key: %v\n", err)
		os.Exit(1)
	}
	pubBytes2 := crypto.FromECDSAPub(pubKey2)
	pubHex2 := hex.EncodeToString(pubBytes2)
	fmt.Println("------ hex public key 2: ", pubHex2)

	// Start server
	go wns.StartWnsServer(wakuNode, serverKeyInfo)

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
		Version:      proto.Uint32(1),
		Timestamp:    proto.Int64(time.Now().UnixNano()),
	}

	err = payload.EncodeWakuMessage(message, serverKeyInfo)
	if err != nil {
		fmt.Printf("Failed to encode message: %v\n", err)
		os.Exit(1)
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
