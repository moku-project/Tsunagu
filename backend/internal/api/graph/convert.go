package graph

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"tsunagu/backend/internal/api/graph/model"
	"tsunagu/backend/internal/config"
	"tsunagu/backend/internal/db/sqlcgen"
	"tsunagu/backend/internal/flaresolverr"
	"tsunagu/backend/internal/metadata"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
	"tsunagu/backend/internal/tracker"
)

func toServerSetting(s config.EffectiveSetting) *model.ServerSetting {
	return &model.ServerSetting{
		Key:         s.Key,
		Value:       s.Value,
		Default:     s.Default,
		Type:        model.SettingType(s.Type),
		Kind:        model.SettingKind(s.Kind),
		Scope:       model.SettingScope(s.Scope),
		Source:      model.SettingSource(s.Source),
		Editable:    s.Editable,
		Description: s.Description,
	}
}

func toCloudflareSolver(s flaresolverr.Status) *model.CloudflareSolver {
	return &model.CloudflareSolver{
		Mode:                model.SolverMode(strings.ToUpper(s.Mode)),
		State:               model.SolverState(s.State),
		DownloadProgress:    s.Progress,
		Version:             s.Version,
		URL:                 s.URL,
		Reachable:           s.Reachable,
		Error:               s.Error,
		SupportedOnPlatform: s.Supported,
	}
}

func toMetadataMatch(l sqlcgen.MetadataLink) *model.MetadataMatch {
	var coverURL *string
	if l.CoverUrl != "" {
		coverURL = &l.CoverUrl
	}
	return &model.MetadataMatch{
		MediaID:    strconv.FormatInt(l.MediaID, 10),
		Provider:   l.Provider,
		ProviderID: l.ProviderID,
		URL:        l.ProviderUrl,
		CoverURL:   coverURL,
		Confidence: l.Confidence,
		Locked:     l.Locked != 0,
		MatchedAt:  l.MatchedAt,
	}
}

func toMetadataCandidate(c metadata.Candidate) *model.MetadataCandidate {
	m := &model.MetadataCandidate{
		Provider:   "anilist",
		ProviderID: c.ProviderID,
		Title:      c.PrimaryTitle,
		URL:        c.URL,
		Genres:     c.Genres,
	}
	if m.Genres == nil {
		m.Genres = []string{}
	}
	if c.CoverURL != "" {
		v := c.CoverURL
		m.CoverURL = &v
	}
	if c.Description != "" {
		v := c.Description
		m.Description = &v
	}
	if c.Status != "" {
		v := c.Status
		m.Status = &v
	}
	if c.StartYear > 0 {
		v := int32(c.StartYear)
		m.StartYear = &v
	}
	return m
}

func proxyImageURL(absURL string) string {
	if absURL == "" {
		return ""
	}
	return "/proxy/img/" + base64.URLEncoding.EncodeToString([]byte(absURL))
}

func toTracker(i tracker.Info) *model.Tracker {
	m := &model.Tracker{
		Key:           i.Key,
		Name:          i.Name,
		Configured:    i.Configured,
		IsLoggedIn:    i.IsLoggedIn,
		ScoreOptions:  i.ScoreOptions,
		StatusOptions: trackStatusOptions(),
	}
	if m.ScoreOptions == nil {
		m.ScoreOptions = []string{}
	}
	if i.AuthURL != "" {
		v := i.AuthURL
		m.AuthURL = &v
	}
	if u := proxyImageURL(i.IconURL); u != "" {
		m.IconURL = &u
	}
	if i.Username != "" {
		v := i.Username
		m.Username = &v
	}
	return m
}

func trackStatusOptions() []*model.TrackStatus {
	out := make([]*model.TrackStatus, 0, len(tracker.AllStatuses))
	for _, s := range tracker.AllStatuses {
		out = append(out, &model.TrackStatus{Value: int32(s), Name: s.String(), AnimeName: s.AnimeString()})
	}
	return out
}

