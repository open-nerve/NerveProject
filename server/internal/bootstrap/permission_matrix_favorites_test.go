package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The rows of the permission matrix of one's own favorites (M7 P3c spec
// 2.2, 2.4, 2.7, R6), on the workspace level's columns alone: a row with a
// project table's column would make its operation a write on a project
// (writesOnAProject), and a favorite is a write of the workspace's, whose
// locks TestFavoriteWritesTakeTheirTargetsLocks pins. favorite.read and
// favorite.write let the workspace's admins and members in, not its guests
// (design 3.10). Each column has one favorite of its own, a folder in the
// workspace its cells target (seedMatrixFavorites): its list holds that
// folder alone, the others' left out, and its change and its deletion name
// it by its id; never's and the removed member's in acme and the deleted
// workspace's admin's in gone, deleted with it, are each
// navigation.favorite_not_found, as an id of none is. A creation's target
// in the workspace its cells target: none, a folder's; its member's view of
// its own, which every role of favorite.write reads; or its project
// (projectOf), in acme the public one, which acme's admin and member read
// only once the row's prepare makes them its members on the cell's copy
// (prepareDraftPublisher): without it, it is no target they may favor, one
// 422 (spec 2.5, R7).

var cellFavoriteNotFound = cell{http.StatusNotFound, "navigation.favorite_not_found"}

// The bodies of the rows that create a folder and rename one.
const (
	newFolder     = `{"entity_type":"folder","name":"New"}`
	renamedFolder = `{"name":"Renamed"}`
)

func favoriteMatrixRows() []matrixRow {
	own := func(method, body string) func(caller, seeded) (string, string, string) {
		return func(c caller, s seeded) (string, string, string) {
			return method, "/api/v0/favorites/" + s.favorite(c).String(), body
		}
	}
	ofOwn := func(admin, member, guest cell) map[caller]cell {
		return workspaceCells(admin, member, guest, cellFavoriteNotFound, cellFavoriteNotFound, cellFavoriteNotFound)
	}
	ofTarget := func(kind string, target func(caller, seeded) uuid.UUID) func(caller, seeded) (string, string, string) {
		return func(c caller, s seeded) (string, string, string) {
			body := `{"entity_type":"` + kind + `","entity_id":"` + target(c, s).String() + `"}`
			return toWorkspace(http.MethodPost, "/favorites", body)(c, s)
		}
	}
	view := func(c caller, s seeded) uuid.UUID { return s.workspaceView(workspaceOf(c), aMembersView) }
	project := func(c caller, s seeded) uuid.UUID { return s.project(projectOf(c)) }
	return []matrixRow{
		{op: "listFavorites", request: toWorkspace(http.MethodGet, "/favorites", ""), cells: inWorkspace(cellOK, cellOK, cellForbidden),
			check: listsTheOwnFolder},
		{op: "createFavorite", variant: "a folder", write: true, request: toWorkspace(http.MethodPost, "/favorites", newFolder),
			cells: inWorkspace(cellCreated, cellCreated, cellForbidden), check: answersFavorite(func(caller, seeded) favoriteAnswer {
				return favoriteAnswer{EntityType: "folder", Name: "New"}
			})},
		{op: "createFavorite", variant: "a workspace's view", write: true, request: ofTarget("view", view),
			cells: inWorkspace(cellCreated, cellCreated, cellForbidden), check: answersFavorite(func(c caller, s seeded) favoriteAnswer {
				id := view(c, s)
				return favoriteAnswer{EntityType: "view", EntityID: &id, Name: workspaceOf(c) + ", " + aMembersView}
			})},
		{op: "createFavorite", variant: "a project, as its member", write: true, prepare: prepareDraftPublisher,
			request: ofTarget("project", project), cells: inWorkspace(cellCreated, cellCreated, cellForbidden),
			check: answersFavorite(func(c caller, s seeded) favoriteAnswer {
				id := project(c, s)
				return favoriteAnswer{ProjectID: &id, EntityType: "project", EntityID: &id, Name: "Web"}
			})},
		{op: "createFavorite", variant: "a project, as none of its members", write: true, request: ofTarget("project", project),
			cells: inWorkspace(cellValidationFailed, cellValidationFailed, cellForbidden), refusal: "entity_id not_allowed"},
		{op: "updateFavorite", write: true, request: own(http.MethodPatch, renamedFolder), cells: ofOwn(cellOK, cellOK, cellForbidden),
			check: answersFavorite(func(c caller, s seeded) favoriteAnswer {
				return favoriteAnswer{ID: s.favorite(c), EntityType: "folder", Name: "Renamed"}
			})},
		{op: "deleteFavorite", write: true, request: own(http.MethodDelete, ""), cells: ofOwn(cellNoContent, cellNoContent, cellForbidden)},
	}
}

