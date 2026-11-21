package httpadapter_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	pinnedID     = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000f7")
	starID       = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000f8")
	favoritePath = "/api/v0/favorites/0199a2b4-0000-7000-8000-0000000000f8"
)

// favoritesPath is acme's favorites, the caller's own.
const favoritesPath = "/api/v0/workspaces/acme/favorites"

// favoriteCall is what a favorites' use case was asked: the caller, the
// operation, and what it names and gives.
type favoriteCall struct {
	Caller uuid.UUID
	Op     string
	ID     uuid.UUID
	Slug   string
	New    domain.NewFavorite
	Patch  domain.FavoritePatch
}

// favoriteFakes are the favorites' use cases, each a fake that records its
// call and answers list, favorite and, a creation's, created, or err.
type favoriteFakes struct {
	calls    []favoriteCall
	list     []app.NamedFavorite
	favorite app.NamedFavorite
	created  bool
	err      error
}

// record keeps c, asked by the caller of ctx (the zero id when none).
func (f *favoriteFakes) record(ctx context.Context, c favoriteCall) {
	if actor, err := shared.RequireActor(ctx); err == nil {
		c.Caller = actor.UserID
	}
	f.calls = append(f.calls, c)
}

type (
	fakeListFavorites  struct{ *favoriteFakes }
	fakeCreateFavorite struct{ *favoriteFakes }
	fakeUpdateFavorite struct{ *favoriteFakes }
	fakeDeleteFavorite struct{ *favoriteFakes }
)

func (f fakeListFavorites) Execute(ctx context.Context, slug string) ([]app.NamedFavorite, error) {
	f.record(ctx, favoriteCall{Op: "listFavorites", Slug: slug})
	return f.list, f.err
}

func (f fakeCreateFavorite) Execute(ctx context.Context, slug string, n domain.NewFavorite) (app.NamedFavorite, bool, error) {
	f.record(ctx, favoriteCall{Op: "createFavorite", Slug: slug, New: n})
	return f.favorite, f.created, f.err
}

func (f fakeUpdateFavorite) Execute(ctx context.Context, id uuid.UUID, p domain.FavoritePatch) (app.NamedFavorite, error) {
	f.record(ctx, favoriteCall{Op: "updateFavorite", ID: id, Patch: p})
	return f.favorite, f.err
}

func (f fakeDeleteFavorite) Execute(ctx context.Context, id uuid.UUID) error {
	f.record(ctx, favoriteCall{Op: "deleteFavorite", ID: id})
	return f.err
}

// useCases are f as the module's favorites' use cases.
func (f *favoriteFakes) useCases() httpadapter.UseCases {
	return httpadapter.UseCases{ListFavorites: fakeListFavorites{f}, CreateFavorite: fakeCreateFavorite{f}, UpdateFavorite: fakeUpdateFavorite{f},
		DeleteFavorite: fakeDeleteFavorite{f}}
}

// viewFavorite is alice's favorite of Web's view Board, in her folder
// Pinned, with each value it may have: its project, its target, its folder,
// a sequence of a fraction, and the view's name and icon now; changed after
// its creation, which the API does not show.
func viewFavorite() app.NamedFavorite {
	web, board, pinned := webID, boardID, pinnedID
	return app.NamedFavorite{
		Favorite: domain.Favorite{ID: starID, WorkspaceID: acmeID, ProjectID: &web, UserID: aliceID, Kind: domain.FavoriteView, EntityID: &board,
			ParentID: &pinned, Sequence: 75535.5, CreatedAt: created, UpdatedAt: changed},
		Name: "Board", LogoProps: shared.LogoProps{InUse: new(shared.LogoInUseIcon), Icon: &shared.LogoIcon{Name: new("bug"), Color: new("#f00")}}}
}

// folderFavorite is alice's folder Pinned, with none of the values it may
// lack: no project, no target, no folder, no icon.
func folderFavorite() app.NamedFavorite {
	return app.NamedFavorite{Favorite: domain.Favorite{ID: pinnedID, WorkspaceID: acmeID, UserID: aliceID, Kind: domain.FavoriteFolder,
		Name: "Pinned", Sequence: 65535, CreatedAt: created, UpdatedAt: created}, Name: "Pinned"}
}

const (
	viewFavoriteJSON = `{"id":"0199a2b4-0000-7000-8000-0000000000f8","workspace_id":"0199a2b4-0000-7000-8000-00000000000a",` +
		`"project_id":"0199a2b4-0000-7000-8000-0000000000a1","entity_type":"view","entity_id":"0199a2b4-0000-7000-8000-0000000000e1",` +
		`"parent_id":"0199a2b4-0000-7000-8000-0000000000f7","sequence":75535.5,"name":"Board",` +
		`"logo_props":{"in_use":"icon","icon":{"name":"bug","color":"#f00"}},"created_at":"2026-11-20T10:00:00.123456Z"}`
	folderJSON = `{"id":"0199a2b4-0000-7000-8000-0000000000f7","workspace_id":"0199a2b4-0000-7000-8000-00000000000a",` +
		`"project_id":null,"entity_type":"folder","entity_id":null,"parent_id":null,"sequence":65535,"name":"Pinned",` +
		`"logo_props":{},"created_at":"2026-11-20T10:00:00.123456Z"}`
)

