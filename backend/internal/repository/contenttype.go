package repository

import "strings"

func ClassifyContentType(packageName string) string {
	switch {
	case strings.HasPrefix(packageName, "eu.kanade.tachiyomi.animeextension."):
		return "anime"
	case strings.HasPrefix(packageName, "eu.kanade.tachiyomi.extension."):
		return "manga"
	default:
		return "manga"
	}
}
