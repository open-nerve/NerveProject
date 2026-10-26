package postgresadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A raw-URL B-tree cannot store this high-entropy value on an 8KiB page.
func longLinkURL() string {
	r := rand.New(rand.NewPCG(71, 27))
	url := []rune("http://169.254.169.254/")
	for len(url) < 2048 {
		url = append(url, rune(0x10000+r.IntN(0x100000)))
	}
	return string(url)
}

func insertLink(t *testing.T, pool *pgxpool.Pool, p project, issue, id, by uuid.UUID, url string) {
	t.Helper()
	exec(t, pool, `INSERT INTO issue_links(id,workspace_id,project_id,issue_id,url,created_by_id,updated_by_id,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$6,$7,$7)`, id, p.workspace, p.id, issue, url, by, now)
}

func wantLinkConstraint(t *testing.T, err error, code, constraint string) {
	t.Helper()
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != code || constraint != "" && pg.ConstraintName != constraint {
		t.Fatalf("constraint error = %v; want %s %s", err, code, constraint)
	}
}

// Missing checks, byte-based length limits and hash-only uniqueness all
// break values accepted or rejected through the real production migration.
func TestLinkSchema(t *testing.T) {
	_, pool := newStore(t)
	by := newAccount(t, pool, "creator@corp.com")
	p := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	issue, other := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, p, issue, 1, by)
	newIssue(t, pool, p, other, 2, by)
	ctx := context.Background()

	var columns []string
	if err := pool.QueryRow(ctx, `SELECT array_agg(column_name::text ORDER BY ordinal_position)
		FROM information_schema.columns WHERE table_schema='public' AND table_name='issue_links'`).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if want := []string{"id", "workspace_id", "project_id", "issue_id", "title", "url", "metadata", "created_by_id", "updated_by_id", "created_at", "updated_at", "deleted_at"}; !slices.Equal(columns, want) {
		t.Fatalf("link columns = %v; want %v", columns, want)
	}
	long := longLinkURL()
	if utf8.RuneCountInString(long) != 2048 || len(long) <= 2704 {
		t.Fatalf("long URL fixture: %d characters, %d bytes", utf8.RuneCountInString(long), len(long))
	}
	id, target := uuid.NewV7(), uuid.NewV7()
	insertLink(t, pool, p, issue, id, by, long)
	insertLink(t, pool, p, issue, target, by, "http://169.254.169.254/short")
	readURL := func(id uuid.UUID, want string) {
		t.Helper()
		var got string
		if err := pool.QueryRow(ctx, "SELECT url FROM issue_links WHERE id=$1", id).Scan(&got); err != nil || got != want {
			t.Fatalf("URL round trip = %q, %v; want exact input (%d bytes)", got, err, len(want))
		}
	}
	readURL(id, long)
	updated := strings.TrimSuffix(long, string([]rune(long)[2047])) + "x"
	exec(t, pool, "UPDATE issue_links SET url=$2 WHERE id=$1", target, updated)
	readURL(target, updated)
	_, err := pool.Exec(ctx, `INSERT INTO issue_links(id,workspace_id,project_id,issue_id,url) VALUES($1,$2,$3,$4,$5)`, uuid.NewV7(), p.workspace, p.id, issue, long)
	wantLinkConstraint(t, err, "23P01", "issue_links_issue_id_url_key")
	_, err = pool.Exec(ctx, "UPDATE issue_links SET url=$2 WHERE id=$1", target, long)
	wantLinkConstraint(t, err, "23P01", "issue_links_issue_id_url_key")
	insertLink(t, pool, p, other, uuid.NewV7(), by, long)
	exec(t, pool, "UPDATE issue_links SET deleted_at=now() WHERE id=$1", id)
	insertLink(t, pool, p, issue, uuid.NewV7(), by, long)
	insertLink(t, pool, p, issue, uuid.NewV7(), by, "HTTP://169.254.169.254/Case")
	insertLink(t, pool, p, issue, uuid.NewV7(), by, "http://169.254.169.254/Case")

	for name, url := range map[string]string{"empty": "", "2049 characters": long + "x"} {
		t.Run(name, func(t *testing.T) {
			_, err := pool.Exec(ctx, `INSERT INTO issue_links(id,workspace_id,project_id,issue_id,url) VALUES($1,$2,$3,$4,$5)`, uuid.NewV7(), p.workspace, p.id, issue, url)
			wantLinkConstraint(t, err, "23514", "")
			_, err = pool.Exec(ctx, "UPDATE issue_links SET url=$2 WHERE id=$1", target, url)
			wantLinkConstraint(t, err, "23514", "")
		})
	}
	for _, metadata := range []string{`[]`, `null`, `"text"`, `{"unknown":"x"}`, `{"title":17}`, `{"error":false}`, `{"favicon":{}}`, `{"url":[]}`} {
		t.Run(metadata, func(t *testing.T) {
			_, err := pool.Exec(ctx, "UPDATE issue_links SET metadata=$2 WHERE id=$1", target, metadata)
			wantLinkConstraint(t, err, "23514", "")
			_, err = pool.Exec(ctx, `INSERT INTO issue_links(id,workspace_id,project_id,issue_id,url,metadata) VALUES($1,$2,$3,$4,'http://169.254.169.254/invalid',$5)`, uuid.NewV7(), p.workspace, p.id, issue, metadata)
			wantLinkConstraint(t, err, "23514", "")
		})
	}
	exec(t, pool, "UPDATE issue_links SET metadata=$2 WHERE id=$1", target, `{"title":"Title","favicon":null,"url":"url","favicon_url":"icon","error":null}`)
	var metadata string
	var title *string
	if err := pool.QueryRow(ctx, "SELECT title,metadata::text FROM issue_links WHERE id=$1", id).Scan(&title, &metadata); err != nil || title != nil || metadata != "{}" {
		t.Fatalf("defaults = %v %s, %v; want NULL and {}", title, metadata, err)
	}
}

func TestLinkHashCollision(t *testing.T) {
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
		t.Fatal("need two distinct fixed collision inputs")
	}
	_, pool := newStore(t)
	by := newAccount(t, pool, "creator@corp.com")
	p := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	newIssue(t, pool, p, fixture.IssueID, 1, by)
	for _, url := range fixture.URLs {
		var hash int32
		if err := pool.QueryRow(context.Background(), "SELECT hashtext($1::uuid::text || ':' || $2::text)", fixture.IssueID, url).Scan(&hash); err != nil || hash != fixture.Hash {
			t.Fatalf("fixed collision hash = %d, %v; want %d", hash, err, fixture.Hash)
		}
	}
	for _, url := range fixture.URLs {
		insertLink(t, pool, p, fixture.IssueID, uuid.NewV7(), by, url)
	}
	var urls []string
	if err := pool.QueryRow(context.Background(), "SELECT array_agg(url ORDER BY url) FROM issue_links WHERE issue_id=$1 AND deleted_at IS NULL", fixture.IssueID).Scan(&urls); err != nil || !slices.Equal(urls, fixture.URLs) {
		t.Fatalf("collision rows = %v, %v; want both original URLs %v", urls, err, fixture.URLs)
	}
}
