package mokusync

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type OpType string

const (
	// OpSet writes an LWW register.
	OpSet OpType = "set"
	// OpAdd adds an element to an observed-remove set under the op's own tag.
	OpAdd OpType = "add"
	// OpRemove removes the add tags it observed; a concurrent add under a new tag survives.
	OpRemove OpType = "remove"
)

// Item kinds. Keys join extension package and source IDs with KeySep.
const (
	KindChapter      = "chapter"       // key: package, media ID, chapter ID; fields completed, position, bookmarked
	KindLibrary      = "library"       // set with key "", elements are media keys
	KindFolder       = "folder"        // key: folder sync ID; fields name, sort_order, include_in_update, include_in_download, content_types, deleted
	KindFolderMember = "folder_member" // set keyed by folder sync ID, elements are media keys
)

const KeySep = "\x1f"

// Op is one change. Origin and Seq identify it everywhere; Clock orders it against concurrent changes.
type Op struct {
	Origin string          `json:"o"`
	Seq    int64           `json:"n"`
	Clock  string          `json:"c"`
	Kind   string          `json:"k"`
	Key    string          `json:"key"`
	Field  string          `json:"f"`
	Type   OpType          `json:"t"`
	Value  json.RawMessage `json:"v,omitempty"`
	Tags   []string        `json:"g,omitempty"`
}

// Tag names this op as an add in an observed-remove set.
func (o Op) Tag() string { return o.Origin + ":" + strconv.FormatInt(o.Seq, 10) }

func parseTag(tag string) (string, int64, error) {
	i := strings.LastIndexByte(tag, ':')
	if i < 0 {
		return "", 0, fmt.Errorf("tag %q: want origin:seq", tag)
	}
	seq, err := strconv.ParseInt(tag[i+1:], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("tag %q: %w", tag, err)
	}
	return tag[:i], seq, nil
}

func (o Op) validate() error {
	switch {
	case o.Origin == "" || o.Seq <= 0:
		return fmt.Errorf("op without origin or seq")
	case o.Kind == "" || o.Field == "":
		return fmt.Errorf("op %s: missing kind or field", o.Tag())
	}
	if _, err := ParseClock(o.Clock); err != nil {
		return fmt.Errorf("op %s: %w", o.Tag(), err)
	}
	switch o.Type {
	case OpSet:
		if !json.Valid(o.Value) {
			return fmt.Errorf("op %s: invalid value", o.Tag())
		}
	case OpAdd:
	case OpRemove:
		for _, t := range o.Tags {
			if _, _, err := parseTag(t); err != nil {
				return fmt.Errorf("op %s: %w", o.Tag(), err)
			}
		}
	default:
		return fmt.Errorf("op %s: unknown type %q", o.Tag(), o.Type)
	}
	return nil
}

// VersionVector holds the highest seq seen from each origin.
type VersionVector map[string]int64

// Lacks reports whether v is missing ops that other has.
func (v VersionVector) Lacks(other VersionVector) bool {
	for origin, seq := range other {
		if v[origin] < seq {
			return true
		}
	}
	return false
}
