package graph

import (
	"database/sql"
	"strconv"
	"time"

	"tsunagu/backend/internal/api/graph/model"
	"tsunagu/backend/internal/db/sqlcgen"
)

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func nullFloat64Ptr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	return &v.Float64
}

func nullBoolPtr(v sql.NullBool) *bool {
	if !v.Valid {
		return nil
	}
	return &v.Bool
}

func nullTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	return &v.Time
}

func nullInt64Int32Ptr(v sql.NullInt64) *int32 {
	if !v.Valid {
		return nil
	}
	n := int32(v.Int64)
	return &n
}

func nullInt64Ptr(v sql.NullInt64) *string {
	if !v.Valid {
		return nil
	}
	s := strconv.FormatInt(v.Int64, 10)
	return &s
}

func epochToTimePtr(v sql.NullInt64) *time.Time {
	if !v.Valid || v.Int64 == 0 {
		return nil
	}
	t := time.Unix(v.Int64, 0).UTC()
	return &t
}

func contentType(s string) model.ContentType {
	switch s {
	case "novel":
		return model.ContentTypeNovel
	case "manga":
		return model.ContentTypeManga
	case "anime":
		return model.ContentTypeAnime
	}
	return model.ContentTypeManga
}

func contentTypeToString(c *model.ContentType) string {
	if c == nil {
		return ""
	}
	switch *c {
	case model.ContentTypeNovel:
		return "novel"
	case model.ContentTypeManga:
		return "manga"
	case model.ContentTypeAnime:
		return "anime"
	}
	return ""
}

func toRepository(r sqlcgen.Repository) *model.Repository {
	return &model.Repository{
		ID:           strconv.FormatInt(r.ID, 10),
		IndexURL:     r.IndexUrl,
		Name:         nullStringPtr(r.Name),
		ContentType:  contentType(r.ContentType),
		AddedAt:      r.AddedAt,
		LastSyncedAt: nullTimePtr(r.LastSyncedAt),
	}
}

func toExtension(e sqlcgen.Extension) *model.Extension {
	return &model.Extension{
		ID:               strconv.FormatInt(e.ID, 10),
		RepositoryID:     strconv.FormatInt(e.RepositoryID, 10),
		PackageName:      e.PackageName,
		Name:             e.Name,
		Version:          e.Version,
		ContentType:      contentType(e.ContentType),
		Lang:             e.Lang,
		IconURL:          nullStringPtr(e.IconUrl),
		ApkURL:           &e.ApkUrl,
		JarURL:           nullStringPtr(e.JarUrl),
		JarPath:          nullStringPtr(e.JarPath),
		Installed:        e.Installed,
		Enabled:          e.Enabled,
		DiscoveredAt:     e.DiscoveredAt,
		InstalledAt:      nullTimePtr(e.InstalledAt),
		InstalledVersion: nullStringPtr(e.InstalledVersion),
		NeedsUpdate:      nullBoolPtr(e.NeedsUpdate),
	}
}

func toLibraryEntry(l sqlcgen.LibraryEntry) *model.LibraryEntry {
	return &model.LibraryEntry{
		ID:                 strconv.FormatInt(l.ID, 10),
		ExtensionID:        nullInt64Ptr(l.ExtensionID),
		ExtensionName:      l.ExtensionName,
		ExternalID:         l.ExternalID,
		ContentType:        contentType(l.ContentType),
		Title:              l.Title,
		CoverPath:          nullStringPtr(l.CoverPath),
		Description:        nullStringPtr(l.Description),
		Status:             nullStringPtr(l.Status),
		ExtensionRemovedAt: nullTimePtr(l.ExtensionRemovedAt),
		AddedAt:            l.AddedAt,
	}
}

func toChapter(c sqlcgen.Chapter) *model.Chapter {
	return &model.Chapter{
		ID:             strconv.FormatInt(c.ID, 10),
		LibraryEntryID: strconv.FormatInt(c.LibraryEntryID, 10),
		ExternalID:     c.ExternalID,
		Title:          nullStringPtr(c.Title),
		Number:         nullFloat64Ptr(c.Number),
		UploadedAt:     epochToTimePtr(c.UploadedAt),
	}
}

func toReadingProgress(p sqlcgen.ReadingProgress) *model.ReadingProgress {
	return &model.ReadingProgress{
		ID:              strconv.FormatInt(p.ID, 10),
		LibraryEntryID:  strconv.FormatInt(p.LibraryEntryID, 10),
		ChapterID:       strconv.FormatInt(p.ChapterID, 10),
		Progress:        p.Progress,
		Completed:       p.Completed,
		PositionSeconds: nullFloat64Ptr(p.PositionSeconds),
		DurationSeconds: nullFloat64Ptr(p.DurationSeconds),
		UpdatedAt:       p.UpdatedAt,
	}
}

func downloadStatus(s string) model.DownloadStatus {
	switch s {
	case "queued":
		return model.DownloadStatusQueued
	case "downloading":
		return model.DownloadStatusDownloading
	case "done":
		return model.DownloadStatusDone
	case "failed":
		return model.DownloadStatusFailed
	}
	return model.DownloadStatusQueued
}

func toDownload(d sqlcgen.Download) *model.Download {
	var finalSize *int32
	if d.Status == "done" {
		finalSize = nullInt64Int32Ptr(d.DownloadedBytes)
	}
	return &model.Download{
		ID:              strconv.FormatInt(d.ID, 10),
		ChapterID:       strconv.FormatInt(d.ChapterID, 10),
		Status:          downloadStatus(d.Status),
		Progress:        d.Progress,
		DownloadedBytes: nullInt64Int32Ptr(d.DownloadedBytes),
		BytesPerSec:     nullFloat64Ptr(d.BytesPerSec),
		FinalSizeBytes:  finalSize,
		Error:           nullStringPtr(d.Error),
		CreatedAt:       d.CreatedAt,
		CompletedAt:     nullTimePtr(d.CompletedAt),
	}
}

func parseID(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}
