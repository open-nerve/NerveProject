package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// LockIssueProject locks the undeleted project id FOR NO KEY UPDATE, its
// workspace not locked, and answers it: the first lock of a write on the
// work-item side (M4 design 3.5, 6.5); found is false for a project that
// is not there or is deleted, also when it was deleted while the lock
// waited. It locks in the transaction ctx carries.
func (s *Store) LockIssueProject(ctx context.Context, id uuid.UUID) (p app.IssueProject, found bool, err error) {
	r, err := s.queries(ctx).LockIssueProject(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.IssueProject{}, false, nil
	case err != nil:
		return app.IssueProject{}, false, fmt.Errorf("lock project %s for its work items: %w", id, err)
	}
	return app.IssueProject{WorkspaceID: r.WorkspaceID, Archived: r.Archived, DefaultAssigneeID: r.DefaultAssigneeID, Identifier: r.Identifier, ArchiveIn: int(r.ArchiveIn)},
		true, nil
}

// LockIssueProjects locks the undeleted projects among ids as
// LockIssueProject does, in one statement, in id order, and answers them
// by id: the first locks of a write across projects (M4 design 3.5,
// convention 7). One that is not there or is deleted, also one deleted
// while the lock waited, is not among them. It locks in the transaction
// ctx carries.
func (s *Store) LockIssueProjects(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]app.IssueProject, error) {
	rows, err := s.queries(ctx).LockIssueProjects(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("lock %d projects for their work items: %w", len(ids), err)
	}
	locked := make(map[uuid.UUID]app.IssueProject, len(rows))
	for _, r := range rows {
		locked[r.ID] = app.IssueProject{WorkspaceID: r.WorkspaceID, Archived: r.Archived, DefaultAssigneeID: r.DefaultAssigneeID,
			Identifier: r.Identifier, ArchiveIn: int(r.ArchiveIn)}
	}
	return locked, nil
}

// NextIssueSequence takes the undeleted project id's next work-item number
// (M4 design 3.7), under its lock: a project that is not there is an
// error, never a number.
func (s *Store) NextIssueSequence(ctx context.Context, id uuid.UUID) (int, error) {
	n, err := s.queries(ctx).NextIssueSequence(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("take the next work-item number of project %s: %w", id, err)
	}
	return int(n), nil
}

// IssueStates reads the undeleted states of the project, its triage state
// too, in id order, with their names (M4 design 6.5).
func (s *Store) IssueStates(ctx context.Context, projectID uuid.UUID) ([]app.IssueState, error) {
	rows, err := s.queries(ctx).IssueStates(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("read the states of project %s: %w", projectID, err)
	}
	out := make([]app.IssueState, len(rows))
	for i, r := range rows {
		out[i] = app.IssueState{ID: r.ID, Name: r.Name, Group: r.Group, Default: r.Default}
	}
	return out, nil
}

