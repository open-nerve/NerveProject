package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
)

// ReactivateMember makes userID's ended membership of the workspace active
// again, its role kept, at now; updated_by_id stays whose it was, as in
// Plane's command (M3 design 3.11). The caller read the membership ended
// under the workspace's lock: a pair without exactly one ended, undeleted
// membership, an active one too, is an error, not app.ErrNotFound, and
// writes nothing.
func (s *Store) ReactivateMember(ctx context.Context, workspaceID, userID uuid.UUID, now time.Time) error {
	restored, err := s.queries(ctx).ReactivateMember(ctx, gen.ReactivateMemberParams{WorkspaceID: workspaceID, MemberID: userID, Now: now})
	switch {
	case err != nil:
		return fmt.Errorf("reactivate workspace member: %w", err)
	case restored != 1:
		return fmt.Errorf("reactivate workspace member %s of %s: %d ended undeleted memberships", userID, workspaceID, restored)
	}
	return nil
}
