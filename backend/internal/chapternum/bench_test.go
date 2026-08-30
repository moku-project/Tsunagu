package chapternum

import "testing"

func BenchmarkFromTitle(b *testing.B) {
	titles := []string{
		"Chapter 700: At His Own Pace",
		"700 - Morning Edition",
		"His Own PaceChapter 701: Adventure",
		"Prologue",
		"Vol. 5 Ch. 42 The Long Road",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FromTitle(titles[i%len(titles)])
	}
}
