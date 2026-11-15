package server

import (
	"context"

	"tsunagu/backend/internal/sandbox"
	"tsunagu/backend/internal/sync"
)

// syncResolver fetches a synced library title through the same path as opening it in the app.
type syncResolver struct {
	sy *sync.Syncer
	sc *sandbox.SupervisedClient
}

func (r syncResolver) Resolve(ctx context.Context, packageName, externalID string) error {
	c, err := r.sc.Ensure(ctx)
	if err != nil {
		return err
	}
	_, err = r.sy.ResolveMedia(ctx, c, packageName, externalID, true)
	return err
}
