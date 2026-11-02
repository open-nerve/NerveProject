package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type assetPurgeTree struct{ workspace, project, state, issue, comment, draft uuid.UUID }

// Isolated persistence fixtures exercise the actual FK graph and owner SQL;
// they do not stand in for the public API stories.
func seedAssetPurgeTree(t *testing.T, pool *pgxpool.Pool, user uuid.UUID) assetPurgeTree {
	t.Helper()
	r := assetPurgeTree{uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO workspaces(id,name,slug) VALUES($1,'Tree',$2)", []any{r.workspace, "a" + r.workspace.String()}},
		{"INSERT INTO projects(id,workspace_id,name,identifier) VALUES($1,$2,'Tree','TREE')", []any{r.project, r.workspace}},
		{"INSERT INTO states(id,workspace_id,project_id,name,color) VALUES($1,$2,$3,'Todo','#ffffff')", []any{r.state, r.workspace, r.project}},
		{"INSERT INTO issues(id,workspace_id,project_id,state_id,name,sequence_id) VALUES($1,$2,$3,$4,'Tree',1)", []any{r.issue, r.workspace, r.project, r.state}},
		{"INSERT INTO issue_comments(id,workspace_id,project_id,issue_id,actor_id) VALUES($1,$2,$3,$4,$5)", []any{r.comment, r.workspace, r.project, r.issue, user}},
		{"INSERT INTO draft_issues(id,workspace_id,project_id,created_by_id) VALUES($1,$2,$3,$4)", []any{r.draft, r.workspace, r.project, user}},
	} {
		if _, err := pool.Exec(t.Context(), q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func (r assetPurgeTree) parent(table string) uuid.UUID {
	switch table {
	case "workspaces":
		return r.workspace
	case "projects":
		return r.project
	case "states":
		return r.state
	case "issues", "issue_self_cascade":
		return r.issue
	case "issue_comments":
		return r.comment
	case "draft_issues":
		return r.draft
	default:
		panic("unknown fixture table")
	}
}

func (r assetPurgeTree) attach(t *testing.T, pool *pgxpool.Pool, user uuid.UUID, table string, at time.Time) assetdomain.Asset {
	t.Helper()
	p := assetdomain.Placement{UserID: user, WorkspaceID: r.workspace, ProjectID: r.project, IssueID: r.issue, Scope: assetdomain.Scope{Kind: assetdomain.ScopeIssue, ID: r.issue}}
	purpose := assetdomain.IssueAttachment
	switch table {
	case "issue_comments":
		p.CommentID = r.comment
		p.Scope = assetdomain.Scope{Kind: assetdomain.ScopeComment, ID: r.comment}
		purpose = assetdomain.CommentAttachment
	case "draft_issues":
		p.IssueID = uuid.Nil()
		p.DraftID = r.draft
		p.Scope = assetdomain.Scope{Kind: assetdomain.ScopeDraft, ID: r.draft}
		purpose = assetdomain.DraftIssueAttachment
	case "issue_self_cascade":
		child := uuid.NewV7()
		if _, err := pool.Exec(t.Context(), "INSERT INTO issues(id,workspace_id,project_id,state_id,parent_id,name,sequence_id) VALUES($1,$2,$3,$4,$5,'Child',2)", child, r.workspace, r.project, r.state, r.issue); err != nil {
			t.Fatal(err)
		}
		p.IssueID = child
		p.Scope.ID = child
	}
	a := assetdomain.Asset{ID: uuid.NewV7(), Purpose: purpose, Placement: p, Name: "asset.txt", ContentType: "text/plain", Size: 3, CreatedAt: at, UpdatedAt: at, DeletedAt: &at, DeletedByID: &user, CreatedByID: &user, UpdatedByID: &user, Metadata: assetdomain.StorageMetadata{SchemaVersion: 1, Origin: assetdomain.OriginCopy, Candidates: []assetdomain.Candidate{}}}
	got, err := assetpg.New(pool).CreateAsset(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestAssetProtectedParentsProgressAndConverge(t *testing.T) {
	for _, table := range []string{"issue_comments", "draft_issues", "issues", "states", "projects", "workspaces", "issue_self_cascade"} {
		for _, batch := range []int{1, 3} {
			t.Run(fmt.Sprintf("%s/batch%d", table, batch), func(t *testing.T) {
				pool := openPool(t, pgtest.NewDatabase(t))
				user := uuid.NewV7()
				if _, err := pool.Exec(t.Context(), "INSERT INTO users(id,email,password,display_name) VALUES($1,'purge@example.com','x','Purge')", user); err != nil {
					t.Fatal(err)
				}
				trees := []assetPurgeTree{seedAssetPurgeTree(t, pool, user), seedAssetPurgeTree(t, pool, user), seedAssetPurgeTree(t, pool, user)}
				slices.SortFunc(trees, func(a, b assetPurgeTree) int { return a.parent(table).Compare(b.parent(table)) })
				now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
				cutoff := now.Add(-60 * 24 * time.Hour)
				target := table
				if target == "issue_self_cascade" {
					target = "issues"
				}
				for _, tree := range trees {
					if _, err := pool.Exec(t.Context(), "UPDATE "+target+" SET deleted_at=$2 WHERE id=$1", tree.parent(table), now.Add(-61*24*time.Hour)); err != nil {
						t.Fatal(err)
					}
				}
				assets := []assetdomain.Asset{trees[0].attach(t, pool, user, table, now.Add(-59*24*time.Hour)), trees[2].attach(t, pool, user, table, now.Add(-59*24*time.Hour))}
				refs := protectedAssetReferences()
				var p Purger
				switch target {
				case "states", "projects":
					p = forProjectPurge(project.Provide(pool, refs).Purger)
				case "workspaces":
					p = forWorkspacePurge(workspace.Provide(pool, refs).Purger)
				default:
					p = forIssuePurge(issue.Provide(pool, refs).Purger)
				}
				scan, deleted, blocked := drainAssetParentTable(t, p, target, cutoff, batch)
				if scan != 3 || deleted != 1 || blocked != 2 {
					t.Fatalf("protected page counts=%d/%d/%d", scan, deleted, blocked)
				}
				ids := queryIDs(t, pool, "SELECT id FROM "+target+" WHERE deleted_at<$1 ORDER BY id", cutoff)
				if !slices.Equal(ids, []uuid.UUID{trees[0].parent(table), trees[2].parent(table)}) {
					t.Fatalf("protected neighbours=%v", ids)
				}
				for _, a := range assets {
					got, err := assetpg.New(pool).FindAsset(t.Context(), a.ID)
					if err != nil || got.DeletedAt == nil || !got.DeletedAt.Equal(*a.DeletedAt) || got.StorageRevision != a.StorageRevision {
						t.Fatalf("parent purge changed asset: %+v %v", got, err)
					}
				}
				// Recover through the asset-owned SQL after its own retention ends.
				n, err := drainPurgeModule(forAssetPurge(asset.Provide(pool).Purger), t.Context(), now.Add(-58*24*time.Hour), 1)
				if err != nil || n != 2 {
					t.Fatalf("asset recovery=%d %v", n, err)
				}
				_, deleted, blocked = drainAssetParentTable(t, p, target, cutoff, batch)
				if deleted != 2 || blocked != 0 {
					t.Fatalf("parent did not converge=%d blocked=%d", deleted, blocked)
				}
			})
		}
	}
}

func drainAssetParentTable(t *testing.T, p Purger, table string, cutoff time.Time, batch int) (scanned, deleted, blocked int) {
	t.Helper()
	cursor := purgePageCursor{Table: table}
	for page := 0; page < 10; page++ {
		r, err := p.Page(context.Background(), cutoff, batch, cursor)
		if err != nil {
			t.Fatal(err)
		}
		scanned += r.Scanned
		deleted += r.Deleted
		blocked += r.Blocked
		if r.Done || r.Cursor.Table != table {
			return scanned, deleted, blocked
		}
		if r.Scanned > 0 && r.Cursor.After.Compare(cursor.After) <= 0 {
			t.Fatal("cursor did not progress")
		}
		cursor = r.Cursor
	}
	t.Fatal("parent table did not exhaust")
	return
}