func toTrackSearchResult(r tracker.SearchResult) *model.TrackSearchResult {
	m := &model.TrackSearchResult{
		RemoteID: r.RemoteID,
		Title:    r.Title,
		URL:      r.URL,
	}
	if u := proxyImageURL(r.CoverURL); u != "" {
		m.CoverURL = &u
	}
	if r.Summary != "" {
		v := r.Summary
		m.Summary = &v
	}
	if r.TotalChapters > 0 {
		v := int32(r.TotalChapters)
		m.TotalChapters = &v
	}
	if r.PublishingStatus != "" {
		v := r.PublishingStatus
		m.PublishingStatus = &v
	}
	if r.MediaType != "" {
		v := r.MediaType
		m.MediaType = &v
	}
	return m
}

func toTrackLink(l sqlcgen.TrackerLink, trackerKey string) *model.TrackLink {
	return &model.TrackLink{
		ID:              strconv.FormatInt(l.ID, 10),
		MediaID:         strconv.FormatInt(l.MediaID, 10),
		TrackerKey:      trackerKey,
		RemoteID:        l.ExternalTrackerID,
		Title:           l.TrackerTitle,
		URL:             l.RemoteUrl,
		Status:          int32(l.Status),
		StatusName:      tracker.Status(l.Status).String(),
		LastChapterRead: l.LastChapterRead,
		TotalChapters:   int32(l.TotalChapters),
		Score:           l.Score,
		StartedAt:       nullTimePtr(l.StartedAt),
		FinishedAt:      nullTimePtr(l.FinishedAt),
		Private:         l.Private != 0,
		LastSyncedAt:    nullTimePtr(l.LastSyncedAt),
	}
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func nullFloat64Ptr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	return &v.Float64
}

func nullBoolPtr(v sql.NullBool) *bool {
	if !v.Valid {
		return nil
	}
	return &v.Bool
}

func nullTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	return &v.Time
}

func nullInt64Int32Ptr(v sql.NullInt64) *int32 {
	if !v.Valid {
		return nil
	}
	n := int32(v.Int64)
	return &n
}

func nullInt64Float64Ptr(v sql.NullInt64) *float64 {
	if !v.Valid {
		return nil
	}
	f := float64(v.Int64)
	return &f
}

func nullInt64Ptr(v sql.NullInt64) *string {
	if !v.Valid {
		return nil
	}
	s := strconv.FormatInt(v.Int64, 10)
	return &s
}

func epochToTimePtr(v sql.NullInt64) *time.Time {
	if !v.Valid || v.Int64 == 0 {
		return nil
	}
	t := time.Unix(v.Int64, 0).UTC()
	return &t
}

func contentType(s string) model.ContentType {
	switch s {
	case "novel":
		return model.ContentTypeNovel
	case "manga":
		return model.ContentTypeManga
	case "anime":
		return model.ContentTypeAnime
	}
	return model.ContentTypeManga
}

func contentTypeToString(c *model.ContentType) string {
	if c == nil {
		return ""
	}
	switch *c {
	case model.ContentTypeNovel:
		return "novel"
	case model.ContentTypeManga:
		return "manga"
	case model.ContentTypeAnime:
		return "anime"
	}
	return ""
}

func toRepository(r sqlcgen.Repository) *model.Repository {
	return &model.Repository{
		ID:           strconv.FormatInt(r.ID, 10),
		IndexURL:     r.IndexUrl,
		Name:         nullStringPtr(r.Name),
		ContentType:  contentType(r.ContentType),
		AddedAt:      r.AddedAt,
		LastSyncedAt: nullTimePtr(r.LastSyncedAt),
	}
}

func extensionDisplayName(name, lang string) string {
	if lang == "" || lang == "all" {
		return name
	}
	return name + " (" + strings.ToUpper(lang) + ")"
}

