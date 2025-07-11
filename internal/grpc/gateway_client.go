package grpc

import (
	"context"
	"fmt"
	"io"
	"math"
	"time"

	proto "github.com/getoptimum/mump2p-cli/proto/grpc"
	grpcClient "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// GatewayClient handles gRPC streaming connections to the gateway
type GatewayClient struct {
	conn   *grpcClient.ClientConn
	client proto.GatewayStreamClient
}

// NewGatewayClient creates a new gRPC gateway client
func NewGatewayClient(gatewayAddr string) (*GatewayClient, error) {
	conn, err := grpcClient.Dial(gatewayAddr,
		grpcClient.WithTransportCredentials(insecure.NewCredentials()),
		grpcClient.WithDefaultCallOptions(
			grpcClient.MaxCallRecvMsgSize(math.MaxInt),
			grpcClient.MaxCallSendMsgSize(math.MaxInt),
		),
		grpcClient.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                2 * time.Minute,
			Timeout:             20 * time.Second,
			PermitWithoutStream: false,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gateway: %v", err)
	}

	client := proto.NewGatewayStreamClient(conn)
	return &GatewayClient{
		conn:   conn,
		client: client,
	}, nil
}

// Subscribe starts a gRPC stream subscription and returns a channel for receiving messages
func (gc *GatewayClient) Subscribe(ctx context.Context, clientID string) (<-chan *proto.GatewayMessage, error) {
	stream, err := gc.client.ClientStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %v", err)
	}

	// Send initial client ID message
	if err := stream.Send(&proto.GatewayMessage{
		ClientId: clientID,
		Type:     "subscribe",
	}); err != nil {
		return nil, fmt.Errorf("failed to send client ID: %v", err)
	}

	// Create channel for receiving messages
	msgChan := make(chan *proto.GatewayMessage, 100)

	// Start goroutine to receive messages
	go func() {
		defer close(msgChan)
		defer stream.CloseSend()

		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				// Log error but don't close channel immediately to allow reconnection
				fmt.Printf("Stream receive error: %v\n", err)
				return
			}

			select {
			case msgChan <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	return msgChan, nil
}

// Close closes the gRPC connection
func (gc *GatewayClient) Close() error {
	return gc.conn.Close()
}
