package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

// P2 has no binding API yet. These rows are explicit storage-layer fixtures;
// their hosts and all deletions use the running application's actual API.
func assetRowForHost(t *testing.T, w issueWorld, kind assetdomain.ScopeKind, id, issue uuid.UUID) assetdomain.Asset {
	t.Helper()
	p := assetdomain.Placement{UserID: w.ids["bob"], WorkspaceID: w.acme, ProjectID: w.web, IssueID: issue, Scope: assetdomain.Scope{Kind: kind, ID: id}}
	purpose := assetdomain.IssueAttachment
	if kind == assetdomain.ScopeComment {
		p.CommentID = id
		purpose = assetdomain.CommentAttachment
	}
	if kind == assetdomain.ScopeDraft {
		p.DraftID = id
		p.IssueID = uuid.Nil()
		purpose = assetdomain.DraftIssueAttachment
	}
	return assetRowAtPlacement(t, w.pool, p, purpose)
}

func assetRowAtPlacement(t *testing.T, pool *pgxpool.Pool, p assetdomain.Placement, purpose assetdomain.Purpose) assetdomain.Asset {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	a := assetdomain.Asset{ID: uuid.NewV7(), Purpose: purpose, Placement: p, Name: "fixture.txt", ContentType: "text/plain", Size: 3, CreatedAt: now, UpdatedAt: now, CreatedByID: &p.UserID, UpdatedByID: &p.UserID, Metadata: assetdomain.StorageMetadata{SchemaVersion: 1, Origin: assetdomain.OriginCopy, Candidates: []assetdomain.Candidate{}}}
	got, err := assetpg.New(pool).CreateAsset(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestAssetDeletionFollowsIssueTree(t *testing.T) {
	w := newIssueWorld(t)
	root := w.newWorkItem(t)
	child := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"child","parent_id":%q}`, root))
	grandchild := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"grandchild","parent_id":%q}`, child))
	comment := w.newComment(t, "bob", grandchild)
	var rows []assetdomain.Asset
	for _, id := range []uuid.UUID{root, child, grandchild} {
		rows = append(rows, assetRowForHost(t, w, assetdomain.ScopeIssue, id, id))
	}
	rows = append(rows, assetRowForHost(t, w, assetdomain.ScopeComment, comment.ID, grandchild))
	other := w.newWorkItem(t)
	neighbour := assetRowForHost(t, w, assetdomain.ScopeIssue, other, other)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+root.String(), "", http.StatusNoContent)
	var deleted time.Time
	if err := w.pool.QueryRow(t.Context(), "SELECT deleted_at FROM issues WHERE id=$1", root).Scan(&deleted); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		got, err := assetpg.New(w.pool).FindAsset(t.Context(), row.ID)
		if err != nil || got.DeletedAt == nil || !got.DeletedAt.Equal(deleted) || got.DeletedByID == nil || *got.DeletedByID != w.ids["alice"] || !got.UpdatedAt.Equal(deleted) {
			t.Fatalf("asset deletion=%+v error=%v", got, err)
		}
	}
	got, err := assetpg.New(w.pool).FindAsset(t.Context(), neighbour.ID)
	if err != nil || got.DeletedAt != nil {
		t.Fatalf("neighbour=%+v %v", got, err)
	}
}

func TestAssetDeletionFollowsCommentAndDraft(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	comment := w.newComment(t, "bob", id)
	attached := assetRowForHost(t, w, assetdomain.ScopeComment, comment.ID, id)
	neighbour := assetRowForHost(t, w, assetdomain.ScopeIssue, id, id)
	w.send(t, "bob", http.MethodDelete, commentPath(comment.ID), "", http.StatusNoContent)
	got, err := assetpg.New(w.pool).FindAsset(t.Context(), attached.ID)
	if err != nil || got.DeletedAt == nil || got.DeletedByID == nil || *got.DeletedByID != w.ids["bob"] {
		t.Fatalf("comment asset=%+v %v", got, err)
	}
	draft := createDraftHTTP(t, w, "bob", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"draft"}}`, w.web))
	attached = assetRowForHost(t, w, assetdomain.ScopeDraft, draft.ID, uuid.Nil())
	w.send(t, "bob", http.MethodDelete, draftPath(draft.ID), "", http.StatusNoContent)
	got, err = assetpg.New(w.pool).FindAsset(t.Context(), attached.ID)
	if err != nil || got.DeletedAt == nil || got.DeletedByID == nil || *got.DeletedByID != w.ids["bob"] {
		t.Fatalf("draft asset=%+v %v", got, err)
	}
	got, err = assetpg.New(w.pool).FindAsset(t.Context(), neighbour.ID)
	if err != nil || got.DeletedAt != nil {
		t.Fatalf("neighbour=%+v %v", got, err)
	}
}

// Publishing consumes the draft internally; that is not an explicit deletion
// of its attachments. Wait for startup recovery so it cannot race this proof.
func TestAssetDraftPublicationDoesNotDeleteAssets(t *testing.T) {
	w := newIssueWorld(t)
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1}, &lockedBuffer{})
	draft := createDraftHTTP(t, w, "bob", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"published"}}`, w.web))
	attached := assetRowForHost(t, w, assetdomain.ScopeDraft, draft.ID, uuid.Nil())
	w.send(t, "bob", http.MethodPost, publishPath(draft.ID), "", http.StatusCreated)
	var consumed bool
	if err := w.pool.QueryRow(t.Context(), "SELECT deleted_at IS NOT NULL FROM draft_issues WHERE id=$1", draft.ID).Scan(&consumed); err != nil || !consumed {
		t.Fatalf("draft consumed=%t %v", consumed, err)
	}
	got, err := assetpg.New(w.pool).FindAsset(t.Context(), attached.ID)
	if err != nil || got.DeletedAt != nil || got.StorageRevision != 0 {
		t.Fatalf("publish deleted its asset: %+v %v", got, err)
	}
}

// Explicit storage fixture until the asset API story lands in T10.
func seedIssueAsset(t *testing.T, pool *pgxpool.Pool, item, user uuid.UUID) assetdomain.Asset {
	t.Helper()
	p := assetdomain.Placement{UserID: user, IssueID: item, Scope: assetdomain.Scope{Kind: assetdomain.ScopeIssue, ID: item}}
	if err := pool.QueryRow(t.Context(), "SELECT workspace_id,project_id FROM issues WHERE id=$1", item).Scan(&p.WorkspaceID, &p.ProjectID); err != nil {
		t.Fatal(err)
	}
	return assetRowAtPlacement(t, pool, p, assetdomain.IssueAttachment)
}
