package postgresadapter

import (
	"context"
	"errors"
	"fmt"
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
	return app.IssueProject{WorkspaceID: r.WorkspaceID, Archived: r.Archived, DefaultAssigneeID: r.DefaultAssigneeID}, true, nil
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
