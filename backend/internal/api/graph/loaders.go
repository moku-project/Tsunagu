package graph

import (
	"context"

	"github.com/vikstrous/dataloadgen"

	"tsunagu/backend/internal/db/sqlcgen"
)

type Loaders struct {
	ChapterCountByMedia    *dataloadgen.Loader[int64, int32]
	UnreadCountByMedia     *dataloadgen.Loader[int64, int32]
	DownloadedCountByMedia *dataloadgen.Loader[int64, int32]
	TagsByMedia            *dataloadgen.Loader[int64, []string]
	GenresByMedia          *dataloadgen.Loader[int64, []string]
	ProgressByMedia        *dataloadgen.Loader[int64, []sqlcgen.ReadingProgress]
	FoldersByMedia         *dataloadgen.Loader[int64, []sqlcgen.Folder]
	NextUnreadByMedia      *dataloadgen.Loader[int64, *sqlcgen.Chapter]
	LatestChapterByMedia   *dataloadgen.Loader[int64, *sqlcgen.Chapter]
	ExtensionByID          *dataloadgen.Loader[int64, *sqlcgen.Extension]
	MediaByID              *dataloadgen.Loader[int64, *sqlcgen.Medium]
	TrackLinksByMedia      *dataloadgen.Loader[int64, []sqlcgen.TrackerLink]
	MetadataLinkByMedia    *dataloadgen.Loader[int64, *sqlcgen.MetadataLink]

	ProgressByChapter        *dataloadgen.Loader[int64, *sqlcgen.ReadingProgress]
	LatestDownloadByChapter  *dataloadgen.Loader[int64, *sqlcgen.Download]
	DownloadedPagesByChapter *dataloadgen.Loader[int64, int32]

	DownloadedByChapter *dataloadgen.Loader[int64, bool]
}

type loadersCtxKey struct{}

func NewLoaders(q *sqlcgen.Queries) *Loaders {
	return &Loaders{
		ChapterCountByMedia:      dataloadgen.NewLoader(chapterCountByMediaFn(q)),
		UnreadCountByMedia:       dataloadgen.NewLoader(unreadCountByMediaFn(q)),
		DownloadedCountByMedia:   dataloadgen.NewLoader(downloadedCountByMediaFn(q)),
		TagsByMedia:              dataloadgen.NewLoader(tagsByMediaFn(q)),
		GenresByMedia:            dataloadgen.NewLoader(genresByMediaFn(q)),
		ProgressByMedia:          dataloadgen.NewLoader(progressByMediaFn(q)),
		FoldersByMedia:           dataloadgen.NewLoader(foldersByMediaFn(q)),
		NextUnreadByMedia:        dataloadgen.NewLoader(nextUnreadByMediaFn(q)),
		LatestChapterByMedia:     dataloadgen.NewLoader(latestChapterByMediaFn(q)),
		ExtensionByID:            dataloadgen.NewLoader(extensionByIDBatchFn(q)),
		MediaByID:                dataloadgen.NewLoader(mediaByIDFn(q)),
		TrackLinksByMedia:        dataloadgen.NewLoader(trackLinksByMediaFn(q)),
		MetadataLinkByMedia:      dataloadgen.NewLoader(metadataLinkByMediaFn(q)),
		ProgressByChapter:        dataloadgen.NewLoader(progressByChapterFn(q)),
		LatestDownloadByChapter:  dataloadgen.NewLoader(latestDownloadByChapterFn(q)),
		DownloadedPagesByChapter: dataloadgen.NewLoader(downloadedPagesByChapterFn(q)),
		DownloadedByChapter:      dataloadgen.NewLoader(downloadedByChapterFn(q)),
	}
}

func WithLoaders(ctx context.Context, q *sqlcgen.Queries) context.Context {
	return context.WithValue(ctx, loadersCtxKey{}, NewLoaders(q))
}

func LoadersFromContext(ctx context.Context) *Loaders {
	l, _ := ctx.Value(loadersCtxKey{}).(*Loaders)
	return l
}

func repeatErr(err error, n int) []error {
	errs := make([]error, n)
	for i := range errs {
		errs[i] = err
	}
	return errs
}

func groupByKey[T any](keys []int64, rows []T, keyOf func(T) int64) [][]T {
	buckets := make(map[int64][]T, len(keys))
	for _, r := range rows {
		k := keyOf(r)
		buckets[k] = append(buckets[k], r)
	}
	out := make([][]T, len(keys))
	for i, k := range keys {
		out[i] = buckets[k]
	}
	return out
}

