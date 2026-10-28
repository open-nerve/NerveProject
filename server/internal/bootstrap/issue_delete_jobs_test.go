package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestIssueDeleteSubscriberRegistration(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	client, err := jobs.NewInserter(pool)
	if err != nil {
		t.Fatal(err)
	}
	subs := issue.Subscribers(issueInserter{client: client})
	index := slices.IndexFunc(subs, func(s shared.EventSubscriber) bool { return s.Name == "delete_rows_on_deletion" })
	if index < 0 || !reflect.DeepEqual(subs[index].Types, []string{"project.deleted", "workspace.deleted"}) {
		t.Fatalf("missing deletion subscriber: %+v", subs)
	}
	observer, err := pgx.Connect(pgtest.Soon(t), pool.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = observer.Close(t.Context()) }()
	ws, p, q, old, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	at := time.Date(2044, 9, 10, 11, 12, 13, 123456000, time.UTC)
	event := shared.Event{Type: "workspace.deleted", Entity: shared.EntityRef{Kind: "workspace", ID: ws}, WorkspaceID: ws, OccurredAt: at, ActorID: &by, Changes: []shared.Change{{Field: "project", OldID: &p}, {Field: "project", OldID: &q}}}
	type payload struct {
		Scope string    `json:"scope"`
		ID    uuid.UUID `json:"id"`
		At    time.Time `json:"at"`
		By    uuid.UUID `json:"by"`
	}
	for _, scenario := range []string{"workspace", "zero projects", "project", "bad shape", "nil actor", "subscriber failure", "insert failure"} {
		t.Run(scenario, func(t *testing.T) {
			before := registrationRows(t, pool)
			e := event
			want := []payload{{"project", p, at, by}, {"project", q, at, by}, {"workspace_drafts", ws, at, by}}
			sentinel := errors.New("reject transaction")
			switch scenario {
			case "zero projects":
				e.Changes = nil
				want = want[2:]
			case "project":
				e.Type = "project.deleted"
				e.Entity = shared.EntityRef{Kind: "project", ID: p}
				e.ProjectID = p
				e.Changes = nil
				want = want[:1]
			case "bad shape":
				e.Changes = []shared.Change{{Field: "project", OldID: &p}, {Field: "project", NewID: &q}}
			case "nil actor":
				e.ActorID = nil
			}
			var caller context.Context
			var tx pgx.Tx
			var visible int
			subscribers := issue.Subscribers(issueInserter{client: client})
			subscribers = append(subscribers, shared.EventSubscriber{Name: "observe original transaction", Types: []string{"project.deleted", "workspace.deleted"}, Handle: func(ctx context.Context, _ shared.Event) error {
				got, ok := postgres.Tx(ctx)
				if !ok || got != tx || ctx != caller {
					t.Error("lost caller transaction/context")
				}
				var n int
				if err := observer.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind='issue.delete_rows'").Scan(&n); err != nil {
					return err
				}
				if n != visible {
					t.Fatal("uncommitted jobs visible")
				}
				if scenario == "subscriber failure" {
					return sentinel
				}
				return nil
			}})
			// Production uses the same module union; a missing declaration must fail construction.
			d, err := newDispatcher(eventTypes(), subscribers)
			if err != nil {
				t.Fatal(err)
			}
			if err := observer.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind='issue.delete_rows'").Scan(&visible); err != nil {
				t.Fatal(err)
			}
			err = postgres.NewTxManager(pool, 2*time.Second).WithinTx(pgtest.Soon(t), func(ctx context.Context) error {
				caller = ctx
				tx, _ = postgres.Tx(ctx)
				if scenario == "insert failure" {
					if _, err := tx.Exec(ctx, "ALTER TABLE river_job ADD CONSTRAINT reject_deletion CHECK (kind<>'issue.delete_rows') NOT VALID"); err != nil {
						return err
					}
				}
				return d.Publish(ctx, e)
			})
			reject := slices.Contains([]string{"bad shape", "nil actor", "subscriber failure", "insert failure"}, scenario)
			if reject {
				if err == nil || (scenario == "subscriber failure" && err != sentinel) {
					t.Fatalf("rejection=%v", err)
				}
				if after := registrationRows(t, pool); after != before {
					t.Fatal("rejected transaction left request task/payload changes")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			rows, err := pool.Query(pgtest.Soon(t), "SELECT args,queue,max_attempts FROM river_job WHERE kind='issue.delete_rows' ORDER BY id OFFSET $1", visible)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()
			var actual []payload
			for rows.Next() {
				var raw []byte
				var queue string
				var attempts int
				if err := rows.Scan(&raw, &queue, &attempts); err != nil {
					t.Fatal(err)
				}
				var got payload
				if err := json.Unmarshal(raw, &got); err != nil {
					t.Fatal(err)
				}
				var fields map[string]any
				if err := json.Unmarshal(raw, &fields); err != nil {
					t.Fatal(err)
				}
				if queue != "default" || attempts != 25 || len(fields) != 4 || got.ID == old {
					t.Fatalf("job=%s %s/%d", raw, queue, attempts)
				}
				actual = append(actual, got)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, want) {
				t.Fatalf("jobs=%+v want=%+v", actual, want)
			}
		})
	}
}

