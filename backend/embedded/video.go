package embedded

import (
	"context"
	"encoding/json"

	"tsunagu/backend/internal/download"
)

// VideoObserver reports progress; engines should check Cancelled from their I/O callbacks to stop promptly.
type VideoObserver interface {
	Cancelled() bool
	Progress(fraction float64, bytes int64, bytesPerSecond float64)
}

// VideoEngine downloads an episode in-process (iOS has no ffmpeg executable) and must write a complete MP4 or fail.
type VideoEngine interface {
	Process(url, headersJSON, outputPath string, observer VideoObserver) error
}

// SetVideoEngine routes episode downloads through engine from the next Start; nil restores ffmpeg.
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