func chapterCountByMediaFn(q *sqlcgen.Queries) func(context.Context, []int64) ([]int32, []error) {
	return func(ctx context.Context, keys []int64) ([]int32, []error) {
		rows, err := q.CountChaptersByMediaIDs(ctx, keys)
		if err != nil {
			return make([]int32, len(keys)), repeatErr(err, len(keys))
		}
		m := make(map[int64]int32, len(rows))
		for _, r := range rows {
			m[r.MediaID] = int32(r.ChapterCount)
		}
		out := make([]int32, len(keys))
		for i, k := range keys {
			out[i] = m[k]
		}
		return out, nil
	}
}

func unreadCountByMediaFn(q *sqlcgen.Queries) func(context.Context, []int64) ([]int32, []error) {
	return func(ctx context.Context, keys []int64) ([]int32, []error) {
		rows, err := q.CountUnreadChaptersByMediaIDs(ctx, keys)
		if err != nil {
			return make([]int32, len(keys)), repeatErr(err, len(keys))
		}
		m := make(map[int64]int32, len(rows))
		for _, r := range rows {
			m[r.MediaID] = int32(r.UnreadCount)
		}
		out := make([]int32, len(keys))
		for i, k := range keys {
			out[i] = m[k]
		}
		return out, nil
	}
}

func downloadedCountByMediaFn(q *sqlcgen.Queries) func(context.Context, []int64) ([]int32, []error) {
	return func(ctx context.Context, keys []int64) ([]int32, []error) {
		rows, err := q.CountDownloadedChaptersByMediaIDs(ctx, keys)
		if err != nil {
			return make([]int32, len(keys)), repeatErr(err, len(keys))
		}
		m := make(map[int64]int32, len(rows))
		for _, r := range rows {
			m[r.MediaID] = int32(r.DownloadCount)
		}
		out := make([]int32, len(keys))
		for i, k := range keys {
			out[i] = m[k]
		}
		return out, nil
	}
}

func tagsByMediaFn(q *sqlcgen.Queries) func(context.Context, []int64) ([][]string, []error) {
	return func(ctx context.Context, keys []int64) ([][]string, []error) {
		rows, err := q.ListTagsByMediaIDs(ctx, keys)
		if err != nil {
			return make([][]string, len(keys)), repeatErr(err, len(keys))
		}
		m := make(map[int64][]string, len(keys))
		for _, r := range rows {
			m[r.MediaID] = append(m[r.MediaID], r.Name)
		}
		out := make([][]string, len(keys))
		for i, k := range keys {
			out[i] = m[k]
		}
		return out, nil
	}
}

func genresByMediaFn(q *sqlcgen.Queries) func(context.Context, []int64) ([][]string, []error) {
	return func(ctx context.Context, keys []int64) ([][]string, []error) {
		rows, err := q.ListGenresByMediaIDs(ctx, keys)
		if err != nil {
			return make([][]string, len(keys)), repeatErr(err, len(keys))
		}
		m := make(map[int64][]string, len(keys))
		for _, r := range rows {
			m[r.MediaID] = append(m[r.MediaID], r.Name)
		}
		out := make([][]string, len(keys))
		for i, k := range keys {
			out[i] = m[k]
		}
		return out, nil
	}
}

func progressByMediaFn(q *sqlcgen.Queries) func(context.Context, []int64) ([][]sqlcgen.ReadingProgress, []error) {
	return func(ctx context.Context, keys []int64) ([][]sqlcgen.ReadingProgress, []error) {
		rows, err := q.ListReadingProgressByMediaIDs(ctx, keys)
		if err != nil {
			return make([][]sqlcgen.ReadingProgress, len(keys)), repeatErr(err, len(keys))
		}
		return groupByKey(keys, rows, func(p sqlcgen.ReadingProgress) int64 { return p.MediaID }), nil
	}
}