func toExtension(e sqlcgen.Extension, mediaDir string) *model.Extension {
	var iconURL *string
	if e.IconUrl.Valid && e.IconUrl.String != "" {
		u := fmt.Sprintf("/proxy/icon/%d", e.ID)
		iconURL = &u
	}
	return &model.Extension{
		ID:               strconv.FormatInt(e.ID, 10),
		RepositoryID:     strconv.FormatInt(e.RepositoryID, 10),
		PackageName:      e.PackageName,
		Name:             e.Name,
		Version:          e.Version,
		ContentType:      contentType(e.ContentType),
		Lang:             e.Lang,
		IconURL:          iconURL,
		ApkURL:           &e.ApkUrl,
		JarURL:           nullStringPtr(e.JarUrl),
		JarPath:          nullStringPtr(e.JarPath),
		Installed:        e.Installed,
		Enabled:          e.Enabled,
		DiscoveredAt:     e.DiscoveredAt,
		InstalledAt:      nullTimePtr(e.InstalledAt),
		InstalledVersion: nullStringPtr(e.InstalledVersion),
		NeedsUpdate:      nullBoolPtr(e.NeedsUpdate),
		IsNsfw:           e.IsNsfw,
		DisplayName:      extensionDisplayName(e.Name, e.Lang),
		SupportsLatest:   e.SupportsLatest,
	}
}

func toMedia(l sqlcgen.Medium, mediaDir string) *model.Media {
	var thumbnailURL *string
	if (l.CoverOverride.Valid && l.CoverOverride.String != "") ||
		(l.CoverPath.Valid && l.CoverPath.String != "") ||
		(l.CoverLocalPath.Valid && l.CoverLocalPath.String != "") {

		u := fmt.Sprintf("/proxy/cover/%d", l.ID)
		thumbnailURL = &u
	}
	sourceName := l.ExtensionName
	if sourceName == "" {
		sourceName = "Unknown source"
	}
	return &model.Media{
		ID:                 strconv.FormatInt(l.ID, 10),
		ExtensionID:        nullInt64Ptr(l.ExtensionID),
		ExtensionName:      l.ExtensionName,
		SourceName:         sourceName,
		ExternalID:         l.ExternalID,
		ContentType:        contentType(l.ContentType),
		Title:              l.Title,
		ThumbnailURL:       thumbnailURL,
		Description:        nullStringPtr(l.Description),
		Status:             nullStringPtr(l.Status),
		Author:             nullStringPtr(l.Author),
		Artist:             nullStringPtr(l.Artist),
		DetailsFetchedAt:   nullTimePtr(l.DetailsFetchedAt),
		ExtensionRemovedAt: nullTimePtr(l.ExtensionRemovedAt),
		AddedAt:            nullTimePtr(l.AddedAt),
		LastViewedAt:       nullTimePtr(l.LastViewedAt),
		InLibrary:          l.AddedAt.Valid,
	}
}

func proxyResourceURL(route, mediaID, chapterID, absURL string, headers map[string]string) string {
	q := url.Values{}
	q.Set("u", base64.RawURLEncoding.EncodeToString([]byte(absURL)))
	if len(headers) > 0 {
		if j, err := json.Marshal(headers); err == nil {
			q.Set("h", base64.RawURLEncoding.EncodeToString(j))
		}
	}
	return fmt.Sprintf("/content/%s/%s/%s?%s", mediaID, chapterID, route, q.Encode())
}

var (
	preferredSubLangs   = parseLangCSV(getenv("TSUNAGU_PREFERRED_SUB_LANGS"))
	preferredAudioLangs = parseLangCSV(getenv("TSUNAGU_PREFERRED_AUDIO_LANGS"))
)

func getenv(k string) string { return os.Getenv(k) }

func parseLangCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.ToLower(strings.TrimSpace(p)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func sortByPreferredLang[T any](tracks []T, prefs []string, lang func(T) string) {
	if len(prefs) == 0 || len(tracks) < 2 {
		return
	}
	rank := func(t T) int {
		l := strings.ToLower(lang(t))
		for i, p := range prefs {
			if strings.Contains(l, p) {
				return i
			}
		}
		return len(prefs)
	}
	sort.SliceStable(tracks, func(i, j int) bool { return rank(tracks[i]) < rank(tracks[j]) })
}

func parseSourceLabel(label string) (kind, server string) {
	l := strings.ToLower(label)
	switch {
	case strings.Contains(l, "hard sub"), strings.Contains(l, "hardsub"):
		kind = "hardsub"
	case strings.Contains(l, "soft sub"), strings.Contains(l, "softsub"):
		kind = "softsub"
	case wordRe("dub").MatchString(l):
		kind = "dub"
	case wordRe("sub").MatchString(l):
		kind = "sub"
	}
	if i := strings.Index(label, " - "); i > 0 && i <= 24 {
		server = strings.TrimSpace(label[:i])
	}
	return kind, server
}

var wordReCache = map[string]*regexp.Regexp{}

func wordRe(w string) *regexp.Regexp {
	if re, ok := wordReCache[w]; ok {
		return re
	}
	re := regexp.MustCompile(`\b` + w + `\b`)
	wordReCache[w] = re
	return re
}

func toVideoStream(info *sandboxv1.StreamInfo, mediaID, chapterID string) *model.VideoStream {
	videoBase := fmt.Sprintf("/content/%s/%s/video", mediaID, chapterID)
	headers := info.GetHeaders()

	vs := &model.VideoStream{
		URL:         videoBase,
		Sources:     []*model.VideoSource{},
		Subtitles:   []*model.SubtitleTrack{},
		AudioTracks: []*model.AudioTrack{},
		SkipMarkers: []*model.SkipMarker{},
	}

	for _, s := range info.GetSources() {
		kind, server := parseSourceLabel(s.GetLabel())
		src := &model.VideoSource{
			Label:     s.GetLabel(),
			Preferred: s.GetPreferred(),
			Kind:      kind,
			Server:    server,
			URL:       videoBase + "?quality=" + url.QueryEscape(s.GetLabel()),
		}
		if res := s.GetResolution(); res > 0 {
			r := res
			src.Resolution = &r
		}
		vs.Sources = append(vs.Sources, src)
	}

	subs := info.GetSubtitles()
	if len(subs) == 0 {
		seen := map[string]bool{}
		for _, s := range info.GetSources() {
			for _, t := range s.GetSubtitles() {
				if t.GetUrl() != "" && !seen[t.GetUrl()] {
					seen[t.GetUrl()] = true
					subs = append(subs, t)
				}
			}
		}
	}
	sortByPreferredLang(subs, preferredSubLangs, func(t *sandboxv1.SubtitleTrack) string { return t.GetLang() })
	for _, t := range subs {
		vs.Subtitles = append(vs.Subtitles, &model.SubtitleTrack{
			Lang: t.GetLang(),
			URL:  proxyResourceURL("subtitle", mediaID, chapterID, t.GetUrl(), headers),
		})
	}
	audio := info.GetAudioTracks()
	sortByPreferredLang(audio, preferredAudioLangs, func(t *sandboxv1.AudioTrack) string { return t.GetLang() })
	for _, t := range audio {
		vs.AudioTracks = append(vs.AudioTracks, &model.AudioTrack{
			Lang: t.GetLang(),
			URL:  proxyResourceURL("hls", mediaID, chapterID, t.GetUrl(), headers),
		})
	}
	for _, ts := range info.GetTimestamps() {
		vs.SkipMarkers = append(vs.SkipMarkers, &model.SkipMarker{
			Type:    ts.GetType(),
			Name:    ts.GetName(),
			StartMs: int32(ts.GetStartMs()),
			EndMs:   int32(ts.GetEndMs()),
		})
	}
	return vs
}

func toChapter(c sqlcgen.Chapter) *model.Chapter {
	var scanlator *string
	if c.Scanlator != "" {
		v := c.Scanlator
		scanlator = &v
	}
	return &model.Chapter{
		ID:          strconv.FormatInt(c.ID, 10),
		MediaID:     strconv.FormatInt(c.MediaID, 10),
		ExternalID:  c.ExternalID,
		Title:       nullStringPtr(c.Title),
		Number:      nullFloat64Ptr(c.Number),
		SourceOrder: nullInt64Int32Ptr(c.SourceOrder),
		Scanlator:   scanlator,
		UploadedAt:  epochToTimePtr(c.UploadedAt),
	}
}

func toReadingProgress(p sqlcgen.ReadingProgress) *model.ReadingProgress {
	return &model.ReadingProgress{
		ID:              strconv.FormatInt(p.ID, 10),
		MediaID:         strconv.FormatInt(p.MediaID, 10),
		ChapterID:       strconv.FormatInt(p.ChapterID, 10),
		Progress:        p.Progress,
		Completed:       p.Completed,
		PositionSeconds: nullFloat64Ptr(p.PositionSeconds),
		DurationSeconds: nullFloat64Ptr(p.DurationSeconds),
		UpdatedAt:       p.UpdatedAt,
	}
}

func downloadStatus(s string) model.DownloadStatus {
	switch s {
	case "queued":
		return model.DownloadStatusQueued
	case "downloading":
		return model.DownloadStatusDownloading
	case "done":
		return model.DownloadStatusDone
	case "failed":
		return model.DownloadStatusFailed
	}
	return model.DownloadStatusQueued
}

func toDownloadFields(id, chapterID, mediaID int64, status string, progress float64, downloadedBytes sql.NullInt64, bytesPerSec sql.NullFloat64, errStr sql.NullString, createdAt time.Time, completedAt sql.NullTime) *model.Download {
	var finalSize *float64
	if status == "done" {
		finalSize = nullInt64Float64Ptr(downloadedBytes)
	}
	return &model.Download{
		ID:              strconv.FormatInt(id, 10),
		MediaID:         strconv.FormatInt(mediaID, 10),
		ChapterID:       strconv.FormatInt(chapterID, 10),
		Status:          downloadStatus(status),
		Progress:        progress,
		DownloadedBytes: nullInt64Float64Ptr(downloadedBytes),
		BytesPerSec:     nullFloat64Ptr(bytesPerSec),
		FinalSizeBytes:  finalSize,
		Error:           nullStringPtr(errStr),
		CreatedAt:       createdAt,
		CompletedAt:     nullTimePtr(completedAt),
	}
}

func parseID(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func contentPageURLs(mediaID, chapterID string, count int) []string {
	out := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		out = append(out, fmt.Sprintf("/content/%s/%s/pages/%d", mediaID, chapterID, i))
	}
	return out
}

func (r *Resolver) resolveExtension(ctx context.Context, extensionID string) (sqlcgen.Extension, error) {
	id, err := parseID(extensionID)
	if err != nil {
		return sqlcgen.Extension{}, fmt.Errorf("invalid extension id %q: %w", extensionID, err)
	}
	ext, err := r.Q.GetExtension(ctx, id)
	if err != nil {
		return sqlcgen.Extension{}, fmt.Errorf("lookup extension %d: %w", id, err)
	}
	return ext, nil
}

func toFolder(f sqlcgen.Folder) *model.Folder {
	var parentID *string
	if f.ParentFolderID.Valid {
		s := strconv.FormatInt(f.ParentFolderID.Int64, 10)
		parentID = &s
	}
	return &model.Folder{
		ID:                strconv.FormatInt(f.ID, 10),
		Name:              f.Name,
		Kind:              f.Kind,
		SystemKey:         nullStringPtr(f.SystemKey),
		ParentFolderID:    parentID,
		SortOrder:         int32(f.SortOrder),
		IncludeInUpdate:   f.IncludeInUpdate != 0,
		IncludeInDownload: f.IncludeInDownload != 0,
	}
}

func toFilterNode(n *sandboxv1.FilterNode) model.FilterNode {
	switch k := n.GetKind().(type) {
	case *sandboxv1.FilterNode_Header:
		return &model.HeaderFilter{Name: n.GetName()}
	case *sandboxv1.FilterNode_Separator:
		return &model.SeparatorFilter{Name: n.GetName()}
	case *sandboxv1.FilterNode_Select:
		return &model.SelectFilter{
			Name:   n.GetName(),
			Values: k.Select.GetValues(),
			State:  k.Select.GetState(),
		}
	case *sandboxv1.FilterNode_Text:
		return &model.TextFilter{Name: n.GetName(), State: k.Text.GetState()}
	case *sandboxv1.FilterNode_Checkbox:
		return &model.CheckBoxFilter{Name: n.GetName(), State: k.Checkbox.GetState()}
	case *sandboxv1.FilterNode_Tristate:
		return &model.TriStateFilter{Name: n.GetName(), State: k.Tristate.GetState()}
	case *sandboxv1.FilterNode_Group:
		children := make([]model.FilterNode, 0, len(k.Group.GetChildren()))
		for _, c := range k.Group.GetChildren() {
			children = append(children, toFilterNode(c))
		}
		return &model.GroupFilter{Name: n.GetName(), Children: children}
	case *sandboxv1.FilterNode_Sort:
		s := k.Sort
		result := &model.SortFilter{Name: n.GetName(), Values: s.GetValues(), HasState: s.GetHasState()}
		if s.GetHasState() {
			idx := s.GetIndex()
			asc := s.GetAscending()
			result.Index = &idx
			result.Ascending = &asc
		}
		return result
	default:
		return &model.HeaderFilter{Name: n.GetName()}
	}
}

func toFilterNodes(ns []*sandboxv1.FilterNode) []model.FilterNode {
	out := make([]model.FilterNode, 0, len(ns))
	for _, n := range ns {
		out = append(out, toFilterNode(n))
	}
	return out
}

func toProtoFilterNode(in *model.FilterInput) *sandboxv1.FilterNode {
	n := &sandboxv1.FilterNode{Name: in.Name}
	switch {
	case in.Select != nil:
		n.Kind = &sandboxv1.FilterNode_Select{Select: &sandboxv1.SelectFilter{State: in.Select.State}}
	case in.Text != nil:
		n.Kind = &sandboxv1.FilterNode_Text{Text: &sandboxv1.TextFilter{State: in.Text.State}}
	case in.Checkbox != nil:
		n.Kind = &sandboxv1.FilterNode_Checkbox{Checkbox: &sandboxv1.CheckBoxFilter{State: in.Checkbox.State}}
	case in.Tristate != nil:
		n.Kind = &sandboxv1.FilterNode_Tristate{Tristate: &sandboxv1.TriStateFilter{State: in.Tristate.State}}
	case in.Group != nil:
		children := make([]*sandboxv1.FilterNode, 0, len(in.Group.Children))
		for _, c := range in.Group.Children {
			children = append(children, toProtoFilterNode(c))
		}
		n.Kind = &sandboxv1.FilterNode_Group{Group: &sandboxv1.GroupFilter{Children: children}}
	case in.Sort != nil:
		sf := &sandboxv1.SortFilter{HasState: in.Sort.HasState}
		if in.Sort.HasState {
			if in.Sort.Index != nil {
				sf.Index = *in.Sort.Index
			}
			if in.Sort.Ascending != nil {
				sf.Ascending = *in.Sort.Ascending
			}
		}
		n.Kind = &sandboxv1.FilterNode_Sort{Sort: sf}
	}
	return n
}

func toProtoFilterNodes(ins []*model.FilterInput) []*sandboxv1.FilterNode {
	out := make([]*sandboxv1.FilterNode, 0, len(ins))
	for _, in := range ins {
		out = append(out, toProtoFilterNode(in))
	}
	return out
}

func (r *Resolver) toSearchResponse(ctx context.Context, ext sqlcgen.Extension, resp *sandboxv1.SearchResponse) (*model.SearchResponse, error) {
	results := make([]*model.Media, 0, len(resp.Results))
	for _, res := range resp.Results {
		row, err := r.Q.UpsertMediaBare(ctx, sqlcgen.UpsertMediaBareParams{
			ExtensionID:   sql.NullInt64{Int64: ext.ID, Valid: true},
			ExtensionName: ext.Name,
			ExternalID:    res.SourceEntryId,
			ContentType:   ext.ContentType,
			Title:         res.Title,
			CoverPath:     sql.NullString{String: res.CoverUrl, Valid: res.CoverUrl != ""},
		})
		if err != nil {
			return nil, fmt.Errorf("record search result %s: %w", res.SourceEntryId, err)
		}
		results = append(results, toMedia(row, r.MediaDir))
	}
	return &model.SearchResponse{Results: results, HasNextPage: resp.HasNextPage}, nil
}
