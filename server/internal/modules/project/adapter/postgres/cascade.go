package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
)

// The steps of deleting a workspace's projects (app.WorkspaceProjectsDeleter):
// each soft-deletes the workspace's undeleted rows of one table, at now, by
// the account by, in one statement.

// DeleteWorkspaceProjects soft-deletes the workspace's projects, archived
// ones too.
func (s *Store) DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeleteWorkspaceProjects(ctx, gen.DeleteWorkspaceProjectsParams{WorkspaceID: workspaceID, DeletedBy: by, Now: now})
	if err != nil {
		return fmt.Errorf("delete the workspace's projects: %w", err)
	}
	return nil
}

// DeleteWorkspaceProjectMembers soft-deletes the memberships of the
// workspace's projects, active or not.
func (s *Store) DeleteWorkspaceProjectMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeleteWorkspaceProjectMembers(ctx, gen.DeleteWorkspaceProjectMembersParams{WorkspaceID: workspaceID, DeletedBy: by, Now: now})
	if err != nil {
		return fmt.Errorf("delete the workspace's project members: %w", err)
	}
	return nil
}

// DeleteWorkspaceProjectPreferences soft-deletes the display settings in
// the workspace's projects.
func (s *Store) DeleteWorkspaceProjectPreferences(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeleteWorkspaceProjectPreferences(ctx, gen.DeleteWorkspaceProjectPreferencesParams{
		WorkspaceID: workspaceID, DeletedBy: by, Now: now,
	})
	if err != nil {
		return fmt.Errorf("delete the workspace's project preferences: %w", err)
	}
	return nil
}

// DeleteWorkspaceStates soft-deletes the states of the workspace's
// projects, the triage states too.
func (s *Store) DeleteWorkspaceStates(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeleteWorkspaceStates(ctx, gen.DeleteWorkspaceStatesParams{WorkspaceID: workspaceID, DeletedBy: by, Now: now})
	if err != nil {
		return fmt.Errorf("delete the workspace's states: %w", err)
	}
	return nil
}

// LockMemberProjects locks FOR NO KEY UPDATE, in id order, the workspace's
// undeleted projects in which userID has an undeleted membership, active or
// not, and returns their ids (app.MemberDemoter).
func (s *Store) LockMemberProjects(ctx context.Context, workspaceID, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).LockMemberProjects(ctx, gen.LockMemberProjectsParams{WorkspaceID: workspaceID, MemberID: userID})
	if err != nil {
		return nil, fmt.Errorf("lock the member's projects: %w", err)
	}
	return ids, nil
}

// DemoteMemberships makes userID's undeleted memberships of projectIDs,
// active or not, a guest's, at now, by the account by (app.MemberDemoter).
func (s *Store) DemoteMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DemoteMemberships(ctx, gen.DemoteMembershipsParams{
		ProjectIds: projectIDs, MemberID: userID, UpdatedBy: by, Now: now,
	})
	if err != nil {
		return fmt.Errorf("demote the member's project memberships: %w", err)
	}
	return nil
}
