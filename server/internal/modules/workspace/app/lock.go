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
