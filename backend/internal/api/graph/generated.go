package graph

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync/atomic"
	"time"
	"tsunagu/backend/internal/api/graph/model"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/introspection"
	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func NewExecutableSchema(cfg Config) graphql.ExecutableSchema {
	return &executableSchema{SchemaData: cfg.Schema, Resolvers: cfg.Resolvers, Directives: cfg.Directives, ComplexityRoot: cfg.Complexity}
}

type Config = graphql.Config[ResolverRoot, DirectiveRoot, ComplexityRoot]

type ResolverRoot interface {
	Chapter() ChapterResolver
	Download() DownloadResolver
	Media() MediaResolver
	Mutation() MutationResolver
	Query() QueryResolver
}

type DirectiveRoot struct {
}

type ComplexityRoot struct {
	AboutServer struct {
		BuildTime func(childComplexity int) int
		Name      func(childComplexity int) int
		Version   func(childComplexity int) int
	}

	AudioTrack struct {
		Lang func(childComplexity int) int
		URL  func(childComplexity int) int
	}

	Chapter struct {
		Completed       func(childComplexity int) int
		Download        func(childComplexity int) int
		Downloaded      func(childComplexity int) int
		ExternalID      func(childComplexity int) int
		ID              func(childComplexity int) int
		MediaID         func(childComplexity int) int
		Number          func(childComplexity int) int
		PageCount       func(childComplexity int) int
		Pages           func(childComplexity int) int
		ReadingProgress func(childComplexity int) int
		Scanlator       func(childComplexity int) int
		SourceOrder     func(childComplexity int) int
		Title           func(childComplexity int) int
		UploadedAt      func(childComplexity int) int
		VideoStream     func(childComplexity int) int
		VideoURL        func(childComplexity int) int
	}

	CheckBoxFilter struct {
		Name  func(childComplexity int) int
		State func(childComplexity int) int
	}

	Download struct {
		BytesPerSec     func(childComplexity int) int
		Chapter         func(childComplexity int) int
		ChapterID       func(childComplexity int) int
		CompletedAt     func(childComplexity int) int
		CreatedAt       func(childComplexity int) int
		DownloadedBytes func(childComplexity int) int
		Error           func(childComplexity int) int
		FinalSizeBytes  func(childComplexity int) int
		ID              func(childComplexity int) int
		MediaID         func(childComplexity int) int
		Progress        func(childComplexity int) int
		Status          func(childComplexity int) int
	}

	DownloaderStatus struct {
		DownloadingCount func(childComplexity int) int
		FailedCount      func(childComplexity int) int
		IsRunning        func(childComplexity int) int
		QueuedCount      func(childComplexity int) int
	}

	Extension struct {
		ApkURL           func(childComplexity int) int
		ContentType      func(childComplexity int) int
		DiscoveredAt     func(childComplexity int) int
		DisplayName      func(childComplexity int) int
		Enabled          func(childComplexity int) int
		ID               func(childComplexity int) int
		IconURL          func(childComplexity int) int
		Installed        func(childComplexity int) int
		InstalledAt      func(childComplexity int) int
		InstalledVersion func(childComplexity int) int
		IsNsfw           func(childComplexity int) int
		JarPath          func(childComplexity int) int
		JarURL           func(childComplexity int) int
		Lang             func(childComplexity int) int
		Name             func(childComplexity int) int
		NeedsUpdate      func(childComplexity int) int
		PackageName      func(childComplexity int) int
		RepositoryID     func(childComplexity int) int
		SupportsLatest   func(childComplexity int) int
		Version          func(childComplexity int) int
	}

	Folder struct {
		ID                func(childComplexity int) int
		IncludeInDownload func(childComplexity int) int
		IncludeInUpdate   func(childComplexity int) int
		Kind              func(childComplexity int) int
		Name              func(childComplexity int) int
		ParentFolderID    func(childComplexity int) int
		SortOrder         func(childComplexity int) int
		SystemKey         func(childComplexity int) int
	}

	GroupFilter struct {
		Children func(childComplexity int) int
		Name     func(childComplexity int) int
	}

	HeaderFilter struct {
		Name func(childComplexity int) int
	}

	LibraryUpdateStatus struct {
		CurrentTitle    func(childComplexity int) int
		Done            func(childComplexity int) int
		FailedTitles    func(childComplexity int) int
		FinishedAt      func(childComplexity int) int
		NewChapterCount func(childComplexity int) int
		Running         func(childComplexity int) int
		StartedAt       func(childComplexity int) int
		Total           func(childComplexity int) int
	}

	Media struct {
		AddedAt            func(childComplexity int) int
		Artist             func(childComplexity int) int
		Author             func(childComplexity int) int
		ChapterCount       func(childComplexity int) int
		Chapters           func(childComplexity int) int
		ContentType        func(childComplexity int) int
		Description        func(childComplexity int) int
		DetailsFetchedAt   func(childComplexity int) int
		DownloadedCount    func(childComplexity int) int
		ExtensionID        func(childComplexity int) int
		ExtensionName      func(childComplexity int) int
		ExtensionRemovedAt func(childComplexity int) int
		ExternalID         func(childComplexity int) int
		Folders            func(childComplexity int) int
		Genres             func(childComplexity int) int
		ID                 func(childComplexity int) int
		InLibrary          func(childComplexity int) int
		LastViewedAt       func(childComplexity int) int
		LatestChapter      func(childComplexity int) int
		Metadata           func(childComplexity int) int
		NextUnreadChapter  func(childComplexity int) int
		ReadingProgress    func(childComplexity int) int
		Source             func(childComplexity int) int
		SourceName         func(childComplexity int) int
		Status             func(childComplexity int) int
		Tags               func(childComplexity int) int
		ThumbnailURL       func(childComplexity int) int
		Title              func(childComplexity int) int
		TrackLinks         func(childComplexity int) int
		UnreadCount        func(childComplexity int) int
	}

	MediaPage struct {
		HasMore func(childComplexity int) int
		Items   func(childComplexity int) int
		Total   func(childComplexity int) int
	}

	MetadataCandidate struct {
		CoverURL    func(childComplexity int) int
		Description func(childComplexity int) int
		Genres      func(childComplexity int) int
		Provider    func(childComplexity int) int
		ProviderID  func(childComplexity int) int
		StartYear   func(childComplexity int) int
		Status      func(childComplexity int) int
		Title       func(childComplexity int) int
		URL         func(childComplexity int) int
	}

	MetadataMatch struct {
		Confidence func(childComplexity int) int
		Locked     func(childComplexity int) int
		MatchedAt  func(childComplexity int) int
		Provider   func(childComplexity int) int
		ProviderID func(childComplexity int) int
		URL        func(childComplexity int) int
	}

	Mutation struct {
		AddMediaToFolder         func(childComplexity int, mediaID string, folderID string) int
		AddRepository            func(childComplexity int, indexURL string, name *string) int
		ApplyMetadataMatch       func(childComplexity int, mediaID string, providerID string, provider *string) int
		BindTrack                func(childComplexity int, mediaID string, trackerKey string, remoteID string) int
		ClearDownloads           func(childComplexity int, status []model.DownloadStatus) int
		ClearImageCache          func(childComplexity int) int
		CreateFolder             func(childComplexity int, name string, parentFolderID *string) int
		DeleteDownload           func(childComplexity int, mediaID string, chapterIds []string) int
		DeleteFolder             func(childComplexity int, folderID string) int
		DeleteRepository         func(childComplexity int, repositoryID string) int
		DequeueDownload          func(childComplexity int, mediaID string, chapterID string) int
		EnqueueDownload          func(childComplexity int, mediaID string, chapterIds []string) int
		InstallExtension         func(childComplexity int, packageName string) int
		InstallExternalExtension func(childComplexity int, url string) int
		MarkChapterRead          func(childComplexity int, mediaID string, chapterID string) int
		MarkChaptersRead         func(childComplexity int, mediaID string, chapterIds []string, read bool) int
		MigrateMedia             func(childComplexity int, fromMediaID string, toExtensionID string, toExternalID string) int
		PullTracker              func(childComplexity int, mediaID string) int
		RefreshFolder            func(childComplexity int, folderID string) int
		RefreshMetadata          func(childComplexity int, mediaID string, syncChapters *bool) int
		RefreshMetadataMatch     func(childComplexity int, mediaID string) int
		RemoveMediaFromFolder    func(childComplexity int, mediaID string, folderID string) int
		RenameFolder             func(childComplexity int, folderID string, name string) int
		RenameRepository         func(childComplexity int, repositoryID string, name string) int
		ReorderDownload          func(childComplexity int, mediaID string, chapterID string, position int32) int
		ReorderFolder            func(childComplexity int, folderID string, sortOrder int32) int
		RescanLocalMedia         func(childComplexity int) int
		ResyncTrack              func(childComplexity int, linkID string) int
		RetryDownload            func(childComplexity int, mediaID string, chapterID string) int
		SetInLibrary             func(childComplexity int, mediaID string, inLibrary bool) int
		SetMediaCover            func(childComplexity int, mediaID string, url *string) int
		StartDownloader          func(childComplexity int) int
		StartLibraryUpdate       func(childComplexity int, folderID *string) int
		StopDownloader           func(childComplexity int) int
		SyncChapters             func(childComplexity int, mediaID string) int
		TrackerLogin             func(childComplexity int, trackerKey string, token string) int
		TrackerLogout            func(childComplexity int, trackerKey string) int
		UnbindTrack              func(childComplexity int, linkID string) int
		UninstallExtension       func(childComplexity int, packageName string) int
		UnlinkMetadata           func(childComplexity int, mediaID string) int
		UpdateExtension          func(childComplexity int, packageName string) int
		UpdateFolderFlags        func(childComplexity int, folderID string, includeInUpdate *bool, includeInDownload *bool) int
		UpdateReadingProgress    func(childComplexity int, mediaID string, chapterID string, progress float64, completed *bool, positionSeconds *float64, durationSeconds *float64) int
		UpdateTrack              func(childComplexity int, linkID string, status *int32, score *float64, lastChapterRead *float64) int
	}

	Query struct {
		About               func(childComplexity int) int
		AvailableExtensions func(childComplexity int, repositoryID string) int
		Chapter             func(childComplexity int, id string) int
		ChapterUpdates      func(childComplexity int, since *time.Time, limit *int32) int
		DownloadQueue       func(childComplexity int) int
		DownloadStatus      func(childComplexity int, mediaID string, chapterID string) int
		DownloaderStatus    func(childComplexity int) int
		FilterOptions       func(childComplexity int, extensionID string) int
		Folder              func(childComplexity int, id string) int
		Folders             func(childComplexity int) int
		InstalledExtensions func(childComplexity int) int
		LatestUpdates       func(childComplexity int, extensionID string, page *int32) int
		Library             func(childComplexity int, filter *model.LibraryFilter, sort *model.LibrarySortInput, limit *int32, offset *int32) int
		LibraryUpdateStatus func(childComplexity int) int
		Media               func(childComplexity int, id string) int
		MediaInFolder       func(childComplexity int, folderID string) int
		PopularManga        func(childComplexity int, extensionID string, page *int32) int
		ReadingProgress     func(childComplexity int, mediaID string) int
		RecentChapters      func(childComplexity int, since *time.Time, limit *int32) int
		Repositories        func(childComplexity int) int
		ResolveMedia        func(childComplexity int, extensionID string, externalID string, syncChapters *bool) int
		Search              func(childComplexity int, extensionID string, query string, page *int32, filters []*model.FilterInput) int
		SearchMetadata      func(childComplexity int, query string, contentType model.ContentType, provider *string) int
		SkipTimestamps      func(childComplexity int, chapterID string, episodeLengthMs *int32) int
		StorageInfo         func(childComplexity int) int
		TrackSearch         func(childComplexity int, trackerKey string, query string, contentType *model.ContentType) int
		Trackers            func(childComplexity int) int
	}

	ReadingProgress struct {
		ChapterID       func(childComplexity int) int
		Completed       func(childComplexity int) int
		DurationSeconds func(childComplexity int) int
		ID              func(childComplexity int) int
		MediaID         func(childComplexity int) int
		PositionSeconds func(childComplexity int) int
		Progress        func(childComplexity int) int
		UpdatedAt       func(childComplexity int) int
	}

	RecentChapter struct {
		Chapter func(childComplexity int) int
		Media   func(childComplexity int) int
	}

	Repository struct {
		AddedAt      func(childComplexity int) int
		ContentType  func(childComplexity int) int
		ID           func(childComplexity int) int
		IndexURL     func(childComplexity int) int
		LastSyncedAt func(childComplexity int) int
		Name         func(childComplexity int) int
	}

	SearchResponse struct {
		HasNextPage func(childComplexity int) int
		Results     func(childComplexity int) int
	}

	SelectFilter struct {
		Name   func(childComplexity int) int
		State  func(childComplexity int) int
		Values func(childComplexity int) int
	}

	SeparatorFilter struct {
		Name func(childComplexity int) int
	}

	SkipMarker struct {
		EndMs   func(childComplexity int) int
		Name    func(childComplexity int) int
		StartMs func(childComplexity int) int
		Type    func(childComplexity int) int
	}

	SortFilter struct {
		Ascending func(childComplexity int) int
		HasState  func(childComplexity int) int
		Index     func(childComplexity int) int
		Name      func(childComplexity int) int
		Values    func(childComplexity int) int
	}

	StorageInfo struct {
		FreeBytes  func(childComplexity int) int
		TotalBytes func(childComplexity int) int
		UsedBytes  func(childComplexity int) int
	}

	SubtitleTrack struct {
		Lang func(childComplexity int) int
		URL  func(childComplexity int) int
	}

	TextFilter struct {
		Name  func(childComplexity int) int
		State func(childComplexity int) int
	}

	TrackLink struct {
		FinishedAt      func(childComplexity int) int
		ID              func(childComplexity int) int
		LastChapterRead func(childComplexity int) int
		LastSyncedAt    func(childComplexity int) int
		MediaID         func(childComplexity int) int
		Private         func(childComplexity int) int
		RemoteID        func(childComplexity int) int
		Score           func(childComplexity int) int
		StartedAt       func(childComplexity int) int
		Status          func(childComplexity int) int
		StatusName      func(childComplexity int) int
		Title           func(childComplexity int) int
		TotalChapters   func(childComplexity int) int
		TrackerKey      func(childComplexity int) int
		URL             func(childComplexity int) int
	}

	TrackSearchResult struct {
		CoverURL         func(childComplexity int) int
		MediaType        func(childComplexity int) int
		PublishingStatus func(childComplexity int) int
		RemoteID         func(childComplexity int) int
		Summary          func(childComplexity int) int
		Title            func(childComplexity int) int
		TotalChapters    func(childComplexity int) int
		URL              func(childComplexity int) int
	}

	TrackStatus struct {
		AnimeName func(childComplexity int) int
		Name      func(childComplexity int) int
		Value     func(childComplexity int) int
	}

	Tracker struct {
		AuthURL       func(childComplexity int) int
		Configured    func(childComplexity int) int
		IconURL       func(childComplexity int) int
		IsLoggedIn    func(childComplexity int) int
		Key           func(childComplexity int) int
		Name          func(childComplexity int) int
		ScoreOptions  func(childComplexity int) int
		StatusOptions func(childComplexity int) int
		Username      func(childComplexity int) int
	}

	TriStateFilter struct {
		Name  func(childComplexity int) int
		State func(childComplexity int) int
	}

	VideoSource struct {
		Kind       func(childComplexity int) int
		Label      func(childComplexity int) int
		Preferred  func(childComplexity int) int
		Resolution func(childComplexity int) int
		Server     func(childComplexity int) int
		URL        func(childComplexity int) int
	}

	VideoStream struct {
		AudioTracks func(childComplexity int) int
		SkipMarkers func(childComplexity int) int
		Sources     func(childComplexity int) int
		Subtitles   func(childComplexity int) int
		URL         func(childComplexity int) int
	}
}

type ChapterResolver interface {
	ReadingProgress(ctx context.Context, obj *model.Chapter) (*model.ReadingProgress, error)
	Completed(ctx context.Context, obj *model.Chapter) (bool, error)
	Downloaded(ctx context.Context, obj *model.Chapter) (bool, error)
	Download(ctx context.Context, obj *model.Chapter) (*model.Download, error)
	Pages(ctx context.Context, obj *model.Chapter) ([]string, error)
	PageCount(ctx context.Context, obj *model.Chapter) (*int32, error)
	VideoURL(ctx context.Context, obj *model.Chapter) (*string, error)
	VideoStream(ctx context.Context, obj *model.Chapter) (*model.VideoStream, error)
}
type DownloadResolver interface {
	Chapter(ctx context.Context, obj *model.Download) (*model.Chapter, error)
}
type MediaResolver interface {
	Chapters(ctx context.Context, obj *model.Media) ([]*model.Chapter, error)
	ChapterCount(ctx context.Context, obj *model.Media) (int32, error)
	UnreadCount(ctx context.Context, obj *model.Media) (int32, error)
	DownloadedCount(ctx context.Context, obj *model.Media) (int32, error)
	NextUnreadChapter(ctx context.Context, obj *model.Media) (*model.Chapter, error)
	LatestChapter(ctx context.Context, obj *model.Media) (*model.Chapter, error)
	ReadingProgress(ctx context.Context, obj *model.Media) ([]*model.ReadingProgress, error)
	Tags(ctx context.Context, obj *model.Media) ([]string, error)
	Genres(ctx context.Context, obj *model.Media) ([]string, error)
	Folders(ctx context.Context, obj *model.Media) ([]*model.Folder, error)
	TrackLinks(ctx context.Context, obj *model.Media) ([]*model.TrackLink, error)
	Metadata(ctx context.Context, obj *model.Media) (*model.MetadataMatch, error)
	Source(ctx context.Context, obj *model.Media) (*model.Extension, error)
}
type MutationResolver interface {
	CreateFolder(ctx context.Context, name string, parentFolderID *string) (*model.Folder, error)
	RenameFolder(ctx context.Context, folderID string, name string) (*model.Folder, error)
	DeleteFolder(ctx context.Context, folderID string) (bool, error)
	AddMediaToFolder(ctx context.Context, mediaID string, folderID string) (bool, error)
	RemoveMediaFromFolder(ctx context.Context, mediaID string, folderID string) (bool, error)
	MarkChaptersRead(ctx context.Context, mediaID string, chapterIds []string, read bool) ([]*model.ReadingProgress, error)
	DequeueDownload(ctx context.Context, mediaID string, chapterID string) (bool, error)
	AddRepository(ctx context.Context, indexURL string, name *string) (*model.Repository, error)
	RenameRepository(ctx context.Context, repositoryID string, name string) (*model.Repository, error)
	DeleteRepository(ctx context.Context, repositoryID string) (bool, error)
	InstallExtension(ctx context.Context, packageName string) (*model.Extension, error)
	InstallExternalExtension(ctx context.Context, url string) (*model.Extension, error)
	UninstallExtension(ctx context.Context, packageName string) (*model.Extension, error)
	UpdateExtension(ctx context.Context, packageName string) (*model.Extension, error)
	SetInLibrary(ctx context.Context, mediaID string, inLibrary bool) (*model.Media, error)
	MigrateMedia(ctx context.Context, fromMediaID string, toExtensionID string, toExternalID string) (*model.Media, error)
	SyncChapters(ctx context.Context, mediaID string) ([]*model.Chapter, error)
	UpdateReadingProgress(ctx context.Context, mediaID string, chapterID string, progress float64, completed *bool, positionSeconds *float64, durationSeconds *float64) (*model.ReadingProgress, error)
	MarkChapterRead(ctx context.Context, mediaID string, chapterID string) (*model.ReadingProgress, error)
	EnqueueDownload(ctx context.Context, mediaID string, chapterIds []string) ([]*model.Download, error)
	RetryDownload(ctx context.Context, mediaID string, chapterID string) (*model.Download, error)
	DeleteDownload(ctx context.Context, mediaID string, chapterIds []string) (bool, error)
	ReorderDownload(ctx context.Context, mediaID string, chapterID string, position int32) (bool, error)
	ClearDownloads(ctx context.Context, status []model.DownloadStatus) (bool, error)
	StartDownloader(ctx context.Context) (bool, error)
	StopDownloader(ctx context.Context) (bool, error)
	RefreshMetadata(ctx context.Context, mediaID string, syncChapters *bool) (*model.Media, error)
	RefreshFolder(ctx context.Context, folderID string) ([]*model.Media, error)
	ReorderFolder(ctx context.Context, folderID string, sortOrder int32) (*model.Folder, error)
	UpdateFolderFlags(ctx context.Context, folderID string, includeInUpdate *bool, includeInDownload *bool) (*model.Folder, error)
	ClearImageCache(ctx context.Context) (bool, error)
	StartLibraryUpdate(ctx context.Context, folderID *string) (bool, error)
	SetMediaCover(ctx context.Context, mediaID string, url *string) (*model.Media, error)
	RescanLocalMedia(ctx context.Context) ([]*model.Media, error)
	TrackerLogin(ctx context.Context, trackerKey string, token string) (*model.Tracker, error)
	TrackerLogout(ctx context.Context, trackerKey string) (bool, error)
	BindTrack(ctx context.Context, mediaID string, trackerKey string, remoteID string) (*model.TrackLink, error)
	UpdateTrack(ctx context.Context, linkID string, status *int32, score *float64, lastChapterRead *float64) (*model.TrackLink, error)
	UnbindTrack(ctx context.Context, linkID string) (bool, error)
	ResyncTrack(ctx context.Context, linkID string) (*model.TrackLink, error)
	PullTracker(ctx context.Context, mediaID string) ([]*model.TrackLink, error)
	ApplyMetadataMatch(ctx context.Context, mediaID string, providerID string, provider *string) (*model.Media, error)
	UnlinkMetadata(ctx context.Context, mediaID string) (bool, error)
	RefreshMetadataMatch(ctx context.Context, mediaID string) (*model.Media, error)
}
type QueryResolver interface {
	About(ctx context.Context) (*model.AboutServer, error)
	Folders(ctx context.Context) ([]*model.Folder, error)
	Folder(ctx context.Context, id string) (*model.Folder, error)
	MediaInFolder(ctx context.Context, folderID string) ([]*model.Media, error)
	Repositories(ctx context.Context) ([]*model.Repository, error)
	AvailableExtensions(ctx context.Context, repositoryID string) ([]*model.Extension, error)
	InstalledExtensions(ctx context.Context) ([]*model.Extension, error)
	Library(ctx context.Context, filter *model.LibraryFilter, sort *model.LibrarySortInput, limit *int32, offset *int32) (*model.MediaPage, error)
	Media(ctx context.Context, id string) (*model.Media, error)
	Chapter(ctx context.Context, id string) (*model.Chapter, error)
	ResolveMedia(ctx context.Context, extensionID string, externalID string, syncChapters *bool) (*model.Media, error)
	ReadingProgress(ctx context.Context, mediaID string) ([]*model.ReadingProgress, error)
	Search(ctx context.Context, extensionID string, query string, page *int32, filters []*model.FilterInput) (*model.SearchResponse, error)
	FilterOptions(ctx context.Context, extensionID string) ([]model.FilterNode, error)
	PopularManga(ctx context.Context, extensionID string, page *int32) (*model.SearchResponse, error)
	LatestUpdates(ctx context.Context, extensionID string, page *int32) (*model.SearchResponse, error)
	DownloadStatus(ctx context.Context, mediaID string, chapterID string) (*model.Download, error)
	DownloadQueue(ctx context.Context) ([]*model.Download, error)
	DownloaderStatus(ctx context.Context) (*model.DownloaderStatus, error)
	RecentChapters(ctx context.Context, since *time.Time, limit *int32) ([]*model.RecentChapter, error)
	ChapterUpdates(ctx context.Context, since *time.Time, limit *int32) ([]*model.RecentChapter, error)
	LibraryUpdateStatus(ctx context.Context) (*model.LibraryUpdateStatus, error)
	StorageInfo(ctx context.Context) (*model.StorageInfo, error)
	Trackers(ctx context.Context) ([]*model.Tracker, error)
	TrackSearch(ctx context.Context, trackerKey string, query string, contentType *model.ContentType) ([]*model.TrackSearchResult, error)
	SearchMetadata(ctx context.Context, query string, contentType model.ContentType, provider *string) ([]*model.MetadataCandidate, error)
	SkipTimestamps(ctx context.Context, chapterID string, episodeLengthMs *int32) ([]*model.SkipMarker, error)
}

type executableSchema graphql.ExecutableSchemaState[ResolverRoot, DirectiveRoot, ComplexityRoot]

func (e *executableSchema) Schema() *ast.Schema {
	if e.SchemaData != nil {
		return e.SchemaData
	}
	return parsedSchema
}

