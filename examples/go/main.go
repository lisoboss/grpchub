package main

import (
	"context"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lisoboss/grpchub-go"
	"github.com/lisoboss/grpchub-go/grpcx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Example service definition (you would normally import this from your proto package)
type EchoService struct{}

func (s *EchoService) Echo(ctx context.Context, req *EchoRequest) (*EchoResponse, error) {
	return &EchoResponse{Message: "Echo: " + req.Message}, nil
}

func (s *EchoService) Ping(ctx context.Context, req *emptypb.Empty) (*PingResponse, error) {
	return &PingResponse{Message: "Pong", Timestamp: time.Now().Unix()}, nil
}

// Example proto messages (you would normally generate these)
type EchoRequest struct {
	Message string `json:"message"`
}

type EchoResponse struct {
	Message string `json:"message"`
}

type PingResponse struct {
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// Load TLS credentials from PEM file
func loadTLSCredentialsFromPEM(pemFile string) ([]byte, []byte, []byte, error) {
	data, err := os.ReadFile(pemFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read pem file: %w", err)
	}

	var certPEM, keyPEM, caPEM []byte
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		switch block.Type {
		case "CERTIFICATE":
			// First certificate is client cert, last one is CA
			if certPEM == nil {
				certPEM = pem.EncodeToMemory(block)
			} else {
				caPEM = append(caPEM, pem.EncodeToMemory(block)...)
			}
		case "PRIVATE KEY":
			keyPEM = pem.EncodeToMemory(block)
		}
	}

	if certPEM == nil || keyPEM == nil || caPEM == nil {
		return nil, nil, nil, fmt.Errorf("incomplete PEM data (cert/key/ca)")
	}

	return caPEM, certPEM, keyPEM, nil
}

// Create GrpcHub client
func createGrpcHubClient() (*grpchub.GrpcHubClient, error) {
	// Load TLS credentials from PEM file
	caPEM, certPEM, keyPEM, err := loadTLSCredentialsFromPEM("../../deploy/certs/client.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
	}

	// Create GrpcHub client
	ghc, err := grpchub.NewGrpcHubClient("[::1]:50055", caPEM, certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to create GrpcHub client: %w", err)
	}

	return ghc, nil
}

// Run server example
func runServer() {
	log.Println("Starting server example...")

	// Create GrpcHub client
	ghc, err := createGrpcHubClient()
	if err != nil {
		log.Fatal("Failed to create GrpcHub client:", err)
	}
	defer ghc.Close()

	// Create gRPC server through GrpcHub
	grpcSrv, err := grpcx.NewServer("echo-server", ghc)
	if err != nil {
		log.Fatal("Failed to create gRPC server:", err)
	}

	// Register your service (in real use, you'd use generated registration functions)
	// pb.RegisterEchoServiceServer(grpcSrv, &EchoService{})
	log.Println("Echo service registered")

	// Start serving
	log.Println("Server listening through GrpcHub...")
	if err := grpcSrv.Serve(); err != nil {
		log.Fatal("Failed to serve:", err)
	}
}

// Run client example
func runClient() {
	log.Println("Starting client example...")

	// Create GrpcHub client
	ghc, err := createGrpcHubClient()
	if err != nil {
		log.Fatal("Failed to create GrpcHub client:", err)
	}
	defer ghc.Close()

	// Create gRPC connection through GrpcHub
	conn, err := grpcx.NewClient("echo-client", ghc)
	if err != nil {
		log.Fatal("Failed to create gRPC client:", err)
	}
	defer conn.Close()

	// Create your service client (in real use, you'd use generated client constructors)
	// client := pb.NewEchoServiceClient(conn)
	log.Println("Echo client created")

	// Example usage
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Make calls (in real use, you'd call actual methods)
	// response, err := client.Echo(ctx, &pb.EchoRequest{Message: "Hello GrpcHub!"})
	// if err != nil {
	//     log.Fatal("Failed to call Echo:", err)
	// }
	// log.Printf("Echo response: %s", response.Message)

	// pingResp, err := client.Ping(ctx, &emptypb.Empty{})
	// if err != nil {
	//     log.Fatal("Failed to call Ping:", err)
	// }
	// log.Printf("Ping response: %s at %d", pingResp.Message, pingResp.Timestamp)

	log.Println("Client calls completed successfully")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [server|client]")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  go run main.go server   # Run as server")
		fmt.Println("  go run main.go client   # Run as client")
		fmt.Println("")
		fmt.Println("Note: Make sure GrpcHub server is running and certificates are available at:")
		fmt.Println("  ../../deploy/certs/client.pem")
		os.Exit(1)
	}

	mode := os.Args[1]
	switch mode {
	case "server":
		runServer()
	case "client":
		runClient()
	default:
		log.Fatal("Invalid mode. Use 'server' or 'client'")
	}
}
