package app_test

import (
	"context"
	"encoding/json"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// fakeStore's display settings: Preferences reads a member's from his
// project, UpsertPreferences applies the patch to them, or to the defaults
// while he has none, and answers them as stored.

func (f *fakeStore) Preferences(ctx context.Context, projectID, userID uuid.UUID) (domain.Preferences, bool, error) {
	f.log.add(ctx, "Preferences %s for %s", projectID, userID)
	if err := f.fail("Preferences"); err != nil {
		return domain.Preferences{}, false, err
	}
	p, ok := f.projects[projectID].prefs[userID]
	return p, ok, nil
}

func (f *fakeStore) UpsertPreferences(ctx context.Context, c app.PreferencesChange) (domain.Preferences, error) {
	patch, _ := json.Marshal(c.Patch)
	f.log.add(ctx, "UpsertPreferences %s/%s for %s %s at %s", c.WorkspaceID, c.ProjectID, c.UserID, patch, c.Now.Format(timeFormat))
	if err := f.fail("UpsertPreferences"); err != nil {
		return domain.Preferences{}, err
	}
	project := f.projects[c.ProjectID]
	p, ok := project.prefs[c.UserID]
	if !ok {
		p = domain.DefaultPreferences()
	}
	if project.prefs == nil {
		project.prefs = map[uuid.UUID]domain.Preferences{}
	}
	project.prefs[c.UserID] = p.Apply(c.Patch)
	return project.prefs[c.UserID], nil
}
