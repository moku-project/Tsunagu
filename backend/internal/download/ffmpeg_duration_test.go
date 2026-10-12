package download

import "testing"

func TestFfmpegStderrPicksUpDuration(t *testing.T) {
	var s ffmpegStderr
	_, _ = s.Write([]byte("Input #0, hls, from 'http://x/master.m3u8':\n  Dura"))
	if s.DurationMs() != 0 {
		t.Fatal("duration must wait for the whole line")
	}
	_, _ = s.Write([]byte("tion: 00:23:40.52, start: 1.4, bitrate: N/A\n"))
	if got, want := s.DurationMs(), int64(23*60_000+40_520); got != want {
		t.Fatalf("DurationMs() = %d, want %d", got, want)
	}
	if s.String() == "" {
		t.Fatal("stderr text must be kept for error messages")
	}
}

func TestFfmpegProgress(t *testing.T) {
	if got := ffmpegProgress(711_000_000, 1_422_000, 50<<20); got < 0.49 || got > 0.51 {
		t.Fatalf("half the duration should be ~0.5, got %f", got)
	}
	if got := ffmpegProgress(0, 0, 150<<20); got != 0.5 {
		t.Fatalf("size fallback = %f, want 0.5", got)
	}
	if got := ffmpegProgress(2_000_000_000, 1_000_000, 0); got != 0.99 {
		t.Fatalf("progress must stay below 1 until the job finishes, got %f", got)
	}
}
