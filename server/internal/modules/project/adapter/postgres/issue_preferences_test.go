package postgresadapter_test

import (
	"context"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// Every flag keeps its neighbors, including when the first PATCH inserts.
// Empty arrays and false flags survive JSON; filters replace rather than merge.
func TestProjectIssuePreferencesStore(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "settings-a@example.com"), newAccount(t, pool, "settings-b@example.com")
	acme := newWorkspace(t, pool, "settings")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	preferencesRow(t, pool, acme, web, alice)
	preferencesRow(t, pool, acme, ops, bob)
	deleted := preferencesRow(t, pool, acme, web, bob)
	exec(t, pool, "UPDATE project_user_properties SET deleted_at=$2 WHERE id=$1", deleted, now)
	before := tableRows(t, pool, "project_user_properties", uuid.Nil())
	want := domain.Preferences{Navigation: domain.Navigation{DefaultTab: "work_items", HideInMoreMenu: []string{}}, SortOrder: 65535,
		DisplayFilters:    domain.DisplayFilters{Layout: "list", OrderBy: "-created_at", SubIssue: true, ShowEmptyGroups: true, Calendar: domain.DisplayCalendar{Layout: "month"}},
		DisplayProperties: domain.DisplayProperties{Assignee: true, StartDate: true, DueDate: true, Labels: true, Key: true, Priority: true, State: true, SubIssueCount: true, Link: true, CreatedOn: true, UpdatedOn: true}}
	defaults := want
	filters := domain.Filters{LabelID: []string{"none"}, Priority: []string{"high"}, AssigneeID: []string{"me"}, StartDate: []string{"2026-10-01.."}}
	display := domain.DisplayFilters{Layout: "kanban", OrderBy: "sort_order", Calendar: domain.DisplayCalendar{Layout: "week", ShowWeekends: true}}
	properties := domain.DisplayProperties{Key: true}
	navigation := domain.Navigation{DefaultTab: "views", HideInMoreMenu: []string{"cycles"}}
	order := 4.5
	id := uuid.NewV7()
	for i, patch := range []domain.PreferencesPatch{
		{Filters: &filters}, {DisplayFilters: &display}, {DisplayProperties: &properties}, {Navigation: &navigation}, {SortOrder: &order},
		{Filters: &domain.Filters{LabelID: []string{}}}, {Filters: &domain.Filters{}}, {},
	} {
		switch i {
		case 0:
			want.Filters = filters
		case 1:
			want.DisplayFilters = display
		case 2:
			want.DisplayProperties = properties
		case 3:
			want.Navigation = navigation
		case 4:
			want.SortOrder = order
		case 5:
			want.Filters = domain.Filters{LabelID: []string{}}
		case 6:
			want.Filters = domain.Filters{}
		}
		got, err := s.UpsertPreferences(ctx, app.PreferencesChange{ID: id, WorkspaceID: acme, ProjectID: web, UserID: bob, Patch: patch, Now: now.Add(time.Duration(i) * time.Minute)})
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("patch %d: %+v, %v; want %+v", i, got, err, want)
		}
		read, found, err := s.Preferences(ctx, web, bob)
		if err != nil || !found || !reflect.DeepEqual(read, want) {
			t.Fatalf("read %d: %+v, %v, %v", i, read, found, err)
		}
		if after := tableRows(t, pool, "project_user_properties", id); after != before {
			t.Fatalf("patch %d changed another row", i)
		}
		if i == 5 {
			var equal bool
			if err := pool.QueryRow(ctx, `SELECT filters='{"label_id":[]}'::jsonb FROM project_user_properties WHERE id=$1`, id).Scan(&equal); err != nil || !equal {
				t.Fatalf("empty array lost: %v, %v", equal, err)
			}
		}
	}
	// Preserve a nonempty filters column too when a membership is restored.
	if _, err := s.UpsertPreferences(ctx, app.PreferencesChange{ID: id, WorkspaceID: acme, ProjectID: web, UserID: bob, Patch: domain.PreferencesPatch{Filters: &filters}, Now: now}); err != nil {
		t.Fatal(err)
	}
	all := tableRows(t, pool, "project_user_properties", uuid.Nil())
	if err := s.EnsurePreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: bob, SortOrder: -10, CreatedBy: alice, Now: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if after := tableRows(t, pool, "project_user_properties", uuid.Nil()); after != all {
		t.Fatal("Ensure overwrote existing values or audit")
	}
	// Each new top-level field must also be honored on the INSERT branch.
	for i, patch := range []domain.PreferencesPatch{{Filters: &filters}, {DisplayFilters: &display}, {DisplayProperties: &properties}} {
		user := newAccount(t, pool, []string{"first-f@example.com", "first-d@example.com", "first-p@example.com"}[i])
		expected := defaults
		switch i {
		case 0:
			expected.Filters = filters
		case 1:
			expected.DisplayFilters = display
		case 2:
			expected.DisplayProperties = properties
		}
		got, err := s.UpsertPreferences(ctx, app.PreferencesChange{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, UserID: user, Patch: patch, Now: now})
		if err != nil || !reflect.DeepEqual(got, expected) {
			t.Fatalf("first field %d: %+v, %v; want %+v", i, got, err, expected)
		}
	}
}
