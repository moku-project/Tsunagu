package repository

import (
	"net/url"
	"strings"
)

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