// The fully composed worker must drain batches even while another transaction
// owns the deleted project's N lock; audit time remains the original event's.
func TestNewAppDeleteRowsJobRuns(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) { cfg.Issue.DeleteBatch = 1 })
	item := w.newWorkItem(t)
	at := time.Now().Add(-61 * 24 * time.Hour).Truncate(time.Microsecond)
	by := w.ids["bob"]
	draft := uuid.NewV7()
	if _, err := w.pool.Exec(pgtest.Soon(t), "INSERT INTO draft_issues(id,workspace_id,created_by_id) VALUES($1,$2,$3)", draft, w.acme, by); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE projects SET deleted_at=$1 WHERE id=$2", at, w.web); err != nil {
		t.Fatal(err)
	}
	blocker, err := w.pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(context.Background()) }()
	if _, err := blocker.Exec(pgtest.Soon(t), "SELECT id FROM projects WHERE id=$1 FOR NO KEY UPDATE", w.web); err != nil {
		t.Fatal(err)
	}
	client, err := jobs.NewInserter(w.pool)
	if err != nil {
		t.Fatal(err)
	}
	d, err := newDispatcher(eventTypes(), issue.Subscribers(issueInserter{client: client}))
	if err != nil {
		t.Fatal(err)
	}
	err = postgres.NewTxManager(w.pool, 2*time.Second).WithinTx(pgtest.Soon(t), func(ctx context.Context) error {
		for _, e := range []shared.Event{
			{Type: "project.deleted", Entity: shared.EntityRef{Kind: "project", ID: w.web}, ProjectID: w.web, WorkspaceID: w.acme, OccurredAt: at, ActorID: &by},
			{Type: "workspace.deleted", Entity: shared.EntityRef{Kind: "workspace", ID: w.acme}, WorkspaceID: w.acme, OccurredAt: at, ActorID: &by},
		} {
			if err := d.Publish(ctx, e); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	for {
		var count, completed int
		if err := w.pool.QueryRow(ctx, "SELECT count(*),count(*) FILTER(WHERE state='completed') FROM river_job WHERE kind='issue.delete_rows'").Scan(&count, &completed); err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Fatalf("deletion jobs=%d want 2", count)
		}
		if completed == 2 {
			break
		}
		select {
		case <-tick.C:
		case <-ctx.Done():
			t.Fatal("composed deletion did not complete while project lock held", ctx.Err())
		}
	}
	for _, target := range []struct {
		table string
		id    uuid.UUID
	}{{"issues", item}, {"draft_issues", draft}} {
		var deleted, updated time.Time
		var actor uuid.UUID
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at,updated_at,updated_by_id FROM "+target.table+" WHERE id=$1", target.id).Scan(&deleted, &updated, &actor); err != nil {
			t.Fatal(err)
		}
		if !deleted.Equal(at) || !updated.Equal(at) || actor != by {
			t.Fatalf("%s audit=%s/%s/%s", target.table, deleted, updated, actor)
		}
	}
}
