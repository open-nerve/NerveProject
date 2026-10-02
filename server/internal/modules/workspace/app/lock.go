package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// lockAndDecide is the first two steps of every write on a workspace named
// by its slug (M3 design 3.6 convention 2), in the transaction ctx carries:
// lock locks the undeleted workspace with slug, then decide. It returns the
// workspace's id and the grant. A workspace that is not there, deleted, or
// not visible to actor is domain.ErrNotFound; a role the rule does not
// allow is the Authorizer's shared.Forbidden.
func lockAndDecide(ctx context.Context, lock func(ctx context.Context, slug string) (uuid.UUID, error),
	auth shared.Authorizer, actor shared.Actor, slug string, action shared.Action) (uuid.UUID, shared.Grant, error) {
	id, err := lock(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return uuid.UUID{}, shared.Grant{}, domain.ErrNotFound
	case err != nil:
		return uuid.UUID{}, shared.Grant{}, err
	}
	grant, err := decide(ctx, auth, actor, action, id, domain.ErrNotFound)
	if err != nil {
		return uuid.UUID{}, shared.Grant{}, err
	}
	return id, grant, nil
}

// lockedMember reads the membership id, locks its workspace FOR NO KEY
// UPDATE, and reads it again under the lock, which must still be of the
// workspace locked (M3 design 3.6 convention 2): a membership or workspace
// deleted meanwhile, or a membership no longer of that workspace, is
// domain.ErrMemberNotFound.
func lockedMember(ctx context.Context, members MemberLocker, id uuid.UUID) (domain.Membership, error) {
	m, err := members.MemberByID(ctx, id)
	if err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	if err := members.LockWorkspace(ctx, m.WorkspaceID); err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	locked, err := members.MemberByID(ctx, id)
	if err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	if locked.WorkspaceID != m.WorkspaceID {
		return domain.Membership{}, domain.ErrMemberNotFound
	}
	return locked, nil
}

// memberNotFound turns ErrNotFound into domain.ErrMemberNotFound.
func memberNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return domain.ErrMemberNotFound
	}
	return err
}

// decide asks the Authorizer for action on the workspace for actor. A write
// calls it under the workspace's lock, so the role it reads is the one
// committed after the lock was granted (M3 design 6.7): a demotion or a
// removal that committed while the write waited is seen. A workspace not
// visible to actor is notFound, the 404 of what the caller named.
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, action shared.Action, workspaceID uuid.UUID,
	notFound error) (shared.Grant, error) {
	grant, err := auth.Authorize(ctx, actor, action, shared.Target{WorkspaceID: workspaceID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return shared.Grant{}, notFound
	case err != nil:
		return shared.Grant{}, err
	}
	return grant, nil
}
