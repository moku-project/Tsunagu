package tracker

import "testing"

func TestStatusRoundTrip(t *testing.T) {
	for _, s := range AllStatuses {
		if got := statusFromAniList(statusToAniList(s)); got != s {
			t.Errorf("round-trip %v -> %q -> %v", s, statusToAniList(s), got)
		}
	}
}

func TestExtractToken(t *testing.T) {
	cases := map[string]string{
		"abc123":     "abc123",
		"  abc123  ": "abc123",
		"https://x/cb#access_token=tok&token_type=Bearer": "tok",
		"https://x/cb?access_token=tok2&state=1":          "tok2",
	}
	for in, want := range cases {
		if got := extractToken(in); got != want {
			t.Errorf("extractToken(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScoreOptionsFormats(t *testing.T) {
	a := NewAniList("x")
	if got := a.ScoreOptions(Auth{ScoreFormat: "POINT_5"}); len(got) != 6 {
		t.Errorf("POINT_5 options = %v", got)
	}
	if got := a.ScoreOptions(Auth{ScoreFormat: "POINT_3"}); len(got) != 4 {
		t.Errorf("POINT_3 options = %v", got)
	}
	if got := a.ScoreOptions(Auth{}); len(got) != 11 {
		t.Errorf("default (POINT_10) options = %v", got)
	}
}
