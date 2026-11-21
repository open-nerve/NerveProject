package app_test

import (
	"context"
	"slices"
	"strings"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The other workspace, the other project of acme, beta's project, and the targets of the favorites' tests: a cycle and
// a module of web, a view of web and a view of acme, each alice's or bob's, and beta's own view.
var (
	beta, ops, site                = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	sprint, docs                   = uuid.NewV7(), uuid.NewV7()
	mine, theirs, board, elsewhere = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	icon                           = shared.LogoProps{InUse: ptr(shared.LogoInUseIcon), Icon: &shared.LogoIcon{Name: ptr("star")}}
)

func ptr[T any](v T) *T { return &v }

// shelf is the world with the fakes of the favorites' ports: Readable, ProjectTargets, PlanningTargets, ViewTargets,
// FavoriteLocker, FavoriteReader and FavoriteWriter, each logging its calls in the world's log. meanwhile, when not
// nil, runs once when the workspace's lock is granted: what another writer committed while the write's check ran and
// its lock waited, which the reads under the locks find.
type shelf struct {
	*world
	meanwhile func(s *shelf)
	// grants are Readable's answer, by workspace.
	grants map[uuid.UUID][]app.ProjectGrant
	// projectTargets, cycles, modules and viewTargets are the undeleted targets, of any workspace.
	projectTargets  map[uuid.UUID]app.ProjectTarget
	cycles, modules map[uuid.UUID]app.PlanningTarget
	viewTargets     map[uuid.UUID]domain.View
	// stored are the favorites stored, deleted those of them deleted; listed is what Favorites answers.
	stored  []domain.Favorite
	deleted map[uuid.UUID]bool
	listed  []domain.Favorite
	// inserted are the favorites InsertFavorite stored, and their writers.
	inserted []insertedFavorite
}

// insertedFavorite is a favorite InsertFavorite stored and its writer.
type insertedFavorite struct {
	favorite domain.Favorite
	by       uuid.UUID
}

// newShelf is newWorld with ops, a project of acme, and beta, another workspace, with its project site; alice reads web
// and ops, each fully. Their targets: web and ops, active; site, beta's; the cycle sprint and the module docs of web;
// the views mine (alice's) and theirs (bob's) of web, board, acme's own (bob's), and elsewhere, beta's own.
func newShelf() *shelf {
	w := newWorld()
	for id, name := range map[uuid.UUID]string{beta: "beta", ops: "ops", site: "site", sprint: "sprint", docs: "docs", mine: "mine",
		theirs: "theirs", board: "board", elsewhere: "elsewhere"} {
		w.names[id] = name
	}
	w.projects[ops] = app.ProjectFacts{WorkspaceID: acme, IssueViewsView: true}
	w.projects[site] = app.ProjectFacts{WorkspaceID: beta, IssueViewsView: true}
	view := func(id, workspace uuid.UUID, project *uuid.UUID, owner uuid.UUID, name string) domain.View {
		return domain.View{ID: id, WorkspaceID: workspace, ProjectID: project, OwnedByID: owner, Content: domain.Content{Name: name, LogoProps: icon}}
	}
	return &shelf{
		world:  w,
		grants: map[uuid.UUID][]app.ProjectGrant{acme: {{ProjectID: web}, {ProjectID: ops}}},
		projectTargets: map[uuid.UUID]app.ProjectTarget{
			web: {WorkspaceID: acme, Name: "Web", LogoProps: icon}, ops: {WorkspaceID: acme, Name: "Ops"}, site: {WorkspaceID: beta, Name: "Site"}},
		cycles:  map[uuid.UUID]app.PlanningTarget{sprint: {WorkspaceID: acme, ProjectID: web, Name: "Sprint 1"}},
		modules: map[uuid.UUID]app.PlanningTarget{docs: {WorkspaceID: acme, ProjectID: web, Name: "Docs"}},
		viewTargets: map[uuid.UUID]domain.View{
			mine: view(mine, acme, ptr(web), alice, "Mine"), theirs: view(theirs, acme, ptr(web), bob, "Theirs"),
			board: view(board, acme, nil, bob, "Board"), elsewhere: view(elsewhere, beta, nil, alice, "Elsewhere")},
		deleted: map[uuid.UUID]bool{},
	}
}

// idsOf names ids as the log does, in id order, comma-separated.
func (s *shelf) idsOf(ids []uuid.UUID) string {
	sorted := slices.Clone(ids)
	slices.SortFunc(sorted, uuid.UUID.Compare)
	names := make([]string, len(sorted))
	for i, id := range sorted {
		names[i] = s.name(id)
	}
	return strings.Join(names, ",")
}

// of are those of all among ids, by id.
func of[V any](all map[uuid.UUID]V, ids []uuid.UUID) map[uuid.UUID]V {
	out := map[uuid.UUID]V{}
	for _, id := range ids {
		if v, found := all[id]; found {
			out[id] = v
		}
	}
	return out
}

func (s *shelf) ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (bool, error) {
	if s.meanwhile != nil {
		s.meanwhile(s)
		s.meanwhile = nil
	}
	return s.world.ShareWorkspaceByID(ctx, id)
}

func (s *shelf) Readable(_ context.Context, actor shared.Actor, workspaceID uuid.UUID) ([]app.ProjectGrant, error) {
	s.log("readable %s %s", s.name(actor.UserID), s.name(workspaceID))
	return slices.Clone(s.grants[workspaceID]), nil
}

func (s *shelf) ProjectTargets(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]app.ProjectTarget, error) {
	s.log("project targets %s", s.idsOf(ids))
	return of(s.projectTargets, ids), nil
}

