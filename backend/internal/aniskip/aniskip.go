package aniskip

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type Marker struct {
	Type    string
	Name    string
	StartMs int64
	EndMs   int64
}

var api = "https://api.aniskip.com/v2/skip-times"

var skipTypes = []string{"op", "ed", "mixed-op", "mixed-ed", "recap"}

var httpClient = &http.Client{Timeout: 8 * time.Second}

type cacheEntry struct {
	markers []Marker
	at      time.Time
}

var (
	mu    sync.Mutex
	cache = map[string]cacheEntry{}
)

const cacheTTL = 6 * time.Hour

func Fetch(ctx context.Context, malID, episode int, lengthSec float64) ([]Marker, error) {
	if malID <= 0 || episode <= 0 {
		return nil, nil
	}
	ck := strconv.Itoa(malID) + "/" + strconv.Itoa(episode)
	mu.Lock()
	if e, ok := cache[ck]; ok && time.Since(e.at) < cacheTTL {
		mu.Unlock()
		return e.markers, nil
	}
	mu.Unlock()

	q := url.Values{}
	for _, t := range skipTypes {
		q.Add("types", t)
	}
	q.Set("episodeLength", strconv.FormatFloat(lengthSec, 'f', 2, 64))
	u := fmt.Sprintf("%s/%d/%d?%s", api, malID, episode, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		markers := []Marker(nil)
		store(ck, markers)
		return markers, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aniskip: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var out struct {
		Found   bool `json:"found"`
		Results []struct {
			Interval struct {
				StartTime float64 `json:"startTime"`
				EndTime   float64 `json:"endTime"`
			} `json:"interval"`
			SkipType string `json:"skipType"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	var markers []Marker
	for _, r := range out.Results {
		if r.Interval.EndTime <= r.Interval.StartTime {
			continue
		}
		markers = append(markers, Marker{
			Type:    normalizeType(r.SkipType),
			Name:    label(r.SkipType),
			StartMs: int64(r.Interval.StartTime * 1000),
			EndMs:   int64(r.Interval.EndTime * 1000),
		})
	}
	store(ck, markers)
	return markers, nil
}

func store(k string, m []Marker) {
	mu.Lock()
	cache[k] = cacheEntry{markers: m, at: time.Now()}
	mu.Unlock()
}

func normalizeType(t string) string {
	switch t {
	case "op", "mixed-op":
		return "opening"
	case "ed", "mixed-ed":
		return "ending"
	case "recap":
		return "recap"
	default:
		return t
	}
}

func label(t string) string {
	switch t {
	case "op", "mixed-op":
		return "Opening"
	case "ed", "mixed-ed":
		return "Ending"
	case "recap":
		return "Recap"
	default:
		return t
	}
}
