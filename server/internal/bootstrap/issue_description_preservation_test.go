package bootstrap

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"uuid"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestExistingIssueWritesPreserveDescription(t *testing.T) {
	w := newIssueWorld(t)
	ctx := pgtest.Soon(t)
	neighbor := w.newWorkItem(t)
	// This old API must explicitly create both defaults, including non-NULL text.
	var html string
	var text *string
	if err := w.pool.QueryRow(ctx, "SELECT description_html,description_stripped FROM issues WHERE id=$1", neighbor).Scan(&html, &text); err != nil {
		t.Fatal(err)
	}
	if html != "<p></p>" || text == nil || *text != "" {
		t.Fatalf("old API defaults: html=%q stripped=%v", html, text)
	}
	store := issuepg.New(w.pool)
	if err := store.WriteDescription(ctx, issueapp.DescriptionWrite{IssueID: neighbor, HTML: "<p>neighbor</p>", Text: "neighbor"}); err != nil {
		t.Fatal(err)
	}
	var beforeNeighbor string
	if err := w.pool.QueryRow(ctx, "SELECT row_to_json(i)::text FROM issues i WHERE id=$1", neighbor).Scan(&beforeNeighbor); err != nil {
		t.Fatal(err)
	}
	empty, target := "", "目标"
	for _, tt := range []struct {
		name, html string
		text       *string
	}{
		{"non-default", "<p>目标</p>", &target}, {"legacy NULL", "<p>legacy</p>", nil}, {"explicit empty", "", &empty},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id := w.newWorkItem(t)
			// NULL is an existing persisted state, distinct from an explicit empty save.
			if tt.text == nil {
				if _, err := w.pool.Exec(ctx, "UPDATE issues SET description_html=$2,description_stripped=NULL WHERE id=$1", id, tt.html); err != nil {
					t.Fatal(err)
				}
			} else if err := store.WriteDescription(ctx, issueapp.DescriptionWrite{IssueID: id, HTML: tt.html, Text: *tt.text}); err != nil {
				t.Fatal(err)
			}
			stamps := w.workItemStamps(t, neighbor)
			w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+id.String(), `{"name":"renamed","priority":"low"}`, http.StatusOK)
			assertStoredDescription(t, ctx, w, id, tt.html, tt.text)
			got, found, err := store.GetIssue(ctx, id)
			if err != nil || !found || got.DescriptionHTML != tt.html || !reflect.DeepEqual(got.DescriptionStripped, tt.text) {
				t.Fatal("read-back description lost", got, err)
			}
			var afterNeighbor string
			if err := w.pool.QueryRow(ctx, "SELECT row_to_json(i)::text FROM issues i WHERE id=$1", neighbor).Scan(&afterNeighbor); err != nil {
				t.Fatal(err)
			}
			if afterNeighbor != beforeNeighbor || !reflect.DeepEqual(stamps, w.workItemStamps(t, neighbor)) {
				t.Fatal("ordinary PATCH changed neighbor")
			}
		})
	}
}

func assertStoredDescription(t *testing.T, ctx context.Context, w issueWorld, id uuid.UUID, wantHTML string, wantText *string) {
	t.Helper()
	var html string
	var text *string
	if err := w.pool.QueryRow(ctx, "SELECT description_html,description_stripped FROM issues WHERE id=$1", id).Scan(&html, &text); err != nil {
		t.Fatal(err)
	}
	if html != wantHTML || !reflect.DeepEqual(text, wantText) {
		t.Fatalf("persisted description changed: html=%q stripped=%v; want html=%q stripped=%v", html, text, wantHTML, wantText)
	}
}
