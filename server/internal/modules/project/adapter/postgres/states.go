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
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// stateOf is a state row as the domain has it. The queries that read a
// state each read the same columns, so their rows convert to
// gen.StateByIDRow.
func stateOf(r gen.StateByIDRow) domain.State {
	return domain.State{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, Name: r.Name, Description: r.Description, Color: r.Color,
		Group: domain.StateGroup(r.Group), Default: r.Default, Sequence: r.Sequence, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

// stateTaken is err, a write's of a state, as a name another undeleted
// state of the project has: domain.ErrStateNameTaken; any other error is
// the write's.
func stateTaken(write string, err error) error {
	if postgres.UniqueViolation(err, "states_project_id_name_key") {
		return domain.ErrStateNameTaken
	}
	return fmt.Errorf("%s: %w", write, err)
}

// CreateState inserts r and returns it as stored (app.StateCreator). A
// name another undeleted state of the project has is
// domain.ErrStateNameTaken. The domain checked every value, so a CHECK
// violation is a bug: an internal error, not a domain error.
func (s *Store) CreateState(ctx context.Context, r app.StateRow) (domain.State, error) {
	row, err := s.queries(ctx).CreateState(ctx, gen.CreateStateParams{
		ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, Name: r.State.Name, Description: r.State.Description, Color: r.State.Color,
		Sequence: r.State.Sequence, StateGroup: string(r.State.Group), IsDefault: r.State.Default, CreatedBy: &r.CreatedBy, Now: r.Now,
	})
	if err != nil {
		return domain.State{}, stateTaken(fmt.Sprintf("create state %q", r.State.Name), err)
	}
	return stateOf(gen.StateByIDRow(row)), nil
}

// StateByID is the undeleted state id, unless it is the triage state;
// found is false when there is none (app.StateFinder).
func (s *Store) StateByID(ctx context.Context, id uuid.UUID) (st domain.State, found bool, err error) {
	r, err := s.queries(ctx).StateByID(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.State{}, false, nil
	case err != nil:
		return domain.State{}, false, fmt.Errorf("read state %s: %w", id, err)
	}
	return stateOf(r), true, nil
}

// ListStates lists projectID's undeleted states but its triage state, by
// sequence, then id; none while the project is archived (app.StateLister).
func (s *Store) ListStates(ctx context.Context, projectID uuid.UUID) ([]domain.State, error) {
	rows, err := s.queries(ctx).ListStates(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list states of project %s: %w", projectID, err)
	}
	out := make([]domain.State, len(rows))
	for i, r := range rows {
		out[i] = stateOf(gen.StateByIDRow(r))
	}
	return out, nil
}

// GreatestSequence is the greatest sequence of projectID's undeleted
// states but its triage state, nil when it has none (app.StateCreator).
func (s *Store) GreatestSequence(ctx context.Context, projectID uuid.UUID) (*float64, error) {
	greatest, err := s.queries(ctx).GreatestSequence(ctx, projectID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read the greatest sequence of project %s: %w", projectID, err)
	}
	return &greatest, nil
}

// UpdateState changes the fields p gives of the undeleted state id, by the
// account by at now, and returns it as stored (app.StateUpdater). A name
// another undeleted state of the project has is domain.ErrStateNameTaken; a
// deleted state and the triage state are an error, not written.
func (s *Store) UpdateState(ctx context.Context, id uuid.UUID, p domain.StatePatch, by uuid.UUID, now time.Time) (domain.State, error) {
	arg := gen.UpdateStateParams{ID: id, Name: p.Name, Description: p.Description, Color: p.Color, Sequence: p.Sequence, UpdatedBy: by, Now: now}
	if p.Group != nil {
		group := string(*p.Group)
		arg.StateGroup = &group
	}
	r, err := s.queries(ctx).UpdateState(ctx, arg)
	if err != nil {
		return domain.State{}, stateTaken(fmt.Sprintf("update state %s", id), err)
	}
	return stateOf(gen.StateByIDRow(r)), nil
}

// CountGroupStates is the number of projectID's undeleted states in group
// (app.GroupCounter).
func (s *Store) CountGroupStates(ctx context.Context, projectID uuid.UUID, group domain.StateGroup) (int, error) {
	n, err := s.queries(ctx).CountGroupStates(ctx, gen.CountGroupStatesParams{ProjectID: projectID, StateGroup: string(group)})
	if err != nil {
		return 0, fmt.Errorf("count the %s states of project %s: %w", group, projectID, err)
	}
	return int(n), nil
}

// DeleteState deletes the undeleted state id, unless it is its project's
// default or its triage state, by the account by at now; deleted is false
// when it did not (app.StateDeleter).
func (s *Store) DeleteState(ctx context.Context, id, by uuid.UUID, now time.Time) (deleted bool, err error) {
	n, err := s.queries(ctx).DeleteState(ctx, gen.DeleteStateParams{ID: id, DeletedBy: by, Now: now})
	if err != nil {
		return false, fmt.Errorf("delete state %s: %w", id, err)
	}
	return n == 1, nil
}

// MarkDefaultState makes the undeleted state id of projectID the project's
// default, by the account by at now, in two statements: the project's
// default state the default no longer, then this one the default
// (app.DefaultMarker). marked is false when the second wrote no row, the
// state deleted, of another project or the triage state, and the caller
// rolls the first back.
func (s *Store) MarkDefaultState(ctx context.Context, projectID, id, by uuid.UUID, now time.Time) (marked bool, err error) {
	q := s.queries(ctx)
	if err := q.ClearDefaultState(ctx, gen.ClearDefaultStateParams{ProjectID: projectID, UpdatedBy: by, Now: now}); err != nil {
		return false, fmt.Errorf("clear the default state of project %s: %w", projectID, err)
	}
	n, err := q.SetDefaultState(ctx, gen.SetDefaultStateParams{ProjectID: projectID, ID: id, UpdatedBy: by, Now: now})
	if err != nil {
		return false, fmt.Errorf("make state %s the default: %w", id, err)
	}
	return n == 1, nil
}

// ListWorkspaceStates lists the undeleted states but the triage states of
// workspaceID's undeleted, unarchived projects that userID is an active
// member of, by project id, then sequence, then id
// (app.WorkspaceStateLister).
func (s *Store) ListWorkspaceStates(ctx context.Context, workspaceID, userID uuid.UUID) ([]domain.State, error) {
	rows, err := s.queries(ctx).ListWorkspaceStates(ctx, gen.ListWorkspaceStatesParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("list the states of workspace %s: %w", workspaceID, err)
	}
	out := make([]domain.State, len(rows))
	for i, r := range rows {
		out[i] = stateOf(gen.StateByIDRow(r))
	}
	return out, nil
}
