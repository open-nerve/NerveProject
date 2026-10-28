package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Each page request has its own release, including repeated requests for A.
// Only the local network response is controlled; the crawler and River worker
// remain the ones installed by newApp.
type heldLinkPage struct {
	path  string
	reply chan string
}

type linkPages struct {
	url     string
	entered <-chan heldLinkPage
}

func newLinkRaceWorld(t *testing.T) (issueWorld, linkPages) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	entered := make(chan heldLinkPage, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			http.NotFound(w, r)
			return
		}
		page := heldLinkPage{path: r.URL.Path, reply: make(chan string, 1)}
		select {
		case entered <- page:
		case <-ctx.Done():
			return
		case <-r.Context().Done():
			return
		}
		select {
		case title := <-page.reply:
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprintf(w, "<title>%s</title>", title)
		case <-ctx.Done():
		case <-r.Context().Done():
		}
	}))
	// Also cover failure while preparing the world; the later cleanup runs first.
	t.Cleanup(func() { cancel(); server.Close() })
	w := newIssueWorldWith(t, func(c *config.Config) { c.Outbound.AllowCIDRs = []string{"127.0.0.1/32"} })
	t.Cleanup(cancel)
	return w, linkPages{url: server.URL, entered: entered}
}

func (p linkPages) wait(t *testing.T, path string) heldLinkPage {
	t.Helper()
	select {
	case page := <-p.entered:
		if page.path != path {
			t.Fatalf("crawl page=%q; want %q", page.path, path)
		}
		return page
	case <-time.After(10 * time.Second):
		t.Fatalf("crawl did not request %q", path)
		return heldLinkPage{}
	}
}

// Snapshot exact job identities after each committed API change. In A -> B -> A
// there must be three distinct IDs, even though two jobs have identical args.
func linkRaceJobs(t *testing.T, w issueWorld, id uuid.UUID, urls ...string) []int64 {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT id,args->>'link_id',args->>'url',queue,max_attempts FROM river_job WHERE kind='issue.crawl_link' AND args->>'link_id'=$1 ORDER BY id`, id.String())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []int64
	var got []string
	for rows.Next() {
		var job int64
		var link, url, queue string
		var attempts int
		if err := rows.Scan(&job, &link, &url, &queue, &attempts); err != nil {
			t.Fatal(err)
		}
		if link != id.String() || queue != "default" || attempts != 3 {
			t.Fatalf("job identity/options=%s %s/%d", link, queue, attempts)
		}
		ids, got = append(ids, job), append(got, url)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, urls) {
		t.Fatalf("crawl job URLs=%v; want %v", got, urls)
	}
	return ids
}

func waitLinkRaceJob(t *testing.T, w issueWorld, job int64, link uuid.UUID, url string) {
	t.Helper()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	for {
		var kind, gotLink, gotURL, state string
		var attempt int
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT kind,args->>'link_id',args->>'url',state::text,attempt FROM river_job WHERE id=$1`, job).Scan(&kind, &gotLink, &gotURL, &state, &attempt); err != nil {
			t.Fatal(err)
		}
		if kind != "issue.crawl_link" || gotLink != link.String() || gotURL != url || attempt > 1 {
			t.Fatalf("job %d identity/attempt=%s %s %q %d", job, kind, gotLink, gotURL, attempt)
		}
		if state == "completed" {
			return
		}
		select {
		case <-tick.C:
		case <-deadline.C:
			t.Fatalf("job %d state=%s; want completed", job, state)
		}
	}
}

func wantLinkRaceMetadata(t *testing.T, w issueWorld, id uuid.UUID, want domain.LinkMetadata) {
	t.Helper()
	var raw []byte
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT metadata FROM issue_links WHERE id=$1", id).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var got domain.LinkMetadata
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("link %s metadata=%s; want %+v", id, raw, want)
	}
}

func linkPageMetadata(url, title string) domain.LinkMetadata {
	return domain.LinkMetadata{"title": &title, "url": &url, "favicon": nil, "favicon_url": nil}
}

