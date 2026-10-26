package postgresadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

var _ app.LinkMetadataWriter = (*postgresadapter.Store)(nil)

// Each excluded row can otherwise match the task URL. A whole-row snapshot
// catches changes to title, ownership, audit stamps or the tombstone.
func TestLinkMetadataGuard(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "creator@corp.com")
	updater := newAccount(t, pool, "updater@corp.com")
	p := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	item := uuid.NewV7()
	newIssue(t, pool, p, item, 1, by)
	live, deleted, replacement := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	const oldURL = "HTTP://Example.com/A"
	const newURL = "http://Example.com/B"
	title := "User title"
	for _, id := range []uuid.UUID{live, deleted} {
		w := app.LinkWrite{Link: domain.Link{ID: id, WorkspaceID: p.workspace, ProjectID: p.id, IssueID: item, URL: oldURL, Title: &title}, By: by, At: now}
		if id == live {
			w.Link.URL = newURL
		}
		if err := s.InsertLink(t.Context(), w); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.DeleteLink(t.Context(), app.LinkDeletion{ID: deleted, By: updater, At: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertLink(t.Context(), app.LinkWrite{Link: domain.Link{ID: replacement, WorkspaceID: p.workspace, ProjectID: p.id, IssueID: item, URL: oldURL}, By: by, At: now}); err != nil {
		t.Fatal(err)
	}
	exec(t, pool, "UPDATE issue_links SET updated_at=$1,updated_by_id=$2 WHERE id=$3", now.Add(2*time.Minute), updater, live)
	exec(t, pool, "INSERT INTO issue_activities (id,workspace_id,project_id,issue_id,verb,comment,created_at,updated_at) VALUES ($1,$2,$3,$4,'created','system',$5,$5)", uuid.NewV7(), p.workspace, p.id, item, now)
	// A deleted project with an as-yet live link is an allowed write window.
	exec(t, pool, "UPDATE projects SET deleted_at=$1 WHERE id=$2", now, p.id)
	snapshot := func() string {
		var got string
		err := pool.QueryRow(t.Context(), `SELECT jsonb_build_object(
   'links',(SELECT jsonb_agg(to_jsonb(l)-'metadata' ORDER BY id) FROM issue_links l),
   'activities',(SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM issue_activities a),
   'issues',(SELECT jsonb_agg(to_jsonb(i) ORDER BY id) FROM issues i))::text`).Scan(&got)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	before := snapshot()
	crawledTitle, resultURL := "Fetched title", newURL
	metadata := domain.LinkMetadata{"title": &crawledTitle, "url": &resultURL, "favicon": nil, "favicon_url": nil}
	for _, tc := range []struct {
		name string
		id   uuid.UUID
		url  string
		want bool
	}{
		{"live id and exact URL", live, newURL, true},
		{"old URL", live, oldURL, false},
		{"case differs", live, "HTTP://Example.com/B", false},
		{"deleted old id with live replacement", deleted, oldURL, false},
		{"another id cannot reach replacement", uuid.NewV7(), oldURL, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed, err := s.SetLinkMetadata(t.Context(), tc.id, tc.url, metadata)
			if err != nil || changed != tc.want {
				t.Fatalf("SetLinkMetadata changed=%v err=%v; want %v", changed, err, tc.want)
			}
			if got := snapshot(); got != before {
				t.Fatalf("metadata write changed title/audit/tombstone/issue/activity: %s", got)
			}
			for _, id := range []uuid.UUID{live, deleted, replacement} {
				var data []byte
				if err := pool.QueryRow(t.Context(), "SELECT metadata FROM issue_links WHERE id=$1", id).Scan(&data); err != nil {
					t.Fatal(err)
				}
				var got domain.LinkMetadata
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatal(err)
				}
				want := domain.LinkMetadata{}
				if id == live {
					want = metadata
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("row %s metadata=%#v; want %#v", id, got, want)
				}
			}
		})
	}
	t.Run("database error retains chain", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		changed, err := s.SetLinkMetadata(ctx, live, newURL, metadata)
		if changed || !errors.Is(err, context.Canceled) {
			t.Fatalf("DB error changed=%v err=%v; want context.Canceled", changed, err)
		}
		if snapshot() != before {
			t.Fatal("failed write changed audit rows")
		}
	})
}
