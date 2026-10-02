package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Locks is the one way a write on a project named by its id takes its
// locks and its decision (M3 design 3.6 convention 2), in the transaction
// ctx carries: the project's workspace, read without a lock; the
// workspace's row FOR SHARE, the write's first lock; the memberships of the
// workspace of the accounts the write makes members of the project, FOR
// SHARE in id order (convention 3); the project's row, FOR NO KEY UPDATE,
// or FOR SHARE for a write under the project that leaves the row and its
// memberships as they are, still undeleted and of that workspace; then the
// decision, under them all. Every cascade over the workspace's projects
// runs under the workspace's FOR NO KEY UPDATE and reads its time after it
// (3.3): while a write holds the workspace FOR SHARE, no cascade touches
// the rows it writes. project.New builds one Locks for every write: a
// write holds no Authorizer of its own, so it decides only under these
// locks.
type Locks struct {
	projects   ProjectLocks
	workspaces WorkspaceSharer
	members    WorkspaceMembers
	auth       shared.Authorizer
}

// NewLocks returns the locks over the project store, the workspace
// module's directory and members' lock, and the Authorizer.
func NewLocks(projects ProjectLocks, workspaces WorkspaceSharer, members WorkspaceMembers, auth shared.Authorizer) Locks {
	return Locks{projects: projects, workspaces: workspaces, members: members, auth: auth}
}

// write is a write on a project, as Locks takes its locks.
type write struct {
	project uuid.UUID
	action  shared.Action
	// share locks the project FOR SHARE, for a write under the project that
	// leaves the project row and its memberships as they are; otherwise it
	// is locked FOR NO KEY UPDATE.
	share bool
	// targets are the accounts the write makes members of the project.
	targets []uuid.UUID
}

// held is a write's locks taken and its decision made: the project as its
// lock read it, the caller's grant, and the active roles in the workspace
// of the write's targets, by account.
type held struct {
	project LockedProject
	grant   shared.Grant
	roles   map[uuid.UUID]shared.Role
}

// lockAndDecide takes w's locks and decides w's action on the project, in
// the order of Locks. A project that is not there, deleted while a lock
// waited, or not visible to actor is domain.ErrNotFound, and so is one
// whose workspace was deleted while its lock waited; a role the rule does
// not allow is the Authorizer's shared.Forbidden.
func (l Locks) lockAndDecide(ctx context.Context, actor shared.Actor, w write) (held, error) {
	workspaceID, found, err := l.projects.ProjectWorkspace(ctx, w.project)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, domain.ErrNotFound
	}
	_, found, err = l.workspaces.ShareWorkspaceByID(ctx, workspaceID)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, domain.ErrNotFound
	}
	var h held
	if len(w.targets) > 0 {
		if h.roles, err = l.members.ShareMembers(ctx, workspaceID, w.targets); err != nil {
			return held{}, err
		}
	}
	lock := l.projects.LockProject
	if w.share {
		lock = l.projects.ShareProject
	}
	h.project, found, err = lock(ctx, w.project)
	switch {
	case err != nil:
		return held{}, err
	case !found, h.project.WorkspaceID != workspaceID:
		return held{}, domain.ErrNotFound
	}
	if h.grant, err = decide(ctx, l.auth, actor, w.action, workspaceID, w.project); err != nil {
		return held{}, err
	}
	return h, nil
}

// findAndDecide is the first two steps of a read under a project named by
// its id (M3 design 6.4), without a transaction: projects finds the
// undeleted project's workspace, then decide decides action on it. A
// project that is not there, or not visible to actor, is domain.ErrNotFound;
// a role the rule does not allow is the Authorizer's shared.Forbidden.
func findAndDecide(ctx context.Context, projects ProjectFinder, auth shared.Authorizer, actor shared.Actor, id uuid.UUID,
	action shared.Action) error {
	workspaceID, found, err := projects.ProjectWorkspace(ctx, id)
	switch {
	case err != nil:
		return err
	case !found:
		return domain.ErrNotFound
	}
	_, err = decide(ctx, auth, actor, action, workspaceID, id)
	return err
}

// decide asks the Authorizer for action on the project id of the workspace
// for actor. A write calls it under its locks, so the facts it reads are
// the ones committed after the locks were granted (M3 design 6.7): a
// demotion or a removal that committed while the write waited is seen. A
// project not visible to actor is domain.ErrNotFound, the 404 of what the
// caller named.
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, action shared.Action, workspaceID, id uuid.UUID) (shared.Grant, error) {
	grant, err := auth.Authorize(ctx, actor, action, shared.Target{WorkspaceID: workspaceID, ProjectID: id})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return shared.Grant{}, domain.ErrNotFound
	case err != nil:
		return shared.Grant{}, err
	}
	return grant, nil
}

// answer reads the project id back as the caller sees it, in the transaction
// ctx carries: a write's answer. A project not there after the write is an
// internal error, not a 404.
func answer(ctx context.Context, projects ProjectReader, id, userID uuid.UUID) (domain.Project, error) {
	p, found, err := projects.GetProject(ctx, id, userID)
	switch {
	case err != nil:
		return domain.Project{}, err
	case !found:
		return domain.Project{}, errors.New("project " + id.String() + " is not there after its write")
	}
	return p, nil
}
