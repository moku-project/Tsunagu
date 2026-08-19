// client.go
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

func (c *Client) AddRepository(ctx context.Context, indexURL string) (*sandboxv1.Repository, error) {
	return c.rpc.AddRepository(ctx, &sandboxv1.AddRepositoryRequest{
		IndexUrl: indexURL,
	})
}

func (c *Client) ListRepositories(ctx context.Context) (*sandboxv1.RepositoryList, error) {
	return c.rpc.ListRepositories(ctx, &sandboxv1.Empty{})
}

func (c *Client) ListAvailableExtensions(ctx context.Context, repositoryID string) (*sandboxv1.ExtensionList, error) {
	return c.rpc.ListAvailableExtensions(ctx, &sandboxv1.ListAvailableExtensionsRequest{
		RepositoryId: repositoryID,
	})
}

func (c *Client) InstallExtension(ctx context.Context, repositoryID, extensionID string) (*sandboxv1.Extension, error) {
	return c.rpc.InstallExtension(ctx, &sandboxv1.InstallExtensionRequest{
		RepositoryId: repositoryID,
		ExtensionId:  extensionID,
	})
}

func (c *Client) ListInstalledExtensions(ctx context.Context) (*sandboxv1.ExtensionList, error) {
	return c.rpc.ListInstalledExtensions(ctx, &sandboxv1.Empty{})
}

func (c *Client) UninstallExtension(ctx context.Context, extensionID string) (*sandboxv1.Empty, error) {
	return c.rpc.UninstallExtension(ctx, &sandboxv1.ExtensionRequest{
		ExtensionId: extensionID,
	})
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