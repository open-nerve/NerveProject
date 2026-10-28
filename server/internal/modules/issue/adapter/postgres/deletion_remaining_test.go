package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestDeletionRemainingUsesCurrentCursor(t *testing.T) {
	order := []app.RowStep{"draft_issues", "issues", "issue_assignees", "issue_labels", "issue_relations", "issue_links", "issue_subscribers", "issue_mentions", "issue_comments", "issue_reactions", "comment_reactions", "issue_activities", "issue_description_versions"}
	if !slices.Equal(app.DeletionRowSteps(), order) {
		t.Fatalf("steps=%v", app.DeletionRowSteps())
	}
	for i, step := range order {
		t.Run(string(step), func(t *testing.T) {
			s, pool := newStore(t)
			by := newAccount(t, pool, "remaining@corp.com")
			p := newProject(t, pool, newWorkspace(t, pool, "remaining"), "P")
			item := uuid.NewV7()
			newIssue(t, pool, p, item, 1, by)
			id := uuid.NewV7()
			for _, statement := range batchStatements {
				if statement.table == string(step) {
					statement.insert(t, pool, p, item, id, by, 1)
				}
			}
			// Remove fixture dependencies as candidates, without removing the graph.
			exec(t, pool, "UPDATE issues SET deleted_at=now() WHERE id<>$1", id)
			exec(t, pool, "UPDATE issue_comments SET deleted_at=now() WHERE id<>$1", id)
			check := func(from app.RowStep, after uuid.UUID, want bool) {
				t.Helper()
				got, err := s.ProjectDeletionRemaining(pgtest.Soon(t), app.ProjectRemainder{ProjectID: p.id, Step: from, After: after})
				if err != nil || got != want {
					t.Fatalf("from=%s after=%s got=%t/%v want=%t", from, after, got, err, want)
				}
			}
			check(step, uuid.UUID{}, true)
			check(step, id, false)
			if i > 0 {
				check(order[i-1], id, true)
			} // Later tables must restart from zero.
			if i+1 < len(order) {
				check(order[i+1], uuid.UUID{}, false)
			} // Finished tables stay finished.
			exec(t, pool, "UPDATE "+string(step)+" SET deleted_at=now() WHERE id=$1", id)
			check(step, uuid.UUID{}, false)
			cancelled, cancel := context.WithCancel(t.Context())
			cancel()
			if _, err := s.ProjectDeletionRemaining(cancelled, app.ProjectRemainder{ProjectID: p.id, Step: step}); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
	t.Run("relations both directions and deleted issues", func(t *testing.T) {
		s, pool := newStore(t)
		by := newAccount(t, pool, "relation@corp.com")
		ws := newWorkspace(t, pool, "relation")
		p, q := newProject(t, pool, ws, "P"), newProject(t, pool, ws, "Q")
		a, b := uuid.NewV7(), uuid.NewV7()
		newIssue(t, pool, p, a, 1, by)
		newIssue(t, pool, q, b, 1, by)
		r := uuid.NewV7()
		newRelation(t, pool, q, b, a, r, "relates_to", by)
		exec(t, pool, "UPDATE issues SET deleted_at=now()")
		// Earlier ownership changes writes, never the existence of a candidate.
		exec(t, pool, "UPDATE projects SET deleted_at=now() WHERE id=$1", q.id)
		for _, id := range []uuid.UUID{p.id, q.id} {
			got, err := s.ProjectDeletionRemaining(pgtest.Soon(t), app.ProjectRemainder{ProjectID: id, Step: "issue_relations"})
			if err != nil || !got {
				t.Fatalf("candidate=%t/%v", got, err)
			}
		}
	})
	t.Run("workspace drafts", func(t *testing.T) {
		s, pool := newStore(t)
		by := newAccount(t, pool, "draft@corp.com")
		ws, other := newWorkspace(t, pool, "draft"), newWorkspace(t, pool, "other")
		p := newProject(t, pool, ws, "P")
		id := uuid.NewV7()
		exec(t, pool, "INSERT INTO draft_issues(id,workspace_id,created_by_id) VALUES($1,$2,$3)", id, ws, by)
		exec(t, pool, "INSERT INTO draft_issues(id,workspace_id,project_id,created_by_id) VALUES($1,$2,$3,$4)", uuid.NewV7(), ws, p.id, by)
		exec(t, pool, "INSERT INTO draft_issues(id,workspace_id,created_by_id) VALUES($1,$2,$3)", uuid.NewV7(), other, by)
		for _, after := range []uuid.UUID{{}, id} {
			got, err := s.WorkspaceDraftDeletionRemaining(pgtest.Soon(t), app.WorkspaceDraftBatch{WorkspaceID: ws, After: after})
			if err != nil || got != (after == uuid.UUID{}) {
				t.Fatalf("candidate=%t/%v", got, err)
			}
		}
		exec(t, pool, "UPDATE draft_issues SET deleted_at=now() WHERE id=$1", id)
		if got, err := s.WorkspaceDraftDeletionRemaining(pgtest.Soon(t), app.WorkspaceDraftBatch{WorkspaceID: ws}); err != nil || got {
			t.Fatalf("project/other drafts counted: %t/%v", got, err)
		}
	})
}