func (e *executableSchema) Complexity(ctx context.Context, typeName, field string, childComplexity int, rawArgs map[string]any) (int, bool) {
	ec := newExecutionContext(nil, e, nil)
	_ = ec
	switch typeName + "." + field {

	case "AboutServer.buildTime":
		if e.ComplexityRoot.AboutServer.BuildTime == nil {
			break
		}

		return e.ComplexityRoot.AboutServer.BuildTime(childComplexity), true
	case "AboutServer.name":
		if e.ComplexityRoot.AboutServer.Name == nil {
			break
		}

		return e.ComplexityRoot.AboutServer.Name(childComplexity), true
	case "AboutServer.version":
		if e.ComplexityRoot.AboutServer.Version == nil {
			break
		}

		return e.ComplexityRoot.AboutServer.Version(childComplexity), true

	case "AudioTrack.lang":
		if e.ComplexityRoot.AudioTrack.Lang == nil {
			break
		}

		return e.ComplexityRoot.AudioTrack.Lang(childComplexity), true
	case "AudioTrack.url":
		if e.ComplexityRoot.AudioTrack.URL == nil {
			break
		}

		return e.ComplexityRoot.AudioTrack.URL(childComplexity), true

	case "Chapter.completed":
		if e.ComplexityRoot.Chapter.Completed == nil {
			break
		}

		return e.ComplexityRoot.Chapter.Completed(childComplexity), true
	case "Chapter.download":
		if e.ComplexityRoot.Chapter.Download == nil {
			break
		}

		return e.ComplexityRoot.Chapter.Download(childComplexity), true
	case "Chapter.downloaded":
		if e.ComplexityRoot.Chapter.Downloaded == nil {
			break
		}

		return e.ComplexityRoot.Chapter.Downloaded(childComplexity), true
	case "Chapter.externalId":
		if e.ComplexityRoot.Chapter.ExternalID == nil {
			break
		}

		return e.ComplexityRoot.Chapter.ExternalID(childComplexity), true
	case "Chapter.id":
		if e.ComplexityRoot.Chapter.ID == nil {
			break
		}

		return e.ComplexityRoot.Chapter.ID(childComplexity), true
	case "Chapter.mediaId":
		if e.ComplexityRoot.Chapter.MediaID == nil {
			break
		}

		return e.ComplexityRoot.Chapter.MediaID(childComplexity), true
	case "Chapter.number":
		if e.ComplexityRoot.Chapter.Number == nil {
			break
		}

		return e.ComplexityRoot.Chapter.Number(childComplexity), true
	case "Chapter.pageCount":
		if e.ComplexityRoot.Chapter.PageCount == nil {
			break
		}

		return e.ComplexityRoot.Chapter.PageCount(childComplexity), true
	case "Chapter.pages":
		if e.ComplexityRoot.Chapter.Pages == nil {
			break
		}

		return e.ComplexityRoot.Chapter.Pages(childComplexity), true
	case "Chapter.readingProgress":
		if e.ComplexityRoot.Chapter.ReadingProgress == nil {
			break
		}

		return e.ComplexityRoot.Chapter.ReadingProgress(childComplexity), true
	case "Chapter.scanlator":
		if e.ComplexityRoot.Chapter.Scanlator == nil {
			break
		}

		return e.ComplexityRoot.Chapter.Scanlator(childComplexity), true
	case "Chapter.sourceOrder":
		if e.ComplexityRoot.Chapter.SourceOrder == nil {
			break
		}

		return e.ComplexityRoot.Chapter.SourceOrder(childComplexity), true
	case "Chapter.title":
		if e.ComplexityRoot.Chapter.Title == nil {
			break
		}

		return e.ComplexityRoot.Chapter.Title(childComplexity), true
	case "Chapter.uploadedAt":
		if e.ComplexityRoot.Chapter.UploadedAt == nil {
			break
		}

		return e.ComplexityRoot.Chapter.UploadedAt(childComplexity), true
	case "Chapter.videoStream":
		if e.ComplexityRoot.Chapter.VideoStream == nil {
			break
		}

		return e.ComplexityRoot.Chapter.VideoStream(childComplexity), true
	case "Chapter.videoUrl":
		if e.ComplexityRoot.Chapter.VideoURL == nil {
			break
		}

		return e.ComplexityRoot.Chapter.VideoURL(childComplexity), true

	case "CheckBoxFilter.name":
		if e.ComplexityRoot.CheckBoxFilter.Name == nil {
			break
		}

		return e.ComplexityRoot.CheckBoxFilter.Name(childComplexity), true
	case "CheckBoxFilter.state":
		if e.ComplexityRoot.CheckBoxFilter.State == nil {
			break
		}

		return e.ComplexityRoot.CheckBoxFilter.State(childComplexity), true

	case "Download.bytesPerSec":
		if e.ComplexityRoot.Download.BytesPerSec == nil {
			break
		}

		return e.ComplexityRoot.Download.BytesPerSec(childComplexity), true
	case "Download.chapter":
		if e.ComplexityRoot.Download.Chapter == nil {
			break
		}

		return e.ComplexityRoot.Download.Chapter(childComplexity), true
	case "Download.chapterId":
		if e.ComplexityRoot.Download.ChapterID == nil {
			break
		}

		return e.ComplexityRoot.Download.ChapterID(childComplexity), true
	case "Download.completedAt":
		if e.ComplexityRoot.Download.CompletedAt == nil {
			break
		}

		return e.ComplexityRoot.Download.CompletedAt(childComplexity), true
	case "Download.createdAt":
		if e.ComplexityRoot.Download.CreatedAt == nil {
			break
		}

		return e.ComplexityRoot.Download.CreatedAt(childComplexity), true
	case "Download.downloadedBytes":
		if e.ComplexityRoot.Download.DownloadedBytes == nil {
			break
		}

		return e.ComplexityRoot.Download.DownloadedBytes(childComplexity), true
	case "Download.error":
		if e.ComplexityRoot.Download.Error == nil {
			break
		}

		return e.ComplexityRoot.Download.Error(childComplexity), true
	case "Download.finalSizeBytes":
		if e.ComplexityRoot.Download.FinalSizeBytes == nil {
			break
		}

		return e.ComplexityRoot.Download.FinalSizeBytes(childComplexity), true
	case "Download.id":
		if e.ComplexityRoot.Download.ID == nil {
			break
		}

		return e.ComplexityRoot.Download.ID(childComplexity), true
	case "Download.mediaId":
		if e.ComplexityRoot.Download.MediaID == nil {
			break
		}

		return e.ComplexityRoot.Download.MediaID(childComplexity), true
	case "Download.progress":
		if e.ComplexityRoot.Download.Progress == nil {
			break
		}

		return e.ComplexityRoot.Download.Progress(childComplexity), true
	case "Download.status":
		if e.ComplexityRoot.Download.Status == nil {
			break
		}

		return e.ComplexityRoot.Download.Status(childComplexity), true

	case "DownloaderStatus.downloadingCount":
		if e.ComplexityRoot.DownloaderStatus.DownloadingCount == nil {
			break
		}

		return e.ComplexityRoot.DownloaderStatus.DownloadingCount(childComplexity), true
	case "DownloaderStatus.failedCount":
		if e.ComplexityRoot.DownloaderStatus.FailedCount == nil {
			break
		}

		return e.ComplexityRoot.DownloaderStatus.FailedCount(childComplexity), true
	case "DownloaderStatus.isRunning":
		if e.ComplexityRoot.DownloaderStatus.IsRunning == nil {
			break
		}

		return e.ComplexityRoot.DownloaderStatus.IsRunning(childComplexity), true
	case "DownloaderStatus.queuedCount":
		if e.ComplexityRoot.DownloaderStatus.QueuedCount == nil {
			break
		}

		return e.ComplexityRoot.DownloaderStatus.QueuedCount(childComplexity), true

	case "Extension.apkUrl":
		if e.ComplexityRoot.Extension.ApkURL == nil {
			break
		}

		return e.ComplexityRoot.Extension.ApkURL(childComplexity), true
	case "Extension.contentType":
		if e.ComplexityRoot.Extension.ContentType == nil {
			break
		}

		return e.ComplexityRoot.Extension.ContentType(childComplexity), true
	case "Extension.discoveredAt":
		if e.ComplexityRoot.Extension.DiscoveredAt == nil {
			break
		}

		return e.ComplexityRoot.Extension.DiscoveredAt(childComplexity), true
	case "Extension.displayName":
		if e.ComplexityRoot.Extension.DisplayName == nil {
			break
		}

		return e.ComplexityRoot.Extension.DisplayName(childComplexity), true
	case "Extension.enabled":
		if e.ComplexityRoot.Extension.Enabled == nil {
			break
		}

		return e.ComplexityRoot.Extension.Enabled(childComplexity), true
	case "Extension.id":
		if e.ComplexityRoot.Extension.ID == nil {
			break
		}

		return e.ComplexityRoot.Extension.ID(childComplexity), true
	case "Extension.iconUrl":
		if e.ComplexityRoot.Extension.IconURL == nil {
			break
		}

		return e.ComplexityRoot.Extension.IconURL(childComplexity), true
	case "Extension.installed":
		if e.ComplexityRoot.Extension.Installed == nil {
			break
		}

		return e.ComplexityRoot.Extension.Installed(childComplexity), true
	case "Extension.installedAt":
		if e.ComplexityRoot.Extension.InstalledAt == nil {
			break
		}

		return e.ComplexityRoot.Extension.InstalledAt(childComplexity), true
	case "Extension.installedVersion":
		if e.ComplexityRoot.Extension.InstalledVersion == nil {
			break
		}

		return e.ComplexityRoot.Extension.InstalledVersion(childComplexity), true
	case "Extension.isNsfw":
		if e.ComplexityRoot.Extension.IsNsfw == nil {
			break
		}

		return e.ComplexityRoot.Extension.IsNsfw(childComplexity), true
	case "Extension.jarPath":
		if e.ComplexityRoot.Extension.JarPath == nil {
			break
		}

		return e.ComplexityRoot.Extension.JarPath(childComplexity), true
	case "Extension.jarUrl":
		if e.ComplexityRoot.Extension.JarURL == nil {
			break
		}

		return e.ComplexityRoot.Extension.JarURL(childComplexity), true
	case "Extension.lang":
		if e.ComplexityRoot.Extension.Lang == nil {
			break
		}

		return e.ComplexityRoot.Extension.Lang(childComplexity), true
	case "Extension.name":
		if e.ComplexityRoot.Extension.Name == nil {
			break
		}

		return e.ComplexityRoot.Extension.Name(childComplexity), true
	case "Extension.needsUpdate":
		if e.ComplexityRoot.Extension.NeedsUpdate == nil {
			break
		}

		return e.ComplexityRoot.Extension.NeedsUpdate(childComplexity), true
	case "Extension.packageName":
		if e.ComplexityRoot.Extension.PackageName == nil {
			break
		}

		return e.ComplexityRoot.Extension.PackageName(childComplexity), true
	case "Extension.repositoryId":
		if e.ComplexityRoot.Extension.RepositoryID == nil {
			break
		}

		return e.ComplexityRoot.Extension.RepositoryID(childComplexity), true
	case "Extension.supportsLatest":
		if e.ComplexityRoot.Extension.SupportsLatest == nil {
			break
		}

		return e.ComplexityRoot.Extension.SupportsLatest(childComplexity), true
	case "Extension.version":
		if e.ComplexityRoot.Extension.Version == nil {
			break
		}

		return e.ComplexityRoot.Extension.Version(childComplexity), true

	case "Folder.id":
		if e.ComplexityRoot.Folder.ID == nil {
			break
		}

		return e.ComplexityRoot.Folder.ID(childComplexity), true
	case "Folder.includeInDownload":
		if e.ComplexityRoot.Folder.IncludeInDownload == nil {
			break
		}

		return e.ComplexityRoot.Folder.IncludeInDownload(childComplexity), true
	case "Folder.includeInUpdate":
		if e.ComplexityRoot.Folder.IncludeInUpdate == nil {
			break
		}

		return e.ComplexityRoot.Folder.IncludeInUpdate(childComplexity), true
	case "Folder.kind":
		if e.ComplexityRoot.Folder.Kind == nil {
			break
		}

		return e.ComplexityRoot.Folder.Kind(childComplexity), true
	case "Folder.name":
		if e.ComplexityRoot.Folder.Name == nil {
			break
		}

		return e.ComplexityRoot.Folder.Name(childComplexity), true
	case "Folder.parentFolderId":
		if e.ComplexityRoot.Folder.ParentFolderID == nil {
			break
		}

		return e.ComplexityRoot.Folder.ParentFolderID(childComplexity), true
	case "Folder.sortOrder":
		if e.ComplexityRoot.Folder.SortOrder == nil {
			break
		}

		return e.ComplexityRoot.Folder.SortOrder(childComplexity), true
	case "Folder.systemKey":
		if e.ComplexityRoot.Folder.SystemKey == nil {
			break
		}

		return e.ComplexityRoot.Folder.SystemKey(childComplexity), true

	case "GroupFilter.children":
		if e.ComplexityRoot.GroupFilter.Children == nil {
			break
		}

		return e.ComplexityRoot.GroupFilter.Children(childComplexity), true
	case "GroupFilter.name":
		if e.ComplexityRoot.GroupFilter.Name == nil {
			break
		}

		return e.ComplexityRoot.GroupFilter.Name(childComplexity), true

	case "HeaderFilter.name":
		if e.ComplexityRoot.HeaderFilter.Name == nil {
			break
		}

		return e.ComplexityRoot.HeaderFilter.Name(childComplexity), true

	case "LibraryUpdateStatus.currentTitle":
		if e.ComplexityRoot.LibraryUpdateStatus.CurrentTitle == nil {
			break
		}

		return e.ComplexityRoot.LibraryUpdateStatus.CurrentTitle(childComplexity), true
	case "LibraryUpdateStatus.done":
		if e.ComplexityRoot.LibraryUpdateStatus.Done == nil {
			break
		}

		return e.ComplexityRoot.LibraryUpdateStatus.Done(childComplexity), true
	case "LibraryUpdateStatus.failedTitles":
		if e.ComplexityRoot.LibraryUpdateStatus.FailedTitles == nil {
			break
		}

		return e.ComplexityRoot.LibraryUpdateStatus.FailedTitles(childComplexity), true
	case "LibraryUpdateStatus.finishedAt":
		if e.ComplexityRoot.LibraryUpdateStatus.FinishedAt == nil {
			break
		}

		return e.ComplexityRoot.LibraryUpdateStatus.FinishedAt(childComplexity), true
	case "LibraryUpdateStatus.newChapterCount":
		if e.ComplexityRoot.LibraryUpdateStatus.NewChapterCount == nil {
			break
		}

		return e.ComplexityRoot.LibraryUpdateStatus.NewChapterCount(childComplexity), true
	case "LibraryUpdateStatus.running":
		if e.ComplexityRoot.LibraryUpdateStatus.Running == nil {
			break
		}

		return e.ComplexityRoot.LibraryUpdateStatus.Running(childComplexity), true
	case "LibraryUpdateStatus.startedAt":
		if e.ComplexityRoot.LibraryUpdateStatus.StartedAt == nil {
			break
		}

		return e.ComplexityRoot.LibraryUpdateStatus.StartedAt(childComplexity), true
	case "LibraryUpdateStatus.total":
		if e.ComplexityRoot.LibraryUpdateStatus.Total == nil {
			break
		}

		return e.ComplexityRoot.LibraryUpdateStatus.Total(childComplexity), true

	case "Media.addedAt":
		if e.ComplexityRoot.Media.AddedAt == nil {
			break
		}

		return e.ComplexityRoot.Media.AddedAt(childComplexity), true
	case "Media.artist":
		if e.ComplexityRoot.Media.Artist == nil {
			break
		}

		return e.ComplexityRoot.Media.Artist(childComplexity), true
	case "Media.author":
		if e.ComplexityRoot.Media.Author == nil {
			break
		}

		return e.ComplexityRoot.Media.Author(childComplexity), true
	case "Media.chapterCount":
		if e.ComplexityRoot.Media.ChapterCount == nil {
			break
		}

		return e.ComplexityRoot.Media.ChapterCount(childComplexity), true
	case "Media.chapters":
		if e.ComplexityRoot.Media.Chapters == nil {
			break
		}

		return e.ComplexityRoot.Media.Chapters(childComplexity), true
	case "Media.contentType":
		if e.ComplexityRoot.Media.ContentType == nil {
			break
		}

		return e.ComplexityRoot.Media.ContentType(childComplexity), true
	case "Media.description":
		if e.ComplexityRoot.Media.Description == nil {
			break
		}

		return e.ComplexityRoot.Media.Description(childComplexity), true
	case "Media.detailsFetchedAt":
		if e.ComplexityRoot.Media.DetailsFetchedAt == nil {
			break
		}

		return e.ComplexityRoot.Media.DetailsFetchedAt(childComplexity), true
	case "Media.downloadedCount":
		if e.ComplexityRoot.Media.DownloadedCount == nil {
			break
		}

		return e.ComplexityRoot.Media.DownloadedCount(childComplexity), true
	case "Media.extensionId":
		if e.ComplexityRoot.Media.ExtensionID == nil {
			break
		}

		return e.ComplexityRoot.Media.ExtensionID(childComplexity), true
	case "Media.extensionName":
		if e.ComplexityRoot.Media.ExtensionName == nil {
			break
		}

		return e.ComplexityRoot.Media.ExtensionName(childComplexity), true
	case "Media.extensionRemovedAt":
		if e.ComplexityRoot.Media.ExtensionRemovedAt == nil {
			break
		}

		return e.ComplexityRoot.Media.ExtensionRemovedAt(childComplexity), true
	case "Media.externalId":
		if e.ComplexityRoot.Media.ExternalID == nil {
			break
		}

		return e.ComplexityRoot.Media.ExternalID(childComplexity), true
	case "Media.folders":
		if e.ComplexityRoot.Media.Folders == nil {
			break
		}

		return e.ComplexityRoot.Media.Folders(childComplexity), true
	case "Media.genres":
		if e.ComplexityRoot.Media.Genres == nil {
			break
		}

		return e.ComplexityRoot.Media.Genres(childComplexity), true
	case "Media.id":
		if e.ComplexityRoot.Media.ID == nil {
			break
		}

		return e.ComplexityRoot.Media.ID(childComplexity), true
	case "Media.inLibrary":
		if e.ComplexityRoot.Media.InLibrary == nil {
			break
		}

		return e.ComplexityRoot.Media.InLibrary(childComplexity), true
	case "Media.lastViewedAt":
		if e.ComplexityRoot.Media.LastViewedAt == nil {
			break
		}

		return e.ComplexityRoot.Media.LastViewedAt(childComplexity), true
	case "Media.latestChapter":
		if e.ComplexityRoot.Media.LatestChapter == nil {
			break
		}

		return e.ComplexityRoot.Media.LatestChapter(childComplexity), true
	case "Media.metadata":
		if e.ComplexityRoot.Media.Metadata == nil {
			break
		}

		return e.ComplexityRoot.Media.Metadata(childComplexity), true
	case "Media.nextUnreadChapter":
		if e.ComplexityRoot.Media.NextUnreadChapter == nil {
			break
		}

		return e.ComplexityRoot.Media.NextUnreadChapter(childComplexity), true
	case "Media.readingProgress":
		if e.ComplexityRoot.Media.ReadingProgress == nil {
			break
		}

		return e.ComplexityRoot.Media.ReadingProgress(childComplexity), true
	case "Media.source":
		if e.ComplexityRoot.Media.Source == nil {
			break
		}

		return e.ComplexityRoot.Media.Source(childComplexity), true
	case "Media.sourceName":
		if e.ComplexityRoot.Media.SourceName == nil {
			break
		}

		return e.ComplexityRoot.Media.SourceName(childComplexity), true
	case "Media.status":
		if e.ComplexityRoot.Media.Status == nil {
			break
		}

		return e.ComplexityRoot.Media.Status(childComplexity), true
	case "Media.tags":
		if e.ComplexityRoot.Media.Tags == nil {
			break
		}

		return e.ComplexityRoot.Media.Tags(childComplexity), true
	case "Media.thumbnailUrl":
		if e.ComplexityRoot.Media.ThumbnailURL == nil {
			break
		}

		return e.ComplexityRoot.Media.ThumbnailURL(childComplexity), true
	case "Media.title":
		if e.ComplexityRoot.Media.Title == nil {
			break
		}

		return e.ComplexityRoot.Media.Title(childComplexity), true
	case "Media.trackLinks":
		if e.ComplexityRoot.Media.TrackLinks == nil {
			break
		}

		return e.ComplexityRoot.Media.TrackLinks(childComplexity), true
	case "Media.unreadCount":
		if e.ComplexityRoot.Media.UnreadCount == nil {
			break
		}

		return e.ComplexityRoot.Media.UnreadCount(childComplexity), true

	case "MediaPage.hasMore":
		if e.ComplexityRoot.MediaPage.HasMore == nil {
			break
		}

		return e.ComplexityRoot.MediaPage.HasMore(childComplexity), true
	case "MediaPage.items":
		if e.ComplexityRoot.MediaPage.Items == nil {
			break
		}

		return e.ComplexityRoot.MediaPage.Items(childComplexity), true
	case "MediaPage.total":
		if e.ComplexityRoot.MediaPage.Total == nil {
			break
		}

		return e.ComplexityRoot.MediaPage.Total(childComplexity), true

	case "MetadataCandidate.coverUrl":
		if e.ComplexityRoot.MetadataCandidate.CoverURL == nil {
			break
		}

		return e.ComplexityRoot.MetadataCandidate.CoverURL(childComplexity), true
	case "MetadataCandidate.description":
		if e.ComplexityRoot.MetadataCandidate.Description == nil {
			break
		}

		return e.ComplexityRoot.MetadataCandidate.Description(childComplexity), true
	case "MetadataCandidate.genres":
		if e.ComplexityRoot.MetadataCandidate.Genres == nil {
			break
		}

		return e.ComplexityRoot.MetadataCandidate.Genres(childComplexity), true
	case "MetadataCandidate.provider":
		if e.ComplexityRoot.MetadataCandidate.Provider == nil {
			break
		}

		return e.ComplexityRoot.MetadataCandidate.Provider(childComplexity), true
	case "MetadataCandidate.providerId":
		if e.ComplexityRoot.MetadataCandidate.ProviderID == nil {
			break
		}

		return e.ComplexityRoot.MetadataCandidate.ProviderID(childComplexity), true
	case "MetadataCandidate.startYear":
		if e.ComplexityRoot.MetadataCandidate.StartYear == nil {
			break
		}

		return e.ComplexityRoot.MetadataCandidate.StartYear(childComplexity), true
	case "MetadataCandidate.status":
		if e.ComplexityRoot.MetadataCandidate.Status == nil {
			break
		}

		return e.ComplexityRoot.MetadataCandidate.Status(childComplexity), true
	case "MetadataCandidate.title":
		if e.ComplexityRoot.MetadataCandidate.Title == nil {
			break
		}

		return e.ComplexityRoot.MetadataCandidate.Title(childComplexity), true
	case "MetadataCandidate.url":
		if e.ComplexityRoot.MetadataCandidate.URL == nil {
			break
		}

		return e.ComplexityRoot.MetadataCandidate.URL(childComplexity), true

	case "MetadataMatch.confidence":
		if e.ComplexityRoot.MetadataMatch.Confidence == nil {
			break
		}

		return e.ComplexityRoot.MetadataMatch.Confidence(childComplexity), true
	case "MetadataMatch.locked":
		if e.ComplexityRoot.MetadataMatch.Locked == nil {
			break
		}

		return e.ComplexityRoot.MetadataMatch.Locked(childComplexity), true
	case "MetadataMatch.matchedAt":
		if e.ComplexityRoot.MetadataMatch.MatchedAt == nil {
			break
		}

		return e.ComplexityRoot.MetadataMatch.MatchedAt(childComplexity), true
	case "MetadataMatch.provider":
		if e.ComplexityRoot.MetadataMatch.Provider == nil {
			break
		}

		return e.ComplexityRoot.MetadataMatch.Provider(childComplexity), true
	case "MetadataMatch.providerId":
		if e.ComplexityRoot.MetadataMatch.ProviderID == nil {
			break
		}

		return e.ComplexityRoot.MetadataMatch.ProviderID(childComplexity), true
	case "MetadataMatch.url":
		if e.ComplexityRoot.MetadataMatch.URL == nil {
			break
		}

		return e.ComplexityRoot.MetadataMatch.URL(childComplexity), true

	case "Mutation.addMediaToFolder":
		if e.ComplexityRoot.Mutation.AddMediaToFolder == nil {
			break
		}

		args, err := ec.field_Mutation_addMediaToFolder_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.AddMediaToFolder(childComplexity, args["mediaId"].(string), args["folderId"].(string)), true
	case "Mutation.addRepository":
		if e.ComplexityRoot.Mutation.AddRepository == nil {
			break
		}

		args, err := ec.field_Mutation_addRepository_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.AddRepository(childComplexity, args["indexUrl"].(string), args["name"].(*string)), true
	case "Mutation.applyMetadataMatch":
		if e.ComplexityRoot.Mutation.ApplyMetadataMatch == nil {
			break
		}

		args, err := ec.field_Mutation_applyMetadataMatch_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.ApplyMetadataMatch(childComplexity, args["mediaId"].(string), args["providerId"].(string), args["provider"].(*string)), true
	case "Mutation.bindTrack":
		if e.ComplexityRoot.Mutation.BindTrack == nil {
			break
		}

		args, err := ec.field_Mutation_bindTrack_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.BindTrack(childComplexity, args["mediaId"].(string), args["trackerKey"].(string), args["remoteId"].(string)), true
	case "Mutation.clearDownloads":
		if e.ComplexityRoot.Mutation.ClearDownloads == nil {
			break
		}

		args, err := ec.field_Mutation_clearDownloads_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.ClearDownloads(childComplexity, args["status"].([]model.DownloadStatus)), true
	case "Mutation.clearImageCache":
		if e.ComplexityRoot.Mutation.ClearImageCache == nil {
			break
		}

		return e.ComplexityRoot.Mutation.ClearImageCache(childComplexity), true
	case "Mutation.createFolder":
		if e.ComplexityRoot.Mutation.CreateFolder == nil {
			break
		}

		args, err := ec.field_Mutation_createFolder_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.CreateFolder(childComplexity, args["name"].(string), args["parentFolderId"].(*string)), true
	case "Mutation.deleteDownload":
		if e.ComplexityRoot.Mutation.DeleteDownload == nil {
			break
		}

		args, err := ec.field_Mutation_deleteDownload_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.DeleteDownload(childComplexity, args["mediaId"].(string), args["chapterIds"].([]string)), true
	case "Mutation.deleteFolder":
		if e.ComplexityRoot.Mutation.DeleteFolder == nil {
			break
		}

		args, err := ec.field_Mutation_deleteFolder_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.DeleteFolder(childComplexity, args["folderId"].(string)), true
	case "Mutation.deleteRepository":
		if e.ComplexityRoot.Mutation.DeleteRepository == nil {
			break
		}

		args, err := ec.field_Mutation_deleteRepository_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.DeleteRepository(childComplexity, args["repositoryId"].(string)), true
	case "Mutation.dequeueDownload":
		if e.ComplexityRoot.Mutation.DequeueDownload == nil {
			break
		}

		args, err := ec.field_Mutation_dequeueDownload_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.DequeueDownload(childComplexity, args["mediaId"].(string), args["chapterId"].(string)), true
	case "Mutation.enqueueDownload":
		if e.ComplexityRoot.Mutation.EnqueueDownload == nil {
			break
		}

		args, err := ec.field_Mutation_enqueueDownload_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.EnqueueDownload(childComplexity, args["mediaId"].(string), args["chapterIds"].([]string)), true
	case "Mutation.installExtension":
		if e.ComplexityRoot.Mutation.InstallExtension == nil {
			break
		}

		args, err := ec.field_Mutation_installExtension_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.InstallExtension(childComplexity, args["packageName"].(string)), true
	case "Mutation.installExternalExtension":
		if e.ComplexityRoot.Mutation.InstallExternalExtension == nil {
			break
		}

		args, err := ec.field_Mutation_installExternalExtension_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.InstallExternalExtension(childComplexity, args["url"].(string)), true
	case "Mutation.markChapterRead":
		if e.ComplexityRoot.Mutation.MarkChapterRead == nil {
			break
		}

		args, err := ec.field_Mutation_markChapterRead_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.MarkChapterRead(childComplexity, args["mediaId"].(string), args["chapterId"].(string)), true
	case "Mutation.markChaptersRead":
		if e.ComplexityRoot.Mutation.MarkChaptersRead == nil {
			break
		}

		args, err := ec.field_Mutation_markChaptersRead_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.MarkChaptersRead(childComplexity, args["mediaId"].(string), args["chapterIds"].([]string), args["read"].(bool)), true
	case "Mutation.migrateMedia":
		if e.ComplexityRoot.Mutation.MigrateMedia == nil {
			break
		}

		args, err := ec.field_Mutation_migrateMedia_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.MigrateMedia(childComplexity, args["fromMediaId"].(string), args["toExtensionId"].(string), args["toExternalId"].(string)), true
	case "Mutation.pullTracker":
		if e.ComplexityRoot.Mutation.PullTracker == nil {
			break
		}

		args, err := ec.field_Mutation_pullTracker_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.PullTracker(childComplexity, args["mediaId"].(string)), true
	case "Mutation.refreshFolder":
		if e.ComplexityRoot.Mutation.RefreshFolder == nil {
			break
		}

		args, err := ec.field_Mutation_refreshFolder_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.RefreshFolder(childComplexity, args["folderId"].(string)), true
	case "Mutation.refreshMetadata":
		if e.ComplexityRoot.Mutation.RefreshMetadata == nil {
			break
		}

		args, err := ec.field_Mutation_refreshMetadata_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.RefreshMetadata(childComplexity, args["mediaId"].(string), args["syncChapters"].(*bool)), true
	case "Mutation.refreshMetadataMatch":
		if e.ComplexityRoot.Mutation.RefreshMetadataMatch == nil {
			break
		}

		args, err := ec.field_Mutation_refreshMetadataMatch_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.RefreshMetadataMatch(childComplexity, args["mediaId"].(string)), true
	case "Mutation.removeMediaFromFolder":
		if e.ComplexityRoot.Mutation.RemoveMediaFromFolder == nil {
			break
		}

		args, err := ec.field_Mutation_removeMediaFromFolder_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.RemoveMediaFromFolder(childComplexity, args["mediaId"].(string), args["folderId"].(string)), true
	case "Mutation.renameFolder":
		if e.ComplexityRoot.Mutation.RenameFolder == nil {
			break
		}

		args, err := ec.field_Mutation_renameFolder_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.RenameFolder(childComplexity, args["folderId"].(string), args["name"].(string)), true
	case "Mutation.renameRepository":
		if e.ComplexityRoot.Mutation.RenameRepository == nil {
			break
		}

		args, err := ec.field_Mutation_renameRepository_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.RenameRepository(childComplexity, args["repositoryId"].(string), args["name"].(string)), true
	case "Mutation.reorderDownload":
		if e.ComplexityRoot.Mutation.ReorderDownload == nil {
			break
		}

		args, err := ec.field_Mutation_reorderDownload_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.ReorderDownload(childComplexity, args["mediaId"].(string), args["chapterId"].(string), args["position"].(int32)), true
	case "Mutation.reorderFolder":
		if e.ComplexityRoot.Mutation.ReorderFolder == nil {
			break
		}

		args, err := ec.field_Mutation_reorderFolder_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.ReorderFolder(childComplexity, args["folderId"].(string), args["sortOrder"].(int32)), true
	case "Mutation.rescanLocalMedia":
		if e.ComplexityRoot.Mutation.RescanLocalMedia == nil {
			break
		}

		return e.ComplexityRoot.Mutation.RescanLocalMedia(childComplexity), true
	case "Mutation.resyncTrack":
		if e.ComplexityRoot.Mutation.ResyncTrack == nil {
			break
		}

		args, err := ec.field_Mutation_resyncTrack_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.ResyncTrack(childComplexity, args["linkId"].(string)), true
	case "Mutation.retryDownload":
		if e.ComplexityRoot.Mutation.RetryDownload == nil {
			break
		}

		args, err := ec.field_Mutation_retryDownload_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.RetryDownload(childComplexity, args["mediaId"].(string), args["chapterId"].(string)), true
	case "Mutation.setInLibrary":
		if e.ComplexityRoot.Mutation.SetInLibrary == nil {
			break
		}

		args, err := ec.field_Mutation_setInLibrary_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.SetInLibrary(childComplexity, args["mediaId"].(string), args["inLibrary"].(bool)), true
	case "Mutation.setMediaCover":
		if e.ComplexityRoot.Mutation.SetMediaCover == nil {
			break
		}

		args, err := ec.field_Mutation_setMediaCover_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.SetMediaCover(childComplexity, args["mediaId"].(string), args["url"].(*string)), true
	case "Mutation.startDownloader":
		if e.ComplexityRoot.Mutation.StartDownloader == nil {
			break
		}

		return e.ComplexityRoot.Mutation.StartDownloader(childComplexity), true
	case "Mutation.startLibraryUpdate":
		if e.ComplexityRoot.Mutation.StartLibraryUpdate == nil {
			break
		}

		args, err := ec.field_Mutation_startLibraryUpdate_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.StartLibraryUpdate(childComplexity, args["folderId"].(*string)), true
	case "Mutation.stopDownloader":
		if e.ComplexityRoot.Mutation.StopDownloader == nil {
			break
		}

		return e.ComplexityRoot.Mutation.StopDownloader(childComplexity), true
	case "Mutation.syncChapters":
		if e.ComplexityRoot.Mutation.SyncChapters == nil {
			break
		}

		args, err := ec.field_Mutation_syncChapters_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.SyncChapters(childComplexity, args["mediaId"].(string)), true
	case "Mutation.trackerLogin":
		if e.ComplexityRoot.Mutation.TrackerLogin == nil {
			break
		}

		args, err := ec.field_Mutation_trackerLogin_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.TrackerLogin(childComplexity, args["trackerKey"].(string), args["token"].(string)), true
	case "Mutation.trackerLogout":
		if e.ComplexityRoot.Mutation.TrackerLogout == nil {
			break
		}

		args, err := ec.field_Mutation_trackerLogout_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.TrackerLogout(childComplexity, args["trackerKey"].(string)), true
	case "Mutation.unbindTrack":
		if e.ComplexityRoot.Mutation.UnbindTrack == nil {
			break
		}

		args, err := ec.field_Mutation_unbindTrack_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.UnbindTrack(childComplexity, args["linkId"].(string)), true
	case "Mutation.uninstallExtension":
		if e.ComplexityRoot.Mutation.UninstallExtension == nil {
			break
		}

		args, err := ec.field_Mutation_uninstallExtension_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.UninstallExtension(childComplexity, args["packageName"].(string)), true
	case "Mutation.unlinkMetadata":
		if e.ComplexityRoot.Mutation.UnlinkMetadata == nil {
			break
		}

		args, err := ec.field_Mutation_unlinkMetadata_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.UnlinkMetadata(childComplexity, args["mediaId"].(string)), true
	case "Mutation.updateExtension":
		if e.ComplexityRoot.Mutation.UpdateExtension == nil {
			break
		}

		args, err := ec.field_Mutation_updateExtension_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.UpdateExtension(childComplexity, args["packageName"].(string)), true
	case "Mutation.updateFolderFlags":
		if e.ComplexityRoot.Mutation.UpdateFolderFlags == nil {
			break
		}

		args, err := ec.field_Mutation_updateFolderFlags_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.UpdateFolderFlags(childComplexity, args["folderId"].(string), args["includeInUpdate"].(*bool), args["includeInDownload"].(*bool)), true
	case "Mutation.updateReadingProgress":
		if e.ComplexityRoot.Mutation.UpdateReadingProgress == nil {
			break
		}

		args, err := ec.field_Mutation_updateReadingProgress_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.UpdateReadingProgress(childComplexity, args["mediaId"].(string), args["chapterId"].(string), args["progress"].(float64), args["completed"].(*bool), args["positionSeconds"].(*float64), args["durationSeconds"].(*float64)), true
	case "Mutation.updateTrack":
		if e.ComplexityRoot.Mutation.UpdateTrack == nil {
			break
		}

		args, err := ec.field_Mutation_updateTrack_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Mutation.UpdateTrack(childComplexity, args["linkId"].(string), args["status"].(*int32), args["score"].(*float64), args["lastChapterRead"].(*float64)), true

	case "Query.about":
		if e.ComplexityRoot.Query.About == nil {
			break
		}

		return e.ComplexityRoot.Query.About(childComplexity), true
	case "Query.availableExtensions":
		if e.ComplexityRoot.Query.AvailableExtensions == nil {
			break
		}

		args, err := ec.field_Query_availableExtensions_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.AvailableExtensions(childComplexity, args["repositoryId"].(string)), true
	case "Query.chapter":
		if e.ComplexityRoot.Query.Chapter == nil {
			break
		}

		args, err := ec.field_Query_chapter_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.Chapter(childComplexity, args["id"].(string)), true
	case "Query.chapterUpdates":
		if e.ComplexityRoot.Query.ChapterUpdates == nil {
			break
		}

		args, err := ec.field_Query_chapterUpdates_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.ChapterUpdates(childComplexity, args["since"].(*time.Time), args["limit"].(*int32)), true
	case "Query.downloadQueue":
		if e.ComplexityRoot.Query.DownloadQueue == nil {
			break
		}

		return e.ComplexityRoot.Query.DownloadQueue(childComplexity), true
	case "Query.downloadStatus":
		if e.ComplexityRoot.Query.DownloadStatus == nil {
			break
		}

		args, err := ec.field_Query_downloadStatus_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.DownloadStatus(childComplexity, args["mediaId"].(string), args["chapterId"].(string)), true
	case "Query.downloaderStatus":
		if e.ComplexityRoot.Query.DownloaderStatus == nil {
			break
		}

		return e.ComplexityRoot.Query.DownloaderStatus(childComplexity), true
	case "Query.filterOptions":
		if e.ComplexityRoot.Query.FilterOptions == nil {
			break
		}

		args, err := ec.field_Query_filterOptions_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.FilterOptions(childComplexity, args["extensionId"].(string)), true
	case "Query.folder":
		if e.ComplexityRoot.Query.Folder == nil {
			break
		}

		args, err := ec.field_Query_folder_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.Folder(childComplexity, args["id"].(string)), true
	case "Query.folders":
		if e.ComplexityRoot.Query.Folders == nil {
			break
		}

		return e.ComplexityRoot.Query.Folders(childComplexity), true
	case "Query.installedExtensions":
		if e.ComplexityRoot.Query.InstalledExtensions == nil {
			break
		}

		return e.ComplexityRoot.Query.InstalledExtensions(childComplexity), true

	case "Query.latestUpdates":
		if e.ComplexityRoot.Query.LatestUpdates == nil {
			break
		}

		args, err := ec.field_Query_latestUpdates_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.LatestUpdates(childComplexity, args["extensionId"].(string), args["page"].(*int32)), true
	case "Query.library":
		if e.ComplexityRoot.Query.Library == nil {
			break
		}

		args, err := ec.field_Query_library_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.Library(childComplexity, args["filter"].(*model.LibraryFilter), args["sort"].(*model.LibrarySortInput), args["limit"].(*int32), args["offset"].(*int32)), true
	case "Query.libraryUpdateStatus":
		if e.ComplexityRoot.Query.LibraryUpdateStatus == nil {
			break
		}

		return e.ComplexityRoot.Query.LibraryUpdateStatus(childComplexity), true
	case "Query.media":
		if e.ComplexityRoot.Query.Media == nil {
			break
		}

		args, err := ec.field_Query_media_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.Media(childComplexity, args["id"].(string)), true
	case "Query.mediaInFolder":
		if e.ComplexityRoot.Query.MediaInFolder == nil {
			break
		}

		args, err := ec.field_Query_mediaInFolder_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.MediaInFolder(childComplexity, args["folderId"].(string)), true
	case "Query.popularManga":
		if e.ComplexityRoot.Query.PopularManga == nil {
			break
		}

		args, err := ec.field_Query_popularManga_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.PopularManga(childComplexity, args["extensionId"].(string), args["page"].(*int32)), true
	case "Query.readingProgress":
		if e.ComplexityRoot.Query.ReadingProgress == nil {
			break
		}

		args, err := ec.field_Query_readingProgress_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.ReadingProgress(childComplexity, args["mediaId"].(string)), true
	case "Query.recentChapters":
		if e.ComplexityRoot.Query.RecentChapters == nil {
			break
		}

		args, err := ec.field_Query_recentChapters_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.RecentChapters(childComplexity, args["since"].(*time.Time), args["limit"].(*int32)), true
	case "Query.repositories":
		if e.ComplexityRoot.Query.Repositories == nil {
			break
		}

		return e.ComplexityRoot.Query.Repositories(childComplexity), true
	case "Query.resolveMedia":
		if e.ComplexityRoot.Query.ResolveMedia == nil {
			break
		}

		args, err := ec.field_Query_resolveMedia_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.ResolveMedia(childComplexity, args["extensionId"].(string), args["externalId"].(string), args["syncChapters"].(*bool)), true
	case "Query.search":
		if e.ComplexityRoot.Query.Search == nil {
			break
		}

		args, err := ec.field_Query_search_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.Search(childComplexity, args["extensionId"].(string), args["query"].(string), args["page"].(*int32), args["filters"].([]*model.FilterInput)), true
	case "Query.searchMetadata":
		if e.ComplexityRoot.Query.SearchMetadata == nil {
			break
		}

		args, err := ec.field_Query_searchMetadata_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.SearchMetadata(childComplexity, args["query"].(string), args["contentType"].(model.ContentType), args["provider"].(*string)), true
	case "Query.skipTimestamps":
		if e.ComplexityRoot.Query.SkipTimestamps == nil {
			break
		}

		args, err := ec.field_Query_skipTimestamps_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.SkipTimestamps(childComplexity, args["chapterId"].(string), args["episodeLengthMs"].(*int32)), true
	case "Query.storageInfo":
		if e.ComplexityRoot.Query.StorageInfo == nil {
			break
		}

		return e.ComplexityRoot.Query.StorageInfo(childComplexity), true
	case "Query.trackSearch":
		if e.ComplexityRoot.Query.TrackSearch == nil {
			break
		}

		args, err := ec.field_Query_trackSearch_args(ctx, rawArgs)
		if err != nil {
			return 0, false
		}

		return e.ComplexityRoot.Query.TrackSearch(childComplexity, args["trackerKey"].(string), args["query"].(string), args["contentType"].(*model.ContentType)), true
	case "Query.trackers":
		if e.ComplexityRoot.Query.Trackers == nil {
			break
		}

		return e.ComplexityRoot.Query.Trackers(childComplexity), true

	case "ReadingProgress.chapterId":
		if e.ComplexityRoot.ReadingProgress.ChapterID == nil {
			break
		}

		return e.ComplexityRoot.ReadingProgress.ChapterID(childComplexity), true
	case "ReadingProgress.completed":
		if e.ComplexityRoot.ReadingProgress.Completed == nil {
			break
		}

		return e.ComplexityRoot.ReadingProgress.Completed(childComplexity), true
	case "ReadingProgress.durationSeconds":
		if e.ComplexityRoot.ReadingProgress.DurationSeconds == nil {
			break
		}

		return e.ComplexityRoot.ReadingProgress.DurationSeconds(childComplexity), true
	case "ReadingProgress.id":
		if e.ComplexityRoot.ReadingProgress.ID == nil {
			break
		}

		return e.ComplexityRoot.ReadingProgress.ID(childComplexity), true
	case "ReadingProgress.mediaId":
		if e.ComplexityRoot.ReadingProgress.MediaID == nil {
			break
		}

		return e.ComplexityRoot.ReadingProgress.MediaID(childComplexity), true
	case "ReadingProgress.positionSeconds":
		if e.ComplexityRoot.ReadingProgress.PositionSeconds == nil {
			break
		}

		return e.ComplexityRoot.ReadingProgress.PositionSeconds(childComplexity), true
	case "ReadingProgress.progress":
		if e.ComplexityRoot.ReadingProgress.Progress == nil {
			break
		}

		return e.ComplexityRoot.ReadingProgress.Progress(childComplexity), true
	case "ReadingProgress.updatedAt":
		if e.ComplexityRoot.ReadingProgress.UpdatedAt == nil {
			break
		}

		return e.ComplexityRoot.ReadingProgress.UpdatedAt(childComplexity), true

	case "RecentChapter.chapter":
		if e.ComplexityRoot.RecentChapter.Chapter == nil {
			break
		}

		return e.ComplexityRoot.RecentChapter.Chapter(childComplexity), true
	case "RecentChapter.media":
		if e.ComplexityRoot.RecentChapter.Media == nil {
			break
		}

		return e.ComplexityRoot.RecentChapter.Media(childComplexity), true

	case "Repository.addedAt":
		if e.ComplexityRoot.Repository.AddedAt == nil {
			break
		}

		return e.ComplexityRoot.Repository.AddedAt(childComplexity), true
	case "Repository.contentType":
		if e.ComplexityRoot.Repository.ContentType == nil {
			break
		}

		return e.ComplexityRoot.Repository.ContentType(childComplexity), true
	case "Repository.id":
		if e.ComplexityRoot.Repository.ID == nil {
			break
		}

		return e.ComplexityRoot.Repository.ID(childComplexity), true
	case "Repository.indexUrl":
		if e.ComplexityRoot.Repository.IndexURL == nil {
			break
		}

		return e.ComplexityRoot.Repository.IndexURL(childComplexity), true
	case "Repository.lastSyncedAt":
		if e.ComplexityRoot.Repository.LastSyncedAt == nil {
			break
		}

		return e.ComplexityRoot.Repository.LastSyncedAt(childComplexity), true
	case "Repository.name":
		if e.ComplexityRoot.Repository.Name == nil {
			break
		}

		return e.ComplexityRoot.Repository.Name(childComplexity), true

	case "SearchResponse.hasNextPage":
		if e.ComplexityRoot.SearchResponse.HasNextPage == nil {
			break
		}

		return e.ComplexityRoot.SearchResponse.HasNextPage(childComplexity), true
	case "SearchResponse.results":
		if e.ComplexityRoot.SearchResponse.Results == nil {
			break
		}

		return e.ComplexityRoot.SearchResponse.Results(childComplexity), true

	case "SelectFilter.name":
		if e.ComplexityRoot.SelectFilter.Name == nil {
			break
		}

		return e.ComplexityRoot.SelectFilter.Name(childComplexity), true
	case "SelectFilter.state":
		if e.ComplexityRoot.SelectFilter.State == nil {
			break
		}

		return e.ComplexityRoot.SelectFilter.State(childComplexity), true
	case "SelectFilter.values":
		if e.ComplexityRoot.SelectFilter.Values == nil {
			break
		}

		return e.ComplexityRoot.SelectFilter.Values(childComplexity), true

	case "SeparatorFilter.name":
		if e.ComplexityRoot.SeparatorFilter.Name == nil {
			break
		}

		return e.ComplexityRoot.SeparatorFilter.Name(childComplexity), true

	case "SkipMarker.endMs":
		if e.ComplexityRoot.SkipMarker.EndMs == nil {
			break
		}

		return e.ComplexityRoot.SkipMarker.EndMs(childComplexity), true
	case "SkipMarker.name":
		if e.ComplexityRoot.SkipMarker.Name == nil {
			break
		}

		return e.ComplexityRoot.SkipMarker.Name(childComplexity), true
	case "SkipMarker.startMs":
		if e.ComplexityRoot.SkipMarker.StartMs == nil {
			break
		}

		return e.ComplexityRoot.SkipMarker.StartMs(childComplexity), true
	case "SkipMarker.type":
		if e.ComplexityRoot.SkipMarker.Type == nil {
			break
		}

		return e.ComplexityRoot.SkipMarker.Type(childComplexity), true

	case "SortFilter.ascending":
		if e.ComplexityRoot.SortFilter.Ascending == nil {
			break
		}

		return e.ComplexityRoot.SortFilter.Ascending(childComplexity), true
	case "SortFilter.hasState":
		if e.ComplexityRoot.SortFilter.HasState == nil {
			break
		}

		return e.ComplexityRoot.SortFilter.HasState(childComplexity), true
	case "SortFilter.index":
		if e.ComplexityRoot.SortFilter.Index == nil {
			break
		}

		return e.ComplexityRoot.SortFilter.Index(childComplexity), true
	case "SortFilter.name":
		if e.ComplexityRoot.SortFilter.Name == nil {
			break
		}

		return e.ComplexityRoot.SortFilter.Name(childComplexity), true
	case "SortFilter.values":
		if e.ComplexityRoot.SortFilter.Values == nil {
			break
		}

		return e.ComplexityRoot.SortFilter.Values(childComplexity), true

	case "StorageInfo.freeBytes":
		if e.ComplexityRoot.StorageInfo.FreeBytes == nil {
			break
		}

		return e.ComplexityRoot.StorageInfo.FreeBytes(childComplexity), true
	case "StorageInfo.totalBytes":
		if e.ComplexityRoot.StorageInfo.TotalBytes == nil {
			break
		}

		return e.ComplexityRoot.StorageInfo.TotalBytes(childComplexity), true
	case "StorageInfo.usedBytes":
		if e.ComplexityRoot.StorageInfo.UsedBytes == nil {
			break
		}

		return e.ComplexityRoot.StorageInfo.UsedBytes(childComplexity), true

	case "SubtitleTrack.lang":
		if e.ComplexityRoot.SubtitleTrack.Lang == nil {
			break
		}

		return e.ComplexityRoot.SubtitleTrack.Lang(childComplexity), true
	case "SubtitleTrack.url":
		if e.ComplexityRoot.SubtitleTrack.URL == nil {
			break
		}

		return e.ComplexityRoot.SubtitleTrack.URL(childComplexity), true

	case "TextFilter.name":
		if e.ComplexityRoot.TextFilter.Name == nil {
			break
		}

		return e.ComplexityRoot.TextFilter.Name(childComplexity), true
	case "TextFilter.state":
		if e.ComplexityRoot.TextFilter.State == nil {
			break
		}

		return e.ComplexityRoot.TextFilter.State(childComplexity), true

	case "TrackLink.finishedAt":
		if e.ComplexityRoot.TrackLink.FinishedAt == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.FinishedAt(childComplexity), true
	case "TrackLink.id":
		if e.ComplexityRoot.TrackLink.ID == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.ID(childComplexity), true
	case "TrackLink.lastChapterRead":
		if e.ComplexityRoot.TrackLink.LastChapterRead == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.LastChapterRead(childComplexity), true
	case "TrackLink.lastSyncedAt":
		if e.ComplexityRoot.TrackLink.LastSyncedAt == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.LastSyncedAt(childComplexity), true
	case "TrackLink.mediaId":
		if e.ComplexityRoot.TrackLink.MediaID == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.MediaID(childComplexity), true
	case "TrackLink.private":
		if e.ComplexityRoot.TrackLink.Private == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.Private(childComplexity), true
	case "TrackLink.remoteId":
		if e.ComplexityRoot.TrackLink.RemoteID == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.RemoteID(childComplexity), true
	case "TrackLink.score":
		if e.ComplexityRoot.TrackLink.Score == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.Score(childComplexity), true
	case "TrackLink.startedAt":
		if e.ComplexityRoot.TrackLink.StartedAt == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.StartedAt(childComplexity), true
	case "TrackLink.status":
		if e.ComplexityRoot.TrackLink.Status == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.Status(childComplexity), true
	case "TrackLink.statusName":
		if e.ComplexityRoot.TrackLink.StatusName == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.StatusName(childComplexity), true
	case "TrackLink.title":
		if e.ComplexityRoot.TrackLink.Title == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.Title(childComplexity), true
	case "TrackLink.totalChapters":
		if e.ComplexityRoot.TrackLink.TotalChapters == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.TotalChapters(childComplexity), true
	case "TrackLink.trackerKey":
		if e.ComplexityRoot.TrackLink.TrackerKey == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.TrackerKey(childComplexity), true
	case "TrackLink.url":
		if e.ComplexityRoot.TrackLink.URL == nil {
			break
		}

		return e.ComplexityRoot.TrackLink.URL(childComplexity), true

	case "TrackSearchResult.coverUrl":
		if e.ComplexityRoot.TrackSearchResult.CoverURL == nil {
			break
		}

		return e.ComplexityRoot.TrackSearchResult.CoverURL(childComplexity), true
	case "TrackSearchResult.mediaType":
		if e.ComplexityRoot.TrackSearchResult.MediaType == nil {
			break
		}

		return e.ComplexityRoot.TrackSearchResult.MediaType(childComplexity), true
	case "TrackSearchResult.publishingStatus":
		if e.ComplexityRoot.TrackSearchResult.PublishingStatus == nil {
			break
		}

		return e.ComplexityRoot.TrackSearchResult.PublishingStatus(childComplexity), true
	case "TrackSearchResult.remoteId":
		if e.ComplexityRoot.TrackSearchResult.RemoteID == nil {
			break
		}

		return e.ComplexityRoot.TrackSearchResult.RemoteID(childComplexity), true
	case "TrackSearchResult.summary":
		if e.ComplexityRoot.TrackSearchResult.Summary == nil {
			break
		}

		return e.ComplexityRoot.TrackSearchResult.Summary(childComplexity), true
	case "TrackSearchResult.title":
		if e.ComplexityRoot.TrackSearchResult.Title == nil {
			break
		}

		return e.ComplexityRoot.TrackSearchResult.Title(childComplexity), true
	case "TrackSearchResult.totalChapters":
		if e.ComplexityRoot.TrackSearchResult.TotalChapters == nil {
			break
		}

		return e.ComplexityRoot.TrackSearchResult.TotalChapters(childComplexity), true
	case "TrackSearchResult.url":
		if e.ComplexityRoot.TrackSearchResult.URL == nil {
			break
		}

		return e.ComplexityRoot.TrackSearchResult.URL(childComplexity), true

	case "TrackStatus.animeName":
		if e.ComplexityRoot.TrackStatus.AnimeName == nil {
			break
		}

		return e.ComplexityRoot.TrackStatus.AnimeName(childComplexity), true
	case "TrackStatus.name":
		if e.ComplexityRoot.TrackStatus.Name == nil {
			break
		}

		return e.ComplexityRoot.TrackStatus.Name(childComplexity), true
	case "TrackStatus.value":
		if e.ComplexityRoot.TrackStatus.Value == nil {
			break
		}

		return e.ComplexityRoot.TrackStatus.Value(childComplexity), true

	case "Tracker.authUrl":
		if e.ComplexityRoot.Tracker.AuthURL == nil {
			break
		}

		return e.ComplexityRoot.Tracker.AuthURL(childComplexity), true
	case "Tracker.configured":
		if e.ComplexityRoot.Tracker.Configured == nil {
			break
		}

		return e.ComplexityRoot.Tracker.Configured(childComplexity), true
	case "Tracker.iconUrl":
		if e.ComplexityRoot.Tracker.IconURL == nil {
			break
		}

		return e.ComplexityRoot.Tracker.IconURL(childComplexity), true
	case "Tracker.isLoggedIn":
		if e.ComplexityRoot.Tracker.IsLoggedIn == nil {
			break
		}

		return e.ComplexityRoot.Tracker.IsLoggedIn(childComplexity), true
	case "Tracker.key":
		if e.ComplexityRoot.Tracker.Key == nil {
			break
		}

		return e.ComplexityRoot.Tracker.Key(childComplexity), true
	case "Tracker.name":
		if e.ComplexityRoot.Tracker.Name == nil {
			break
		}

		return e.ComplexityRoot.Tracker.Name(childComplexity), true
	case "Tracker.scoreOptions":
		if e.ComplexityRoot.Tracker.ScoreOptions == nil {
			break
		}

		return e.ComplexityRoot.Tracker.ScoreOptions(childComplexity), true
	case "Tracker.statusOptions":
		if e.ComplexityRoot.Tracker.StatusOptions == nil {
			break
		}

		return e.ComplexityRoot.Tracker.StatusOptions(childComplexity), true
	case "Tracker.username":
		if e.ComplexityRoot.Tracker.Username == nil {
			break
		}

		return e.ComplexityRoot.Tracker.Username(childComplexity), true

	case "TriStateFilter.name":
		if e.ComplexityRoot.TriStateFilter.Name == nil {
			break
		}

		return e.ComplexityRoot.TriStateFilter.Name(childComplexity), true
	case "TriStateFilter.state":
		if e.ComplexityRoot.TriStateFilter.State == nil {
			break
		}

		return e.ComplexityRoot.TriStateFilter.State(childComplexity), true

	case "VideoSource.kind":
		if e.ComplexityRoot.VideoSource.Kind == nil {
			break
		}

		return e.ComplexityRoot.VideoSource.Kind(childComplexity), true
	case "VideoSource.label":
		if e.ComplexityRoot.VideoSource.Label == nil {
			break
		}

		return e.ComplexityRoot.VideoSource.Label(childComplexity), true
	case "VideoSource.preferred":
		if e.ComplexityRoot.VideoSource.Preferred == nil {
			break
		}

		return e.ComplexityRoot.VideoSource.Preferred(childComplexity), true
	case "VideoSource.resolution":
		if e.ComplexityRoot.VideoSource.Resolution == nil {
			break
		}

		return e.ComplexityRoot.VideoSource.Resolution(childComplexity), true
	case "VideoSource.server":
		if e.ComplexityRoot.VideoSource.Server == nil {
			break
		}

		return e.ComplexityRoot.VideoSource.Server(childComplexity), true
	case "VideoSource.url":
		if e.ComplexityRoot.VideoSource.URL == nil {
			break
		}

		return e.ComplexityRoot.VideoSource.URL(childComplexity), true

	case "VideoStream.audioTracks":
		if e.ComplexityRoot.VideoStream.AudioTracks == nil {
			break
		}

		return e.ComplexityRoot.VideoStream.AudioTracks(childComplexity), true
	case "VideoStream.skipMarkers":
		if e.ComplexityRoot.VideoStream.SkipMarkers == nil {
			break
		}

		return e.ComplexityRoot.VideoStream.SkipMarkers(childComplexity), true
	case "VideoStream.sources":
		if e.ComplexityRoot.VideoStream.Sources == nil {
			break
		}

		return e.ComplexityRoot.VideoStream.Sources(childComplexity), true
	case "VideoStream.subtitles":
		if e.ComplexityRoot.VideoStream.Subtitles == nil {
			break
		}

		return e.ComplexityRoot.VideoStream.Subtitles(childComplexity), true
	case "VideoStream.url":
		if e.ComplexityRoot.VideoStream.URL == nil {
			break
		}

		return e.ComplexityRoot.VideoStream.URL(childComplexity), true

	}
	return 0, false
}

