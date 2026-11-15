package mokusync

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
)

// TestRandomHistoriesConverge runs random edits on several devices with skewed clocks, syncs random
// pairs in random order, then checks every device ends with the same sync state and the same library.
func TestRandomHistoriesConverge(t *testing.T) {
	topologies := map[string]func(i, j, n int) bool{
		// Any device may sync with any other.
		"mesh": func(i, j, n int) bool { return true },
		// Devices only reach their neighbours, so changes must be forwarded.
		"chain": func(i, j, n int) bool { return j == i+1 || i == j+1 },
		// Phones only ever reach the hub (device 0).
		"hub": func(i, j, n int) bool { return i == 0 || j == 0 },
	}
	// MOKU_SYNC_SEEDS raises the count for a longer soak run.
	seeds := 25
	if v, err := strconv.Atoi(os.Getenv("MOKU_SYNC_SEEDS")); err == nil && v > 0 {
		seeds = v
	} else if testing.Short() {
		seeds = 5
	}
	for name, reach := range topologies {
		for seed := int64(1); seed <= int64(seeds); seed++ {
			t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
				runHistory(t, seed, 4, 250, reach)
			})
		}
	}
}

// folderNames is the pool random folder names come from; a small pool makes devices merge folders often.
var folderNames = 100

func TestRandomHistoriesConvergeWithFrequentFolderMerges(t *testing.T) {
	defer func(n int) { folderNames = n }(folderNames)
	folderNames = 4
	seeds := 25
	if v, err := strconv.Atoi(os.Getenv("MOKU_SYNC_SEEDS")); err == nil && v > 0 {
		seeds = v
	} else if testing.Short() {
		seeds = 5
	}
	for seed := int64(500); seed < 500+int64(seeds); seed++ {
		t.Run(fmt.Sprintf("seed%d", seed), func(t *testing.T) {
			runHistory(t, seed, 4, 250, func(i, j, n int) bool { return true })
		})
	}
}

func TestRandomHistoriesConvergeInSmallBatches(t *testing.T) {
	defer func(l int) { batchLimit = l }(batchLimit)
	batchLimit = 4
	for seed := int64(100); seed < 108; seed++ {
		t.Run(fmt.Sprintf("seed%d", seed), func(t *testing.T) {
			runHistory(t, seed, 3, 200, func(i, j, n int) bool { return true })
		})
	}
}

func runHistory(t *testing.T, seed int64, devices, steps int, reach func(i, j, n int) bool) {
	r := rand.New(rand.NewSource(seed))
	nodes := make([]*node, devices)
	skew := make([]int64, devices)
	for i := range nodes {
		nodes[i] = newNode(t, fmt.Sprintf("d%d", i), r.Perm(testTitles))
		skew[i] = int64(r.Intn(20_000) - 10_000)
	}
	now := int64(1_700_000_000_000)
	var log []string

	for step := 0; step < steps; step++ {
		now += int64(r.Intn(50))
		i := r.Intn(devices)
		n := nodes[i]
		if r.Intn(10) < 3 {
			j := r.Intn(devices)
			if j != i && reach(i, j, devices) {
				log = append(log, fmt.Sprintf("%d: d%d syncs with d%d", step, i, j))
				n.syncWith(nodes[j])
			}
			continue
		}
		n.wall = now + skew[i]
		log = append(log, fmt.Sprintf("%d: d%d %s", step, i, randomEdit(r, n)))
		n.capture()
	}

	// Settle: every reachable pair syncs, twice over, so changes cross any path.
	for pass := 0; pass < devices+1; pass++ {
		for i := range nodes {
			for j := range nodes {
				if i != j && reach(i, j, devices) {
					nodes[i].syncWith(nodes[j])
				}
			}
		}
	}

	want, wantLib := nodes[0].crdtState(), nodes[0].libraryState()
	for i, n := range nodes[1:] {
		if d := diff(want, n.crdtState()); d != "" {
			t.Fatalf("sync state of d%d differs from d0:\n%s\n\nhistory:\n%s", i+1, d, tail(log, 40))
		}
		if d := diff(wantLib, n.libraryState()); d != "" {
			t.Fatalf("library of d%d differs from d0:\n%s\n\nhistory:\n%s", i+1, d, tail(log, 40))
		}
	}
	for i, n := range nodes {
		if got := n.capture(); got != 0 {
			t.Fatalf("d%d still had %d uncaptured changes after settling", i, got)
		}
	}
}

func randomEdit(r *rand.Rand, n *node) string {
	m, c := r.Intn(testTitles), r.Intn(testChapters)
	folders := n.folderIDs()
	switch k := r.Intn(11); {
	case k == 0:
		read := r.Intn(3) > 0
		n.setRead(m, c, read)
		return fmt.Sprintf("read %d/%d=%v", m, c, read)
	case k == 1:
		p := float64(r.Intn(100)) / 100
		n.setPosition(m, c, p)
		return fmt.Sprintf("position %d/%d=%v", m, c, p)
	case k == 2:
		s := float64(r.Intn(1440))
		n.setEpisodePosition(m, c, s, 1440)
		return fmt.Sprintf("episode %d/%d at %vs", m, c, s)
	case k == 3:
		b := r.Intn(2) == 0
		n.setBookmark(m, c, b)
		return fmt.Sprintf("bookmark %d/%d=%v", m, c, b)
	case k <= 5:
		in := r.Intn(3) > 0
		n.setInLibrary(m, in)
		return fmt.Sprintf("library %d=%v", m, in)
	case k == 6 || len(folders) == 0:
		id := n.createFolder(fmt.Sprintf("F%d", r.Intn(folderNames)))
		return "create folder " + id[:6]
	case k == 7:
		id := folders[r.Intn(len(folders))]
		name := fmt.Sprintf("F%d", r.Intn(folderNames))
		n.renameFolder(id, name)
		return fmt.Sprintf("rename %s=%s", id[:6], name)
	case k == 8 && r.Intn(3) == 0:
		id := folders[r.Intn(len(folders))]
		n.deleteFolder(id)
		return "delete folder " + id[:6]
	default:
		id := folders[r.Intn(len(folders))]
		in := r.Intn(3) > 0
		n.setInFolder(id, m, in)
		return fmt.Sprintf("folder %s has %d=%v", id[:6], m, in)
	}
}

func tail(lines []string, n int) string {
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}
