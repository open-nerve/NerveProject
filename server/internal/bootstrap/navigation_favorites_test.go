package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The icons of Web, of alice's view of it and of bob's view of acme's own.
const (
	webLogo    = `{"in_use":"emoji","emoji":{"value":"128640"}}`
	alicesLogo = `{"in_use":"icon","icon":{"name":"star","color":"#ff0000"}}`
	acmesLogo  = `{"in_use":"emoji","emoji":{"value":"9889"}}`
)

// favoritesWorld is TestFavoritesOfEveryKindThroughTheApp's world:
// issueWorld's acme and Web, in an app whose deletion jobs outlast the
// test's row locks (barriersOutlasted). Web has an icon, its cycles,
// modules and views turned on, its cycle Sprint and its module Core, Old,
// a cycle that has ended, archived, alice's view of it with an icon and
// bob's; acme has bob's view of its own, with an icon. Vault, acme's
// private project, is alice's alone; Shelf, a project of acme's whose
// member bob is, is archived. Elsewhere is bob's own workspace, with his
// project Side and his view Away of its own. Besides issueWorld's accounts: adam, acme's admin who is
// Web's guest (adminGuest's case), and gus, acme's member. Each is made
// through the API, by alice but for bob's.
type favoritesWorld struct {
	issueWorld
	sprint, core, old          uuid.UUID
	alicesView, bobsView       uuid.UUID
	acmesView                  uuid.UUID
	vault, shelf, side, away   uuid.UUID
	bobsFolders, bobsElsewhere map[string]uuid.UUID // bob's folders by name, in acme and in Elsewhere, as the cells make them
}

func newFavoritesWorld(t *testing.T) favoritesWorld {
	t.Helper()
	w := favoritesWorld{issueWorld: newIssueWorldWith(t, barriersOutlasted), bobsFolders: map[string]uuid.UUID{}, bobsElsewhere: map[string]uuid.UUID{}}
	w.joinAcme(t, "adam", shared.RoleMember)
	w.makeMember(t, "adam", w.web, shared.RoleGuest)
	w.promoteInAcme(t, "adam")
	w.joinAcme(t, "gus", shared.RoleMember)
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	w.changeProject(t, w.web, `{"issue_views_view":true,"logo_props":`+webLogo+`}`)
	w.sprint = w.newCycle(t, "alice", `{"name":"Sprint"}`)
	w.core = w.newModule(t, `{"name":"Core"}`)
	now := time.Now()
	w.old = w.newCycle(t, "alice", datesBody("Old", shanghaiDay(t, now, -30), shanghaiDay(t, now, -20)))
	w.send(t, "alice", http.MethodPost, "/api/v0/cycles/"+w.old.String()+"/archive", "", http.StatusOK)
	alices, _ := w.newView(t, "alice", projectViewsPath(w.web), `{"name":"Alice's","logo_props":`+alicesLogo+`}`)
	bobs, _ := w.newView(t, "bob", projectViewsPath(w.web), `{"name":"Bob's"}`)
	acmes, _ := w.newView(t, "bob", workspaceViewsPath("acme"), `{"name":"Acme's","logo_props":`+acmesLogo+`}`)
	w.alicesView, w.bobsView, w.acmesView = alices.ID, bobs.ID, acmes.ID
	w.vault = createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Vault","identifier":"VLT","network":0}`)
	w.shelf = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Shelf", "SHF")
	w.makeMember(t, "bob", w.shelf, shared.RoleMember)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.shelf.String()+"/archive", "", http.StatusOK)
	w.send(t, "bob", http.MethodPost, "/api/v0/workspaces", `{"name":"Elsewhere","slug":"elsewhere"}`, http.StatusCreated)
	w.side = createdProject(t, w.contract, w.base, w.tokens["bob"], "elsewhere", "Side", "SID")
	away, _ := w.newView(t, "bob", workspaceViewsPath("elsewhere"), `{"name":"Away"}`)
	w.away = away.ID
	return w
}

