package postgresadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

var (
	_ app.LinkReader  = (*postgresadapter.Store)(nil)
	_ app.LinkLister  = (*postgresadapter.Store)(nil)
	_ app.LinkCreator = (*postgresadapter.Store)(nil)
	_ app.LinkUpdater = (*postgresadapter.Store)(nil)
	_ app.LinkDeleter = (*postgresadapter.Store)(nil)
)

func TestLinkStore(t *testing.T) {
	t.Run("reads and audit", testLinkReads)
	t.Run("metadata and returning", testLinkUpdate)
	t.Run("long exact URLs and constraints", testLinkExactURLs)
	t.Run("hash collision", testLinkCollision)
	t.Run("errors preserve chain", testLinkErrors)
}

// Reverse insertion, tied creation times and a newer low id distinguish both
// descending keys; foreign and deleted rows distinguish the collection scope.
func testLinkReads(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "creator@corp.com")
	deleter := newAccount(t, pool, "deleter@corp.com")
	p := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	one, two := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, p, one, 1, by)
	newIssue(t, pool, p, two, 2, by)
	ids := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	slices.SortFunc(ids, uuid.UUID.Compare)
	ctx := context.Background()
	for _, n := range []int{4, 3, 2, 1, 0} {
		issue, at := one, now
		if n == 3 {
			issue = two
		}
		if n == 0 {
			at = now.Add(time.Minute)
		}
		link := domain.Link{ID: ids[n], WorkspaceID: p.workspace, ProjectID: p.id, IssueID: issue, URL: "HTTP://Example.com/" + ids[n].String()}
		if err := s.InsertLink(ctx, app.LinkWrite{Link: link, By: by, At: at}); err != nil {
			t.Fatal(err)
		}
	}
	deletedAt := now.Add(2 * time.Minute)
	if err := s.DeleteLink(ctx, app.LinkDeletion{ID: ids[4], By: deleter, At: deletedAt}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteLink(ctx, app.LinkDeletion{ID: ids[4], By: by, At: deletedAt.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Links(ctx, one)
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]uuid.UUID, len(got))
	for n, link := range got {
		actual[n] = link.ID
		at := now
		if link.ID == ids[0] {
			at = now.Add(time.Minute)
		}
		want := domain.Link{ID: link.ID, WorkspaceID: p.workspace, ProjectID: p.id, IssueID: one, URL: "HTTP://Example.com/" + link.ID.String(), Metadata: domain.LinkMetadata{}, CreatedByID: &by, CreatedAt: at, UpdatedAt: at}
		if !reflect.DeepEqual(link, want) {
			t.Fatalf("read link = %#v; want %#v", link, want)
		}
		read, found, err := s.GetLink(ctx, link.ID)
		if err != nil || !found || !reflect.DeepEqual(read, want) {
			t.Fatalf("get link = %#v, %v, %v; want %#v", read, found, err, want)
		}
	}
	if !slices.Equal(actual, []uuid.UUID{ids[0], ids[2], ids[1]}) {
		t.Fatalf("order = %v; want newer then descending tied ids", actual)
	}
	for _, id := range []uuid.UUID{ids[4], uuid.NewV7()} {
		link, found, err := s.GetLink(ctx, id)
		if err != nil || found || !reflect.DeepEqual(link, domain.Link{}) {
			t.Fatalf("missing = %#v, %v, %v", link, found, err)
		}
	}
	empty, err := s.Links(ctx, uuid.NewV7())
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty collection = %#v, %v", empty, err)
	}
	var at, updated time.Time
	var deletedBy uuid.UUID
	if err := pool.QueryRow(ctx, "SELECT deleted_at,updated_at,updated_by_id FROM issue_links WHERE id=$1", ids[4]).Scan(&at, &updated, &deletedBy); err != nil || !at.Equal(deletedAt) || !updated.Equal(deletedAt) || deletedBy != deleter {
		t.Fatalf("deletion audit = %v, %v, %v, %v", at, updated, deletedBy, err)
	}
}

