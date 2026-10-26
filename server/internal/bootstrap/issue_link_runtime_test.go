package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The same restricted login serves CRUD, inserts request jobs and runs the
// crawler's metadata UPDATE. The owner only supplies fixtures and observes.
func TestIssueLinkRuntimeJob(t *testing.T) {
	roles := newSplitRoles(t)
	t.Run("unlisted relation stays private after grants rerun", func(t *testing.T) {
		assertRuntimeUnlistedTable(t, roles)
	})
	var user string
	var superuser, bypassRLS, ownsLinks bool
	if err := roles.server.QueryRow(pgtest.Soon(t), `SELECT current_user,r.rolsuper,r.rolbypassrls,c.relowner=r.oid
		FROM pg_roles r,pg_class c WHERE r.rolname=current_user AND c.oid='issue_links'::regclass`).Scan(&user, &superuser, &bypassRLS, &ownsLinks); err != nil {
		t.Fatal(err)
	}
	if user != roles.serverName || superuser || bypassRLS || ownsLinks {
		t.Fatalf("runtime login=%s superuser=%v bypassRLS=%v ownsLinks=%v; want restricted %s", user, superuser, bypassRLS, ownsLinks, roles.serverName)
	}
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("runtime icon"))
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, "<title>runtime %s</title>", r.URL.Path)
	}))
	defer fixture.Close()
	w := newIssueWorldOn(t, roles.serverURL, roles.owner, func(cfg *config.Config) {
		cfg.Outbound.AllowCIDRs = []string{"127.0.0.1/32"}
	})
	item := w.newWorkItem(t)
	link := createLink(t, w, "bob", item, fixture.URL+"/first")
	if link.Metadata == nil || len(link.Metadata) != 0 {
		t.Fatalf("created link metadata=%+v; want pending {}", link.Metadata)
	}
	assertPreview := func(link answeredLink, previewTitle string) {
		t.Helper()
		want := domain.LinkMetadata{"title": eventValue(previewTitle), "url": &link.URL,
			"favicon": eventValue("data:image/png;base64,cnVudGltZSBpY29u"), "favicon_url": eventValue(fixture.URL + "/favicon.ico")}
		var raw []byte
		if err := roles.owner.QueryRow(pgtest.Soon(t), "SELECT metadata FROM issue_links WHERE id=$1", link.ID).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var stored domain.LinkMetadata
		if err := json.Unmarshal(raw, &stored); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(stored, want) {
			t.Fatalf("runtime worker metadata=%s; want %+v", raw, want)
		}
		listed := listIn[answeredLink](t, w.send(t, "alice", http.MethodGet, linksOf(item), "", http.StatusOK))
		if len(listed) != 1 || listed[0].ID != link.ID || listed[0].URL != link.URL || !reflect.DeepEqual(listed[0].Title, link.Title) || !reflect.DeepEqual(listed[0].Metadata, want) {
			t.Fatalf("runtime HTTP link=%+v; want id/url/title and completed DB metadata %+v", listed, link)
		}
	}
	waitUniqueLinkCrawl(t, w, link)
	assertPreview(link, "runtime /first")
	var titled answeredLink
	decodeAnswer(t, w.send(t, "alice", http.MethodPatch, linkPath(link.ID), `{"title":"Runtime title"}`, http.StatusOK), &titled)
	if titled.Title == nil || *titled.Title != "Runtime title" {
		t.Fatalf("runtime title update=%+v", titled)
	}
	assertPreview(titled, "runtime /first")
	if !reflect.DeepEqual(titled.Metadata, domain.LinkMetadata{"title": eventValue("runtime /first"), "url": &link.URL,
		"favicon": eventValue("data:image/png;base64,cnVudGltZSBpY29u"), "favicon_url": eventValue(fixture.URL + "/favicon.ico")}) {
		t.Fatalf("runtime title response lost completed metadata: %+v", titled)
	}
	var jobs int
	if err := roles.owner.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind='issue.crawl_link' AND args->>'link_id'=$1", link.ID.String()).Scan(&jobs); err != nil || jobs != 1 {
		t.Fatalf("title-only jobs=%d error=%v; want original one", jobs, err)
	}
	var changed answeredLink
	decodeAnswer(t, w.send(t, "alice", http.MethodPatch, linkPath(link.ID), linkBody(fixture.URL+"/second", titled.Title), http.StatusOK), &changed)
	if changed.URL != fixture.URL+"/second" || changed.Metadata == nil || len(changed.Metadata) != 0 {
		t.Fatalf("runtime URL update=%+v; want changed URL and pending {}", changed)
	}
	waitUniqueLinkCrawl(t, w, changed)
	assertPreview(changed, "runtime /second")
	w.send(t, "alice", http.MethodDelete, linkPath(link.ID), "", http.StatusNoContent)
	var deleted bool
	if err := roles.owner.QueryRow(pgtest.Soon(t), "SELECT deleted_at IS NOT NULL FROM issue_links WHERE id=$1", link.ID).Scan(&deleted); err != nil || !deleted {
		t.Fatalf("runtime delete=%v error=%v; want soft-deleted", deleted, err)
	}
	if listed := listIn[answeredLink](t, w.send(t, "alice", http.MethodGet, linksOf(item), "", http.StatusOK)); len(listed) != 0 {
		t.Fatalf("runtime list after delete=%+v; want empty", listed)
	}
	if err := roles.owner.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind='issue.crawl_link' AND args->>'link_id'=$1", link.ID.String()).Scan(&jobs); err != nil || jobs != 2 {
		t.Fatalf("runtime lifecycle jobs=%d error=%v; want only first/second URL jobs", jobs, err)
	}
}

func assertRuntimeUnlistedTable(t *testing.T, roles splitRoles) {
	t.Helper()
	for _, sql := range []string{"CREATE TABLE runtime_scratch (id integer PRIMARY KEY, note text)",
		"INSERT INTO runtime_scratch VALUES (1,'fixture')", "UPDATE runtime_scratch SET note='owner can write' WHERE id=1"} {
		if _, err := roles.owner.Exec(pgtest.Soon(t), sql); err != nil {
			t.Fatal(err)
		}
	}
	// Rerun the unchanged file after creating the table: ON ALL TABLES would
	// accidentally grant access here, unlike the explicit deployment list.
	applyRuntimeGrants(t, roles.owner)
	for _, sql := range []string{"SELECT * FROM runtime_scratch", "INSERT INTO runtime_scratch VALUES (2,'forbidden')",
		"UPDATE runtime_scratch SET note='forbidden' WHERE id=1", "DELETE FROM runtime_scratch WHERE id=1"} {
		_, err := roles.server.Exec(pgtest.Soon(t), sql)
		var denied *pgconn.PgError
		if !errors.As(err, &denied) || denied.Code != "42501" {
			t.Errorf("runtime unlisted table: %s returned %v; want permission denied (42501)", sql, err)
		}
	}
}
