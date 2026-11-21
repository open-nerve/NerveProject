package bootstrap

import (
	"net/http"
	"testing"
	"time"
	"uuid"
)

// What follows a favorite's target on the real app (H16, M7 P3c spec 2.6,
// 4, R3, R10–R12), each cell a subtest in order on one world: issueWorld's
// acme and Web, in an app whose deletion jobs outlast the test's row locks
// (barriersOutlasted), with Web's cycles, modules and views turned on.
// Seeded so: alice and bob favor each target, the same one, through the
// API (bothFavor), and both favor Web, its cycle Kept, its module Kept,
// bob's view Kept of it and his view Kept of acme's own, which no cell
// touches: their rows stay as they were, byte for byte, through every
// cell (kept). Bob's undeleted favorites of an archived cycle, module and
// project are left behind by SQL (leftFavorite), and the start-time runs
// are waited for before the deletion job is held
// (followsDeletionsThroughTheJob).
func TestFavoritesFollowTheirTargets(t *testing.T) {
	w := newIssueWorldWith(t, barriersOutlasted)
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	w.changeProject(t, w.web, `{"issue_views_view":true}`)
	keptView, _ := w.newView(t, "bob", projectViewsPath(w.web), `{"name":"Kept"}`)
	keptOwn, _ := w.newView(t, "bob", workspaceViewsPath("acme"), `{"name":"Kept"}`)
	var kept []uuid.UUID
	for _, target := range []struct {
		kind string
		id   uuid.UUID
	}{{"project", w.web}, {"cycle", w.newCycle(t, "bob", `{"name":"Kept"}`)}, {"module", w.createModule(t, "bob", `{"name":"Kept"}`, http.StatusCreated).ID},
		{"view", keptView.ID}, {"view", keptOwn.ID}} {
		kept = append(kept, w.bothFavor(t, target.kind, target.id)...)
	}
	rows := favoriteRows(t, w.pool, kept...)
	t.Run("targets", func(t *testing.T) { followsEachTarget(t, w) })
	t.Run("project archive", func(t *testing.T) { followsAProjectsArchive(t, w) })
	t.Run("deletions through the job", func(t *testing.T) { followsDeletionsThroughTheJob(t, w) })
	t.Run("a malformed archive", func(t *testing.T) { refusesAMalformedArchive(t, w) })
	if got := favoriteRows(t, w.pool, kept...); got != rows {
		t.Errorf("the favorites of targets no cell touched changed:\n%s\nwere\n%s", got, rows)
	}
}

// bothFavor is the ids of alice's and bob's new favorites of the target id
// of kind, in acme, at the top.
func (w issueWorld) bothFavor(t *testing.T, kind string, id uuid.UUID) []uuid.UUID {
	t.Helper()
	return []uuid.UUID{w.favorID(t, "alice", "acme", favoriteBody(kind, id)), w.favorID(t, "bob", "acme", favoriteBody(kind, id))}
}

