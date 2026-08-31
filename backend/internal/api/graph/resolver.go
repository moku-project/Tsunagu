package graph

import (
	"context"
	"fmt"
	"log"

	"tsunagu/backend/internal/config"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/download"
	"tsunagu/backend/internal/flaresolverr"
	"tsunagu/backend/internal/localsource"
	"tsunagu/backend/internal/metadata"
	"tsunagu/backend/internal/sandbox"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
	"tsunagu/backend/internal/streamresolve"
	"tsunagu/backend/internal/sync"
	"tsunagu/backend/internal/tracker"
)

type Resolver struct {
	Sy        *sync.Syncer
	Sc        *sandbox.SupervisedClient
	Dm        *download.Manager
	Ls        *localsource.Scanner
	Tk        *tracker.Manager
	Md        *metadata.Manager
	Sr        *streamresolve.Resolver
	Q         *sqlcgen.Queries
	Fs        *flaresolverr.Manager
	Cfg       *config.Store
	MediaDir  string
	Name      string
	Version   string
	BuildTime string
}

func (r *Resolver) validateChapterMedia(ctx context.Context, mediaID, chapterID int64) (sqlcgen.Chapter, error) {
	ch, err := r.Q.GetChapter(ctx, chapterID)
	if err != nil {
		return ch, fmt.Errorf("chapter %d: %w", chapterID, err)
	}
	if ch.MediaID != mediaID {
		return ch, fmt.Errorf("chapter %d does not belong to media %d", chapterID, mediaID)
	}
	return ch, nil
}

func (r *Resolver) persistSupportsLatest(ctx context.Context, ext sqlcgen.Extension, loaded *sandboxv1.ExtensionList) sqlcgen.Extension {
	if len(loaded.GetExtensions()) == 0 {
		return ext
	}
	supportsLatest := loaded.GetExtensions()[0].GetSupportsLatest()
	if supportsLatest == ext.SupportsLatest {
		return ext
	}
	updated, err := r.Q.UpdateExtensionSupportsLatest(ctx, sqlcgen.UpdateExtensionSupportsLatestParams{
		SupportsLatest: supportsLatest,
		ID:             ext.ID,
	})
	if err != nil {
		log.Printf("resolver: persisting supportsLatest for %s failed: %v", ext.PackageName, err)
		return ext
	}
	return updated
}
