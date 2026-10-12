package download

import (
	"bytes"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
)

var ffmpegDurationRe = regexp.MustCompile(`Duration: (\d+):(\d{2}):(\d{2}(?:\.\d+)?)`)

// ffmpegStderr collects ffmpeg's stderr (for error messages) and picks up the
// input duration it prints while probing, so progress can be out_time/duration
// instead of a guess from the bytes written so far.
type ffmpegStderr struct {
	mu         sync.Mutex
	buf        bytes.Buffer
	durationMs atomic.Int64
}

func (s *ffmpegStderr) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.Write(p)
	if s.durationMs.Load() == 0 {
		if ms := parseFfmpegDurationMs(s.buf.Bytes()); ms > 0 {
			s.durationMs.Store(ms)
		}
	}
	return len(p), nil
}

func (s *ffmpegStderr) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func (s *ffmpegStderr) DurationMs() int64 { return s.durationMs.Load() }

func parseFfmpegDurationMs(out []byte) int64 {
	m := ffmpegDurationRe.FindSubmatch(out)
	if m == nil {
		return 0
	}
	h, _ := strconv.ParseInt(string(m[1]), 10, 64)
	min, _ := strconv.ParseInt(string(m[2]), 10, 64)
	sec, _ := strconv.ParseFloat(string(m[3]), 64)
	return h*3_600_000 + min*60_000 + int64(sec*1000)
}

// ffmpegProgress maps ffmpeg's output position onto the job's progress range.
// Without a known duration it falls back to the old size-based guess.
func ffmpegProgress(outTimeUs, durationMs, totalSize int64) float64 {
	if durationMs > 0 && outTimeUs > 0 {
		p := float64(outTimeUs/1000) / float64(durationMs)
		return clampFloat(p, 0.01, 0.99)
	}
	if totalSize > 0 {
		return clampFloat(float64(totalSize)/(300*1024*1024), 0.1, 0.95)
	}
	return 0.1
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
