package tracker

import (
	"context"
	"errors"
	"net/url"
	"time"
)

var ErrReauth = errors.New("tracker: re-authentication required")

type Status int

const (
	StatusReading Status = iota + 1
	StatusPlanToRead
	StatusCompleted
	StatusOnHold
	StatusDropped
	StatusRereading
)

func (s Status) String() string {
	switch s {
	case StatusReading:
		return "Reading"
	case StatusPlanToRead:
		return "Plan to read"
	case StatusCompleted:
		return "Completed"
	case StatusOnHold:
		return "On hold"
	case StatusDropped:
		return "Dropped"
	case StatusRereading:
		return "Rereading"
	default:
		return "Unknown"
	}
}

func (s Status) AnimeString() string {
	switch s {
	case StatusReading:
		return "Watching"
	case StatusPlanToRead:
		return "Plan to watch"
	case StatusRereading:
		return "Rewatching"
	default:
		return s.String()
	}
}

var AllStatuses = []Status{
	StatusReading, StatusPlanToRead, StatusCompleted,
	StatusOnHold, StatusDropped, StatusRereading,
}

type Auth struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
	Username     string
	ScoreFormat  string
}

type Track struct {
	RemoteID        string
	LibraryID       string
	Title           string
	URL             string
	Status          Status
	LastChapterRead float64
	TotalChapters   int
	Score           float64
	StartedAt       *time.Time
	FinishedAt      *time.Time
	Private         bool
}

type SearchResult struct {
	RemoteID         string
	Title            string
	URL              string
	CoverURL         string
	Summary          string
	TotalChapters    int
	PublishingStatus string
	MediaType        string
}

type Service interface {
	Key() string
	Name() string
	Configured() bool
	IconURL() string

	AuthURL() string

	Exchange(ctx context.Context, pasted string) (Auth, error)

	Search(ctx context.Context, a Auth, query, contentType string) ([]SearchResult, error)

	Bind(ctx context.Context, a Auth, remoteID string) (Track, error)

	Push(ctx context.Context, a Auth, t Track) (Track, error)

	Refresh(ctx context.Context, a Auth, t Track) (Track, error)

	ScoreOptions(a Auth) []string
}

type CallbackAuther interface {
	CompleteAuth(ctx context.Context, q url.Values) (Auth, error)
}

type Refresher interface {
	RefreshAuth(ctx context.Context, a Auth) (Auth, error)
}
