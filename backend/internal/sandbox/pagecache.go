package sandbox

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const pageListTTL = 15 * time.Minute

type pageListEntry struct {
	urls []string
	at   time.Time
}

var (
	pageListMu    sync.RWMutex
	pageListStore = map[string]pageListEntry{}
	pageListGroup singleflight.Group
)

func pageListKey(extensionID, sourceEntryID, sourceChapterID string) string {
	return extensionID + "\x00" + sourceEntryID + "\x00" + sourceChapterID
}

func pageListGet(key string) ([]string, bool) {
	pageListMu.RLock()
	defer pageListMu.RUnlock()
	e, ok := pageListStore[key]
	if !ok || time.Since(e.at) > pageListTTL {
		return nil, false
	}
	return e.urls, true
}

func pageListPut(key string, urls []string) {
	pageListMu.Lock()
	pageListStore[key] = pageListEntry{urls: urls, at: time.Now()}
	pageListMu.Unlock()
}

// InvalidatePageList drops any cached page URLs for a chapter.
func InvalidatePageList(extensionID, sourceEntryID, sourceChapterID string) {
	key := pageListKey(extensionID, sourceEntryID, sourceChapterID)
	pageListMu.Lock()
	delete(pageListStore, key)
	pageListMu.Unlock()
}

// GetPageURLs returns the source image URLs for a chapter, memoized for
// pageListTTL and coalesced so concurrent readers of the same chapter issue a
// single sandbox call. Callers get the raw upstream image URLs, not proxy paths.
func (c *Client) GetPageURLs(ctx context.Context, extensionID, sourceEntryID, sourceChapterID string) ([]string, error) {
	key := pageListKey(extensionID, sourceEntryID, sourceChapterID)
	if urls, ok := pageListGet(key); ok {
		return urls, nil
	}
	v, err, _ := pageListGroup.Do(key, func() (any, error) {
		if urls, ok := pageListGet(key); ok {
			return urls, nil
		}
		pl, err := c.GetPages(ctx, extensionID, sourceEntryID, sourceChapterID)
		if err != nil {
			return nil, err
		}
		urls := pl.GetPageUrls()
		if len(urls) > 0 {
			pageListPut(key, urls)
		}
		return urls, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]string), nil
}
