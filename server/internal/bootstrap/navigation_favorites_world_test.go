package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The helpers of the favorites' composition tests on the real app (M7 P3c
// spec 4): TestFavoritesOfEveryKindThroughTheApp and
// TestFavoritesFollowTheirTargets.

// favoriteHTTP is a favorite as the API answers it (Favorite).
type favoriteHTTP struct {
	ID          uuid.UUID       `json:"id"`
	WorkspaceID uuid.UUID       `json:"workspace_id"`
	ProjectID   *uuid.UUID      `json:"project_id"`
	EntityType  string          `json:"entity_type"`
	EntityID    *uuid.UUID      `json:"entity_id"`
	ParentID    *uuid.UUID      `json:"parent_id"`
	Sequence    float64         `json:"sequence"`
	Name        string          `json:"name"`
	LogoProps   json.RawMessage `json:"logo_props"`
	CreatedAt   time.Time       `json:"created_at"`
}

// favoritesPath is a workspace's favorites' route; favoritePath one
// favorite's.
func favoritesPath(slug string) string { return "/api/v0/workspaces/" + slug + "/favorites" }
func favoritePath(id uuid.UUID) string { return "/api/v0/favorites/" + id.String() }

// favoriteBody is the body of a favorite of the target id of kind; in the
// folder parent, when one is given.
func favoriteBody(kind string, id uuid.UUID, parent ...uuid.UUID) string {
	if len(parent) == 0 {
		return fmt.Sprintf(`{"entity_type":%q,"entity_id":%q}`, kind, id)
	}
	return fmt.Sprintf(`{"entity_type":%q,"entity_id":%q,"parent_id":%q}`, kind, id, parent[0])
}

// folderBody is the body of a folder named name.
func folderBody(name string) string { return fmt.Sprintf(`{"entity_type":"folder","name":%q}`, name) }

// favor is actor's createFavorite in the workspace slug with body: want,
// 201 or 200, the favorite and the answer's body.
func (w issueWorld) favor(t *testing.T, actor, slug, body string, want int) (favoriteHTTP, string) {
	t.Helper()
	answer := w.send(t, actor, http.MethodPost, favoritesPath(slug), body, want)
	var f favoriteHTTP
	decodeAnswer(t, answer, &f)
	return f, answer
}

// favorID is the id of actor's new favorite in the workspace slug with
// body (201).
func (w issueWorld) favorID(t *testing.T, actor, slug, body string) uuid.UUID {
	t.Helper()
	f, _ := w.favor(t, actor, slug, body, http.StatusCreated)
	return f.ID
}

// changeFavorite is actor's updateFavorite of id with body: 200, the
// favorite and the answer's body.
func (w issueWorld) changeFavorite(t *testing.T, actor string, id uuid.UUID, body string) (favoriteHTTP, string) {
	t.Helper()
	answer := w.send(t, actor, http.MethodPatch, favoritePath(id), body, http.StatusOK)
	var f favoriteHTTP
	decodeAnswer(t, answer, &f)
	return f, answer
}

// listedFavorites is actor's list in the workspace slug, in the answer's
// order.
func (w issueWorld) listedFavorites(t *testing.T, actor, slug string) []favoriteHTTP {
	t.Helper()
	var list struct{ Data []favoriteHTTP }
	decodeAnswer(t, w.send(t, actor, http.MethodGet, favoritesPath(slug), "", http.StatusOK), &list)
	return list.Data
}

// listedFavoriteIDs is the ids of actor's list in the workspace slug, in
// the answer's order.
func (w issueWorld) listedFavoriteIDs(t *testing.T, actor, slug string) []uuid.UUID {
	t.Helper()
	return favoriteIDsOf(w.listedFavorites(t, actor, slug))
}

// favoriteIDsOf is the ids of fs, in their order.
func favoriteIDsOf(fs []favoriteHTTP) []uuid.UUID {
	ids := make([]uuid.UUID, len(fs))
	for n, f := range fs {
		ids[n] = f.ID
	}
	return ids
}

// containsAll tells whether ids has each of want; containsAny whether it
// has any.
func containsAll(ids []uuid.UUID, want ...uuid.UUID) bool {
	for _, id := range want {
		if !slices.Contains(ids, id) {
			return false
		}
	}
	return true
}

func containsAny(ids []uuid.UUID, want ...uuid.UUID) bool {
	for _, id := range want {
		if slices.Contains(ids, id) {
			return true
		}
	}
	return false
}

// favoriteAnsweredAsAbsent checks that actor's change and deletion of his
// favorite id answer as an id no favorite has, byte for byte (spec 2.7).
func (w issueWorld) favoriteAnsweredAsAbsent(t *testing.T, actor string, id uuid.UUID) {
	t.Helper()
	absent := favoritePath(uuid.NewV7())
	for _, op := range []struct{ method, body string }{{http.MethodPatch, `{"sequence":1}`}, {http.MethodDelete, ""}} {
		w.answeredAsAbsent(t, actor, op.method, favoritePath(id), absent, op.body, "navigation.favorite_not_found")
	}
}

// refusedAlike checks that actor's createFavorite in the workspace slug
// with each of bodies answers want, byte for byte, its status, body and
// headers.
func (w issueWorld) refusedAlike(t *testing.T, actor, slug string, want wholeAnswer, bodies ...string) {
	t.Helper()
	for _, body := range bodies {
		if got := w.whole(t, actor, http.MethodPost, favoritesPath(slug), body); got != want {
			t.Errorf("POST %s as %s with %s:\n%+v\nwant\n%+v", favoritesPath(slug), actor, body, got, want)
		}
	}
}

