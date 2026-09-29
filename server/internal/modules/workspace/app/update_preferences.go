package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateWorkspacePreferences changes the caller's display settings in a
// workspace: PATCH /api/v0/me/workspaces/{slug}/preferences.
type UpdateWorkspacePreferences struct {
	preferences PreferencesWriter
	auth        shared.Authorizer
	tx          shared.TxManager
	clock       Clock
}

// NewUpdateWorkspacePreferences returns the use case.
func NewUpdateWorkspacePreferences(preferences PreferencesWriter, auth shared.Authorizer, tx shared.TxManager, clock Clock) *UpdateWorkspacePreferences {
	return &UpdateWorkspacePreferences{preferences: preferences, auth: auth, tx: tx, clock: clock}
}

// Execute checks p, then in one transaction (M3 design 3.6): the workspace
// row FOR SHARE, the decision, the caller's row changed or inserted
// (M3 design 3.18).
func (u *UpdateWorkspacePreferences) Execute(ctx context.Context, slug string, p domain.PreferencesPatch) (domain.Preferences, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Preferences{}, err
	}
	if err := domain.CheckPreferencesPatch(p); err != nil {
		return domain.Preferences{}, err
	}
	now := u.clock.Now()
	var stored domain.Preferences
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		id, _, err := lockAndDecide(ctx, u.preferences.ShareWorkspaceBySlug, u.auth, actor, slug, domain.ActionPreferencesUpdate)
		if err != nil {
			return err
		}
		stored, err = u.preferences.UpsertPreferences(ctx, PreferencesRow{
			ID: uuid.NewV7(), WorkspaceID: id, UserID: actor.UserID, Patch: p, Now: now,
		})
		return err
	})
	if err != nil {
		return domain.Preferences{}, err
	}
	return stored, nil
}
