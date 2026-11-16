// Package mokusync syncs a Tsunagu library between instances with CRDTs: LWW registers for single values,
// observed-remove sets for memberships, and version vectors so any instance can sync with any other.
package mokusync

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Clock is a hybrid logical clock stamp. Its string form sorts in clock order.
type Clock struct {
	Wall    int64 // Unix milliseconds
	Counter int64
	Node    string
}

const maxCounter = 999999

func (c Clock) String() string { return fmt.Sprintf("%013d.%06d.%s", c.Wall, c.Counter, c.Node) }

// Less orders clocks by wall time, then counter, then node, so every device picks the same winner.
func (c Clock) Less(o Clock) bool {
	if c.Wall != o.Wall {
		return c.Wall < o.Wall
	}
	if c.Counter != o.Counter {
		return c.Counter < o.Counter
	}
	return c.Node < o.Node
}

func ParseClock(s string) (Clock, error) {
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return Clock{}, fmt.Errorf("clock %q: want wall.counter.node", s)
	}
	wall, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Clock{}, fmt.Errorf("clock %q wall: %w", s, err)
	}
	counter, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return Clock{}, fmt.Errorf("clock %q counter: %w", s, err)
	}
	return Clock{Wall: wall, Counter: counter, Node: parts[2]}, nil
}

// HLC issues stamps for one node. A stamp is never behind any stamp the node issued or observed,
// so a device with a slow wall clock still orders its edits after what it has already seen.
type HLC struct {
	mu   sync.Mutex
	node string
	last Clock
}

func NewHLC(node string, last Clock) *HLC {
	return &HLC{node: node, last: Clock{Wall: last.Wall, Counter: last.Counter, Node: node}}
}

// Now stamps a local event that happened at wall (Unix ms).
func (h *HLC) Now(wall int64) Clock {
	h.mu.Lock()
	defer h.mu.Unlock()
	if wall > h.last.Wall {
		h.last = Clock{Wall: wall, Node: h.node}
	} else if h.last.Counter < maxCounter {
		h.last = Clock{Wall: h.last.Wall, Counter: h.last.Counter + 1, Node: h.node}
	} else {
		h.last = Clock{Wall: h.last.Wall + 1, Node: h.node}
	}
	return h.last
}

// Observe moves the clock past a stamp received from another node.
func (h *HLC) Observe(c Clock) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.Wall > h.last.Wall || (c.Wall == h.last.Wall && c.Counter > h.last.Counter) {
		h.last = Clock{Wall: c.Wall, Counter: c.Counter, Node: h.node}
	}
}

func (h *HLC) Last() Clock {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.last
}
