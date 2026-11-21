package app_test

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The list (P3c spec 2.4, 2.5, 2.7) reads, in one read-only snapshot, the workspace and favorite.read's decision on it,
// then alice's favorites there as the store orders them, then their targets, each kind once by its ids, their
// projects, and the projects she reads; it answers them in the store's order, a folder with its own name and no icon,
// any other with its target's name and icon now, a cycle's and a module's {}. It reads no project's facts: the toggles
// do not matter.
func TestListFavoritesNamedByTheirTargets(t *testing.T) {
	s := newShelf()
	folder := favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Pinned", nil, 900)
	s.listed = []domain.Favorite{
		folder,
		favored(uuid.NewV7(), acme, domain.FavoriteProject, ptr(web), "", &folder.ID, 800),
		favored(uuid.NewV7(), acme, domain.FavoriteCycle, ptr(sprint), "", nil, 700),
		favored(uuid.NewV7(), acme, domain.FavoriteModule, ptr(docs), "", &folder.ID, 700),
		favored(uuid.NewV7(), acme, domain.FavoriteView, ptr(theirs), "", nil, 600),
		favored(uuid.NewV7(), acme, domain.FavoriteView, ptr(board), "", nil, 500),
		favored(uuid.NewV7(), acme, domain.FavoriteProject, ptr(ops), "", nil, 400),
	}
	s.projects[web] = app.ProjectFacts{WorkspaceID: acme}
	got, err := s.listFavorites().Execute(as(t), "acme")
	if err != nil {
		t.Fatal(err)
	}
	want := []app.NamedFavorite{
		{Favorite: s.listed[0], Name: "Pinned"}, {Favorite: s.listed[1], Name: "Web", LogoProps: icon}, {Favorite: s.listed[2], Name: "Sprint 1"},
		{Favorite: s.listed[3], Name: "Docs"}, {Favorite: s.listed[4], Name: "Theirs", LogoProps: icon}, {Favorite: s.listed[5], Name: "Board", LogoProps: icon},
		{Favorite: s.listed[6], Name: "Ops"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("listed:\n got %+v\nwant %+v", got, want)
	}
	wantCalls(t, s.world, "begin read", "slug acme", "authorize alice favorite.read acme/-", "favorites alice acme", "cycles sprint", "modules docs",
		"view targets "+s.idsOf([]uuid.UUID{theirs, board}), "project targets "+s.idsOf([]uuid.UUID{web, ops}), "readable alice acme", "end read")
	wroteNothing(t, s.world)
}

// Folders alone are listed without a read of any target or of the projects she reads; none is [].
func TestListFavoritesOfFoldersAlone(t *testing.T) {
	s := newShelf()
	got, err := s.listFavorites().Execute(as(t), "acme")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("listed %#v, %v; want []", got, err)
	}
	s.calls = nil
	s.listed = []domain.Favorite{favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Pinned", nil, 1)}
	if got, err := s.listFavorites().Execute(as(t), "acme"); err != nil || len(got) != 1 || got[0].Name != "Pinned" {
		t.Fatalf("listed %+v, %v; want the folder", got, err)
	}
	wantCalls(t, s.world, "begin read", "slug acme", "authorize alice favorite.read acme/-", "favorites alice acme", "end read")
}

// visibleRows are spec 2.5's table, row by row: a favorite of each target as alice sees it (listed) or not, alike for
// the list, and for a creation, which may favor what she sees in the workspace. Each change is another writer's, or
// her grants'.
var visibleRows = []struct {
	name   string
	change func(s *shelf)
	kind   domain.FavoriteKind
	target uuid.UUID
	listed bool
}{
	{"a project she reads", nil, domain.FavoriteProject, web, true},
	{"a project she does not read", func(s *shelf) { s.grants[acme] = s.grants[acme][1:] }, domain.FavoriteProject, web, false},
	{"the workspace's admin who reads no project", func(s *shelf) {
		s.grants[acme] = nil
		s.decide = func(shared.Action, shared.Target, bool) (shared.Grant, error) {
			return shared.Grant{WorkspaceRole: shared.RoleAdmin}, nil
		}
	}, domain.FavoriteProject, web, false},
	{"an archived project", func(s *shelf) { archive(s.projectTargets, web) }, domain.FavoriteProject, web, false},
	{"a deleted project", func(s *shelf) { delete(s.projectTargets, web) }, domain.FavoriteProject, web, false},
	{"a project whose views are turned off", func(s *shelf) { s.projects[web] = app.ProjectFacts{WorkspaceID: acme} }, domain.FavoriteProject, web, true},
	{"a cycle", nil, domain.FavoriteCycle, sprint, true},
	{"an archived cycle", func(s *shelf) { archive(s.cycles, sprint) }, domain.FavoriteCycle, sprint, false},
	{"a deleted cycle", func(s *shelf) { delete(s.cycles, sprint) }, domain.FavoriteCycle, sprint, false},
	{"a cycle of a project she does not read", func(s *shelf) { s.grants[acme] = s.grants[acme][1:] }, domain.FavoriteCycle, sprint, false},
	{"a cycle of an archived project", func(s *shelf) { archive(s.projectTargets, web) }, domain.FavoriteCycle, sprint, false},
	{"a cycle of a deleted project", func(s *shelf) { delete(s.projectTargets, web) }, domain.FavoriteCycle, sprint, false},
	{"a module", nil, domain.FavoriteModule, docs, true},
	{"an archived module", func(s *shelf) { archive(s.modules, docs) }, domain.FavoriteModule, docs, false},
	{"a deleted module", func(s *shelf) { delete(s.modules, docs) }, domain.FavoriteModule, docs, false},
	{"a module of an archived project", func(s *shelf) { archive(s.projectTargets, web) }, domain.FavoriteModule, docs, false},
	{"a module of a project she does not read", func(s *shelf) { s.grants[acme] = s.grants[acme][1:] }, domain.FavoriteModule, docs, false},
	{"another's view", nil, domain.FavoriteView, theirs, true},
	{"another's view while she sees her own alone", ownOnly, domain.FavoriteView, theirs, false},
	{"her own view while she sees her own alone", ownOnly, domain.FavoriteView, mine, true},
	{"a view whose project's views are turned off", func(s *shelf) { s.projects[web] = app.ProjectFacts{WorkspaceID: acme} }, domain.FavoriteView, theirs, true},
	{"a view of an archived project", func(s *shelf) { archive(s.projectTargets, web) }, domain.FavoriteView, mine, false},
	{"a view of a project she does not read", func(s *shelf) { s.grants[acme] = s.grants[acme][1:] }, domain.FavoriteView, mine, false},
	{"a deleted view", func(s *shelf) { delete(s.viewTargets, mine) }, domain.FavoriteView, mine, false},
	{"a workspace's view", nil, domain.FavoriteView, board, true},
	{"a deleted workspace's view", func(s *shelf) { delete(s.viewTargets, board) }, domain.FavoriteView, board, false},
}

// archive marks the target id of targets archived.
func archive[V app.ProjectTarget | app.PlanningTarget](targets map[uuid.UUID]V, id uuid.UUID) {
	switch t := any(targets[id]).(type) {
	case app.ProjectTarget:
		t.Archived = true
		targets[id] = any(t).(V)
	case app.PlanningTarget:
		t.Archived = true
		targets[id] = any(t).(V)
	}
}

// ownOnly makes alice a guest of web who sees her own alone.
func ownOnly(s *shelf) { s.grants[acme][0].OwnOnly = true }

// The list keeps every favorite and answers those whose target she sees (spec 2.5): one row of the table each, a
// folder always.
func TestListFavoritesSheSees(t *testing.T) {
	for _, row := range visibleRows {
		t.Run(row.name, func(t *testing.T) {
			s := newShelf()
			if row.change != nil {
				row.change(s)
			}
			f := favored(uuid.NewV7(), acme, row.kind, ptr(row.target), "", nil, 1)
			folder := favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Pinned", nil, 0)
			s.listed = []domain.Favorite{f, folder}
			got, err := s.listFavorites().Execute(as(t), "acme")
			if err != nil {
				t.Fatal(err)
			}
			want := []domain.Favorite{folder}
			if row.listed {
				want = []domain.Favorite{f, folder}
			}
			listed := make([]domain.Favorite, len(got))
			for i, n := range got {
				listed[i] = n.Favorite
			}
			if !reflect.DeepEqual(listed, want) {
				t.Errorf("listed %+v; want %+v", listed, want)
			}
		})
	}
}

// Every list it cannot answer is workspace.not_found, the one value: a slug of no workspace, and a workspace she does
// not see, also when her role would be refused; a role refused on one she sees is the Authorizer's 403. Neither reads a
// favorite.
func TestListFavoritesRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		slug   string
		decide func(shared.Action, shared.Target, bool) (shared.Grant, error)
		calls  []string
	}{
		{"no workspace", "nowhere", nil, []string{"begin read", "slug nowhere", "end read"}},
		{"a workspace she does not see", "acme", refuse(shared.ErrNotVisible), []string{"begin read", "slug acme", "authorize alice favorite.read acme/-", "end read"}},
		{"a guest", "acme", refuse(shared.Forbidden()), []string{"begin read", "slug acme", "authorize alice favorite.read acme/-", "end read"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := newShelf()
			if tt.decide != nil {
				s.decide = tt.decide
			}
			_, err := s.listFavorites().Execute(as(t), tt.slug)
			if tt.name == "a guest" {
				wantForbidden(t, err)
			} else {
				wantSame(t, err, domain.ErrWorkspaceNotFound)
			}
			wantCalls(t, s.world, tt.calls...)
		})
	}
}

// refuse is a decision that refuses with err.
func refuse(err error) func(shared.Action, shared.Target, bool) (shared.Grant, error) {
	return func(shared.Action, shared.Target, bool) (shared.Grant, error) { return shared.Grant{}, err }
}
