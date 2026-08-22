package sandbox

import (
	"context"
	"fmt"
	"time"

	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const defaultCallTimeout = 15 * time.Second

type Client struct {
	conn        *grpc.ClientConn
	rpc         sandboxv1.ExtensionServiceClient
	callTimeout time.Duration
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`),
	)
	if err != nil {
		return nil, fmt.Errorf("dial sandbox at %s: %w", addr, err)
	}
	return &Client{
		conn:        conn,
		rpc:         sandboxv1.NewExtensionServiceClient(conn),
		callTimeout: defaultCallTimeout,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.callTimeout)
}

func (c *Client) LoadExtensions(ctx context.Context, exts []*sandboxv1.ExtensionToLoad) (*sandboxv1.ExtensionList, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.rpc.LoadExtensions(ctx, &sandboxv1.LoadExtensionsRequest{Extensions: exts})
}

func (c *Client) ListLoadedExtensions(ctx context.Context) (*sandboxv1.ExtensionList, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.rpc.ListLoadedExtensions(ctx, &sandboxv1.Empty{})
}

func (c *Client) UnloadExtension(ctx context.Context, extensionID string) (*sandboxv1.Empty, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.rpc.UnloadExtension(ctx, &sandboxv1.ExtensionRequest{ExtensionId: extensionID})
}

func (c *Client) Search(ctx context.Context, extensionID, query string, page int32) (*sandboxv1.SearchResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.rpc.Search(ctx, &sandboxv1.SearchRequest{ExtensionId: extensionID, Query: query, Page: page})
}

func (c *Client) GetDetails(ctx context.Context, extensionID, sourceEntryID string) (*sandboxv1.EntryDetails, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.rpc.GetDetails(ctx, &sandboxv1.EntryRequest{ExtensionId: extensionID, SourceEntryId: sourceEntryID})
}

func (c *Client) GetChapters(ctx context.Context, extensionID, sourceEntryID string) (*sandboxv1.ChapterList, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.rpc.GetChapters(ctx, &sandboxv1.EntryRequest{ExtensionId: extensionID, SourceEntryId: sourceEntryID})
}

func (c *Client) GetPages(ctx context.Context, extensionID, sourceEntryID, sourceChapterID string) (*sandboxv1.PageList, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.rpc.GetPages(ctx, &sandboxv1.ChapterRequest{
		ExtensionId:     extensionID,
		SourceEntryId:   sourceEntryID,
		SourceChapterId: sourceChapterID,
	})
}

func (c *Client) GetChapterText(ctx context.Context, extensionID, sourceEntryID, sourceChapterID string) (*sandboxv1.TextContent, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.rpc.GetChapterText(ctx, &sandboxv1.ChapterRequest{
		ExtensionId:     extensionID,
		SourceEntryId:   sourceEntryID,
		SourceChapterId: sourceChapterID,
	})
}

func (c *Client) GetEpisodes(ctx context.Context, extensionID, sourceEntryID string) (*sandboxv1.EpisodeList, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.rpc.GetEpisodes(ctx, &sandboxv1.EntryRequest{ExtensionId: extensionID, SourceEntryId: sourceEntryID})
}

func (c *Client) GetVideoStream(ctx context.Context, extensionID, sourceEntryID, sourceEpisodeID string) (*sandboxv1.StreamInfo, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.rpc.GetVideoStream(ctx, &sandboxv1.EpisodeRequest{
		ExtensionId:      extensionID,
		SourceEntryId:    sourceEntryID,
		SourceEpisodeId:  sourceEpisodeID,
	})
}

func (c *Client) GetImageBytes(ctx context.Context, extensionID, imageURL string) (*sandboxv1.ImageData, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.rpc.GetImageBytes(ctx, &sandboxv1.ImageRequest{
		ExtensionId: extensionID,
		ImageUrl:    imageURL,
	})
}