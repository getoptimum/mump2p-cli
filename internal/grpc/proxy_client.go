package grpc

import (
	"context"
	"fmt"
	"io"
	"math"

	proto "github.com/getoptimum/mump2p-cli/proto"
	grpcClient "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ProxyClient handles gRPC streaming connections to the proxy
type ProxyClient struct {
	conn   *grpcClient.ClientConn
	client proto.ProxyStreamClient
}

// NewProxyClient creates a new gRPC proxy client
func NewProxyClient(proxyAddr string) (*ProxyClient, error) {
	conn, err := grpcClient.Dial(proxyAddr,
		grpcClient.WithTransportCredentials(insecure.NewCredentials()),
		grpcClient.WithDefaultCallOptions(
			grpcClient.MaxCallRecvMsgSize(math.MaxInt),
			grpcClient.MaxCallSendMsgSize(math.MaxInt),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to proxy: %v", err)
	}

	client := proto.NewProxyStreamClient(conn)
	return &ProxyClient{
		conn:   conn,
		client: client,
	}, nil
}

// Subscribe starts a gRPC stream subscription and returns a channel for receiving messages
func (pc *ProxyClient) Subscribe(ctx context.Context, clientID string, bufferSize int) (<-chan *proto.ProxyMessage, error) {
	stream, err := pc.client.ClientStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %v", err)
	}

	// Send initial client ID message
	if err := stream.Send(&proto.ProxyMessage{
		ClientId: clientID,
		Type:     "subscribe",
	}); err != nil {
		return nil, fmt.Errorf("failed to send client ID: %v", err)
	}

	// Create channel for receiving messages
	msgChan := make(chan *proto.ProxyMessage, bufferSize)

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
func (pc *ProxyClient) Close() error {
	return pc.conn.Close()
}

// Publish sends a message to a topic via gRPC
func (pc *ProxyClient) Publish(ctx context.Context, clientID, topic string, message []byte) error {
	req := &proto.PublishRequest{
		ClientId: clientID,
		Topic:    topic,
		Message:  message,
	}

	resp, err := pc.client.Publish(ctx, req)
	if err != nil {
		return fmt.Errorf("gRPC publish failed: %v", err)
	}

	if resp.Status != "published" {
		return fmt.Errorf("publish failed with status: %s", resp.Status)
	}

	return nil
}
