package app_test

import (
	"context"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A list row's intake is its detail's, value for value (M7 P2b spec 2.3,
// DP-17): over one world, its work item 7 with its intake row as stored,
// the list's row, read at the moment the detail reads at, answers the
// effective status, a snooze's time while it lasts, and the work item a
// duplicate duplicates only when the caller sees it, as the detail
// answers them: a target not there, of another project, or hidden by the
// guest rule is none in both, one the guest made his in both.
func TestAListRowAnswersItsIntakeRowAsItsDetailDoes(t *testing.T) {
	missing, elsewhere := uuid.NewV7(), uuid.NewV7()
	guest := func(w *world) {
		w.issues.issue.By = alice
		w.auth.grants = []shared.Grant{{OwnOnly: true}, {OwnOnly: true}}
	}
	for _, tt := range []struct {
		name   string
		stored *domain.IntakeFacts
		set    func(w *world)
	}{
		{"pending", intakeOf(domain.IntakePending, nil, nil), func(*world) {}},
		{"snoozed", intakeOf(domain.IntakeSnoozed, &soon, nil), func(*world) {}},
		{"a snooze run out at the moment", intakeOf(domain.IntakeSnoozed, &clockNow, nil), func(*world) {}},
		{"a snooze run out before it", intakeOf(domain.IntakeSnoozed, &gone, nil), func(*world) {}},
		{"accepted", intakeOf(domain.IntakeAccepted, nil, nil), func(*world) {}},
		{"declined", intakeOf(domain.IntakeDeclined, nil, nil), func(*world) {}},
		{"a duplicate of one seen", intakeOf(domain.IntakeDuplicate, nil, &dup), func(*world) {}},
		{"a duplicate of one not there", intakeOf(domain.IntakeDuplicate, nil, &missing), func(*world) {}},
		{"a duplicate of another project's", intakeOf(domain.IntakeDuplicate, nil, &elsewhere), func(w *world) {
			w.issues.others[elsewhere] = domain.Issue{ID: elsewhere, WorkspaceID: acme, ProjectID: ops, CreatedByID: &carol}
		}},
		{"a duplicate the guest rule hides", intakeOf(domain.IntakeDuplicate, nil, &dup), guest},
		{"a duplicate the guest made", intakeOf(domain.IntakeDuplicate, nil, &dup), func(w *world) {
			guest(w)
			o := w.issues.others[dup]
			o.CreatedByID = &alice
			w.issues.others[dup] = o
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			detail, w, intake, id := newIntakeRead(tt.stored)
			tt.set(w)
			want, err := detail.Execute(as(alice), id)
			if err != nil || want.Intake == nil {
				t.Fatalf("the detail %+v, %v; want its intake row", want.Intake, err)
			}
			stored, _, _ := intake.GetIssue(context.Background(), id)
			rows := &intakeLister{log: w.log, pages: [][]app.IntakeRecord{{{Issue: stored, Total: 1,
				Position: domain.IntakeCursor{Order: "-created_at", At: stored.CreatedAt, ID: id}}}}}
			list := app.NewListIntakeIssues(app.ListIntakeIssuesDeps{ReadTx: fakeTx{log: w.log}, Locks: w.locks, Rows: rows,
				Refs: app.NewIssueRefs(storeRefs{w.issues}), Clock: fixedAt(clockNow)})
			page, err := list.Execute(as(alice), web, nil)
			if err != nil || len(page.Data) != 1 || !reflect.DeepEqual(page.Data[0].Intake, want.Intake) {
				t.Errorf("the list's row %+v, %v; want its intake %+v, the detail's", page.Data, err, *want.Intake)
			}
		})
	}
}
