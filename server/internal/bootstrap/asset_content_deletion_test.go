package bootstrap

import (
	"fmt"
	"reflect"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestContentAssetDeletion(t *testing.T) {
	for _, kind := range []string{"issue", "comment", "draft", "project", "workspace"} {
		t.Run(kind, func(t *testing.T) {
			rw := newContentRetentionWorld(t)
			w := rw.issueWorld
			g := newContentAssetTree(t, w, w.web, "acme")
			for _, host := range []struct {
				kind, prefix string
				id           uuid.UUID
			}{
				{"issue", "ISSUE", g.issue}, {"comment", "COMMENT", g.comment}, {"draft", "DRAFT_ISSUE", g.draft},
			} {
				unused := uploadedAssetHTTP(t, w, "bob", host.prefix+"_DESCRIPTION", assetScopeBody(host.kind, host.id), "unused.png", "image/png", assetTestPNG(t))
				pending := prepareAssetHTTP(t, w, "bob", host.prefix+"_ATTACHMENT", assetScopeBody(host.kind, host.id), "pending.txt", "text/plain", 3)
				for _, id := range []uuid.UUID{unused.Asset.ID, pending.Asset.ID} {
					row, err := assetpg.New(w.pool).FindAsset(t.Context(), id)
					if err != nil {
						t.Fatal(err)
					}
					g.rows = append(g.rows, row)
				}
			}
			preparation := uploadedAssetHTTP(t, w, "bob", "COMMENT_ATTACHMENT", assetScopeBody("new_comment", g.issue), "prepared.png", "image/png", assetTestPNG(t))
			row, err := assetpg.New(w.pool).FindAsset(t.Context(), preparation.Asset.ID)
			if err != nil {
				t.Fatal(err)
			}
			g.rows = append(g.rows, row)
			if kind == "issue" {
				parent := g.issue
				for range 2 {
					child := w.newWorkItemBy(t, "bob", w.web, fmt.Sprintf(`{"name":"Nested","parent_id":%q}`, parent))
					image := uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", assetScopeBody("issue", child), "nested.png", "image/png", assetTestPNG(t))
					saveIssueImage(t, w, "bob", child, issueImageHTML(image.Asset.ID), false, 200)
					a, e := assetpg.New(w.pool).FindAsset(t.Context(), image.Asset.ID)
					if e != nil {
						t.Fatal(e)
					}
					g.rows = append(g.rows, a)
					parent = child
				}
			}
			// The neighbour belongs to an independent workspace; deleting the
			// target workspace cannot legitimately touch any of its file columns.
			var other struct{ ID uuid.UUID }
			decodeAnswer(t, w.send(t, "bob", "POST", "/api/v0/workspaces", `{"name":"Neighbour","slug":"neighbour"}`, 201), &other)
			nw := w
			nw.acme = other.ID
			nw.web = createdProject(t, w.contract, w.base, w.tokens["bob"], "neighbour", "Keep", "KEEP")
			neighbor := newContentAssetTree(t, nw, nw.web, "neighbour")
			beforeNeighbor := contentRowsSnapshot(t, w, neighbor.rows)
			// Preserve an earlier, differently authored deletion exactly.
			assetAPICall(t, w, "bob", "DELETE", "/api/v0/assets/"+g.rows[0].ID.String(), "", 204)
			old, err := assetpg.New(w.pool).FindAsset(t.Context(), g.rows[0].ID)
			if err != nil {
				t.Fatal(err)
			}
			activities := attachmentActivityCount(t, w, g.issue)
			actor, table, id := "alice", "issues", g.issue
			switch kind {
			case "issue":
				w.send(t, actor, "DELETE", "/api/v0/issues/"+id.String(), "", 204)
			case "comment":
				table, id = "issue_comments", g.comment
				w.send(t, actor, "DELETE", commentPath(id), "", 204)
			case "draft":
				actor, table, id = "bob", "draft_issues", g.draft
				draftRequest(t, w, actor, "DELETE", draftPath(id), "", 204)
			case "project":
				table, id = "projects", g.project
				deleteContentProject(t, w, g)
			case "workspace":
				table, id = "workspaces", g.workspace
				w.send(t, actor, "DELETE", "/api/v0/workspaces/acme", "", 204)
				waitWorkspaceDeletion(t, w.pool, g.workspace)
			}
			var at time.Time
			if err := w.pool.QueryRow(t.Context(), "SELECT deleted_at FROM "+table+" WHERE id=$1", id).Scan(&at); err != nil {
				t.Fatal(err)
			}
			for _, before := range g.rows {
				a, err := assetpg.New(w.pool).FindAsset(t.Context(), before.ID)
				if err != nil {
					t.Fatal(err)
				}
				if a.ID == old.ID {
					if !reflect.DeepEqual(a, old) {
						t.Fatal("old tombstone overwritten", a)
					}
					continue
				}
				deleted := kind == "workspace" || kind == "project" || kind == "issue" && before.Placement.IssueID != uuid.Nil() || kind == "comment" && before.Placement.CommentID == g.comment || kind == "draft" && before.Placement.DraftID == g.draft
				if !deleted {
					if !reflect.DeepEqual(a, before) {
						t.Fatal("unrelated host changed", a.ID)
					}
					continue
				}
				if a.DeletedAt == nil || !a.DeletedAt.Equal(at) || a.DeletedByID == nil || *a.DeletedByID != w.ids[actor] || !a.UpdatedAt.Equal(at) || a.UpdatedByID == nil || *a.UpdatedByID != w.ids[actor] || !reflect.DeepEqual(a.CreatedByID, before.CreatedByID) || !reflect.DeepEqual(a.BoundAt, before.BoundAt) || !reflect.DeepEqual(a.Metadata, before.Metadata) {
					t.Fatal("content deletion lost original binding/recovery/audit", before, a)
				}
				assetAPICall(t, w, "bob", "GET", "/api/v0/assets/"+a.ID.String()+"/content", "", 404)
				if a.Metadata.Final != nil {
					assertContentBytes(t, rw, a)
				}
			}
			if beforeNeighbor != contentRowsSnapshot(t, w, neighbor.rows) || attachmentActivityCount(t, w, g.issue) != activities {
				t.Fatal("parent deletion changed neighbour or emitted individual attachment activity")
			}
		})
	}
}

func newContentProjectTree(t *testing.T, w issueWorld, name string) contentAssetTree {
	t.Helper()
	project := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", name, name)
	w.makeMember(t, "bob", project, shared.RoleMember)
	return newContentAssetTree(t, w, project, "acme")
}
