package app

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// NamedFavorite is a favorite as the module answers it (P3c spec 2.4, R8):
// the favorite, and what it is called and shown with now: a folder's own
// name and no icon; its target's name and, a project's and a view's, its
// icon; a cycle and a module have none.
type NamedFavorite struct {
	Favorite  domain.Favorite
	Name      string
	LogoProps shared.LogoProps
}

// entity is the target a favorite of one names (spec 2.3): its kind,
// entity_type, and its id, entity_id. A folder names none.
type entity struct {
	kind domain.FavoriteKind
	id   uuid.UUID
}

// entityOf is the target f names; false for a folder.
func entityOf(f domain.Favorite) (entity, bool) {
	if f.Kind == domain.FavoriteFolder {
		return entity{}, false
	}
	return entity{kind: f.Kind, id: *f.EntityID}, true
}

// Targets are the reads of what favorites name and of what a caller sees
// of it (spec 2.5): the projects he reads (Readable), and the projects,
// the cycles, the modules and the views favorites name, each read by ids
// in one statement. A list's reads and a write's under its locks are the
// same reads, and sight decides on what they read with the one predicate.
type Targets struct {
	Readable Readable
	Projects ProjectTargets
	Planning PlanningTargets
	Views    ViewTargets
}

// sight is what a caller sees, in a workspace, of the targets some
// favorites name (spec 2.5): the projects he reads, and those of the
// targets, and the targets' projects, that are undeleted, as one read
// found them. A target or a project that is not among them is deleted or
// not there.
type sight struct {
	workspaceID, actor uuid.UUID
	grants             map[uuid.UUID]ProjectGrant
	projects           map[uuid.UUID]ProjectTarget
	cycles, modules    map[uuid.UUID]PlanningTarget
	views              map[uuid.UUID]domain.View
}

// read reads, in the transaction ctx carries, the targets ts name and the
// projects of those it finds, each kind once, by its ids in id order, and
// none of a kind none of ts names: what a sight decides on, but for the
// projects a caller reads.
func (t Targets) read(ctx context.Context, ts []entity) (sight, error) {
	ids := map[domain.FavoriteKind][]uuid.UUID{}
	for _, x := range ts {
		ids[x.kind] = append(ids[x.kind], x.id)
	}
	var s sight
	var err error
	if s.cycles, err = readSome(ctx, t.Planning.Cycles, ids[domain.FavoriteCycle]); err != nil {
		return sight{}, err
	}
	if s.modules, err = readSome(ctx, t.Planning.Modules, ids[domain.FavoriteModule]); err != nil {
		return sight{}, err
	}
	if s.views, err = readSome(ctx, t.Views.ViewTargets, ids[domain.FavoriteView]); err != nil {
		return sight{}, err
	}
	projects := ids[domain.FavoriteProject]
	for _, planned := range []map[uuid.UUID]PlanningTarget{s.cycles, s.modules} {
		for _, p := range planned {
			projects = append(projects, p.ProjectID)
		}
	}
	for _, v := range s.views {
		if v.ProjectID != nil {
			projects = append(projects, *v.ProjectID)
		}
	}
	if s.projects, err = readSome(ctx, t.Projects.ProjectTargets, projects); err != nil {
		return sight{}, err
	}
	return s, nil
}

// sight reads, in the transaction ctx carries, what actor sees in the
// workspace of the targets ts name (read), and, when any of them is in a
// project, the projects he reads there (Readable).
func (t Targets) sight(ctx context.Context, actor shared.Actor, workspaceID uuid.UUID, ts []entity) (sight, error) {
	s, err := t.read(ctx, ts)
	if err != nil {
		return sight{}, err
	}
	s.workspaceID, s.actor = workspaceID, actor.UserID
	if len(s.projects) == 0 {
		return s, nil
	}
	grants, err := t.Readable.Readable(ctx, actor, workspaceID)
	if err != nil {
		return sight{}, err
	}
	s.grants = make(map[uuid.UUID]ProjectGrant, len(grants))
	for _, g := range grants {
		s.grants[g.ProjectID] = g
	}
	return s, nil
}