func foldersByMediaFn(q *sqlcgen.Queries) func(context.Context, []int64) ([][]sqlcgen.Folder, []error) {
	return func(ctx context.Context, keys []int64) ([][]sqlcgen.Folder, []error) {
		rows, err := q.ListFoldersByMediaIDs(ctx, keys)
		if err != nil {
			return make([][]sqlcgen.Folder, len(keys)), repeatErr(err, len(keys))
		}
		m := make(map[int64][]sqlcgen.Folder, len(keys))
		for _, r := range rows {
			m[r.MediaID] = append(m[r.MediaID], sqlcgen.Folder{
				ID:                r.ID,
				Name:              r.Name,
				Kind:              r.Kind,
				SystemKey:         r.SystemKey,
				ParentFolderID:    r.ParentFolderID,
				SortOrder:         r.SortOrder,
				CreatedAt:         r.CreatedAt,
				IncludeInUpdate:   r.IncludeInUpdate,
				IncludeInDownload: r.IncludeInDownload,
			})
		}
		out := make([][]sqlcgen.Folder, len(keys))
		for i, k := range keys {
			out[i] = m[k]
		}
		return out, nil
	}
}

func nextUnreadByMediaFn(q *sqlcgen.Queries) func(context.Context, []int64) ([]*sqlcgen.Chapter, []error) {
	return func(ctx context.Context, keys []int64) ([]*sqlcgen.Chapter, []error) {
		rows, err := q.NextUnreadChapterByMediaIDs(ctx, keys)
		if err != nil {
			return make([]*sqlcgen.Chapter, len(keys)), repeatErr(err, len(keys))
		}
		m := make(map[int64]sqlcgen.Chapter, len(rows))
		for _, r := range rows {
			m[r.MediaID] = r
		}
		out := make([]*sqlcgen.Chapter, len(keys))
		for i, k := range keys {
			if c, ok := m[k]; ok {
				cc := c
				out[i] = &cc
			}
		}
		return out, nil
	}
}

func latestChapterByMediaFn(q *sqlcgen.Queries) func(context.Context, []int64) ([]*sqlcgen.Chapter, []error) {
	return func(ctx context.Context, keys []int64) ([]*sqlcgen.Chapter, []error) {
		rows, err := q.LatestChapterByMediaIDs(ctx, keys)
		if err != nil {
			return make([]*sqlcgen.Chapter, len(keys)), repeatErr(err, len(keys))
		}
		m := make(map[int64]sqlcgen.Chapter, len(rows))
		for _, r := range rows {
			m[r.MediaID] = r
		}
		out := make([]*sqlcgen.Chapter, len(keys))
		for i, k := range keys {
			if c, ok := m[k]; ok {
				cc := c
				out[i] = &cc
			}
		}
		return out, nil
	}
}

func extensionByIDBatchFn(q *sqlcgen.Queries) func(ctx context.Context, keys []int64) ([]*sqlcgen.Extension, []error) {
	return func(ctx context.Context, keys []int64) ([]*sqlcgen.Extension, []error) {
		rows, err := q.GetExtensionsByIDs(ctx, keys)
		if err != nil {
			return make([]*sqlcgen.Extension, len(keys)), repeatErr(err, len(keys))
		}
		byID := make(map[int64]sqlcgen.Extension, len(rows))
		for _, row := range rows {
			byID[row.ID] = row
		}
		out := make([]*sqlcgen.Extension, len(keys))
		for i, k := range keys {
			if e, ok := byID[k]; ok {
				ext := e
				out[i] = &ext
			}
		}
		return out, nil
	}
}

func mediaByIDFn(q *sqlcgen.Queries) func(context.Context, []int64) ([]*sqlcgen.Medium, []error) {
	return func(ctx context.Context, keys []int64) ([]*sqlcgen.Medium, []error) {
		rows, err := q.ListMediaByIDs(ctx, keys)
		if err != nil {
			return make([]*sqlcgen.Medium, len(keys)), repeatErr(err, len(keys))
		}
		byID := make(map[int64]sqlcgen.Medium, len(rows))
		for _, r := range rows {
			byID[r.ID] = r
		}
		out := make([]*sqlcgen.Medium, len(keys))
		for i, k := range keys {
			if m, ok := byID[k]; ok {
				mm := m
				out[i] = &mm
			}
		}
		return out, nil
	}
}

func progressByChapterFn(q *sqlcgen.Queries) func(context.Context, []int64) ([]*sqlcgen.ReadingProgress, []error) {
	return func(ctx context.Context, keys []int64) ([]*sqlcgen.ReadingProgress, []error) {
		rows, err := q.ListReadingProgressByChapterIDs(ctx, keys)
		if err != nil {
			return make([]*sqlcgen.ReadingProgress, len(keys)), repeatErr(err, len(keys))
		}
		byChapter := make(map[int64]sqlcgen.ReadingProgress, len(rows))
		for _, r := range rows {
			byChapter[r.ChapterID] = r
		}
		out := make([]*sqlcgen.ReadingProgress, len(keys))
		for i, k := range keys {
			if p, ok := byChapter[k]; ok {
				pp := p
				out[i] = &pp
			}
		}
		return out, nil
	}
}

