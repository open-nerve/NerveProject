package app

import (
	"context"
	"time"
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

// ProjectTarget is a project as a favorite or a recent visit reads it (P3c
// spec 2.4, 2.5, P3d spec 2.4, R4): its workspace; its name and its icon,
// which the favorite is answered with, and its identifier too, which a
// recent visit is; and whether it is archived, which hides it from the
// favorites alone (P3d R5). Its fields are project's NavigationTarget's,
// which bootstrap converts into it.
type ProjectTarget struct {
	WorkspaceID uuid.UUID
	Name        string
	Identifier  string
	LogoProps   shared.LogoProps
	Archived    bool
}

// ProjectTargets reads the projects favorites and recent visits name, a
// project's and the projects of their other targets (P3c spec 2.5, P3d
// spec 2.5): project's NavigationTargets, which bootstrap converts. It
// reads the undeleted projects among ids, archived or not, of any
// workspace, by id, without a lock, in the transaction ctx carries or on
// the pool; a deleted project and an id of none are not among them.
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
// target's project's FOR SHARE (spec 2.8), the favorites' rows in their one
// total order, a folder's before an item's, each lock its own statement.
// ShareView locks FOR SHARE the undeleted view a favorite names, a deleted
// one and an id of none not at all, which the reads under the locks find.
// ShareFolder locks FOR SHARE the parent a write names, before the
// favorite's own row: found is true when id is user's own undeleted folder
// in workspace; any other id is no parent, and is not locked (spec 2.5).
// ShareFavoriteOfTarget locks FOR SHARE user's undeleted favorite of the
// target of kind with id, after the parent's FOR SHARE, and reads it: a
// creation's of a target last lock, under which it answers that favorite
// as it is, for its deletion waits until the creation ends; found is false
// when he has none, as when the one he had was deleted while the lock
// waited, and another's is not locked. LockFavorite locks FOR UPDATE user's
// undeleted favorite id, after its new parent's FOR SHARE, and reads it: a
// change's or a deletion's read under its locks; found is false when there
// is none, as when id is another's, which it does not lock.
// LockFolderChildren locks FOR UPDATE, in id order, the undeleted favorites
// the folder holds, after the folder's own FOR UPDATE: a folder's
// deletion's last locks.
type FavoriteLocker interface {
	ShareView(ctx context.Context, id uuid.UUID) error
	ShareFolder(ctx context.Context, user, workspace, id uuid.UUID) (found bool, err error)
	ShareFavoriteOfTarget(ctx context.Context, user uuid.UUID, kind domain.FavoriteKind, id uuid.UUID) (f domain.Favorite, found bool, err error)
	LockFavorite(ctx context.Context, user, id uuid.UUID) (f domain.Favorite, found bool, err error)
	LockFolderChildren(ctx context.Context, folder uuid.UUID) error
}

// FavoriteReader reads the favorites without a lock, in the transaction ctx
// carries or on the pool. Favorites lists user's undeleted favorites in
// workspace, those in his folders too, by sequence descending, then by id
// descending (spec 2.4). Favorite reads user's undeleted favorite id, a
// change's or a deletion's read before its locks; found is false when
// there is none, as when id is another's (spec 2.7).
type FavoriteReader interface {
	Favorites(ctx context.Context, user, workspace uuid.UUID) ([]domain.Favorite, error)
	Favorite(ctx context.Context, user, id uuid.UUID) (f domain.Favorite, found bool, err error)
}

// FavoriteWriter writes the favorites in the transaction ctx carries, which
// holds a write's locks (spec 2.5, 2.8). GreatestFavoriteSequence is the
// greatest sequence of user's undeleted favorites in workspace, nil when he
// has none (domain.SequenceAfter). InsertFavorite stores the new favorite
// f, made by by at its CreatedAt, and answers it, created; for a favorite
// of a target whose owner has an undeleted favorite of it, his other
// creation's at once, it stores nothing and answers that one as it is, not
// created; a folder's name, in any case, another of its owner's undeleted
// folders in the workspace has is domain.ErrFavoriteFolderNameTaken.
// UpdateFavorite writes the parent, the sequence and, a folder's, the name
// of the undeleted favorite f.ID as f has them, last changed by by at
// f.UpdatedAt; a folder's name, in any case, that another of its owner's
// undeleted folders in the workspace has is
// domain.ErrFavoriteFolderNameTaken, its own in another case is not.
// DeleteFavorite soft-deletes, in one statement, the undeleted favorite id
// and, a folder, the undeleted favorites it holds, each deleted and last
// changed by by at at; one deleted already keeps its deletion.
type FavoriteWriter interface {
	GreatestFavoriteSequence(ctx context.Context, user, workspace uuid.UUID) (*float64, error)
	InsertFavorite(ctx context.Context, f domain.Favorite, by uuid.UUID) (stored domain.Favorite, created bool, err error)
	UpdateFavorite(ctx context.Context, f domain.Favorite, by uuid.UUID) error
	DeleteFavorite(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error
}

// FavoriteCascades soft-deletes, in the transaction ctx carries, which
// holds the lock of what took them away (spec 2.6, 2.8), every account's
// undeleted favorites of a target deleted or archived, each locked FOR NO
// KEY UPDATE in id order, deleted and last changed by by at at; one deleted
// before keeps its deletion. DeleteFavoritesOfTarget deletes those of the
// target of kind with id id, a cycle, a module or a view;
// DeleteFavoritesOfProject those whose project is project, its own and its
// cycles', modules' and views', and no folder, which has no project.
type FavoriteCascades interface {
	DeleteFavoritesOfTarget(ctx context.Context, kind domain.FavoriteKind, id uuid.UUID, at time.Time, by uuid.UUID) error
	DeleteFavoritesOfProject(ctx context.Context, project uuid.UUID, at time.Time, by uuid.UUID) error
}
