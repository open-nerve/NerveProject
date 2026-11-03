package postgresadapter

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type assetWorld struct {
	pool          *pgxpool.Pool
	store         *Store
	user, remover uuid.UUID
	workspace     uuid.UUID
	project       uuid.UUID
	issue         uuid.UUID
	comment       uuid.UUID
	draft         uuid.UUID
	now           time.Time
}

func newAssetWorld(t *testing.T) assetWorld {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), pgtest.NewDatabase(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	w := assetWorld{pool: pool, store: New(pool), user: uuid.NewV7(), remover: uuid.NewV7(), workspace: uuid.NewV7(), project: uuid.NewV7(), issue: uuid.NewV7(), comment: uuid.NewV7(), draft: uuid.NewV7(), now: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)}
	for _, query := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO users(id,email,password,display_name) VALUES($1,'asset@example.com','x','owner'),($2,'remover@example.com','x','remover')`, []any{w.user, w.remover}},
		{`INSERT INTO workspaces(id,name,slug) VALUES($1,'Acme','acme')`, []any{w.workspace}},
		{`INSERT INTO projects(id,workspace_id,name,identifier) VALUES($1,$2,'Web','WEB')`, []any{w.project, w.workspace}},
		{`INSERT INTO states(id,workspace_id,project_id,name,color) VALUES($1,$2,$3,'Todo','#ffffff')`, []any{w.project, w.workspace, w.project}},
		{`INSERT INTO issues(id,workspace_id,project_id,state_id,name,sequence_id) VALUES($1,$2,$3,$3,'Issue',1)`, []any{w.issue, w.workspace, w.project}},
		{`INSERT INTO issue_comments(id,workspace_id,project_id,issue_id,actor_id) VALUES($1,$2,$3,$4,$5)`, []any{w.comment, w.workspace, w.project, w.issue, w.user}},
		{`INSERT INTO draft_issues(id,workspace_id,project_id,created_by_id) VALUES($1,$2,$3,$4)`, []any{w.draft, w.workspace, w.project, w.user}},
	} {
		if _, err := pool.Exec(t.Context(), query.sql, query.args...); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

func (w assetWorld) asset(kind domain.ScopeKind) domain.Asset {
	p := domain.Placement{UserID: w.user, Scope: domain.Scope{Kind: kind}, WorkspaceID: w.workspace, ProjectID: w.project, IssueID: w.issue}
	purpose := domain.IssueAttachment
	switch kind {
	case domain.ScopeUser:
		p, purpose = domain.Placement{UserID: w.user, Scope: domain.Scope{Kind: kind}}, domain.UserAvatar
	case domain.ScopeWorkspace:
		p.Scope.ID, p.ProjectID, p.IssueID, purpose = w.workspace, uuid.Nil(), uuid.Nil(), domain.WorkspaceLogo
	case domain.ScopeProject:
		p.Scope.ID, p.IssueID, purpose = w.project, uuid.Nil(), domain.ProjectCover
	case domain.ScopeIssue:
		p.Scope.ID = w.issue
	case domain.ScopeComment:
		p.Scope.ID, p.CommentID, purpose = w.comment, w.comment, domain.CommentAttachment
	case domain.ScopeDraft:
		p.Scope.ID, p.DraftID, p.IssueID, purpose = w.draft, w.draft, uuid.Nil(), domain.DraftIssueAttachment
	}
	return domain.Asset{ID: uuid.NewV7(), Purpose: purpose, Placement: p, Name: "fixture.png", ContentType: "image/png", Size: 3, CreatedAt: w.now, UpdatedAt: w.now, CreatedByID: &w.user, UpdatedByID: &w.user, Metadata: domain.StorageMetadata{SchemaVersion: 1, Origin: domain.OriginCopy, Candidates: []domain.Candidate{}}}
}

func TestAssetDeletionGraph(t *testing.T) {
	for _, parent := range []struct {
		kind  domain.ScopeKind
		table string
	}{{domain.ScopeUser, "users"}, {domain.ScopeWorkspace, "workspaces"}, {domain.ScopeProject, "projects"}, {domain.ScopeIssue, "issues"}, {domain.ScopeComment, "issue_comments"}, {domain.ScopeDraft, "draft_issues"}} {
		t.Run(parent.table, func(t *testing.T) {
			w := newAssetWorld(t)
			a := w.asset(parent.kind)
			if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
				t.Fatal(err)
			}
			got, err := w.store.FindAsset(t.Context(), a.ID)
			if err != nil || !reflect.DeepEqual(a, got) {
				t.Fatalf("persisted asset=%+v error=%v want=%+v", got, err, a)
			}
			id := a.Placement.Scope.ID
			if parent.kind == domain.ScopeUser {
				id = w.user
			}
			_, err = w.pool.Exec(t.Context(), "DELETE FROM "+parent.table+" WHERE id=$1", id)
			var pe *pgconn.PgError
			if !errors.As(err, &pe) || pe.Code != "23503" || pe.SchemaName != "public" || pe.TableName != "file_assets" {
				t.Fatalf("parent deletion lost asset protection: %v", err)
			}
		})
	}
}

func TestAssetRevisionAndDeletionStamp(t *testing.T) {
	w := newAssetWorld(t)
	a := w.asset(domain.ScopeIssue)
	if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	tx := postgres.NewTxManager(w.pool, time.Second)
	err := tx.WithinTx(t.Context(), func(ctx context.Context) error {
		rows, err := w.store.LockAssets(ctx, []uuid.UUID{a.ID})
		if err != nil {
			return err
		}
		if len(rows) != 1 {
			t.Fatalf("locked rows=%d", len(rows))
		}
		_, err = w.store.SaveAsset(ctx, rows[0])
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.store.SaveAsset(t.Context(), a); !errors.Is(err, domain.ErrAssetScopeChanged) {
		t.Fatalf("stale overwrite=%v", err)
	}
	stamp := app.DeletionStamp{At: w.now.Add(time.Hour), By: &w.remover}
	if err := w.store.DeleteScopeRows(t.Context(), domain.ScopeIssue, w.issue, stamp); err != nil {
		t.Fatal(err)
	}
	if err := w.store.DeleteScopeRows(t.Context(), domain.ScopeIssue, w.issue, app.DeletionStamp{At: stamp.At.Add(time.Hour), By: &w.user}); err != nil {
		t.Fatal(err)
	}
	got, err := w.store.FindAsset(t.Context(), a.ID)
	if err != nil || got.DeletedAt == nil || !got.DeletedAt.Equal(stamp.At) || got.DeletedByID == nil || *got.DeletedByID != w.remover || !got.UpdatedAt.Equal(stamp.At) || got.UpdatedByID == nil || *got.UpdatedByID != w.remover || got.StorageRevision != 2 {
		t.Fatalf("stamp=%+v err=%v", got, err)
	}
}

func TestAssetPurgePreservesMetadataAndBoundary(t *testing.T) {
	w := newAssetWorld(t)
	cutoff := w.now.Add(-60 * 24 * time.Hour)
	for _, row := range []struct {
		name   string
		at     time.Time
		object bool
		want   int
	}{{"exact boundary", cutoff, false, 0}, {"old empty", cutoff.Add(-time.Microsecond), false, 1}, {"old candidate", cutoff.Add(-time.Hour), true, 0}} {
		t.Run(row.name, func(t *testing.T) {
			a := w.asset(domain.ScopeIssue)
			a.DeletedAt = &row.at
			a.DeletedByID = &w.remover
			if row.object {
				key, _ := domain.NewObjectKey('o', a.ID, uuid.NewV7())
				a.Metadata.Candidates = []domain.Candidate{{ObjectRecord: domain.ObjectRecord{Key: key, Phase: domain.ObjectActive, SafeAfter: w.now.Add(time.Hour), Size: 3, ContentType: a.ContentType}, LeaseUntil: w.now}}
			}
			if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
				t.Fatal(err)
			}
			page, err := w.store.purgeUnreferencedTestAssets(t.Context(), cutoff, uuid.Nil(), 100)
			if err != nil || page.Deleted != row.want {
				t.Fatalf("purge=%+v err=%v", page, err)
			}
			_, err = w.store.FindAsset(t.Context(), a.ID)
			if (errors.Is(err, domain.ErrAssetNotFound)) != (row.want == 1) {
				t.Fatalf("asset disappeared=%v", err)
			}
		})
	}
}