func (s *shelf) Cycles(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]app.PlanningTarget, error) {
	s.log("cycles %s", s.idsOf(ids))
	return of(s.cycles, ids), nil
}

func (s *shelf) Modules(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]app.PlanningTarget, error) {
	s.log("modules %s", s.idsOf(ids))
	return of(s.modules, ids), nil
}

func (s *shelf) ViewTargets(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.View, error) {
	s.log("view targets %s", s.idsOf(ids))
	return of(s.viewTargets, ids), nil
}

func (s *shelf) ShareView(_ context.Context, id uuid.UUID) error {
	s.log("share view %s", s.name(id))
	return nil
}

// ShareFolder mirrors the store's rule (favorites.sql ShareFolder): the parent is user's own undeleted folder in
// workspace.
func (s *shelf) ShareFolder(_ context.Context, user, workspace, id uuid.UUID) (bool, error) {
	s.log("share folder %s", s.name(id))
	for _, f := range s.stored {
		if f.ID == id {
			return f.UserID == user && f.WorkspaceID == workspace && f.Kind == domain.FavoriteFolder && !s.deleted[id], nil
		}
	}
	return false, nil
}

func (s *shelf) Favorites(_ context.Context, user, workspace uuid.UUID) ([]domain.Favorite, error) {
	s.log("favorites %s %s", s.name(user), s.name(workspace))
	return s.listed, nil
}

// live are the stored favorites of user in workspace that are not deleted.
func (s *shelf) live(user, workspace uuid.UUID) []domain.Favorite {
	var out []domain.Favorite
	for _, f := range s.stored {
		if f.UserID == user && f.WorkspaceID == workspace && !s.deleted[f.ID] {
			out = append(out, f)
		}
	}
	return out
}

func (s *shelf) GreatestFavoriteSequence(_ context.Context, user, workspace uuid.UUID) (*float64, error) {
	s.log("greatest %s %s", s.name(user), s.name(workspace))
	var greatest *float64
	for _, f := range s.live(user, workspace) {
		if greatest == nil || f.Sequence > *greatest {
			greatest = ptr(f.Sequence)
		}
	}
	return greatest, nil
}

// InsertFavorite mirrors the store's two insertions (favorites.sql): a favorite of a target its owner has an undeleted
// favorite of answers that one, not created; a folder's name, in any case, of another of his undeleted folders in the
// workspace is taken.
func (s *shelf) InsertFavorite(_ context.Context, f domain.Favorite, by uuid.UUID) (domain.Favorite, bool, error) {
	s.log("insert %s by %s", f.Kind, s.name(by))
	for _, o := range s.stored {
		if s.deleted[o.ID] || o.UserID != f.UserID {
			continue
		}
		if f.Kind != domain.FavoriteFolder && o.Kind == f.Kind && *o.EntityID == *f.EntityID {
			return o, false, nil
		}
		if f.Kind == domain.FavoriteFolder && o.Kind == f.Kind && o.WorkspaceID == f.WorkspaceID && strings.EqualFold(o.Name, f.Name) {
			return domain.Favorite{}, false, domain.ErrFavoriteFolderNameTaken
		}
	}
	s.stored = append(s.stored, f)
	s.inserted = append(s.inserted, insertedFavorite{f, by})
	return f, true, nil
}

// targets are the shelf's reads of targets.
func (s *shelf) targets() app.Targets {
	return app.Targets{Readable: s, Projects: s, Planning: s, Views: s}
}

// listFavorites is the shelf's list, in its "read" transactions.
func (s *shelf) listFavorites() *app.ListFavorites {
	return app.NewListFavorites(app.ListFavoritesDeps{Workspaces: s, Authorizer: s, Favorites: s, Targets: s.targets(), Tx: tx{s.world, "read"}})
}

// createFavorite is the shelf's creation, in its "write" transactions.
func (s *shelf) createFavorite() *app.CreateFavorite {
	return app.NewCreateFavorite(app.FavoriteWriteDeps{Locks: app.NewFavoriteLocks(s, s, s, s.targets(), s, tx{s.world, "write"}), Favorites: s, Clock: s})
}

// favored is alice's stored favorite numbered by id, of kind, naming entity (nil for a folder), named name (a
// folder's), in parent, at sequence, in workspace.
func favored(id uuid.UUID, workspace uuid.UUID, kind domain.FavoriteKind, entity *uuid.UUID, name string, parent *uuid.UUID, sequence float64) domain.Favorite {
	f := domain.Favorite{ID: id, WorkspaceID: workspace, UserID: alice, Kind: kind, EntityID: entity, Name: name, ParentID: parent, Sequence: sequence,
		CreatedAt: moment.Add(-1), UpdatedAt: moment.Add(-1)}
	switch kind {
	case domain.FavoriteProject:
		f.ProjectID = entity
	case domain.FavoriteCycle, domain.FavoriteModule:
		f.ProjectID = ptr(web)
	case domain.FavoriteView:
		f.ProjectID = map[uuid.UUID]*uuid.UUID{mine: ptr(web), theirs: ptr(web)}[*entity]
	}
	return f
}
