package postgresadapter

import (
	"context"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
)

// InsertModuleMembers writes the module's new members' rows r, in one
// statement, in their order (module_members.sql).
func (s *Store) InsertModuleMembers(ctx context.Context, r app.MemberRows) error {
	err := s.queries(ctx).InsertModuleMembers(ctx, gen.InsertModuleMembersParams{Ids: r.IDs, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID,
		ModuleID: r.ModuleID, MemberIds: r.Accounts, By: &r.By, At: r.At})
	if err != nil {
		return fmt.Errorf("insert the members of module %s: %w", r.ModuleID, err)
	}
	return nil
}
