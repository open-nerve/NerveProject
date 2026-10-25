package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type activityReads struct {
	log      *callLog
	rows     []app.ActivityRecord
	projects []app.VisibleProject
	received []app.VisibleProject
	limit    int
	after    *domain.ActivityCursor
	actor    uuid.UUID
}

func (f *activityReads) Historical(ctx context.Context, ws, actor uuid.UUID) ([]app.VisibleProject, error) {
	return f.projects, f.log.add(ctx, "Historical", ws, actor)
}
func (f *activityReads) Visible(context.Context, uuid.UUID, uuid.UUID) ([]app.VisibleProject, error) {
	panic("activity references must use Historical, not Visible")
}
func (f *activityReads) Activities(ctx context.Context, id, actor uuid.UUID, projects []app.VisibleProject, after *domain.ActivityCursor, limit int) ([]app.ActivityRecord, error) {
	f.limit, f.after, f.received, f.actor = limit, after, projects, actor
	return f.rows, f.log.add(ctx, "Activities", id)
}
func activityList(w *world, f *activityReads) *app.ListIssueActivities {
	return app.NewListIssueActivities(app.ListIssueActivitiesDeps{Locks: w.locks, Activities: f, Projects: f, ReadTx: fakeTx{log: w.log}})
}
func TestActivityPagesFilterBeforeLimit(t *testing.T) {
	_, w, id := newUpdate()
	f := &activityReads{log: w.log, projects: []app.VisibleProject{{ID: web, OwnOnly: true}}, rows: []app.ActivityRecord{{ID: bob, CreatedAt: clockNow}, {ID: carol, CreatedAt: clockNow}}}
	limit := 1
	page, err := activityList(w, f).Execute(as(alice), id, &limit, nil)
	if err != nil || len(page.Data) != 1 || page.Data[0].ID != bob {
		t.Fatal("visible page missing", page, err)
	}
	var c domain.ActivityCursor
	if err := shared.DecodeCursor(page.NextCursor, &c); err != nil || c.ID != page.Data[0].ID || !c.CreatedAt.Equal(clockNow) {
		t.Fatal("cursor names probe row", c, err)
	}
	if f.limit != 2 || f.actor != alice || !reflect.DeepEqual(f.received, f.projects) {
		t.Fatal("page query scope", f)
	}
	calls := []string{"Begin", line("GetIssue", id), line("Authorize", alice, domain.ActionRead, acme, web), line("Historical", acme, alice), line("Activities", id)}
	if !slices.Equal(w.log.calls, calls) {
		t.Fatal("entire read must share snapshot", w.log.calls)
	}
	f.rows = nil
	page, err = activityList(w, f).Execute(as(alice), id, nil, &page.NextCursor)
	if err != nil || page.Data == nil || page.NextCursor != "" || f.after.ID != bob || f.limit != 51 {
		t.Fatal("terminal read", page, err)
	}
}
func TestActivityReferenceVisibility(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		_, w, id := newUpdate()
		f := &activityReads{log: w.log}
		if hidden {
			w.auth.grants = []shared.Grant{ownOnly}
		}
		_, err := activityList(w, f).Execute(as(alice), id, nil, nil)
		if hidden {
			wantRefused(t, err, domain.ErrIssueNotFound)
			if len(w.log.calls) != 3 {
				t.Fatal(w.log.calls)
			}
		} else if err != nil {
			t.Fatal(err)
		}
	}
	for _, port := range []string{"GetIssue", "Authorize", "Historical", "Activities"} {
		t.Run(port, func(t *testing.T) {
			_, w, id := newUpdate()
			w.log.fail = port
			f := &activityReads{log: w.log}
			_, err := activityList(w, f).Execute(as(alice), id, nil, nil)
			if !errors.Is(err, errDisk) {
				t.Fatal("port error lost", err)
			}
		})
	}
}
func TestActivityReadValidation(t *testing.T) {
	_, w, id := newUpdate()
	f := &activityReads{log: w.log}
	u := activityList(w, f)
	bad, zero := "bad", 0
	_, err := u.Execute(context.Background(), id, nil, nil)
	wantRefused(t, err, shared.Unauthenticated())
	_, err = u.Execute(as(alice), id, &zero, &bad)
	wantRefused(t, err, shared.InvalidCursor())
	_, err = u.Execute(as(alice), id, &zero, nil)
	var se *shared.Error
	if !errors.As(err, &se) || se.Code != "validation_failed" {
		t.Fatal(err)
	}
	if len(w.log.calls) != 0 {
		t.Fatal("invalid read reached ports", w.log.calls)
	}
}

// activityReadFailureTx refuses a snapshot before any target or catalog read.
type activityReadFailureTx struct{}

func (activityReadFailureTx) WithinTx(context.Context, func(context.Context) error) error {
	return errDisk
}
func TestActivityReadTxFailure(t *testing.T) {
	_, w, id := newUpdate()
	f := &activityReads{log: w.log}
	u := app.NewListIssueActivities(app.ListIssueActivitiesDeps{Locks: w.locks, Activities: f, Projects: f, ReadTx: activityReadFailureTx{}})
	_, err := u.Execute(as(alice), id, nil, nil)
	if !errors.Is(err, errDisk) {
		t.Fatal("snapshot error lost", err)
	}
	if len(w.log.calls) != 0 {
		t.Fatal("failed snapshot reached read ports", w.log.calls)
	}
}
