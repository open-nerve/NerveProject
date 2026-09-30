package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// prefsKey is one (workspace, user) pair of fakeWorkspaces' preferences.
type prefsKey struct{ workspace, user uuid.UUID }

func (f *fakeWorkspaces) Preferences(ctx context.Context, workspaceID, userID uuid.UUID) (domain.Preferences, bool, error) {
	f.log.add(ctx, "Preferences %s %s", workspaceID, userID)
	if f.prefsErr != nil {
		return domain.Preferences{}, false, fmt.Errorf("read workspace preferences: %w", f.prefsErr)
	}
	p, found := f.prefs[prefsKey{workspaceID, userID}]
	return p, found, nil
}

// UpsertPreferences logs the row without its id, which the use case makes
// anew each time; upserts keeps it whole. A row it inserts takes r's id and
// keeps it through later changes, as the store's does (prefIDs).
func (f *fakeWorkspaces) UpsertPreferences(ctx context.Context, r app.PreferencesRow) (domain.Preferences, error) {
	var limit *string
	if r.Patch.NavigationProjectLimit != nil {
		limit = ptr(fmt.Sprint(*r.Patch.NavigationProjectLimit))
	}
	f.log.add(ctx, "UpsertPreferences %s %s mode=%s limit=%s at %s", r.WorkspaceID, r.UserID, show(r.Patch.NavigationControl), show(limit),
		r.Now.Format(time.RFC3339Nano))
	f.upserts = append(f.upserts, r)
	if f.prefsErr != nil {
		return domain.Preferences{}, fmt.Errorf("write workspace preferences: %w", f.prefsErr)
	}
	key := prefsKey{r.WorkspaceID, r.UserID}
	p, found := f.prefs[key]
	if !found {
		p = domain.DefaultPreferences()
		if f.prefIDs == nil {
			f.prefIDs = map[prefsKey]uuid.UUID{}
		}
		f.prefIDs[key] = r.ID
	}
	if f.prefs == nil {
		f.prefs = map[prefsKey]domain.Preferences{}
	}
	f.prefs[key] = p.Apply(r.Patch)
	return f.prefs[key], nil
}
