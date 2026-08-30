package tracker

import (
	"net/url"
	"strings"
	"testing"
)

func TestMALAuthURLAndPKCE(t *testing.T) {
	m := NewMAL(MALConfig{ClientID: "cid", ClientSecret: "sec"})
	if !m.Configured() {
		t.Fatal("should be configured")
	}
	raw := m.AuthURL()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("code_challenge_method") != "plain" {
		t.Errorf("method = %q", q.Get("code_challenge_method"))
	}
	state, chal := q.Get("state"), q.Get("code_challenge")
	if state == "" || chal == "" {
		t.Fatal("missing state/challenge")
	}
	m.mu.Lock()
	e, ok := m.pkce[state]
	m.mu.Unlock()
	if !ok || e.verifier != chal {
		t.Fatal("verifier not stored / != challenge (plain)")
	}

	if _, err := m.CompleteAuth(t.Context(), url.Values{"code": {"x"}, "state": {"nope"}}); err == nil {
		t.Error("unknown state should fail")
	}
}

func TestMALUnconfigured(t *testing.T) {
	m := NewMAL(MALConfig{})
	if m.Configured() || m.AuthURL() != "" {
		t.Fatal("blank config must be unconfigured with no auth URL")
	}
}

func TestMALIDAndStatus(t *testing.T) {
	if k, id := malSplitID("anime:678"); k != "anime" || id != "678" {
		t.Errorf("split = %q %q", k, id)
	}
	if k, id := malSplitID("12345"); k != "manga" || id != "12345" {
		t.Errorf("bare split = %q %q", k, id)
	}
	if canonicalToMALStatus(StatusReading, "anime") != "watching" {
		t.Error("anime reading -> watching")
	}
	if canonicalToMALStatus(StatusReading, "manga") != "reading" {
		t.Error("manga reading -> reading")
	}
	if malStatusToCanonical("watching", true) != StatusRereading {
		t.Error("rereading flag")
	}
	if malStatusToCanonical("plan_to_watch", false) != StatusPlanToRead {
		t.Error("plan_to_watch -> PlanToRead")
	}
}

func TestMALExchangeNeedsCode(t *testing.T) {
	m := NewMAL(MALConfig{ClientID: "c", ClientSecret: "s"})
	if _, err := m.Exchange(t.Context(), "https://x/cb?state=abc"); err == nil || !strings.Contains(err.Error(), "code") {
		t.Errorf("want code error, got %v", err)
	}
}
