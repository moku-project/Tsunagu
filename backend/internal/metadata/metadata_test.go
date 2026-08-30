package metadata

import "testing"

func TestBestMatch(t *testing.T) {
	cands := []Candidate{
		{ProviderID: "1", Titles: []string{"Berserk"}, StartYear: 1989},
		{ProviderID: "2", Titles: []string{"Bleach"}, StartYear: 2001},
		{ProviderID: "3", Titles: []string{"Berserk of Gluttony", "Bōshoku no Berserk"}, StartYear: 2020},
	}

	got, score := bestMatch("Berserk", 0, cands)
	if got.ProviderID != "1" {
		t.Fatalf("exact title should win, got id=%s score=%.2f", got.ProviderID, score)
	}
	if score < AutoApplyScore {
		t.Errorf("exact match should auto-apply, score=%.2f", score)
	}

	got, _ = bestMatch("Berserk (2016)", 2016, cands)
	if got.ProviderID != "1" {
		t.Errorf("year+title should still pick canonical Berserk, got id=%s", got.ProviderID)
	}

	_, score = bestMatch("One Piece", 0, cands)
	if score >= AutoApplyScore {
		t.Errorf("unrelated title should not auto-apply, score=%.2f", score)
	}
}

func TestNormalizeTitle(t *testing.T) {
	cases := map[string]string{
		"Berserk, Vol. 1":         "berserk 1",
		"The Promised Neverland":  "promised neverland",
		"Re:ZERO -Starting Life-": "re zero starting life",
	}
	for in, want := range cases {
		if got := normalizeTitle(in); got != want {
			t.Errorf("normalizeTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMapStatus(t *testing.T) {
	if mapStatus("RELEASING") != "Ongoing" || mapStatus("FINISHED") != "Completed" {
		t.Error("status map wrong")
	}
	if mapStatus("WHATEVER") != "" {
		t.Error("unknown status should map to empty")
	}
}

func TestCleanDescription(t *testing.T) {
	in := "A <i>hero</i> rises.<br><br>Then falls.<br>The end."
	want := "A hero rises.\n\nThen falls.\nThe end."
	if got := cleanDescription(in); got != want {
		t.Errorf("cleanDescription = %q, want %q", got, want)
	}
}