// IssueLabels reads those of ids that are undeleted labels of the project,
// in id order, with their names (M4 design 6.5).
func (s *Store) IssueLabels(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]app.IssueLabel, error) {
	rows, err := s.queries(ctx).IssueLabels(ctx, gen.IssueLabelsParams{ProjectID: projectID, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("read labels of project %s: %w", projectID, err)
	}
	out := make([]app.IssueLabel, len(rows))
	for i, r := range rows {
		out[i] = app.IssueLabel{ID: r.ID, Name: r.Name}
	}
	return out, nil
}

// ProjectIdentifiers reads the workspace's undeleted projects, archived or
// not, each with its identifier, in id order, without a lock (M4 design
// 3.7).
func (s *Store) ProjectIdentifiers(ctx context.Context, workspaceID uuid.UUID) ([]app.IssueProjectIdentifier, error) {
	rows, err := s.queries(ctx).ProjectIdentifiers(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("read the identifiers of the projects of workspace %s: %w", workspaceID, err)
	}
	out := make([]app.IssueProjectIdentifier, len(rows))
	for i, r := range rows {
		out[i] = app.IssueProjectIdentifier{ID: r.ID, Identifier: r.Identifier}
	}
	return out, nil
}

// ActiveRoles reads the project roles of those of the accounts ids that
// are active members of the project, by account (M4 design 6.5).
func (s *Store) ActiveRoles(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	rows, err := s.queries(ctx).ActiveRoles(ctx, gen.ActiveRolesParams{ProjectID: projectID, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("read the roles of members of project %s: %w", projectID, err)
	}
	roles := make(map[uuid.UUID]shared.Role, len(rows))
	for _, r := range rows {
		roles[r.MemberID] = shared.Role(r.Role)
	}
	return roles, nil
}

// IssueMentionMembers reads active candidates with their guest visibility.
func (s *Store) IssueMentionMembers(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]app.IssueMemberVisibility, error) {
	rows, err := s.queries(ctx).IssueMentionMembers(ctx, gen.IssueMentionMembersParams{ProjectID: projectID, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("read mention members of project %s: %w", projectID, err)
	}
	out := make([]app.IssueMemberVisibility, len(rows))
	for n, r := range rows {
		out[n] = app.IssueMemberVisibility{ID: r.MemberID, Role: shared.Role(r.Role), OwnOnly: r.OwnOnly}
	}
	return out, nil
}

// VisibleIssueProjects reads ordinary membership scope without widening it
// through workspace roles or public project visibility.
func (s *Store) VisibleIssueProjects(ctx context.Context, workspaceID, actorID uuid.UUID) ([]app.VisibleIssueProject, error) {
	rows, err := s.queries(ctx).VisibleIssueProjects(ctx, gen.VisibleIssueProjectsParams{WorkspaceID: workspaceID, ActorID: actorID})
	if err != nil {
		return nil, err
	}
	out := make([]app.VisibleIssueProject, len(rows))
	for i, r := range rows {
		out[i] = app.VisibleIssueProject{ID: r.ID, Role: shared.Role(r.Role), OwnOnly: r.OwnOnly}
	}
	return out, nil
}

// IssueListStates reads the selected projects' complete state catalogs for
// lists. Archived projects remain eligible; triage states do not.
func (s *Store) IssueListStates(ctx context.Context, projectIDs []uuid.UUID) ([]app.IssueState, error) {
	rows, err := s.queries(ctx).IssueListStates(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	out := make([]app.IssueState, len(rows))
	for i, r := range rows {
		out[i] = app.IssueState{ID: r.ID, Name: r.Name, Group: r.Group, Default: r.Default}
	}
	return out, nil
}

// IssueListLabels reads the selected projects' complete label catalogs for
// lists, including labels unused by any work item.
func (s *Store) IssueListLabels(ctx context.Context, projectIDs []uuid.UUID) ([]app.IssueLabel, error) {
	rows, err := s.queries(ctx).IssueListLabels(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	out := make([]app.IssueLabel, len(rows))
	for i, r := range rows {
		out[i] = app.IssueLabel{ID: r.ID, Name: r.Name}
	}
	return out, nil
}

// UndeletedIssueProjectIDs includes archived projects regardless of membership.
func (s *Store) UndeletedIssueProjectIDs(ctx context.Context, wid uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).UndeletedIssueProjectIDs(ctx, wid)
	if err != nil {
		return nil, fmt.Errorf("read undeleted projects in workspace %s: %w", wid, err)
	}
	return ids, nil
}

// ShareIssueProjects acquires the requested undeleted rows in one ascending
// SHARE statement, without locking the workspace or taking an issue write gate.
func (s *Store) ShareIssueProjects(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	rows, err := s.queries(ctx).ShareIssueProjects(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("share %d projects for drafts: %w", len(ids), err)
	}
	out := make(map[uuid.UUID]uuid.UUID, len(rows))
	for _, r := range rows {
		out[r.ID] = r.WorkspaceID
	}
	return out, nil
}

// AllUndeletedIssueProjectIDs includes archived projects, without policy or membership filters.
func (s *Store) AllUndeletedIssueProjectIDs(ctx context.Context) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).AllUndeletedIssueProjectIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("read all undeleted projects for issue retention: %w", err)
	}
	return ids, nil
}

// DeletedIssueProjects returns only stored soft-deletion times, without locks.
func (s *Store) DeletedIssueProjects(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	out := make(map[uuid.UUID]time.Time)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.queries(ctx).DeletedIssueProjects(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = *row.DeletedAt
	}
	return out, nil
}
