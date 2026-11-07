package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func (f *fakeIssueProjects) AssetProjectArchived(_ context.Context, id uuid.UUID) (bool, bool, error) {
	f.asked = append(f.asked, "archive state")
	return f.locked.Archived, id == f.web, f.err
}

func TestAssetParentLocksAndFacts(t *testing.T) {
	w := newIssueWorld(t)
	// A start-time run queued on a held project takes it as the holder lets
	// go, and locks each project in turn, which the probes that Web is free
	// would see.
	waitForStartupRuns(t, w.pool)
	var ws struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", "POST", "/api/v0/workspaces", `{"name":"Second","slug":"second"}`, 201), &ws)
	d1 := createDraftHTTP(t, w, "alice", `{"payload":{"name":"First"}}`)
	d2 := createDraftHTTP(t, w, "alice", `{"payload":{"name":"Second"}}`)
	secondProject := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Second", "TWO")
	i := w.newWorkItem(t)
	c := w.newComment(t, "bob", i)
	workspaces, issues, projects := workspace.Provide(w.pool, nil).AssetParents, issue.Provide(w.pool, nil).AssetParents, project.Provide(w.pool, nil)
	for _, tc := range []struct {
		table, conflict, compatible string
		ids                         []uuid.UUID
		lock                        func(context.Context, []uuid.UUID) ([]uuid.UUID, error)
	}{
		{"workspaces", "FOR NO KEY UPDATE", "FOR SHARE", []uuid.UUID{ws.ID, w.acme}, assetWorkspaceLocks{workspaces}.ShareMany},
		{"draft_issues", "FOR KEY SHARE", "", []uuid.UUID{d1.ID, d2.ID}, assetDraftLocks{issues}.LockMany},
		{"projects", "FOR SHARE", "FOR KEY SHARE", []uuid.UUID{w.web, secondProject}, assetProjectLocks{projects.IssueProjects}.LockMany},
	} {
		t.Run(tc.table, func(t *testing.T) {
			ids := slices.Clone(tc.ids)
			slices.SortFunc(ids, func(a, b uuid.UUID) int { return a.Compare(b) })
			// Deliberately request descending IDs. The later row is held by a
			// separate, identified backend; the earlier row must already be held.
			holder := holding(t, w.pool, "SELECT 1 FROM "+tc.table+" WHERE id=$1 FOR UPDATE", ids[1])
			probe := requestTx(w.pool)
			collect := requestBackground(t, func() error {
				return probe.WithinTx(t.Context(), func(ctx context.Context) error {
					got, err := tc.lock(ctx, []uuid.UUID{ids[1], ids[0], ids[1], uuid.New()})
					if err != nil {
						return err
					}
					if !slices.Equal(got, ids) {
						return fmt.Errorf("wrong locked set: %v", got)
					}
					return nil
				})
			}, func() { requestRelease(t, holder) })
			requestBlocked(t, w.pool, probe, holder.Conn().PgConn().PID(), tc.table)
			if !pgtest.RowWaits(t, w.pool, tc.table, ids[0], tc.conflict) {
				t.Error("lower ID not locked before waiting")
			}
			if tc.compatible != "" && pgtest.RowWaits(t, w.pool, tc.table, ids[0], tc.compatible) {
				t.Error("lock stronger than specified compatible mode")
			}
			requestRelease(t, holder)
			collect()
		})
	}
	at := time.Date(2020, 3, 4, 5, 6, 7, 123456000, time.UTC)
	by := w.ids["bob"]
	if archived, found, err := projects.IssueProjects.AssetProjectArchived(t.Context(), w.web); err != nil || !found || archived {
		t.Fatalf("live archive state: %v %v %v", archived, found, err)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE projects SET archived_at=$2 WHERE id=$1", w.web, at); err != nil {
		t.Fatal(err)
	}
	if archived, found, err := projects.IssueProjects.AssetProjectArchived(t.Context(), w.web); err != nil || !found || !archived {
		t.Fatalf("archived state: %v %v %v", archived, found, err)
	}
	for table, id := range map[string]uuid.UUID{"workspaces": ws.ID, "projects": w.web, "issues": i, "issue_comments": c.ID, "draft_issues": d1.ID} {
		if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE "+table+" SET deleted_at=$2,updated_by_id=$3 WHERE id=$1", id, at, by); err != nil {
			t.Fatal(err)
		}
	}
	err := postgres.NewTxManager(w.pool, 2*time.Second).WithinTx(t.Context(), func(ctx context.Context) error {
		wf, err := workspaces.AssetWorkspaceFacts(ctx, []uuid.UUID{ws.ID, w.acme, uuid.New()})
		if err != nil {
			return err
		}
		pf, err := projects.AssetParents.AssetProjectFacts(ctx, []uuid.UUID{w.web, uuid.New()})
		if err != nil {
			return err
		}
		ifc, err := issues.AssetIssueFacts(ctx, []uuid.UUID{i})
		if err != nil {
			return err
		}
		cf, err := issues.AssetCommentFacts(ctx, []uuid.UUID{c.ID})
		if err != nil {
			return err
		}
		df, err := issues.AssetDraftFacts(ctx, []uuid.UUID{d1.ID, d2.ID})
		if err != nil {
			return err
		}
		if len(wf) != 2 || len(pf) != 1 || len(ifc) != 1 || len(cf) != 1 || len(df) != 2 {
			return fmt.Errorf("facts lost: W%d P%d I%d C%d D%d", len(wf), len(pf), len(ifc), len(cf), len(df))
		}
		check := func(id uuid.UUID, deletedAt *time.Time, deletedBy *uuid.UUID) {
			t.Helper()
			if id == w.acme || id == d2.ID {
				if deletedAt != nil {
					t.Error("invented deletion")
				}
				return
			}
			if deletedAt == nil || !deletedAt.Equal(at) || deletedBy == nil || *deletedBy != by {
				t.Errorf("deletion provenance lost for %s", id)
			}
		}
		for _, r := range wf {
			check(r.ID, r.DeletedAt, r.DeletedByID)
		}
		for _, r := range pf {
			check(r.ID, r.DeletedAt, r.DeletedByID)
		}
		for _, rs := range [][]issue.AssetParent{ifc, cf, df} {
			for _, r := range rs {
				check(r.ID, r.DeletedAt, r.DeletedByID)
			}
		}
		for table, id := range map[string]uuid.UUID{"workspaces": ws.ID, "projects": w.web, "issues": i, "issue_comments": c.ID, "draft_issues": d1.ID} {
			if pgtest.RowWaits(t, w.pool, table, id, "FOR UPDATE") {
				t.Errorf("fact read locked %s", table)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = postgres.NewTxManager(w.pool, 2*time.Second).WithinTx(t.Context(), func(ctx context.Context) error {
		ids, err := issues.LockAssetDrafts(ctx, []uuid.UUID{d1.ID})
		if err != nil {
			return err
		}
		if !slices.Equal(ids, []uuid.UUID{d1.ID}) || !pgtest.RowWaits(t, w.pool, "draft_issues", d1.ID, "FOR KEY SHARE") {
			return fmt.Errorf("deleted draft not locked")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, found, err := projects.IssueProjects.AssetProjectArchived(t.Context(), w.web); err != nil || found {
		t.Fatalf("deleted project archive state: %v %v", found, err)
	}
}
