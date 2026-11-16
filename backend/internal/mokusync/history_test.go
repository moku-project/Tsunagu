package mokusync

import "testing"

func (n *node) addSession(id string, m, c int, endedAt int64) {
	n.exec(`INSERT INTO reading_sessions (id, media_id, start_chapter_id, end_chapter_id, started_at, ended_at, duration_ms, chapters)
		VALUES (?, `+mediaID+`, `+chapterIDs+`, `+chapterIDs+`, ?, ?, 60000, 1)`,
		id, mediaExt(m), mediaExt(m), chapterExt(c), mediaExt(m), chapterExt(c), endedAt-60000, endedAt)
}

func (n *node) sessions() []string {
	return n.strings(`SELECT s.id || ' ' || m.external_id || COALESCE(c.external_id, ' -') || ' hidden=' || s.hidden
		FROM reading_sessions s JOIN media m ON m.id = s.media_id LEFT JOIN chapters c ON c.id = s.end_chapter_id ORDER BY s.id`)
}

func TestSessionsSyncBetweenDevices(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", []int{5, 4, 3, 2, 1, 0})
	a.addSession("s1", 2, 3, 1_700_000_100_000)
	a.addSession("s2", 4, 0, 1_700_000_200_000)
	a.capture()
	a.syncWith(b)
	if d := diff(a.sessions(), b.sessions()); d != "" || len(b.sessions()) != 2 {
		t.Fatalf("sessions differ:\n%s\n%v", d, b.sessions())
	}

	// Clearing history on one device hides it on the other; deleting removes it.
	b.wall++
	b.exec(`UPDATE reading_sessions SET hidden = 1 WHERE id = 's1'`)
	b.capture()
	a.wall += 2
	a.exec(`DELETE FROM reading_sessions WHERE id = 's2'`)
	a.capture()
	a.syncWith(b)
	if d := diff(a.sessions(), b.sessions()); d != "" || len(a.sessions()) != 1 {
		t.Fatalf("after hide and delete:\n%s\n%v", d, a.sessions())
	}

	// A title deleted on one device takes its sessions there, not everywhere.
	b.exec(`DELETE FROM media WHERE external_id = ?`, mediaExt(2))
	b.capture()
	b.syncWith(a)
	if len(a.sessions()) != 1 {
		t.Fatalf("deleting a title on b deleted its history on a: %v", a.sessions())
	}
}

func TestSessionsWaitForTheirTitle(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", []int{0})
	a.addSession("s1", 3, 2, 1_700_000_100_000)
	a.exec(`INSERT INTO reading_days_imported (source, day, seconds, chapters) VALUES ('moku', '2026-01-02', 900, 3)`)
	a.capture()
	a.syncWith(b)
	if len(b.sessions()) != 0 {
		t.Fatalf("a session landed without its title: %v", b.sessions())
	}
	if got := b.strings(`SELECT source || ' ' || day || ' ' || seconds FROM reading_days_imported`); len(got) != 1 || got[0] != "moku 2026-01-02 900" {
		t.Fatalf("imported days %v", got)
	}
	b.addTitle(3)
	if _, err := b.e.ProjectPending(b.ctx); err != nil {
		t.Fatal(err)
	}
	if d := diff(a.sessions(), b.sessions()); d != "" {
		t.Fatalf("after the title arrived:\n%s", d)
	}
	if got := b.capture(); got != 0 {
		t.Fatalf("applying a synced session made %d new ops", got)
	}
}
