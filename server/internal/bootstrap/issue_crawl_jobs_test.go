package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Manual insertion isolates the production worker composition from the
// event subscriber, whose transaction contract is covered separately.
func TestNewAppCrawlJobRuns(t *testing.T) {
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
	item := w.newWorkItem(t)
	link := createLink(t, w, "bob", item, fixture.URL+"/page")
	beforeIssue, beforeActivity := issueRow(t, w, item), descriptionRows(t, w, "issue_activities", item)
	var beforeLink string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT (to_jsonb(l)-'metadata')::text FROM issue_links l WHERE id=$1", link.ID).Scan(&beforeLink); err != nil {
		t.Fatal(err)
	}
	inserter, err := jobs.NewInserter(w.pool)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := w.pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(t.Context()) }()
	if err := inserter.InsertTx(t.Context(), tx, riveradapter.CrawlLinkArgs{LinkID: link.ID, URL: link.URL}, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	for {
		var state, queue string
		var maxAttempts int
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT state,queue,max_attempts FROM river_job WHERE kind='issue.crawl_link' AND args->>'link_id'=$1 AND args->>'url'=$2`, link.ID.String(), link.URL).Scan(&state, &queue, &maxAttempts); err != nil {
			t.Fatal(err)
		}
		if queue != "default" || maxAttempts != 3 {
			t.Fatalf("crawl queue=%s attempts=%d; want default/3", queue, maxAttempts)
		}
		if state == "completed" {
			break
		}
		select {
		case <-tick.C:
		case <-deadline.C:
			t.Fatalf("crawl job state=%s; want completed", state)
		}
	}
	var metadata []byte
	var afterLink string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT metadata,(to_jsonb(l)-'metadata')::text FROM issue_links l WHERE id=$1", link.ID).Scan(&metadata, &afterLink); err != nil {
		t.Fatal(err)
	}
	var got domain.LinkMetadata
	if err := json.Unmarshal(metadata, &got); err != nil {
		t.Fatal(err)
	}
	title, icon, iconURL := "Nerve/"+buildinfo.Get().Version+" (+link preview)", "data:image/png;base64,aWNvbg==", fixture.URL+"/favicon.ico"
	want := domain.LinkMetadata{"title": &title, "url": &link.URL, "favicon": &icon, "favicon_url": &iconURL}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("composed crawl metadata=%s; want %+v", metadata, want)
	}
	if beforeLink != afterLink || beforeIssue != issueRow(t, w, item) || beforeActivity != descriptionRows(t, w, "issue_activities", item) {
		t.Fatal("composed crawl changed link audit, issue or activity")
	}
}
