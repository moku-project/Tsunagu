package rest

import (
	"container/list"
	"sync"
)

type segCache struct {
	mu   sync.Mutex
	max  int64
	size int64
	ll   *list.List
	m    map[string]*list.Element
}

type segEntry struct {
	key  string
	data []byte
	ct   string
}

func newSegCache(maxBytes int64) *segCache {
	return &segCache{max: maxBytes, ll: list.New(), m: map[string]*list.Element{}}
}

func (c *segCache) get(key string) (data []byte, ct string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.m[key]
	if !ok {
		return nil, "", false
	}
	c.ll.MoveToFront(el)
	e := el.Value.(*segEntry)
	return e.data, e.ct, true
}

func (c *segCache) put(key string, data []byte, ct string) {
	n := int64(len(data))
	if n == 0 || n > c.max {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.m[key]; ok {
		e := el.Value.(*segEntry)
		c.size += n - int64(len(e.data))
		e.data, e.ct = data, ct
		c.ll.MoveToFront(el)
	} else {
		c.m[key] = c.ll.PushFront(&segEntry{key: key, data: data, ct: ct})
		c.size += n
	}
	for c.size > c.max {
		back := c.ll.Back()
		if back == nil {
			break
		}
		e := back.Value.(*segEntry)
		c.size -= int64(len(e.data))
		delete(c.m, e.key)
		c.ll.Remove(back)
	}
}
