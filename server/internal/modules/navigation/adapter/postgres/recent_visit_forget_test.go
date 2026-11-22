package postgresadapter_test

import (
	"context"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A work item's deletion, as the issue module publishes it, through the module's subscriber over the store in the
// publisher's transaction (P3d spec 2.6): every account's undeleted visits of the work item in the event's workspace
// are deleted at the event's moment by its actor, who visited none of them, their visits kept; their visits of
// another work item and of its project, and a visit of the same id in another workspace, stay as they were. A
// deletion without an actor writes none.
func TestAWorkItemsDeletionForgetsEveryonesVisitsOfIt(t *testing.T) {
	s, pool := newStore(t)
	w := newViews(t, pool)
	deleter := uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'dan@corp.com', 'x', 'dan')", deleter)
	item, other := uuid.NewV7(), uuid.NewV7()
	made, visited := now.Add(-2*time.Hour), now.Add(-time.Hour)
	of := func(n int, workspace, project, user uuid.UUID, kind domain.RecentVisitKind, target uuid.UUID) visitRow {
		return visitRow{numberedID(n), workspace, project, user, kind, target, made, visited, nil}
	}
	seedVisit(t, pool, of(1, w.workspace, w.project, w.owner, domain.RecentVisitIssue, item), of(2, w.workspace, w.project, w.other, domain.RecentVisitIssue, item),
		of(3, w.workspace, w.project, w.other, domain.RecentVisitIssue, other), of(4, w.workspace, w.project, w.owner, domain.RecentVisitProject, w.project),
		of(5, w.elsewhere, w.elsewhereProject, w.owner, domain.RecentVisitIssue, item))
	forget := app.NewForgetDeletedIssueVisits(s)
	deletion := func(issue uuid.UUID, at time.Time, by *uuid.UUID) error {
		e := shared.Event{ID: uuid.NewV7(), Type: "issue.deleted", OccurredAt: at, WorkspaceID: w.workspace, ProjectID: w.project, ActorID: by,
			Entity: shared.EntityRef{Kind: "issue", ID: issue}}
		return postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error { return forget.Handle(ctx, e) })
	}

	gone, kept := []uuid.UUID{numberedID(1), numberedID(2)}, []uuid.UUID{numberedID(3), numberedID(4), numberedID(5)}
	want, unchanged := readVisits(t, pool, gone...), readVisits(t, pool, kept...)
	at := now.Add(time.Hour)
	for i := range want {
		want[i].Deleted, want[i].Changed, want[i].Changer = &at, at, &deleter
	}
	if err := deletion(item, at, &deleter); err != nil {
		t.Fatal(err)
	}
	if got := readVisits(t, pool, gone...); !reflect.DeepEqual(got, want) {
		t.Fatalf("both accounts' visits of the work item %+v; want %+v", got, want)
	}
	if got := readVisits(t, pool, kept...); !reflect.DeepEqual(got, unchanged) {
		t.Fatalf("another work item's, the project's and another workspace's %+v; want %+v", got, unchanged)
	}

	later := at.Add(time.Minute)
	nobody := readVisits(t, pool, numberedID(3))
	nobody[0].Deleted, nobody[0].Changed, nobody[0].Changer = &later, later, nil
	if err := deletion(other, later, nil); err != nil {
		t.Fatal(err)
	}
	if got := readVisits(t, pool, numberedID(3)); !reflect.DeepEqual(got, nobody) {
		t.Fatalf("deleted without an actor %+v; want %+v", got, nobody)
	}
}
