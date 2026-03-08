package node

import (
	"context"
	"fmt"
	"io"
	"math"

	pb "github.com/getoptimum/mump2p-cli/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	CommandPublishData       int32 = 1
	CommandSubscribeToTopic  int32 = 2
	CommandSubscribeToTopics int32 = 4
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.CommandStreamClient
}

func NewClient(nodeAddr string) (*Client, error) {
	conn, err := grpc.NewClient(nodeAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt),
			grpc.MaxCallSendMsgSize(math.MaxInt),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to node %s: %w", nodeAddr, err)
	}
	return &Client{
		conn:   conn,
		client: pb.NewCommandStreamClient(conn),
	}, nil
}

// Subscribe opens a bidi stream, sends a subscribe command, and returns a
// channel that delivers raw message payloads received from the mesh.
func (c *Client) Subscribe(ctx context.Context, ticket, topic string, bufSize int) (<-chan *pb.Response, error) {
	stream, err := c.client.ListenCommands(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open command stream: %w", err)
	}

	if err := stream.Send(&pb.Request{
		Command:  CommandSubscribeToTopic,
		Topic:    topic,
		JwtToken: ticket,
	}); err != nil {
		return nil, fmt.Errorf("failed to send subscribe command: %w", err)
	}

	ch := make(chan *pb.Response, bufSize)
	go func() {
		defer close(ch)
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				select {
				case <-ctx.Done():
				default:
					fmt.Printf("stream error: %v\n", err)
				}
				return
			}
			select {
			case ch <- resp:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

// Publish opens a bidi stream, sends a publish command, and reads back the
// first response (typically a MessageTrace confirmation).
func (c *Client) Publish(ctx context.Context, ticket, topic string, data []byte) (*pb.Response, error) {
	stream, err := c.client.ListenCommands(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open command stream: %w", err)
	}

	if err := stream.Send(&pb.Request{
		Command:  CommandPublishData,
		Topic:    topic,
		Data:     data,
		JwtToken: ticket,
	}); err != nil {
		return nil, fmt.Errorf("failed to send publish command: %w", err)
	}

	if err := stream.CloseSend(); err != nil {
		return nil, fmt.Errorf("failed to close send: %w", err)
	}

	resp, err := stream.Recv()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to receive publish response: %w", err)
	}
	return resp, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
