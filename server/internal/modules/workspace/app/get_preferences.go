package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetWorkspacePreferences reads the caller's display settings in a
// workspace: GET /api/v0/me/workspaces/{slug}/preferences.
type GetWorkspacePreferences struct {
	preferences PreferencesReader
	auth        shared.Authorizer
}

// NewGetWorkspacePreferences returns the use case.
func NewGetWorkspacePreferences(preferences PreferencesReader, auth shared.Authorizer) *GetWorkspacePreferences {
	return &GetWorkspacePreferences{preferences: preferences, auth: auth}
}

// Execute returns the caller's settings, or domain.DefaultPreferences while
// he has changed none. A read opens no transaction and writes nothing (M3
// design 3.18): the row is created by the first change.
func (u *GetWorkspacePreferences) Execute(ctx context.Context, slug string) (domain.Preferences, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Preferences{}, err
	}
	w, err := u.preferences.WorkspaceBySlug(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return domain.Preferences{}, domain.ErrNotFound
	case err != nil:
		return domain.Preferences{}, err
	}
	_, err = u.auth.Authorize(ctx, actor, domain.ActionPreferencesRead, shared.Target{WorkspaceID: w.ID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return domain.Preferences{}, domain.ErrNotFound
	case err != nil:
		return domain.Preferences{}, err
	}
	p, found, err := u.preferences.Preferences(ctx, w.ID, actor.UserID)
	switch {
	case err != nil:
		return domain.Preferences{}, err
	case !found:
		return domain.DefaultPreferences(), nil
	}
	return p, nil
}
