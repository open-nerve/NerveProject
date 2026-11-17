package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ActiveRole is access's WorkspaceRoles (M3 design 6.5): userID's role in
// workspaceID when the membership is active, its row not deleted and the
// workspace not deleted; ok is false otherwise. It reads in the transaction
// ctx carries.
func (s *Store) ActiveRole(ctx context.Context, workspaceID, userID uuid.UUID) (shared.Role, bool, error) {
	role, err := s.queries(ctx).ActiveRole(ctx, gen.ActiveRoleParams{WorkspaceID: workspaceID, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return 0, false, nil
	case err != nil:
		return 0, false, fmt.Errorf("read the workspace role: %w", err)
	}
	return shared.Role(role), true, nil
}

// ActiveRoles is ActiveRole for each of ids at once, access's
// WorkspaceRolesMany (M7 P1b spec 2.7): the roles of those whose membership
// of workspaceID is active, by account; one without is not in it. It reads
// in the transaction ctx carries.
func (s *Store) ActiveRoles(ctx context.Context, workspaceID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	rows, err := s.queries(ctx).ActiveRoles(ctx, gen.ActiveRolesParams{WorkspaceID: workspaceID, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("read the workspace roles of %d accounts: %w", len(ids), err)
	}
	roles := make(map[uuid.UUID]shared.Role, len(rows))
	for _, r := range rows {
		roles[r.MemberID] = shared.Role(r.Role)
	}
	return roles, nil
}
