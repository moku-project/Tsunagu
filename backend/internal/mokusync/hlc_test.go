package mokusync

import (
	"math/rand"
	"sort"
	"testing"
)

func TestClockNeverGoesBackwards(t *testing.T) {
	h := NewHLC("a", Clock{})
	prev := h.Now(1000)
	for _, wall := range []int64{1000, 999, 500, 1000, 1001, 1001, 2} {
		next := h.Now(wall)
		if !prev.Less(next) {
			t.Fatalf("stamp %v is not after %v (wall %d)", next, prev, wall)
		}
		prev = next
	}
}

func TestObserveMovesPastRemoteStamps(t *testing.T) {
	h := NewHLC("slow", Clock{})
	remote := Clock{Wall: 50_000, Counter: 7, Node: "fast"}
	h.Observe(remote)
	if next := h.Now(10); !remote.Less(next) {
		t.Fatalf("local stamp %v after observing %v should come later", next, remote)
	}
}

func TestCounterOverflowAdvancesWall(t *testing.T) {
	h := NewHLC("a", Clock{Wall: 10, Counter: maxCounter})
	if c := h.Now(5); c.Wall != 11 || c.Counter != 0 {
		t.Fatalf("got %v, want wall 11 counter 0", c)
	}
}

func TestClockStringsSortInClockOrder(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	clocks := make([]Clock, 500)
	for i := range clocks {
		clocks[i] = Clock{Wall: r.Int63n(1 << 42), Counter: r.Int63n(maxCounter + 1), Node: []string{"a1", "b2", "c3"}[r.Intn(3)]}
	}
	byLess := append([]Clock{}, clocks...)
	sort.Slice(byLess, func(i, j int) bool { return byLess[i].Less(byLess[j]) })
	byString := append([]Clock{}, clocks...)
	sort.Slice(byString, func(i, j int) bool { return byString[i].String() < byString[j].String() })
	for i := range byLess {
		if byLess[i] != byString[i] {
			t.Fatalf("order differs at %d: %v vs %v", i, byLess[i], byString[i])
		}
		back, err := ParseClock(byLess[i].String())
		if err != nil || back != byLess[i] {
			t.Fatalf("round trip of %v gave %v, %v", byLess[i], back, err)
		}
	}
}

func TestOpValidationRejectsMalformedOps(t *testing.T) {
	good := Op{Origin: "a", Seq: 1, Clock: Clock{Wall: 1, Node: "a"}.String(), Kind: KindChapter, Key: "k", Field: "completed", Type: OpSet, Value: []byte("true")}
	if err := good.validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Op){
		"no origin": func(o *Op) { o.Origin = "" },
		"zero seq":  func(o *Op) { o.Seq = 0 },
		"bad clock": func(o *Op) { o.Clock = "soon" },
		"bad value": func(o *Op) { o.Value = []byte("{") },
		"bad type":  func(o *Op) { o.Type = "merge" },
		"no field":  func(o *Op) { o.Field = "" },
		"bad tag":   func(o *Op) { o.Type, o.Tags = OpRemove, []string{"nocolon"} },
	} {
		op := good
		mutate(&op)
		if op.validate() == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
