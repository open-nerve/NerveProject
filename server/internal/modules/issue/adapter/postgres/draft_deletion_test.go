package postgresadapter_test

import (
	"context"
	"maps"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// Missing workspace batches, selecting project drafts in that batch, and
// revisiting deleted rows violate these independent audit expectations.
func TestDraftDeletionRows(t *testing.T) {
	for _, route := range []string{"project then workspace", "workspace batch", "zero projects", "rollback project", "rollback workspace"} {
		t.Run(route, func(t *testing.T) {
			s, pool := newStore(t)
			creator, actor := newAccount(t, pool, "creator@corp.com"), newAccount(t, pool, "actor@corp.com")
			workspace, other := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "other")
			before, at := now.Add(-time.Hour), now.Add(time.Hour)
			var projectID, oldProjectID *uuid.UUID
			if route != "zero projects" {
				current, old := newProject(t, pool, workspace, "WEB"), newProject(t, pool, workspace, "OLD")
				projectID, oldProjectID = &current.id, &old.id
				exec(t, pool, "UPDATE projects SET deleted_at=$2,updated_at=$2 WHERE id=$1", old.id, before)
			}
			a, b, looseA, looseB, old, far, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			// Reverse physical insertion order; ids still define each batch's boundary.
			for _, row := range []struct {
				id, workspace uuid.UUID
				project       *uuid.UUID
				deleted       *time.Time
			}{
				{far, other, nil, nil}, {gone, workspace, nil, &before}, {looseB, workspace, nil, nil}, {looseA, workspace, nil, nil},
			} {
				exec(t, pool, `INSERT INTO draft_issues(id,workspace_id,project_id,payload,created_by_id,updated_by_id,created_at,updated_at,deleted_at)
     VALUES($1,$2,$3,'{"name":"draft","parent_id":"unvalidated"}',$4,$4,$5::timestamptz,coalesce($6::timestamptz,$5::timestamptz),$6::timestamptz)`, row.id, row.workspace, row.project, creator, now, row.deleted)
			}
			if projectID != nil {
				for _, id := range []uuid.UUID{b, a, old} {
					p, deleted := projectID, (*time.Time)(nil)
					if id == old {
						p, deleted = oldProjectID, &before
					}
					exec(t, pool, `INSERT INTO draft_issues(id,workspace_id,project_id,created_by_id,updated_by_id,created_at,updated_at,deleted_at)
      VALUES($1,$2,$3,$4,$4,$5::timestamptz,coalesce($6::timestamptz,$5::timestamptz),$6::timestamptz)`, id, workspace, p, creator, now, deleted)
				}
			}
			rows, stamps := rowsAsText(t, pool, "draft_issues"), audits(t, pool, "draft_issues")
			cascade := app.NewCascade(s, nil, 1)
			tx := postgres.NewTxManager(pool, 5*time.Second)
			if route == "rollback project" || route == "rollback workspace" {
				rejected, earlier := b, a
				if route == "rollback workspace" {
					rejected, earlier = looseB, looseA
				}
				exec(t, pool, `CREATE FUNCTION reject_later_draft() RETURNS trigger LANGUAGE plpgsql AS $$
     BEGIN
      IF NEW.id='`+rejected.String()+`'::uuid THEN
       IF NOT EXISTS(SELECT 1 FROM draft_issues WHERE id='`+earlier.String()+`'::uuid AND deleted_at IS NOT NULL AND updated_by_id='`+actor.String()+`'::uuid) THEN
        RAISE EXCEPTION 'earlier batch did not write';
       END IF;
       RAISE EXCEPTION 'later draft batch rejected';
      END IF;
      RETURN NEW;
     END $$`)
				exec(t, pool, `CREATE TRIGGER reject_later_draft BEFORE UPDATE ON draft_issues FOR EACH ROW EXECUTE FUNCTION reject_later_draft()`)
				err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
					if route == "rollback project" {
						return cascade.DeleteProjectIssues(ctx, *projectID, actor, at)
					}
					return cascade.DeleteWorkspaceIssues(ctx, workspace, []uuid.UUID{*projectID}, actor, at)
				})
				if err == nil || !strings.Contains(err.Error(), "later draft batch rejected") {
					t.Fatalf("later batch error %v; want trigger failure", err)
				}
				if got := audits(t, pool, "draft_issues"); !maps.Equal(got, stamps) {
					t.Errorf("failed later batch left writes: %v; want %v", got, stamps)
				}
			} else {
				want := maps.Clone(stamps)
				if projectID != nil && route != "workspace batch" {
					if err := tx.WithinTx(context.Background(), func(ctx context.Context) error { return cascade.DeleteProjectIssues(ctx, *projectID, actor, at) }); err != nil {
						t.Fatal(err)
					}
					want[a], want[b] = deletion(at, actor), deletion(at, actor)
					if got := audits(t, pool, "draft_issues"); !maps.Equal(got, want) {
						t.Fatalf("project audits: %v; want %v", got, want)
					}
				}
				// The workspace id must suffice for both projectless batches.
				// Project drafts belong to their project cascade.
				if err := tx.WithinTx(context.Background(), func(ctx context.Context) error { return cascade.DeleteWorkspaceIssues(ctx, workspace, nil, actor, at) }); err != nil {
					t.Fatal(err)
				}
				want[looseA], want[looseB] = deletion(at, actor), deletion(at, actor)
				if got := audits(t, pool, "draft_issues"); !maps.Equal(got, want) {
					t.Errorf("workspace audits: %v; want %v", got, want)
				}
			}
			if got := rowsAsText(t, pool, "draft_issues"); !maps.Equal(got, rows) {
				t.Errorf("non-audit columns changed: %v; want %v", got, rows)
			}
		})
	}
}
