package rest

import (
	"context"
	"time"

	"tsunagu/backend/internal/sandbox"

	"golang.org/x/sync/singleflight"
)

var (
	mangaImages     = newSegCache(384 << 20)
	mangaImageGroup singleflight.Group
	prefetchSem     = make(chan struct{}, 6)
)

func mangaImageKey(extensionID, url string) string {
	return extensionID + "\x00" + url
}

func fetchMangaImage(ctx context.Context, client *sandbox.Client, extensionID, url string) ([]byte, string, error) {
	key := mangaImageKey(extensionID, url)
	if data, ct, ok := mangaImages.get(key); ok {
		return data, ct, nil
	}
	v, err, _ := mangaImageGroup.Do(key, func() (any, error) {
		if data, ct, ok := mangaImages.get(key); ok {
			return [2]any{data, ct}, nil
		}
		img, err := client.GetImageBytes(ctx, extensionID, url)
		if err != nil {
			return nil, err
		}
		data := img.GetData()
		ct := img.GetContentType()
		if len(data) > 0 {
			mangaImages.put(key, data, ct)
		}
		return [2]any{data, ct}, nil
	})
	if err != nil {
		return nil, "", err
	}
	pair := v.([2]any)
	data, _ := pair[0].([]byte)
	ct, _ := pair[1].(string)
	return data, ct, nil
}

// prefetchMangaImages warms the image cache for the given upstream URLs in the
// background, bounded by prefetchSem. Already-cached URLs are skipped cheaply.
func prefetchMangaImages(client *sandbox.Client, extensionID string, urls []string) {
	for _, u := range urls {
		if u == "" {
			continue
		}
		if _, _, ok := mangaImages.get(mangaImageKey(extensionID, u)); ok {
			continue
		}
		u := u
		select {
		case prefetchSem <- struct{}{}:
		default:
			return
		}
		go func() {
			defer func() { <-prefetchSem }()
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			_, _, _ = fetchMangaImage(ctx, client, extensionID, u)
		}()
	}
}
