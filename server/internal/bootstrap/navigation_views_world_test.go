package bootstrap

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	navigationpg "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	navigationdomain "github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The helpers of the views' composition tests on the real app (M7 P3b spec
// 4): TestProjectViewsThroughTheApp and TestWorkspaceViewsThroughTheApp.

// viewHTTP is a view as the API answers it (View).
type viewHTTP struct {
	ID                uuid.UUID                    `json:"id"`
	WorkspaceID       uuid.UUID                    `json:"workspace_id"`
	ProjectID         *uuid.UUID                   `json:"project_id"`
	Name              string                       `json:"name"`
	Description       string                       `json:"description"`
	LogoProps         json.RawMessage              `json:"logo_props"`
	Filters           map[string][]string          `json:"filters"`
	DisplayFilters    shared.ListDisplayFilters    `json:"display_filters"`
	DisplayProperties shared.ListDisplayProperties `json:"display_properties"`
	OwnedByID         uuid.UUID                    `json:"owned_by_id"`
	CreatedAt         time.Time                    `json:"created_at"`
	UpdatedAt         time.Time                    `json:"updated_at"`
}

// projectViewsPath, workspaceViewsPath and viewPath are the views' routes:
// a project's, a workspace's own, and one view's of either kind.
func projectViewsPath(project uuid.UUID) string {
	return "/api/v0/projects/" + project.String() + "/views"
}
func workspaceViewsPath(slug string) string { return "/api/v0/workspaces/" + slug + "/views" }
func viewPath(id uuid.UUID) string          { return "/api/v0/views/" + id.String() }

// joinAcme registers the account name through the API and makes it acme's
// role through the workspace store (inWorkspaceOf), as prepareIssueWorld
// makes bob, walt and wanda.
func (w issueWorld) joinAcme(t *testing.T, name string, role shared.Role) {
	t.Helper()
	w.tokens[name] = registerAccount(t, w.contract, w.base, name+"@example.com").AccessToken
	w.ids[name] = accountID(t, w.contract, w.base, w.tokens[name])
	inWorkspaceOf(t, w.pool, w.web, w.ids[name], w.ids["alice"], role)
}

// promoteInAcme has alice make the account name acme's admin through the
// API (updateWorkspaceMember).
func (w issueWorld) promoteInAcme(t *testing.T, name string) {
	t.Helper()
	membership := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.acme, w.ids[name])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":20}`, http.StatusOK)
}

// changeProject has alice change the project with body (updateProject),
// e.g. its issue_views_view or its guest_view_all_features.
func (w issueWorld) changeProject(t *testing.T, project uuid.UUID, body string) {
	t.Helper()
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+project.String(), body, http.StatusOK)
}

// newView has actor create a view at path, a project's or a workspace's
// views, with body: 201, the view and the answer's body.
func (w issueWorld) newView(t *testing.T, actor, path, body string) (viewHTTP, string) {
	t.Helper()
	answer := w.send(t, actor, http.MethodPost, path, body, http.StatusCreated)
	var v viewHTTP
	decodeAnswer(t, answer, &v)
	return v, answer
}

// readView is actor's getView of id: 200, the view and the answer's body.
func (w issueWorld) readView(t *testing.T, actor string, id uuid.UUID) (viewHTTP, string) {
	t.Helper()
	answer := w.send(t, actor, http.MethodGet, viewPath(id), "", http.StatusOK)
	var v viewHTTP
	decodeAnswer(t, answer, &v)
	return v, answer
}

// changeView is actor's updateView of id with body: 200, the view and the
// answer's body.
func (w issueWorld) changeView(t *testing.T, actor string, id uuid.UUID, body string) (viewHTTP, string) {
	t.Helper()
	answer := w.send(t, actor, http.MethodPatch, viewPath(id), body, http.StatusOK)
	var v viewHTTP
	decodeAnswer(t, answer, &v)
	return v, answer
}

// listedViews is the ids of actor's list at path, in the answer's order.
func (w issueWorld) listedViews(t *testing.T, actor, path string) []uuid.UUID {
	t.Helper()
	var list struct{ Data []viewHTTP }
	decodeAnswer(t, w.send(t, actor, http.MethodGet, path, "", http.StatusOK), &list)
	ids := make([]uuid.UUID, len(list.Data))
	for n, v := range list.Data {
		ids[n] = v.ID
	}
	return ids
}

