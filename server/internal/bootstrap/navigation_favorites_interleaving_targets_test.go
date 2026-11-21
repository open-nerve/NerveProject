package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The targets' groups of TestFavoriteWritesInterleaveWithArchives
// (navigation_favorites_interleaving_test.go).

// favoriteFate is what becomes of a target's favorites once the write that
// decides the target has committed.
type favoriteFate int

const (
	// deletedWithIt: its own transaction deletes every account's favorites
	// of the target, at its moment, by its writer (spec 2.6: the cascades of
	// the archives and of the deletions, deleteView's).
	deletedWithIt favoriteFate = iota
	// deletedByItsJob: the project's deletion job deletes them, at the
	// deletion's moment, by its deleter (spec 2.6, R11); until then the
	// reads hide them (R14).
	deletedByItsJob
	// keptHidden: they stay, and the reads hide them (spec 2.2: a member's
	// removal from the project).
	keptHidden
)

// favoriteCompetitor is a write that decides a favorite's target, raced
// with writer's creation, change and deletion of his favorite of it, a
// target of kind: make makes the cell n's own target and returns it, the
// row of table both writes lock, which the other transaction holds with
// lock, and the competing write, verb, which answers answers; the
// favorite's write waits for that row at waits. stamp reads, once it is
// written, the moment and the writer of the deletion of the target's
// favorites, for the fates that delete them.
type favoriteCompetitor struct {
	name, verb, writer, kind string
	table, lock, waits       string
	answers                  int
	fate                     favoriteFate
	make                     func(t *testing.T, n int) (target, held uuid.UUID, competitor favoriteRacer)
	stamp                    func(t *testing.T, target uuid.UUID) (time.Time, uuid.UUID)
}

