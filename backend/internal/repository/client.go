package repository

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type FetchError struct {
	msg string
}

func (e *FetchError) Error() string { return e.msg }

func fetchErrorf(format string, args ...any) error {
	return &FetchError{msg: fmt.Sprintf(format, args...)}
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

func FetchIndex(indexURL string) ([]ParsedExtension, error) {
	resp, err := http.Get(indexURL)
	if err != nil {
		return nil, fetchErrorf("failed to fetch %s: %v", indexURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fetchErrorf("failed to fetch %s: HTTP %d", indexURL, resp.StatusCode)
	}

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fetchErrorf("failed to read response from %s: %v", indexURL, err)
	}
	if len(rawBytes) == 0 {
		return nil, fetchErrorf("empty response body from %s", indexURL)
	}

	data := rawBytes
	if len(rawBytes) >= 2 && rawBytes[0] == 0x1f && rawBytes[1] == 0x8b {
		gz, err := gzip.NewReader(bytes.NewReader(rawBytes))
		if err != nil {
			return nil, fetchErrorf("failed to gunzip %s: %v", indexURL, err)
		}
		defer gz.Close()
		unzipped, err := io.ReadAll(gz)
		if err != nil {
			return nil, fetchErrorf("failed to gunzip %s: %v", indexURL, err)
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
