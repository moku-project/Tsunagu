package model

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"time"
)

type FilterNode interface {
	IsFilterNode()
}

type AboutServer struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	BuildTime string `json:"buildTime"`
}

type AudioTrack struct {
	Lang string `json:"lang"`
	URL  string `json:"url"`
}

type Chapter struct {
	ID              string           `json:"id"`
	MediaID         string           `json:"mediaId"`
	ExternalID      string           `json:"externalId"`
	Title           *string          `json:"title,omitempty"`
	Number          *float64         `json:"number,omitempty"`
	SourceOrder     *int32           `json:"sourceOrder,omitempty"`
	Scanlator       *string          `json:"scanlator,omitempty"`
	UploadedAt      *time.Time       `json:"uploadedAt,omitempty"`
	ReadingProgress *ReadingProgress `json:"readingProgress,omitempty"`
	Completed       bool             `json:"completed"`
	Downloaded      bool             `json:"downloaded"`
	Download        *Download        `json:"download,omitempty"`
	Pages           []string         `json:"pages,omitempty"`
	PageCount       *int32           `json:"pageCount,omitempty"`
	VideoURL        *string          `json:"videoUrl,omitempty"`
	VideoStream     *VideoStream     `json:"videoStream,omitempty"`
}

type CheckBoxFilter struct {
	Name  string `json:"name"`
	State bool   `json:"state"`
}

func (CheckBoxFilter) IsFilterNode() {}

type CheckBoxFilterInput struct {
	State bool `json:"state"`
}

type Download struct {
	ID              string         `json:"id"`
	MediaID         string         `json:"mediaId"`
	ChapterID       string         `json:"chapterId"`
	Chapter         *Chapter       `json:"chapter"`
	Status          DownloadStatus `json:"status"`
	Progress        float64        `json:"progress"`
	DownloadedBytes *float64       `json:"downloadedBytes,omitempty"`
	BytesPerSec     *float64       `json:"bytesPerSec,omitempty"`
	FinalSizeBytes  *float64       `json:"finalSizeBytes,omitempty"`
	Error           *string        `json:"error,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	CompletedAt     *time.Time     `json:"completedAt,omitempty"`
}

type DownloaderStatus struct {
	IsRunning        bool  `json:"isRunning"`
	QueuedCount      int32 `json:"queuedCount"`
	DownloadingCount int32 `json:"downloadingCount"`
	FailedCount      int32 `json:"failedCount"`
}

type Extension struct {
	ID               string      `json:"id"`
	RepositoryID     string      `json:"repositoryId"`
	PackageName      string      `json:"packageName"`
	Name             string      `json:"name"`
	Version          string      `json:"version"`
	ContentType      ContentType `json:"contentType"`
	Lang             string      `json:"lang"`
	IconURL          *string     `json:"iconUrl,omitempty"`
	ApkURL           *string     `json:"apkUrl,omitempty"`
	JarURL           *string     `json:"jarUrl,omitempty"`
	JarPath          *string     `json:"jarPath,omitempty"`
	Installed        bool        `json:"installed"`
	Enabled          bool        `json:"enabled"`
	DiscoveredAt     time.Time   `json:"discoveredAt"`
	InstalledAt      *time.Time  `json:"installedAt,omitempty"`
	InstalledVersion *string     `json:"installedVersion,omitempty"`
	NeedsUpdate      *bool       `json:"needsUpdate,omitempty"`
	IsNsfw           bool        `json:"isNsfw"`
	DisplayName      string      `json:"displayName"`
	SupportsLatest   bool        `json:"supportsLatest"`
}

type FilterInput struct {
	Name     string               `json:"name"`
	Select   *SelectFilterInput   `json:"select,omitempty"`
	Text     *TextFilterInput     `json:"text,omitempty"`
	Checkbox *CheckBoxFilterInput `json:"checkbox,omitempty"`
	Tristate *TriStateFilterInput `json:"tristate,omitempty"`
	Group    *GroupFilterInput    `json:"group,omitempty"`
	Sort     *SortFilterInput     `json:"sort,omitempty"`
}

type Folder struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Kind              string  `json:"kind"`
	SystemKey         *string `json:"systemKey,omitempty"`
	ParentFolderID    *string `json:"parentFolderId,omitempty"`
	SortOrder         int32   `json:"sortOrder"`
	IncludeInUpdate   bool    `json:"includeInUpdate"`
	IncludeInDownload bool    `json:"includeInDownload"`
}

type GroupFilter struct {
	Name     string       `json:"name"`
	Children []FilterNode `json:"children"`
}

func (GroupFilter) IsFilterNode() {}

type GroupFilterInput struct {
	Children []*FilterInput `json:"children"`
}

type HeaderFilter struct {
	Name string `json:"name"`
}

func (HeaderFilter) IsFilterNode() {}

type LibraryFilter struct {
	ContentType    *ContentType `json:"contentType,omitempty"`
	InLibrary      *bool        `json:"inLibrary,omitempty"`
	UnreadOnly     *bool        `json:"unreadOnly,omitempty"`
	DownloadedOnly *bool        `json:"downloadedOnly,omitempty"`
	TagIds         []string     `json:"tagIds,omitempty"`
	FolderID       *string      `json:"folderId,omitempty"`
	Query          *string      `json:"query,omitempty"`
}

type LibrarySortInput struct {
	By        LibrarySort `json:"by"`
	Ascending bool        `json:"ascending"`
}

type LibraryUpdateStatus struct {
	Running         bool       `json:"running"`
	Total           int32      `json:"total"`
	Done            int32      `json:"done"`
	CurrentTitle    *string    `json:"currentTitle,omitempty"`
	NewChapterCount int32      `json:"newChapterCount"`
	FailedTitles    []string   `json:"failedTitles"`
	StartedAt       *time.Time `json:"startedAt,omitempty"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty"`
}