// Writing stale app metadata on a title-only update loses a completed crawl;
// returning the input instead of UPDATE RETURNING reports stale values too.
func testLinkUpdate(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "creator@corp.com")
	updater := newAccount(t, pool, "updater@corp.com")
	p := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	issue, id := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, p, issue, 1, by)
	ctx := context.Background()
	link := domain.Link{ID: id, WorkspaceID: p.workspace, ProjectID: p.id, IssueID: issue, URL: "HTTP://Example.com/P?q=A", Metadata: domain.LinkMetadata{}}
	if err := s.InsertLink(ctx, app.LinkWrite{Link: link, By: by, At: now}); err != nil {
		t.Fatal(err)
	}
	exec(t, pool, "UPDATE issue_links SET metadata=$2 WHERE id=$1", id, `{"title":"Fresh crawler","favicon":null,"url":"HTTP://Example.com/P?q=A","favicon_url":null}`)
	title := "  User title  "
	link.Title = &title
	for n, url := range []string{link.URL, "http://Example.com/P?q=A"} {
		link.URL = url
		at := now.Add(time.Duration(n+1) * time.Minute)
		got, err := s.UpdateLink(ctx, app.LinkWrite{Link: link, By: updater, At: at})
		if err != nil {
			t.Fatal(err)
		}
		want := link
		want.CreatedByID, want.CreatedAt, want.UpdatedAt = &by, now, at
		want.Metadata = domain.LinkMetadata{}
		if n == 0 {
			fresh := "Fresh crawler"
			want.Metadata = domain.LinkMetadata{"title": &fresh, "favicon": nil, "url": &url, "favicon_url": nil}
		}
		var stored domain.Link
		var raw []byte
		var updatedBy uuid.UUID
		if err := pool.QueryRow(ctx, "SELECT id,workspace_id,project_id,issue_id,url,title,metadata,created_by_id,created_at,updated_at,updated_by_id FROM issue_links WHERE id=$1", id).Scan(&stored.ID, &stored.WorkspaceID, &stored.ProjectID, &stored.IssueID, &stored.URL, &stored.Title, &raw, &stored.CreatedByID, &stored.CreatedAt, &stored.UpdatedAt, &updatedBy); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &stored.Metadata); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, stored) || !reflect.DeepEqual(got, want) || updatedBy != updater {
			t.Fatalf("returned = %#v; stored = %#v; want %#v; by=%v", got, stored, want, updatedBy)
		}
	}
	link.Title = nil
	got, err := s.UpdateLink(ctx, app.LinkWrite{Link: link, By: updater, At: now.Add(3 * time.Minute)})
	if err != nil || got.Title != nil {
		t.Fatalf("clear title = %#v, %v", got, err)
	}
	exec(t, pool, "UPDATE issue_links SET created_by_id=NULL WHERE id=$1", id)
	got, found, err := s.GetLink(ctx, id)
	if err != nil || !found || got.CreatedByID != nil {
		t.Fatalf("nullable creator = %#v, %v, %v", got, found, err)
	}
}

