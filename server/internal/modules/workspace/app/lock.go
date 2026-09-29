package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// lockAndDecide is the first two steps of every write on a workspace (M3
// design 3.6 convention 2), in the transaction ctx carries: lock locks the
// undeleted workspace with slug, then the Authorizer decides action on it
// for actor and reads the role committed before the lock was granted. It
// returns the workspace's id and the grant. A workspace that is not there,
// deleted, or not visible to actor is domain.ErrNotFound; a role the rule
// does not allow is the Authorizer's shared.Forbidden.
func lockAndDecide(ctx context.Context, lock func(ctx context.Context, slug string) (uuid.UUID, error),
	auth shared.Authorizer, actor shared.Actor, slug string, action shared.Action) (uuid.UUID, shared.Grant, error) {
	id, err := lock(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return uuid.UUID{}, shared.Grant{}, domain.ErrNotFound
	case err != nil:
		return uuid.UUID{}, shared.Grant{}, err
	}
	grant, err := auth.Authorize(ctx, actor, action, shared.Target{WorkspaceID: id})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return uuid.UUID{}, shared.Grant{}, domain.ErrNotFound
	case err != nil:
		return uuid.UUID{}, shared.Grant{}, err
	}
	return id, grant, nil
}