type Media struct {
	ID                 string             `json:"id"`
	ExtensionID        *string            `json:"extensionId,omitempty"`
	ExtensionName      string             `json:"extensionName"`
	SourceName         string             `json:"sourceName"`
	ExternalID         string             `json:"externalId"`
	ContentType        ContentType        `json:"contentType"`
	Title              string             `json:"title"`
	ThumbnailURL       *string            `json:"thumbnailUrl,omitempty"`
	Description        *string            `json:"description,omitempty"`
	Status             *string            `json:"status,omitempty"`
	Author             *string            `json:"author,omitempty"`
	Artist             *string            `json:"artist,omitempty"`
	DetailsFetchedAt   *time.Time         `json:"detailsFetchedAt,omitempty"`
	ExtensionRemovedAt *time.Time         `json:"extensionRemovedAt,omitempty"`
	AddedAt            *time.Time         `json:"addedAt,omitempty"`
	LastViewedAt       *time.Time         `json:"lastViewedAt,omitempty"`
	InLibrary          bool               `json:"inLibrary"`
	Chapters           []*Chapter         `json:"chapters"`
	ChapterCount       int32              `json:"chapterCount"`
	UnreadCount        int32              `json:"unreadCount"`
	DownloadedCount    int32              `json:"downloadedCount"`
	NextUnreadChapter  *Chapter           `json:"nextUnreadChapter,omitempty"`
	LatestChapter      *Chapter           `json:"latestChapter,omitempty"`
	ReadingProgress    []*ReadingProgress `json:"readingProgress"`
	Tags               []string           `json:"tags"`
	Genres             []string           `json:"genres"`
	Folders            []*Folder          `json:"folders"`
	TrackLinks         []*TrackLink       `json:"trackLinks"`
	Metadata           *MetadataMatch     `json:"metadata,omitempty"`
	Source             *Extension         `json:"source,omitempty"`
}

type MediaPage struct {
	Items   []*Media `json:"items"`
	Total   int32    `json:"total"`
	HasMore bool     `json:"hasMore"`
}