// Storage must retain exact high-entropy values and the PG constraint chain,
// with no lowercase/hash-only duplicate checks or byte truncation.
func testLinkExactURLs(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "creator@corp.com")
	p := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	issue, other := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, p, issue, 1, by)
	newIssue(t, pool, p, other, 2, by)
	ctx := context.Background()
	long := longLinkURL()
	makeWrite := func(issue uuid.UUID, url string) app.LinkWrite {
		return app.LinkWrite{Link: domain.Link{ID: uuid.NewV7(), WorkspaceID: p.workspace, ProjectID: p.id, IssueID: issue, URL: url}, By: by, At: now}
	}
	w := makeWrite(issue, long)
	if err := s.InsertLink(ctx, w); err != nil {
		t.Fatal(err)
	}
	got, found, err := s.GetLink(ctx, w.Link.ID)
	if err != nil || !found || got.URL != long {
		t.Fatalf("insert long URL = %q, %v, %v", got.URL, found, err)
	}
	for _, tc := range []struct {
		issue, exclude uuid.UUID
		url            string
		want           bool
	}{
		{issue, uuid.UUID{}, long, true}, {issue, w.Link.ID, long, false}, {other, uuid.UUID{}, long, false},
	} {
		exists, err := s.LinkURLExists(ctx, tc.issue, tc.exclude, tc.url)
		if err != nil || exists != tc.want {
			t.Fatalf("URL exists = %v, %v; want %v", exists, err, tc.want)
		}
	}
	wantLinkConstraint(t, s.InsertLink(ctx, makeWrite(issue, long)), "23P01", "issue_links_issue_id_url_key")
	target := makeWrite(issue, "http://169.254.169.254/target")
	if err := s.InsertLink(ctx, target); err != nil {
		t.Fatal(err)
	}
	target.Link.URL = long
	_, err = s.UpdateLink(ctx, target)
	wantLinkConstraint(t, err, "23P01", "issue_links_issue_id_url_key")
	changed := string([]rune(long)[:2047]) + "x"
	target.Link.URL = changed
	got, err = s.UpdateLink(ctx, target)
	if err != nil || got.URL != changed {
		t.Fatalf("update long URL = %q, %v", got.URL, err)
	}
	got, found, err = s.GetLink(ctx, target.Link.ID)
	if err != nil || !found || got.URL != changed {
		t.Fatalf("reread long URL = %q, %v, %v", got.URL, found, err)
	}
	wantLinkConstraint(t, s.InsertLink(ctx, makeWrite(issue, long+"界")), "23514", "")
	target.Link.URL = long + "界"
	_, err = s.UpdateLink(ctx, target)
	wantLinkConstraint(t, err, "23514", "")
	if err := s.InsertLink(ctx, makeWrite(other, long)); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteLink(ctx, app.LinkDeletion{ID: w.Link.ID, By: by, At: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	exists, err := s.LinkURLExists(ctx, issue, uuid.UUID{}, long)
	if err != nil || exists {
		t.Fatalf("deleted URL exists = %v, %v", exists, err)
	}
	if err := s.InsertLink(ctx, makeWrite(issue, long)); err != nil {
		t.Fatal(err)
	}
	for _, url := range []string{"HTTP://Example.com/P", "http://Example.com/P"} {
		if err := s.InsertLink(ctx, makeWrite(issue, url)); err != nil {
			t.Fatal(err)
		}
	}
}

func testLinkCollision(t *testing.T) {
	var fixture struct {
		IssueID uuid.UUID `json:"issue_id"`
		URLs    []string  `json:"urls"`
		Hash    int32     `json:"hash"`
	}
	data, err := os.ReadFile("testdata/link_url_hash_collision.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.URLs) != 2 || fixture.URLs[0] == fixture.URLs[1] {
		t.Fatal("need distinct collision originals")
	}
	s, pool := newStore(t)
	by := newAccount(t, pool, "creator@corp.com")
	p := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	newIssue(t, pool, p, fixture.IssueID, 1, by)
	ctx := context.Background()
	for _, url := range fixture.URLs {
		var hash int32
		if err := pool.QueryRow(ctx, "SELECT hashtext($1::uuid::text || ':' || $2::text)", fixture.IssueID, url).Scan(&hash); err != nil || hash != fixture.Hash {
			t.Fatalf("collision hash = %d, %v; want %d", hash, err, fixture.Hash)
		}
	}
	id := uuid.NewV7()
	w := app.LinkWrite{Link: domain.Link{ID: id, WorkspaceID: p.workspace, ProjectID: p.id, IssueID: fixture.IssueID, URL: fixture.URLs[0]}, By: by, At: now}
	if err := s.InsertLink(ctx, w); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		url     string
		exclude uuid.UUID
		want    bool
	}{
		{fixture.URLs[1], uuid.UUID{}, false}, {fixture.URLs[0], uuid.UUID{}, true}, {fixture.URLs[0], id, false}, {fixture.URLs[1], id, false},
	} {
		got, err := s.LinkURLExists(ctx, fixture.IssueID, tc.exclude, tc.url)
		if err != nil || got != tc.want {
			t.Fatalf("collision URL exists (%s) = %v, %v; want %v", tc.url, got, err, tc.want)
		}
	}
	w.Link.ID, w.Link.URL = uuid.NewV7(), fixture.URLs[1]
	if err := s.InsertLink(ctx, w); err != nil {
		t.Fatal(err)
	}
	links, err := s.Links(ctx, fixture.IssueID)
	if err != nil || len(links) != 2 {
		t.Fatalf("collision rows = %#v, %v", links, err)
	}
	urls := []string{links[0].URL, links[1].URL}
	slices.Sort(urls)
	if !slices.Equal(urls, fixture.URLs) {
		t.Fatalf("collision URLs = %v; want %v", urls, fixture.URLs)
	}
}

func testLinkErrors(t *testing.T) {
	s, _ := newStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, readErr := s.GetLink(ctx, uuid.NewV7())
	_, listErr := s.Links(ctx, uuid.NewV7())
	_, existsErr := s.LinkURLExists(ctx, uuid.NewV7(), uuid.UUID{}, "http://example.com")
	insertErr := s.InsertLink(ctx, app.LinkWrite{})
	_, updateErr := s.UpdateLink(ctx, app.LinkWrite{})
	deleteErr := s.DeleteLink(ctx, app.LinkDeletion{})
	for _, err := range []error{readErr, listErr, existsErr, insertErr, updateErr, deleteErr} {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error chain = %v; want context.Canceled", err)
		}
	}
}
