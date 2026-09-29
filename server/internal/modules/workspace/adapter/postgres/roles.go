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
