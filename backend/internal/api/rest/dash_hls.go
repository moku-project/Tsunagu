package rest

import (
	"encoding/xml"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// AVPlayer can't play DASH, so players that ask for format=hls get the same fMP4 segments as HLS playlists.

type mpdSegmentTemplate struct {
	Timescale      int64  `xml:"timescale,attr"`
	Initialization string `xml:"initialization,attr"`
	Media          string `xml:"media,attr"`
	StartNumber    *int64 `xml:"startNumber,attr"`
	Duration       int64  `xml:"duration,attr"`
	Timeline       []struct {
		T *int64 `xml:"t,attr"`
		D int64  `xml:"d,attr"`
		R int64  `xml:"r,attr"`
	} `xml:"SegmentTimeline>S"`
}

type mpdRepresentation struct {
	ID        string              `xml:"id,attr"`
	MimeType  string              `xml:"mimeType,attr"`
	Codecs    string              `xml:"codecs,attr"`
	Bandwidth int64               `xml:"bandwidth,attr"`
	Width     int                 `xml:"width,attr"`
	Height    int                 `xml:"height,attr"`
	BaseURL   string              `xml:"BaseURL"`
	Template  *mpdSegmentTemplate `xml:"SegmentTemplate"`
}

type mpdAdaptationSet struct {
	ContentType     string              `xml:"contentType,attr"`
	MimeType        string              `xml:"mimeType,attr"`
	Codecs          string              `xml:"codecs,attr"`
	Lang            string              `xml:"lang,attr"`
	BaseURL         string              `xml:"BaseURL"`
	Template        *mpdSegmentTemplate `xml:"SegmentTemplate"`
	Representations []mpdRepresentation `xml:"Representation"`
}

type mpdDocument struct {
	Duration string `xml:"mediaPresentationDuration,attr"`
	BaseURL  string `xml:"BaseURL"`
	Periods  []struct {
		BaseURL string             `xml:"BaseURL"`
		Sets    []mpdAdaptationSet `xml:"AdaptationSet"`
	} `xml:"Period"`
}

type hlsTrack struct {
	set  mpdAdaptationSet
	rep  mpdRepresentation
	kind string
	base string
}

// dashToHLS returns the master playlist, or the media playlist for representation rep. self is the request URL.
func dashToHLS(manifest []byte, rep string, self *url.URL) ([]byte, error) {
	var doc mpdDocument
	if err := xml.Unmarshal(manifest, &doc); err != nil {
		return nil, fmt.Errorf("parse dash manifest: %w", err)
	}
	if len(doc.Periods) == 0 {
		return nil, fmt.Errorf("dash manifest has no period")
	}
	period := doc.Periods[0]
	var tracks []hlsTrack
	for _, set := range period.Sets {
		for _, r := range set.Representations {
			kind := set.ContentType
			if kind == "" {
				kind, _, _ = strings.Cut(firstNonEmpty(r.MimeType, set.MimeType), "/")
			}
			if kind != "video" && kind != "audio" {
				continue
			}
			base := joinBase(doc.BaseURL, period.BaseURL, set.BaseURL, r.BaseURL)
			tracks = append(tracks, hlsTrack{set: set, rep: r, kind: kind, base: base})
		}
	}
	if rep != "" {
		for _, t := range tracks {
			if t.rep.ID == rep {
				return mediaPlaylist(t, isoSeconds(doc.Duration))
			}
		}
		return nil, fmt.Errorf("dash representation %q not found", rep)
	}
	return masterPlaylist(tracks, self)
}

func masterPlaylist(tracks []hlsTrack, self *url.URL) ([]byte, error) {
	var b strings.Builder
	b.WriteString("#EXTM3U\n#EXT-X-VERSION:7\n#EXT-X-INDEPENDENT-SEGMENTS\n")
	audioCodecs, audioBandwidth, hasAudio := "", int64(0), false
	for _, t := range tracks {
		if t.kind != "audio" {
			continue
		}
		name := firstNonEmpty(t.set.Lang, "Audio "+t.rep.ID)
		def := "NO"
		if !hasAudio {
			def, audioCodecs = "YES", firstNonEmpty(t.rep.Codecs, t.set.Codecs)
		}
		hasAudio = true
		audioBandwidth = max(audioBandwidth, t.rep.Bandwidth)
		fmt.Fprintf(&b, "#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID=\"aud\",NAME=%q,DEFAULT=%s,AUTOSELECT=YES", name, def)
		if t.set.Lang != "" {
			fmt.Fprintf(&b, ",LANGUAGE=%q", t.set.Lang)
		}
		fmt.Fprintf(&b, ",URI=%q\n", variantURL(self, t.rep.ID))
	}
	videos := 0
	for _, t := range tracks {
		if t.kind != "video" {
			continue
		}
		videos++
		codecs := firstNonEmpty(t.rep.Codecs, t.set.Codecs)
		if audioCodecs != "" {
			codecs += "," + audioCodecs
		}
		fmt.Fprintf(&b, "#EXT-X-STREAM-INF:BANDWIDTH=%d,CODECS=%q", t.rep.Bandwidth+audioBandwidth, codecs)
		if t.rep.Width > 0 && t.rep.Height > 0 {
			fmt.Fprintf(&b, ",RESOLUTION=%dx%d", t.rep.Width, t.rep.Height)
		}
		if hasAudio {
			b.WriteString(",AUDIO=\"aud\"")
		}
		b.WriteString("\n" + variantURL(self, t.rep.ID) + "\n")
	}
	if videos == 0 {
		return nil, fmt.Errorf("dash manifest has no video")
	}
	return []byte(b.String()), nil
}

func mediaPlaylist(t hlsTrack, total float64) ([]byte, error) {
	tmpl := t.rep.Template
	if tmpl == nil {
		tmpl = t.set.Template
	}
	if tmpl == nil || tmpl.Media == "" {
		return nil, fmt.Errorf("dash representation %q has no segment template", t.rep.ID)
	}
	scale := float64(max(tmpl.Timescale, 1))
	number := int64(1)
	if tmpl.StartNumber != nil {
		number = *tmpl.StartNumber
	}
	type segment struct {
		number, time int64
		seconds      float64
	}
	var segments []segment
	if len(tmpl.Timeline) > 0 {
		var at int64
		for i, s := range tmpl.Timeline {
			if s.T != nil {
				at = *s.T
			}
			repeat := s.R
			if repeat < 0 {
				// r=-1 repeats to the next S or the end of the presentation.
				end := int64(total * scale)
				if i+1 < len(tmpl.Timeline) && tmpl.Timeline[i+1].T != nil {
					end = *tmpl.Timeline[i+1].T
				}
				repeat = max(0, int64(math.Ceil(float64(end-at)/float64(max(s.D, 1))))-1)
			}
			for n := int64(0); n <= repeat; n++ {
				segments = append(segments, segment{number, at, float64(s.D) / scale})
				number++
				at += s.D
			}
		}
	} else if tmpl.Duration > 0 && total > 0 {
		each := float64(tmpl.Duration) / scale
		count := int64(math.Ceil(total / each))
		for n := int64(0); n < count; n++ {
			seconds := math.Min(each, total-float64(n)*each)
			segments = append(segments, segment{number + n, n * tmpl.Duration, seconds})
		}
	} else {
		return nil, fmt.Errorf("dash representation %q has no segment timing", t.rep.ID)
	}
	target := 1.0
	for _, s := range segments {
		target = math.Max(target, s.seconds)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "#EXTM3U\n#EXT-X-VERSION:7\n#EXT-X-TARGETDURATION:%d\n#EXT-X-PLAYLIST-TYPE:VOD\n#EXT-X-INDEPENDENT-SEGMENTS\n", int(math.Ceil(target)))
	if tmpl.Initialization != "" {
		fmt.Fprintf(&b, "#EXT-X-MAP:URI=%q\n", t.base+fillTemplate(tmpl.Initialization, t.rep, 0, 0))
	}
	for _, s := range segments {
		fmt.Fprintf(&b, "#EXTINF:%.3f,\n%s\n", s.seconds, t.base+fillTemplate(tmpl.Media, t.rep, s.number, s.time))
	}
	b.WriteString("#EXT-X-ENDLIST\n")
	return []byte(b.String()), nil
}

var dashTemplateField = regexp.MustCompile(`\$(RepresentationID|Number|Time|Bandwidth)(%0?\d*d)?\$`)

func fillTemplate(tmpl string, rep mpdRepresentation, number, time int64) string {
	out := dashTemplateField.ReplaceAllStringFunc(tmpl, func(m string) string {
		parts := dashTemplateField.FindStringSubmatch(m)
		format := firstNonEmpty(parts[2], "%d")
		switch parts[1] {
		case "RepresentationID":
			return rep.ID
		case "Number":
			return fmt.Sprintf(format, number)
		case "Time":
			return fmt.Sprintf(format, time)
		default:
			return fmt.Sprintf(format, rep.Bandwidth)
		}
	})
	return strings.ReplaceAll(out, "$$", "$")
}

// joinBase applies nested DASH BaseURLs; an absolute one replaces what came before.
func joinBase(parts ...string) string {
	base := ""
	for _, p := range parts {
		p = strings.TrimSpace(p)
		switch {
		case p == "":
		case strings.Contains(p, "://") || strings.HasPrefix(p, "/"):
			base = p
		default:
			base += p
		}
	}
	return base
}

func variantURL(self *url.URL, rep string) string {
	q := self.Query()
	q.Set("format", "hls")
	q.Set("rep", rep)
	return "video?" + q.Encode()
}

var isoDurationPart = regexp.MustCompile(`([\d.]+)([HMS])`)

// isoSeconds reads the time part of an ISO 8601 duration such as PT23M49.8S.
func isoSeconds(d string) float64 {
	_, timePart, _ := strings.Cut(d, "T")
	total := 0.0
	for _, m := range isoDurationPart.FindAllStringSubmatch(timePart, -1) {
		v, _ := strconv.ParseFloat(m[1], 64)
		total += v * map[string]float64{"H": 3600, "M": 60, "S": 1}[m[2]]
	}
	return total
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
