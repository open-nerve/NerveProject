package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"uuid"
)

// The group of a favorite created again of TestFavoriteWritesInterleaveWithArchives
// (navigation_favorites_interleaving_test.go).

// favoritesRaceAgain races alice's deletion of her favorite of a view of
// acme's own, and her deletion of a folder that holds it, with her creation
// of a favorite of the view again. Each in both orders, each cell on a view
// of its own.
func favoritesRaceAgain(t *testing.T, w issueWorld) {
	n := 0
	for _, deletion := range []string{"its deletion", "its folder's deletion"} {
		for _, ours := range []bool{false, true} {
			n++
			first := "the deletion first"
			if ours {
				first = "the creation first"
			}
			t.Run("by "+deletion+", "+first, func(t *testing.T) { raceAgain(t, w, n, deletion == "its folder's deletion", ours) })
		}
	}
}

// raceAgain is one cell of favoritesRaceAgain's (spec 2.5, 2.8): the other
// transaction holds her favorite of the view FOR UPDATE; its deletion
// waits for it at LockFavorite, a folder's at LockFolderChildren once it
// holds the folder, and the creation at ShareFavoriteOfTarget, its last
// lock before its write. Neither answers 500: the creation does not insert
// while the deletion holds the favorite it is to delete.
//
// The deletion answers 204, her favorite deleted by her, with its folder at
// the folder's moment. Second, the creation finds under its lock that she
// has no favorite of the view and makes one, 201, at the top: the one
// undeleted favorite of the view. First, it answers the one she has as it
// is, 200, the same bytes as its creation answered, in its folder; the
// deletion deletes it then, and no favorite of the view is left undeleted.
func raceAgain(t *testing.T, w issueWorld, n int, inFolder, ours bool) {
	view, _ := w.newView(t, "alice", workspaceViewsPath("acme"), fmt.Sprintf(`{"name":"Again %d"}`, n))
	var folder uuid.UUID
	var parent []uuid.UUID
	if inFolder {
		folder = w.favorID(t, "alice", "acme", folderBody(fmt.Sprintf("Again %d", n)))
		parent = []uuid.UUID{folder}
	}
	had, created := w.favor(t, "alice", "acme", favoriteBody("view", view.ID, parent...), http.StatusCreated)
	deleted, deletion := had.ID, racedWrite{actor: "alice", method: http.MethodDelete, path: favoritePath(had.ID), statement: "LockFavorite"}
	if inFolder {
		deleted, deletion.path, deletion.statement = folder, favoritePath(folder), "LockFolderChildren"
	}
	creation := racedWrite{actor: "alice", method: http.MethodPost, path: favoritesPath("acme"), body: favoriteBody("view", view.ID),
		statement: "ShareFavoriteOfTarget"}
	var mine, theirs racedAnswer
	if ours {
		mine, theirs = w.race(t, "user_favorites", "FOR UPDATE", had.ID, creation, deletion)
	} else {
		theirs, mine = w.race(t, "user_favorites", "FOR UPDATE", had.ID, deletion, creation)
	}
	for _, a := range []racedAnswer{theirs, mine} {
		if a.whole.status == http.StatusInternalServerError {
			t.Errorf("%s %s = 500 %s; want the creation to decide under its lock", a.req.Method, a.req.URL.Path, a.whole.body)
		}
	}
	answeredRace(t, theirs, http.StatusNoContent)
	at, by := stampOf(t, w.pool, "user_favorites", "deleted_at", deleted)
	if by != w.ids["alice"] {
		t.Errorf("deleted by %s; want alice", by)
	}
	favoritesDeletedAt(t, w.pool, at, by, had.ID)
	if ours {
		if body := answeredRace(t, mine, http.StatusOK); body != created {
			t.Errorf("created again first: %s; want the one she has as it is, %s", body, created)
		}
		if left := liveFavoritesOf(t, w.pool, view.ID); left != 0 {
			t.Errorf("%d undeleted favorites of the view once the deletion committed; want none", left)
		}
		return
	}
	var f favoriteHTTP
	decodeAnswer(t, answeredRace(t, mine, http.StatusCreated), &f)
	if f.ID == had.ID || f.ParentID != nil {
		t.Errorf("created again second: %+v; want a new favorite at the top", f)
	}
	if left := liveFavoritesOf(t, w.pool, view.ID); left != 1 {
		t.Errorf("%d undeleted favorites of the view; want the new one alone", left)
	}
	favoritesLive(t, w.pool, f.ID)
}
