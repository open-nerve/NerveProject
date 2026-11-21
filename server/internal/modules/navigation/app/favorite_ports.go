package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectGrant is a project whose targets a caller sees (P3c spec 2.5): one
// he reads the work items of, only those he created when OwnOnly, which
// hides from him the views of others (View.HiddenFrom).
type ProjectGrant struct {
	ProjectID uuid.UUID
	OwnOnly   bool
}

// Readable are the projects a caller reads in a workspace now (spec 2.5):
// the access module's ReadableProjects, which bootstrap converts. It decides
// each by Authorize's rule of issue.read, the rule of view.read too: the
// workspace's undeleted projects, archived or not, he is an active member
// of, a guest among them, while he is one of the workspace, each OwnOnly by
// his project role; none when he is no member of the workspace. So the
// workspace's admin who is a member of none reads none. It reads without a
// lock, in the transaction ctx carries or on the pool.
type Readable interface {
	Readable(ctx context.Context, actor shared.Actor, workspaceID uuid.UUID) ([]ProjectGrant, error)
}

// ProjectTarget is a project as a favorite reads it (spec 2.4, 2.5): its
// workspace, its name and its icon, which the favorite is answered with,
// and whether it is archived, which hides it.
type ProjectTarget struct {
	WorkspaceID uuid.UUID
	Name        string
	LogoProps   shared.LogoProps
	Archived    bool
}

// ProjectTargets reads the projects favorites name, a project's favorites
// and the projects of their other targets (spec 2.5): project's
// NavigationTargets, which bootstrap converts. It reads the undeleted
// projects among ids, archived or not, of any workspace, by id, without a
// lock, in the transaction ctx carries or on the pool; a deleted project
// and an id of none are not among them.
type ProjectTargets interface {
	ProjectTargets(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]ProjectTarget, error)
}

// PlanningTarget is a cycle or a module as a favorite reads it (spec 2.4,
// 2.5): its workspace and its project, its name, and whether it is
// archived, which hides it. Neither has an icon.
type PlanningTarget struct {
	WorkspaceID, ProjectID uuid.UUID
	Name                   string
	Archived               bool
}

// PlanningTargets reads the cycles and the modules favorites name (spec
// 2.5): planning's NavigationTargets, which bootstrap converts. Each reads
// the undeleted ones among ids, archived or not, of any workspace, by id,
// without a lock, in the transaction ctx carries or on the pool, whatever
// their projects' toggles; a deleted one and an id of none are not among
// them.
type PlanningTargets interface {
	Cycles(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]PlanningTarget, error)
	Modules(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]PlanningTarget, error)
}

// ViewTargets reads the views favorites name (spec 2.5): the undeleted
// views among ids, of either kind and any workspace, by id, without a lock,
// in the transaction ctx carries or on the pool; a deleted view and an id
// of none are not among them.
type ViewTargets interface {
	ViewTargets(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.View, error)
}

// FavoriteLocker takes, in the transaction ctx carries, the locks a write of
// a favorite takes on the module's own rows, after its workspace's and its
// target's project's FOR SHARE (spec 2.8). ShareView locks FOR SHARE the
// undeleted view a favorite names, a deleted one and an id of none not at
// all, which the reads under the locks find. ShareFolder locks FOR SHARE
// the parent a write names, before the favorite's own row: found is true
// when id is user's own undeleted folder in workspace; any other id is no
// parent, and is not locked (spec 2.5).
type FavoriteLocker interface {
	ShareView(ctx context.Context, id uuid.UUID) error
	ShareFolder(ctx context.Context, user, workspace, id uuid.UUID) (found bool, err error)
}

// FavoriteReader lists, in the transaction ctx carries or on the pool,
// user's undeleted favorites in workspace, those in his folders too, by
// sequence descending, then by id descending (spec 2.4).
type FavoriteReader interface {
	Favorites(ctx context.Context, user, workspace uuid.UUID) ([]domain.Favorite, error)
}

// FavoriteWriter writes the favorites in the transaction ctx carries, which
// holds a write's locks (spec 2.5, 2.8). GreatestFavoriteSequence is the
// greatest sequence of user's undeleted favorites in workspace, nil when he
// has none (domain.SequenceAfter). InsertFavorite stores the new favorite
// f, made by by at its CreatedAt, and answers it, created; for a favorite
// of a target whose owner has an undeleted favorite of it, it stores
// nothing and answers that one as it is, not created; a folder's name, in
// any case, another of its owner's undeleted folders in the workspace has
// is domain.ErrFavoriteFolderNameTaken.
type FavoriteWriter interface {
	GreatestFavoriteSequence(ctx context.Context, user, workspace uuid.UUID) (*float64, error)
	InsertFavorite(ctx context.Context, f domain.Favorite, by uuid.UUID) (stored domain.Favorite, created bool, err error)
}
