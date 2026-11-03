package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func contentAssetState(t *testing.T, w issueWorld) map[string]string {
	t.Helper()
	out := collaborationState(t, w)
	for _, table := range []string{"projects", "file_assets"} {
		var rows string
		if err := w.pool.QueryRow(t.Context(), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]'::jsonb)::text FROM "+table+" r").Scan(&rows); err != nil {
			t.Fatal(err)
		}
		out[table] = rows
	}
	return out
}

func TestIssueContentAssetsRollback(t *testing.T) {
	for _, stage := range []string{"body", "version", "asset", "commit", "subscriber"} {
		t.Run(stage, func(t *testing.T) {
			w := newIssueWorld(t)
			id := w.newWorkItem(t)
			img := uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", attachmentScope("issue", id), "body.png", "image/png", assetTestPNG(t))
			before := contentAssetState(t, w)
			body := fmt.Sprintf(`{"description_html":%q}`, issueImageHTML(img.Asset.ID)+mentionHTML(w.ids["alice"]))
			path := "/api/v0/issues/" + id.String()
			if stage == "subscriber" {
				var received []shared.Event
				fault := errors.New("content subscriber fault")
				route := issueEventRouteFor(t, w, clock.System{}, &received, &fault, []string{"issue.updated"}, func(ctx context.Context, e shared.Event) error {
					row, err := assetpg.New(w.pool).FindAsset(ctx, img.Asset.ID)
					if err != nil {
						return err
					}
					if row.BoundAt == nil || !row.BoundAt.Equal(e.OccurredAt) {
						t.Error("event preceded binding or used another time")
					}
					outside, err := assetpg.New(w.pool).FindAsset(t.Context(), img.Asset.ID)
					if err != nil {
						return err
					}
					if outside.BoundAt != nil {
						t.Error("binding visible before commit")
					}
					return nil
				})
				req, res := route.send("PATCH", path, w.ids["bob"], body)
				w.contract.CheckResponse(t, req, res.Result())
				if res.Code != 500 || len(received) != 1 || issueEventCount(t, w) != 0 {
					t.Fatalf("subscriber failure not atomic: %d %v", res.Code, received)
				}
			} else {
				table, operation := "issues", "UPDATE"
				switch stage {
				case "version":
					table = "issue_description_versions"
					operation = "INSERT"
				case "asset", "commit":
					table = "file_assets"
				}
				trigger := "CREATE TRIGGER reject_content AFTER " + operation + " ON " + table + " FOR EACH ROW EXECUTE FUNCTION reject_content()"
				if stage == "commit" {
					trigger = "CREATE CONSTRAINT TRIGGER reject_content AFTER UPDATE ON file_assets DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_content()"
				}
				if _, err := w.pool.Exec(t.Context(), `CREATE SEQUENCE content_fault_seen; CREATE FUNCTION reject_content() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM nextval('content_fault_seen'); RAISE EXCEPTION 'content write rejected'; END $$;`+trigger); err != nil {
					t.Fatal(err)
				}
				w.send(t, "bob", "PATCH", path, body, 500)
				var reached bool
				if err := w.pool.QueryRow(t.Context(), "SELECT is_called FROM content_fault_seen").Scan(&reached); err != nil || !reached {
					t.Fatal("fault statement was not reached", err)
				}
				if _, err := w.pool.Exec(t.Context(), "DROP TRIGGER reject_content ON "+table); err != nil {
					t.Fatal(err)
				}
			}
			if after := contentAssetState(t, w); !reflect.DeepEqual(before, after) {
				for table, rows := range before {
					if rows != after[table] {
						t.Error("rollback left changed table", table)
					}
				}
			}
			w.send(t, "bob", "PATCH", path, body, 200)
			bound, err := assetpg.New(w.pool).FindAsset(t.Context(), img.Asset.ID)
			if err != nil || bound.BoundAt == nil {
				t.Fatal("retry lost binding", err)
			}
		})
	}
}

func TestIssueContentAssetsCreateSequenceRollback(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	img := uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", attachmentScope("issue", id), "wrong-host.png", "image/png", assetTestPNG(t))
	before := contentAssetState(t, w)
	w.send(t, "bob", "POST", "/api/v0/projects/"+w.web.String()+"/issues", fmt.Sprintf(`{"name":"Rejected","description_html":%q}`, issueImageHTML(img.Asset.ID)), 404)
	if after := contentAssetState(t, w); !reflect.DeepEqual(before, after) {
		t.Fatal("rejected reference consumed sequence or wrote part of creation graph")
	}
	created := w.newWorkItemBy(t, "bob", w.web, `{"name":"After refusal"}`)
	var sequence int
	if err := w.pool.QueryRow(t.Context(), "SELECT sequence_id FROM issues WHERE id=$1", created).Scan(&sequence); err != nil || sequence != 2 {
		t.Fatal("sequence did not roll back", sequence, err)
	}
	// A late count failure must also roll back the otherwise complete new graph.
	if _, err := w.pool.Exec(t.Context(), "ALTER TABLE file_assets RENAME TO missing_creation_count"); err != nil {
		t.Fatal(err)
	}
	var old string
	if err := w.pool.QueryRow(t.Context(), "SELECT to_jsonb(p)::text FROM projects p WHERE id=$1", w.web).Scan(&old); err != nil {
		t.Fatal(err)
	}
	w.send(t, "bob", "POST", "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"Count failure"}`, 500)
	var now string
	if err := w.pool.QueryRow(t.Context(), "SELECT to_jsonb(p)::text FROM projects p WHERE id=$1", w.web).Scan(&now); err != nil || old != now {
		t.Fatal("late count failure consumed sequence", err)
	}
	if rows := queryIDs(t, w.pool, "SELECT id FROM issues WHERE name=$1", "Count failure"); len(rows) != 0 {
		t.Fatal("late count failure created an issue")
	}
}

func TestIssueContentAssetsUnchangedLegacyReference(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	img := uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", attachmentScope("issue", id), "legacy.png", "image/png", assetTestPNG(t))
	html := issueImageHTML(img.Asset.ID)
	// Only a historical fixture can have a stored internal image without binding.
	if _, err := w.pool.Exec(t.Context(), "UPDATE issues SET description_html=$2 WHERE id=$1", id, html); err != nil {
		t.Fatal(err)
	}
	old := collaborationState(t, w)
	saveIssueImage(t, w, "bob", id, html, false, 200)
	if after := collaborationState(t, w); !reflect.DeepEqual(old, after) {
		t.Fatal("unchanged content wrote an issue version or audit")
	}
	row, err := assetpg.New(w.pool).FindAsset(t.Context(), img.Asset.ID)
	if err != nil || row.BoundAt == nil {
		t.Fatal("unchanged reference bypassed binding", err)
	}
	// Validate the unchanged reference even for another author; an unbound
	// preparation is private, despite a legacy body already containing its ID.
	other := uploadedAssetHTTP(t, w, "alice", "ISSUE_DESCRIPTION", attachmentScope("issue", id), "private.png", "image/png", assetTestPNG(t))
	html = issueImageHTML(other.Asset.ID)
	if _, err := w.pool.Exec(t.Context(), "UPDATE issues SET description_html=$2 WHERE id=$1", id, html); err != nil {
		t.Fatal(err)
	}
	oldAll := contentAssetState(t, w)
	saveIssueImage(t, w, "bob", id, html, false, 404)
	if !reflect.DeepEqual(oldAll, contentAssetState(t, w)) {
		t.Fatal("unchanged private reference wrote state")
	}
}