type MetadataCandidate struct {
	Provider    string   `json:"provider"`
	ProviderID  string   `json:"providerId"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	CoverURL    *string  `json:"coverUrl,omitempty"`
	Description *string  `json:"description,omitempty"`
	Status      *string  `json:"status,omitempty"`
	Genres      []string `json:"genres"`
	StartYear   *int32   `json:"startYear,omitempty"`
}

type MetadataMatch struct {
	Provider   string    `json:"provider"`
	ProviderID string    `json:"providerId"`
	URL        string    `json:"url"`
	Confidence float64   `json:"confidence"`
	Locked     bool      `json:"locked"`
	MatchedAt  time.Time `json:"matchedAt"`
}

type Mutation struct {
}

type Query struct {
}

type ReadingProgress struct {
	ID              string    `json:"id"`
	MediaID         string    `json:"mediaId"`
	ChapterID       string    `json:"chapterId"`
	Progress        float64   `json:"progress"`
	Completed       bool      `json:"completed"`
	PositionSeconds *float64  `json:"positionSeconds,omitempty"`
	DurationSeconds *float64  `json:"durationSeconds,omitempty"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type RecentChapter struct {
	Chapter *Chapter `json:"chapter"`
	Media   *Media   `json:"media"`
}

type Repository struct {
	ID           string      `json:"id"`
	IndexURL     string      `json:"indexUrl"`
	Name         *string     `json:"name,omitempty"`
	ContentType  ContentType `json:"contentType"`
	AddedAt      time.Time   `json:"addedAt"`
	LastSyncedAt *time.Time  `json:"lastSyncedAt,omitempty"`
}

type SearchResponse struct {
	Results     []*Media `json:"results"`
	HasNextPage bool     `json:"hasNextPage"`
}

type SelectFilter struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
	State  int32    `json:"state"`
}

func (SelectFilter) IsFilterNode() {}

type SelectFilterInput struct {
	State int32 `json:"state"`
}

type SeparatorFilter struct {
	Name string `json:"name"`
}

func (SeparatorFilter) IsFilterNode() {}

type SkipMarker struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	StartMs int32  `json:"startMs"`
	EndMs   int32  `json:"endMs"`
}

type SortFilter struct {
	Name      string   `json:"name"`
	Values    []string `json:"values"`
	HasState  bool     `json:"hasState"`
	Index     *int32   `json:"index,omitempty"`
	Ascending *bool    `json:"ascending,omitempty"`
}

func (SortFilter) IsFilterNode() {}

type SortFilterInput struct {
	HasState  bool   `json:"hasState"`
	Index     *int32 `json:"index,omitempty"`
	Ascending *bool  `json:"ascending,omitempty"`
}

type StorageInfo struct {
	UsedBytes  float64 `json:"usedBytes"`
	TotalBytes float64 `json:"totalBytes"`
	FreeBytes  float64 `json:"freeBytes"`
}

type SubtitleTrack struct {
	Lang string `json:"lang"`
	URL  string `json:"url"`
}

type TextFilter struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

func (TextFilter) IsFilterNode() {}

type TextFilterInput struct {
	State string `json:"state"`
}

type TrackLink struct {
	ID              string     `json:"id"`
	MediaID         string     `json:"mediaId"`
	TrackerKey      string     `json:"trackerKey"`
	RemoteID        string     `json:"remoteId"`
	Title           string     `json:"title"`
	URL             string     `json:"url"`
	Status          int32      `json:"status"`
	StatusName      string     `json:"statusName"`
	LastChapterRead float64    `json:"lastChapterRead"`
	TotalChapters   int32      `json:"totalChapters"`
	Score           float64    `json:"score"`
	StartedAt       *time.Time `json:"startedAt,omitempty"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty"`
	Private         bool       `json:"private"`
	LastSyncedAt    *time.Time `json:"lastSyncedAt,omitempty"`
}

type TrackSearchResult struct {
	RemoteID         string  `json:"remoteId"`
	Title            string  `json:"title"`
	URL              string  `json:"url"`
	CoverURL         *string `json:"coverUrl,omitempty"`
	Summary          *string `json:"summary,omitempty"`
	TotalChapters    *int32  `json:"totalChapters,omitempty"`
	PublishingStatus *string `json:"publishingStatus,omitempty"`
	MediaType        *string `json:"mediaType,omitempty"`
}

type TrackStatus struct {
	Value     int32  `json:"value"`
	Name      string `json:"name"`
	AnimeName string `json:"animeName"`
}

type Tracker struct {
	Key           string         `json:"key"`
	Name          string         `json:"name"`
	Configured    bool           `json:"configured"`
	IsLoggedIn    bool           `json:"isLoggedIn"`
	AuthURL       *string        `json:"authUrl,omitempty"`
	Username      *string        `json:"username,omitempty"`
	ScoreOptions  []string       `json:"scoreOptions"`
	StatusOptions []*TrackStatus `json:"statusOptions"`
	IconURL       *string        `json:"iconUrl,omitempty"`
}

