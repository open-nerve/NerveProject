package bootstrap

import (
	"bytes"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"
)

// keepsFoldersOneLevel is the folders' cell (spec 2.4, 2.5, D41, D42): a
// folder's name is kept without its white space; another of bob's folders
// in acme named so in any case is refused 409, made or renamed, his own in
// another case is not. A folder holds no folder: one made in a folder and
// one moved into another are refused 422. A parent he may not name, alice's
// folder, his favorite of Web, which is none, his deleted folder, his
// folder in Elsewhere and an id of none, are each the one 422, parent_id
// not_allowed, made or moved. His favorite of Web moves into Work and out
// again, and takes another sequence; a change of nothing, {} or what it
// has, the name's white space dropped, answers the same bytes as before and
// writes nothing.
func keepsFoldersOneLevel(t *testing.T, w favoritesWorld, bobs map[string]favoriteHTTP) {
	work, _ := w.favor(t, "bob", "acme", folderBody("  Work  "), http.StatusCreated)
	if work.Name != "Work" || work.EntityID != nil || work.ProjectID != nil {
		t.Fatalf("bob's folder Work: %+v", work)
	}
	w.bobsFolders["Work"] = work.ID
	for _, name := range []string{"work", " WORK "} {
		w.refusedAs(t, "bob", http.MethodPost, favoritesPath("acme"), folderBody(name), http.StatusConflict, "navigation.favorite_folder_name_taken")
	}
	home := w.favorID(t, "bob", "acme", folderBody("Home"))
	w.refusedAs(t, "bob", http.MethodPatch, favoritePath(home), `{"name":"wORK"}`, http.StatusConflict, "navigation.favorite_folder_name_taken")
	if f, _ := w.changeFavorite(t, "bob", work.ID, `{"name":"WORK"}`); f.Name != "WORK" {
		t.Fatalf("renamed in another case: %+v", f)
	}
	inFolder := fmt.Sprintf(`{"entity_type":"folder","name":"Inner","parent_id":%q}`, work.ID)
	w.refusedAs(t, "bob", http.MethodPost, favoritesPath("acme"), inFolder, http.StatusUnprocessableEntity, "validation_failed parent_id not_allowed")
	w.refusedAs(t, "bob", http.MethodPatch, favoritePath(home), parentBody(work.ID), http.StatusUnprocessableEntity, "validation_failed parent_id not_allowed")

	alices := w.favorID(t, "alice", "acme", folderBody("Alice's"))
	gone := w.favorID(t, "bob", "acme", folderBody("Gone"))
	w.send(t, "bob", http.MethodDelete, favoritePath(gone), "", http.StatusNoContent)
	away := w.favorID(t, "bob", "elsewhere", folderBody("Away"))
	w.bobsElsewhere["Away"] = away
	next := w.newCycle(t, "alice", `{"name":"Next"}`)
	web := bobs["project"].ID
	badParents := []uuid.UUID{alices, web, gone, away, uuid.NewV7()}
	first := w.whole(t, "bob", http.MethodPost, favoritesPath("acme"), favoriteBody("cycle", next, badParents[0]))
	if first.status != http.StatusUnprocessableEntity || refusalOf(t, []byte(first.body), false) != "validation_failed parent_id not_allowed" {
		t.Fatalf("a favorite in alice's folder: %+v", first)
	}
	for _, parent := range badParents[1:] {
		w.refusedAlike(t, "bob", "acme", first, favoriteBody("cycle", next, parent))
	}
	for _, parent := range badParents {
		w.refusedAs(t, "bob", http.MethodPatch, favoritePath(web), parentBody(parent), http.StatusUnprocessableEntity, "validation_failed parent_id not_allowed")
	}

	if in, _ := w.changeFavorite(t, "bob", web, parentBody(work.ID)); in.ParentID == nil || *in.ParentID != work.ID {
		t.Fatalf("moved into Work: %+v", in)
	}
	if listed := w.listedFavorites(t, "bob", "acme"); !slices.ContainsFunc(listed, func(f favoriteHTTP) bool {
		return f.ID == web && f.ParentID != nil && *f.ParentID == work.ID
	}) {
		t.Fatalf("bob lists %+v; want his favorite of Web in Work", listed)
	}
	if out, _ := w.changeFavorite(t, "bob", web, `{"parent_id":null}`); out.ParentID != nil {
		t.Fatalf("moved out: %+v", out)
	}
	was := favoriteRowOf(t, w.pool, web)
	f, before := w.changeFavorite(t, "bob", web, `{"sequence":42.5}`)
	if r := favoriteRowOf(t, w.pool, web); f.Sequence != 42.5 || !r.updatedAt.After(was.updatedAt) || r.updatedBy != w.ids["bob"] {
		t.Fatalf("a new sequence: %+v, its row %+v; was %+v", f, r, was)
	}
	_, folder := w.changeFavorite(t, "bob", work.ID, `{}`)
	rows := favoriteRows(t, w.pool, web, work.ID)
	for _, c := range []struct {
		id         uuid.UUID
		body, want string
	}{{web, `{}`, before}, {web, `{"sequence":42.5}`, before}, {web, `{"parent_id":null,"sequence":42.5}`, before},
		{work.ID, `{"name":"  WORK "}`, folder}, {work.ID, `{"parent_id":null}`, folder}} {
		if _, after := w.changeFavorite(t, "bob", c.id, c.body); after != c.want {
			t.Errorf("PATCH %s with %s: %s; want %s", c.id, c.body, after, c.want)
		}
	}
	if got := favoriteRows(t, w.pool, web, work.ID); got != rows {
		t.Errorf("a change of nothing wrote:\n%s\nwas\n%s", got, rows)
	}
}