// GET of one's favorites goes to the use case for the caller and the path's
// workspace; the answer is 200 with the favorites in their order, each with
// every key, null for a value it lacks, and [] for none.
func TestListingTheFavorites(t *testing.T) {
	for _, tt := range []struct {
		list []app.NamedFavorite
		want string
	}{
		{[]app.NamedFavorite{viewFavorite(), folderFavorite()}, `{"data":[` + viewFavoriteJSON + `,` + folderJSON + `]}`},
		{[]app.NamedFavorite{}, `{"data":[]}`},
		{nil, `{"data":[]}`},
	} {
		f := &favoriteFakes{list: tt.list}
		res, body := do(t, newServer(t, f.useCases()), request(http.MethodGet, favoritesPath, ""))
		if res.StatusCode != http.StatusOK || !sameJSON(t, body, tt.want) {
			t.Errorf("GET of %d = %d %s, want 200 %s", len(tt.list), res.StatusCode, body, tt.want)
		}
		if want := []favoriteCall{{Caller: aliceID, Op: "listFavorites", Slug: "acme"}}; !reflect.DeepEqual(f.calls, want) {
			t.Errorf("GET of %d: the use cases got %+v, want %+v alone", len(tt.list), f.calls, want)
		}
	}
}

// POST goes to the use case for the caller and the path's workspace with
// the body's values as given, unchecked: a kind none of the five, a folder's
// name with its white space, and fields a kind does not take among them;
// each field left out is nil. The answer is the favorite the use case
// gives: 201 when it created it, 200 when the caller had it already, the
// same body either way.
func TestCreatingAFavorite(t *testing.T) {
	web, board, pinned := webID, boardID, pinnedID
	for _, tt := range []struct {
		body string
		want domain.NewFavorite
	}{
		{`{"entity_type":"project","entity_id":"0199a2b4-0000-7000-8000-0000000000a1"}`, domain.NewFavorite{Kind: domain.FavoriteProject, EntityID: &web}},
		{`{"entity_type":"view","entity_id":"0199a2b4-0000-7000-8000-0000000000e1","parent_id":"0199a2b4-0000-7000-8000-0000000000f7"}`,
			domain.NewFavorite{Kind: domain.FavoriteView, EntityID: &board, ParentID: &pinned}},
		{`{"entity_type":"folder","name":"  Pinned "}`, domain.NewFavorite{Kind: domain.FavoriteFolder, Name: new("  Pinned ")}},
		{`{"entity_type":"folder","entity_id":"0199a2b4-0000-7000-8000-0000000000e1","name":"","parent_id":"0199a2b4-0000-7000-8000-0000000000f7"}`,
			domain.NewFavorite{Kind: domain.FavoriteFolder, EntityID: &board, Name: new(""), ParentID: &pinned}},
		{`{"entity_type":"issue"}`, domain.NewFavorite{Kind: "issue"}},
	} {
		for _, created := range []bool{true, false} {
			f := &favoriteFakes{favorite: viewFavorite(), created: created}
			res, body := do(t, newServer(t, f.useCases()), request(http.MethodPost, favoritesPath, tt.body))
			status := map[bool]int{true: http.StatusCreated, false: http.StatusOK}[created]
			if res.StatusCode != status || !sameJSON(t, body, viewFavoriteJSON) {
				t.Errorf("POST %s, created %v = %d %s, want %d %s", tt.body, created, res.StatusCode, body, status, viewFavoriteJSON)
			}
			if want := []favoriteCall{{Caller: aliceID, Op: "createFavorite", Slug: "acme", New: tt.want}}; !reflect.DeepEqual(f.calls, want) {
				t.Errorf("POST %s: the use cases got %+v, want %+v alone", tt.body, f.calls, want)
			}
		}
	}
}

// PATCH goes to the use case for the caller and the path's favorite with
// each field the body gives, unchecked, and each it leaves out nil: {} gives
// nothing; parent_id null is given, out of its folder, apart from a
// parent_id left out. The answer is 200 with the favorite.
func TestUpdatingAFavorite(t *testing.T) {
	pinned := pinnedID
	for _, tt := range []struct {
		body string
		want domain.FavoritePatch
	}{
		{`{}`, domain.FavoritePatch{}},
		{`{"parent_id":null}`, domain.FavoritePatch{SetParentID: true}},
		{`{"parent_id":"0199a2b4-0000-7000-8000-0000000000f7"}`, domain.FavoritePatch{SetParentID: true, ParentID: &pinned}},
		{`{"sequence":-12.25}`, domain.FavoritePatch{Sequence: new(-12.25)}},
		{`{"sequence":0}`, domain.FavoritePatch{Sequence: new(0.0)}},
		{`{"name":" Work "}`, domain.FavoritePatch{Name: new(" Work ")}},
		{`{"parent_id":"0199a2b4-0000-7000-8000-0000000000f7","sequence":85535,"name":""}`,
			domain.FavoritePatch{SetParentID: true, ParentID: &pinned, Sequence: new(85535.0), Name: new("")}},
	} {
		f := &favoriteFakes{favorite: folderFavorite()}
		res, body := do(t, newServer(t, f.useCases()), request(http.MethodPatch, favoritePath, tt.body))
		if res.StatusCode != http.StatusOK || !sameJSON(t, body, folderJSON) {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, folderJSON)
		}
		if want := []favoriteCall{{Caller: aliceID, Op: "updateFavorite", ID: starID, Patch: tt.want}}; !reflect.DeepEqual(f.calls, want) {
			t.Errorf("PATCH %s: the use cases got %+v, want %+v alone", tt.body, f.calls, want)
		}
	}
}

// DELETE goes to the use case for the caller and the path's favorite; the
// answer is 204 without a body.
func TestDeletingAFavorite(t *testing.T) {
	f := &favoriteFakes{}
	res, body := do(t, newServer(t, f.useCases()), request(http.MethodDelete, favoritePath, ""))
	if res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []favoriteCall{{Caller: aliceID, Op: "deleteFavorite", ID: starID}}; !reflect.DeepEqual(f.calls, want) {
		t.Errorf("the use cases got %+v, want %+v alone", f.calls, want)
	}
}
