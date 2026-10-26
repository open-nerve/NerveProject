package app_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type capturedSettings struct {
	calls      int
	filters    map[string][]string
	display    *app.ListDisplayFilters
	properties map[string]bool
	errors     []shared.FieldError
}

func (s *capturedSettings) Validate(f map[string][]string, d *app.ListDisplayFilters, p map[string]bool) []shared.FieldError {
	s.calls++
	s.filters, s.display, s.properties = f, d, p
	return s.errors
}

func TestIssuePreferencesValidation(t *testing.T) {
	f := newPreferences()
	settings := &capturedSettings{errors: []shared.FieldError{
		{Field: "filters.label_id[0]", Code: shared.FieldNotAllowed},
		{Field: "display_filters.layout", Code: shared.FieldNotAllowed},
	}}
	uc := app.NewUpdateProjectPreferences(f.store, f.locks(), f.tx, clockAt{clockNow, f.log}, settings)
	patch := domain.PreferencesPatch{Navigation: &domain.Navigation{DefaultTab: "bad", HideInMoreMenu: []string{"views", "views"}}, Filters: &domain.Filters{LabelID: []string{"bad"}}}
	if _, err := uc.Execute(context.Background(), webID, patch); !errors.Is(err, shared.Unauthenticated()) {
		var e *shared.Error
		if !errors.As(err, &e) || e.Code != shared.CodeUnauthorized {
			t.Fatalf("without actor: %v", err)
		}
	}
	if settings.calls != 0 || len(f.log.calls) != 0 {
		t.Fatal("validation ran without actor")
	}
	_, err := uc.Execute(as(bob), webID, patch)
	var refusal *shared.Error
	if !errors.As(err, &refusal) {
		t.Fatal(err)
	}
	got := []string{}
	for _, e := range refusal.Fields {
		got = append(got, e.Field+" "+e.Code)
	}
	want := []string{"navigation.default_tab invalid_format", "navigation.hide_in_more_menu[1] duplicate", "filters.label_id[0] not_allowed", "display_filters.layout not_allowed"}
	if !reflect.DeepEqual(got, want) || settings.calls != 1 || len(f.log.calls) != 0 {
		t.Fatalf("errors %v; calls %d/%v", got, settings.calls, f.log.calls)
	}
}

func TestIssuePreferencesValuesReachThePort(t *testing.T) {
	f := newPreferences()
	settings := &capturedSettings{}
	uc := app.NewUpdateProjectPreferences(f.store, f.locks(), f.tx, clockAt{clockNow, f.log}, settings)
	display := domain.DisplayFilters{Layout: "calendar", GroupBy: ptr("state_id"), SubGroupBy: ptr("priority"), OrderBy: "name", Calendar: domain.DisplayCalendar{Layout: "week", ShowWeekends: true}}
	filters := domain.Filters{StateID: []string{"state"}, StateGroup: []string{"started"}, Priority: []string{"low"}, LabelID: []string{}, AssigneeID: []string{"me"}, CreatedByID: []string{"me"}, MentionID: []string{"me"}, SubscriberID: []string{"me"}, StartDate: []string{"none"}, TargetDate: []string{"2026-10-26"}}
	properties := domain.DisplayProperties{Assignee: true, DueDate: true, Key: true, State: true, Link: true, UpdatedOn: true}
	patch := domain.PreferencesPatch{Filters: &filters, DisplayFilters: &display, DisplayProperties: &properties}
	got, err := uc.Execute(as(bob), webID, patch)
	wantF := map[string][]string{"state_id": {"state"}, "state_group": {"started"}, "priority": {"low"}, "label_id": {}, "assignee_id": {"me"}, "created_by_id": {"me"}, "mention_id": {"me"}, "subscriber_id": {"me"}, "start_date": {"none"}, "target_date": {"2026-10-26"}}
	wantD := &app.ListDisplayFilters{Layout: "calendar", GroupBy: ptr("state_id"), SubGroupBy: ptr("priority"), OrderBy: "name", Calendar: app.ListCalendar{Layout: "week", ShowWeekends: true}}
	wantP := map[string]bool{"assignee": true, "start_date": false, "due_date": true, "labels": false, "key": true, "priority": false, "state": true, "sub_issue_count": false, "link": true, "created_on": false, "updated_on": true}
	if err != nil || !reflect.DeepEqual(settings.filters, wantF) || !reflect.DeepEqual(settings.display, wantD) || !reflect.DeepEqual(settings.properties, wantP) || !reflect.DeepEqual(got.Filters, filters) || got.DisplayProperties != properties {
		t.Fatalf("port=%+v answer=%+v err=%v", settings, got, err)
	}
	for _, p := range []domain.PreferencesPatch{{}, {Filters: &domain.Filters{}}} {
		if _, err := uc.Execute(as(bob), webID, p); err != nil {
			t.Fatal(err)
		}
		if (settings.filters == nil) != (p.Filters == nil) || len(settings.filters) != 0 || settings.display != nil || settings.properties != nil {
			t.Fatalf("omission/clear: %+v", settings)
		}
	}
}
