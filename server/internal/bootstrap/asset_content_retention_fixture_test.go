package bootstrap

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type contentAssetTree struct {
	assetPurgeTree
	rows []assetdomain.Asset
}

func newContentRetentionWorld(t *testing.T) imageRetentionWorld {
	t.Helper()
	w := newImageRetentionWorld(t)
	w.tokens["bob"] = registerAccount(t, w.contract, w.base, "bob@example.com").AccessToken
	w.ids["bob"] = accountID(t, w.contract, w.base, w.tokens["bob"])
	link := inviteAs(t, w.contract, w.base, w.tokens["alice"], "acme", "bob@example.com", shared.RoleMember)
	answerInvitation(t, w.contract, w.base, w.tokens["bob"], "accept", link, 200)
	w.makeMember(t, "bob", w.web, shared.RoleMember)
	return w
}

// Every positive host, upload and binding is an actual public API operation.
// The separately named age helper only adjusts completed tombstone fixtures.
func newContentAssetTree(t *testing.T, w issueWorld, project uuid.UUID, slug string) contentAssetTree {
	t.Helper()
	g := contentAssetTree{assetPurgeTree: assetPurgeTree{workspace: w.acme, project: project}}
	g.issue = w.newWorkItemBy(t, "bob", project, `{"name":"Content retention"}`)
	g.comment = w.newComment(t, "bob", g.issue).ID
	var draft draftHTTP
	decodeAnswer(t, w.send(t, "bob", "POST", "/api/v0/workspaces/"+slug+"/drafts", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Private draft"}}`, project), 201), &draft)
	g.draft = draft.ID
	for _, host := range []struct {
		kind, prefix string
		id           uuid.UUID
	}{
		{"issue", "ISSUE", g.issue}, {"comment", "COMMENT", g.comment}, {"draft", "DRAFT_ISSUE", g.draft},
	} {
		for _, suffix := range []string{"ATTACHMENT", "DESCRIPTION"} {
			file := uploadedAssetHTTP(t, w, "bob", host.prefix+"_"+suffix, assetScopeBody(host.kind, host.id), "content.png", "image/png", assetTestPNG(t))
			if suffix == "DESCRIPTION" {
				html := issueImageHTML(file.Asset.ID)
				switch host.kind {
				case "issue":
					saveIssueImage(t, w, "bob", g.issue, html, false, 200)
				case "comment":
					w.send(t, "bob", "PATCH", commentPath(g.comment), commentAssetsBody(t, html), 200)
				case "draft":
					draftRequest(t, w, "bob", "PATCH", draftPath(g.draft), draftPayloadBody(t, map[string]any{"name": "Private draft", "description_html": html}), 200)
				}
			}
			row, err := assetpg.New(w.pool).FindAsset(t.Context(), file.Asset.ID)
			if err != nil || row.BoundAt == nil || row.Metadata.Final == nil {
				t.Fatal("actual content binding missing", row, err)
			}
			g.rows = append(g.rows, row)
		}
	}
	return g
}

func contentRowsSnapshot(t *testing.T, w issueWorld, rows []assetdomain.Asset) string {
	t.Helper()
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	var out string
	if err := w.pool.QueryRow(t.Context(), "SELECT coalesce(jsonb_agg(to_jsonb(a) ORDER BY id),'[]'::jsonb)::text FROM file_assets a WHERE id=ANY($1::uuid[])", ids).Scan(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func assertContentRowsPresent(t *testing.T, w issueWorld, rows []assetdomain.Asset, want bool) {
	t.Helper()
	for _, row := range rows {
		var present bool
		if err := w.pool.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM file_assets WHERE id=$1)", row.ID).Scan(&present); err != nil || present != want {
			t.Fatal("asset presence", row.ID, present, want, err)
		}
	}
}

func assertContentProjectPresent(t *testing.T, w issueWorld, g contentAssetTree, want bool) {
	t.Helper()
	var present bool
	if err := w.pool.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM projects WHERE id=$1)", g.project).Scan(&present); err != nil || present != want {
		t.Fatal("project presence", g.project, present, want, err)
	}
}

func assertContentBytes(t *testing.T, w imageRetentionWorld, row assetdomain.Asset, expected ...[]byte) {
	t.Helper()
	if row.Metadata.Final == nil {
		t.Fatal("no final object", row.ID)
	}
	info, err := w.objects.Inspect(t.Context(), row.Metadata.Final.Key)
	if err != nil {
		t.Fatal(err)
	}
	body, err := w.objects.Open(t.Context(), row.Metadata.Final.Key, info.ETag)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(body)
	want := assetTestPNG(t)
	if len(expected) != 0 {
		want = expected[0]
	}
	if err != nil || !bytes.Equal(data, want) {
		t.Fatal("retained bytes differ", row.ID, err)
	}
}

func deleteContentProject(t *testing.T, w issueWorld, g contentAssetTree) {
	t.Helper()
	w.send(t, "alice", "DELETE", "/api/v0/projects/"+g.project.String(), "", 204)
	waitProjectDeletion(t, w.pool, g.project)
}

func ageContentTree(t *testing.T, w imageRetentionWorld, g contentAssetTree, days int) {
	t.Helper()
	at := w.now.Add(-time.Duration(days) * 24 * time.Hour)
	for _, row := range g.rows {
		a, err := assetpg.New(w.pool).FindAsset(t.Context(), row.ID)
		if err != nil || a.DeletedAt == nil {
			t.Fatal("age requires actual deletion", row.ID, err)
		}
		a.DeletedAt = &at
		for _, record := range []*assetdomain.ObjectRecord{a.Metadata.Staging, a.Metadata.Final} {
			if record != nil {
				record.SafeAfter = w.now.Add(-time.Hour)
			}
		}
		if _, err := assetpg.New(w.pool).SaveAsset(t.Context(), a); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"projects", "issues", "issue_comments", "draft_issues"} {
		if _, err := w.pool.Exec(t.Context(), "UPDATE "+table+" SET deleted_at=$2 WHERE id=$1 AND deleted_at IS NOT NULL", g.parent(table), at); err != nil {
			t.Fatal(err)
		}
	}
}
