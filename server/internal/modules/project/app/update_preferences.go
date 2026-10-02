package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateProjectPreferences changes the caller's display settings in a
// project: PATCH /api/v0/me/projects/{project_id}/preferences.
type UpdateProjectPreferences struct {
	preferences PreferencesWriter
	locks       Locks
	tx          shared.TxManager
	clock       Clock
}

// NewUpdateProjectPreferences returns the use case.
func NewUpdateProjectPreferences(preferences PreferencesWriter, locks Locks, tx shared.TxManager, clock Clock) *UpdateProjectPreferences {
	return &UpdateProjectPreferences{preferences: preferences, locks: locks, tx: tx, clock: clock}
}

// Execute checks p, then in one transaction (M3 design 3.6): the project's
// locks (Locks: its workspace FOR SHARE, then the project FOR SHARE) and
// the decision on project_preferences.update, the clock read under the
// locks, the caller's row changed or inserted (3.18). An archived
// project's settings change as any other's (3.19). The answer is the
// settings as stored.
func (u *UpdateProjectPreferences) Execute(ctx context.Context, projectID uuid.UUID, p domain.PreferencesPatch) (domain.Preferences, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Preferences{}, err
	}
	if err := domain.CheckPreferencesPatch(p); err != nil {
		return domain.Preferences{}, err
	}
	var stored domain.Preferences
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: projectID, action: domain.ActionPreferencesUpdate, share: true})
		if err != nil {
			return err
		}
		stored, err = u.preferences.UpsertPreferences(ctx, PreferencesChange{
			ID: uuid.NewV7(), WorkspaceID: h.project.WorkspaceID, ProjectID: projectID, UserID: actor.UserID, Patch: p, Now: u.clock.Now(),
		})
		return err
	})
	if err != nil {
		return domain.Preferences{}, err
	}
	return stored, nil
}
