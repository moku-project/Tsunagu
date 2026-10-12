package repository

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var repoClient = &http.Client{Timeout: 60 * time.Second}

type FetchError struct {
	msg string
}

func (e *FetchError) Error() string { return e.msg }

func fetchErrorf(format string, args ...any) error {
	return &FetchError{msg: fmt.Sprintf(format, args...)}
}

var validPackageName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func IsValidPackageName(s string) bool { return validPackageName.MatchString(s) }

func ClassifyContentType(packageName string) string {
	switch {
	case strings.HasPrefix(packageName, "eu.kanade.tachiyomi.animeextension."):
		return "anime"
	case strings.HasPrefix(packageName, "eu.kanade.tachiyomi.novelextension."):
		return "novel"
	}
	return "manga"
}

func DeriveRepoName(indexURL string) string {
	u, err := url.Parse(indexURL)
	if err != nil || u.Host == "" {
		return indexURL
	}
	if u.Host == "raw.githubusercontent.com" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	return u.Host
}

func DownloadExtensionFile(cacheDir, packageName, version, downloadURL, ext string) (string, error) {
	if downloadURL == "" {
		return "", fetchErrorf("no download url available for %s", packageName)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir %s: %w", cacheDir, err)
	}

	dest := filepath.Join(cacheDir, fmt.Sprintf("%s-%s.%s", packageName, version, ext))
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}

	resp, err := repoClient.Get(downloadURL)
	if err != nil {
		return "", fetchErrorf("failed to fetch %s %s: %v", ext, downloadURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fetchErrorf("failed to fetch %s %s: HTTP %d", ext, downloadURL, resp.StatusCode)
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", fmt.Errorf("write %s to %s: %w", ext, tmp, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("close %s file: %w", ext, err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("finalize %s at %s: %w", ext, dest, err)
	}
	return dest, nil
}

func filterValid(exts []ParsedExtension) []ParsedExtension {
	out := make([]ParsedExtension, 0, len(exts))
	for _, ext := range exts {
		if !IsValidPackageName(ext.PackageName) {
			log.Printf("skipping repo entry with invalid package name %q", ext.PackageName)
			continue
		}
		out = append(out, ext)
	}
	return out
}

// indexV2URL returns the protobuf index a Mihon-style repo advertises in the
// repo.json next to its index (or at indexURL itself), or "" when it has none.
// Such repos (Keiyoushi, NovelSourcery) serve a placeholder index.min.json to
// apps that predate index_v2.
func indexV2URL(indexURL string) string {
	if !strings.HasSuffix(indexURL, ".json") {
		return ""
	}
	repoJSON := indexURL
	if !strings.HasSuffix(indexURL, "/repo.json") {
		i := strings.LastIndex(indexURL, "/")
		if i < 0 {
			return ""
		}
		repoJSON = indexURL[:i] + "/repo.json"
	}
	resp, err := repoClient.Get(repoJSON)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var meta struct {
		IndexV2 string `json:"index_v2"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&meta); err != nil {
		return ""
	}
	return meta.IndexV2
}

func FetchIndex(indexURL string) ([]ParsedExtension, error) {
	if v2 := indexV2URL(indexURL); v2 != "" {
		indexURL = v2
	}
	resp, err := repoClient.Get(indexURL)
	if err != nil {
		return nil, fetchErrorf("failed to fetch %s: %v", indexURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fetchErrorf("failed to fetch %s: HTTP %d", indexURL, resp.StatusCode)
	}

	// Repo indexes are JSON, never legitimately huge -- cap fetch/decompress size.
	const maxIndexBytes = 32 << 20

	rawBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxIndexBytes+1))
	if err != nil {
		return nil, fetchErrorf("failed to read response from %s: %v", indexURL, err)
	}
	if len(rawBytes) == 0 {
		return nil, fetchErrorf("empty response body from %s", indexURL)
	}
	if len(rawBytes) > maxIndexBytes {
		return nil, fetchErrorf("response from %s exceeds %d bytes", indexURL, maxIndexBytes)
	}

	data := rawBytes
	if len(rawBytes) >= 2 && rawBytes[0] == 0x1f && rawBytes[1] == 0x8b {
		gz, err := gzip.NewReader(bytes.NewReader(rawBytes))
		if err != nil {
			return nil, fetchErrorf("failed to gunzip %s: %v", indexURL, err)
		}
		defer gz.Close()
		unzipped, err := io.ReadAll(io.LimitReader(gz, maxIndexBytes+1))
		if err != nil {
			return nil, fetchErrorf("failed to gunzip %s: %v", indexURL, err)
		}
		if len(unzipped) > maxIndexBytes {
			return nil, fetchErrorf("decompressed response from %s exceeds %d bytes", indexURL, maxIndexBytes)
		}
		data = unzipped
	}

	apkBaseURL := indexURL
	if i := strings.LastIndex(indexURL, "/"); i >= 0 {
		apkBaseURL = indexURL[:i]
	}
	apkBaseURL = strings.TrimSuffix(apkBaseURL, "/") + "/apk/"

	if trimmed := strings.TrimSpace(string(data)); strings.HasPrefix(trimmed, "[") {
		var probe []map[string]json.RawMessage
		if err := json.Unmarshal(data, &probe); err == nil && len(probe) > 0 {
			_, hasID := probe[0]["id"]
			_, hasURL := probe[0]["url"]
			_, hasPkg := probe[0]["pkg"]
			if hasID && hasURL && !hasPkg {
				var novel []novelRepoEntry
				if err := json.Unmarshal(data, &novel); err == nil {
					return filterValid(novelToParsedExtensions(novel)), nil
				}
			}
			var legacy []legacyRepoExtension
			if err := json.Unmarshal(data, &legacy); err == nil {
				return filterValid(legacyToParsedExtensions(legacy, apkBaseURL)), nil
			}
		}
	}

	idx, err := decodeRepoIndex(data)
	if err != nil {
		return nil, fetchErrorf("could not parse %s as JSON or protobuf index: %v", indexURL, err)
	}

	if idx.ExtensionList == nil && idx.ExtensionListURL != "" {
		return nil, fetchErrorf(
			"index at %s references an external extensionListUrl (%s); paginated repo indices are not supported yet",
			indexURL, idx.ExtensionListURL,
		)
	}

	return filterValid(idx.toParsedExtensions(apkBaseURL)), nil
}
