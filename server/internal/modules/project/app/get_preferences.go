package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetProjectPreferences reads the caller's display settings in a project:
// GET /api/v0/me/projects/{project_id}/preferences.
type GetProjectPreferences struct {
	preferences PreferencesReader
	auth        shared.Authorizer
}

// NewGetProjectPreferences returns the use case.
func NewGetProjectPreferences(preferences PreferencesReader, auth shared.Authorizer) *GetProjectPreferences {
	return &GetProjectPreferences{preferences: preferences, auth: auth}
}

// Execute returns the caller's settings in the project, or
// domain.DefaultPreferences while he has no row of them, after the
// decision on project_preferences.read. A read opens no transaction and
// writes nothing (M3 design 3.18): a row missing is created by the first
// change.
func (u *GetProjectPreferences) Execute(ctx context.Context, projectID uuid.UUID) (domain.Preferences, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Preferences{}, err
	}
	if err := findAndDecide(ctx, u.preferences, u.auth, actor, projectID, domain.ActionPreferencesRead); err != nil {
		return domain.Preferences{}, err
	}
	p, found, err := u.preferences.Preferences(ctx, projectID, actor.UserID)
	switch {
	case err != nil:
		return domain.Preferences{}, err
	case !found:
		return domain.DefaultPreferences(), nil
	}
	return p, nil
}
