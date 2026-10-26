package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestIssueLinkSubscriberRegistration(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	client, err := jobs.NewInserter(pool)
	if err != nil {
		t.Fatal(err)
	}
	// The production subscriber must be constructible before issue.New.
	subscribers := issue.Subscribers(issueInserter{client: client})
	if len(subscribers) != 1 || subscribers[0].Name != "crawl_on_link_change" || !reflect.DeepEqual(subscribers[0].Types, []string{"issue.link_created", "issue.link_updated"}) {
		t.Fatalf("production subscribers=%+v; want unique link change consumer", subscribers)
	}
	d, err := newDispatcher(issue.EventTypes(), subscribers)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewV7()
	a, b := "http://169.254.169.254/A", "http://169.254.169.254/B"
	err = postgres.NewTxManager(pool, 2*time.Second).WithinTx(pgtest.Soon(t), func(ctx context.Context) error {
		// A -> B -> A must keep all three jobs, even when an old URL returns.
		for i, url := range []string{a, b, a} {
			kind := "issue.link_updated"
			if i == 0 {
				kind = "issue.link_created"
			}
			if err := d.Publish(ctx, shared.Event{ID: uuid.NewV7(), Type: kind, Entity: shared.EntityRef{Kind: "issue_link", ID: id}, Changes: []shared.Change{{Field: "url", New: eventValue(url)}}}); err != nil {
				return err
			}
		}
		for _, kind := range []string{"issue.link_updated", "issue.link_deleted"} {
			if err := d.Publish(ctx, shared.Event{ID: uuid.NewV7(), Type: kind, Entity: shared.EntityRef{Kind: "issue_link", ID: id}, Changes: []shared.Change{{Field: "title", New: eventValue("named")}}}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(pgtest.Soon(t), `SELECT args->>'link_id',args->>'url',queue,max_attempts FROM river_job WHERE kind='issue.crawl_link' ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var urls []string
	for rows.Next() {
		var link, url, queue string
		var attempts int
		if err := rows.Scan(&link, &url, &queue, &attempts); err != nil {
			t.Fatal(err)
		}
		if link != id.String() || queue != "default" || attempts != 3 {
			t.Fatalf("job=%s %s %s/%d", link, url, queue, attempts)
		}
		urls = append(urls, url)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(urls, []string{a, b, a}) {
		t.Fatalf("jobs=%v; want each URL change once", urls)
	}
}

type linkJobPublisher func(context.Context, shared.Event) error

func (f linkJobPublisher) Publish(ctx context.Context, e shared.Event) error { return f(ctx, e) }

// Running production workers and an independent connection observe the real
// business transaction: publication cannot expose a job before its link.
func TestLinkJobCommitRollback(t *testing.T) {
	w := newIssueWorld(t)
	observer, err := pgx.Connect(pgtest.Soon(t), w.pool.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = observer.Close(context.Background()) })
	client, err := jobs.NewInserter(w.pool)
	if err != nil {
		t.Fatal(err)
	}
	store := issuepg.New(w.pool)
	locks := issueapp.NewLocks(issueProjects{projects: project.Provide(w.pool).IssueProjects}, store, authorizerOn(w.pool), newWriteGate(testIssueConfig.WriteGate), postgres.NewTxManager(w.pool, 2*time.Second))
	for _, scenario := range []string{"commit created", "commit updated", "rollback", "title only", "created missing url", "created duplicate url", "updated duplicate url", "created nil url", "updated nil url", "enqueue error", "second fails"} {
		t.Run(scenario, func(t *testing.T) {
			item := w.newWorkItem(t)
			at := time.Date(2044, 9, 10, 11, 12, 13, 123456000, time.UTC)
			type contextKey struct{}
			ctx, cancel := context.WithCancel(context.WithValue(shared.WithActor(pgtest.Soon(t), shared.Actor{UserID: w.ids["bob"]}), contextKey{}, scenario))
			defer cancel()
			isUpdate := scenario == "title only" || scenario == "commit updated" || scenario == "updated duplicate url" || scenario == "updated nil url"
			var linkID uuid.UUID
			if isUpdate {
				seed, err := issueapp.NewCreateIssueLink(issueapp.CreateIssueLinkDeps{Locks: locks, Links: store, Clock: clockAt(at.Add(-time.Hour)), Events: discardIssueEvents{}}).Execute(ctx, item, domain.NewLink{URL: "http://169.254.169.254/before"})
				if err != nil {
					t.Fatal(err)
				}
				linkID = seed.ID
			}
			before, _ := linkJobRows(t, observer, item, linkID)
			sentinel := errors.New("publication rejected")
			subscribers := issue.Subscribers(issueInserter{client: client})
			var published shared.Event
			var caller context.Context
			var transaction pgx.Tx
			var calls []string
			subscribers = append(subscribers,
				shared.EventSubscriber{Name: "second checks original transaction", Types: []string{"issue.link_created", "issue.link_updated"}, Handle: func(got context.Context, e shared.Event) error {
					calls = append(calls, "second")
					tx, ok := postgres.Tx(got)
					if !ok || tx != transaction || got != caller || got.Value(contextKey{}) != scenario {
						t.Error("subscriber lost original caller context/transaction")
					}
					var inside int
					if err := tx.QueryRow(got, `SELECT count(*) FROM river_job WHERE kind='issue.crawl_link' AND args->>'link_id'=$1`, e.Entity.ID.String()).Scan(&inside); err != nil {
						return err
					}
					want := 1
					if scenario == "title only" {
						want = 0
					}
					if inside != want {
						t.Errorf("jobs in original transaction=%d; want %d", inside, want)
					}
					if scenario == "second fails" {
						return sentinel
					}
					return nil
				}},
				shared.EventSubscriber{Name: "third observes", Types: []string{"issue.link_created", "issue.link_updated"}, Handle: func(context.Context, shared.Event) error { calls = append(calls, "third"); return nil }},
			)
			d, err := newDispatcher(issue.EventTypes(), subscribers)
			if err != nil {
				t.Fatal(err)
			}
			publisher := linkJobPublisher(func(got context.Context, e shared.Event) error {
				published = e
				linkID = e.Entity.ID
				caller = got
				var ok bool
				transaction, ok = postgres.Tx(got)
				if !ok {
					t.Fatal("business publication has no transaction")
				}
				if e.ID == e.Entity.ID || e.Entity.Kind != "issue_link" {
					t.Fatal("event identity is not independent")
				}
				if e.ActorID == nil || *e.ActorID != w.ids["bob"] || !e.OccurredAt.Equal(at) {
					t.Fatal("event lost actor/time")
				}
				switch scenario {
				case "created missing url":
					e.Changes = []shared.Change{{Field: "title", New: eventValue("named")}}
				case "created duplicate url", "updated duplicate url":
					e.Changes = append(e.Changes, shared.Change{Field: "url", New: eventValue("http://169.254.169.254/duplicate")})
				case "created nil url", "updated nil url":
					e.Changes = []shared.Change{{Field: "url"}}
				case "enqueue error":
					cancel()
				}
				err := d.Publish(got, e)
				outside, n := linkJobRows(t, observer, item, linkID)
				if outside != before || n != 0 {
					t.Errorf("uncommitted business/job visible on independent connection: jobs=%d business changed=%t", n, outside != before)
				}
				if err != nil {
					return err
				}
				if scenario == "rollback" {
					return sentinel
				}
				return nil
			})
			var result domain.Link
			url := "http://169.254.169.254/after?Exact=%2F"
			if isUpdate {
				patch := domain.LinkPatch{URL: &url}
				if scenario == "title only" {
					patch = domain.LinkPatch{SetTitle: true, Title: eventValue("Changed title")}
				}
				result, err = issueapp.NewUpdateIssueLink(issueapp.UpdateIssueLinkDeps{Locks: locks, Links: store, Clock: clockAt(at), Events: publisher}).Execute(ctx, linkID, patch)
			} else {
				result, err = issueapp.NewCreateIssueLink(issueapp.CreateIssueLinkDeps{Locks: locks, Links: store, Clock: clockAt(at), Events: publisher}).Execute(ctx, item, domain.NewLink{URL: url})
			}
			success := scenario == "commit created" || scenario == "commit updated" || scenario == "title only"
			if (err == nil) != success {
				t.Fatalf("write error=%v; want success=%t", err, success)
			}
			if published.ID == (uuid.UUID{}) {
				t.Fatal("business write did not reach publication")
			}
			if scenario == "rollback" || scenario == "second fails" {
				if !errors.Is(err, sentinel) {
					t.Fatalf("lost subscriber error: %v", err)
				}
			}
			if scenario == "enqueue error" && !errors.Is(err, context.Canceled) {
				t.Fatalf("lost insertion error: %v", err)
			}
			var wantCalls []string
			switch scenario {
			case "commit created", "commit updated", "title only", "rollback":
				wantCalls = []string{"second", "third"}
			case "second fails":
				wantCalls = []string{"second"}
			}
			if !reflect.DeepEqual(calls, wantCalls) {
				t.Errorf("subscriber calls=%v; want %v", calls, wantCalls)
			}
			after, n := linkJobRows(t, observer, item, linkID)
			if !success {
				if before != after || n != 0 {
					t.Fatalf("rollback changed business/activity/jobs: jobs=%d business changed=%t", n, before != after)
				}
				return
			}
			wantJobs := 1
			if scenario == "title only" {
				wantJobs = 0
			}
			if n != wantJobs {
				t.Fatalf("committed jobs=%d; want %d", n, wantJobs)
			}
			var stamped bool
			if err := observer.QueryRow(pgtest.Soon(t), `SELECT l.updated_at=$2 AND l.updated_by_id=$3 AND i.updated_at=$2 AND i.updated_by_id=$3 FROM issue_links l JOIN issues i ON i.id=l.issue_id WHERE l.id=$1`, linkID, at, w.ids["bob"]).Scan(&stamped); err != nil || !stamped {
				t.Fatal("committed link/issue audit", stamped, err)
			}
			if scenario == "title only" {
				var title string
				var activities int
				if err := observer.QueryRow(pgtest.Soon(t), `SELECT title,(SELECT count(*) FROM issue_activities WHERE issue_id=$2 AND field='link') FROM issue_links WHERE id=$1`, linkID, item).Scan(&title, &activities); err != nil {
					t.Fatal(err)
				}
				if title != "Changed title" || activities != 1 || before == after {
					t.Fatalf("title-only not committed independently: title=%q activities=%d", title, activities)
				}
				return
			}
			var argsID, argsURL, queue string
			var attempts, activity int
			if err := observer.QueryRow(pgtest.Soon(t), `SELECT args->>'link_id',args->>'url',queue,max_attempts FROM river_job WHERE kind='issue.crawl_link' AND args->>'link_id'=$1`, linkID.String()).Scan(&argsID, &argsURL, &queue, &attempts); err != nil {
				t.Fatal(err)
			}
			if argsID != result.ID.String() || argsURL != url || queue != "default" || attempts != 3 {
				t.Fatalf("committed job=%s %s %s/%d", argsID, argsURL, queue, attempts)
			}
			if err := observer.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='link' AND actor_id=$2 AND created_at=$3 AND updated_at=$3 AND created_by_id=$2 AND updated_by_id=$2 AND new_value=$4`, item, w.ids["bob"], at, url).Scan(&activity); err != nil || activity != 1 {
				t.Fatal("committed link activity", activity, err)
			}
		})
	}
}

func linkJobRows(t *testing.T, observer *pgx.Conn, item, link uuid.UUID) (string, int) {
	t.Helper()
	var rows string
	var n int
	err := observer.QueryRow(pgtest.Soon(t), `SELECT jsonb_build_object('issue',(SELECT to_jsonb(i) FROM issues i WHERE id=$1),'links',(SELECT coalesce(jsonb_agg(to_jsonb(l) ORDER BY id),'[]') FROM issue_links l WHERE issue_id=$1),'activities',(SELECT coalesce(jsonb_agg(to_jsonb(a) ORDER BY id),'[]') FROM issue_activities a WHERE issue_id=$1))::text,(SELECT count(*) FROM river_job WHERE kind='issue.crawl_link' AND args->>'link_id'=$2)`, item, link.String()).Scan(&rows, &n)
	if err != nil {
		t.Fatal(err)
	}
	return rows, n
}
