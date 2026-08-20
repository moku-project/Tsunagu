package sandbox

import sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"

func ContentTypeToProto(s string) sandboxv1.ContentType {
	switch s {
	case "manga":
		return sandboxv1.ContentType_MANGA
	case "novel":
		return sandboxv1.ContentType_NOVEL
	case "anime":
		return sandboxv1.ContentType_ANIME
	default:
		return sandboxv1.ContentType_CONTENT_TYPE_UNSPECIFIED
	}
}
