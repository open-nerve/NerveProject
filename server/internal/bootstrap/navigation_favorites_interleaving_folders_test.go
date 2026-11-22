package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"uuid"
)

// The folders' group of TestFavoriteWritesInterleaveWithArchives
// (navigation_favorites_interleaving_test.go).

// favoritesRaceAFolder races alice's deletion of a folder of hers with her
// write of her favorite of a view of acme's own: a move of it into the
// folder, its creation in the folder, and, the favorite made before the
// folder, so that its id is the lower, then moved into it, its change that
// names the folder its parent again. Each in both orders, each cell on a
// folder and a view of its own.
func favoritesRaceAFolder(t *testing.T, w issueWorld) {
	n := 0
	for _, write := range []string{"moved into it", "created in it", "in it changed, naming it again"} {
		for _, ours := range []bool{false, true} {
			n++
			first := "the deletion first"
			if ours {
				first = "the favorite's write first"
			}
			t.Run("a favorite "+write+", "+first, func(t *testing.T) { raceAFolder(t, w, n, write, ours) })
		}
	}
}

// raceAFolder is one cell of favoritesRaceAFolder's (spec 2.5, 2.8, re-91
// I2): the other transaction holds the folder's row FOR UPDATE; the
// deletion waits for it at LockFavorite, the favorite's write at
// ShareFolder, its parent's FOR SHARE, holding no lock on its own row
// meanwhile, a folder's row before an item's in the total order of
// favorites' rows. Neither answers 500: no deadlock (40P01) of the two,
// though the favorite changed is in the folder and its id the lower, which
// a lock of rows in id order would take before the folder's.
//
// The deletion answers 204, the folder deleted by alice. Second, the
// favorite's write decides under its locks on the folder deleted: a move
// into it and a creation in it are refused, 422 parent_id not_allowed, the
// favorite as before, no favorite made; the change finds its favorite
// deleted with the folder, 404 navigation.favorite_not_found byte for byte
// as an absent favorite's. First, it is written, 201 or 200, in the folder,
// and the deletion deletes it with the folder, at its moment, by her. In
// every order no favorite in the folder is left undeleted: the purge's
// foreign key would delete such a one with the folder.
func raceAFolder(t *testing.T, w issueWorld, n int, write string, ours bool) {
	view, _ := w.newView(t, "alice", workspaceViewsPath("acme"), fmt.Sprintf(`{"name":"Filed %d"}`, n))
	racer := racedWrite{actor: "alice", statement: "ShareFolder"}
	var folder, item uuid.UUID
	if write == "created in it" {
		folder = w.favorID(t, "alice", "acme", folderBody(fmt.Sprintf("Folder %d", n)))
		racer.method, racer.path, racer.body = http.MethodPost, favoritesPath("acme"), favoriteBody("view", view.ID, folder)
	} else {
		item = w.favorID(t, "alice", "acme", favoriteBody("view", view.ID))
		folder = w.favorID(t, "alice", "acme", folderBody(fmt.Sprintf("Folder %d", n)))
		racer.method, racer.path, racer.body = http.MethodPatch, favoritePath(item), fmt.Sprintf(`{"parent_id":%q}`, folder)
		racer.free = []lockedRow{favoriteRowLocked(item)}
		if write != "moved into it" {
			if item.Compare(folder) >= 0 {
				t.Fatalf("the favorite %s made before the folder %s has the greater id", item, folder)
			}
			w.changeFavorite(t, "alice", item, racer.body)
			racer.body = fmt.Sprintf(`{"parent_id":%q,"sequence":3}`, folder)
		}
	}
	before := ""
	if item != (uuid.UUID{}) {
		before = favoriteRows(t, w.pool, item)
	}
	deletion := racedWrite{actor: "alice", method: http.MethodDelete, path: favoritePath(folder), statement: "LockFavorite"}
	var mine, theirs racedAnswer
	if ours {
		mine, theirs = w.race(t, "user_favorites", "FOR UPDATE", folder, racer, deletion)
	} else {
		theirs, mine = w.race(t, "user_favorites", "FOR UPDATE", folder, deletion, racer)
	}
	for _, a := range []racedAnswer{theirs, mine} {
		if a.whole.status == http.StatusInternalServerError {
			t.Errorf("%s %s = 500 %s; want no deadlock of the folder's deletion and the favorite's write", a.req.Method, a.req.URL.Path, a.whole.body)
		}
	}
	answeredRace(t, theirs, http.StatusNoContent)
	at, by := stampOf(t, w.pool, "user_favorites", "deleted_at", folder)
	if by != w.ids["alice"] {
		t.Errorf("the folder deleted by %s; want alice", by)
	}
	switch {
	case !ours && write == "in it changed, naming it again":
		w.refusedRaceAsAbsent(t, mine, "alice", racer.body)
	case !ours:
		refusedRace(t, mine, http.StatusUnprocessableEntity, "validation_failed parent_id not_allowed")
	default:
		var f favoriteHTTP
		decodeAnswer(t, answeredRace(t, mine, map[bool]int{true: http.StatusCreated, false: http.StatusOK}[write == "created in it"]), &f)
		if f.ParentID == nil || *f.ParentID != folder || (item != (uuid.UUID{}) && f.ID != item) {
			t.Errorf("written %+v; want it in the folder %s", f, folder)
		}
		item = f.ID
	}
	switch {
	case item == (uuid.UUID{}):
		if left := liveFavoritesOf(t, w.pool, view.ID); left != 0 {
			t.Errorf("%d undeleted favorites of the view once the creation was refused; want none", left)
		}
	case !ours && write == "moved into it":
		if got := favoriteRows(t, w.pool, item); got != before {
			t.Errorf("the favorite refused under its lock is\n%s\nwas\n%s", got, before)
		}
	default:
		favoritesDeletedAt(t, w.pool, at, by, item)
	}
	if left := liveFavoritesOf(t, w.pool, folder); left != 0 {
		t.Errorf("%d undeleted favorites in the deleted folder %s; want none", left, folder)
	}
}
