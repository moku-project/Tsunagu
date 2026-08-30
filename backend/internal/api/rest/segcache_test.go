package rest

import (
	"bytes"
	"testing"
)

func TestSegCacheLRUEviction(t *testing.T) {
	c := newSegCache(100)
	c.put("a", bytes.Repeat([]byte{1}, 40), "video/mp4")
	c.put("b", bytes.Repeat([]byte{2}, 40), "video/mp4")
	if _, _, ok := c.get("a"); !ok {
		t.Fatal("a should still be cached")
	}

	c.put("c", bytes.Repeat([]byte{3}, 40), "video/mp4")
	if _, _, ok := c.get("b"); ok {
		t.Error("b should have been evicted")
	}
	if _, _, ok := c.get("a"); !ok {
		t.Error("a should survive (was touched)")
	}
	if _, ct, ok := c.get("c"); !ok || ct != "video/mp4" {
		t.Error("c missing or wrong content-type")
	}
}

func TestSegCacheRejectsOversized(t *testing.T) {
	c := newSegCache(50)
	c.put("big", bytes.Repeat([]byte{0}, 100), "x")
	if _, _, ok := c.get("big"); ok {
		t.Error("object larger than cache must not be stored")
	}
}
