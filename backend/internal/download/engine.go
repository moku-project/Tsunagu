package download

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"tsunagu/backend/internal/db/sqlcgen"
)

// VideoDownloadEngine downloads an episode in-process, for embedders that
// cannot run an ffmpeg executable (iOS). It must write a complete, playable
// MP4 to OutputPath, honour ctx cancellation, and never report success for
// partial output. Without one, the manager runs ffmpeg.
type VideoDownloadEngine interface {
	Download(ctx context.Context, req VideoRequest, progress func(VideoProgress)) error
}

type VideoRequest struct {
	URL string
	// Headers the stream must be fetched with (Referer, Cookie, User-Agent…).
	Headers    map[string]string
	OutputPath string
}

type VideoProgress struct {
	Fraction       float64
	Bytes          int64
	BytesPerSecond float64
}

// SetVideoEngine routes episode downloads through engine instead of ffmpeg.
// Call it before Start.
func (m *Manager) SetVideoEngine(engine VideoDownloadEngine) { m.videoEngine = engine }

func (m *Manager) downloadWithEngine(ctx context.Context, jobID int64, streamURL string, headers map[string]string, outputPath string) error {
	_ = os.Remove(outputPath)
	var last time.Time
	report := func(p VideoProgress) {
		// Engines can report per packet; the database only needs a few updates a second.
		if time.Since(last) < 500*time.Millisecond || p.Fraction <= 0 {
			return
		}
		last = time.Now()
		// 0.1 is set before the stream starts; keep below 1 until the file is recorded.
		progress := 0.1 + 0.85*min(p.Fraction, 1)
		if err := m.q.UpdateDownloadProgress(ctx, sqlcgen.UpdateDownloadProgressParams{
			Status: "downloading", Progress: progress, ID: jobID,
		}); err != nil && ctx.Err() == nil {
			log.Printf("download: progress update failed for job %d: %v", jobID, err)
		}
	}
	err := m.videoEngine.Download(ctx, VideoRequest{URL: streamURL, Headers: headers, OutputPath: outputPath}, report)
	if ctx.Err() != nil {
		_ = os.Remove(outputPath)
		return ctx.Err()
	}
	if err != nil {
		_ = os.Remove(outputPath)
		return fmt.Errorf("video engine: %w", err)
	}
	if fi, statErr := os.Stat(outputPath); statErr != nil || fi.Size() == 0 {
		return errors.New("video engine reported success but produced no output")
	}
	return nil
}