func (e *executableSchema) Exec(ctx context.Context) graphql.ResponseHandler {
	opCtx := graphql.GetOperationContext(ctx)
	ec := newExecutionContext(opCtx, e, make(chan graphql.DeferredResult))
	inputUnmarshalMap := graphql.BuildUnmarshalerMap(
		ec.unmarshalInputCheckBoxFilterInput,
		ec.unmarshalInputFilterInput,
		ec.unmarshalInputGroupFilterInput,
		ec.unmarshalInputLibraryFilter,
		ec.unmarshalInputLibrarySortInput,
		ec.unmarshalInputSelectFilterInput,
		ec.unmarshalInputSortFilterInput,
		ec.unmarshalInputTextFilterInput,
		ec.unmarshalInputTriStateFilterInput,
	)
	first := true

	switch opCtx.Operation.Operation {
	case ast.Query:
		return func(ctx context.Context) *graphql.Response {
			var response graphql.Response
			var data graphql.Marshaler
			if first {
				first = false
				ctx = graphql.WithUnmarshalerMap(ctx, inputUnmarshalMap)
				data = ec._Query(ctx, opCtx.Operation.SelectionSet)
			} else {
				if atomic.LoadInt32(&ec.PendingDeferred) > 0 {
					result := <-ec.DeferredResults
					atomic.AddInt32(&ec.PendingDeferred, -1)
					data = result.Result
					response.Path = result.Path
					response.Label = result.Label
					response.Errors = result.Errors
				} else {
					return nil
				}
			}
			var buf bytes.Buffer
			data.MarshalGQL(&buf)
			response.Data = buf.Bytes()
			if atomic.LoadInt32(&ec.Deferred) > 0 {
				hasNext := atomic.LoadInt32(&ec.PendingDeferred) > 0
				response.HasNext = &hasNext
			}

			return &response
		}
	case ast.Mutation:
		return func(ctx context.Context) *graphql.Response {
			if !first {
				return nil
			}
			first = false
			ctx = graphql.WithUnmarshalerMap(ctx, inputUnmarshalMap)
			data := ec._Mutation(ctx, opCtx.Operation.SelectionSet)
			var buf bytes.Buffer
			data.MarshalGQL(&buf)

			return &graphql.Response{
				Data: buf.Bytes(),
			}
		}

	default:
		return graphql.OneShot(graphql.ErrorResponse(ctx, "unsupported GraphQL operation"))
	}
}

type executionContext struct {
	*graphql.ExecutionContextState[ResolverRoot, DirectiveRoot, ComplexityRoot]
}

func newExecutionContext(
	opCtx *graphql.OperationContext,
	execSchema *executableSchema,
	deferredResults chan graphql.DeferredResult,
) *executionContext {
	return &executionContext{
		ExecutionContextState: graphql.NewExecutionContextState[ResolverRoot, DirectiveRoot, ComplexityRoot](
			opCtx,
			(*graphql.ExecutableSchemaState[ResolverRoot, DirectiveRoot, ComplexityRoot])(execSchema),
			parsedSchema,
			deferredResults,
		),
	}
}

//go:embed "schema.graphqls"
var sourcesFS embed.FS

func sourceData(filename string) string {
	data, err := sourcesFS.ReadFile(filename)
	if err != nil {
		panic(fmt.Sprintf("codegen problem: %s not available", filename))
	}
	return string(data)
}

var sources = []*ast.Source{
	{Name: "schema.graphqls", Input: sourceData("schema.graphqls"), BuiltIn: false},
}
var parsedSchema = gqlparser.MustLoadSchema(sources...)

