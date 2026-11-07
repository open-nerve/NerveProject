package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// moduleLinkCrawls are the planning.crawl_link jobs of the module link id, each as its address, oldest first.
func (w issueWorld) moduleLinkCrawls(t *testing.T, id uuid.UUID) []string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT args->>'url' FROM river_job WHERE kind = 'planning.crawl_link' AND args->>'link_id' = $1
		ORDER BY id`, id.String())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	urls := []string{}
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			t.Fatal(err)
		}
		urls = append(urls, url)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return urls
}

// waitModuleLinkCrawl waits for the one planning.crawl_link job of the link at its address, on the default queue with
// three attempts, to complete; a second job, or none, fails the test.
func (w issueWorld) waitModuleLinkCrawl(t *testing.T, link moduleLinkAnswer) {
	t.Helper()
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	for {
		var count, attempts int
		var state, queue string
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*), coalesce(min(state::text), ''), coalesce(min(queue), ''), coalesce(min(max_attempts), 0)
			FROM river_job WHERE kind = 'planning.crawl_link' AND args->>'link_id' = $1 AND args->>'url' = $2`, link.ID.String(), link.URL).Scan(
			&count, &state, &queue, &attempts); err != nil {
			t.Fatal(err)
		}
		if count != 1 || queue != "default" || attempts != 3 {
			t.Fatalf("crawl jobs = %d on %s with %d attempts; want one on default with 3 for %s %q", count, queue, attempts, link.ID, link.URL)
		}
		if state == "completed" {
			return
		}
		select {
		case <-tick.C:
		case <-deadline.C:
			t.Fatalf("crawl job %s; want completed for %s %q", state, link.ID, link.URL)
		}
	}
}

