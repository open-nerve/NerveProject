package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// NotificationState reads, alone and without a lock, the name and the group
// of the project's undeleted state stateID, its triage state too: what
// project.Provide offers the notification module's snapshot (M7 P1b spec
// 2.5, 2.6); found is false when there is none. It reads in the
// transaction ctx carries.
func (s *Store) NotificationState(ctx context.Context, projectID, stateID uuid.UUID) (app.NotificationState, bool, error) {
	r, err := s.queries(ctx).NotificationState(ctx, gen.NotificationStateParams{ID: stateID, ProjectID: projectID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.NotificationState{}, false, nil
	case err != nil:
		return app.NotificationState{}, false, fmt.Errorf("read state %s of project %s for a notification: %w", stateID, projectID, err)
	}
	return app.NotificationState{Name: r.Name, Group: r.Group}, true, nil
}

// NotificationIdentifier reads, alone and without a lock, the identifier
// of the workspace's undeleted project projectID, archived or not, as
// NotificationState; found is false when there is none.
func (s *Store) NotificationIdentifier(ctx context.Context, workspaceID, projectID uuid.UUID) (string, bool, error) {
	identifier, err := s.queries(ctx).NotificationIdentifier(ctx, gen.NotificationIdentifierParams{ID: projectID, WorkspaceID: workspaceID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("read the identifier of project %s of workspace %s for a notification: %w", projectID, workspaceID, err)
	}
	return identifier, true, nil
}

// NotificationIdentifiers reads, in one statement and without a lock, the
// identifiers of the undeleted projects among ids, archived or not, by
// project (notification_names.sql): what project.Provide offers the
// notification module's names of the work items a page of notifications
// names (M7 P1c spec 2.6). A deleted project and an id of none are not
// among them. It reads in the transaction ctx carries or on the pool.
func (s *Store) NotificationIdentifiers(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	rows, err := s.queries(ctx).NotificationIdentifiers(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read the identifiers of %d projects for notifications: %w", len(ids), err)
	}
	identifiers := make(map[uuid.UUID]string, len(rows))
	for _, r := range rows {
		identifiers[r.ID] = r.Identifier
	}
	return identifiers, nil
}
