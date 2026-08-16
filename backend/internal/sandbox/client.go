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

func (c *Client) GetDetails(ctx context.Context, extensionID, sourceEntryID string) (*sandboxv1.EntryDetails, error) {
	return c.rpc.GetDetails(ctx, &sandboxv1.EntryRequest{
		ExtensionId:   extensionID,
		SourceEntryId: sourceEntryID,
	})
}

func (c *Client) GetChapters(ctx context.Context, extensionID, sourceEntryID string) (*sandboxv1.ChapterList, error) {
	return c.rpc.GetChapters(ctx, &sandboxv1.EntryRequest{
		ExtensionId:   extensionID,
		SourceEntryId: sourceEntryID,
	})
}

func (c *Client) GetPages(ctx context.Context, extensionID, sourceEntryID, sourceChapterID string) (*sandboxv1.PageList, error) {
	return c.rpc.GetPages(ctx, &sandboxv1.ChapterRequest{
		ExtensionId:     extensionID,
		SourceEntryId:   sourceEntryID,
		SourceChapterId: sourceChapterID,
	})
}