type TriStateFilter struct {
	Name  string `json:"name"`
	State int32  `json:"state"`
}

func (TriStateFilter) IsFilterNode() {}

type TriStateFilterInput struct {
	State int32 `json:"state"`
}

type VideoSource struct {
	Label      string `json:"label"`
	Resolution *int32 `json:"resolution,omitempty"`
	Preferred  bool   `json:"preferred"`
	Kind       string `json:"kind"`
	Server     string `json:"server"`
	URL        string `json:"url"`
}

type VideoStream struct {
	URL         string           `json:"url"`
	Sources     []*VideoSource   `json:"sources"`
	Subtitles   []*SubtitleTrack `json:"subtitles"`
	AudioTracks []*AudioTrack    `json:"audioTracks"`
	SkipMarkers []*SkipMarker    `json:"skipMarkers"`
}

type ContentType string

const (
	ContentTypeNovel ContentType = "NOVEL"
	ContentTypeManga ContentType = "MANGA"
	ContentTypeAnime ContentType = "ANIME"
)

var AllContentType = []ContentType{
	ContentTypeNovel,
	ContentTypeManga,
	ContentTypeAnime,
}

func (e ContentType) IsValid() bool {
	switch e {
	case ContentTypeNovel, ContentTypeManga, ContentTypeAnime:
		return true
	}
	return false
}

func (e ContentType) String() string {
	return string(e)
}

func (e *ContentType) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = ContentType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid ContentType", str)
	}
	return nil
}

func (e ContentType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

func (e *ContentType) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	return e.UnmarshalGQL(s)
}

func (e ContentType) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	e.MarshalGQL(&buf)
	return buf.Bytes(), nil
}

type DownloadStatus string

const (
	DownloadStatusQueued      DownloadStatus = "QUEUED"
	DownloadStatusDownloading DownloadStatus = "DOWNLOADING"
	DownloadStatusDone        DownloadStatus = "DONE"
	DownloadStatusFailed      DownloadStatus = "FAILED"
)

var AllDownloadStatus = []DownloadStatus{
	DownloadStatusQueued,
	DownloadStatusDownloading,
	DownloadStatusDone,
	DownloadStatusFailed,
}

func (e DownloadStatus) IsValid() bool {
	switch e {
	case DownloadStatusQueued, DownloadStatusDownloading, DownloadStatusDone, DownloadStatusFailed:
		return true
	}
	return false
}

func (e DownloadStatus) String() string {
	return string(e)
}

func (e *DownloadStatus) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = DownloadStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid DownloadStatus", str)
	}
	return nil
}

func (e DownloadStatus) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

func (e *DownloadStatus) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	return e.UnmarshalGQL(s)
}

func (e DownloadStatus) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	e.MarshalGQL(&buf)
	return buf.Bytes(), nil
}

type LibrarySort string

const (
	LibrarySortTitle           LibrarySort = "TITLE"
	LibrarySortAddedAt         LibrarySort = "ADDED_AT"
	LibrarySortLastReadAt      LibrarySort = "LAST_READ_AT"
	LibrarySortLatestChapterAt LibrarySort = "LATEST_CHAPTER_AT"
	LibrarySortUnreadCount     LibrarySort = "UNREAD_COUNT"
)

var AllLibrarySort = []LibrarySort{
	LibrarySortTitle,
	LibrarySortAddedAt,
	LibrarySortLastReadAt,
	LibrarySortLatestChapterAt,
	LibrarySortUnreadCount,
}

func (e LibrarySort) IsValid() bool {
	switch e {
	case LibrarySortTitle, LibrarySortAddedAt, LibrarySortLastReadAt, LibrarySortLatestChapterAt, LibrarySortUnreadCount:
		return true
	}
	return false
}

func (e LibrarySort) String() string {
	return string(e)
}

func (e *LibrarySort) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = LibrarySort(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid LibrarySort", str)
	}
	return nil
}

func (e LibrarySort) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

func (e *LibrarySort) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	return e.UnmarshalGQL(s)
}

func (e LibrarySort) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	e.MarshalGQL(&buf)
	return buf.Bytes(), nil
}