func latestDownloadByChapterFn(q *sqlcgen.Queries) func(context.Context, []int64) ([]*sqlcgen.Download, []error) {
	return func(ctx context.Context, keys []int64) ([]*sqlcgen.Download, []error) {
		rows, err := q.GetLatestDownloadsByChapterIDs(ctx, keys)
		if err != nil {
			return make([]*sqlcgen.Download, len(keys)), repeatErr(err, len(keys))
		}
		byChapter := make(map[int64]sqlcgen.Download, len(rows))
		for _, r := range rows {
			byChapter[r.ChapterID] = r
		}
		out := make([]*sqlcgen.Download, len(keys))
		for i, k := range keys {
			if d, ok := byChapter[k]; ok {
				dd := d
				out[i] = &dd
			}
		}
		return out, nil
	}
}

func downloadedPagesByChapterFn(q *sqlcgen.Queries) func(context.Context, []int64) ([]int32, []error) {
	return func(ctx context.Context, keys []int64) ([]int32, []error) {
		rows, err := q.CountDownloadedPagesByChapterIDs(ctx, keys)
		if err != nil {
			return make([]int32, len(keys)), repeatErr(err, len(keys))
		}
		m := make(map[int64]int32, len(rows))
		for _, r := range rows {
			m[r.ChapterID] = int32(r.PageCount)
		}
		out := make([]int32, len(keys))
		for i, k := range keys {
			out[i] = m[k]
		}
		return out, nil
	}
}

func downloadedByChapterFn(q *sqlcgen.Queries) func(context.Context, []int64) ([]bool, []error) {
	return func(ctx context.Context, keys []int64) ([]bool, []error) {
		has := make(map[int64]bool, len(keys))

		pages, err := q.CountDownloadedPagesByChapterIDs(ctx, keys)
		if err != nil {
			return make([]bool, len(keys)), repeatErr(err, len(keys))
		}
		for _, r := range pages {
			if r.PageCount > 0 {
				has[r.ChapterID] = true
			}
		}

		novels, err := q.ListDownloadedNovelContentByChapterIDs(ctx, keys)
		if err != nil {
			return make([]bool, len(keys)), repeatErr(err, len(keys))
		}
		for _, r := range novels {
			has[r.ChapterID] = true
		}

		streams, err := q.ListDownloadedEpisodeStreamsByChapterIDs(ctx, keys)
		if err != nil {
			return make([]bool, len(keys)), repeatErr(err, len(keys))
		}
		for _, r := range streams {
			has[r.ChapterID] = true
		}

		out := make([]bool, len(keys))
		for i, k := range keys {
			out[i] = has[k]
		}
		return out, nil
	}
}

func trackLinksByMediaFn(q *sqlcgen.Queries) func(context.Context, []int64) ([][]sqlcgen.TrackerLink, []error) {
	return func(ctx context.Context, keys []int64) ([][]sqlcgen.TrackerLink, []error) {
		rows, err := q.ListTrackerLinksByMediaIDs(ctx, keys)
		if err != nil {
			return make([][]sqlcgen.TrackerLink, len(keys)), repeatErr(err, len(keys))
		}
		m := make(map[int64][]sqlcgen.TrackerLink, len(keys))
		for _, r := range rows {
			m[r.MediaID] = append(m[r.MediaID], r)
		}
		out := make([][]sqlcgen.TrackerLink, len(keys))
		for i, k := range keys {
			out[i] = m[k]
		}
		return out, nil
	}
}

func metadataLinkByMediaFn(q *sqlcgen.Queries) func(context.Context, []int64) ([]*sqlcgen.MetadataLink, []error) {
	return func(ctx context.Context, keys []int64) ([]*sqlcgen.MetadataLink, []error) {
		rows, err := q.ListMetadataLinksByMediaIDs(ctx, keys)
		if err != nil {
			return make([]*sqlcgen.MetadataLink, len(keys)), repeatErr(err, len(keys))
		}
		m := make(map[int64]*sqlcgen.MetadataLink, len(keys))
		for i := range rows {
			if rows[i].Provider == "anilist" {
				r := rows[i]
				m[r.MediaID] = &r
			}
		}
		out := make([]*sqlcgen.MetadataLink, len(keys))
		for i, k := range keys {
			out[i] = m[k]
		}
		return out, nil
	}
}