func changeLinkRaceURL(t *testing.T, w issueWorld, link answeredLink, url string) answeredLink {
	t.Helper()
	var got answeredLink
	decodeAnswer(t, w.send(t, "alice", http.MethodPatch, linkPath(link.ID), linkBody(url, nil), 200), &got)
	if got.ID != link.ID || got.URL != url || got.Metadata == nil || len(got.Metadata) != 0 {
		t.Fatalf("URL update did not return pending metadata: %+v", got)
	}
	wantLinkRaceMetadata(t, w, link.ID, domain.LinkMetadata{})
	return got
}

// The clock is reached after the locked reread and before MarkChanged/UpdateLink.
// The real project lock and the still-pending link are checked by the caller.
type linkReadClock struct {
	gate *gate
	ctx  context.Context
	err  error
}

func (c *linkReadClock) Now() time.Time {
	c.err = c.gate.wait(c.ctx)
	return time.Date(2044, 10, 11, 12, 13, 14, 0, time.UTC)
}

func linkRaceTitleRoute(t *testing.T, w issueWorld, clock *linkReadClock) moduleRoute {
	t.Helper()
	client, err := jobs.NewInserter(w.pool)
	if err != nil {
		t.Fatal(err)
	}
	subscribers := issue.Subscribers(issueInserter{client: client})
	subscribers = append(subscribers, shared.EventSubscriber{Name: "observe title transaction", Types: []string{"issue.link_updated"}, Handle: func(ctx context.Context, e shared.Event) error {
		if _, ok := postgres.Tx(ctx); !ok {
			return fmt.Errorf("title publication lost original transaction")
		}
		if !reflect.DeepEqual(e.Changes, []shared.Change{{Field: "title", New: eventValue("User title")}}) {
			return fmt.Errorf("title publication changes=%+v", e.Changes)
		}
		return nil
	}})
	publisher, err := newDispatcher(eventTypes(), subscribers)
	if err != nil {
		t.Fatal(err)
	}
	return newIssueRouteWithPublisher(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), clock, publisher)
}

// Every failed assertion releases and joins the request before pool cleanup.
func startLinkRaceTitle(t *testing.T, w issueWorld, link answeredLink) (*linkReadClock, func() answeredLink) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	clock := &linkReadClock{gate: newGate(), ctx: ctx}
	route := linkRaceTitleRoute(t, w, clock)
	var answer exchange
	done := run(func() error {
		answer.req, answer.rec = route.sendContext(ctx, http.MethodPatch, linkPath(link.ID), w.ids["alice"], `{"title":"User title"}`)
		return nil
	})
	release := func() {
		select {
		case <-clock.gate.open:
		default:
			close(clock.gate.open)
		}
	}
	t.Cleanup(func() {
		release()
		cancel()
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		select {
		case <-done:
		case <-cleanup.Done():
			t.Error("title request did not end during cleanup")
		}
	})
	held(t, ctx, clock.gate, done, "title locked reread")
	assertLinkProjectLock(t, w)
	if pgtest.RowWaits(t, w.pool, "issue_links", link.ID, "FOR UPDATE") {
		t.Fatal("title already locked link before the intended write point")
	}
	wantLinkRaceMetadata(t, w, link.ID, domain.LinkMetadata{})
	return clock, func() answeredLink {
		release()
		if err := result(t, ctx, done, "title update"); err != nil {
			t.Fatal(err)
		}
		if clock.err != nil {
			t.Fatal(clock.err)
		}
		w.answered(t, answer, http.StatusOK)
		var got answeredLink
		decodeAnswer(t, answer.rec.Body.String(), &got)
		return got
	}
}

// Keep every link field except metadata in the worker's no-business-write check.
func linkRaceAudit(t *testing.T, w issueWorld, item uuid.UUID) string {
	t.Helper()
	var links string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(jsonb_agg(to_jsonb(l)-'metadata' ORDER BY id),'[]')::text FROM issue_links l WHERE issue_id=$1`, item).Scan(&links); err != nil {
		t.Fatal(err)
	}
	return descriptionAudit(t, w, item) + links
}
