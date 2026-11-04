package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func preparationID(n int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("019a0000-0000-7000-8000-%012d", n))
}

func preparationQuery(w assetWorld) app.DraftPreparationQuery {
	return app.DraftPreparationQuery{ActorID: w.user, Host: app.Host{Scope: domain.Scope{Kind: domain.ScopeDraft, ID: w.draft}, WorkspaceID: w.workspace, ProjectID: w.project, DraftID: w.draft}, Limit: 51}
}

func readPreparations(t *testing.T, w assetWorld, q app.DraftPreparationQuery) ([]domain.Asset, error) {
	t.Helper()
	var rows []domain.Asset
	err := postgres.NewTxManager(w.pool, time.Second, postgres.ReadOnlySnapshot()).WithinTx(t.Context(), func(ctx context.Context) error {
		var err error
		rows, err = w.store.ListDraftPreparations(ctx, q)
		return err
	})
	return rows, err
}

func TestDraftPreparationsRows(t *testing.T) {
	for _, noProject := range []bool{false, true} {
		t.Run(fmt.Sprintf("null_project_%v", noProject), func(t *testing.T) {
			w := newAssetWorld(t)
			q := preparationQuery(w)
			if noProject {
				if _, err := w.pool.Exec(t.Context(), "UPDATE draft_issues SET project_id=NULL WHERE id=$1", w.draft); err != nil {
					t.Fatal(err)
				}
				q.Host.ProjectID = uuid.Nil()
			}
			// Isolated repository fixtures exercise SQL filters, not HTTP authorization.
			wanted := make([]domain.Asset, 101)
			for i := 101; i >= 1; i-- {
				a := w.asset(domain.ScopeDraft)
				a.ID = preparationID(i)
				a.Name = fmt.Sprintf("same-name-%d.png", i%2)
				a.CreatedAt = w.now.Add(time.Duration(101-i) * time.Second)
				a.UpdatedAt = a.CreatedAt
				a.Placement.ProjectID = q.Host.ProjectID
				if i%2 == 0 {
					a.Purpose = domain.DraftIssueDescription
				}
				if i == 18 {
					ready := w.preparedImage(t, domain.ScopeDraft, domain.DraftIssueDescription)
					key, err := domain.NewObjectKey('o', a.ID, uuid.NewV7())
					if err != nil {
						t.Fatal(err)
					}
					a.Uploaded, a.Metadata = true, ready.Metadata
					a.Metadata.Final.Key = key
				}
				wanted[i-1] = w.putImage(t, a)
			}
			for _, mode := range []string{"bound description", "bound attachment", "deleted", "other applicant", "other creator", "other user", "other draft", "other scope"} {
				a := w.preparedImage(t, domain.ScopeDraft, domain.DraftIssueDescription)
				a.Placement.ProjectID = q.Host.ProjectID
				switch mode {
				case "bound description":
					a.BoundAt = &w.now
				case "bound attachment":
					a.Purpose, a.BoundAt = domain.DraftIssueAttachment, &w.now
				case "deleted":
					a.DeletedAt, a.DeletedByID = &w.now, &w.user
				case "other applicant":
					a.Placement.UserID, a.CreatedByID = w.remover, &w.remover
				case "other creator":
					a.CreatedByID = &w.remover
				case "other user":
					a.Placement.UserID = w.remover
				case "other draft":
					other := uuid.NewV7()
					if _, err := w.pool.Exec(t.Context(), "INSERT INTO draft_issues(id,workspace_id,created_by_id) VALUES($1,$2,$3)", other, w.workspace, w.user); err != nil {
						t.Fatal(err)
					}
					a.Placement.DraftID, a.Placement.Scope.ID = other, other
				case "other scope":
					a = w.preparedImage(t, domain.ScopeIssue, domain.IssueDescription)
				}
				w.putImage(t, a)
			}
			for _, size := range []int{1, 51, 100, 101} {
				q.Limit, q.After = size, uuid.Nil()
				var all []domain.Asset
				for range 103 {
					page, err := readPreparations(t, w, q)
					if err != nil || len(page) > size {
						t.Fatalf("size=%d rows=%d err=%v", size, len(page), err)
					}
					all = append(all, page...)
					if len(page) < size {
						break
					}
					q.After = page[len(page)-1].ID
				}
				if !reflect.DeepEqual(all, wanted) {
					t.Fatalf("size=%d got %d rows, expected complete ordered 101 with uploaded/pending and all metadata", size, len(all))
				}
			}
			q.After = uuid.Nil()
			q.Host.ProjectID = w.project
			if !noProject {
				q.Host.ProjectID = uuid.Nil()
			}
			if rows, err := readPreparations(t, w, q); err != nil || len(rows) != 0 {
				t.Fatalf("wrong project exposed rows=%d err=%v", len(rows), err)
			}
		})
	}
}

func TestDraftPreparationsRowsFailures(t *testing.T) {
	w := newAssetWorld(t)
	q := preparationQuery(w)
	if _, err := w.store.ListDraftPreparations(t.Context(), q); err == nil {
		t.Fatal("query escaped caller transaction")
	}
	for _, change := range []func(*app.DraftPreparationQuery){
		func(q *app.DraftPreparationQuery) { q.Limit = 0 },
		func(q *app.DraftPreparationQuery) { q.Limit = 102 },
		func(q *app.DraftPreparationQuery) { q.ActorID = uuid.Nil() },
		func(q *app.DraftPreparationQuery) { q.Host.WorkspaceID = uuid.Nil() },
		func(q *app.DraftPreparationQuery) { q.Host.DraftID = uuid.Nil() },
		func(q *app.DraftPreparationQuery) { q.Host.Scope.Kind = domain.ScopeIssue },
		func(q *app.DraftPreparationQuery) { q.Host.Scope.ID = w.issue },
		func(q *app.DraftPreparationQuery) { q.Host.IssueID = w.issue },
		func(q *app.DraftPreparationQuery) { q.Host.CommentID = w.comment },
	} {
		bad := q
		change(&bad)
		if _, err := readPreparations(t, w, bad); !errors.Is(err, domain.ErrInvalidAsset) {
			t.Fatalf("invalid query accepted: %+v %v", bad, err)
		}
	}
	a := w.putImage(t, w.asset(domain.ScopeDraft))
	// Explicit corruption injection: schema permits this JSON scalar but the codec must reject it.
	if _, err := w.pool.Exec(t.Context(), "UPDATE file_assets SET storage_metadata=jsonb_set(storage_metadata,'{candidates}','[{}]') WHERE id=$1", a.ID); err != nil {
		t.Fatal(err)
	}
	if rows, err := readPreparations(t, w, q); err == nil || len(rows) != 0 {
		t.Fatalf("corrupt record became empty/partial success: %+v %v", rows, err)
	}
	if _, err := w.pool.Exec(t.Context(), "ALTER TABLE file_assets RENAME TO inaccessible_assets"); err != nil {
		t.Fatal(err)
	}
	if rows, err := readPreparations(t, w, q); err == nil || len(rows) != 0 {
		t.Fatalf("database error swallowed: %+v %v", rows, err)
	}
}
