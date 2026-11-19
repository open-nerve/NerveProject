package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A content edit's body (M7 P2b spec 2.4) is the patch of the same fields
// updateIssue's would be, checked by its rules: each field it gives, and
// none it does not; the name trimmed; a problem in the same 422.
func TestCheckIntakeIssueUpdate(t *testing.T) {
	name, html, skip, high := "  Fix login  ", "<p>Steps</p>", true, Priority("high")
	ids := []uuid.UUID{uuid.NewV7()}
	day, _ := shared.ParseDate("2026-12-01")
	trimmed := "Fix login"
	for _, tc := range []struct {
		u    IntakeIssueUpdate
		want IssuePatch
	}{
		{IntakeIssueUpdate{}, IssuePatch{}},
		{IntakeIssueUpdate{Name: &name, Priority: &high, AssigneeIDs: &ids, LabelIDs: &ids, SetTargetDate: true, TargetDate: &day},
			IssuePatch{Name: &trimmed, Priority: &high, AssigneeIDs: &ids, LabelIDs: &ids, SetTargetDate: true, TargetDate: &day}},
		{IntakeIssueUpdate{DescriptionHTML: &html, SkipActivity: &skip}, IssuePatch{DescriptionHTML: &html, SkipActivity: &skip}},
		{IntakeIssueUpdate{SetTargetDate: true}, IssuePatch{SetTargetDate: true}},
	} {
		got, err := CheckIntakeIssueUpdate(tc.u)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("CheckIntakeIssueUpdate(%+v) = %+v, %v; want %+v", tc.u, got, err, tc.want)
		}
	}
	blank, bad, many := "  ", Priority("huge"), make([]uuid.UUID, MaxLabels+1)
	for n := range many {
		many[n] = uuid.NewV7()
	}
	for _, tc := range []struct {
		u      IntakeIssueUpdate
		fields []string
	}{
		{IntakeIssueUpdate{Name: &blank}, []string{"name"}},
		{IntakeIssueUpdate{Priority: &bad, LabelIDs: &many}, []string{"priority", "label_ids"}},
		{IntakeIssueUpdate{Name: &name, DescriptionHTML: &html, SkipActivity: &skip}, []string{"skip_activity"}},
		{IntakeIssueUpdate{SkipActivity: &skip}, []string{"skip_activity"}},
	} {
		_, err := CheckIntakeIssueUpdate(tc.u)
		var problem *shared.Error
		if !errors.As(err, &problem) || problem.Kind != shared.KindInvalid || len(problem.Fields) != len(tc.fields) {
			t.Fatalf("CheckIntakeIssueUpdate(%+v) = %v; want 422 on %v", tc.u, err, tc.fields)
		}
		for n, f := range problem.Fields {
			if !strings.HasPrefix(f.Field, tc.fields[n]) {
				t.Errorf("CheckIntakeIssueUpdate(%+v) problem %d on %s; want %s", tc.u, n, f.Field, tc.fields[n])
			}
		}
	}
}

// A guest who submitted a work item changes its name and its description
// alone (spec 2.4, R4): an edit that gives nothing else is content only,
// skip_activity, a switch of the description's save, and none at all
// included; one that gives any other field is not, a target date cleared
// (given as null) too.
func TestIntakeIssueUpdateContentOnly(t *testing.T) {
	name, html, skip, low, ids := "Name", "<p>x</p>", true, Priority("low"), []uuid.UUID{}
	day, _ := shared.ParseDate("2026-12-01")
	for _, tc := range []struct {
		u    IntakeIssueUpdate
		only bool
	}{
		{IntakeIssueUpdate{}, true},
		{IntakeIssueUpdate{Name: &name}, true},
		{IntakeIssueUpdate{DescriptionHTML: &html}, true},
		{IntakeIssueUpdate{DescriptionHTML: &html, SkipActivity: &skip}, true},
		{IntakeIssueUpdate{Name: &name, DescriptionHTML: &html}, true},
		{IntakeIssueUpdate{Name: &name, Priority: &low}, false},
		{IntakeIssueUpdate{AssigneeIDs: &ids}, false},
		{IntakeIssueUpdate{LabelIDs: &ids}, false},
		{IntakeIssueUpdate{SetTargetDate: true, TargetDate: &day}, false},
		{IntakeIssueUpdate{SetTargetDate: true}, false},
	} {
		if got := tc.u.ContentOnly(); got != tc.only {
			t.Errorf("(%+v).ContentOnly() = %v, want %v", tc.u, got, tc.only)
		}
	}
}

// An intake work item is open to a content edit while its row is stored
// pending or snoozed (spec 2.4, DP-7): accepted, declined and duplicate
// are closed, 409 issue.intake_closed. The stored status alone decides,
// no moment: a snooze open whether its time has come or not.
func TestIntakeClosedStatuses(t *testing.T) {
	if ErrIntakeClosed.Code != "issue.intake_closed" || ErrIntakeClosed.ProblemStatus() != 409 {
		t.Fatalf("ErrIntakeClosed = %+v; want 409 issue.intake_closed", ErrIntakeClosed)
	}
	past, future, target := intakeNow.Add(-time.Hour), intakeNow.Add(time.Hour), uuid.NewV7()
	for _, tc := range []struct {
		facts  IntakeFacts
		closed bool
	}{
		{IntakeFacts{Status: IntakePending}, false},
		{IntakeFacts{Status: IntakeSnoozed, SnoozedTill: &future}, false},
		{IntakeFacts{Status: IntakeSnoozed, SnoozedTill: &past}, false},
		{IntakeFacts{Status: IntakeAccepted}, true},
		{IntakeFacts{Status: IntakeDeclined}, true},
		{IntakeFacts{Status: IntakeDuplicate, DuplicateToID: &target}, true},
	} {
		err := tc.facts.CheckOpen()
		if tc.closed != errors.Is(err, ErrIntakeClosed) || (!tc.closed && err != nil) {
			t.Errorf("CheckOpen() of %s = %v; closed %v", tc.facts.Status, err, tc.closed)
		}
	}
}
