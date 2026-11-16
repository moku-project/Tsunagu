package graph

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"tsunagu/backend/internal/api/graph/model"
	"tsunagu/backend/internal/auth"
	"tsunagu/backend/internal/automation"
	"tsunagu/backend/internal/config"
	"tsunagu/backend/internal/contentfilter"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/download"
	"tsunagu/backend/internal/flaresolverr"
	"tsunagu/backend/internal/localsource"
	"tsunagu/backend/internal/metadata"
	"tsunagu/backend/internal/mokusync"
	"tsunagu/backend/internal/sandbox"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
	"tsunagu/backend/internal/streamresolve"
	syncpkg "tsunagu/backend/internal/sync"
	"tsunagu/backend/internal/tracker"
)

type Resolver struct {
	Sy        *syncpkg.Syncer
	Sc        *sandbox.SupervisedClient
	Dm        *download.Manager
	Ls        *localsource.Scanner
	Tk        *tracker.Manager
	Md        *metadata.Manager
	Sr        *streamresolve.Resolver
	Q         *sqlcgen.Queries
	DB        *sql.DB
	Fs        *flaresolverr.Manager
	Cfg       *config.Store
	Cf        *contentfilter.Manager
	Au        *automation.Manager
	Am        *auth.Manager
	Ms        *mokusync.Engine
	MediaDir  string
	Name      string
	Version   string
	BuildTime string

	storageInfoMu      sync.Mutex
	storageInfoCache   *model.StorageInfo
	storageInfoCacheAt time.Time
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
	return r.Sy.PersistExtensionMeta(ctx, ext, loaded.GetExtensions()[0])
}

func (r *mutationResolver) refreshMediaFull(ctx context.Context, c *sandbox.Client, id int64, syncChapters bool) (sqlcgen.Medium, error) {
	entry, err := r.Sy.RefreshMetadata(ctx, c, id, syncChapters)
	if err != nil {
		return sqlcgen.Medium{}, err
	}
	if enriched, mErr := r.Md.Refresh(ctx, id); mErr == nil {
		entry = enriched
	}
	r.Tk.SyncMediaProgress(ctx, id)
	if m, mErr := r.Q.GetMedia(ctx, id); mErr == nil {
		entry = m
	}
	return entry, nil
}