// noTarget is actor's createFavorite in the workspace slug of a project no
// project is: the one 422 of a target he may not favor (spec 2.5, R7).
func (w issueWorld) noTarget(t *testing.T, actor, slug string) wholeAnswer {
	t.Helper()
	a := w.whole(t, actor, http.MethodPost, favoritesPath(slug), favoriteBody("project", uuid.NewV7()))
	if a.status != http.StatusUnprocessableEntity || refusalOf(t, []byte(a.body), false) != "validation_failed entity_id not_allowed" {
		t.Fatalf("a favorite of no project as %s: %+v", actor, a)
	}
	return a
}

// favoriteRow is a favorite's row as the API does not show it: whose it
// is, who made it and changed it last and when, when it was deleted.
type favoriteRow struct {
	userID, createdBy, updatedBy uuid.UUID
	createdAt, updatedAt         time.Time
	deletedAt                    *time.Time
}

func (r favoriteRow) String() string {
	deleted := "undeleted"
	if r.deletedAt != nil {
		deleted = "deleted at " + r.deletedAt.UTC().Format(time.RFC3339Nano)
	}
	return fmt.Sprintf("{%s's, made by %s at %s, last changed by %s at %s, %s}", r.userID, r.createdBy, r.createdAt.UTC().Format(time.RFC3339Nano),
		r.updatedBy, r.updatedAt.UTC().Format(time.RFC3339Nano), deleted)
}

// favoriteRowOf reads the row of the favorite id, deleted or not.
func favoriteRowOf(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) favoriteRow {
	t.Helper()
	var r favoriteRow
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT user_id, created_by_id, updated_by_id, created_at, updated_at, deleted_at
		FROM user_favorites WHERE id = $1`, id).Scan(&r.userID, &r.createdBy, &r.updatedBy, &r.createdAt, &r.updatedAt, &r.deletedAt); err != nil {
		t.Fatalf("the favorite %s: %v", id, err)
	}
	return r
}

// favoritesLive checks that each of the favorites ids is undeleted.
func favoritesLive(t *testing.T, pool *pgxpool.Pool, ids ...uuid.UUID) {
	t.Helper()
	for _, id := range ids {
		if r := favoriteRowOf(t, pool, id); r.deletedAt != nil {
			t.Errorf("the favorite %s is deleted at %s; want it undeleted", id, r.deletedAt)
		}
	}
}

// favoritesDeletedAt checks that each of the favorites ids is deleted and
// last changed at at, by by, its owner and maker kept.
func favoritesDeletedAt(t *testing.T, pool *pgxpool.Pool, at time.Time, by uuid.UUID, ids ...uuid.UUID) {
	t.Helper()
	for _, id := range ids {
		r := favoriteRowOf(t, pool, id)
		if r.deletedAt == nil || !r.deletedAt.Equal(at) || !r.updatedAt.Equal(at) || r.updatedBy != by || r.createdBy != r.userID {
			t.Errorf("the favorite %s: %+v; want deleted and last changed at %s by %s", id, r, at, by)
		}
	}
}

// favoriteRows is the whole rows of the favorites ids, in id order, as
// text: what a write that leaves them untouched keeps byte for byte.
func favoriteRows(t *testing.T, pool *pgxpool.Pool, ids ...uuid.UUID) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(string_agg(f::text, E'\\n' ORDER BY f.id), '') FROM user_favorites AS f WHERE f.id = ANY($1)",
		ids).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// stampOf is the moment of the row id of table in its column, and who
// changed it last: a target's deletion or archive, as its writer wrote it.
func stampOf(t *testing.T, pool *pgxpool.Pool, table, column string, id uuid.UUID) (time.Time, uuid.UUID) {
	t.Helper()
	var at time.Time
	var by uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT "+column+", updated_by_id FROM "+table+" WHERE id = $1 AND "+column+" IS NOT NULL", id).
		Scan(&at, &by); err != nil {
		t.Fatalf("the %s of %s %s: %v", column, table, id, err)
	}
	return at, by
}

// leftFavorite writes, directly, owner's undeleted favorite of the target
// id of kind, whose project is project, made by him now: the row a
// creation that read its target before the target's archive and wrote after
// it would leave behind (M7 design 3.8). The app's writes decide under the
// target's project's lock, which its archive takes, and leave none; so only
// SQL makes it, to show that the list's own predicate keeps it out and
// what an archive's cascade does to such a row.
func leftFavorite(t *testing.T, pool *pgxpool.Pool, owner uuid.UUID, kind string, id, project uuid.UUID) uuid.UUID {
	t.Helper()
	favorite := uuid.NewV7()
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO user_favorites (id, workspace_id, project_id, user_id, entity_type, entity_identifier, sequence,
			created_by_id, updated_by_id, created_at, updated_at)
		SELECT $1, p.workspace_id, p.id, $2, $3, $4, 1, $2, $2, now(), now() FROM projects AS p WHERE p.id = $5`,
		favorite, owner, kind, id, project); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("leaving %s's favorite of the %s %s: %d rows, %v; want 1", owner, kind, id, tag.RowsAffected(), err)
	}
	return favorite
}