// A module link's crawl runs through the app as bootstrap wires it (M6 P2 spec R11, R12): its creation enqueues one
// planning.crawl_link, whose worker fetches the page through the outbound client every module shares, as itself, and
// writes the metadata it found, the link's other columns and its module as they were.
func TestModuleLinkCrawlRuns(t *testing.T) {
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("icon"))
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, "<title>%s</title>", r.UserAgent())
	}))
	defer fixture.Close()
	w := newIssueWorldWith(t, func(cfg *config.Config) { cfg.Outbound.AllowCIDRs = []string{"127.0.0.1/32"} })
	module := w.linkedModule(t, "Billing")
	link := w.addLink(t, "bob", module, `{"url":"`+fixture.URL+`/page"}`, http.StatusCreated)
	rows := func() string {
		var s string
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT jsonb_build_object('link', (SELECT to_jsonb(l) - 'metadata' FROM module_links l WHERE id = $1),
			'module', (SELECT to_jsonb(m) FROM modules m WHERE id = $2))::text`, link.ID, module).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := rows()
	w.waitModuleLinkCrawl(t, link)
	var metadata []byte
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT metadata FROM module_links WHERE id = $1", link.ID).Scan(&metadata); err != nil {
		t.Fatal(err)
	}
	var got shared.LinkMetadata
	if err := json.Unmarshal(metadata, &got); err != nil {
		t.Fatal(err)
	}
	title, icon, iconURL := "Nerve/"+buildinfo.Get().Version+" (+link preview)", "data:image/png;base64,aWNvbg==", fixture.URL+"/favicon.ico"
	if want := (shared.LinkMetadata{"title": &title, "url": &link.URL, "favicon": &icon, "favicon_url": &iconURL}); !reflect.DeepEqual(got, want) {
		t.Fatalf("the crawl's metadata = %s; want %v", metadata, want)
	}
	if after := rows(); after != before {
		t.Fatalf("the crawl changed the link or its module:\n%s\nwas\n%s", after, before)
	}
}

// crawlsSeen are the planning.crawl_link jobs a write's transaction sees once its body has run, and those an
// independent connection sees then.
type crawlsSeen struct{ inside, outside int }

// observedTx runs each write in tx and, once its body has run and before it commits, counts the crawl jobs in its
// transaction and on pool into seen; then it commits, or answers rollback, which rolls it back.
type observedTx struct {
	tx       shared.TxManager
	pool     *pgxpool.Pool
	seen     *crawlsSeen
	rollback error
}

func (o observedTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return o.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := fn(ctx); err != nil {
			return err
		}
		const count = "SELECT count(*) FROM river_job WHERE kind = 'planning.crawl_link'"
		tx, ok := postgres.Tx(ctx)
		if !ok {
			return errors.New("a write without a transaction")
		}
		if err := tx.QueryRow(ctx, count).Scan(&o.seen.inside); err != nil {
			return err
		}
		if err := o.pool.QueryRow(ctx, count).Scan(&o.seen.outside); err != nil {
			return err
		}
		return o.rollback
	})
}

// A link's creation and each change of its address enqueue one crawl of the address written, in the write's
// transaction, unseen outside it before it commits (M6 P2 spec R11); an address it had before is crawled again. A
// title alone, a change of nothing, a refused write and a write rolled back enqueue none.
func TestModuleLinkCrawlJobs(t *testing.T) {
	w := newIssueWorld(t)
	module := w.linkedModule(t, "Billing")
	var seen crawlsSeen
	tx := postgres.NewTxManager(w.pool, 2*time.Second)
	route := newPlanningRoute(t, w.pool, observedTx{tx: tx, pool: w.pool, seen: &seen})
	send := func(method, path, body string, want int) string {
		t.Helper()
		seen = crawlsSeen{-1, -1}
		req, rec := route.send(method, path, w.ids["bob"], body)
		w.contract.CheckResponse(t, req, rec.Result())
		if rec.Code != want {
			t.Fatalf("%s %s with %s = %d %s, want %d", method, path, body, rec.Code, rec.Body, want)
		}
		return rec.Body.String()
	}
	var link moduleLinkAnswer
	decodeAnswer(t, send(http.MethodPost, "/api/v0/modules/"+module.String()+"/links", `{"url":" 169.254.169.254/a "}`, http.StatusCreated), &link)
	if seen != (crawlsSeen{1, 0}) {
		t.Fatalf("the creation's transaction sees %d crawls, another connection %d; want 1 and 0", seen.inside, seen.outside)
	}
	path := "/api/v0/module-links/" + link.ID.String()
	for _, tt := range []struct {
		body   string
		crawls []string
	}{
		{`{"title":"Spec"}`, []string{"http://169.254.169.254/a"}},
		{`{"url":"http://169.254.169.254/a","title":"Spec"}`, []string{"http://169.254.169.254/a"}},
		{`{"url":"169.254.169.254/b"}`, []string{"http://169.254.169.254/a", "http://169.254.169.254/b"}},
		{`{"url":"169.254.169.254/a","title":null}`, []string{"http://169.254.169.254/a", "http://169.254.169.254/b", "http://169.254.169.254/a"}},
	} {
		send(http.MethodPatch, path, tt.body, http.StatusOK)
		if got := w.moduleLinkCrawls(t, link.ID); !reflect.DeepEqual(got, tt.crawls) {
			t.Fatalf("after %s the crawls are %q; want %q", tt.body, got, tt.crawls)
		}
	}
	other := w.addLink(t, "alice", module, `{"url":"http://169.254.169.254/other"}`, http.StatusCreated)
	send(http.MethodPost, "/api/v0/modules/"+module.String()+"/links", `{"url":"http://169.254.169.254/other"}`, http.StatusConflict)
	send(http.MethodPatch, path, `{"url":"http://169.254.169.254/other"}`, http.StatusConflict)
	if got := w.moduleLinkCrawls(t, link.ID); len(got) != 3 {
		t.Fatalf("a refused change enqueued: %q", got)
	}
	if got := w.moduleLinkCrawls(t, other.ID); len(got) != 1 {
		t.Fatalf("the other link's crawls %q; want its creation's", got)
	}

	// A write rolled back after its body ran: neither the link nor its crawl.
	rolledBack := observedTx{tx: tx, pool: w.pool, seen: &seen, rollback: errors.New("rolled back")}
	route = newPlanningRoute(t, w.pool, rolledBack)
	send(http.MethodPost, "/api/v0/modules/"+module.String()+"/links", `{"url":"http://169.254.169.254/rolled-back"}`, http.StatusInternalServerError)
	send(http.MethodPatch, path, `{"url":"http://169.254.169.254/rolled-back"}`, http.StatusInternalServerError)
	var links, crawls int
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT (SELECT count(*) FROM module_links WHERE url = $1),
		(SELECT count(*) FROM river_job WHERE kind = 'planning.crawl_link' AND args->>'url' = $1)`, "http://169.254.169.254/rolled-back").Scan(
		&links, &crawls); err != nil || links != 0 || crawls != 0 {
		t.Fatalf("rolled back: %d links and %d crawls, %v; want none", links, crawls, err)
	}
}