// followsEachTarget is the cascade of each target's deletion and archive
// (spec 2.6, R3, R10, P3b spec 5): bob deletes his cycle and his module,
// archives alice's cycle and module that have ended, and alice deletes
// bob's view of Web and his view of acme's own. Each deletes every
// account's favorites of it, alice's and bob's, as the target's own row
// was written: at its deletion's or archive's moment, last changed by its
// writer, the view's deletion's at the moment and by the writer of the
// view's (deleteView); neither lists them. Unarchived, the cycle and the
// module bring none back. Archived again, a cycle or a module writes
// nothing and publishes nothing: bob's favorite of it left behind by SQL
// meanwhile (leftFavorite) stays undeleted.
func followsEachTarget(t *testing.T, w issueWorld) {
	now := time.Now()
	cycles, modules := "/api/v0/cycles/", "/api/v0/modules/"
	for _, c := range []struct {
		name, kind, table, column, writer string
		make                              func(t *testing.T) uuid.UUID
		trigger                           func(t *testing.T, id uuid.UUID)
		path                              string // an archive's, unarchived and archived again
	}{
		{"cycle deletion", "cycle", "cycles", "deleted_at", "bob", func(t *testing.T) uuid.UUID { return w.newCycle(t, "bob", `{"name":"Mine"}`) },
			deletesFavored(w, "bob", cycles), ""},
		{"module deletion", "module", "modules", "deleted_at", "bob",
			func(t *testing.T) uuid.UUID {
				return w.createModule(t, "bob", `{"name":"Mine"}`, http.StatusCreated).ID
			}, deletesFavored(w, "bob", modules), ""},
		{"cycle archive", "cycle", "cycles", "archived_at", "bob",
			func(t *testing.T) uuid.UUID {
				return w.newCycle(t, "alice", datesBody("Ended", shanghaiDay(t, now, -30), shanghaiDay(t, now, -20)))
			}, archivesFavored(w, "bob", cycles), cycles},
		{"module archive", "module", "modules", "archived_at", "bob",
			func(t *testing.T) uuid.UUID {
				return w.createModule(t, "alice", `{"name":"Ended","status":"completed"}`, http.StatusCreated).ID
			}, archivesFavored(w, "bob", modules), modules},
		{"project view deletion", "view", "issue_views", "deleted_at", "alice",
			func(t *testing.T) uuid.UUID {
				v, _ := w.newView(t, "bob", projectViewsPath(w.web), `{"name":"Mine"}`)
				return v.ID
			},
			deletesFavored(w, "alice", "/api/v0/views/"), ""},
		{"workspace view deletion", "view", "issue_views", "deleted_at", "alice",
			func(t *testing.T) uuid.UUID {
				v, _ := w.newView(t, "bob", workspaceViewsPath("acme"), `{"name":"Mine"}`)
				return v.ID
			},
			deletesFavored(w, "alice", "/api/v0/views/"), ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			target := c.make(t)
			favorites := w.bothFavor(t, c.kind, target)
			c.trigger(t, target)
			at, by := stampOf(t, w.pool, c.table, c.column, target)
			if by != w.ids[c.writer] {
				t.Fatalf("the %s of %s by %s; want %s's", c.column, target, by, c.writer)
			}
			favoritesDeletedAt(t, w.pool, at, by, favorites...)
			for _, actor := range []string{"alice", "bob"} {
				if containsAny(w.listedFavoriteIDs(t, actor, "acme"), favorites...) {
					t.Errorf("%s lists a favorite of the %s %s", actor, c.kind, target)
				}
			}
			if c.path == "" {
				return
			}
			left := leftFavorite(t, w.pool, w.ids["bob"], c.kind, target, w.web)
			w.send(t, "alice", http.MethodPost, c.path+target.String()+"/archive", "", http.StatusOK)
			if again, _ := stampOf(t, w.pool, c.table, c.column, target); !again.Equal(at) {
				t.Fatalf("archived again at %s; was %s", again, at)
			}
			favoritesLive(t, w.pool, left)
			w.send(t, "alice", http.MethodPost, c.path+target.String()+"/unarchive", "", http.StatusOK)
			favoritesDeletedAt(t, w.pool, at, by, favorites...)
			for _, actor := range []string{"alice", "bob"} {
				if containsAny(w.listedFavoriteIDs(t, actor, "acme"), favorites...) {
					t.Errorf("%s lists a favorite of the %s %s, unarchived", actor, c.kind, target)
				}
			}
		})
	}
}

// deletesFavored is actor's deletion of a target by its id under prefix, its
// route's.
func deletesFavored(w issueWorld, actor, prefix string) func(t *testing.T, id uuid.UUID) {
	return func(t *testing.T, id uuid.UUID) {
		w.send(t, actor, http.MethodDelete, prefix+id.String(), "", http.StatusNoContent)
	}
}

// archivesFavored is actor's archive of a target by its id under prefix, its
// route's.
func archivesFavored(w issueWorld, actor, prefix string) func(t *testing.T, id uuid.UUID) {
	return func(t *testing.T, id uuid.UUID) {
		w.send(t, actor, http.MethodPost, prefix+id.String()+"/archive", "", http.StatusOK)
	}
}