// refusedAs checks that actor's method on path with body is refused with
// status and the refusal want (refusalOf: the code, then each field's).
func (w issueWorld) refusedAs(t *testing.T, actor, method, path, body string, status int, want string) {
	t.Helper()
	if got := refusalOf(t, []byte(w.send(t, actor, method, path, body, status)), false); got != want {
		t.Errorf("%s %s as %s: %s; want %s", method, path, actor, got, want)
	}
}

// whole is actor's method on path with body as a wholeAnswer: its status,
// its body and its headers but the date and the request id, the request
// and the answer checked against the contract.
func (w issueWorld) whole(t *testing.T, actor, method, path, body string) wholeAnswer {
	t.Helper()
	var b []byte
	if body != "" {
		b = []byte(body)
	}
	req := newRequest(t, method, w.base+path, w.tokens[actor], b)
	w.contract.CheckRequest(t, req)
	res, raw := send(t, req)
	w.contract.CheckResponse(t, req, res)
	return wholeAnswerOf(res, raw)
}

// answeredAsAbsent checks that actor's method on path with body answers 404
// code, the same answer byte for byte, its status, body and headers, as on
// absent, which names nothing that ever was (P3b spec 2.2, 2.5).
func (w issueWorld) answeredAsAbsent(t *testing.T, actor, method, path, absent, body, code string) {
	t.Helper()
	got, want := w.whole(t, actor, method, path, body), w.whole(t, actor, method, absent, body)
	if want.status != http.StatusNotFound || problemCode(t, []byte(want.body)) != code {
		t.Fatalf("%s %s as %s: %d %s; want 404 %s", method, absent, actor, want.status, want.body, code)
	}
	if got != want {
		t.Errorf("%s %s as %s differs from %s:\n%+v\n%+v", method, path, actor, absent, got, want)
	}
}

// viewRow is a view's row as the API does not show it: who made it, who
// changed it last and when, when it was deleted.
type viewRow struct {
	ownedBy, createdBy, updatedBy uuid.UUID
	createdAt, updatedAt          time.Time
	deletedAt                     *time.Time
}

// viewRowOf reads the row of the view id, deleted or not.
func viewRowOf(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) viewRow {
	t.Helper()
	var r viewRow
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT owned_by_id, created_by_id, updated_by_id, created_at, updated_at, deleted_at
		FROM issue_views WHERE id = $1`, id).Scan(&r.ownedBy, &r.createdBy, &r.updatedBy, &r.createdAt, &r.updatedAt, &r.deletedAt); err != nil {
		t.Fatalf("the view %s: %v", id, err)
	}
	return r
}

// seedView writes a view, named name, of the workspace and, when project
// is not nil, of its project, owned by owner, made by him at at, through
// the navigation store as prepareMatrix seeds its views
// (seedMatrixViews): no description, icon or filter, the shared display
// settings. A view a guest owns is one only a store makes (P3b spec R7: a
// guest creates none); two made at one moment are one only a store makes,
// the app's clock reading a moment per request.
func seedView(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID, project *uuid.UUID, name string, owner uuid.UUID, at time.Time) uuid.UUID {
	t.Helper()
	v := navigationdomain.View{ID: uuid.NewV7(), WorkspaceID: workspace, ProjectID: project, OwnedByID: owner, CreatedAt: at, UpdatedAt: at,
		Content: navigationdomain.Content{Name: name, DisplayFilters: shared.DefaultListDisplayFilters(),
			DisplayProperties: shared.DefaultListDisplayProperties()}}
	if err := navigationpg.New(pool).InsertView(pgtest.Soon(t), v, owner); err != nil {
		t.Fatalf("seeding the view %s: %v", name, err)
	}
	return v.ID
}

// displayJSON is d as a body gives it.
func displayJSON(t *testing.T, d shared.ListDisplayFilters) string {
	t.Helper()
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// viewDeletedBy checks that the view id is soft-deleted by by, deleted and
// last changed at one moment within [from, to], its owner and its maker
// kept as owner (P3b spec 2.4: a view's deletion is written at one moment
// by its deleter, its owner or not).
func viewDeletedBy(t *testing.T, pool *pgxpool.Pool, id, owner, by uuid.UUID, from, to time.Time) {
	t.Helper()
	r := viewRowOf(t, pool, id)
	if r.deletedAt == nil || !r.deletedAt.Equal(r.updatedAt) || r.deletedAt.Before(from) || r.deletedAt.After(to) || r.updatedBy != by ||
		r.ownedBy != owner || r.createdBy != owner {
		t.Errorf("the view %s after its deletion by %s within [%s, %s]: %+v", id, by, from, to, r)
	}
}