// parentBody is the body of a change that moves a favorite into the
// folder parent.
func parentBody(parent uuid.UUID) string { return fmt.Sprintf(`{"parent_id":%q}`, parent) }

// listsTiesByID is the list order's cell (spec 2.4, 2.5): by sequence,
// then by id, both descending. Seeded so: bob's folders Low, then High, in
// Elsewhere, Low's id the smaller, each changed in that order to one
// sequence, 7, below his other favorites there: the table holds their new
// versions in that order, which its scan and its index scan read alike, and
// the sort keeps equal rows so for so few. Without the tie-break by id, Low
// would come before High.
func listsTiesByID(t *testing.T, w favoritesWorld) {
	low := w.favorID(t, "bob", "elsewhere", folderBody("Low"))
	high := w.favorID(t, "bob", "elsewhere", folderBody("High"))
	if bytes.Compare(low[:], high[:]) >= 0 {
		t.Fatalf("made %s before %s", low, high)
	}
	for _, id := range []uuid.UUID{low, high} {
		w.changeFavorite(t, "bob", id, `{"sequence":7}`)
	}
	want := []uuid.UUID{w.bobsElsewhere["Away"], w.bobsElsewhere["Side"], high, low}
	if got := w.listedFavoriteIDs(t, "bob", "elsewhere"); !slices.Equal(got, want) {
		t.Fatalf("bob lists %v in Elsewhere; want %v", got, want)
	}
}

// favoritesNotFoundAlike is /favorites/{favorite_id}'s 404 (spec 2.7):
// alice's folder, bob's deleted folder, and his favorites of Old and of
// Shelf, archived (left, by listsNoArchivedTarget), whose targets he no
// longer sees, each answer his change and his deletion as an id no
// favorite has, byte for byte.
func favoritesNotFoundAlike(t *testing.T, w favoritesWorld, left []uuid.UUID) {
	alices := w.favorID(t, "alice", "acme", folderBody("Alice's own"))
	deleted := w.favorID(t, "bob", "acme", folderBody("Deleted"))
	w.send(t, "bob", http.MethodDelete, favoritePath(deleted), "", http.StatusNoContent)
	for _, id := range append([]uuid.UUID{alices, deleted}, left...) {
		w.favoriteAnsweredAsAbsent(t, "bob", id)
	}
	favoritesLive(t, w.pool, append([]uuid.UUID{alices}, left...)...)
}

// deletesAFolderWithItsFavorites is a folder's deletion (spec 2.5, R10):
// bob's folder Box, into which he moves his favorites of Web and of acme's
// view, is deleted with them, each deleted and last changed by him at one
// moment; his other favorites are kept as they were, and he lists none of
// the three.
func deletesAFolderWithItsFavorites(t *testing.T, w favoritesWorld, bobs map[string]favoriteHTTP) {
	box := w.favorID(t, "bob", "acme", folderBody("Box"))
	held := []uuid.UUID{bobs["project"].ID, bobs["workspace view"].ID}
	for _, id := range held {
		w.changeFavorite(t, "bob", id, parentBody(box))
	}
	kept := []uuid.UUID{bobs["module"].ID, bobs["cycle"].ID, w.bobsFolders["Daily"], w.bobsFolders["Work"]}
	rows := favoriteRows(t, w.pool, kept...)
	from := time.Now().UTC().Truncate(time.Microsecond)
	w.send(t, "bob", http.MethodDelete, favoritePath(box), "", http.StatusNoContent)
	r := favoriteRowOf(t, w.pool, box)
	if r.deletedAt == nil || r.deletedAt.Before(from) || r.deletedAt.After(time.Now()) {
		t.Fatalf("Box: %+v; want deleted within [%s, now]", r, from)
	}
	favoritesDeletedAt(t, w.pool, *r.deletedAt, w.ids["bob"], append([]uuid.UUID{box}, held...)...)
	if got := favoriteRows(t, w.pool, kept...); got != rows {
		t.Errorf("Box's deletion changed others:\n%s\nwas\n%s", got, rows)
	}
	listed := w.listedFavoriteIDs(t, "bob", "acme")
	for _, id := range append([]uuid.UUID{box}, held...) {
		if slices.Contains(listed, id) {
			t.Errorf("bob lists %s, deleted with Box", id)
		}
	}
}