// favoriteCompetitors are, each alice's through the app's API:
//   - The archive (archiveProject) and the deletion (deleteProject) of a
//     project of hers in acme, her favorite of it raced, waiting for the
//     project's row; the archive's cascade deletes it with the archive, the
//     deletion's job after it.
//   - The archive of a cycle of Web's that has ended (archiveCycle) and the
//     deletion of another (deleteCycle), her favorite of it raced, waiting
//     for Web's row, which the planning writes lock (spec 2.8); each one's
//     cascade deletes it with it.
//   - The deletion of her view of Web (deleteView), her favorite of it
//     raced, waiting for the view's row; it deletes the favorite with the
//     view, at its moment.
//   - The removal of bob from a project of hers, his favorite of it raced,
//     waiting for the project's row; the favorite stays, hidden from him.
func favoriteCompetitors(w issueWorld) []favoriteCompetitor {
	project := func(t *testing.T, name, identifier string) uuid.UUID {
		return createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", name, identifier)
	}
	alices := func(method, path, statement string) favoriteRacer {
		return favoriteRacer{actor: "alice", method: method, path: path, statement: statement}
	}
	stamped := func(table, column string) func(t *testing.T, target uuid.UUID) (time.Time, uuid.UUID) {
		return func(t *testing.T, target uuid.UUID) (time.Time, uuid.UUID) {
			return stampOf(t, w.pool, table, column, target)
		}
	}
	now := time.Now()
	return []favoriteCompetitor{
		{name: "a project's archive", verb: "the archive", writer: "alice", kind: "project", table: "projects", lock: "FOR NO KEY UPDATE",
			waits: "ShareNavigationProject", answers: http.StatusOK, fate: deletedWithIt, stamp: stamped("projects", "archived_at"),
			make: func(t *testing.T, n int) (uuid.UUID, uuid.UUID, favoriteRacer) {
				p := project(t, fmt.Sprintf("Archived %d", n), fmt.Sprintf("ARC%d", n))
				return p, p, alices(http.MethodPost, "/api/v0/projects/"+p.String()+"/archive", "LockProject")
			}},
		{name: "a cycle's archive", verb: "the archive", writer: "alice", kind: "cycle", table: "projects", lock: "FOR NO KEY UPDATE",
			waits: "ShareNavigationProject", answers: http.StatusOK, fate: deletedWithIt, stamp: stamped("cycles", "archived_at"),
			make: func(t *testing.T, n int) (uuid.UUID, uuid.UUID, favoriteRacer) {
				c := w.newCycle(t, "alice", datesBody(fmt.Sprintf("Ended %d", n), shanghaiDay(t, now, 10*n-400), shanghaiDay(t, now, 10*n-397)))
				return c, w.web, alices(http.MethodPost, "/api/v0/cycles/"+c.String()+"/archive", "LockPlanningProject")
			}},
		{name: "a cycle's deletion", verb: "the deletion", writer: "alice", kind: "cycle", table: "projects", lock: "FOR NO KEY UPDATE",
			waits: "ShareNavigationProject", answers: http.StatusNoContent, fate: deletedWithIt, stamp: stamped("cycles", "deleted_at"),
			make: func(t *testing.T, n int) (uuid.UUID, uuid.UUID, favoriteRacer) {
				c := w.newCycle(t, "alice", fmt.Sprintf(`{"name":"Deleted %d"}`, n))
				return c, w.web, alices(http.MethodDelete, "/api/v0/cycles/"+c.String(), "LockPlanningProject")
			}},
		{name: "a view's deletion", verb: "the deletion", writer: "alice", kind: "view", table: "issue_views", lock: "FOR UPDATE",
			waits: "ShareView", answers: http.StatusNoContent, fate: deletedWithIt, stamp: stamped("issue_views", "deleted_at"),
			make: func(t *testing.T, n int) (uuid.UUID, uuid.UUID, favoriteRacer) {
				v, _ := w.newView(t, "alice", projectViewsPath(w.web), fmt.Sprintf(`{"name":"Deleted %d"}`, n))
				return v.ID, v.ID, alices(http.MethodDelete, viewPath(v.ID), "LockView")
			}},
		{name: "a project's deletion", verb: "the deletion", writer: "alice", kind: "project", table: "projects", lock: "FOR NO KEY UPDATE",
			waits: "ShareNavigationProject", answers: http.StatusNoContent, fate: deletedByItsJob,
			stamp: func(t *testing.T, target uuid.UUID) (time.Time, uuid.UUID) {
				waitProjectDeletion(t, w.pool, target)
				return stampOf(t, w.pool, "projects", "deleted_at", target)
			},
			make: func(t *testing.T, n int) (uuid.UUID, uuid.UUID, favoriteRacer) {
				p := project(t, fmt.Sprintf("Deleted %d", n), fmt.Sprintf("DEL%d", n))
				return p, p, alices(http.MethodDelete, "/api/v0/projects/"+p.String(), "LockProject")
			}},
		{name: "a member's removal", verb: "the removal", writer: "bob", kind: "project", table: "projects", lock: "FOR NO KEY UPDATE",
			waits: "ShareNavigationProject", answers: http.StatusNoContent, fate: keptHidden,
			make: func(t *testing.T, n int) (uuid.UUID, uuid.UUID, favoriteRacer) {
				p := project(t, fmt.Sprintf("Left %d", n), fmt.Sprintf("LFT%d", n))
				w.makeMember(t, "bob", p, shared.RoleMember)
				return p, p, alices(http.MethodDelete, "/api/v0/project-members/"+projectMemberships(t, w.pool, w.ids["bob"], p)[0].String(), "LockProject")
			}},
	}
}

// favoritesRaceATarget races c with its writer's creation, change and
// deletion of his favorite of a target of c's, each in both orders, each
// cell on a target of its own.
func favoritesRaceATarget(t *testing.T, w issueWorld, c favoriteCompetitor) {
	n := 0
	for _, write := range []string{"created", "changed", "deleted"} {
		for _, ours := range []bool{false, true} {
			n++
			first := c.verb + " first"
			if ours {
				first = "the favorite's write first"
			}
			t.Run("its favorite "+write+", "+first, func(t *testing.T) { raceAFavorite(t, w, c, n, write, ours) })
		}
	}
}