// readSome reads the ids, each once, in id order, by read; none, and no
// read, for no ids.
func readSome[V any](ctx context.Context, read func(context.Context, []uuid.UUID) (map[uuid.UUID]V, error), ids []uuid.UUID) (map[uuid.UUID]V, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ids = slices.Clone(ids)
	slices.SortFunc(ids, uuid.UUID.Compare)
	return read(ctx, slices.Compact(ids))
}

// aim is a target as a sight finds it: its workspace, its project (nil for
// a workspace's view), its name and its icon, and whether the caller sees
// it.
type aim struct {
	workspaceID uuid.UUID
	projectID   *uuid.UUID
	name        string
	logo        shared.LogoProps
	visible     bool
}

// find is the target t as s read it, found when it is undeleted, and
// whether the caller sees it now (spec 2.5): a project he reads, unarchived
// (sees); an unarchived cycle or module of a project he sees; a project's
// view in a project he sees, unless the guest rule hides it from him, the
// one HiddenFrom of the views, with his grant there; a workspace's view,
// which every role of favorite.* reads (workspace_view.read). The
// projects' toggles do not matter (spec 2.5, R5).
func (s sight) find(t entity) (aim, bool) {
	switch t.kind {
	case domain.FavoriteProject:
		p, found := s.projects[t.id]
		id := t.id
		return aim{workspaceID: p.WorkspaceID, projectID: &id, name: p.Name, logo: p.LogoProps, visible: s.sees(id)}, found
	case domain.FavoriteCycle, domain.FavoriteModule:
		planned := s.cycles
		if t.kind == domain.FavoriteModule {
			planned = s.modules
		}
		p, found := planned[t.id]
		project := p.ProjectID
		return aim{workspaceID: p.WorkspaceID, projectID: &project, name: p.Name, visible: !p.Archived && s.sees(project)}, found
	case domain.FavoriteView:
		v, found := s.views[t.id]
		a := aim{workspaceID: v.WorkspaceID, projectID: v.ProjectID, name: v.Name, logo: v.LogoProps, visible: true}
		if v.ProjectID != nil {
			a.visible = s.sees(*v.ProjectID) && !v.HiddenFrom(s.actor, shared.Grant{OwnOnly: s.grants[*v.ProjectID].OwnOnly})
		}
		return a, found
	}
	return aim{}, false
}

// sees tells whether the caller sees the project id (spec 2.5): one he
// reads (Readable), undeleted and unarchived.
func (s sight) sees(id uuid.UUID) bool {
	p, found := s.projects[id]
	_, reads := s.grants[id]
	return found && reads && !p.Archived
}

// eligible is the target t as s found it, and whether the caller may favor
// it (spec 2.5, R7): one he sees, of the workspace. One that is not there,
// is deleted, is another workspace's, he does not see or is archived is
// no target he may favor, all alike.
func (s sight) eligible(t entity) (aim, bool) {
	a, found := s.find(t)
	return a, found && a.visible && a.workspaceID == s.workspaceID
}

// visible are those of fs the caller sees, in their order, each answered
// with what s found of its target (shows); one whose target he no longer
// sees is kept, not listed (M7 design 3.8).
func (s sight) visible(fs []domain.Favorite) []NamedFavorite {
	out := make([]NamedFavorite, 0, len(fs))
	for _, f := range fs {
		if n, shown := s.shows(f); shown {
			out = append(out, n)
		}
	}
	return out
}

// shows is f answered with what s found of its target, and whether the
// caller sees it (spec 2.4, 2.5, 2.7): a folder always; a favorite of a
// target he sees (find). The list lists, and a change and a deletion
// reach, those he sees alone: the one predicate of the three.
func (s sight) shows(f domain.Favorite) (NamedFavorite, bool) {
	t, ofTarget := entityOf(f)
	if !ofTarget {
		return answered(f, aim{}), true
	}
	a, found := s.find(t)
	return answered(f, a), found && a.visible
}

// answered is f answered with what a sight found of its target, a (spec
// 2.4): a folder with its own name and no icon, any other with its
// target's name and icon.
func answered(f domain.Favorite, a aim) NamedFavorite {
	if f.Kind == domain.FavoriteFolder {
		return NamedFavorite{Favorite: f, Name: f.Name}
	}
	return NamedFavorite{Favorite: f, Name: a.name, LogoProps: a.logo}
}