// One's favorites on the real app (H15, H19's project-level cells, M7 P3c
// spec 2.2–2.5, 2.7, 4, R4, R5, R7–R10, R14), each cell a subtest in
// order on one world (newFavoritesWorld): bob's favorite of each kind,
// made in the first, is the one the later cells move, change, hide and
// delete. Seeded so: two of bob's folders in Elsewhere at one sequence
// (listsTiesByID); walt, Web's guest who sees only his own, and bob's and
// alice's views of Web (favorsAsTheGuestRuleSays); bob's undeleted
// favorites of Old and Shelf, archived, left behind by SQL
// (listsNoArchivedTarget); the start-time runs waited for before the
// deletion window holds its locks (favoritesDeletionWindow).
func TestFavoritesOfEveryKindThroughTheApp(t *testing.T) {
	w := newFavoritesWorld(t)
	var bobs map[string]favoriteHTTP
	var left []uuid.UUID
	t.Run("every kind", func(t *testing.T) { bobs = favorsEveryKind(t, w) })
	t.Run("again", func(t *testing.T) { favorsAgain(t, w, bobs) })
	t.Run("no target he may favor", func(t *testing.T) { refusesWhatHeMayNotFavor(t, w) })
	t.Run("workspace guest", func(t *testing.T) { refusesAWorkspaceGuest(t, w) })
	t.Run("folders", func(t *testing.T) { keepsFoldersOneLevel(t, w, bobs) })
	t.Run("list order", func(t *testing.T) { listsTiesByID(t, w) })
	t.Run("one 422", func(t *testing.T) {
		// A target and a parent he may not have, both in one 422.
		w.refusedAs(t, "bob", http.MethodPost, favoritesPath("acme"), favoriteBody("project", w.vault, uuid.NewV7()), http.StatusUnprocessableEntity,
			"validation_failed entity_id not_allowed; parent_id not_allowed")
	})
	t.Run("read-time filtering", func(t *testing.T) { filtersOnRead(t, w, bobs) })
	t.Run("own only", func(t *testing.T) { favorsAsTheGuestRuleSays(t, w) })
	t.Run("toggles off", func(t *testing.T) { ignoresTheToggles(t, w, bobs) })
	t.Run("archived left behind", func(t *testing.T) { left = listsNoArchivedTarget(t, w) })
	t.Run("favorite not found", func(t *testing.T) { favoritesNotFoundAlike(t, w, left) })
	t.Run("folder deletion", func(t *testing.T) { deletesAFolderWithItsFavorites(t, w, bobs) })
	t.Run("deletion window", func(t *testing.T) { favoritesDeletionWindow(t, w) })
}

// favorsEveryKind is the five kinds' cell (spec 2.3, 2.4, R4, R8): bob
// favors his folder Daily, Web, Sprint, Core, alice's view of Web and his
// view of acme's own. Each is new, 201, his, at the top, made and last
// changed by him at once, one step of 10000 past his last from 65535 (spec
// 2.5), of the project the server finds from its target, none for the view
// of acme's own and the folder, named and shown with its target's name and
// icon now, a cycle's and a module's none, a folder with its own name
// without its white space. His list answers them alike, the last made
// first. It returns them by kind, "workspace view" for acme's own view's.
func favorsEveryKind(t *testing.T, w favoritesWorld) map[string]favoriteHTTP {
	cases := []struct {
		key, kind string
		target    *uuid.UUID
		project   *uuid.UUID
		name      string
		logo      string
	}{
		{"folder", "folder", nil, nil, "Daily", `{}`},
		{"project", "project", &w.web, &w.web, "Web", webLogo},
		{"cycle", "cycle", &w.sprint, &w.web, "Sprint", `{}`},
		{"module", "module", &w.core, &w.web, "Core", `{}`},
		{"view", "view", &w.alicesView, &w.web, "Alice's", alicesLogo},
		{"workspace view", "view", &w.acmesView, nil, "Acme's", acmesLogo},
	}
	bobs := map[string]favoriteHTTP{}
	var made []favoriteHTTP
	for n, c := range cases {
		body := folderBody("  Daily  ")
		if c.target != nil {
			body = favoriteBody(c.kind, *c.target)
		}
		from := time.Now().UTC().Truncate(time.Microsecond)
		f, answer := w.favor(t, "bob", "acme", body, http.StatusCreated)
		want := favoriteHTTP{ID: f.ID, WorkspaceID: w.acme, ProjectID: c.project, EntityType: c.kind, EntityID: c.target,
			Sequence: 65535 + 10000*float64(n), Name: c.name, LogoProps: f.LogoProps, CreatedAt: f.CreatedAt}
		if !reflect.DeepEqual(f, want) || !sameJSON(t, f.LogoProps, c.logo) || f.ID == uuid.Nil() || f.CreatedAt.Before(from) || f.CreatedAt.After(time.Now()) {
			t.Errorf("bob's favorite of the %s: %s; want %+v, its icon %s", c.key, answer, want, c.logo)
		}
		if r := favoriteRowOf(t, w.pool, f.ID); r.userID != w.ids["bob"] || r.createdBy != r.userID || r.updatedBy != r.userID ||
			!r.createdAt.Equal(f.CreatedAt) || !r.updatedAt.Equal(f.CreatedAt) || r.deletedAt != nil {
			t.Errorf("its row: %+v", r)
		}
		bobs[c.key] = f
		made = append([]favoriteHTTP{f}, made...)
	}
	if got := w.listedFavorites(t, "bob", "acme"); !reflect.DeepEqual(got, made) {
		t.Errorf("bob lists %+v; want %+v", got, made)
	}
	w.bobsFolders["Daily"] = bobs["folder"].ID
	return bobs
}