// raceAFavorite is one cell of c's: its writer's favorite of the cell's
// target, of c.kind, written (write: created, changed to sequence 7, or
// deleted), first when ours is set, else second, each write waiting for
// the row the other transaction holds at its statement, the favorite's
// holding no lock on its own row meanwhile (spec 2.8).
//
// c's write answers as it would alone. Second, the favorite's write
// decides under its locks on what c's left: a creation is refused, the one
// 422 of a target one may not favor, byte for byte (spec 2.5, R7); a
// change and a deletion are 404 navigation.favorite_not_found, byte for
// byte as an absent favorite's (spec 2.7). First, it is written: 201, 200
// with its sequence, 204. Then: when c deletes its target's favorites, none
// is left undeleted, the one written first among them, at c's moment and
// by its writer, or by the job at the deletion's, as c wrote it; one the
// favorite's write deleted first keeps its own deletion, by its writer,
// before c's moment. When c keeps them, one refused under its lock is as
// before, byte for byte, one written is as written, and the writer's list
// shows neither. A creation refused left no favorite of the target.
func raceAFavorite(t *testing.T, w issueWorld, c favoriteCompetitor, n int, write string, ours bool) {
	target, held, competitor := c.make(t, n)
	racer := favoriteRacer{actor: c.writer, statement: c.waits}
	var favorite uuid.UUID
	before := ""
	if write == "created" {
		racer.method, racer.path, racer.body = http.MethodPost, favoritesPath("acme"), favoriteBody(c.kind, target)
	} else {
		favorite = w.favorID(t, c.writer, "acme", favoriteBody(c.kind, target))
		racer.method, racer.path, racer.free = http.MethodDelete, favoritePath(favorite), []lockedRow{favoriteRowLocked(favorite)}
		if write == "changed" {
			racer.method, racer.body = http.MethodPatch, `{"sequence":7}`
		}
		before = favoriteRows(t, w.pool, favorite)
	}
	var mine, theirs racedAnswer
	if ours {
		mine, theirs = w.race(t, c.table, c.lock, held, racer, competitor)
	} else {
		theirs, mine = w.race(t, c.table, c.lock, held, competitor, racer)
	}
	answeredRace(t, theirs, c.answers)
	var at time.Time
	var by uuid.UUID
	if c.fate != keptHidden {
		at, by = c.stamp(t, target)
	}
	switch {
	case !ours && write == "created":
		if want := w.noTarget(t, c.writer, "acme"); mine.whole != want {
			t.Errorf("POST %s once %s has committed:\n%+v\nwant the one 422\n%+v", mine.req.URL.Path, c.verb, mine.whole, want)
		}
	case !ours:
		w.refusedRaceAsAbsent(t, mine, c.writer, racer.body)
	case write == "deleted":
		answeredRace(t, mine, http.StatusNoContent)
	default:
		var f favoriteHTTP
		decodeAnswer(t, answeredRace(t, mine, map[string]int{"created": http.StatusCreated, "changed": http.StatusOK}[write]), &f)
		if write == "changed" && (f.ID != favorite || f.Sequence != 7) {
			t.Errorf("changed %+v; want the favorite %s at sequence 7", f, favorite)
		}
		favorite = f.ID
	}
	switch {
	case favorite == (uuid.UUID{}):
		if left := liveFavoritesOf(t, w.pool, target); left != 0 {
			t.Errorf("%d undeleted favorites of %s once the creation was refused; want none", left, target)
		}
		return
	case ours && write == "deleted":
		favoriteDeletedBefore(t, w.pool, favorite, w.ids[c.writer], at)
	case c.fate != keptHidden:
		favoritesDeletedAt(t, w.pool, at, by, favorite)
	case !ours:
		if got := favoriteRows(t, w.pool, favorite); got != before {
			t.Errorf("the favorite refused under its lock is\n%s\nwas\n%s", got, before)
		}
	default:
		favoritesLive(t, w.pool, favorite)
	}
	if c.fate != keptHidden {
		if left := liveFavoritesOf(t, w.pool, target); left != 0 {
			t.Errorf("%d undeleted favorites of %s once %s has committed; want none", left, target, c.verb)
		}
	} else if slices.Contains(w.listedFavoriteIDs(t, c.writer, "acme"), favorite) {
		t.Errorf("%s lists his favorite %s of a project he was removed from", c.writer, favorite)
	}
}
