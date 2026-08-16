package sandbox

import (
	"context"
	"fmt"

	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn *grpc.ClientConn
	rpc  sandboxv1.ExtensionServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial sandbox at %s: %w", addr, err)
	}
	return &Client{
		conn: conn,
		rpc:  sandboxv1.NewExtensionServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Search(ctx context.Context, extensionID, query string, page int32) (*sandboxv1.SearchResponse, error) {
	return c.rpc.Search(ctx, &sandboxv1.SearchRequest{
		ExtensionId: extensionID,
		Query:       query,
		Page:        page,
	})
}