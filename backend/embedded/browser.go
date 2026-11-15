package embedded

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Browser verification sessions pin one source and one public URL, and are not exposed over HTTP.

const (
	maxBrowserSessions = 8
	browserSessionTTL  = 15 * time.Minute
	maxBrowserState    = 256 * 1024
)

type browserSession struct {
	ID          string `json:"id"`
	ExtensionID string `json:"extensionId"`
	URL         string `json:"url"`
	State       string `json:"state"`
	expires     time.Time
}

var (
	browserMu       sync.Mutex
	browserSessions = map[string]*browserSession{}
)

// BeginBrowserVerification opens a session and returns {"id","extensionId","url","state"} as JSON.
func BeginBrowserVerification(extensionID, target string) (string, error) {
	parsed, err := url.Parse(target)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Hostname() == "" || parsed.User != nil {
		return "", errors.New("invalid source URL")
	}
	if !publicHost(parsed.Hostname()) {
		return "", errors.New("browser verification requires a public source URL")
	}
	if _, err := browserExchange(extensionID, target, "", false); err != nil {
		return "", err
	}

	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	browserMu.Lock()
	defer browserMu.Unlock()
	now := time.Now()
	for id, s := range browserSessions {
		if now.After(s.expires) {
			delete(browserSessions, id)
		}
	}
	if len(browserSessions) >= maxBrowserSessions {
		return "", errors.New("too many pending browser sessions")
	}
	session := &browserSession{
		ID: hex.EncodeToString(secret), ExtensionID: extensionID, URL: target,
		State: "pending", expires: now.Add(browserSessionTTL),
	}
	browserSessions[session.ID] = session
	body, err := json.Marshal(session)
	return string(body), err
}

// BrowserCookies returns the session's cookies and User-Agent as JSON, to seed the browser.
func BrowserCookies(sessionID string) (string, error) {
	s, err := pendingSession(sessionID)
	if err != nil {
		return "", err
	}
	return browserExchange(s.ExtensionID, s.URL, "", false)
}

// CompleteBrowserVerification stores the browser's cookies and User-Agent for the source and ends the session.
func CompleteBrowserVerification(sessionID, stateJSON string) error {
	if len(stateJSON) > maxBrowserState {
		return errors.New("browser state is too large")
	}
	s, err := pendingSession(sessionID)
	if err != nil {
		return err
	}
	if _, err := browserExchange(s.ExtensionID, s.URL, stateJSON, true); err != nil {
		return err
	}
	CancelBrowserVerification(sessionID)
	return nil
}

// CancelBrowserVerification ends a session without changing anything.
func CancelBrowserVerification(sessionID string) {
	browserMu.Lock()
	defer browserMu.Unlock()
	delete(browserSessions, sessionID)
}

func pendingSession(id string) (browserSession, error) {
	browserMu.Lock()
	defer browserMu.Unlock()
	s := browserSessions[id]
	if s == nil || time.Now().After(s.expires) {
		delete(browserSessions, id)
		return browserSession{}, errors.New("browser session expired or unknown")
	}
	return *s, nil
}

func browserExchange(extensionID, target, stateJSON string, apply bool) (string, error) {
	mu.Lock()
	srv := running
	mu.Unlock()
	if srv == nil {
		return "", errors.New("tsunagu is not running")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.BrowserCookies(ctx, extensionID, target, stateJSON, apply)
}

// publicHost rejects local and private hosts so a session can't plant cookies for local services.
func publicHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast())
	}
	return true
}