func (ec *executionContext) childFields_AboutServer(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "name":
		return ec.fieldContext_AboutServer_name(ctx, field)
	case "version":
		return ec.fieldContext_AboutServer_version(ctx, field)
	case "buildTime":
		return ec.fieldContext_AboutServer_buildTime(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type AboutServer", field.Name)
}

func (ec *executionContext) childFields_AudioTrack(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "lang":
		return ec.fieldContext_AudioTrack_lang(ctx, field)
	case "url":
		return ec.fieldContext_AudioTrack_url(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type AudioTrack", field.Name)
}

func (ec *executionContext) childFields_Chapter(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "id":
		return ec.fieldContext_Chapter_id(ctx, field)
	case "mediaId":
		return ec.fieldContext_Chapter_mediaId(ctx, field)
	case "externalId":
		return ec.fieldContext_Chapter_externalId(ctx, field)
	case "title":
		return ec.fieldContext_Chapter_title(ctx, field)
	case "number":
		return ec.fieldContext_Chapter_number(ctx, field)
	case "sourceOrder":
		return ec.fieldContext_Chapter_sourceOrder(ctx, field)
	case "scanlator":
		return ec.fieldContext_Chapter_scanlator(ctx, field)
	case "uploadedAt":
		return ec.fieldContext_Chapter_uploadedAt(ctx, field)
	case "readingProgress":
		return ec.fieldContext_Chapter_readingProgress(ctx, field)
	case "completed":
		return ec.fieldContext_Chapter_completed(ctx, field)
	case "downloaded":
		return ec.fieldContext_Chapter_downloaded(ctx, field)
	case "download":
		return ec.fieldContext_Chapter_download(ctx, field)
	case "pages":
		return ec.fieldContext_Chapter_pages(ctx, field)
	case "pageCount":
		return ec.fieldContext_Chapter_pageCount(ctx, field)
	case "videoUrl":
		return ec.fieldContext_Chapter_videoUrl(ctx, field)
	case "videoStream":
		return ec.fieldContext_Chapter_videoStream(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type Chapter", field.Name)
}

func (ec *executionContext) childFields_Download(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "id":
		return ec.fieldContext_Download_id(ctx, field)
	case "mediaId":
		return ec.fieldContext_Download_mediaId(ctx, field)
	case "chapterId":
		return ec.fieldContext_Download_chapterId(ctx, field)
	case "chapter":
		return ec.fieldContext_Download_chapter(ctx, field)
	case "status":
		return ec.fieldContext_Download_status(ctx, field)
	case "progress":
		return ec.fieldContext_Download_progress(ctx, field)
	case "downloadedBytes":
		return ec.fieldContext_Download_downloadedBytes(ctx, field)
	case "bytesPerSec":
		return ec.fieldContext_Download_bytesPerSec(ctx, field)
	case "finalSizeBytes":
		return ec.fieldContext_Download_finalSizeBytes(ctx, field)
	case "error":
		return ec.fieldContext_Download_error(ctx, field)
	case "createdAt":
		return ec.fieldContext_Download_createdAt(ctx, field)
	case "completedAt":
		return ec.fieldContext_Download_completedAt(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type Download", field.Name)
}

func (ec *executionContext) childFields_DownloaderStatus(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "isRunning":
		return ec.fieldContext_DownloaderStatus_isRunning(ctx, field)
	case "queuedCount":
		return ec.fieldContext_DownloaderStatus_queuedCount(ctx, field)
	case "downloadingCount":
		return ec.fieldContext_DownloaderStatus_downloadingCount(ctx, field)
	case "failedCount":
		return ec.fieldContext_DownloaderStatus_failedCount(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type DownloaderStatus", field.Name)
}

func (ec *executionContext) childFields_Extension(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "id":
		return ec.fieldContext_Extension_id(ctx, field)
	case "repositoryId":
		return ec.fieldContext_Extension_repositoryId(ctx, field)
	case "packageName":
		return ec.fieldContext_Extension_packageName(ctx, field)
	case "name":
		return ec.fieldContext_Extension_name(ctx, field)
	case "version":
		return ec.fieldContext_Extension_version(ctx, field)
	case "contentType":
		return ec.fieldContext_Extension_contentType(ctx, field)
	case "lang":
		return ec.fieldContext_Extension_lang(ctx, field)
	case "iconUrl":
		return ec.fieldContext_Extension_iconUrl(ctx, field)
	case "apkUrl":
		return ec.fieldContext_Extension_apkUrl(ctx, field)
	case "jarUrl":
		return ec.fieldContext_Extension_jarUrl(ctx, field)
	case "jarPath":
		return ec.fieldContext_Extension_jarPath(ctx, field)
	case "installed":
		return ec.fieldContext_Extension_installed(ctx, field)
	case "enabled":
		return ec.fieldContext_Extension_enabled(ctx, field)
	case "discoveredAt":
		return ec.fieldContext_Extension_discoveredAt(ctx, field)
	case "installedAt":
		return ec.fieldContext_Extension_installedAt(ctx, field)
	case "installedVersion":
		return ec.fieldContext_Extension_installedVersion(ctx, field)
	case "needsUpdate":
		return ec.fieldContext_Extension_needsUpdate(ctx, field)
	case "isNsfw":
		return ec.fieldContext_Extension_isNsfw(ctx, field)
	case "displayName":
		return ec.fieldContext_Extension_displayName(ctx, field)
	case "supportsLatest":
		return ec.fieldContext_Extension_supportsLatest(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type Extension", field.Name)
}

func (ec *executionContext) childFields_Folder(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "id":
		return ec.fieldContext_Folder_id(ctx, field)
	case "name":
		return ec.fieldContext_Folder_name(ctx, field)
	case "kind":
		return ec.fieldContext_Folder_kind(ctx, field)
	case "systemKey":
		return ec.fieldContext_Folder_systemKey(ctx, field)
	case "parentFolderId":
		return ec.fieldContext_Folder_parentFolderId(ctx, field)
	case "sortOrder":
		return ec.fieldContext_Folder_sortOrder(ctx, field)
	case "includeInUpdate":
		return ec.fieldContext_Folder_includeInUpdate(ctx, field)
	case "includeInDownload":
		return ec.fieldContext_Folder_includeInDownload(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type Folder", field.Name)
}

func (ec *executionContext) childFields_LibraryUpdateStatus(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "running":
		return ec.fieldContext_LibraryUpdateStatus_running(ctx, field)
	case "total":
		return ec.fieldContext_LibraryUpdateStatus_total(ctx, field)
	case "done":
		return ec.fieldContext_LibraryUpdateStatus_done(ctx, field)
	case "currentTitle":
		return ec.fieldContext_LibraryUpdateStatus_currentTitle(ctx, field)
	case "newChapterCount":
		return ec.fieldContext_LibraryUpdateStatus_newChapterCount(ctx, field)
	case "failedTitles":
		return ec.fieldContext_LibraryUpdateStatus_failedTitles(ctx, field)
	case "startedAt":
		return ec.fieldContext_LibraryUpdateStatus_startedAt(ctx, field)
	case "finishedAt":
		return ec.fieldContext_LibraryUpdateStatus_finishedAt(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type LibraryUpdateStatus", field.Name)
}

func (ec *executionContext) childFields_Media(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "id":
		return ec.fieldContext_Media_id(ctx, field)
	case "extensionId":
		return ec.fieldContext_Media_extensionId(ctx, field)
	case "extensionName":
		return ec.fieldContext_Media_extensionName(ctx, field)
	case "sourceName":
		return ec.fieldContext_Media_sourceName(ctx, field)
	case "externalId":
		return ec.fieldContext_Media_externalId(ctx, field)
	case "contentType":
		return ec.fieldContext_Media_contentType(ctx, field)
	case "title":
		return ec.fieldContext_Media_title(ctx, field)
	case "thumbnailUrl":
		return ec.fieldContext_Media_thumbnailUrl(ctx, field)
	case "description":
		return ec.fieldContext_Media_description(ctx, field)
	case "status":
		return ec.fieldContext_Media_status(ctx, field)
	case "author":
		return ec.fieldContext_Media_author(ctx, field)
	case "artist":
		return ec.fieldContext_Media_artist(ctx, field)
	case "detailsFetchedAt":
		return ec.fieldContext_Media_detailsFetchedAt(ctx, field)
	case "extensionRemovedAt":
		return ec.fieldContext_Media_extensionRemovedAt(ctx, field)
	case "addedAt":
		return ec.fieldContext_Media_addedAt(ctx, field)
	case "lastViewedAt":
		return ec.fieldContext_Media_lastViewedAt(ctx, field)
	case "inLibrary":
		return ec.fieldContext_Media_inLibrary(ctx, field)
	case "chapters":
		return ec.fieldContext_Media_chapters(ctx, field)
	case "chapterCount":
		return ec.fieldContext_Media_chapterCount(ctx, field)
	case "unreadCount":
		return ec.fieldContext_Media_unreadCount(ctx, field)
	case "downloadedCount":
		return ec.fieldContext_Media_downloadedCount(ctx, field)
	case "nextUnreadChapter":
		return ec.fieldContext_Media_nextUnreadChapter(ctx, field)
	case "latestChapter":
		return ec.fieldContext_Media_latestChapter(ctx, field)
	case "readingProgress":
		return ec.fieldContext_Media_readingProgress(ctx, field)
	case "tags":
		return ec.fieldContext_Media_tags(ctx, field)
	case "genres":
		return ec.fieldContext_Media_genres(ctx, field)
	case "folders":
		return ec.fieldContext_Media_folders(ctx, field)
	case "trackLinks":
		return ec.fieldContext_Media_trackLinks(ctx, field)
	case "metadata":
		return ec.fieldContext_Media_metadata(ctx, field)
	case "source":
		return ec.fieldContext_Media_source(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type Media", field.Name)
}

func (ec *executionContext) childFields_MediaPage(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "items":
		return ec.fieldContext_MediaPage_items(ctx, field)
	case "total":
		return ec.fieldContext_MediaPage_total(ctx, field)
	case "hasMore":
		return ec.fieldContext_MediaPage_hasMore(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type MediaPage", field.Name)
}

func (ec *executionContext) childFields_MetadataCandidate(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "provider":
		return ec.fieldContext_MetadataCandidate_provider(ctx, field)
	case "providerId":
		return ec.fieldContext_MetadataCandidate_providerId(ctx, field)
	case "title":
		return ec.fieldContext_MetadataCandidate_title(ctx, field)
	case "url":
		return ec.fieldContext_MetadataCandidate_url(ctx, field)
	case "coverUrl":
		return ec.fieldContext_MetadataCandidate_coverUrl(ctx, field)
	case "description":
		return ec.fieldContext_MetadataCandidate_description(ctx, field)
	case "status":
		return ec.fieldContext_MetadataCandidate_status(ctx, field)
	case "genres":
		return ec.fieldContext_MetadataCandidate_genres(ctx, field)
	case "startYear":
		return ec.fieldContext_MetadataCandidate_startYear(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type MetadataCandidate", field.Name)
}

func (ec *executionContext) childFields_MetadataMatch(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "provider":
		return ec.fieldContext_MetadataMatch_provider(ctx, field)
	case "providerId":
		return ec.fieldContext_MetadataMatch_providerId(ctx, field)
	case "url":
		return ec.fieldContext_MetadataMatch_url(ctx, field)
	case "confidence":
		return ec.fieldContext_MetadataMatch_confidence(ctx, field)
	case "locked":
		return ec.fieldContext_MetadataMatch_locked(ctx, field)
	case "matchedAt":
		return ec.fieldContext_MetadataMatch_matchedAt(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type MetadataMatch", field.Name)
}

func (ec *executionContext) childFields_ReadingProgress(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "id":
		return ec.fieldContext_ReadingProgress_id(ctx, field)
	case "mediaId":
		return ec.fieldContext_ReadingProgress_mediaId(ctx, field)
	case "chapterId":
		return ec.fieldContext_ReadingProgress_chapterId(ctx, field)
	case "progress":
		return ec.fieldContext_ReadingProgress_progress(ctx, field)
	case "completed":
		return ec.fieldContext_ReadingProgress_completed(ctx, field)
	case "positionSeconds":
		return ec.fieldContext_ReadingProgress_positionSeconds(ctx, field)
	case "durationSeconds":
		return ec.fieldContext_ReadingProgress_durationSeconds(ctx, field)
	case "updatedAt":
		return ec.fieldContext_ReadingProgress_updatedAt(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type ReadingProgress", field.Name)
}

func (ec *executionContext) childFields_RecentChapter(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "chapter":
		return ec.fieldContext_RecentChapter_chapter(ctx, field)
	case "media":
		return ec.fieldContext_RecentChapter_media(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type RecentChapter", field.Name)
}

func (ec *executionContext) childFields_Repository(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "id":
		return ec.fieldContext_Repository_id(ctx, field)
	case "indexUrl":
		return ec.fieldContext_Repository_indexUrl(ctx, field)
	case "name":
		return ec.fieldContext_Repository_name(ctx, field)
	case "contentType":
		return ec.fieldContext_Repository_contentType(ctx, field)
	case "addedAt":
		return ec.fieldContext_Repository_addedAt(ctx, field)
	case "lastSyncedAt":
		return ec.fieldContext_Repository_lastSyncedAt(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type Repository", field.Name)
}

func (ec *executionContext) childFields_SearchResponse(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "results":
		return ec.fieldContext_SearchResponse_results(ctx, field)
	case "hasNextPage":
		return ec.fieldContext_SearchResponse_hasNextPage(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type SearchResponse", field.Name)
}

func (ec *executionContext) childFields_SkipMarker(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "type":
		return ec.fieldContext_SkipMarker_type(ctx, field)
	case "name":
		return ec.fieldContext_SkipMarker_name(ctx, field)
	case "startMs":
		return ec.fieldContext_SkipMarker_startMs(ctx, field)
	case "endMs":
		return ec.fieldContext_SkipMarker_endMs(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type SkipMarker", field.Name)
}

func (ec *executionContext) childFields_StorageInfo(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "usedBytes":
		return ec.fieldContext_StorageInfo_usedBytes(ctx, field)
	case "totalBytes":
		return ec.fieldContext_StorageInfo_totalBytes(ctx, field)
	case "freeBytes":
		return ec.fieldContext_StorageInfo_freeBytes(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type StorageInfo", field.Name)
}

func (ec *executionContext) childFields_SubtitleTrack(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "lang":
		return ec.fieldContext_SubtitleTrack_lang(ctx, field)
	case "url":
		return ec.fieldContext_SubtitleTrack_url(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type SubtitleTrack", field.Name)
}

func (ec *executionContext) childFields_TrackLink(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "id":
		return ec.fieldContext_TrackLink_id(ctx, field)
	case "mediaId":
		return ec.fieldContext_TrackLink_mediaId(ctx, field)
	case "trackerKey":
		return ec.fieldContext_TrackLink_trackerKey(ctx, field)
	case "remoteId":
		return ec.fieldContext_TrackLink_remoteId(ctx, field)
	case "title":
		return ec.fieldContext_TrackLink_title(ctx, field)
	case "url":
		return ec.fieldContext_TrackLink_url(ctx, field)
	case "status":
		return ec.fieldContext_TrackLink_status(ctx, field)
	case "statusName":
		return ec.fieldContext_TrackLink_statusName(ctx, field)
	case "lastChapterRead":
		return ec.fieldContext_TrackLink_lastChapterRead(ctx, field)
	case "totalChapters":
		return ec.fieldContext_TrackLink_totalChapters(ctx, field)
	case "score":
		return ec.fieldContext_TrackLink_score(ctx, field)
	case "startedAt":
		return ec.fieldContext_TrackLink_startedAt(ctx, field)
	case "finishedAt":
		return ec.fieldContext_TrackLink_finishedAt(ctx, field)
	case "private":
		return ec.fieldContext_TrackLink_private(ctx, field)
	case "lastSyncedAt":
		return ec.fieldContext_TrackLink_lastSyncedAt(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type TrackLink", field.Name)
}

func (ec *executionContext) childFields_TrackSearchResult(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "remoteId":
		return ec.fieldContext_TrackSearchResult_remoteId(ctx, field)
	case "title":
		return ec.fieldContext_TrackSearchResult_title(ctx, field)
	case "url":
		return ec.fieldContext_TrackSearchResult_url(ctx, field)
	case "coverUrl":
		return ec.fieldContext_TrackSearchResult_coverUrl(ctx, field)
	case "summary":
		return ec.fieldContext_TrackSearchResult_summary(ctx, field)
	case "totalChapters":
		return ec.fieldContext_TrackSearchResult_totalChapters(ctx, field)
	case "publishingStatus":
		return ec.fieldContext_TrackSearchResult_publishingStatus(ctx, field)
	case "mediaType":
		return ec.fieldContext_TrackSearchResult_mediaType(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type TrackSearchResult", field.Name)
}

func (ec *executionContext) childFields_TrackStatus(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "value":
		return ec.fieldContext_TrackStatus_value(ctx, field)
	case "name":
		return ec.fieldContext_TrackStatus_name(ctx, field)
	case "animeName":
		return ec.fieldContext_TrackStatus_animeName(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type TrackStatus", field.Name)
}

func (ec *executionContext) childFields_Tracker(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "key":
		return ec.fieldContext_Tracker_key(ctx, field)
	case "name":
		return ec.fieldContext_Tracker_name(ctx, field)
	case "configured":
		return ec.fieldContext_Tracker_configured(ctx, field)
	case "isLoggedIn":
		return ec.fieldContext_Tracker_isLoggedIn(ctx, field)
	case "authUrl":
		return ec.fieldContext_Tracker_authUrl(ctx, field)
	case "username":
		return ec.fieldContext_Tracker_username(ctx, field)
	case "scoreOptions":
		return ec.fieldContext_Tracker_scoreOptions(ctx, field)
	case "statusOptions":
		return ec.fieldContext_Tracker_statusOptions(ctx, field)
	case "iconUrl":
		return ec.fieldContext_Tracker_iconUrl(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type Tracker", field.Name)
}

func (ec *executionContext) childFields_VideoSource(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "label":
		return ec.fieldContext_VideoSource_label(ctx, field)
	case "resolution":
		return ec.fieldContext_VideoSource_resolution(ctx, field)
	case "preferred":
		return ec.fieldContext_VideoSource_preferred(ctx, field)
	case "kind":
		return ec.fieldContext_VideoSource_kind(ctx, field)
	case "server":
		return ec.fieldContext_VideoSource_server(ctx, field)
	case "url":
		return ec.fieldContext_VideoSource_url(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type VideoSource", field.Name)
}

func (ec *executionContext) childFields_VideoStream(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "url":
		return ec.fieldContext_VideoStream_url(ctx, field)
	case "sources":
		return ec.fieldContext_VideoStream_sources(ctx, field)
	case "subtitles":
		return ec.fieldContext_VideoStream_subtitles(ctx, field)
	case "audioTracks":
		return ec.fieldContext_VideoStream_audioTracks(ctx, field)
	case "skipMarkers":
		return ec.fieldContext_VideoStream_skipMarkers(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type VideoStream", field.Name)
}

func (ec *executionContext) childFields___Directive(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "name":
		return ec.fieldContext___Directive_name(ctx, field)
	case "description":
		return ec.fieldContext___Directive_description(ctx, field)
	case "isRepeatable":
		return ec.fieldContext___Directive_isRepeatable(ctx, field)
	case "locations":
		return ec.fieldContext___Directive_locations(ctx, field)
	case "args":
		return ec.fieldContext___Directive_args(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type __Directive", field.Name)
}

func (ec *executionContext) childFields___EnumValue(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "name":
		return ec.fieldContext___EnumValue_name(ctx, field)
	case "description":
		return ec.fieldContext___EnumValue_description(ctx, field)
	case "isDeprecated":
		return ec.fieldContext___EnumValue_isDeprecated(ctx, field)
	case "deprecationReason":
		return ec.fieldContext___EnumValue_deprecationReason(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type __EnumValue", field.Name)
}

func (ec *executionContext) childFields___Field(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "name":
		return ec.fieldContext___Field_name(ctx, field)
	case "description":
		return ec.fieldContext___Field_description(ctx, field)
	case "args":
		return ec.fieldContext___Field_args(ctx, field)
	case "type":
		return ec.fieldContext___Field_type(ctx, field)
	case "isDeprecated":
		return ec.fieldContext___Field_isDeprecated(ctx, field)
	case "deprecationReason":
		return ec.fieldContext___Field_deprecationReason(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type __Field", field.Name)
}

func (ec *executionContext) childFields___InputValue(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "name":
		return ec.fieldContext___InputValue_name(ctx, field)
	case "description":
		return ec.fieldContext___InputValue_description(ctx, field)
	case "type":
		return ec.fieldContext___InputValue_type(ctx, field)
	case "defaultValue":
		return ec.fieldContext___InputValue_defaultValue(ctx, field)
	case "isDeprecated":
		return ec.fieldContext___InputValue_isDeprecated(ctx, field)
	case "deprecationReason":
		return ec.fieldContext___InputValue_deprecationReason(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type __InputValue", field.Name)
}

func (ec *executionContext) childFields___Schema(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "description":
		return ec.fieldContext___Schema_description(ctx, field)
	case "types":
		return ec.fieldContext___Schema_types(ctx, field)
	case "queryType":
		return ec.fieldContext___Schema_queryType(ctx, field)
	case "mutationType":
		return ec.fieldContext___Schema_mutationType(ctx, field)
	case "subscriptionType":
		return ec.fieldContext___Schema_subscriptionType(ctx, field)
	case "directives":
		return ec.fieldContext___Schema_directives(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type __Schema", field.Name)
}

func (ec *executionContext) childFields___Type(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
	switch field.Name {
	case "kind":
		return ec.fieldContext___Type_kind(ctx, field)
	case "name":
		return ec.fieldContext___Type_name(ctx, field)
	case "description":
		return ec.fieldContext___Type_description(ctx, field)
	case "specifiedByURL":
		return ec.fieldContext___Type_specifiedByURL(ctx, field)
	case "fields":
		return ec.fieldContext___Type_fields(ctx, field)
	case "interfaces":
		return ec.fieldContext___Type_interfaces(ctx, field)
	case "possibleTypes":
		return ec.fieldContext___Type_possibleTypes(ctx, field)
	case "enumValues":
		return ec.fieldContext___Type_enumValues(ctx, field)
	case "inputFields":
		return ec.fieldContext___Type_inputFields(ctx, field)
	case "ofType":
		return ec.fieldContext___Type_ofType(ctx, field)
	case "isOneOf":
		return ec.fieldContext___Type_isOneOf(ctx, field)
	}
	return nil, fmt.Errorf("no field named %q was found under type __Type", field.Name)
}

func (ec *executionContext) field_Mutation_addMediaToFolder_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "folderId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["folderId"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_addRepository_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "indexUrl",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["indexUrl"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "name",
		func(ctx context.Context, v any) (*string, error) {
			return ec.unmarshalOString2ᚖstring(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["name"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_applyMetadataMatch_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "providerId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["providerId"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "provider",
		func(ctx context.Context, v any) (*string, error) {
			return ec.unmarshalOString2ᚖstring(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["provider"] = arg2
	return args, nil
}

func (ec *executionContext) field_Mutation_bindTrack_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "trackerKey",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["trackerKey"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "remoteId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["remoteId"] = arg2
	return args, nil
}

func (ec *executionContext) field_Mutation_clearDownloads_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "status",
		func(ctx context.Context, v any) ([]model.DownloadStatus, error) {
			return ec.unmarshalODownloadStatus2ᚕtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloadStatusᚄ(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["status"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_createFolder_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "name",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["name"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "parentFolderId",
		func(ctx context.Context, v any) (*string, error) {
			return ec.unmarshalOID2ᚖstring(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["parentFolderId"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_deleteDownload_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "chapterIds",
		func(ctx context.Context, v any) ([]string, error) {
			return ec.unmarshalNID2ᚕstringᚄ(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["chapterIds"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_deleteFolder_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "folderId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["folderId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_deleteRepository_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "repositoryId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["repositoryId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_dequeueDownload_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "chapterId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["chapterId"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_enqueueDownload_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "chapterIds",
		func(ctx context.Context, v any) ([]string, error) {
			return ec.unmarshalNID2ᚕstringᚄ(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["chapterIds"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_installExtension_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "packageName",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["packageName"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_installExternalExtension_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "url",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["url"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_markChapterRead_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "chapterId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["chapterId"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_markChaptersRead_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "chapterIds",
		func(ctx context.Context, v any) ([]string, error) {
			return ec.unmarshalNID2ᚕstringᚄ(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["chapterIds"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "read",
		func(ctx context.Context, v any) (bool, error) {
			return ec.unmarshalNBoolean2bool(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["read"] = arg2
	return args, nil
}

func (ec *executionContext) field_Mutation_migrateMedia_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "fromMediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["fromMediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "toExtensionId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["toExtensionId"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "toExternalId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["toExternalId"] = arg2
	return args, nil
}

func (ec *executionContext) field_Mutation_pullTracker_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_refreshFolder_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "folderId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["folderId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_refreshMetadataMatch_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_refreshMetadata_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "syncChapters",
		func(ctx context.Context, v any) (*bool, error) {
			return ec.unmarshalOBoolean2ᚖbool(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["syncChapters"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_removeMediaFromFolder_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "folderId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["folderId"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_renameFolder_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "folderId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["folderId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "name",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["name"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_renameRepository_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "repositoryId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["repositoryId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "name",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["name"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_reorderDownload_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "chapterId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["chapterId"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "position",
		func(ctx context.Context, v any) (int32, error) {
			return ec.unmarshalNInt2int32(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["position"] = arg2
	return args, nil
}

func (ec *executionContext) field_Mutation_reorderFolder_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "folderId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["folderId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "sortOrder",
		func(ctx context.Context, v any) (int32, error) {
			return ec.unmarshalNInt2int32(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["sortOrder"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_resyncTrack_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "linkId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["linkId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_retryDownload_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "chapterId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["chapterId"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_setInLibrary_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "inLibrary",
		func(ctx context.Context, v any) (bool, error) {
			return ec.unmarshalNBoolean2bool(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["inLibrary"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_setMediaCover_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "url",
		func(ctx context.Context, v any) (*string, error) {
			return ec.unmarshalOString2ᚖstring(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["url"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_startLibraryUpdate_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "folderId",
		func(ctx context.Context, v any) (*string, error) {
			return ec.unmarshalOID2ᚖstring(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["folderId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_syncChapters_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_trackerLogin_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "trackerKey",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["trackerKey"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "token",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["token"] = arg1
	return args, nil
}

func (ec *executionContext) field_Mutation_trackerLogout_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "trackerKey",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["trackerKey"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_unbindTrack_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "linkId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["linkId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_uninstallExtension_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "packageName",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["packageName"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_unlinkMetadata_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_updateExtension_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "packageName",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["packageName"] = arg0
	return args, nil
}

func (ec *executionContext) field_Mutation_updateFolderFlags_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "folderId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["folderId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "includeInUpdate",
		func(ctx context.Context, v any) (*bool, error) {
			return ec.unmarshalOBoolean2ᚖbool(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["includeInUpdate"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "includeInDownload",
		func(ctx context.Context, v any) (*bool, error) {
			return ec.unmarshalOBoolean2ᚖbool(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["includeInDownload"] = arg2
	return args, nil
}

func (ec *executionContext) field_Mutation_updateReadingProgress_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "chapterId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["chapterId"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "progress",
		func(ctx context.Context, v any) (float64, error) {
			return ec.unmarshalNFloat2float64(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["progress"] = arg2
	arg3, err := graphql.ProcessArgField(ctx, rawArgs, "completed",
		func(ctx context.Context, v any) (*bool, error) {
			return ec.unmarshalOBoolean2ᚖbool(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["completed"] = arg3
	arg4, err := graphql.ProcessArgField(ctx, rawArgs, "positionSeconds",
		func(ctx context.Context, v any) (*float64, error) {
			return ec.unmarshalOFloat2ᚖfloat64(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["positionSeconds"] = arg4
	arg5, err := graphql.ProcessArgField(ctx, rawArgs, "durationSeconds",
		func(ctx context.Context, v any) (*float64, error) {
			return ec.unmarshalOFloat2ᚖfloat64(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["durationSeconds"] = arg5
	return args, nil
}

func (ec *executionContext) field_Mutation_updateTrack_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "linkId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["linkId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "status",
		func(ctx context.Context, v any) (*int32, error) {
			return ec.unmarshalOInt2ᚖint32(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["status"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "score",
		func(ctx context.Context, v any) (*float64, error) {
			return ec.unmarshalOFloat2ᚖfloat64(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["score"] = arg2
	arg3, err := graphql.ProcessArgField(ctx, rawArgs, "lastChapterRead",
		func(ctx context.Context, v any) (*float64, error) {
			return ec.unmarshalOFloat2ᚖfloat64(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["lastChapterRead"] = arg3
	return args, nil
}

func (ec *executionContext) field_Query___type_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "name",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["name"] = arg0
	return args, nil
}

func (ec *executionContext) field_Query_availableExtensions_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "repositoryId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["repositoryId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Query_chapterUpdates_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "since",
		func(ctx context.Context, v any) (*time.Time, error) {
			return ec.unmarshalOTime2ᚖtimeᚐTime(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["since"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "limit",
		func(ctx context.Context, v any) (*int32, error) {
			return ec.unmarshalOInt2ᚖint32(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["limit"] = arg1
	return args, nil
}

func (ec *executionContext) field_Query_chapter_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "id",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["id"] = arg0
	return args, nil
}

func (ec *executionContext) field_Query_downloadStatus_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "chapterId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["chapterId"] = arg1
	return args, nil
}

func (ec *executionContext) field_Query_filterOptions_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "extensionId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["extensionId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Query_folder_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "id",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["id"] = arg0
	return args, nil
}

func (ec *executionContext) field_Query_latestUpdates_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "extensionId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["extensionId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "page",
		func(ctx context.Context, v any) (*int32, error) {
			return ec.unmarshalOInt2ᚖint32(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["page"] = arg1
	return args, nil
}

func (ec *executionContext) field_Query_library_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "filter",
		func(ctx context.Context, v any) (*model.LibraryFilter, error) {
			return ec.unmarshalOLibraryFilter2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐLibraryFilter(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["filter"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "sort",
		func(ctx context.Context, v any) (*model.LibrarySortInput, error) {
			return ec.unmarshalOLibrarySortInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐLibrarySortInput(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["sort"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "limit",
		func(ctx context.Context, v any) (*int32, error) {
			return ec.unmarshalOInt2ᚖint32(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["limit"] = arg2
	arg3, err := graphql.ProcessArgField(ctx, rawArgs, "offset",
		func(ctx context.Context, v any) (*int32, error) {
			return ec.unmarshalOInt2ᚖint32(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["offset"] = arg3
	return args, nil
}

func (ec *executionContext) field_Query_mediaInFolder_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "folderId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["folderId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Query_media_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "id",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["id"] = arg0
	return args, nil
}

func (ec *executionContext) field_Query_popularManga_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "extensionId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["extensionId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "page",
		func(ctx context.Context, v any) (*int32, error) {
			return ec.unmarshalOInt2ᚖint32(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["page"] = arg1
	return args, nil
}

func (ec *executionContext) field_Query_readingProgress_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "mediaId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["mediaId"] = arg0
	return args, nil
}

func (ec *executionContext) field_Query_recentChapters_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "since",
		func(ctx context.Context, v any) (*time.Time, error) {
			return ec.unmarshalOTime2ᚖtimeᚐTime(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["since"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "limit",
		func(ctx context.Context, v any) (*int32, error) {
			return ec.unmarshalOInt2ᚖint32(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["limit"] = arg1
	return args, nil
}

func (ec *executionContext) field_Query_resolveMedia_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "extensionId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["extensionId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "externalId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["externalId"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "syncChapters",
		func(ctx context.Context, v any) (*bool, error) {
			return ec.unmarshalOBoolean2ᚖbool(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["syncChapters"] = arg2
	return args, nil
}

func (ec *executionContext) field_Query_searchMetadata_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "query",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["query"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "contentType",
		func(ctx context.Context, v any) (model.ContentType, error) {
			return ec.unmarshalNContentType2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐContentType(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["contentType"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "provider",
		func(ctx context.Context, v any) (*string, error) {
			return ec.unmarshalOString2ᚖstring(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["provider"] = arg2
	return args, nil
}

func (ec *executionContext) field_Query_search_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "extensionId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["extensionId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "query",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["query"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "page",
		func(ctx context.Context, v any) (*int32, error) {
			return ec.unmarshalOInt2ᚖint32(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["page"] = arg2
	arg3, err := graphql.ProcessArgField(ctx, rawArgs, "filters",
		func(ctx context.Context, v any) ([]*model.FilterInput, error) {
			return ec.unmarshalOFilterInput2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFilterInputᚄ(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["filters"] = arg3
	return args, nil
}

func (ec *executionContext) field_Query_skipTimestamps_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "chapterId",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNID2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["chapterId"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "episodeLengthMs",
		func(ctx context.Context, v any) (*int32, error) {
			return ec.unmarshalOInt2ᚖint32(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["episodeLengthMs"] = arg1
	return args, nil
}

func (ec *executionContext) field_Query_trackSearch_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "trackerKey",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["trackerKey"] = arg0
	arg1, err := graphql.ProcessArgField(ctx, rawArgs, "query",
		func(ctx context.Context, v any) (string, error) {
			return ec.unmarshalNString2string(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["query"] = arg1
	arg2, err := graphql.ProcessArgField(ctx, rawArgs, "contentType",
		func(ctx context.Context, v any) (*model.ContentType, error) {
			return ec.unmarshalOContentType2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐContentType(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["contentType"] = arg2
	return args, nil
}

func (ec *executionContext) field___Directive_args_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "includeDeprecated",
		func(ctx context.Context, v any) (*bool, error) {
			return ec.unmarshalOBoolean2ᚖbool(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["includeDeprecated"] = arg0
	return args, nil
}

func (ec *executionContext) field___Field_args_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "includeDeprecated",
		func(ctx context.Context, v any) (*bool, error) {
			return ec.unmarshalOBoolean2ᚖbool(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["includeDeprecated"] = arg0
	return args, nil
}

func (ec *executionContext) field___Type_enumValues_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "includeDeprecated",
		func(ctx context.Context, v any) (bool, error) {
			return ec.unmarshalOBoolean2bool(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["includeDeprecated"] = arg0
	return args, nil
}

func (ec *executionContext) field___Type_fields_args(ctx context.Context, rawArgs map[string]any) (map[string]any, error) {
	var err error
	args := map[string]any{}
	arg0, err := graphql.ProcessArgField(ctx, rawArgs, "includeDeprecated",
		func(ctx context.Context, v any) (bool, error) {
			return ec.unmarshalOBoolean2bool(ctx, v)
		})
	if err != nil {
		return nil, err
	}
	args["includeDeprecated"] = arg0
	return args, nil
}

func (ec *executionContext) _AboutServer_name(ctx context.Context, field graphql.CollectedField, obj *model.AboutServer) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_AboutServer_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_AboutServer_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("AboutServer", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _AboutServer_version(ctx context.Context, field graphql.CollectedField, obj *model.AboutServer) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_AboutServer_version(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Version, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_AboutServer_version(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("AboutServer", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _AboutServer_buildTime(ctx context.Context, field graphql.CollectedField, obj *model.AboutServer) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_AboutServer_buildTime(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.BuildTime, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_AboutServer_buildTime(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("AboutServer", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _AudioTrack_lang(ctx context.Context, field graphql.CollectedField, obj *model.AudioTrack) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_AudioTrack_lang(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Lang, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_AudioTrack_lang(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("AudioTrack", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _AudioTrack_url(ctx context.Context, field graphql.CollectedField, obj *model.AudioTrack) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_AudioTrack_url(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.URL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_AudioTrack_url(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("AudioTrack", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Chapter_id(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_id(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Chapter_id(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _Chapter_mediaId(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_mediaId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.MediaID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Chapter_mediaId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _Chapter_externalId(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_externalId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ExternalID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Chapter_externalId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Chapter_title(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_title(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Title, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Chapter_title(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Chapter_number(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_number(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Number, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *float64) graphql.Marshaler {
			return ec.marshalOFloat2ᚖfloat64(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Chapter_number(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _Chapter_sourceOrder(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_sourceOrder(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.SourceOrder, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *int32) graphql.Marshaler {
			return ec.marshalOInt2ᚖint32(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Chapter_sourceOrder(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _Chapter_scanlator(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_scanlator(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Scanlator, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Chapter_scanlator(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Chapter_uploadedAt(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_uploadedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.UploadedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Chapter_uploadedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _Chapter_readingProgress(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_readingProgress(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Chapter().ReadingProgress(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.ReadingProgress) graphql.Marshaler {
			return ec.marshalOReadingProgress2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐReadingProgress(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Chapter_readingProgress(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Chapter",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_ReadingProgress(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Chapter_completed(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_completed(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Chapter().Completed(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Chapter_completed(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, true, true, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Chapter_downloaded(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_downloaded(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Chapter().Downloaded(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Chapter_downloaded(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, true, true, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Chapter_download(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_download(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Chapter().Download(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Download) graphql.Marshaler {
			return ec.marshalODownload2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownload(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Chapter_download(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Chapter",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Download(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Chapter_pages(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_pages(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Chapter().Pages(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []string) graphql.Marshaler {
			return ec.marshalOString2ᚕstringᚄ(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Chapter_pages(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, true, true, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Chapter_pageCount(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_pageCount(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Chapter().PageCount(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *int32) graphql.Marshaler {
			return ec.marshalOInt2ᚖint32(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Chapter_pageCount(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, true, true, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _Chapter_videoUrl(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_videoUrl(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Chapter().VideoURL(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Chapter_videoUrl(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Chapter", field, true, true, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Chapter_videoStream(ctx context.Context, field graphql.CollectedField, obj *model.Chapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Chapter_videoStream(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Chapter().VideoStream(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.VideoStream) graphql.Marshaler {
			return ec.marshalOVideoStream2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐVideoStream(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Chapter_videoStream(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Chapter",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_VideoStream(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _CheckBoxFilter_name(ctx context.Context, field graphql.CollectedField, obj *model.CheckBoxFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_CheckBoxFilter_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_CheckBoxFilter_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("CheckBoxFilter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _CheckBoxFilter_state(ctx context.Context, field graphql.CollectedField, obj *model.CheckBoxFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_CheckBoxFilter_state(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.State, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_CheckBoxFilter_state(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("CheckBoxFilter", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Download_id(ctx context.Context, field graphql.CollectedField, obj *model.Download) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Download_id(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Download_id(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Download", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _Download_mediaId(ctx context.Context, field graphql.CollectedField, obj *model.Download) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Download_mediaId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.MediaID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Download_mediaId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Download", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _Download_chapterId(ctx context.Context, field graphql.CollectedField, obj *model.Download) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Download_chapterId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ChapterID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Download_chapterId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Download", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _Download_chapter(ctx context.Context, field graphql.CollectedField, obj *model.Download) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Download_chapter(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Download().Chapter(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Chapter) graphql.Marshaler {
			return ec.marshalNChapter2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐChapter(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Download_chapter(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Download",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Chapter(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Download_status(ctx context.Context, field graphql.CollectedField, obj *model.Download) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Download_status(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Status, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v model.DownloadStatus) graphql.Marshaler {
			return ec.marshalNDownloadStatus2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloadStatus(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Download_status(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Download", field, false, false, errors.New("field of type DownloadStatus does not have child fields"))
}

func (ec *executionContext) _Download_progress(ctx context.Context, field graphql.CollectedField, obj *model.Download) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Download_progress(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Progress, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v float64) graphql.Marshaler {
			return ec.marshalNFloat2float64(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Download_progress(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Download", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _Download_downloadedBytes(ctx context.Context, field graphql.CollectedField, obj *model.Download) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Download_downloadedBytes(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.DownloadedBytes, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *float64) graphql.Marshaler {
			return ec.marshalOFloat2ᚖfloat64(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Download_downloadedBytes(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Download", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _Download_bytesPerSec(ctx context.Context, field graphql.CollectedField, obj *model.Download) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Download_bytesPerSec(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.BytesPerSec, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *float64) graphql.Marshaler {
			return ec.marshalOFloat2ᚖfloat64(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Download_bytesPerSec(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Download", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _Download_finalSizeBytes(ctx context.Context, field graphql.CollectedField, obj *model.Download) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Download_finalSizeBytes(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.FinalSizeBytes, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *float64) graphql.Marshaler {
			return ec.marshalOFloat2ᚖfloat64(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Download_finalSizeBytes(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Download", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _Download_error(ctx context.Context, field graphql.CollectedField, obj *model.Download) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Download_error(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Error, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Download_error(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Download", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Download_createdAt(ctx context.Context, field graphql.CollectedField, obj *model.Download) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Download_createdAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.CreatedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v time.Time) graphql.Marshaler {
			return ec.marshalNTime2timeᚐTime(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Download_createdAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Download", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _Download_completedAt(ctx context.Context, field graphql.CollectedField, obj *model.Download) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Download_completedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.CompletedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Download_completedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Download", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _DownloaderStatus_isRunning(ctx context.Context, field graphql.CollectedField, obj *model.DownloaderStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_DownloaderStatus_isRunning(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IsRunning, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_DownloaderStatus_isRunning(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("DownloaderStatus", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _DownloaderStatus_queuedCount(ctx context.Context, field graphql.CollectedField, obj *model.DownloaderStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_DownloaderStatus_queuedCount(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.QueuedCount, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_DownloaderStatus_queuedCount(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("DownloaderStatus", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _DownloaderStatus_downloadingCount(ctx context.Context, field graphql.CollectedField, obj *model.DownloaderStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_DownloaderStatus_downloadingCount(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.DownloadingCount, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_DownloaderStatus_downloadingCount(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("DownloaderStatus", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _DownloaderStatus_failedCount(ctx context.Context, field graphql.CollectedField, obj *model.DownloaderStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_DownloaderStatus_failedCount(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.FailedCount, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_DownloaderStatus_failedCount(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("DownloaderStatus", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _Extension_id(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_id(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_id(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _Extension_repositoryId(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_repositoryId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.RepositoryID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_repositoryId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _Extension_packageName(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_packageName(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.PackageName, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_packageName(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Extension_name(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Extension_version(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_version(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Version, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_version(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Extension_contentType(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_contentType(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ContentType, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v model.ContentType) graphql.Marshaler {
			return ec.marshalNContentType2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐContentType(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_contentType(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type ContentType does not have child fields"))
}

func (ec *executionContext) _Extension_lang(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_lang(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Lang, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_lang(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Extension_iconUrl(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_iconUrl(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IconURL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Extension_iconUrl(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Extension_apkUrl(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_apkUrl(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ApkURL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Extension_apkUrl(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Extension_jarUrl(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_jarUrl(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.JarURL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Extension_jarUrl(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Extension_jarPath(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_jarPath(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.JarPath, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Extension_jarPath(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Extension_installed(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_installed(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Installed, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_installed(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Extension_enabled(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_enabled(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Enabled, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_enabled(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Extension_discoveredAt(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_discoveredAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.DiscoveredAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v time.Time) graphql.Marshaler {
			return ec.marshalNTime2timeᚐTime(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_discoveredAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _Extension_installedAt(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_installedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.InstalledAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Extension_installedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _Extension_installedVersion(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_installedVersion(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.InstalledVersion, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Extension_installedVersion(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Extension_needsUpdate(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_needsUpdate(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.NeedsUpdate, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *bool) graphql.Marshaler {
			return ec.marshalOBoolean2ᚖbool(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Extension_needsUpdate(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Extension_isNsfw(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_isNsfw(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IsNsfw, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_isNsfw(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Extension_displayName(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_displayName(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.DisplayName, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_displayName(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Extension_supportsLatest(ctx context.Context, field graphql.CollectedField, obj *model.Extension) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Extension_supportsLatest(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.SupportsLatest, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Extension_supportsLatest(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Extension", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Folder_id(ctx context.Context, field graphql.CollectedField, obj *model.Folder) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Folder_id(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Folder_id(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Folder", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _Folder_name(ctx context.Context, field graphql.CollectedField, obj *model.Folder) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Folder_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Folder_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Folder", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Folder_kind(ctx context.Context, field graphql.CollectedField, obj *model.Folder) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Folder_kind(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Kind, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Folder_kind(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Folder", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Folder_systemKey(ctx context.Context, field graphql.CollectedField, obj *model.Folder) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Folder_systemKey(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.SystemKey, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Folder_systemKey(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Folder", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Folder_parentFolderId(ctx context.Context, field graphql.CollectedField, obj *model.Folder) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Folder_parentFolderId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ParentFolderID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOID2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Folder_parentFolderId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Folder", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _Folder_sortOrder(ctx context.Context, field graphql.CollectedField, obj *model.Folder) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Folder_sortOrder(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.SortOrder, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Folder_sortOrder(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Folder", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _Folder_includeInUpdate(ctx context.Context, field graphql.CollectedField, obj *model.Folder) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Folder_includeInUpdate(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IncludeInUpdate, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Folder_includeInUpdate(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Folder", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Folder_includeInDownload(ctx context.Context, field graphql.CollectedField, obj *model.Folder) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Folder_includeInDownload(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IncludeInDownload, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Folder_includeInDownload(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Folder", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _GroupFilter_name(ctx context.Context, field graphql.CollectedField, obj *model.GroupFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_GroupFilter_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_GroupFilter_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("GroupFilter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _GroupFilter_children(ctx context.Context, field graphql.CollectedField, obj *model.GroupFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_GroupFilter_children(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Children, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []model.FilterNode) graphql.Marshaler {
			return ec.marshalNFilterNode2ᚕtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFilterNodeᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_GroupFilter_children(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("GroupFilter", field, false, false, errors.New("field of type FilterNode does not have child fields"))
}

func (ec *executionContext) _HeaderFilter_name(ctx context.Context, field graphql.CollectedField, obj *model.HeaderFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_HeaderFilter_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_HeaderFilter_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("HeaderFilter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _LibraryUpdateStatus_running(ctx context.Context, field graphql.CollectedField, obj *model.LibraryUpdateStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_LibraryUpdateStatus_running(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Running, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_LibraryUpdateStatus_running(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("LibraryUpdateStatus", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _LibraryUpdateStatus_total(ctx context.Context, field graphql.CollectedField, obj *model.LibraryUpdateStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_LibraryUpdateStatus_total(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Total, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_LibraryUpdateStatus_total(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("LibraryUpdateStatus", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _LibraryUpdateStatus_done(ctx context.Context, field graphql.CollectedField, obj *model.LibraryUpdateStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_LibraryUpdateStatus_done(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Done, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_LibraryUpdateStatus_done(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("LibraryUpdateStatus", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _LibraryUpdateStatus_currentTitle(ctx context.Context, field graphql.CollectedField, obj *model.LibraryUpdateStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_LibraryUpdateStatus_currentTitle(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.CurrentTitle, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_LibraryUpdateStatus_currentTitle(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("LibraryUpdateStatus", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _LibraryUpdateStatus_newChapterCount(ctx context.Context, field graphql.CollectedField, obj *model.LibraryUpdateStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_LibraryUpdateStatus_newChapterCount(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.NewChapterCount, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_LibraryUpdateStatus_newChapterCount(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("LibraryUpdateStatus", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _LibraryUpdateStatus_failedTitles(ctx context.Context, field graphql.CollectedField, obj *model.LibraryUpdateStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_LibraryUpdateStatus_failedTitles(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.FailedTitles, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []string) graphql.Marshaler {
			return ec.marshalNString2ᚕstringᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_LibraryUpdateStatus_failedTitles(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("LibraryUpdateStatus", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _LibraryUpdateStatus_startedAt(ctx context.Context, field graphql.CollectedField, obj *model.LibraryUpdateStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_LibraryUpdateStatus_startedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.StartedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_LibraryUpdateStatus_startedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("LibraryUpdateStatus", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _LibraryUpdateStatus_finishedAt(ctx context.Context, field graphql.CollectedField, obj *model.LibraryUpdateStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_LibraryUpdateStatus_finishedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.FinishedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_LibraryUpdateStatus_finishedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("LibraryUpdateStatus", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _Media_id(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_id(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_id(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _Media_extensionId(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_extensionId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ExtensionID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOID2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_extensionId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _Media_extensionName(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_extensionName(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ExtensionName, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_extensionName(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Media_sourceName(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_sourceName(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.SourceName, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_sourceName(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Media_externalId(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_externalId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ExternalID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_externalId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Media_contentType(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_contentType(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ContentType, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v model.ContentType) graphql.Marshaler {
			return ec.marshalNContentType2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐContentType(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_contentType(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type ContentType does not have child fields"))
}

func (ec *executionContext) _Media_title(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_title(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Title, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_title(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Media_thumbnailUrl(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_thumbnailUrl(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ThumbnailURL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_thumbnailUrl(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Media_description(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_description(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Description, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_description(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Media_status(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_status(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Status, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_status(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Media_author(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_author(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Author, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_author(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Media_artist(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_artist(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Artist, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_artist(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Media_detailsFetchedAt(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_detailsFetchedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.DetailsFetchedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_detailsFetchedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _Media_extensionRemovedAt(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_extensionRemovedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ExtensionRemovedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_extensionRemovedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _Media_addedAt(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_addedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.AddedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_addedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _Media_lastViewedAt(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_lastViewedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.LastViewedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_lastViewedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _Media_inLibrary(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_inLibrary(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.InLibrary, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_inLibrary(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Media_chapters(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_chapters(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().Chapters(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Chapter) graphql.Marshaler {
			return ec.marshalNChapter2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐChapterᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_chapters(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Media",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Chapter(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Media_chapterCount(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_chapterCount(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().ChapterCount(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_chapterCount(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, true, true, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _Media_unreadCount(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_unreadCount(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().UnreadCount(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_unreadCount(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, true, true, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _Media_downloadedCount(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_downloadedCount(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().DownloadedCount(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_downloadedCount(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, true, true, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _Media_nextUnreadChapter(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_nextUnreadChapter(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().NextUnreadChapter(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Chapter) graphql.Marshaler {
			return ec.marshalOChapter2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐChapter(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_nextUnreadChapter(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Media",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Chapter(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Media_latestChapter(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_latestChapter(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().LatestChapter(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Chapter) graphql.Marshaler {
			return ec.marshalOChapter2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐChapter(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_latestChapter(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Media",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Chapter(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Media_readingProgress(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_readingProgress(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().ReadingProgress(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.ReadingProgress) graphql.Marshaler {
			return ec.marshalNReadingProgress2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐReadingProgressᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_readingProgress(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Media",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_ReadingProgress(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Media_tags(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_tags(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().Tags(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []string) graphql.Marshaler {
			return ec.marshalNString2ᚕstringᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_tags(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, true, true, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Media_genres(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_genres(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().Genres(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []string) graphql.Marshaler {
			return ec.marshalNString2ᚕstringᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_genres(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Media", field, true, true, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Media_folders(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_folders(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().Folders(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Folder) graphql.Marshaler {
			return ec.marshalNFolder2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFolderᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_folders(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Media",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Folder(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Media_trackLinks(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_trackLinks(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().TrackLinks(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.TrackLink) graphql.Marshaler {
			return ec.marshalNTrackLink2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackLinkᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Media_trackLinks(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Media",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_TrackLink(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Media_metadata(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_metadata(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().Metadata(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.MetadataMatch) graphql.Marshaler {
			return ec.marshalOMetadataMatch2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMetadataMatch(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_metadata(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Media",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_MetadataMatch(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Media_source(ctx context.Context, field graphql.CollectedField, obj *model.Media) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Media_source(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Media().Source(ctx, obj)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Extension) graphql.Marshaler {
			return ec.marshalOExtension2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐExtension(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Media_source(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Media",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Extension(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _MediaPage_items(ctx context.Context, field graphql.CollectedField, obj *model.MediaPage) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MediaPage_items(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Items, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMediaᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MediaPage_items(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "MediaPage",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _MediaPage_total(ctx context.Context, field graphql.CollectedField, obj *model.MediaPage) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MediaPage_total(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Total, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MediaPage_total(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MediaPage", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _MediaPage_hasMore(ctx context.Context, field graphql.CollectedField, obj *model.MediaPage) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MediaPage_hasMore(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.HasMore, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MediaPage_hasMore(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MediaPage", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _MetadataCandidate_provider(ctx context.Context, field graphql.CollectedField, obj *model.MetadataCandidate) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataCandidate_provider(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Provider, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MetadataCandidate_provider(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataCandidate", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _MetadataCandidate_providerId(ctx context.Context, field graphql.CollectedField, obj *model.MetadataCandidate) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataCandidate_providerId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ProviderID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MetadataCandidate_providerId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataCandidate", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _MetadataCandidate_title(ctx context.Context, field graphql.CollectedField, obj *model.MetadataCandidate) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataCandidate_title(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Title, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MetadataCandidate_title(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataCandidate", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _MetadataCandidate_url(ctx context.Context, field graphql.CollectedField, obj *model.MetadataCandidate) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataCandidate_url(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.URL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MetadataCandidate_url(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataCandidate", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _MetadataCandidate_coverUrl(ctx context.Context, field graphql.CollectedField, obj *model.MetadataCandidate) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataCandidate_coverUrl(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.CoverURL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_MetadataCandidate_coverUrl(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataCandidate", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _MetadataCandidate_description(ctx context.Context, field graphql.CollectedField, obj *model.MetadataCandidate) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataCandidate_description(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Description, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_MetadataCandidate_description(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataCandidate", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _MetadataCandidate_status(ctx context.Context, field graphql.CollectedField, obj *model.MetadataCandidate) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataCandidate_status(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Status, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_MetadataCandidate_status(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataCandidate", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _MetadataCandidate_genres(ctx context.Context, field graphql.CollectedField, obj *model.MetadataCandidate) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataCandidate_genres(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Genres, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []string) graphql.Marshaler {
			return ec.marshalNString2ᚕstringᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MetadataCandidate_genres(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataCandidate", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _MetadataCandidate_startYear(ctx context.Context, field graphql.CollectedField, obj *model.MetadataCandidate) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataCandidate_startYear(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.StartYear, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *int32) graphql.Marshaler {
			return ec.marshalOInt2ᚖint32(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_MetadataCandidate_startYear(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataCandidate", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _MetadataMatch_provider(ctx context.Context, field graphql.CollectedField, obj *model.MetadataMatch) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataMatch_provider(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Provider, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MetadataMatch_provider(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataMatch", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _MetadataMatch_providerId(ctx context.Context, field graphql.CollectedField, obj *model.MetadataMatch) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataMatch_providerId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ProviderID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MetadataMatch_providerId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataMatch", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _MetadataMatch_url(ctx context.Context, field graphql.CollectedField, obj *model.MetadataMatch) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataMatch_url(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.URL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MetadataMatch_url(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataMatch", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _MetadataMatch_confidence(ctx context.Context, field graphql.CollectedField, obj *model.MetadataMatch) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataMatch_confidence(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Confidence, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v float64) graphql.Marshaler {
			return ec.marshalNFloat2float64(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MetadataMatch_confidence(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataMatch", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _MetadataMatch_locked(ctx context.Context, field graphql.CollectedField, obj *model.MetadataMatch) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataMatch_locked(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Locked, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MetadataMatch_locked(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataMatch", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _MetadataMatch_matchedAt(ctx context.Context, field graphql.CollectedField, obj *model.MetadataMatch) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_MetadataMatch_matchedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.MatchedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v time.Time) graphql.Marshaler {
			return ec.marshalNTime2timeᚐTime(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_MetadataMatch_matchedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("MetadataMatch", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _Mutation_createFolder(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_createFolder(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().CreateFolder(ctx, fc.Args["name"].(string), fc.Args["parentFolderId"].(*string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Folder) graphql.Marshaler {
			return ec.marshalNFolder2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFolder(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_createFolder(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Folder(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_createFolder_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_renameFolder(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_renameFolder(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().RenameFolder(ctx, fc.Args["folderId"].(string), fc.Args["name"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Folder) graphql.Marshaler {
			return ec.marshalNFolder2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFolder(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_renameFolder(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Folder(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_renameFolder_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_deleteFolder(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_deleteFolder(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().DeleteFolder(ctx, fc.Args["folderId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_deleteFolder(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Boolean does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_deleteFolder_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_addMediaToFolder(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_addMediaToFolder(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().AddMediaToFolder(ctx, fc.Args["mediaId"].(string), fc.Args["folderId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_addMediaToFolder(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Boolean does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_addMediaToFolder_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_removeMediaFromFolder(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_removeMediaFromFolder(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().RemoveMediaFromFolder(ctx, fc.Args["mediaId"].(string), fc.Args["folderId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_removeMediaFromFolder(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Boolean does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_removeMediaFromFolder_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_markChaptersRead(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_markChaptersRead(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().MarkChaptersRead(ctx, fc.Args["mediaId"].(string), fc.Args["chapterIds"].([]string), fc.Args["read"].(bool))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.ReadingProgress) graphql.Marshaler {
			return ec.marshalNReadingProgress2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐReadingProgressᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_markChaptersRead(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_ReadingProgress(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_markChaptersRead_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_dequeueDownload(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_dequeueDownload(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().DequeueDownload(ctx, fc.Args["mediaId"].(string), fc.Args["chapterId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_dequeueDownload(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Boolean does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_dequeueDownload_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_addRepository(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_addRepository(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().AddRepository(ctx, fc.Args["indexUrl"].(string), fc.Args["name"].(*string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Repository) graphql.Marshaler {
			return ec.marshalNRepository2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐRepository(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_addRepository(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Repository(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_addRepository_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_renameRepository(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_renameRepository(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().RenameRepository(ctx, fc.Args["repositoryId"].(string), fc.Args["name"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Repository) graphql.Marshaler {
			return ec.marshalNRepository2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐRepository(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_renameRepository(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Repository(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_renameRepository_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_deleteRepository(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_deleteRepository(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().DeleteRepository(ctx, fc.Args["repositoryId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_deleteRepository(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Boolean does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_deleteRepository_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_installExtension(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_installExtension(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().InstallExtension(ctx, fc.Args["packageName"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Extension) graphql.Marshaler {
			return ec.marshalNExtension2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐExtension(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_installExtension(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Extension(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_installExtension_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_installExternalExtension(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_installExternalExtension(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().InstallExternalExtension(ctx, fc.Args["url"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Extension) graphql.Marshaler {
			return ec.marshalNExtension2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐExtension(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_installExternalExtension(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Extension(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_installExternalExtension_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_uninstallExtension(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_uninstallExtension(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().UninstallExtension(ctx, fc.Args["packageName"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Extension) graphql.Marshaler {
			return ec.marshalNExtension2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐExtension(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_uninstallExtension(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Extension(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_uninstallExtension_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_updateExtension(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_updateExtension(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().UpdateExtension(ctx, fc.Args["packageName"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Extension) graphql.Marshaler {
			return ec.marshalNExtension2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐExtension(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_updateExtension(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Extension(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_updateExtension_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_setInLibrary(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_setInLibrary(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().SetInLibrary(ctx, fc.Args["mediaId"].(string), fc.Args["inLibrary"].(bool))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_setInLibrary(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_setInLibrary_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_migrateMedia(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_migrateMedia(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().MigrateMedia(ctx, fc.Args["fromMediaId"].(string), fc.Args["toExtensionId"].(string), fc.Args["toExternalId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_migrateMedia(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_migrateMedia_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_syncChapters(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_syncChapters(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().SyncChapters(ctx, fc.Args["mediaId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Chapter) graphql.Marshaler {
			return ec.marshalNChapter2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐChapterᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_syncChapters(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Chapter(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_syncChapters_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_updateReadingProgress(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_updateReadingProgress(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().UpdateReadingProgress(ctx, fc.Args["mediaId"].(string), fc.Args["chapterId"].(string), fc.Args["progress"].(float64), fc.Args["completed"].(*bool), fc.Args["positionSeconds"].(*float64), fc.Args["durationSeconds"].(*float64))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.ReadingProgress) graphql.Marshaler {
			return ec.marshalNReadingProgress2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐReadingProgress(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_updateReadingProgress(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_ReadingProgress(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_updateReadingProgress_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_markChapterRead(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_markChapterRead(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().MarkChapterRead(ctx, fc.Args["mediaId"].(string), fc.Args["chapterId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.ReadingProgress) graphql.Marshaler {
			return ec.marshalNReadingProgress2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐReadingProgress(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_markChapterRead(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_ReadingProgress(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_markChapterRead_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_enqueueDownload(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_enqueueDownload(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().EnqueueDownload(ctx, fc.Args["mediaId"].(string), fc.Args["chapterIds"].([]string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Download) graphql.Marshaler {
			return ec.marshalNDownload2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloadᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_enqueueDownload(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Download(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_enqueueDownload_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_retryDownload(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_retryDownload(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().RetryDownload(ctx, fc.Args["mediaId"].(string), fc.Args["chapterId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Download) graphql.Marshaler {
			return ec.marshalNDownload2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownload(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_retryDownload(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Download(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_retryDownload_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_deleteDownload(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_deleteDownload(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().DeleteDownload(ctx, fc.Args["mediaId"].(string), fc.Args["chapterIds"].([]string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_deleteDownload(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Boolean does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_deleteDownload_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_reorderDownload(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_reorderDownload(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().ReorderDownload(ctx, fc.Args["mediaId"].(string), fc.Args["chapterId"].(string), fc.Args["position"].(int32))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_reorderDownload(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Boolean does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_reorderDownload_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_clearDownloads(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_clearDownloads(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().ClearDownloads(ctx, fc.Args["status"].([]model.DownloadStatus))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_clearDownloads(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Boolean does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_clearDownloads_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_startDownloader(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_startDownloader(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Mutation().StartDownloader(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_startDownloader(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Mutation", field, true, true, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Mutation_stopDownloader(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_stopDownloader(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Mutation().StopDownloader(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_stopDownloader(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Mutation", field, true, true, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Mutation_refreshMetadata(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_refreshMetadata(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().RefreshMetadata(ctx, fc.Args["mediaId"].(string), fc.Args["syncChapters"].(*bool))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_refreshMetadata(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_refreshMetadata_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_refreshFolder(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_refreshFolder(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().RefreshFolder(ctx, fc.Args["folderId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMediaᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_refreshFolder(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_refreshFolder_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_reorderFolder(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_reorderFolder(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().ReorderFolder(ctx, fc.Args["folderId"].(string), fc.Args["sortOrder"].(int32))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Folder) graphql.Marshaler {
			return ec.marshalNFolder2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFolder(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_reorderFolder(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Folder(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_reorderFolder_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_updateFolderFlags(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_updateFolderFlags(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().UpdateFolderFlags(ctx, fc.Args["folderId"].(string), fc.Args["includeInUpdate"].(*bool), fc.Args["includeInDownload"].(*bool))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Folder) graphql.Marshaler {
			return ec.marshalNFolder2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFolder(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_updateFolderFlags(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Folder(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_updateFolderFlags_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_clearImageCache(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_clearImageCache(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Mutation().ClearImageCache(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_clearImageCache(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Mutation", field, true, true, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Mutation_startLibraryUpdate(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_startLibraryUpdate(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().StartLibraryUpdate(ctx, fc.Args["folderId"].(*string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_startLibraryUpdate(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Boolean does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_startLibraryUpdate_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_setMediaCover(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_setMediaCover(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().SetMediaCover(ctx, fc.Args["mediaId"].(string), fc.Args["url"].(*string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_setMediaCover(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_setMediaCover_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_rescanLocalMedia(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_rescanLocalMedia(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Mutation().RescanLocalMedia(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMediaᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_rescanLocalMedia(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_trackerLogin(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_trackerLogin(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().TrackerLogin(ctx, fc.Args["trackerKey"].(string), fc.Args["token"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Tracker) graphql.Marshaler {
			return ec.marshalNTracker2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTracker(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_trackerLogin(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Tracker(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_trackerLogin_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_trackerLogout(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_trackerLogout(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().TrackerLogout(ctx, fc.Args["trackerKey"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_trackerLogout(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Boolean does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_trackerLogout_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_bindTrack(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_bindTrack(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().BindTrack(ctx, fc.Args["mediaId"].(string), fc.Args["trackerKey"].(string), fc.Args["remoteId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.TrackLink) graphql.Marshaler {
			return ec.marshalNTrackLink2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackLink(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_bindTrack(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_TrackLink(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_bindTrack_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_updateTrack(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_updateTrack(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().UpdateTrack(ctx, fc.Args["linkId"].(string), fc.Args["status"].(*int32), fc.Args["score"].(*float64), fc.Args["lastChapterRead"].(*float64))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.TrackLink) graphql.Marshaler {
			return ec.marshalNTrackLink2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackLink(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_updateTrack(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_TrackLink(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_updateTrack_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_unbindTrack(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_unbindTrack(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().UnbindTrack(ctx, fc.Args["linkId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_unbindTrack(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Boolean does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_unbindTrack_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_resyncTrack(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_resyncTrack(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().ResyncTrack(ctx, fc.Args["linkId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.TrackLink) graphql.Marshaler {
			return ec.marshalNTrackLink2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackLink(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_resyncTrack(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_TrackLink(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_resyncTrack_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_pullTracker(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_pullTracker(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().PullTracker(ctx, fc.Args["mediaId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.TrackLink) graphql.Marshaler {
			return ec.marshalNTrackLink2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackLinkᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_pullTracker(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_TrackLink(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_pullTracker_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_applyMetadataMatch(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_applyMetadataMatch(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().ApplyMetadataMatch(ctx, fc.Args["mediaId"].(string), fc.Args["providerId"].(string), fc.Args["provider"].(*string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_applyMetadataMatch(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_applyMetadataMatch_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_unlinkMetadata(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_unlinkMetadata(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().UnlinkMetadata(ctx, fc.Args["mediaId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_unlinkMetadata(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type Boolean does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_unlinkMetadata_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Mutation_refreshMetadataMatch(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Mutation_refreshMetadataMatch(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Mutation().RefreshMetadataMatch(ctx, fc.Args["mediaId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Mutation_refreshMetadataMatch(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Mutation",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Mutation_refreshMetadataMatch_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_about(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_about(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Query().About(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.AboutServer) graphql.Marshaler {
			return ec.marshalNAboutServer2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐAboutServer(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_about(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_AboutServer(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Query_folders(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_folders(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Query().Folders(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Folder) graphql.Marshaler {
			return ec.marshalNFolder2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFolderᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_folders(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Folder(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Query_folder(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_folder(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().Folder(ctx, fc.Args["id"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Folder) graphql.Marshaler {
			return ec.marshalOFolder2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFolder(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Query_folder(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Folder(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_folder_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_mediaInFolder(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_mediaInFolder(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().MediaInFolder(ctx, fc.Args["folderId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMediaᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_mediaInFolder(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_mediaInFolder_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_repositories(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_repositories(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Query().Repositories(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Repository) graphql.Marshaler {
			return ec.marshalNRepository2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐRepositoryᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_repositories(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Repository(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Query_availableExtensions(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_availableExtensions(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().AvailableExtensions(ctx, fc.Args["repositoryId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Extension) graphql.Marshaler {
			return ec.marshalNExtension2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐExtensionᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_availableExtensions(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Extension(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_availableExtensions_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_installedExtensions(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_installedExtensions(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Query().InstalledExtensions(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Extension) graphql.Marshaler {
			return ec.marshalNExtension2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐExtensionᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_installedExtensions(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Extension(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Query_library(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_library(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().Library(ctx, fc.Args["filter"].(*model.LibraryFilter), fc.Args["sort"].(*model.LibrarySortInput), fc.Args["limit"].(*int32), fc.Args["offset"].(*int32))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.MediaPage) graphql.Marshaler {
			return ec.marshalNMediaPage2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMediaPage(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_library(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_MediaPage(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_library_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_media(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_media(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().Media(ctx, fc.Args["id"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Media) graphql.Marshaler {
			return ec.marshalOMedia2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Query_media(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_media_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_chapter(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_chapter(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().Chapter(ctx, fc.Args["id"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Chapter) graphql.Marshaler {
			return ec.marshalOChapter2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐChapter(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Query_chapter(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Chapter(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_chapter_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_resolveMedia(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_resolveMedia(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().ResolveMedia(ctx, fc.Args["extensionId"].(string), fc.Args["externalId"].(string), fc.Args["syncChapters"].(*bool))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_resolveMedia(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_resolveMedia_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_readingProgress(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_readingProgress(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().ReadingProgress(ctx, fc.Args["mediaId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.ReadingProgress) graphql.Marshaler {
			return ec.marshalNReadingProgress2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐReadingProgressᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_readingProgress(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_ReadingProgress(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_readingProgress_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_search(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_search(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().Search(ctx, fc.Args["extensionId"].(string), fc.Args["query"].(string), fc.Args["page"].(*int32), fc.Args["filters"].([]*model.FilterInput))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.SearchResponse) graphql.Marshaler {
			return ec.marshalNSearchResponse2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSearchResponse(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_search(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_SearchResponse(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_search_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_filterOptions(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_filterOptions(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().FilterOptions(ctx, fc.Args["extensionId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []model.FilterNode) graphql.Marshaler {
			return ec.marshalNFilterNode2ᚕtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFilterNodeᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_filterOptions(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type FilterNode does not have child fields")
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_filterOptions_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_popularManga(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_popularManga(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().PopularManga(ctx, fc.Args["extensionId"].(string), fc.Args["page"].(*int32))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.SearchResponse) graphql.Marshaler {
			return ec.marshalNSearchResponse2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSearchResponse(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_popularManga(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_SearchResponse(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_popularManga_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_latestUpdates(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_latestUpdates(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().LatestUpdates(ctx, fc.Args["extensionId"].(string), fc.Args["page"].(*int32))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.SearchResponse) graphql.Marshaler {
			return ec.marshalNSearchResponse2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSearchResponse(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_latestUpdates(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_SearchResponse(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_latestUpdates_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_downloadStatus(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_downloadStatus(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().DownloadStatus(ctx, fc.Args["mediaId"].(string), fc.Args["chapterId"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Download) graphql.Marshaler {
			return ec.marshalODownload2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownload(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Query_downloadStatus(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Download(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_downloadStatus_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_downloadQueue(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_downloadQueue(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Query().DownloadQueue(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Download) graphql.Marshaler {
			return ec.marshalNDownload2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloadᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_downloadQueue(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Download(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Query_downloaderStatus(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_downloaderStatus(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Query().DownloaderStatus(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.DownloaderStatus) graphql.Marshaler {
			return ec.marshalNDownloaderStatus2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloaderStatus(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_downloaderStatus(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_DownloaderStatus(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Query_recentChapters(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_recentChapters(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().RecentChapters(ctx, fc.Args["since"].(*time.Time), fc.Args["limit"].(*int32))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.RecentChapter) graphql.Marshaler {
			return ec.marshalNRecentChapter2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐRecentChapterᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_recentChapters(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_RecentChapter(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_recentChapters_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_chapterUpdates(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_chapterUpdates(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().ChapterUpdates(ctx, fc.Args["since"].(*time.Time), fc.Args["limit"].(*int32))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.RecentChapter) graphql.Marshaler {
			return ec.marshalNRecentChapter2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐRecentChapterᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_chapterUpdates(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_RecentChapter(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_chapterUpdates_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_libraryUpdateStatus(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_libraryUpdateStatus(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Query().LibraryUpdateStatus(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.LibraryUpdateStatus) graphql.Marshaler {
			return ec.marshalNLibraryUpdateStatus2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐLibraryUpdateStatus(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_libraryUpdateStatus(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_LibraryUpdateStatus(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Query_storageInfo(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_storageInfo(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Query().StorageInfo(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.StorageInfo) graphql.Marshaler {
			return ec.marshalNStorageInfo2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐStorageInfo(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_storageInfo(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_StorageInfo(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Query_trackers(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_trackers(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.Resolvers.Query().Trackers(ctx)
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Tracker) graphql.Marshaler {
			return ec.marshalNTracker2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackerᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_trackers(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Tracker(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Query_trackSearch(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_trackSearch(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().TrackSearch(ctx, fc.Args["trackerKey"].(string), fc.Args["query"].(string), fc.Args["contentType"].(*model.ContentType))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.TrackSearchResult) graphql.Marshaler {
			return ec.marshalNTrackSearchResult2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackSearchResultᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_trackSearch(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_TrackSearchResult(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_trackSearch_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_searchMetadata(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_searchMetadata(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().SearchMetadata(ctx, fc.Args["query"].(string), fc.Args["contentType"].(model.ContentType), fc.Args["provider"].(*string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.MetadataCandidate) graphql.Marshaler {
			return ec.marshalNMetadataCandidate2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMetadataCandidateᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_searchMetadata(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_MetadataCandidate(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_searchMetadata_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query_skipTimestamps(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query_skipTimestamps(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.Resolvers.Query().SkipTimestamps(ctx, fc.Args["chapterId"].(string), fc.Args["episodeLengthMs"].(*int32))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.SkipMarker) graphql.Marshaler {
			return ec.marshalNSkipMarker2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSkipMarkerᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Query_skipTimestamps(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_SkipMarker(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query_skipTimestamps_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query___type(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query___type(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return ec.IntrospectType(fc.Args["name"].(string))
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *introspection.Type) graphql.Marshaler {
			return ec.marshalO__Type2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐType(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Query___type(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Type(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Query___type_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Query___schema(ctx context.Context, field graphql.CollectedField) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Query___schema(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return ec.IntrospectSchema()
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *introspection.Schema) graphql.Marshaler {
			return ec.marshalO__Schema2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐSchema(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Query___schema(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Query",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Schema(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _ReadingProgress_id(ctx context.Context, field graphql.CollectedField, obj *model.ReadingProgress) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_ReadingProgress_id(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_ReadingProgress_id(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("ReadingProgress", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _ReadingProgress_mediaId(ctx context.Context, field graphql.CollectedField, obj *model.ReadingProgress) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_ReadingProgress_mediaId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.MediaID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_ReadingProgress_mediaId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("ReadingProgress", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _ReadingProgress_chapterId(ctx context.Context, field graphql.CollectedField, obj *model.ReadingProgress) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_ReadingProgress_chapterId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ChapterID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_ReadingProgress_chapterId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("ReadingProgress", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _ReadingProgress_progress(ctx context.Context, field graphql.CollectedField, obj *model.ReadingProgress) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_ReadingProgress_progress(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Progress, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v float64) graphql.Marshaler {
			return ec.marshalNFloat2float64(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_ReadingProgress_progress(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("ReadingProgress", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _ReadingProgress_completed(ctx context.Context, field graphql.CollectedField, obj *model.ReadingProgress) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_ReadingProgress_completed(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Completed, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_ReadingProgress_completed(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("ReadingProgress", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _ReadingProgress_positionSeconds(ctx context.Context, field graphql.CollectedField, obj *model.ReadingProgress) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_ReadingProgress_positionSeconds(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.PositionSeconds, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *float64) graphql.Marshaler {
			return ec.marshalOFloat2ᚖfloat64(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_ReadingProgress_positionSeconds(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("ReadingProgress", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _ReadingProgress_durationSeconds(ctx context.Context, field graphql.CollectedField, obj *model.ReadingProgress) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_ReadingProgress_durationSeconds(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.DurationSeconds, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *float64) graphql.Marshaler {
			return ec.marshalOFloat2ᚖfloat64(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_ReadingProgress_durationSeconds(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("ReadingProgress", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _ReadingProgress_updatedAt(ctx context.Context, field graphql.CollectedField, obj *model.ReadingProgress) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_ReadingProgress_updatedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.UpdatedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v time.Time) graphql.Marshaler {
			return ec.marshalNTime2timeᚐTime(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_ReadingProgress_updatedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("ReadingProgress", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _RecentChapter_chapter(ctx context.Context, field graphql.CollectedField, obj *model.RecentChapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_RecentChapter_chapter(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Chapter, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Chapter) graphql.Marshaler {
			return ec.marshalNChapter2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐChapter(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_RecentChapter_chapter(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "RecentChapter",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Chapter(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _RecentChapter_media(ctx context.Context, field graphql.CollectedField, obj *model.RecentChapter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_RecentChapter_media(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Media, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_RecentChapter_media(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "RecentChapter",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Repository_id(ctx context.Context, field graphql.CollectedField, obj *model.Repository) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Repository_id(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Repository_id(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Repository", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _Repository_indexUrl(ctx context.Context, field graphql.CollectedField, obj *model.Repository) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Repository_indexUrl(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IndexURL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Repository_indexUrl(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Repository", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Repository_name(ctx context.Context, field graphql.CollectedField, obj *model.Repository) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Repository_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Repository_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Repository", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Repository_contentType(ctx context.Context, field graphql.CollectedField, obj *model.Repository) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Repository_contentType(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ContentType, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v model.ContentType) graphql.Marshaler {
			return ec.marshalNContentType2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐContentType(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Repository_contentType(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Repository", field, false, false, errors.New("field of type ContentType does not have child fields"))
}

func (ec *executionContext) _Repository_addedAt(ctx context.Context, field graphql.CollectedField, obj *model.Repository) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Repository_addedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.AddedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v time.Time) graphql.Marshaler {
			return ec.marshalNTime2timeᚐTime(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Repository_addedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Repository", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _Repository_lastSyncedAt(ctx context.Context, field graphql.CollectedField, obj *model.Repository) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Repository_lastSyncedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.LastSyncedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Repository_lastSyncedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Repository", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _SearchResponse_results(ctx context.Context, field graphql.CollectedField, obj *model.SearchResponse) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SearchResponse_results(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Results, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.Media) graphql.Marshaler {
			return ec.marshalNMedia2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMediaᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SearchResponse_results(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "SearchResponse",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_Media(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _SearchResponse_hasNextPage(ctx context.Context, field graphql.CollectedField, obj *model.SearchResponse) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SearchResponse_hasNextPage(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.HasNextPage, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SearchResponse_hasNextPage(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SearchResponse", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _SelectFilter_name(ctx context.Context, field graphql.CollectedField, obj *model.SelectFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SelectFilter_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SelectFilter_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SelectFilter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _SelectFilter_values(ctx context.Context, field graphql.CollectedField, obj *model.SelectFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SelectFilter_values(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Values, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []string) graphql.Marshaler {
			return ec.marshalNString2ᚕstringᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SelectFilter_values(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SelectFilter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _SelectFilter_state(ctx context.Context, field graphql.CollectedField, obj *model.SelectFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SelectFilter_state(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.State, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SelectFilter_state(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SelectFilter", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _SeparatorFilter_name(ctx context.Context, field graphql.CollectedField, obj *model.SeparatorFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SeparatorFilter_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SeparatorFilter_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SeparatorFilter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _SkipMarker_type(ctx context.Context, field graphql.CollectedField, obj *model.SkipMarker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SkipMarker_type(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Type, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SkipMarker_type(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SkipMarker", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _SkipMarker_name(ctx context.Context, field graphql.CollectedField, obj *model.SkipMarker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SkipMarker_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SkipMarker_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SkipMarker", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _SkipMarker_startMs(ctx context.Context, field graphql.CollectedField, obj *model.SkipMarker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SkipMarker_startMs(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.StartMs, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SkipMarker_startMs(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SkipMarker", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _SkipMarker_endMs(ctx context.Context, field graphql.CollectedField, obj *model.SkipMarker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SkipMarker_endMs(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.EndMs, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SkipMarker_endMs(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SkipMarker", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _SortFilter_name(ctx context.Context, field graphql.CollectedField, obj *model.SortFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SortFilter_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SortFilter_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SortFilter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _SortFilter_values(ctx context.Context, field graphql.CollectedField, obj *model.SortFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SortFilter_values(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Values, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []string) graphql.Marshaler {
			return ec.marshalNString2ᚕstringᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SortFilter_values(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SortFilter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _SortFilter_hasState(ctx context.Context, field graphql.CollectedField, obj *model.SortFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SortFilter_hasState(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.HasState, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SortFilter_hasState(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SortFilter", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _SortFilter_index(ctx context.Context, field graphql.CollectedField, obj *model.SortFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SortFilter_index(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Index, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *int32) graphql.Marshaler {
			return ec.marshalOInt2ᚖint32(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_SortFilter_index(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SortFilter", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _SortFilter_ascending(ctx context.Context, field graphql.CollectedField, obj *model.SortFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SortFilter_ascending(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Ascending, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *bool) graphql.Marshaler {
			return ec.marshalOBoolean2ᚖbool(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_SortFilter_ascending(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SortFilter", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _StorageInfo_usedBytes(ctx context.Context, field graphql.CollectedField, obj *model.StorageInfo) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_StorageInfo_usedBytes(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.UsedBytes, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v float64) graphql.Marshaler {
			return ec.marshalNFloat2float64(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_StorageInfo_usedBytes(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("StorageInfo", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _StorageInfo_totalBytes(ctx context.Context, field graphql.CollectedField, obj *model.StorageInfo) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_StorageInfo_totalBytes(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.TotalBytes, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v float64) graphql.Marshaler {
			return ec.marshalNFloat2float64(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_StorageInfo_totalBytes(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("StorageInfo", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _StorageInfo_freeBytes(ctx context.Context, field graphql.CollectedField, obj *model.StorageInfo) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_StorageInfo_freeBytes(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.FreeBytes, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v float64) graphql.Marshaler {
			return ec.marshalNFloat2float64(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_StorageInfo_freeBytes(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("StorageInfo", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _SubtitleTrack_lang(ctx context.Context, field graphql.CollectedField, obj *model.SubtitleTrack) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SubtitleTrack_lang(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Lang, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SubtitleTrack_lang(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SubtitleTrack", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _SubtitleTrack_url(ctx context.Context, field graphql.CollectedField, obj *model.SubtitleTrack) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_SubtitleTrack_url(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.URL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_SubtitleTrack_url(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("SubtitleTrack", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TextFilter_name(ctx context.Context, field graphql.CollectedField, obj *model.TextFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TextFilter_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TextFilter_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TextFilter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TextFilter_state(ctx context.Context, field graphql.CollectedField, obj *model.TextFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TextFilter_state(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.State, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TextFilter_state(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TextFilter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackLink_id(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_id(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackLink_id(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _TrackLink_mediaId(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_mediaId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.MediaID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNID2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackLink_mediaId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type ID does not have child fields"))
}

func (ec *executionContext) _TrackLink_trackerKey(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_trackerKey(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.TrackerKey, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackLink_trackerKey(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackLink_remoteId(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_remoteId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.RemoteID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackLink_remoteId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackLink_title(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_title(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Title, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackLink_title(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackLink_url(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_url(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.URL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackLink_url(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackLink_status(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_status(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Status, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackLink_status(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _TrackLink_statusName(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_statusName(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.StatusName, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackLink_statusName(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackLink_lastChapterRead(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_lastChapterRead(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.LastChapterRead, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v float64) graphql.Marshaler {
			return ec.marshalNFloat2float64(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackLink_lastChapterRead(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _TrackLink_totalChapters(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_totalChapters(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.TotalChapters, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackLink_totalChapters(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _TrackLink_score(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_score(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Score, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v float64) graphql.Marshaler {
			return ec.marshalNFloat2float64(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackLink_score(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type Float does not have child fields"))
}

func (ec *executionContext) _TrackLink_startedAt(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_startedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.StartedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_TrackLink_startedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _TrackLink_finishedAt(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_finishedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.FinishedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_TrackLink_finishedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _TrackLink_private(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_private(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Private, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackLink_private(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _TrackLink_lastSyncedAt(ctx context.Context, field graphql.CollectedField, obj *model.TrackLink) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackLink_lastSyncedAt(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.LastSyncedAt, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *time.Time) graphql.Marshaler {
			return ec.marshalOTime2ᚖtimeᚐTime(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_TrackLink_lastSyncedAt(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackLink", field, false, false, errors.New("field of type Time does not have child fields"))
}

func (ec *executionContext) _TrackSearchResult_remoteId(ctx context.Context, field graphql.CollectedField, obj *model.TrackSearchResult) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackSearchResult_remoteId(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.RemoteID, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackSearchResult_remoteId(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackSearchResult", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackSearchResult_title(ctx context.Context, field graphql.CollectedField, obj *model.TrackSearchResult) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackSearchResult_title(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Title, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackSearchResult_title(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackSearchResult", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackSearchResult_url(ctx context.Context, field graphql.CollectedField, obj *model.TrackSearchResult) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackSearchResult_url(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.URL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackSearchResult_url(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackSearchResult", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackSearchResult_coverUrl(ctx context.Context, field graphql.CollectedField, obj *model.TrackSearchResult) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackSearchResult_coverUrl(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.CoverURL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_TrackSearchResult_coverUrl(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackSearchResult", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackSearchResult_summary(ctx context.Context, field graphql.CollectedField, obj *model.TrackSearchResult) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackSearchResult_summary(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Summary, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_TrackSearchResult_summary(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackSearchResult", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackSearchResult_totalChapters(ctx context.Context, field graphql.CollectedField, obj *model.TrackSearchResult) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackSearchResult_totalChapters(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.TotalChapters, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *int32) graphql.Marshaler {
			return ec.marshalOInt2ᚖint32(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_TrackSearchResult_totalChapters(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackSearchResult", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _TrackSearchResult_publishingStatus(ctx context.Context, field graphql.CollectedField, obj *model.TrackSearchResult) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackSearchResult_publishingStatus(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.PublishingStatus, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_TrackSearchResult_publishingStatus(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackSearchResult", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackSearchResult_mediaType(ctx context.Context, field graphql.CollectedField, obj *model.TrackSearchResult) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackSearchResult_mediaType(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.MediaType, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_TrackSearchResult_mediaType(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackSearchResult", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackStatus_value(ctx context.Context, field graphql.CollectedField, obj *model.TrackStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackStatus_value(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Value, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackStatus_value(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackStatus", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _TrackStatus_name(ctx context.Context, field graphql.CollectedField, obj *model.TrackStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackStatus_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackStatus_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackStatus", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TrackStatus_animeName(ctx context.Context, field graphql.CollectedField, obj *model.TrackStatus) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TrackStatus_animeName(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.AnimeName, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TrackStatus_animeName(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TrackStatus", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Tracker_key(ctx context.Context, field graphql.CollectedField, obj *model.Tracker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Tracker_key(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Key, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Tracker_key(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Tracker", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Tracker_name(ctx context.Context, field graphql.CollectedField, obj *model.Tracker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Tracker_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Tracker_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Tracker", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Tracker_configured(ctx context.Context, field graphql.CollectedField, obj *model.Tracker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Tracker_configured(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Configured, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Tracker_configured(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Tracker", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Tracker_isLoggedIn(ctx context.Context, field graphql.CollectedField, obj *model.Tracker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Tracker_isLoggedIn(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IsLoggedIn, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Tracker_isLoggedIn(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Tracker", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _Tracker_authUrl(ctx context.Context, field graphql.CollectedField, obj *model.Tracker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Tracker_authUrl(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.AuthURL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Tracker_authUrl(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Tracker", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Tracker_username(ctx context.Context, field graphql.CollectedField, obj *model.Tracker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Tracker_username(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Username, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Tracker_username(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Tracker", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Tracker_scoreOptions(ctx context.Context, field graphql.CollectedField, obj *model.Tracker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Tracker_scoreOptions(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.ScoreOptions, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []string) graphql.Marshaler {
			return ec.marshalNString2ᚕstringᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Tracker_scoreOptions(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Tracker", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _Tracker_statusOptions(ctx context.Context, field graphql.CollectedField, obj *model.Tracker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Tracker_statusOptions(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.StatusOptions, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.TrackStatus) graphql.Marshaler {
			return ec.marshalNTrackStatus2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackStatusᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_Tracker_statusOptions(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Tracker",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_TrackStatus(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Tracker_iconUrl(ctx context.Context, field graphql.CollectedField, obj *model.Tracker) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_Tracker_iconUrl(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IconURL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_Tracker_iconUrl(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("Tracker", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TriStateFilter_name(ctx context.Context, field graphql.CollectedField, obj *model.TriStateFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TriStateFilter_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TriStateFilter_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TriStateFilter", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _TriStateFilter_state(ctx context.Context, field graphql.CollectedField, obj *model.TriStateFilter) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_TriStateFilter_state(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.State, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v int32) graphql.Marshaler {
			return ec.marshalNInt2int32(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_TriStateFilter_state(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("TriStateFilter", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _VideoSource_label(ctx context.Context, field graphql.CollectedField, obj *model.VideoSource) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_VideoSource_label(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Label, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_VideoSource_label(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("VideoSource", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _VideoSource_resolution(ctx context.Context, field graphql.CollectedField, obj *model.VideoSource) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_VideoSource_resolution(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Resolution, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *int32) graphql.Marshaler {
			return ec.marshalOInt2ᚖint32(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext_VideoSource_resolution(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("VideoSource", field, false, false, errors.New("field of type Int does not have child fields"))
}

func (ec *executionContext) _VideoSource_preferred(ctx context.Context, field graphql.CollectedField, obj *model.VideoSource) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_VideoSource_preferred(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Preferred, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_VideoSource_preferred(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("VideoSource", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) _VideoSource_kind(ctx context.Context, field graphql.CollectedField, obj *model.VideoSource) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_VideoSource_kind(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Kind, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_VideoSource_kind(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("VideoSource", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _VideoSource_server(ctx context.Context, field graphql.CollectedField, obj *model.VideoSource) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_VideoSource_server(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Server, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_VideoSource_server(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("VideoSource", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _VideoSource_url(ctx context.Context, field graphql.CollectedField, obj *model.VideoSource) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_VideoSource_url(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.URL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_VideoSource_url(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("VideoSource", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _VideoStream_url(ctx context.Context, field graphql.CollectedField, obj *model.VideoStream) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_VideoStream_url(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.URL, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_VideoStream_url(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("VideoStream", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) _VideoStream_sources(ctx context.Context, field graphql.CollectedField, obj *model.VideoStream) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_VideoStream_sources(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Sources, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.VideoSource) graphql.Marshaler {
			return ec.marshalNVideoSource2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐVideoSourceᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_VideoStream_sources(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "VideoStream",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_VideoSource(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _VideoStream_subtitles(ctx context.Context, field graphql.CollectedField, obj *model.VideoStream) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_VideoStream_subtitles(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Subtitles, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.SubtitleTrack) graphql.Marshaler {
			return ec.marshalNSubtitleTrack2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSubtitleTrackᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_VideoStream_subtitles(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "VideoStream",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_SubtitleTrack(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _VideoStream_audioTracks(ctx context.Context, field graphql.CollectedField, obj *model.VideoStream) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_VideoStream_audioTracks(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.AudioTracks, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.AudioTrack) graphql.Marshaler {
			return ec.marshalNAudioTrack2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐAudioTrackᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_VideoStream_audioTracks(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "VideoStream",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_AudioTrack(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) _VideoStream_skipMarkers(ctx context.Context, field graphql.CollectedField, obj *model.VideoStream) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext_VideoStream_skipMarkers(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.SkipMarkers, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []*model.SkipMarker) graphql.Marshaler {
			return ec.marshalNSkipMarker2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSkipMarkerᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext_VideoStream_skipMarkers(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "VideoStream",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields_SkipMarker(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) ___Directive_name(ctx context.Context, field graphql.CollectedField, obj *introspection.Directive) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Directive_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___Directive_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Directive", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___Directive_description(ctx context.Context, field graphql.CollectedField, obj *introspection.Directive) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Directive_description(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Description(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Directive_description(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Directive", field, true, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___Directive_isRepeatable(ctx context.Context, field graphql.CollectedField, obj *introspection.Directive) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Directive_isRepeatable(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IsRepeatable, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___Directive_isRepeatable(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Directive", field, false, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) ___Directive_locations(ctx context.Context, field graphql.CollectedField, obj *introspection.Directive) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Directive_locations(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Locations, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []string) graphql.Marshaler {
			return ec.marshalN__DirectiveLocation2ᚕstringᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___Directive_locations(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Directive", field, false, false, errors.New("field of type __DirectiveLocation does not have child fields"))
}

func (ec *executionContext) ___Directive_args(ctx context.Context, field graphql.CollectedField, obj *introspection.Directive) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Directive_args(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Args, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []introspection.InputValue) graphql.Marshaler {
			return ec.marshalN__InputValue2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐInputValueᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___Directive_args(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Directive",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___InputValue(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field___Directive_args_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) ___EnumValue_name(ctx context.Context, field graphql.CollectedField, obj *introspection.EnumValue) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___EnumValue_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___EnumValue_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__EnumValue", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___EnumValue_description(ctx context.Context, field graphql.CollectedField, obj *introspection.EnumValue) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___EnumValue_description(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Description(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___EnumValue_description(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__EnumValue", field, true, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___EnumValue_isDeprecated(ctx context.Context, field graphql.CollectedField, obj *introspection.EnumValue) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___EnumValue_isDeprecated(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IsDeprecated(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___EnumValue_isDeprecated(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__EnumValue", field, true, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) ___EnumValue_deprecationReason(ctx context.Context, field graphql.CollectedField, obj *introspection.EnumValue) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___EnumValue_deprecationReason(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.DeprecationReason(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___EnumValue_deprecationReason(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__EnumValue", field, true, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___Field_name(ctx context.Context, field graphql.CollectedField, obj *introspection.Field) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Field_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___Field_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Field", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___Field_description(ctx context.Context, field graphql.CollectedField, obj *introspection.Field) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Field_description(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Description(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Field_description(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Field", field, true, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___Field_args(ctx context.Context, field graphql.CollectedField, obj *introspection.Field) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Field_args(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Args, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []introspection.InputValue) graphql.Marshaler {
			return ec.marshalN__InputValue2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐInputValueᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___Field_args(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Field",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___InputValue(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field___Field_args_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) ___Field_type(ctx context.Context, field graphql.CollectedField, obj *introspection.Field) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Field_type(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Type, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *introspection.Type) graphql.Marshaler {
			return ec.marshalN__Type2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐType(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___Field_type(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Field",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Type(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) ___Field_isDeprecated(ctx context.Context, field graphql.CollectedField, obj *introspection.Field) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Field_isDeprecated(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IsDeprecated(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___Field_isDeprecated(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Field", field, true, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) ___Field_deprecationReason(ctx context.Context, field graphql.CollectedField, obj *introspection.Field) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Field_deprecationReason(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.DeprecationReason(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Field_deprecationReason(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Field", field, true, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___InputValue_name(ctx context.Context, field graphql.CollectedField, obj *introspection.InputValue) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___InputValue_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalNString2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___InputValue_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__InputValue", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___InputValue_description(ctx context.Context, field graphql.CollectedField, obj *introspection.InputValue) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___InputValue_description(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Description(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___InputValue_description(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__InputValue", field, true, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___InputValue_type(ctx context.Context, field graphql.CollectedField, obj *introspection.InputValue) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___InputValue_type(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Type, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *introspection.Type) graphql.Marshaler {
			return ec.marshalN__Type2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐType(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___InputValue_type(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__InputValue",
		Field:      field,
		IsMethod:   false,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Type(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) ___InputValue_defaultValue(ctx context.Context, field graphql.CollectedField, obj *introspection.InputValue) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___InputValue_defaultValue(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.DefaultValue, nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___InputValue_defaultValue(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__InputValue", field, false, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___InputValue_isDeprecated(ctx context.Context, field graphql.CollectedField, obj *introspection.InputValue) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___InputValue_isDeprecated(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IsDeprecated(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalNBoolean2bool(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___InputValue_isDeprecated(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__InputValue", field, true, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) ___InputValue_deprecationReason(ctx context.Context, field graphql.CollectedField, obj *introspection.InputValue) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___InputValue_deprecationReason(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.DeprecationReason(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___InputValue_deprecationReason(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__InputValue", field, true, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___Schema_description(ctx context.Context, field graphql.CollectedField, obj *introspection.Schema) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Schema_description(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Description(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Schema_description(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Schema", field, true, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___Schema_types(ctx context.Context, field graphql.CollectedField, obj *introspection.Schema) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Schema_types(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Types(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []introspection.Type) graphql.Marshaler {
			return ec.marshalN__Type2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐTypeᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___Schema_types(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Schema",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Type(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) ___Schema_queryType(ctx context.Context, field graphql.CollectedField, obj *introspection.Schema) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Schema_queryType(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.QueryType(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *introspection.Type) graphql.Marshaler {
			return ec.marshalN__Type2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐType(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___Schema_queryType(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Schema",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Type(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) ___Schema_mutationType(ctx context.Context, field graphql.CollectedField, obj *introspection.Schema) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Schema_mutationType(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.MutationType(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *introspection.Type) graphql.Marshaler {
			return ec.marshalO__Type2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐType(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Schema_mutationType(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Schema",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Type(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) ___Schema_subscriptionType(ctx context.Context, field graphql.CollectedField, obj *introspection.Schema) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Schema_subscriptionType(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.SubscriptionType(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *introspection.Type) graphql.Marshaler {
			return ec.marshalO__Type2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐType(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Schema_subscriptionType(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Schema",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Type(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) ___Schema_directives(ctx context.Context, field graphql.CollectedField, obj *introspection.Schema) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Schema_directives(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Directives(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []introspection.Directive) graphql.Marshaler {
			return ec.marshalN__Directive2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐDirectiveᚄ(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___Schema_directives(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Schema",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Directive(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) ___Type_kind(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Type_kind(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Kind(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v string) graphql.Marshaler {
			return ec.marshalN__TypeKind2string(ctx, selections, v)
		},
		true,
		true,
	)
}
func (ec *executionContext) fieldContext___Type_kind(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Type", field, true, false, errors.New("field of type __TypeKind does not have child fields"))
}

func (ec *executionContext) ___Type_name(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Type_name(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Name(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Type_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Type", field, true, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___Type_description(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Type_description(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Description(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Type_description(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Type", field, true, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___Type_specifiedByURL(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Type_specifiedByURL(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.SpecifiedByURL(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *string) graphql.Marshaler {
			return ec.marshalOString2ᚖstring(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Type_specifiedByURL(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Type", field, true, false, errors.New("field of type String does not have child fields"))
}

func (ec *executionContext) ___Type_fields(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Type_fields(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return obj.Fields(fc.Args["includeDeprecated"].(bool)), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []introspection.Field) graphql.Marshaler {
			return ec.marshalO__Field2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐFieldᚄ(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Type_fields(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Type",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Field(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field___Type_fields_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) ___Type_interfaces(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Type_interfaces(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.Interfaces(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []introspection.Type) graphql.Marshaler {
			return ec.marshalO__Type2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐTypeᚄ(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Type_interfaces(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Type",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Type(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) ___Type_possibleTypes(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Type_possibleTypes(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.PossibleTypes(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []introspection.Type) graphql.Marshaler {
			return ec.marshalO__Type2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐTypeᚄ(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Type_possibleTypes(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Type",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Type(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) ___Type_enumValues(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Type_enumValues(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			fc := graphql.GetFieldContext(ctx)
			return obj.EnumValues(fc.Args["includeDeprecated"].(bool)), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []introspection.EnumValue) graphql.Marshaler {
			return ec.marshalO__EnumValue2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐEnumValueᚄ(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Type_enumValues(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Type",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___EnumValue(ctx, field)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field___Type_enumValues_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) ___Type_inputFields(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Type_inputFields(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.InputFields(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v []introspection.InputValue) graphql.Marshaler {
			return ec.marshalO__InputValue2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐInputValueᚄ(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Type_inputFields(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Type",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___InputValue(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) ___Type_ofType(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Type_ofType(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.OfType(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v *introspection.Type) graphql.Marshaler {
			return ec.marshalO__Type2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐType(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Type_ofType(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "__Type",
		Field:      field,
		IsMethod:   true,
		IsResolver: false,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.childFields___Type(ctx, field)
		},
	}
	return fc, nil
}

func (ec *executionContext) ___Type_isOneOf(ctx context.Context, field graphql.CollectedField, obj *introspection.Type) (ret graphql.Marshaler) {
	return graphql.ResolveField(
		ctx,
		ec.OperationContext,
		field,
		func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return ec.fieldContext___Type_isOneOf(ctx, field)
		},
		func(ctx context.Context) (any, error) {
			return obj.IsOneOf(), nil
		},
		nil,
		func(ctx context.Context, selections ast.SelectionSet, v bool) graphql.Marshaler {
			return ec.marshalOBoolean2bool(ctx, selections, v)
		},
		true,
		false,
	)
}
func (ec *executionContext) fieldContext___Type_isOneOf(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	return graphql.NewScalarFieldContext("__Type", field, true, false, errors.New("field of type Boolean does not have child fields"))
}

func (ec *executionContext) unmarshalInputCheckBoxFilterInput(ctx context.Context, obj any) (model.CheckBoxFilterInput, error) {
	var it model.CheckBoxFilterInput
	if obj == nil {
		return it, nil
	}

	asMap := map[string]any{}
	for k, v := range obj.(map[string]any) {
		asMap[k] = v
	}

	fieldsInOrder := [...]string{"state"}
	for _, k := range fieldsInOrder {
		v, ok := asMap[k]
		if !ok {
			continue
		}
		switch k {
		case "state":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("state"))
			data, err := ec.unmarshalNBoolean2bool(ctx, v)
			if err != nil {
				return it, err
			}
			it.State = data
		}
	}
	return it, nil
}

func (ec *executionContext) unmarshalInputFilterInput(ctx context.Context, obj any) (model.FilterInput, error) {
	var it model.FilterInput
	if obj == nil {
		return it, nil
	}

	asMap := map[string]any{}
	for k, v := range obj.(map[string]any) {
		asMap[k] = v
	}

	fieldsInOrder := [...]string{"name", "select", "text", "checkbox", "tristate", "group", "sort"}
	for _, k := range fieldsInOrder {
		v, ok := asMap[k]
		if !ok {
			continue
		}
		switch k {
		case "name":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("name"))
			data, err := ec.unmarshalNString2string(ctx, v)
			if err != nil {
				return it, err
			}
			it.Name = data
		case "select":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("select"))
			data, err := ec.unmarshalOSelectFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSelectFilterInput(ctx, v)
			if err != nil {
				return it, err
			}
			it.Select = data
		case "text":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("text"))
			data, err := ec.unmarshalOTextFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTextFilterInput(ctx, v)
			if err != nil {
				return it, err
			}
			it.Text = data
		case "checkbox":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("checkbox"))
			data, err := ec.unmarshalOCheckBoxFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐCheckBoxFilterInput(ctx, v)
			if err != nil {
				return it, err
			}
			it.Checkbox = data
		case "tristate":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("tristate"))
			data, err := ec.unmarshalOTriStateFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTriStateFilterInput(ctx, v)
			if err != nil {
				return it, err
			}
			it.Tristate = data
		case "group":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("group"))
			data, err := ec.unmarshalOGroupFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐGroupFilterInput(ctx, v)
			if err != nil {
				return it, err
			}
			it.Group = data
		case "sort":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("sort"))
			data, err := ec.unmarshalOSortFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSortFilterInput(ctx, v)
			if err != nil {
				return it, err
			}
			it.Sort = data
		}
	}
	return it, nil
}

func (ec *executionContext) unmarshalInputGroupFilterInput(ctx context.Context, obj any) (model.GroupFilterInput, error) {
	var it model.GroupFilterInput
	if obj == nil {
		return it, nil
	}

	asMap := map[string]any{}
	for k, v := range obj.(map[string]any) {
		asMap[k] = v
	}

	fieldsInOrder := [...]string{"children"}
	for _, k := range fieldsInOrder {
		v, ok := asMap[k]
		if !ok {
			continue
		}
		switch k {
		case "children":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("children"))
			data, err := ec.unmarshalNFilterInput2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFilterInputᚄ(ctx, v)
			if err != nil {
				return it, err
			}
			it.Children = data
		}
	}
	return it, nil
}

func (ec *executionContext) unmarshalInputLibraryFilter(ctx context.Context, obj any) (model.LibraryFilter, error) {
	var it model.LibraryFilter
	if obj == nil {
		return it, nil
	}

	asMap := map[string]any{}
	for k, v := range obj.(map[string]any) {
		asMap[k] = v
	}

	if _, present := asMap["inLibrary"]; !present {
		asMap["inLibrary"] = true
	}

	fieldsInOrder := [...]string{"contentType", "inLibrary", "unreadOnly", "downloadedOnly", "tagIds", "folderId", "query"}
	for _, k := range fieldsInOrder {
		v, ok := asMap[k]
		if !ok {
			continue
		}
		switch k {
		case "contentType":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("contentType"))
			data, err := ec.unmarshalOContentType2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐContentType(ctx, v)
			if err != nil {
				return it, err
			}
			it.ContentType = data
		case "inLibrary":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("inLibrary"))
			data, err := ec.unmarshalOBoolean2ᚖbool(ctx, v)
			if err != nil {
				return it, err
			}
			it.InLibrary = data
		case "unreadOnly":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("unreadOnly"))
			data, err := ec.unmarshalOBoolean2ᚖbool(ctx, v)
			if err != nil {
				return it, err
			}
			it.UnreadOnly = data
		case "downloadedOnly":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("downloadedOnly"))
			data, err := ec.unmarshalOBoolean2ᚖbool(ctx, v)
			if err != nil {
				return it, err
			}
			it.DownloadedOnly = data
		case "tagIds":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("tagIds"))
			data, err := ec.unmarshalOID2ᚕstringᚄ(ctx, v)
			if err != nil {
				return it, err
			}
			it.TagIds = data
		case "folderId":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("folderId"))
			data, err := ec.unmarshalOID2ᚖstring(ctx, v)
			if err != nil {
				return it, err
			}
			it.FolderID = data
		case "query":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("query"))
			data, err := ec.unmarshalOString2ᚖstring(ctx, v)
			if err != nil {
				return it, err
			}
			it.Query = data
		}
	}
	return it, nil
}

func (ec *executionContext) unmarshalInputLibrarySortInput(ctx context.Context, obj any) (model.LibrarySortInput, error) {
	var it model.LibrarySortInput
	if obj == nil {
		return it, nil
	}

	asMap := map[string]any{}
	for k, v := range obj.(map[string]any) {
		asMap[k] = v
	}

	if _, present := asMap["by"]; !present {
		asMap["by"] = "ADDED_AT"
	}
	if _, present := asMap["ascending"]; !present {
		asMap["ascending"] = false
	}

	fieldsInOrder := [...]string{"by", "ascending"}
	for _, k := range fieldsInOrder {
		v, ok := asMap[k]
		if !ok {
			continue
		}
		switch k {
		case "by":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("by"))
			data, err := ec.unmarshalNLibrarySort2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐLibrarySort(ctx, v)
			if err != nil {
				return it, err
			}
			it.By = data
		case "ascending":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("ascending"))
			data, err := ec.unmarshalNBoolean2bool(ctx, v)
			if err != nil {
				return it, err
			}
			it.Ascending = data
		}
	}
	return it, nil
}

func (ec *executionContext) unmarshalInputSelectFilterInput(ctx context.Context, obj any) (model.SelectFilterInput, error) {
	var it model.SelectFilterInput
	if obj == nil {
		return it, nil
	}

	asMap := map[string]any{}
	for k, v := range obj.(map[string]any) {
		asMap[k] = v
	}

	fieldsInOrder := [...]string{"state"}
	for _, k := range fieldsInOrder {
		v, ok := asMap[k]
		if !ok {
			continue
		}
		switch k {
		case "state":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("state"))
			data, err := ec.unmarshalNInt2int32(ctx, v)
			if err != nil {
				return it, err
			}
			it.State = data
		}
	}
	return it, nil
}

func (ec *executionContext) unmarshalInputSortFilterInput(ctx context.Context, obj any) (model.SortFilterInput, error) {
	var it model.SortFilterInput
	if obj == nil {
		return it, nil
	}

	asMap := map[string]any{}
	for k, v := range obj.(map[string]any) {
		asMap[k] = v
	}

	fieldsInOrder := [...]string{"hasState", "index", "ascending"}
	for _, k := range fieldsInOrder {
		v, ok := asMap[k]
		if !ok {
			continue
		}
		switch k {
		case "hasState":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("hasState"))
			data, err := ec.unmarshalNBoolean2bool(ctx, v)
			if err != nil {
				return it, err
			}
			it.HasState = data
		case "index":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("index"))
			data, err := ec.unmarshalOInt2ᚖint32(ctx, v)
			if err != nil {
				return it, err
			}
			it.Index = data
		case "ascending":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("ascending"))
			data, err := ec.unmarshalOBoolean2ᚖbool(ctx, v)
			if err != nil {
				return it, err
			}
			it.Ascending = data
		}
	}
	return it, nil
}

func (ec *executionContext) unmarshalInputTextFilterInput(ctx context.Context, obj any) (model.TextFilterInput, error) {
	var it model.TextFilterInput
	if obj == nil {
		return it, nil
	}

	asMap := map[string]any{}
	for k, v := range obj.(map[string]any) {
		asMap[k] = v
	}

	fieldsInOrder := [...]string{"state"}
	for _, k := range fieldsInOrder {
		v, ok := asMap[k]
		if !ok {
			continue
		}
		switch k {
		case "state":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("state"))
			data, err := ec.unmarshalNString2string(ctx, v)
			if err != nil {
				return it, err
			}
			it.State = data
		}
	}
	return it, nil
}

func (ec *executionContext) unmarshalInputTriStateFilterInput(ctx context.Context, obj any) (model.TriStateFilterInput, error) {
	var it model.TriStateFilterInput
	if obj == nil {
		return it, nil
	}

	asMap := map[string]any{}
	for k, v := range obj.(map[string]any) {
		asMap[k] = v
	}

	fieldsInOrder := [...]string{"state"}
	for _, k := range fieldsInOrder {
		v, ok := asMap[k]
		if !ok {
			continue
		}
		switch k {
		case "state":
			ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("state"))
			data, err := ec.unmarshalNInt2int32(ctx, v)
			if err != nil {
				return it, err
			}
			it.State = data
		}
	}
	return it, nil
}

func (ec *executionContext) _FilterNode(ctx context.Context, sel ast.SelectionSet, obj model.FilterNode) graphql.Marshaler {
	switch obj := (obj).(type) {
	case nil:
		return graphql.Null
	case model.TriStateFilter:
		return ec._TriStateFilter(ctx, sel, &obj)
	case *model.TriStateFilter:
		if obj == nil {
			return graphql.Null
		}
		return ec._TriStateFilter(ctx, sel, obj)
	case model.TextFilter:
		return ec._TextFilter(ctx, sel, &obj)
	case *model.TextFilter:
		if obj == nil {
			return graphql.Null
		}
		return ec._TextFilter(ctx, sel, obj)
	case model.SortFilter:
		return ec._SortFilter(ctx, sel, &obj)
	case *model.SortFilter:
		if obj == nil {
			return graphql.Null
		}
		return ec._SortFilter(ctx, sel, obj)
	case model.SeparatorFilter:
		return ec._SeparatorFilter(ctx, sel, &obj)
	case *model.SeparatorFilter:
		if obj == nil {
			return graphql.Null
		}
		return ec._SeparatorFilter(ctx, sel, obj)
	case model.SelectFilter:
		return ec._SelectFilter(ctx, sel, &obj)
	case *model.SelectFilter:
		if obj == nil {
			return graphql.Null
		}
		return ec._SelectFilter(ctx, sel, obj)
	case model.HeaderFilter:
		return ec._HeaderFilter(ctx, sel, &obj)
	case *model.HeaderFilter:
		if obj == nil {
			return graphql.Null
		}
		return ec._HeaderFilter(ctx, sel, obj)
	case model.GroupFilter:
		return ec._GroupFilter(ctx, sel, &obj)
	case *model.GroupFilter:
		if obj == nil {
			return graphql.Null
		}
		return ec._GroupFilter(ctx, sel, obj)
	case model.CheckBoxFilter:
		return ec._CheckBoxFilter(ctx, sel, &obj)
	case *model.CheckBoxFilter:
		if obj == nil {
			return graphql.Null
		}
		return ec._CheckBoxFilter(ctx, sel, obj)
	default:
		if typedObj, ok := obj.(graphql.Marshaler); ok {
			return typedObj
		} else {
			panic(fmt.Errorf("unexpected type %T; non-generated variants of FilterNode must implement graphql.Marshaler", obj))
		}
	}
}

var aboutServerImplementors = []string{"AboutServer"}

func (ec *executionContext) _AboutServer(ctx context.Context, sel ast.SelectionSet, obj *model.AboutServer) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, aboutServerImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("AboutServer")
		case "name":
			out.Values[i] = ec._AboutServer_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "version":
			out.Values[i] = ec._AboutServer_version(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "buildTime":
			out.Values[i] = ec._AboutServer_buildTime(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var audioTrackImplementors = []string{"AudioTrack"}

func (ec *executionContext) _AudioTrack(ctx context.Context, sel ast.SelectionSet, obj *model.AudioTrack) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, audioTrackImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("AudioTrack")
		case "lang":
			out.Values[i] = ec._AudioTrack_lang(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "url":
			out.Values[i] = ec._AudioTrack_url(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var chapterImplementors = []string{"Chapter"}

func (ec *executionContext) _Chapter(ctx context.Context, sel ast.SelectionSet, obj *model.Chapter) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, chapterImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Chapter")
		case "id":
			out.Values[i] = ec._Chapter_id(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "mediaId":
			out.Values[i] = ec._Chapter_mediaId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "externalId":
			out.Values[i] = ec._Chapter_externalId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "title":
			out.Values[i] = ec._Chapter_title(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "number":
			out.Values[i] = ec._Chapter_number(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "sourceOrder":
			out.Values[i] = ec._Chapter_sourceOrder(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "scanlator":
			out.Values[i] = ec._Chapter_scanlator(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "uploadedAt":
			out.Values[i] = ec._Chapter_uploadedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "readingProgress":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Chapter_readingProgress(ctx, field, obj)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "completed":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Chapter_completed(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "downloaded":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Chapter_downloaded(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "download":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Chapter_download(ctx, field, obj)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "pages":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Chapter_pages(ctx, field, obj)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "pageCount":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Chapter_pageCount(ctx, field, obj)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "videoUrl":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Chapter_videoUrl(ctx, field, obj)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "videoStream":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Chapter_videoStream(ctx, field, obj)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var checkBoxFilterImplementors = []string{"CheckBoxFilter", "FilterNode"}

func (ec *executionContext) _CheckBoxFilter(ctx context.Context, sel ast.SelectionSet, obj *model.CheckBoxFilter) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, checkBoxFilterImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("CheckBoxFilter")
		case "name":
			out.Values[i] = ec._CheckBoxFilter_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "state":
			out.Values[i] = ec._CheckBoxFilter_state(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var downloadImplementors = []string{"Download"}

func (ec *executionContext) _Download(ctx context.Context, sel ast.SelectionSet, obj *model.Download) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, downloadImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Download")
		case "id":
			out.Values[i] = ec._Download_id(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "mediaId":
			out.Values[i] = ec._Download_mediaId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "chapterId":
			out.Values[i] = ec._Download_chapterId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "chapter":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Download_chapter(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "status":
			out.Values[i] = ec._Download_status(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "progress":
			out.Values[i] = ec._Download_progress(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "downloadedBytes":
			out.Values[i] = ec._Download_downloadedBytes(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "bytesPerSec":
			out.Values[i] = ec._Download_bytesPerSec(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "finalSizeBytes":
			out.Values[i] = ec._Download_finalSizeBytes(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "error":
			out.Values[i] = ec._Download_error(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "createdAt":
			out.Values[i] = ec._Download_createdAt(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "completedAt":
			out.Values[i] = ec._Download_completedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var downloaderStatusImplementors = []string{"DownloaderStatus"}

func (ec *executionContext) _DownloaderStatus(ctx context.Context, sel ast.SelectionSet, obj *model.DownloaderStatus) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, downloaderStatusImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("DownloaderStatus")
		case "isRunning":
			out.Values[i] = ec._DownloaderStatus_isRunning(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "queuedCount":
			out.Values[i] = ec._DownloaderStatus_queuedCount(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "downloadingCount":
			out.Values[i] = ec._DownloaderStatus_downloadingCount(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "failedCount":
			out.Values[i] = ec._DownloaderStatus_failedCount(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var extensionImplementors = []string{"Extension"}

func (ec *executionContext) _Extension(ctx context.Context, sel ast.SelectionSet, obj *model.Extension) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, extensionImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Extension")
		case "id":
			out.Values[i] = ec._Extension_id(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "repositoryId":
			out.Values[i] = ec._Extension_repositoryId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "packageName":
			out.Values[i] = ec._Extension_packageName(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "name":
			out.Values[i] = ec._Extension_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "version":
			out.Values[i] = ec._Extension_version(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "contentType":
			out.Values[i] = ec._Extension_contentType(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "lang":
			out.Values[i] = ec._Extension_lang(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "iconUrl":
			out.Values[i] = ec._Extension_iconUrl(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "apkUrl":
			out.Values[i] = ec._Extension_apkUrl(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "jarUrl":
			out.Values[i] = ec._Extension_jarUrl(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "jarPath":
			out.Values[i] = ec._Extension_jarPath(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "installed":
			out.Values[i] = ec._Extension_installed(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "enabled":
			out.Values[i] = ec._Extension_enabled(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "discoveredAt":
			out.Values[i] = ec._Extension_discoveredAt(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "installedAt":
			out.Values[i] = ec._Extension_installedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "installedVersion":
			out.Values[i] = ec._Extension_installedVersion(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "needsUpdate":
			out.Values[i] = ec._Extension_needsUpdate(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "isNsfw":
			out.Values[i] = ec._Extension_isNsfw(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "displayName":
			out.Values[i] = ec._Extension_displayName(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "supportsLatest":
			out.Values[i] = ec._Extension_supportsLatest(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var folderImplementors = []string{"Folder"}

func (ec *executionContext) _Folder(ctx context.Context, sel ast.SelectionSet, obj *model.Folder) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, folderImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Folder")
		case "id":
			out.Values[i] = ec._Folder_id(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "name":
			out.Values[i] = ec._Folder_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "kind":
			out.Values[i] = ec._Folder_kind(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "systemKey":
			out.Values[i] = ec._Folder_systemKey(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "parentFolderId":
			out.Values[i] = ec._Folder_parentFolderId(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "sortOrder":
			out.Values[i] = ec._Folder_sortOrder(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "includeInUpdate":
			out.Values[i] = ec._Folder_includeInUpdate(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "includeInDownload":
			out.Values[i] = ec._Folder_includeInDownload(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var groupFilterImplementors = []string{"GroupFilter", "FilterNode"}

func (ec *executionContext) _GroupFilter(ctx context.Context, sel ast.SelectionSet, obj *model.GroupFilter) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, groupFilterImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("GroupFilter")
		case "name":
			out.Values[i] = ec._GroupFilter_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "children":
			out.Values[i] = ec._GroupFilter_children(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var headerFilterImplementors = []string{"HeaderFilter", "FilterNode"}

func (ec *executionContext) _HeaderFilter(ctx context.Context, sel ast.SelectionSet, obj *model.HeaderFilter) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, headerFilterImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("HeaderFilter")
		case "name":
			out.Values[i] = ec._HeaderFilter_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var libraryUpdateStatusImplementors = []string{"LibraryUpdateStatus"}

func (ec *executionContext) _LibraryUpdateStatus(ctx context.Context, sel ast.SelectionSet, obj *model.LibraryUpdateStatus) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, libraryUpdateStatusImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("LibraryUpdateStatus")
		case "running":
			out.Values[i] = ec._LibraryUpdateStatus_running(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "total":
			out.Values[i] = ec._LibraryUpdateStatus_total(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "done":
			out.Values[i] = ec._LibraryUpdateStatus_done(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "currentTitle":
			out.Values[i] = ec._LibraryUpdateStatus_currentTitle(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "newChapterCount":
			out.Values[i] = ec._LibraryUpdateStatus_newChapterCount(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "failedTitles":
			out.Values[i] = ec._LibraryUpdateStatus_failedTitles(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "startedAt":
			out.Values[i] = ec._LibraryUpdateStatus_startedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "finishedAt":
			out.Values[i] = ec._LibraryUpdateStatus_finishedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var mediaImplementors = []string{"Media"}

func (ec *executionContext) _Media(ctx context.Context, sel ast.SelectionSet, obj *model.Media) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, mediaImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Media")
		case "id":
			out.Values[i] = ec._Media_id(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "extensionId":
			out.Values[i] = ec._Media_extensionId(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "extensionName":
			out.Values[i] = ec._Media_extensionName(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "sourceName":
			out.Values[i] = ec._Media_sourceName(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "externalId":
			out.Values[i] = ec._Media_externalId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "contentType":
			out.Values[i] = ec._Media_contentType(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "title":
			out.Values[i] = ec._Media_title(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "thumbnailUrl":
			out.Values[i] = ec._Media_thumbnailUrl(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "description":
			out.Values[i] = ec._Media_description(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "status":
			out.Values[i] = ec._Media_status(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "author":
			out.Values[i] = ec._Media_author(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "artist":
			out.Values[i] = ec._Media_artist(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "detailsFetchedAt":
			out.Values[i] = ec._Media_detailsFetchedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "extensionRemovedAt":
			out.Values[i] = ec._Media_extensionRemovedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "addedAt":
			out.Values[i] = ec._Media_addedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "lastViewedAt":
			out.Values[i] = ec._Media_lastViewedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "inLibrary":
			out.Values[i] = ec._Media_inLibrary(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "chapters":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_chapters(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "chapterCount":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_chapterCount(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "unreadCount":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_unreadCount(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "downloadedCount":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_downloadedCount(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "nextUnreadChapter":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_nextUnreadChapter(ctx, field, obj)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "latestChapter":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_latestChapter(ctx, field, obj)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "readingProgress":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_readingProgress(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "tags":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_tags(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "genres":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_genres(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "folders":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_folders(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "trackLinks":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_trackLinks(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "metadata":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_metadata(ctx, field, obj)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "source":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Media_source(ctx, field, obj)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.IsDeferred() {
				deferredFieldSet.AddField(field)
				fieldIndex := len(deferredFieldSet.Values) - 1
				deferredFieldSet.Concurrently(fieldIndex, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, deferredFieldSet)
				})

				for _, deferrable := range field.Deferrables {
					view, ok := deferLabelToView[deferrable.Label]
					if !ok {
						view = deferredFieldSet.NewView()
						deferLabelToView[deferrable.Label] = view
					}
					view.AddIndices(fieldIndex)
				}

				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var mediaPageImplementors = []string{"MediaPage"}

func (ec *executionContext) _MediaPage(ctx context.Context, sel ast.SelectionSet, obj *model.MediaPage) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, mediaPageImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("MediaPage")
		case "items":
			out.Values[i] = ec._MediaPage_items(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "total":
			out.Values[i] = ec._MediaPage_total(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "hasMore":
			out.Values[i] = ec._MediaPage_hasMore(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var metadataCandidateImplementors = []string{"MetadataCandidate"}

func (ec *executionContext) _MetadataCandidate(ctx context.Context, sel ast.SelectionSet, obj *model.MetadataCandidate) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, metadataCandidateImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("MetadataCandidate")
		case "provider":
			out.Values[i] = ec._MetadataCandidate_provider(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "providerId":
			out.Values[i] = ec._MetadataCandidate_providerId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "title":
			out.Values[i] = ec._MetadataCandidate_title(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "url":
			out.Values[i] = ec._MetadataCandidate_url(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "coverUrl":
			out.Values[i] = ec._MetadataCandidate_coverUrl(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "description":
			out.Values[i] = ec._MetadataCandidate_description(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "status":
			out.Values[i] = ec._MetadataCandidate_status(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "genres":
			out.Values[i] = ec._MetadataCandidate_genres(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "startYear":
			out.Values[i] = ec._MetadataCandidate_startYear(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var metadataMatchImplementors = []string{"MetadataMatch"}

func (ec *executionContext) _MetadataMatch(ctx context.Context, sel ast.SelectionSet, obj *model.MetadataMatch) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, metadataMatchImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("MetadataMatch")
		case "provider":
			out.Values[i] = ec._MetadataMatch_provider(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "providerId":
			out.Values[i] = ec._MetadataMatch_providerId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "url":
			out.Values[i] = ec._MetadataMatch_url(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "confidence":
			out.Values[i] = ec._MetadataMatch_confidence(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "locked":
			out.Values[i] = ec._MetadataMatch_locked(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "matchedAt":
			out.Values[i] = ec._MetadataMatch_matchedAt(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var mutationImplementors = []string{"Mutation"}

func (ec *executionContext) _Mutation(ctx context.Context, sel ast.SelectionSet) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, mutationImplementors)
	ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{
		Object: "Mutation",
	})

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		innerCtx := graphql.WithRootFieldContext(ctx, &graphql.RootFieldContext{
			Object: field.Name,
			Field:  field,
		})

		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Mutation")
		case "createFolder":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_createFolder(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "renameFolder":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_renameFolder(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "deleteFolder":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_deleteFolder(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "addMediaToFolder":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_addMediaToFolder(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "removeMediaFromFolder":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_removeMediaFromFolder(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "markChaptersRead":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_markChaptersRead(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "dequeueDownload":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_dequeueDownload(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "addRepository":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_addRepository(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "renameRepository":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_renameRepository(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "deleteRepository":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_deleteRepository(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "installExtension":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_installExtension(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "installExternalExtension":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_installExternalExtension(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "uninstallExtension":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_uninstallExtension(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "updateExtension":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_updateExtension(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "setInLibrary":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_setInLibrary(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "migrateMedia":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_migrateMedia(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "syncChapters":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_syncChapters(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "updateReadingProgress":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_updateReadingProgress(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "markChapterRead":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_markChapterRead(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "enqueueDownload":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_enqueueDownload(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "retryDownload":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_retryDownload(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "deleteDownload":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_deleteDownload(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "reorderDownload":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_reorderDownload(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "clearDownloads":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_clearDownloads(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "startDownloader":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_startDownloader(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "stopDownloader":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_stopDownloader(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "refreshMetadata":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_refreshMetadata(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "refreshFolder":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_refreshFolder(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "reorderFolder":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_reorderFolder(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "updateFolderFlags":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_updateFolderFlags(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "clearImageCache":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_clearImageCache(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "startLibraryUpdate":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_startLibraryUpdate(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "setMediaCover":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_setMediaCover(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "rescanLocalMedia":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_rescanLocalMedia(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "trackerLogin":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_trackerLogin(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "trackerLogout":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_trackerLogout(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "bindTrack":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_bindTrack(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "updateTrack":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_updateTrack(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "unbindTrack":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_unbindTrack(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "resyncTrack":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_resyncTrack(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "pullTracker":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_pullTracker(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "applyMetadataMatch":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_applyMetadataMatch(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "unlinkMetadata":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_unlinkMetadata(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "refreshMetadataMatch":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Mutation_refreshMetadataMatch(ctx, field)
			})
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var queryImplementors = []string{"Query"}

func (ec *executionContext) _Query(ctx context.Context, sel ast.SelectionSet) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, queryImplementors)
	ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{
		Object: "Query",
	})

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		innerCtx := graphql.WithRootFieldContext(ctx, &graphql.RootFieldContext{
			Object: field.Name,
			Field:  field,
		})

		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Query")
		case "about":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_about(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "folders":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_folders(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "folder":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_folder(ctx, field)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "mediaInFolder":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_mediaInFolder(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "repositories":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_repositories(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "availableExtensions":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_availableExtensions(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "installedExtensions":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_installedExtensions(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "library":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_library(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "media":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_media(ctx, field)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "chapter":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_chapter(ctx, field)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "resolveMedia":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_resolveMedia(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "readingProgress":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_readingProgress(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "search":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_search(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "filterOptions":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_filterOptions(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "popularManga":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_popularManga(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "latestUpdates":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_latestUpdates(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "downloadStatus":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_downloadStatus(ctx, field)
				if res == graphql.RequiredNull {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "downloadQueue":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_downloadQueue(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "downloaderStatus":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_downloaderStatus(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "recentChapters":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_recentChapters(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "chapterUpdates":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_chapterUpdates(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "libraryUpdateStatus":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_libraryUpdateStatus(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "storageInfo":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_storageInfo(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "trackers":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_trackers(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "trackSearch":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_trackSearch(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "searchMetadata":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_searchMetadata(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "skipTimestamps":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Query_skipTimestamps(ctx, field)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			rrm := func(ctx context.Context) graphql.Marshaler {
				return ec.OperationContext.RootResolverMiddleware(ctx,
					func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return rrm(innerCtx) })
		case "__type":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Query___type(ctx, field)
			})
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		case "__schema":
			out.Values[i] = ec.OperationContext.RootResolverMiddleware(innerCtx, func(ctx context.Context) (res graphql.Marshaler) {
				return ec._Query___schema(ctx, field)
			})
			if out.Values[i] == graphql.RequiredNull {
				atomic.AddUint32(&out.Invalids, 1)
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var readingProgressImplementors = []string{"ReadingProgress"}

func (ec *executionContext) _ReadingProgress(ctx context.Context, sel ast.SelectionSet, obj *model.ReadingProgress) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, readingProgressImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("ReadingProgress")
		case "id":
			out.Values[i] = ec._ReadingProgress_id(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "mediaId":
			out.Values[i] = ec._ReadingProgress_mediaId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "chapterId":
			out.Values[i] = ec._ReadingProgress_chapterId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "progress":
			out.Values[i] = ec._ReadingProgress_progress(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "completed":
			out.Values[i] = ec._ReadingProgress_completed(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "positionSeconds":
			out.Values[i] = ec._ReadingProgress_positionSeconds(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "durationSeconds":
			out.Values[i] = ec._ReadingProgress_durationSeconds(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "updatedAt":
			out.Values[i] = ec._ReadingProgress_updatedAt(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var recentChapterImplementors = []string{"RecentChapter"}

func (ec *executionContext) _RecentChapter(ctx context.Context, sel ast.SelectionSet, obj *model.RecentChapter) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, recentChapterImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("RecentChapter")
		case "chapter":
			out.Values[i] = ec._RecentChapter_chapter(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "media":
			out.Values[i] = ec._RecentChapter_media(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var repositoryImplementors = []string{"Repository"}

func (ec *executionContext) _Repository(ctx context.Context, sel ast.SelectionSet, obj *model.Repository) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, repositoryImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Repository")
		case "id":
			out.Values[i] = ec._Repository_id(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "indexUrl":
			out.Values[i] = ec._Repository_indexUrl(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "name":
			out.Values[i] = ec._Repository_name(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "contentType":
			out.Values[i] = ec._Repository_contentType(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "addedAt":
			out.Values[i] = ec._Repository_addedAt(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "lastSyncedAt":
			out.Values[i] = ec._Repository_lastSyncedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var searchResponseImplementors = []string{"SearchResponse"}

func (ec *executionContext) _SearchResponse(ctx context.Context, sel ast.SelectionSet, obj *model.SearchResponse) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, searchResponseImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("SearchResponse")
		case "results":
			out.Values[i] = ec._SearchResponse_results(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "hasNextPage":
			out.Values[i] = ec._SearchResponse_hasNextPage(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var selectFilterImplementors = []string{"SelectFilter", "FilterNode"}

func (ec *executionContext) _SelectFilter(ctx context.Context, sel ast.SelectionSet, obj *model.SelectFilter) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, selectFilterImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("SelectFilter")
		case "name":
			out.Values[i] = ec._SelectFilter_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "values":
			out.Values[i] = ec._SelectFilter_values(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "state":
			out.Values[i] = ec._SelectFilter_state(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var separatorFilterImplementors = []string{"SeparatorFilter", "FilterNode"}

func (ec *executionContext) _SeparatorFilter(ctx context.Context, sel ast.SelectionSet, obj *model.SeparatorFilter) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, separatorFilterImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("SeparatorFilter")
		case "name":
			out.Values[i] = ec._SeparatorFilter_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var skipMarkerImplementors = []string{"SkipMarker"}

func (ec *executionContext) _SkipMarker(ctx context.Context, sel ast.SelectionSet, obj *model.SkipMarker) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, skipMarkerImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("SkipMarker")
		case "type":
			out.Values[i] = ec._SkipMarker_type(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "name":
			out.Values[i] = ec._SkipMarker_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "startMs":
			out.Values[i] = ec._SkipMarker_startMs(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "endMs":
			out.Values[i] = ec._SkipMarker_endMs(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var sortFilterImplementors = []string{"SortFilter", "FilterNode"}

func (ec *executionContext) _SortFilter(ctx context.Context, sel ast.SelectionSet, obj *model.SortFilter) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, sortFilterImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("SortFilter")
		case "name":
			out.Values[i] = ec._SortFilter_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "values":
			out.Values[i] = ec._SortFilter_values(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "hasState":
			out.Values[i] = ec._SortFilter_hasState(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "index":
			out.Values[i] = ec._SortFilter_index(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "ascending":
			out.Values[i] = ec._SortFilter_ascending(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var storageInfoImplementors = []string{"StorageInfo"}

func (ec *executionContext) _StorageInfo(ctx context.Context, sel ast.SelectionSet, obj *model.StorageInfo) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, storageInfoImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("StorageInfo")
		case "usedBytes":
			out.Values[i] = ec._StorageInfo_usedBytes(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "totalBytes":
			out.Values[i] = ec._StorageInfo_totalBytes(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "freeBytes":
			out.Values[i] = ec._StorageInfo_freeBytes(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var subtitleTrackImplementors = []string{"SubtitleTrack"}

func (ec *executionContext) _SubtitleTrack(ctx context.Context, sel ast.SelectionSet, obj *model.SubtitleTrack) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, subtitleTrackImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("SubtitleTrack")
		case "lang":
			out.Values[i] = ec._SubtitleTrack_lang(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "url":
			out.Values[i] = ec._SubtitleTrack_url(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var textFilterImplementors = []string{"TextFilter", "FilterNode"}

func (ec *executionContext) _TextFilter(ctx context.Context, sel ast.SelectionSet, obj *model.TextFilter) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, textFilterImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("TextFilter")
		case "name":
			out.Values[i] = ec._TextFilter_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "state":
			out.Values[i] = ec._TextFilter_state(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var trackLinkImplementors = []string{"TrackLink"}

func (ec *executionContext) _TrackLink(ctx context.Context, sel ast.SelectionSet, obj *model.TrackLink) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, trackLinkImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("TrackLink")
		case "id":
			out.Values[i] = ec._TrackLink_id(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "mediaId":
			out.Values[i] = ec._TrackLink_mediaId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "trackerKey":
			out.Values[i] = ec._TrackLink_trackerKey(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "remoteId":
			out.Values[i] = ec._TrackLink_remoteId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "title":
			out.Values[i] = ec._TrackLink_title(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "url":
			out.Values[i] = ec._TrackLink_url(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "status":
			out.Values[i] = ec._TrackLink_status(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "statusName":
			out.Values[i] = ec._TrackLink_statusName(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "lastChapterRead":
			out.Values[i] = ec._TrackLink_lastChapterRead(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "totalChapters":
			out.Values[i] = ec._TrackLink_totalChapters(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "score":
			out.Values[i] = ec._TrackLink_score(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "startedAt":
			out.Values[i] = ec._TrackLink_startedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "finishedAt":
			out.Values[i] = ec._TrackLink_finishedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "private":
			out.Values[i] = ec._TrackLink_private(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "lastSyncedAt":
			out.Values[i] = ec._TrackLink_lastSyncedAt(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var trackSearchResultImplementors = []string{"TrackSearchResult"}

func (ec *executionContext) _TrackSearchResult(ctx context.Context, sel ast.SelectionSet, obj *model.TrackSearchResult) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, trackSearchResultImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("TrackSearchResult")
		case "remoteId":
			out.Values[i] = ec._TrackSearchResult_remoteId(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "title":
			out.Values[i] = ec._TrackSearchResult_title(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "url":
			out.Values[i] = ec._TrackSearchResult_url(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "coverUrl":
			out.Values[i] = ec._TrackSearchResult_coverUrl(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "summary":
			out.Values[i] = ec._TrackSearchResult_summary(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "totalChapters":
			out.Values[i] = ec._TrackSearchResult_totalChapters(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "publishingStatus":
			out.Values[i] = ec._TrackSearchResult_publishingStatus(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "mediaType":
			out.Values[i] = ec._TrackSearchResult_mediaType(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var trackStatusImplementors = []string{"TrackStatus"}

func (ec *executionContext) _TrackStatus(ctx context.Context, sel ast.SelectionSet, obj *model.TrackStatus) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, trackStatusImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("TrackStatus")
		case "value":
			out.Values[i] = ec._TrackStatus_value(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "name":
			out.Values[i] = ec._TrackStatus_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "animeName":
			out.Values[i] = ec._TrackStatus_animeName(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var trackerImplementors = []string{"Tracker"}

func (ec *executionContext) _Tracker(ctx context.Context, sel ast.SelectionSet, obj *model.Tracker) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, trackerImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Tracker")
		case "key":
			out.Values[i] = ec._Tracker_key(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "name":
			out.Values[i] = ec._Tracker_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "configured":
			out.Values[i] = ec._Tracker_configured(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "isLoggedIn":
			out.Values[i] = ec._Tracker_isLoggedIn(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "authUrl":
			out.Values[i] = ec._Tracker_authUrl(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "username":
			out.Values[i] = ec._Tracker_username(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "scoreOptions":
			out.Values[i] = ec._Tracker_scoreOptions(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "statusOptions":
			out.Values[i] = ec._Tracker_statusOptions(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "iconUrl":
			out.Values[i] = ec._Tracker_iconUrl(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var triStateFilterImplementors = []string{"TriStateFilter", "FilterNode"}

func (ec *executionContext) _TriStateFilter(ctx context.Context, sel ast.SelectionSet, obj *model.TriStateFilter) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, triStateFilterImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("TriStateFilter")
		case "name":
			out.Values[i] = ec._TriStateFilter_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "state":
			out.Values[i] = ec._TriStateFilter_state(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var videoSourceImplementors = []string{"VideoSource"}

func (ec *executionContext) _VideoSource(ctx context.Context, sel ast.SelectionSet, obj *model.VideoSource) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, videoSourceImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("VideoSource")
		case "label":
			out.Values[i] = ec._VideoSource_label(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "resolution":
			out.Values[i] = ec._VideoSource_resolution(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "preferred":
			out.Values[i] = ec._VideoSource_preferred(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "kind":
			out.Values[i] = ec._VideoSource_kind(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "server":
			out.Values[i] = ec._VideoSource_server(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "url":
			out.Values[i] = ec._VideoSource_url(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var videoStreamImplementors = []string{"VideoStream"}

func (ec *executionContext) _VideoStream(ctx context.Context, sel ast.SelectionSet, obj *model.VideoStream) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, videoStreamImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("VideoStream")
		case "url":
			out.Values[i] = ec._VideoStream_url(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "sources":
			out.Values[i] = ec._VideoStream_sources(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "subtitles":
			out.Values[i] = ec._VideoStream_subtitles(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "audioTracks":
			out.Values[i] = ec._VideoStream_audioTracks(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "skipMarkers":
			out.Values[i] = ec._VideoStream_skipMarkers(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var __DirectiveImplementors = []string{"__Directive"}

func (ec *executionContext) ___Directive(ctx context.Context, sel ast.SelectionSet, obj *introspection.Directive) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, __DirectiveImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("__Directive")
		case "name":
			out.Values[i] = ec.___Directive_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "description":
			out.Values[i] = ec.___Directive_description(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "isRepeatable":
			out.Values[i] = ec.___Directive_isRepeatable(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "locations":
			out.Values[i] = ec.___Directive_locations(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "args":
			out.Values[i] = ec.___Directive_args(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var __EnumValueImplementors = []string{"__EnumValue"}

func (ec *executionContext) ___EnumValue(ctx context.Context, sel ast.SelectionSet, obj *introspection.EnumValue) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, __EnumValueImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("__EnumValue")
		case "name":
			out.Values[i] = ec.___EnumValue_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "description":
			out.Values[i] = ec.___EnumValue_description(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "isDeprecated":
			out.Values[i] = ec.___EnumValue_isDeprecated(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "deprecationReason":
			out.Values[i] = ec.___EnumValue_deprecationReason(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var __FieldImplementors = []string{"__Field"}

func (ec *executionContext) ___Field(ctx context.Context, sel ast.SelectionSet, obj *introspection.Field) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, __FieldImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("__Field")
		case "name":
			out.Values[i] = ec.___Field_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "description":
			out.Values[i] = ec.___Field_description(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "args":
			out.Values[i] = ec.___Field_args(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "type":
			out.Values[i] = ec.___Field_type(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "isDeprecated":
			out.Values[i] = ec.___Field_isDeprecated(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "deprecationReason":
			out.Values[i] = ec.___Field_deprecationReason(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var __InputValueImplementors = []string{"__InputValue"}

func (ec *executionContext) ___InputValue(ctx context.Context, sel ast.SelectionSet, obj *introspection.InputValue) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, __InputValueImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("__InputValue")
		case "name":
			out.Values[i] = ec.___InputValue_name(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "description":
			out.Values[i] = ec.___InputValue_description(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "type":
			out.Values[i] = ec.___InputValue_type(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "defaultValue":
			out.Values[i] = ec.___InputValue_defaultValue(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "isDeprecated":
			out.Values[i] = ec.___InputValue_isDeprecated(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "deprecationReason":
			out.Values[i] = ec.___InputValue_deprecationReason(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var __SchemaImplementors = []string{"__Schema"}

func (ec *executionContext) ___Schema(ctx context.Context, sel ast.SelectionSet, obj *introspection.Schema) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, __SchemaImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("__Schema")
		case "description":
			out.Values[i] = ec.___Schema_description(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "types":
			out.Values[i] = ec.___Schema_types(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "queryType":
			out.Values[i] = ec.___Schema_queryType(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "mutationType":
			out.Values[i] = ec.___Schema_mutationType(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "subscriptionType":
			out.Values[i] = ec.___Schema_subscriptionType(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "directives":
			out.Values[i] = ec.___Schema_directives(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

var __TypeImplementors = []string{"__Type"}

func (ec *executionContext) ___Type(ctx context.Context, sel ast.SelectionSet, obj *introspection.Type) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, __TypeImplementors)

	out := graphql.NewFieldSet(fields)
	deferredFieldSet := graphql.NewFieldSet(nil)
	deferLabelToView := make(map[string]*graphql.FieldSetView)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("__Type")
		case "kind":
			out.Values[i] = ec.___Type_kind(ctx, field, obj)
			if out.Values[i] == graphql.Null {
				out.Invalids++
			}
		case "name":
			out.Values[i] = ec.___Type_name(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "description":
			out.Values[i] = ec.___Type_description(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "specifiedByURL":
			out.Values[i] = ec.___Type_specifiedByURL(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "fields":
			out.Values[i] = ec.___Type_fields(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "interfaces":
			out.Values[i] = ec.___Type_interfaces(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "possibleTypes":
			out.Values[i] = ec.___Type_possibleTypes(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "enumValues":
			out.Values[i] = ec.___Type_enumValues(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "inputFields":
			out.Values[i] = ec.___Type_inputFields(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "ofType":
			out.Values[i] = ec.___Type_ofType(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		case "isOneOf":
			out.Values[i] = ec.___Type_isOneOf(ctx, field, obj)
			if out.Values[i] == graphql.RequiredNull {
				out.Invalids++
			}
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.Deferred, int32(min(len(deferLabelToView), math.MaxInt32)))

	ec.ProcessDeferredGroup(graphql.DeferredGroup{
		Defers:   deferLabelToView,
		Path:     graphql.GetPath(ctx),
		FieldSet: deferredFieldSet,
		Context:  ctx,
	})

	return out
}

func (ec *executionContext) marshalNAboutServer2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐAboutServer(ctx context.Context, sel ast.SelectionSet, v model.AboutServer) graphql.Marshaler {
	return ec._AboutServer(ctx, sel, &v)
}

func (ec *executionContext) marshalNAboutServer2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐAboutServer(ctx context.Context, sel ast.SelectionSet, v *model.AboutServer) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._AboutServer(ctx, sel, v)
}

func (ec *executionContext) marshalNAudioTrack2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐAudioTrackᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.AudioTrack) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNAudioTrack2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐAudioTrack(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNAudioTrack2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐAudioTrack(ctx context.Context, sel ast.SelectionSet, v *model.AudioTrack) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._AudioTrack(ctx, sel, v)
}

func (ec *executionContext) unmarshalNBoolean2bool(ctx context.Context, v any) (bool, error) {
	res, err := graphql.UnmarshalBoolean(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalNBoolean2bool(ctx context.Context, sel ast.SelectionSet, v bool) graphql.Marshaler {
	_ = sel
	res := graphql.MarshalBoolean(v)
	if res == graphql.Null {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
	}
	return res
}

func (ec *executionContext) marshalNChapter2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐChapter(ctx context.Context, sel ast.SelectionSet, v model.Chapter) graphql.Marshaler {
	return ec._Chapter(ctx, sel, &v)
}

func (ec *executionContext) marshalNChapter2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐChapterᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.Chapter) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNChapter2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐChapter(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNChapter2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐChapter(ctx context.Context, sel ast.SelectionSet, v *model.Chapter) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._Chapter(ctx, sel, v)
}

func (ec *executionContext) unmarshalNContentType2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐContentType(ctx context.Context, v any) (model.ContentType, error) {
	var res model.ContentType
	err := res.UnmarshalGQL(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalNContentType2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐContentType(ctx context.Context, sel ast.SelectionSet, v model.ContentType) graphql.Marshaler {
	return v
}

func (ec *executionContext) marshalNDownload2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownload(ctx context.Context, sel ast.SelectionSet, v model.Download) graphql.Marshaler {
	return ec._Download(ctx, sel, &v)
}

func (ec *executionContext) marshalNDownload2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloadᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.Download) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNDownload2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownload(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNDownload2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownload(ctx context.Context, sel ast.SelectionSet, v *model.Download) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._Download(ctx, sel, v)
}

func (ec *executionContext) unmarshalNDownloadStatus2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloadStatus(ctx context.Context, v any) (model.DownloadStatus, error) {
	var res model.DownloadStatus
	err := res.UnmarshalGQL(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalNDownloadStatus2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloadStatus(ctx context.Context, sel ast.SelectionSet, v model.DownloadStatus) graphql.Marshaler {
	return v
}

func (ec *executionContext) marshalNDownloaderStatus2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloaderStatus(ctx context.Context, sel ast.SelectionSet, v model.DownloaderStatus) graphql.Marshaler {
	return ec._DownloaderStatus(ctx, sel, &v)
}

func (ec *executionContext) marshalNDownloaderStatus2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloaderStatus(ctx context.Context, sel ast.SelectionSet, v *model.DownloaderStatus) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._DownloaderStatus(ctx, sel, v)
}

func (ec *executionContext) marshalNExtension2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐExtension(ctx context.Context, sel ast.SelectionSet, v model.Extension) graphql.Marshaler {
	return ec._Extension(ctx, sel, &v)
}

func (ec *executionContext) marshalNExtension2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐExtensionᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.Extension) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNExtension2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐExtension(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNExtension2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐExtension(ctx context.Context, sel ast.SelectionSet, v *model.Extension) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._Extension(ctx, sel, v)
}

func (ec *executionContext) unmarshalNFilterInput2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFilterInputᚄ(ctx context.Context, v any) ([]*model.FilterInput, error) {
	vSlice := graphql.CoerceList(v)
	var err error
	res := make([]*model.FilterInput, len(vSlice))
	for i := range vSlice {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithIndex(i))
		res[i], err = ec.unmarshalNFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFilterInput(ctx, vSlice[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (ec *executionContext) unmarshalNFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFilterInput(ctx context.Context, v any) (*model.FilterInput, error) {
	res, err := ec.unmarshalInputFilterInput(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalNFilterNode2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFilterNode(ctx context.Context, sel ast.SelectionSet, v model.FilterNode) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._FilterNode(ctx, sel, v)
}

func (ec *executionContext) marshalNFilterNode2ᚕtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFilterNodeᚄ(ctx context.Context, sel ast.SelectionSet, v []model.FilterNode) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNFilterNode2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFilterNode(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) unmarshalNFloat2float64(ctx context.Context, v any) (float64, error) {
	res, err := graphql.UnmarshalFloatContext(ctx, v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalNFloat2float64(ctx context.Context, sel ast.SelectionSet, v float64) graphql.Marshaler {
	_ = sel
	res := graphql.MarshalFloatContext(v)
	if res == graphql.Null {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
	}
	return graphql.WrapContextMarshaler(ctx, res)
}

func (ec *executionContext) marshalNFolder2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFolder(ctx context.Context, sel ast.SelectionSet, v model.Folder) graphql.Marshaler {
	return ec._Folder(ctx, sel, &v)
}

func (ec *executionContext) marshalNFolder2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFolderᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.Folder) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNFolder2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFolder(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNFolder2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFolder(ctx context.Context, sel ast.SelectionSet, v *model.Folder) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._Folder(ctx, sel, v)
}

func (ec *executionContext) unmarshalNID2string(ctx context.Context, v any) (string, error) {
	res, err := graphql.UnmarshalID(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalNID2string(ctx context.Context, sel ast.SelectionSet, v string) graphql.Marshaler {
	_ = sel
	res := graphql.MarshalID(v)
	if res == graphql.Null {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
	}
	return res
}

func (ec *executionContext) unmarshalNID2ᚕstringᚄ(ctx context.Context, v any) ([]string, error) {
	vSlice := graphql.CoerceList(v)
	var err error
	res := make([]string, len(vSlice))
	for i := range vSlice {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithIndex(i))
		res[i], err = ec.unmarshalNID2string(ctx, vSlice[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (ec *executionContext) marshalNID2ᚕstringᚄ(ctx context.Context, sel ast.SelectionSet, v []string) graphql.Marshaler {
	ret := make(graphql.Array, len(v))
	for i := range v {
		ret[i] = ec.marshalNID2string(ctx, sel, v[i])
	}

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) unmarshalNInt2int32(ctx context.Context, v any) (int32, error) {
	res, err := graphql.UnmarshalInt32(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalNInt2int32(ctx context.Context, sel ast.SelectionSet, v int32) graphql.Marshaler {
	_ = sel
	res := graphql.MarshalInt32(v)
	if res == graphql.Null {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
	}
	return res
}

func (ec *executionContext) unmarshalNLibrarySort2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐLibrarySort(ctx context.Context, v any) (model.LibrarySort, error) {
	var res model.LibrarySort
	err := res.UnmarshalGQL(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalNLibrarySort2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐLibrarySort(ctx context.Context, sel ast.SelectionSet, v model.LibrarySort) graphql.Marshaler {
	return v
}

func (ec *executionContext) marshalNLibraryUpdateStatus2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐLibraryUpdateStatus(ctx context.Context, sel ast.SelectionSet, v model.LibraryUpdateStatus) graphql.Marshaler {
	return ec._LibraryUpdateStatus(ctx, sel, &v)
}

func (ec *executionContext) marshalNLibraryUpdateStatus2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐLibraryUpdateStatus(ctx context.Context, sel ast.SelectionSet, v *model.LibraryUpdateStatus) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._LibraryUpdateStatus(ctx, sel, v)
}

func (ec *executionContext) marshalNMedia2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx context.Context, sel ast.SelectionSet, v model.Media) graphql.Marshaler {
	return ec._Media(ctx, sel, &v)
}

func (ec *executionContext) marshalNMedia2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMediaᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.Media) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNMedia2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNMedia2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx context.Context, sel ast.SelectionSet, v *model.Media) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._Media(ctx, sel, v)
}

func (ec *executionContext) marshalNMediaPage2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMediaPage(ctx context.Context, sel ast.SelectionSet, v model.MediaPage) graphql.Marshaler {
	return ec._MediaPage(ctx, sel, &v)
}

func (ec *executionContext) marshalNMediaPage2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMediaPage(ctx context.Context, sel ast.SelectionSet, v *model.MediaPage) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._MediaPage(ctx, sel, v)
}

func (ec *executionContext) marshalNMetadataCandidate2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMetadataCandidateᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.MetadataCandidate) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNMetadataCandidate2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMetadataCandidate(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNMetadataCandidate2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMetadataCandidate(ctx context.Context, sel ast.SelectionSet, v *model.MetadataCandidate) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._MetadataCandidate(ctx, sel, v)
}

func (ec *executionContext) marshalNReadingProgress2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐReadingProgress(ctx context.Context, sel ast.SelectionSet, v model.ReadingProgress) graphql.Marshaler {
	return ec._ReadingProgress(ctx, sel, &v)
}

func (ec *executionContext) marshalNReadingProgress2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐReadingProgressᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.ReadingProgress) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNReadingProgress2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐReadingProgress(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNReadingProgress2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐReadingProgress(ctx context.Context, sel ast.SelectionSet, v *model.ReadingProgress) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._ReadingProgress(ctx, sel, v)
}

func (ec *executionContext) marshalNRecentChapter2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐRecentChapterᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.RecentChapter) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNRecentChapter2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐRecentChapter(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNRecentChapter2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐRecentChapter(ctx context.Context, sel ast.SelectionSet, v *model.RecentChapter) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._RecentChapter(ctx, sel, v)
}

func (ec *executionContext) marshalNRepository2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐRepository(ctx context.Context, sel ast.SelectionSet, v model.Repository) graphql.Marshaler {
	return ec._Repository(ctx, sel, &v)
}

func (ec *executionContext) marshalNRepository2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐRepositoryᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.Repository) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNRepository2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐRepository(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNRepository2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐRepository(ctx context.Context, sel ast.SelectionSet, v *model.Repository) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._Repository(ctx, sel, v)
}

func (ec *executionContext) marshalNSearchResponse2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSearchResponse(ctx context.Context, sel ast.SelectionSet, v model.SearchResponse) graphql.Marshaler {
	return ec._SearchResponse(ctx, sel, &v)
}

func (ec *executionContext) marshalNSearchResponse2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSearchResponse(ctx context.Context, sel ast.SelectionSet, v *model.SearchResponse) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._SearchResponse(ctx, sel, v)
}

func (ec *executionContext) marshalNSkipMarker2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSkipMarkerᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.SkipMarker) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNSkipMarker2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSkipMarker(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNSkipMarker2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSkipMarker(ctx context.Context, sel ast.SelectionSet, v *model.SkipMarker) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._SkipMarker(ctx, sel, v)
}

func (ec *executionContext) marshalNStorageInfo2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐStorageInfo(ctx context.Context, sel ast.SelectionSet, v model.StorageInfo) graphql.Marshaler {
	return ec._StorageInfo(ctx, sel, &v)
}

func (ec *executionContext) marshalNStorageInfo2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐStorageInfo(ctx context.Context, sel ast.SelectionSet, v *model.StorageInfo) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._StorageInfo(ctx, sel, v)
}

func (ec *executionContext) unmarshalNString2string(ctx context.Context, v any) (string, error) {
	res, err := graphql.UnmarshalString(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalNString2string(ctx context.Context, sel ast.SelectionSet, v string) graphql.Marshaler {
	_ = sel
	res := graphql.MarshalString(v)
	if res == graphql.Null {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
	}
	return res
}

func (ec *executionContext) unmarshalNString2ᚕstringᚄ(ctx context.Context, v any) ([]string, error) {
	vSlice := graphql.CoerceList(v)
	var err error
	res := make([]string, len(vSlice))
	for i := range vSlice {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithIndex(i))
		res[i], err = ec.unmarshalNString2string(ctx, vSlice[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (ec *executionContext) marshalNString2ᚕstringᚄ(ctx context.Context, sel ast.SelectionSet, v []string) graphql.Marshaler {
	ret := make(graphql.Array, len(v))
	for i := range v {
		ret[i] = ec.marshalNString2string(ctx, sel, v[i])
	}

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNSubtitleTrack2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSubtitleTrackᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.SubtitleTrack) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNSubtitleTrack2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSubtitleTrack(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNSubtitleTrack2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSubtitleTrack(ctx context.Context, sel ast.SelectionSet, v *model.SubtitleTrack) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._SubtitleTrack(ctx, sel, v)
}

func (ec *executionContext) unmarshalNTime2timeᚐTime(ctx context.Context, v any) (time.Time, error) {
	res, err := graphql.UnmarshalTime(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalNTime2timeᚐTime(ctx context.Context, sel ast.SelectionSet, v time.Time) graphql.Marshaler {
	_ = sel
	res := graphql.MarshalTime(v)
	if res == graphql.Null {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
	}
	return res
}

func (ec *executionContext) marshalNTrackLink2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackLink(ctx context.Context, sel ast.SelectionSet, v model.TrackLink) graphql.Marshaler {
	return ec._TrackLink(ctx, sel, &v)
}

func (ec *executionContext) marshalNTrackLink2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackLinkᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.TrackLink) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNTrackLink2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackLink(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNTrackLink2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackLink(ctx context.Context, sel ast.SelectionSet, v *model.TrackLink) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._TrackLink(ctx, sel, v)
}

func (ec *executionContext) marshalNTrackSearchResult2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackSearchResultᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.TrackSearchResult) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNTrackSearchResult2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackSearchResult(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNTrackSearchResult2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackSearchResult(ctx context.Context, sel ast.SelectionSet, v *model.TrackSearchResult) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._TrackSearchResult(ctx, sel, v)
}

func (ec *executionContext) marshalNTrackStatus2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackStatusᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.TrackStatus) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNTrackStatus2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackStatus(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNTrackStatus2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackStatus(ctx context.Context, sel ast.SelectionSet, v *model.TrackStatus) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._TrackStatus(ctx, sel, v)
}

func (ec *executionContext) marshalNTracker2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTracker(ctx context.Context, sel ast.SelectionSet, v model.Tracker) graphql.Marshaler {
	return ec._Tracker(ctx, sel, &v)
}

func (ec *executionContext) marshalNTracker2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTrackerᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.Tracker) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNTracker2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTracker(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNTracker2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTracker(ctx context.Context, sel ast.SelectionSet, v *model.Tracker) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._Tracker(ctx, sel, v)
}

func (ec *executionContext) marshalNVideoSource2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐVideoSourceᚄ(ctx context.Context, sel ast.SelectionSet, v []*model.VideoSource) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNVideoSource2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐVideoSource(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalNVideoSource2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐVideoSource(ctx context.Context, sel ast.SelectionSet, v *model.VideoSource) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec._VideoSource(ctx, sel, v)
}

func (ec *executionContext) marshalN__Directive2githubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐDirective(ctx context.Context, sel ast.SelectionSet, v introspection.Directive) graphql.Marshaler {
	return ec.___Directive(ctx, sel, &v)
}

func (ec *executionContext) marshalN__Directive2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐDirectiveᚄ(ctx context.Context, sel ast.SelectionSet, v []introspection.Directive) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalN__Directive2githubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐDirective(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) unmarshalN__DirectiveLocation2string(ctx context.Context, v any) (string, error) {
	res, err := graphql.UnmarshalString(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalN__DirectiveLocation2string(ctx context.Context, sel ast.SelectionSet, v string) graphql.Marshaler {
	_ = sel
	res := graphql.MarshalString(v)
	if res == graphql.Null {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
	}
	return res
}

func (ec *executionContext) unmarshalN__DirectiveLocation2ᚕstringᚄ(ctx context.Context, v any) ([]string, error) {
	vSlice := graphql.CoerceList(v)
	var err error
	res := make([]string, len(vSlice))
	for i := range vSlice {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithIndex(i))
		res[i], err = ec.unmarshalN__DirectiveLocation2string(ctx, vSlice[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (ec *executionContext) marshalN__DirectiveLocation2ᚕstringᚄ(ctx context.Context, sel ast.SelectionSet, v []string) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalN__DirectiveLocation2string(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalN__EnumValue2githubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐEnumValue(ctx context.Context, sel ast.SelectionSet, v introspection.EnumValue) graphql.Marshaler {
	return ec.___EnumValue(ctx, sel, &v)
}

func (ec *executionContext) marshalN__Field2githubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐField(ctx context.Context, sel ast.SelectionSet, v introspection.Field) graphql.Marshaler {
	return ec.___Field(ctx, sel, &v)
}

func (ec *executionContext) marshalN__InputValue2githubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐInputValue(ctx context.Context, sel ast.SelectionSet, v introspection.InputValue) graphql.Marshaler {
	return ec.___InputValue(ctx, sel, &v)
}

func (ec *executionContext) marshalN__InputValue2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐInputValueᚄ(ctx context.Context, sel ast.SelectionSet, v []introspection.InputValue) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalN__InputValue2githubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐInputValue(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalN__Type2githubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐType(ctx context.Context, sel ast.SelectionSet, v introspection.Type) graphql.Marshaler {
	return ec.___Type(ctx, sel, &v)
}

func (ec *executionContext) marshalN__Type2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐTypeᚄ(ctx context.Context, sel ast.SelectionSet, v []introspection.Type) graphql.Marshaler {
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalN__Type2githubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐType(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalN__Type2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐType(ctx context.Context, sel ast.SelectionSet, v *introspection.Type) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	return ec.___Type(ctx, sel, v)
}

func (ec *executionContext) unmarshalN__TypeKind2string(ctx context.Context, v any) (string, error) {
	res, err := graphql.UnmarshalString(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalN__TypeKind2string(ctx context.Context, sel ast.SelectionSet, v string) graphql.Marshaler {
	_ = sel
	res := graphql.MarshalString(v)
	if res == graphql.Null {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
	}
	return res
}

func (ec *executionContext) unmarshalOBoolean2bool(ctx context.Context, v any) (bool, error) {
	res, err := graphql.UnmarshalBoolean(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOBoolean2bool(ctx context.Context, sel ast.SelectionSet, v bool) graphql.Marshaler {
	_ = sel
	_ = ctx
	res := graphql.MarshalBoolean(v)
	return res
}

func (ec *executionContext) unmarshalOBoolean2ᚖbool(ctx context.Context, v any) (*bool, error) {
	if v == nil {
		return nil, nil
	}
	res, err := graphql.UnmarshalBoolean(v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOBoolean2ᚖbool(ctx context.Context, sel ast.SelectionSet, v *bool) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	_ = sel
	_ = ctx
	res := graphql.MarshalBoolean(*v)
	return res
}

func (ec *executionContext) marshalOChapter2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐChapter(ctx context.Context, sel ast.SelectionSet, v *model.Chapter) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec._Chapter(ctx, sel, v)
}

func (ec *executionContext) unmarshalOCheckBoxFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐCheckBoxFilterInput(ctx context.Context, v any) (*model.CheckBoxFilterInput, error) {
	if v == nil {
		return nil, nil
	}
	res, err := ec.unmarshalInputCheckBoxFilterInput(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) unmarshalOContentType2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐContentType(ctx context.Context, v any) (*model.ContentType, error) {
	if v == nil {
		return nil, nil
	}
	var res = new(model.ContentType)
	err := res.UnmarshalGQL(v)
	return res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOContentType2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐContentType(ctx context.Context, sel ast.SelectionSet, v *model.ContentType) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return v
}

func (ec *executionContext) marshalODownload2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownload(ctx context.Context, sel ast.SelectionSet, v *model.Download) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec._Download(ctx, sel, v)
}

func (ec *executionContext) unmarshalODownloadStatus2ᚕtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloadStatusᚄ(ctx context.Context, v any) ([]model.DownloadStatus, error) {
	if v == nil {
		return nil, nil
	}
	vSlice := graphql.CoerceList(v)
	var err error
	res := make([]model.DownloadStatus, len(vSlice))
	for i := range vSlice {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithIndex(i))
		res[i], err = ec.unmarshalNDownloadStatus2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloadStatus(ctx, vSlice[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (ec *executionContext) marshalODownloadStatus2ᚕtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloadStatusᚄ(ctx context.Context, sel ast.SelectionSet, v []model.DownloadStatus) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalNDownloadStatus2tsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐDownloadStatus(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalOExtension2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐExtension(ctx context.Context, sel ast.SelectionSet, v *model.Extension) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec._Extension(ctx, sel, v)
}

func (ec *executionContext) unmarshalOFilterInput2ᚕᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFilterInputᚄ(ctx context.Context, v any) ([]*model.FilterInput, error) {
	if v == nil {
		return nil, nil
	}
	vSlice := graphql.CoerceList(v)
	var err error
	res := make([]*model.FilterInput, len(vSlice))
	for i := range vSlice {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithIndex(i))
		res[i], err = ec.unmarshalNFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFilterInput(ctx, vSlice[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (ec *executionContext) unmarshalOFloat2ᚖfloat64(ctx context.Context, v any) (*float64, error) {
	if v == nil {
		return nil, nil
	}
	res, err := graphql.UnmarshalFloatContext(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOFloat2ᚖfloat64(ctx context.Context, sel ast.SelectionSet, v *float64) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	_ = sel
	res := graphql.MarshalFloatContext(*v)
	return graphql.WrapContextMarshaler(ctx, res)
}

func (ec *executionContext) marshalOFolder2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐFolder(ctx context.Context, sel ast.SelectionSet, v *model.Folder) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec._Folder(ctx, sel, v)
}

func (ec *executionContext) unmarshalOGroupFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐGroupFilterInput(ctx context.Context, v any) (*model.GroupFilterInput, error) {
	if v == nil {
		return nil, nil
	}
	res, err := ec.unmarshalInputGroupFilterInput(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) unmarshalOID2ᚕstringᚄ(ctx context.Context, v any) ([]string, error) {
	if v == nil {
		return nil, nil
	}
	vSlice := graphql.CoerceList(v)
	var err error
	res := make([]string, len(vSlice))
	for i := range vSlice {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithIndex(i))
		res[i], err = ec.unmarshalNID2string(ctx, vSlice[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (ec *executionContext) marshalOID2ᚕstringᚄ(ctx context.Context, sel ast.SelectionSet, v []string) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	ret := make(graphql.Array, len(v))
	for i := range v {
		ret[i] = ec.marshalNID2string(ctx, sel, v[i])
	}

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) unmarshalOID2ᚖstring(ctx context.Context, v any) (*string, error) {
	if v == nil {
		return nil, nil
	}
	res, err := graphql.UnmarshalID(v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOID2ᚖstring(ctx context.Context, sel ast.SelectionSet, v *string) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	_ = sel
	_ = ctx
	res := graphql.MarshalID(*v)
	return res
}

func (ec *executionContext) unmarshalOInt2ᚖint32(ctx context.Context, v any) (*int32, error) {
	if v == nil {
		return nil, nil
	}
	res, err := graphql.UnmarshalInt32(v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOInt2ᚖint32(ctx context.Context, sel ast.SelectionSet, v *int32) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	_ = sel
	_ = ctx
	res := graphql.MarshalInt32(*v)
	return res
}

func (ec *executionContext) unmarshalOLibraryFilter2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐLibraryFilter(ctx context.Context, v any) (*model.LibraryFilter, error) {
	if v == nil {
		return nil, nil
	}
	res, err := ec.unmarshalInputLibraryFilter(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) unmarshalOLibrarySortInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐLibrarySortInput(ctx context.Context, v any) (*model.LibrarySortInput, error) {
	if v == nil {
		return nil, nil
	}
	res, err := ec.unmarshalInputLibrarySortInput(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOMedia2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMedia(ctx context.Context, sel ast.SelectionSet, v *model.Media) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec._Media(ctx, sel, v)
}

func (ec *executionContext) marshalOMetadataMatch2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐMetadataMatch(ctx context.Context, sel ast.SelectionSet, v *model.MetadataMatch) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec._MetadataMatch(ctx, sel, v)
}

func (ec *executionContext) marshalOReadingProgress2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐReadingProgress(ctx context.Context, sel ast.SelectionSet, v *model.ReadingProgress) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec._ReadingProgress(ctx, sel, v)
}

func (ec *executionContext) unmarshalOSelectFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSelectFilterInput(ctx context.Context, v any) (*model.SelectFilterInput, error) {
	if v == nil {
		return nil, nil
	}
	res, err := ec.unmarshalInputSelectFilterInput(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) unmarshalOSortFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐSortFilterInput(ctx context.Context, v any) (*model.SortFilterInput, error) {
	if v == nil {
		return nil, nil
	}
	res, err := ec.unmarshalInputSortFilterInput(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) unmarshalOString2ᚕstringᚄ(ctx context.Context, v any) ([]string, error) {
	if v == nil {
		return nil, nil
	}
	vSlice := graphql.CoerceList(v)
	var err error
	res := make([]string, len(vSlice))
	for i := range vSlice {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithIndex(i))
		res[i], err = ec.unmarshalNString2string(ctx, vSlice[i])
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (ec *executionContext) marshalOString2ᚕstringᚄ(ctx context.Context, sel ast.SelectionSet, v []string) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	ret := make(graphql.Array, len(v))
	for i := range v {
		ret[i] = ec.marshalNString2string(ctx, sel, v[i])
	}

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) unmarshalOString2ᚖstring(ctx context.Context, v any) (*string, error) {
	if v == nil {
		return nil, nil
	}
	res, err := graphql.UnmarshalString(v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOString2ᚖstring(ctx context.Context, sel ast.SelectionSet, v *string) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	_ = sel
	_ = ctx
	res := graphql.MarshalString(*v)
	return res
}

func (ec *executionContext) unmarshalOTextFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTextFilterInput(ctx context.Context, v any) (*model.TextFilterInput, error) {
	if v == nil {
		return nil, nil
	}
	res, err := ec.unmarshalInputTextFilterInput(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) unmarshalOTime2ᚖtimeᚐTime(ctx context.Context, v any) (*time.Time, error) {
	if v == nil {
		return nil, nil
	}
	res, err := graphql.UnmarshalTime(v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOTime2ᚖtimeᚐTime(ctx context.Context, sel ast.SelectionSet, v *time.Time) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	_ = sel
	_ = ctx
	res := graphql.MarshalTime(*v)
	return res
}

func (ec *executionContext) unmarshalOTriStateFilterInput2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐTriStateFilterInput(ctx context.Context, v any) (*model.TriStateFilterInput, error) {
	if v == nil {
		return nil, nil
	}
	res, err := ec.unmarshalInputTriStateFilterInput(ctx, v)
	return &res, graphql.ErrorOnPath(ctx, err)
}

func (ec *executionContext) marshalOVideoStream2ᚖtsunaguᚋbackendᚋinternalᚋapiᚋgraphᚋmodelᚐVideoStream(ctx context.Context, sel ast.SelectionSet, v *model.VideoStream) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec._VideoStream(ctx, sel, v)
}

func (ec *executionContext) marshalO__EnumValue2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐEnumValueᚄ(ctx context.Context, sel ast.SelectionSet, v []introspection.EnumValue) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalN__EnumValue2githubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐEnumValue(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalO__Field2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐFieldᚄ(ctx context.Context, sel ast.SelectionSet, v []introspection.Field) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalN__Field2githubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐField(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalO__InputValue2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐInputValueᚄ(ctx context.Context, sel ast.SelectionSet, v []introspection.InputValue) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalN__InputValue2githubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐInputValue(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalO__Schema2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐSchema(ctx context.Context, sel ast.SelectionSet, v *introspection.Schema) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec.___Schema(ctx, sel, v)
}

func (ec *executionContext) marshalO__Type2ᚕgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐTypeᚄ(ctx context.Context, sel ast.SelectionSet, v []introspection.Type) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	ret := graphql.MarshalSliceConcurrently(ctx, len(v), 0, false, func(ctx context.Context, i int) graphql.Marshaler {
		fc := graphql.GetFieldContext(ctx)
		fc.Result = &v[i]
		return ec.marshalN__Type2githubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐType(ctx, sel, v[i])
	})

	for _, e := range ret {
		if e == graphql.Null {
			return graphql.Null
		}
	}

	return ret
}

func (ec *executionContext) marshalO__Type2ᚖgithubᚗcomᚋ99designsᚋgqlgenᚋgraphqlᚋintrospectionᚐType(ctx context.Context, sel ast.SelectionSet, v *introspection.Type) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec.___Type(ctx, sel, v)
}
