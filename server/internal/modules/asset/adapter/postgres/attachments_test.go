package postgresadapter

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// These state-combination fixtures exercise SQL predicates independently;
// real prepare/PUT/complete positives are in bootstrap/TestAttachmentLists.
func TestAttachmentListsSQLFilters(t *testing.T) {
	w := newAssetWorld(t)
	for _, kind := range []domain.ScopeKind{domain.ScopeIssue, domain.ScopeComment, domain.ScopeDraft} {
		purpose := map[domain.ScopeKind]domain.Purpose{domain.ScopeIssue: domain.IssueAttachment, domain.ScopeComment: domain.CommentAttachment, domain.ScopeDraft: domain.DraftIssueAttachment}[kind]
		var want []uuid.UUID
		for _, state := range []string{"live", "another applicant", "pending", "unbound", "deleted", "description"} {
			a := w.preparedImage(t, kind, purpose)
			a.BoundAt = &w.now
			switch state {
			case "another applicant":
				a.Placement.UserID = w.remover
				a.CreatedByID = &w.remover
			case "pending":
				a.Uploaded = false
			case "unbound":
				a.BoundAt = nil
			case "deleted":
				a.DeletedAt, a.DeletedByID = &w.now, &w.user
			case "description":
				a.Purpose = map[domain.ScopeKind]domain.Purpose{domain.ScopeIssue: domain.IssueDescription, domain.ScopeComment: domain.CommentDescription, domain.ScopeDraft: domain.DraftIssueDescription}[kind]
			}
			w.putImage(t, a)
			if state == "live" || state == "another applicant" {
				want = append(want, a.ID)
			}
		}
		h := app.ContentHost{WorkspaceID: w.workspace, ProjectID: w.project, IssueID: w.issue}
		if kind == domain.ScopeComment {
			h.CommentID = w.comment
		}
		if kind == domain.ScopeDraft {
			h.IssueID, h.DraftID = uuid.Nil(), w.draft
		}
		err := postgres.NewTxManager(w.pool, time.Second, postgres.ReadOnlySnapshot()).WithinTx(t.Context(), func(ctx context.Context) error {
			q := app.AttachmentRead{Host: h, Limit: 101}
			rows, err := w.store.ListAttachments(ctx, q)
			if err != nil {
				return err
			}
			got := []uuid.UUID{}
			for _, row := range rows {
				got = append(got, row.ID)
				if row.Name != "fixture.png" || row.ContentType != "image/png" || row.Size != 3 || !row.CreatedAt.Equal(w.now) {
					t.Fatalf("metadata %+v", row)
				}
			}
			slices.SortFunc(want, uuid.UUID.Compare)
			if !slices.Equal(got, want) {
				t.Fatalf("%s filters: %v want %v", kind, got, want)
			}
			q.After = want[0]
			q.Limit = 1
			rows, err = w.store.ListAttachments(ctx, q)
			if err != nil || len(rows) != 1 || rows[0].ID != want[1] {
				t.Fatal("after/limit", rows, err)
			}
			q.After = want[1]
			rows, err = w.store.ListAttachments(ctx, q)
			if err != nil || len(rows) != 0 {
				t.Fatal("tail", rows, err)
			}
			q.After = uuid.Nil()
			q.Host.WorkspaceID = uuid.NewV7()
			rows, err = w.store.ListAttachments(ctx, q)
			if err != nil || len(rows) != 0 {
				t.Fatal("wrong workspace leaked", rows, err)
			}
			q.Host = h
			q.Host.ProjectID = uuid.NewV7()
			rows, err = w.store.ListAttachments(ctx, q)
			if err != nil || len(rows) != 0 {
				t.Fatal("wrong project leaked", rows, err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	reader := app.NewAttachmentReads(w.store)
	err := postgres.NewTxManager(w.pool, time.Second, postgres.ReadOnlySnapshot()).WithinTx(t.Context(), func(ctx context.Context) error {
		counts, err := reader.CountIssueAttachments(ctx, []uuid.UUID{w.issue, w.issue, w.draft})
		if err != nil {
			return err
		}
		if len(counts) != 2 {
			t.Fatal(counts)
		}
		for _, row := range counts {
			want := int64(0)
			if row.IssueID == w.issue {
				want = 2
			}
			if row.Count != want {
				t.Fatal("counts included filtered rows", counts)
			}
		}
		catalog, err := reader.ProjectAttachmentCounts(ctx, []uuid.UUID{w.project}, uuid.Nil(), 1000)
		if err != nil || len(catalog) != 1 || catalog[0].IssueID != w.issue || catalog[0].Count != 2 {
			t.Fatal("catalog filters", catalog, err)
		}
		outside, err := reader.ProjectAttachmentCounts(ctx, []uuid.UUID{uuid.NewV7()}, uuid.Nil(), 1000)
		if err != nil || len(outside) != 0 {
			t.Fatal("catalog escaped project scope", outside, err)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	// A real SQL fault must stay an error, even when the desired page is empty.
	if _, err := w.pool.Exec(t.Context(), "ALTER TABLE file_assets RENAME TO unavailable_file_assets"); err != nil {
		t.Fatal(err)
	}
	for _, read := range []func(context.Context) error{
		func(ctx context.Context) error {
			_, err := reader.CountIssueAttachments(ctx, []uuid.UUID{w.issue})
			return err
		},
		func(ctx context.Context) error {
			_, err := reader.ProjectAttachmentCounts(ctx, []uuid.UUID{w.project}, uuid.Nil(), 1000)
			return err
		},
		func(ctx context.Context) error {
			_, err := reader.ListAttachments(ctx, app.AttachmentRead{Host: app.ContentHost{WorkspaceID: w.workspace, ProjectID: w.project, IssueID: w.issue}, Limit: 1})
			return err
		},
	} {
		err := postgres.NewTxManager(w.pool, time.Second, postgres.ReadOnlySnapshot()).WithinTx(t.Context(), read)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "42P01" {
			t.Fatal("SQL failure was hidden", err)
		}
	}
}

func TestAttachmentCountsSQL1001(t *testing.T) {
	w := newAssetWorld(t)
	ids := make([]uuid.UUID, 1001)
	for n := range ids {
		ids[n] = uuid.NewV7()
		if _, err := w.pool.Exec(t.Context(), "INSERT INTO issues(id,workspace_id,project_id,state_id,name,sequence_id) VALUES($1,$2,$3,$3,'capacity fixture',$4)", ids[n], w.workspace, w.project, n+2); err != nil {
			t.Fatal(err)
		}
		a := w.preparedImage(t, domain.ScopeIssue, domain.IssueAttachment)
		a.Placement.IssueID, a.Placement.Scope.ID = ids[n], ids[n]
		a.BoundAt = &w.now
		w.putImage(t, a)
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	r := app.NewAttachmentReads(w.store)
	err := postgres.NewTxManager(w.pool, time.Second, postgres.ReadOnlySnapshot()).WithinTx(t.Context(), func(ctx context.Context) error {
		counts, err := r.CountIssueAttachments(ctx, ids)
		if err != nil {
			return err
		}
		if len(counts) != 1001 {
			t.Fatalf("truncated ID batch: %d", len(counts))
		}
		for n, row := range counts {
			if row.IssueID != ids[n] || row.Count != 1 {
				t.Fatalf("count %d=%+v", n, row)
			}
		}
		first, err := r.ProjectAttachmentCounts(ctx, []uuid.UUID{w.project}, uuid.Nil(), 1000)
		if err != nil {
			return err
		}
		if len(first) != 1000 || first[999].IssueID != ids[999] {
			t.Fatalf("first page %d", len(first))
		}
		last, err := r.ProjectAttachmentCounts(ctx, []uuid.UUID{w.project}, first[999].IssueID, 1000)
		if err != nil || len(last) != 1 || last[0].IssueID != ids[1000] || last[0].Count != 1 {
			t.Fatal("page 1001 lost", last, err)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAttachmentReadsRequireCallerTransaction(t *testing.T) {
	s := &Store{}
	if _, err := s.ListAttachments(t.Context(), app.AttachmentRead{}); err == nil {
		t.Fatal("list opened own read")
	}
	if _, err := s.AttachmentCounts(t.Context(), []uuid.UUID{uuid.NewV7()}); err == nil {
		t.Fatal("count opened own read")
	}
	if _, err := s.ProjectAttachmentCounts(t.Context(), []uuid.UUID{uuid.NewV7()}, uuid.Nil(), 1000); err == nil {
		t.Fatal("catalog opened own read")
	}
	rows, err := app.NewAttachmentReads(s).CountIssueAttachments(t.Context(), nil)
	if err != nil || len(rows) != 0 {
		t.Fatal("empty IDs queried DB", rows, err)
	}
}