// favoriteAnswer is a favorite as the rows' checks read an answer.
type favoriteAnswer struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	ProjectID   *uuid.UUID `json:"project_id"`
	EntityType  string     `json:"entity_type"`
	EntityID    *uuid.UUID `json:"entity_id"`
	ParentID    *uuid.UUID `json:"parent_id"`
	Name        string     `json:"name"`
}

// answersFavorite is the check of a row whose answer is one favorite of
// the caller's, in the workspace his cells target, at the top, as want
// gives it for his column: its kind, its target and the target's project,
// nil for none, and its name; and its id, a new one's when want gives
// none, which no seeded favorite has.
func answersFavorite(want func(c caller, s seeded) favoriteAnswer) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var got favoriteAnswer
		decodeAnswer(t, answer, &got)
		w := want(c, s)
		w.WorkspaceID = s.workspace(workspaceOf(c))
		if w.ID == uuid.Nil() {
			if _, known := keyOfRow(s.favorites, got.ID); known || got.ID == uuid.Nil() {
				t.Errorf("%s's new favorite = %s; want an id of its own", c, answer)
			}
			w.ID = got.ID
		}
		if !reflect.DeepEqual(got, w) {
			t.Errorf("%s's favorite in %s = %s; want %+v", c, workspaceOf(c), answer, w)
		}
	}
}

// listsTheOwnFolder: the caller's list holds his own folder alone, not the
// other columns' in his workspace.
func listsTheOwnFolder(t *testing.T, c caller, s seeded, answer string) {
	var list struct {
		Data []favoriteAnswer `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	want := []favoriteAnswer{{ID: s.favorite(c), WorkspaceID: s.workspace(workspaceOf(c)), EntityType: "folder", Name: ownFolder(c)}}
	if !reflect.DeepEqual(list.Data, want) {
		t.Errorf("%s's favorites in %s = %s; want his folder alone, %+v", c, workspaceOf(c), answer, want)
	}
}

// ownFolder is the name of the folder of c's own that seedMatrixFavorites
// writes.
func ownFolder(c caller) string { return string(c) + "'s" }

// matrixFavoriteIDs names, for newSeeded, the id of each workspace
// column's own favorite, by its workspace's slug, "/" and the column, as
// its draft's and its asset's are.
func matrixFavoriteIDs() map[string]uuid.UUID {
	ids := map[string]uuid.UUID{}
	for _, c := range workspaceColumns {
		ids[workspaceOf(c)+"/"+string(c)] = uuid.NewV7()
	}
	return ids
}

// favorite is the id of c's own favorite, his folder in the workspace his
// cells target; one never seeded fails the test at once, as membership's
// does.
func (s seeded) favorite(c caller) uuid.UUID {
	id, ok := s.favorites[workspaceOf(c)+"/"+string(c)]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no favorite of %s in %s is seeded", c, workspaceOf(c))
	}
	return id
}

// seedMatrixFavorites writes each workspace column's own favorite with the
// id newSeeded named, directly, as the deletion guards write theirs
// (seedFavorites): a folder (ownFolder), his, at the top of the workspace
// his cells target, his first, made and last changed by him at at.
// prepareMatrix writes them before it deletes gone, whose deletion deletes
// gone's (spec 2.6).
func seedMatrixFavorites(t *testing.T, pool *pgxpool.Pool, s seeded, at time.Time) {
	t.Helper()
	for _, c := range workspaceColumns {
		tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO user_favorites (id, workspace_id, user_id, entity_type, name, sequence, created_by_id,
				updated_by_id, created_at, updated_at)
			VALUES ($1, $2, $3, 'folder', $4, 65535, $3, $3, $5, $5)`, s.favorite(c), s.workspace(workspaceOf(c)), s.account(c), ownFolder(c), at)
		if err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("seeding %s's folder in %s: %d rows, %v; want 1", c, workspaceOf(c), tag.RowsAffected(), err)
		}
	}
}
