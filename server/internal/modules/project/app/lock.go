package app

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Locks is the one way a write on a project takes its locks and its
// decision (M3 design 3.6 convention 2), in the transaction ctx carries:
// the project's workspace, read without a lock (from the project, or from
// the membership a write on one names); the workspace's row FOR SHARE, the
// write's first lock; the memberships of the workspace of the accounts the
// write makes members of the project, or whose role it changes, FOR SHARE
// in id order (convention 3); the project's row, FOR NO KEY UPDATE, or FOR
// SHARE for a write under the project that leaves the row and its
// memberships as they are, still undeleted and of that workspace; the
// membership a write on one names, read again, still of that project; then
// the decision, under them all. Every cascade over the workspace's projects
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
	// targets are the accounts the write makes members of the project, or
	// whose role in it it changes.
	targets []uuid.UUID
}

// held is a write's locks taken and its decision made: the project as its
// lock read it, the caller's grant, the active roles in the workspace of
// the write's targets, by account, and, for a write on a membership, the
// membership as read under the locks.
type held struct {
	project LockedProject
	grant   shared.Grant
	roles   map[uuid.UUID]shared.Role
	member  ProjectMembership
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
	h, err := l.lock(ctx, workspaceID, w, domain.ErrNotFound)
	if err != nil {
		return held{}, err
	}
	if h.grant, err = decide(ctx, l.auth, actor, w.action, workspaceID, w.project, domain.ErrNotFound); err != nil {
		return held{}, err
	}
	return h, nil
}

// lockMemberAndDecide takes the locks of a write on the project membership
// id, a write addressed by its resource (M3 design 3.6 convention 2), and
// decides action on its project, in the order of Locks: the membership,
// read without a lock, names its project and the project's workspace; then
// their locks, with the membership's member as the write's target when
// target is set; then the membership read again, which must still be of
// that project; then the decision. A membership that is not there or is
// deleted, a workspace or project deleted while its lock waited, a
// membership no longer of the project, and a project not visible to actor
// are each domain.ErrMemberNotFound; a role the rule does not allow is the
// Authorizer's shared.Forbidden. An ended membership is the use case's to
// refuse, after the decision. A membership read for another id than asked
// is an error.
func (l Locks) lockMemberAndDecide(ctx context.Context, actor shared.Actor, id uuid.UUID, action shared.Action, target bool) (held, error) {
	m, err := l.member(ctx, id)
	if err != nil {
		return held{}, err
	}
	w := write{project: m.ProjectID, action: action}
	if target {
		w.targets = []uuid.UUID{m.MemberID}
	}
	h, err := l.lock(ctx, m.WorkspaceID, w, domain.ErrMemberNotFound)
	if err != nil {
		return held{}, err
	}
	if h.member, err = l.member(ctx, id); err != nil {
		return held{}, err
	}
	if h.member.ProjectID != m.ProjectID {
		return held{}, domain.ErrMemberNotFound
	}
	if h.grant, err = decide(ctx, l.auth, actor, action, m.WorkspaceID, m.ProjectID, domain.ErrMemberNotFound); err != nil {
		return held{}, err
	}
	return h, nil
}

// member reads the undeleted membership id: domain.ErrMemberNotFound when
// there is none, and an error when the store answers another one.
func (l Locks) member(ctx context.Context, id uuid.UUID) (ProjectMembership, error) {
	m, found, err := l.projects.MemberByID(ctx, id)
	switch {
	case err != nil:
		return ProjectMembership{}, err
	case !found:
		return ProjectMembership{}, domain.ErrMemberNotFound
	case m.ID != id:
		return ProjectMembership{}, fmt.Errorf("project membership %s read as %s", id, m.ID)
	}
	return m, nil
}

// lock takes w's locks after the read that named its project's workspace,
// workspaceID, in the order of Locks: the workspace FOR SHARE, the targets'
// memberships of it, the project. A workspace or project deleted while its
// lock waited, or a project not of workspaceID, is notFound, the 404 of
// what the caller named.
func (l Locks) lock(ctx context.Context, workspaceID uuid.UUID, w write, notFound error) (held, error) {
	_, found, err := l.workspaces.ShareWorkspaceByID(ctx, workspaceID)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, notFound
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
		return held{}, notFound
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
	_, err = decide(ctx, auth, actor, action, workspaceID, id, domain.ErrNotFound)
	return err
}

// decide asks the Authorizer for action on the project id of the workspace
// for actor. A write calls it under its locks, so the facts it reads are
// the ones committed after the locks were granted (M3 design 6.7): a
// demotion or a removal that committed while the write waited is seen; a
// read calls it after the read that names the project's workspace. A
// project not visible to actor is notFound, the 404 of what the caller
// named.
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, action shared.Action, workspaceID, id uuid.UUID,
	notFound error) (shared.Grant, error) {
	grant, err := auth.Authorize(ctx, actor, action, shared.Target{WorkspaceID: workspaceID, ProjectID: id})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return shared.Grant{}, notFound
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