// favorsAgain is the cell of a target favored again (spec 2.5): bob moves
// his favorite of Sprint into Daily, with another sequence, then favors
// Sprint again, at the top and into his folder Weekly: each answers the one
// he has, 200, the same bytes as its change answered, its folder and its
// sequence kept, and writes nothing, no row besides it.
func favorsAgain(t *testing.T, w favoritesWorld, bobs map[string]favoriteHTTP) {
	weekly := w.favorID(t, "bob", "acme", folderBody("Weekly"))
	w.bobsFolders["Weekly"] = weekly
	sprint := bobs["cycle"].ID
	_, moved := w.changeFavorite(t, "bob", sprint, fmt.Sprintf(`{"parent_id":%q,"sequence":1.5}`, bobs["folder"].ID))
	rows := favoriteRows(t, w.pool, sprint)
	for _, body := range []string{favoriteBody("cycle", w.sprint), favoriteBody("cycle", w.sprint, weekly)} {
		if _, again := w.favor(t, "bob", "acme", body, http.StatusOK); again != moved {
			t.Errorf("favored again with %s: %s; want %s", body, again, moved)
		}
	}
	var count int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM user_favorites WHERE user_id = $1 AND entity_type = 'cycle' AND entity_identifier = $2",
		w.ids["bob"], w.sprint).Scan(&count); err != nil || count != 1 {
		t.Errorf("bob's favorites of Sprint: %d, %v; want his one", count, err)
	}
	if got := favoriteRows(t, w.pool, sprint); got != rows {
		t.Errorf("favored again, its row %s; was %s", got, rows)
	}
}

// refusesWhatHeMayNotFavor is the cell of the targets one may not favor
// (spec 2.5, R7): a project and a cycle no target is; Side, bob's own
// project in Elsewhere, which he favors there but not in acme, and Away,
// his view of Elsewhere's own, which no project's read refuses, its
// workspace alone; Vault, acme's private project he is none of; Old,
// Web's archived cycle; and Shelf, his archived project. And Web and Vault
// for wanda, acme's admin who is none of their members: a workspace's
// admin reads none of its projects he is not a member of (Readable). Each
// is the one 422, byte for byte, entity_id not_allowed, and nothing is
// written.
func refusesWhatHeMayNotFavor(t *testing.T, w favoritesWorld) {
	count := func() int {
		var n int
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM user_favorites").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := count()
	w.refusedAlike(t, "bob", "acme", w.noTarget(t, "bob", "acme"), favoriteBody("cycle", uuid.NewV7()), favoriteBody("project", w.side),
		favoriteBody("view", w.away), favoriteBody("project", w.vault), favoriteBody("cycle", w.old), favoriteBody("project", w.shelf))
	w.refusedAlike(t, "wanda", "acme", w.noTarget(t, "wanda", "acme"), favoriteBody("project", w.web), favoriteBody("project", w.vault))
	if after := count(); after != before {
		t.Fatalf("the refusals wrote: %d favorites; were %d", after, before)
	}
	w.bobsElsewhere["Side"] = w.favorID(t, "bob", "elsewhere", favoriteBody("project", w.side))
}

// refusesAWorkspaceGuest is the workspace guest's cell (spec 2.2): gus,
// acme's member, makes his folder; made acme's guest by alice, his list,
// a creation, a change and a deletion of his folder are each refused 403,
// his folder kept; made its member again, he lists it.
func refusesAWorkspaceGuest(t *testing.T, w favoritesWorld) {
	folder := w.favorID(t, "gus", "acme", folderBody("Gus's"))
	membership := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.acme, w.ids["gus"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":5}`, http.StatusOK)
	for _, op := range []struct{ method, path, body string }{
		{http.MethodGet, favoritesPath("acme"), ""}, {http.MethodPost, favoritesPath("acme"), folderBody("Guest's")},
		{http.MethodPatch, favoritePath(folder), `{"name":"Guest's"}`}, {http.MethodDelete, favoritePath(folder), ""},
	} {
		w.refusedAs(t, "gus", op.method, op.path, op.body, http.StatusForbidden, "forbidden")
	}
	favoritesLive(t, w.pool, folder)
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":15}`, http.StatusOK)
	if got := w.listedFavoriteIDs(t, "gus", "acme"); !slices.Equal(got, []uuid.UUID{folder}) {
		t.Fatalf("gus lists %v; want his folder %s", got, folder)
	}
}

// sameJSON tells whether the JSON values a and b are equal, whatever their
// keys' order.
func sameJSON(t *testing.T, a json.RawMessage, b string) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(b), &y); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(x, y)
}
