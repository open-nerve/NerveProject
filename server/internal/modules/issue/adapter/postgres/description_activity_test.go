package postgresadapter_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestLatestActivity(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	issue, neighbor := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, issue, 1, alice)
	newIssue(t, pool, web, neighbor, 2, alice)
	if got, found, err := s.LatestActivity(ctx, issue); err != nil || found || !reflect.DeepEqual(got, app.ActivityRecord{}) {
		t.Fatalf("empty latest = %+v, %v, %v", got, found, err)
	}
	description, name, old, next := "description", "name", "<p>origin</p>", "<p>first</p>"
	activity := domain.Activity{Field: &description, Verb: domain.VerbUpdated, OldValue: &old, NewValue: &next, Comment: "updated the description to"}
	insert := func(id, under, by uuid.UUID, at time.Time, a domain.Activity) {
		t.Helper()
		if err := s.InsertActivities(ctx, []app.ActivityRow{{ID: id, WorkspaceID: web.workspace, ProjectID: web.id, IssueID: under, ActorID: by, At: at, Activity: a}}); err != nil {
			t.Fatal(err)
		}
	}
	assertLatest := func(id, by uuid.UUID, at time.Time, a domain.Activity) {
		t.Helper()
		want := app.ActivityRecord{ID: id, WorkspaceID: web.workspace, ProjectID: web.id, IssueID: issue, ActorID: &by, CreatedAt: at, UpdatedAt: at, Activity: a}
		got, found, err := s.LatestActivity(ctx, issue)
		if err != nil || !found || !reflect.DeepEqual(got, want) {
			t.Fatalf("latest = %+v, %v, %v; want %+v", got, found, err, want)
		}
	}
	first, higher, lower := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	// Use the higher UUID for the older row, and insert tied rows in ascending id order.
	insert(lower, issue, alice, now.Add(-time.Hour), activity)
	insert(uuid.NewV7(), neighbor, alice, now.Add(time.Hour), activity)
	gone := uuid.NewV7()
	insert(gone, issue, alice, now.Add(2*time.Hour), activity)
	exec(t, pool, "UPDATE issue_activities SET deleted_at = $2 WHERE id = $1", gone, now)
	assertLatest(lower, alice, now.Add(-time.Hour), activity)
	insert(first, issue, alice, now, activity)
	title := domain.Activity{Field: &name, Verb: domain.VerbUpdated, OldValue: &old, NewValue: &next, Comment: "updated the name to"}
	insert(higher, issue, alice, now, title)
	assertLatest(higher, alice, now, title)
	t.Run("same time uses id without incidental index ordering", func(t *testing.T) {
		// Remove the index's incidental tie ordering, so only ORDER BY can decide the latest row.
		err := postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
			if _, err := postgres.DB(txctx, pool).Exec(txctx, "SET LOCAL enable_indexscan = off; SET LOCAL enable_indexonlyscan = off; SET LOCAL enable_bitmapscan = off"); err != nil {
				return err
			}
			got, found, err := s.LatestActivity(txctx, issue)
			if err != nil || !found || got.ID != higher {
				t.Fatalf("same-time latest id = %s, found %v, error %v; want larger id %s", got.ID, found, err, higher)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	later := uuid.NewV7()
	insert(later, issue, bob, now.Add(time.Minute), activity)
	assertLatest(later, bob, now.Add(time.Minute), activity)

	t.Run("merge keeps origin and changes current content and audit", func(t *testing.T) {
		before := textRows(t, pool, "SELECT row_to_json(a)::text FROM issue_activities a WHERE id <> $1 ORDER BY id", later)
		at := now.Add(2 * time.Minute)
		m := app.DescriptionActivityMerge{ID: later, By: alice, NewHTML: "<p>latest</p>", At: at}
		if err := s.MergeDescriptionActivity(ctx, m); err != nil {
			t.Fatal(err)
		}
		wantActivity := activity
		wantActivity.NewValue = &m.NewHTML
		got, found, err := s.LatestActivity(ctx, issue)
		want := app.ActivityRecord{ID: later, WorkspaceID: web.workspace, ProjectID: web.id, IssueID: issue, ActorID: &bob, CreatedAt: at, UpdatedAt: at, Activity: wantActivity}
		if got.Activity.NewValue == nil || *got.Activity.NewValue != m.NewHTML {
			t.Fatalf("merge new_value = %v, want %q", got.Activity.NewValue, m.NewHTML)
		}
		if err != nil || !found || !reflect.DeepEqual(got, want) {
			t.Fatalf("merged latest = %+v, %v, %v; want preserved origin and new HTML/time %+v", got, found, err, want)
		}
		var createdBy, updatedBy uuid.UUID
		if err := pool.QueryRow(ctx, "SELECT created_by_id, updated_by_id FROM issue_activities WHERE id = $1", later).Scan(&createdBy, &updatedBy); err != nil {
			t.Fatal(err)
		}
		if createdBy != bob || updatedBy != alice {
			t.Fatalf("merge audit = %s/%s, want original %s / writer %s", createdBy, updatedBy, bob, alice)
		}
		after := textRows(t, pool, "SELECT row_to_json(a)::text FROM issue_activities a WHERE id <> $1 ORDER BY id", later)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("merge changed a neighboring or deleted activity")
		}
	})

	t.Run("transaction context and nullable actor", func(t *testing.T) {
		aborted := errors.New("rollback")
		err := postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
			id := uuid.NewV7()
			at := now.Add(3 * time.Minute)
			if _, err := postgres.DB(txctx, pool).Exec(txctx, `INSERT INTO issue_activities (id, workspace_id, project_id, issue_id, verb, comment, created_at, updated_at) VALUES ($1,$2,$3,$4,'created','system',$5,$5)`, id, web.workspace, web.id, issue, at); err != nil {
				return err
			}
			got, found, err := s.LatestActivity(txctx, issue)
			if err != nil || !found || got.ID != id || got.ActorID != nil {
				t.Fatalf("transaction latest = %+v, %v, %v", got, found, err)
			}
			if err := s.MergeDescriptionActivity(txctx, app.DescriptionActivityMerge{ID: later, By: alice, NewHTML: "rolled back", At: at}); err != nil {
				return err
			}
			return aborted
		})
		if err != aborted {
			t.Fatal(err)
		}
		got, _, err := s.LatestActivity(ctx, issue)
		if err != nil || got.ID != later || got.Activity.NewValue == nil || *got.Activity.NewValue != "<p>latest</p>" {
			t.Fatalf("rollback latest = %+v, %v", got, err)
		}
	})

	t.Run("statement errors remain errors", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if _, found, err := s.LatestActivity(cancelled, issue); !errors.Is(err, context.Canceled) || found {
			t.Fatalf("latest canceled = %v, %v", found, err)
		}
		if err := s.MergeDescriptionActivity(cancelled, app.DescriptionActivityMerge{ID: later, By: alice, At: now}); !errors.Is(err, context.Canceled) {
			t.Fatalf("merge canceled = %v", err)
		}
	})
}
