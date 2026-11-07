package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

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

// RemoveModuleMembers soft-deletes the module's undeleted rows of the
// accounts, deleted and last changed by by at at, in one statement
// (module_members.sql); a row deleted before keeps its deletion.
func (s *Store) RemoveModuleMembers(ctx context.Context, module uuid.UUID, accounts []uuid.UUID, at time.Time, by uuid.UUID) error {
	err := s.queries(ctx).RemoveModuleMembers(ctx, gen.RemoveModuleMembersParams{ModuleID: module, MemberIds: accounts, At: at, By: by})
	if err != nil {
		return fmt.Errorf("remove members of module %s: %w", module, err)
	}
	return nil
}

// DeleteModuleMemberRows soft-deletes the deleted module's undeleted
// members' rows, locked in id order, deleted and last changed by by at at,
// in one statement (module_members.sql).
func (s *Store) DeleteModuleMemberRows(ctx context.Context, module uuid.UUID, at time.Time, by uuid.UUID) error {
	err := s.queries(ctx).DeleteModuleMemberRows(ctx, gen.DeleteModuleMemberRowsParams{ModuleID: module, At: at, By: by})
	if err != nil {
		return fmt.Errorf("delete the members of module %s: %w", module, err)
	}
	return nil
}
