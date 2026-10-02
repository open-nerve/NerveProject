package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// The steps of deleting projects (app.ProjectsDeleter): each soft-deletes
// the undeleted rows of one table under the workspace's projects, or under
// the one project the deletion names, at its moment, by its account, in one
// statement.

// DeleteProjects soft-deletes the projects, archived ones too.
func (s *Store) DeleteProjects(ctx context.Context, d app.Deletion) error {
	err := s.queries(ctx).DeleteProjects(ctx, gen.DeleteProjectsParams{WorkspaceID: d.WorkspaceID, ProjectID: d.ProjectID, DeletedBy: d.By, Now: d.Now})
	if err != nil {
		return fmt.Errorf("delete the projects: %w", err)
	}
	return nil
}

// DeleteProjectMembers soft-deletes the projects' memberships, active or
// not.
func (s *Store) DeleteProjectMembers(ctx context.Context, d app.Deletion) error {
	err := s.queries(ctx).DeleteProjectMembers(ctx, gen.DeleteProjectMembersParams{
		WorkspaceID: d.WorkspaceID, ProjectID: d.ProjectID, DeletedBy: d.By, Now: d.Now,
	})
	if err != nil {
		return fmt.Errorf("delete the project members: %w", err)
	}
	return nil
}

// DeleteProjectPreferences soft-deletes the display settings in the
// projects.
func (s *Store) DeleteProjectPreferences(ctx context.Context, d app.Deletion) error {
	err := s.queries(ctx).DeleteProjectPreferences(ctx, gen.DeleteProjectPreferencesParams{
		WorkspaceID: d.WorkspaceID, ProjectID: d.ProjectID, DeletedBy: d.By, Now: d.Now,
	})
	if err != nil {
		return fmt.Errorf("delete the project preferences: %w", err)
	}
	return nil
}

// DeleteStates soft-deletes the projects' states, the triage states too.
func (s *Store) DeleteStates(ctx context.Context, d app.Deletion) error {
	err := s.queries(ctx).DeleteStates(ctx, gen.DeleteStatesParams{WorkspaceID: d.WorkspaceID, ProjectID: d.ProjectID, DeletedBy: d.By, Now: d.Now})
	if err != nil {
		return fmt.Errorf("delete the states: %w", err)
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

// LockActiveMemberProjects locks FOR NO KEY UPDATE, in id order, the
// workspaces' undeleted projects in which userID has an active membership,
// and returns their ids (app.MembershipEnder).
func (s *Store) LockActiveMemberProjects(ctx context.Context, workspaceIDs []uuid.UUID, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).LockActiveMemberProjects(ctx, gen.LockActiveMemberProjectsParams{WorkspaceIds: workspaceIDs, MemberID: userID})
	if err != nil {
		return nil, fmt.Errorf("lock the member's active projects: %w", err)
	}
	return ids, nil
}

// SoleAdmin reports whether userID is the only active admin of one of
// projectIDs that has another active member (app.MembershipEnder).
func (s *Store) SoleAdmin(ctx context.Context, projectIDs []uuid.UUID, userID uuid.UUID) (bool, error) {
	sole, err := s.queries(ctx).SoleAdmin(ctx, gen.SoleAdminParams{ProjectIds: projectIDs, MemberID: userID})
	if err != nil {
		return false, fmt.Errorf("look for a project he is the only admin of: %w", err)
	}
	return sole, nil
}

// EndMemberships ends userID's active memberships of projectIDs, at now, by
// the account by (app.MembershipEnder).
func (s *Store) EndMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).EndMemberships(ctx, gen.EndMembershipsParams{ProjectIds: projectIDs, MemberID: userID, EndedBy: by, Now: now})
	if err != nil {
		return fmt.Errorf("end the member's project memberships: %w", err)
	}
	return nil
}
