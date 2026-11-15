package embedded

import (
	"context"
	"encoding/json"

	"tsunagu/backend/internal/download"
)

// VideoObserver lets a native engine report progress and notice cancellation.
// Engines should check Cancelled from their I/O callbacks, not only between
// packets, so a cancelled download stops promptly.
type VideoObserver interface {
	Cancelled() bool
	Progress(fraction float64, bytes int64, bytesPerSecond float64)
}

// VideoEngine downloads an episode in the app's process, for platforms that
// cannot run ffmpeg as a program (iOS). headersJSON is a JSON object of HTTP
// headers the stream must be fetched with. Process must write a complete MP4
// to outputPath and return an error for anything less.
type VideoEngine interface {
	Process(url, headersJSON, outputPath string, observer VideoObserver) error
}

// SetVideoEngine routes episode downloads through engine. Takes effect on
// the next Start; pass nil to go back to ffmpeg.
func SetVideoEngine(engine VideoEngine) {
	mu.Lock()
	defer mu.Unlock()
	videoEngine = engine
}

type videoAdapter struct{ engine VideoEngine }

func (a videoAdapter) Download(ctx context.Context, req download.VideoRequest, progress func(download.VideoProgress)) error {
	headers := req.Headers
	if headers == nil {
		headers = map[string]string{}
	}
	encoded, err := json.Marshal(headers)
	if err != nil {
		return err
	}
	return a.engine.Process(req.URL, string(encoded), req.OutputPath, &videoObserver{ctx, progress})
}

type videoObserver struct {
	ctx      context.Context
	progress func(download.VideoProgress)
}

func (o *videoObserver) Cancelled() bool { return o.ctx.Err() != nil }

func (o *videoObserver) Progress(fraction float64, bytes int64, bytesPerSecond float64) {
	o.progress(download.VideoProgress{Fraction: fraction, Bytes: bytes, BytesPerSecond: bytesPerSecond})
}
