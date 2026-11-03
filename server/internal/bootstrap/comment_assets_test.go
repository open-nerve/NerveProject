package bootstrap

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func commentAssetsBody(t *testing.T, html string, ids ...uuid.UUID) string {
	t.Helper()
	body, err := json.Marshal(struct {
		HTML string      `json:"comment_html"`
		IDs  []uuid.UUID `json:"attachment_ids,omitempty"`
	}{html, ids})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func commentAsset(t *testing.T, w issueWorld, who, kind string, id uuid.UUID, image bool) preparedAssetAnswer {
	t.Helper()
	purpose, name, mime, data := "COMMENT_ATTACHMENT", "file.txt", "text/plain", []byte("comment bytes")
	if image {
		purpose, name, mime, data = "COMMENT_DESCRIPTION", "image.png", "image/png", assetTestPNG(t)
	}
	return uploadedAssetHTTP(t, w, who, purpose, attachmentScope(kind, id), name, mime, data)
}

func TestCommentAssets(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	a := commentAsset(t, w, "bob", "new_comment", item, true)
	b := commentAsset(t, w, "bob", "new_comment", item, true)
	x := commentAsset(t, w, "bob", "new_comment", item, false)
	y := commentAsset(t, w, "bob", "new_comment", item, false)
	spare := commentAsset(t, w, "bob", "new_comment", item, false)
	ids := []uuid.UUID{a.Asset.ID, b.Asset.ID, x.Asset.ID, y.Asset.ID}
	origins := make(map[uuid.UUID]assetdomain.Asset)
	for _, id := range ids {
		row, err := assetpg.New(w.pool).FindAsset(t.Context(), id)
		if err != nil || row.BoundAt != nil {
			t.Fatal(row, err)
		}
		origins[id] = row
	}
	var c answeredComment
	decodeAnswer(t, w.send(t, "bob", "POST", commentsOf(item), commentAssetsBody(t, issueImageHTML(b.Asset.ID, a.Asset.ID, b.Asset.ID), y.Asset.ID, x.Asset.ID), 201), &c)
	for _, id := range ids {
		row, err := assetpg.New(w.pool).FindAsset(t.Context(), id)
		before := origins[id]
		if err != nil || row.BoundAt == nil || !row.BoundAt.Equal(c.CreatedAt) || !row.UpdatedAt.Equal(c.CreatedAt) || row.UpdatedByID == nil || *row.UpdatedByID != w.ids["bob"] || row.Placement.Scope != (assetdomain.Scope{Kind: assetdomain.ScopeComment, ID: c.ID}) || row.Placement.CommentID != c.ID || row.Placement.IssueID != item || row.Placement.UserID != w.ids["bob"] {
			t.Fatalf("comment did not atomically claim all preparations: %+v %v", row, err)
		}
		if !row.CreatedAt.Equal(before.CreatedAt) || !reflect.DeepEqual(row.Metadata, before.Metadata) || row.Purpose != before.Purpose {
			t.Fatal("claim changed origin, purpose or bytes")
		}
		readAssetHTTP(t, w, "alice", id)
	}
	untouched, err := assetpg.New(w.pool).FindAsset(t.Context(), spare.Asset.ID)
	if err != nil || untouched.BoundAt != nil || untouched.Placement.Scope.Kind != assetdomain.ScopeNewComment {
		t.Fatal("consumed an unlisted preparation", err)
	}
	assetAPICall(t, w, "alice", "GET", "/api/v0/assets/"+spare.Asset.ID.String()+"/content", "", 404)
	page := attachmentListRead(t, w, "alice", commentPath(c.ID)+"/attachments", 200)
	got := make([]uuid.UUID, 0, len(page.Data))
	for _, row := range page.Data {
		got = append(got, row.ID)
	}
	want := []uuid.UUID{x.Asset.ID, y.Asset.ID}
	slices.SortFunc(want, uuid.UUID.Compare)
	if !slices.Equal(got, want) || len(c.Reactions) != 0 || c.Reactions == nil {
		t.Fatal("attachments mixed with images/reactions", got, c)
	}
	if attachmentActivityCount(t, w, item) != 0 {
		t.Fatal("comment files produced issue attachment activity")
	}
	// A second composer for the same issue can consume its own remaining set,
	// but cannot reclaim the first comment's already moved files.
	before := contentAssetState(t, w)
	w.send(t, "bob", "POST", commentsOf(item), commentAssetsBody(t, "<p>Again</p>", x.Asset.ID), 404)
	if !reflect.DeepEqual(before, contentAssetState(t, w)) {
		t.Fatal("reclaim wrote part of another comment")
	}
	var second answeredComment
	decodeAnswer(t, w.send(t, "bob", "POST", commentsOf(item), commentAssetsBody(t, "<p>Second</p>", spare.Asset.ID), 201), &second)
	if second.ID == c.ID {
		t.Fatal("comment identity reused")
	}
}

func TestCommentAssetsImagesOnlyAndEdits(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	image := commentAsset(t, w, "bob", "new_comment", item, true)
	var c answeredComment
	// Only an image still satisfies the existing nonempty-comment rule.
	html := fmt.Sprintf(`<image-component src="%s"></image-component>`, image.Asset.ID)
	decodeAnswer(t, w.send(t, "bob", "POST", commentsOf(item), commentAssetsBody(t, html), 201), &c)
	bound, err := assetpg.New(w.pool).FindAsset(t.Context(), image.Asset.ID)
	if err != nil || bound.BoundAt == nil {
		t.Fatal("image-only creation did not bind", err)
	}
	next := commentAsset(t, w, "bob", "comment", c.ID, true)
	body := commentAssetsBody(t, issueImageHTML(next.Asset.ID, image.Asset.ID, next.Asset.ID))
	addCommentReaction(t, w, "alice", c.ID, "128077")
	var edited answeredComment
	decodeAnswer(t, w.send(t, "bob", "PATCH", commentPath(c.ID), body, 200), &edited)
	row, err := assetpg.New(w.pool).FindAsset(t.Context(), next.Asset.ID)
	if err != nil || row.BoundAt == nil || !row.BoundAt.Equal(edited.UpdatedAt) || edited.EditedAt == nil || len(edited.Reactions) != 1 {
		t.Fatal("edit lost binding, stamp or reactions", row, edited, err)
	}
	before := contentAssetState(t, w)
	w.send(t, "bob", "PATCH", commentPath(c.ID), body, 200)
	if !reflect.DeepEqual(before, contentAssetState(t, w)) {
		t.Fatal("unchanged bound content wrote state")
	}
	w.send(t, "bob", "PATCH", commentPath(c.ID), commentAssetsBody(t, "<p>Removed</p>"), 200)
	after, err := assetpg.New(w.pool).FindAsset(t.Context(), image.Asset.ID)
	if err != nil || !reflect.DeepEqual(bound, after) {
		t.Fatal("removing node unbound historical image", err)
	}
	commentAsset(t, w, "bob", "comment", c.ID, false)
	assetAPICall(t, w, "alice", "POST", "/api/v0/assets", assetPrepareBody("COMMENT_ATTACHMENT", attachmentScope("comment", c.ID), "no.txt", "text/plain", 1), 403)
	if attachmentActivityCount(t, w, item) != 0 {
		t.Fatal("comment update/append emitted issue attachment activity")
	}
	w.send(t, "bob", "DELETE", commentPath(c.ID), "", 204)
	for _, id := range []uuid.UUID{image.Asset.ID, next.Asset.ID} {
		assetAPICall(t, w, "bob", "GET", "/api/v0/assets/"+id.String()+"/content", "", 404)
	}
	attachmentListRead(t, w, "bob", commentPath(c.ID)+"/attachments", 404)
}

func TestCommentAssetsRejectWholeSet(t *testing.T) {
	for _, mode := range []string{"foreign preparation", "other issue", "existing comment", "wrong purpose", "pending", "deleted", "zero ID", "missing last", "empty comment"} {
		t.Run(mode, func(t *testing.T) {
			w := newIssueWorld(t)
			item := w.newWorkItem(t)
			good := commentAsset(t, w, "bob", "new_comment", item, false)
			who, kind, host, purpose := "bob", "new_comment", item, "COMMENT_ATTACHMENT"
			switch mode {
			case "foreign preparation":
				who = "alice"
			case "other issue":
				host = w.newWorkItem(t)
			case "existing comment":
				kind, host = "comment", w.newComment(t, "bob", item).ID
			case "wrong purpose":
				purpose = "COMMENT_DESCRIPTION"
			}
			data := assetTestPNG(t)
			bad := prepareAssetHTTP(t, w, who, purpose, attachmentScope(kind, host), "private.png", "image/png", len(data))
			if mode != "pending" {
				putAssetHTTP(t, w, bad, data, 204)
				assetAPICall(t, w, who, "POST", completeAssetPath(bad.Asset.ID), `{}`, 200)
			}
			if mode == "deleted" {
				assetAPICall(t, w, who, "DELETE", "/api/v0/assets/"+bad.Asset.ID.String(), "", 204)
			}
			id, html, status := bad.Asset.ID, "<p>Files</p>", 404
			switch mode {
			case "zero ID":
				id = uuid.Nil()
			case "missing last":
				id = uuid.NewV7()
			case "pending":
				status = 409
			case "empty comment":
				html, status = "<p> </p>", 422
			}
			before := contentAssetState(t, w)
			w.send(t, "bob", "POST", commentsOf(item), commentAssetsBody(t, html, good.Asset.ID, id), status)
			if !reflect.DeepEqual(before, contentAssetState(t, w)) {
				t.Fatal("rejected set wrote comment, subscribers, audit or consumed preparations")
			}
			// Every refusal leaves the good file usable by a subsequent valid request.
			w.send(t, "bob", "POST", commentsOf(item), commentAssetsBody(t, "<p>Retry</p>", good.Asset.ID), 201)
		})
	}
}

func TestCommentAssetsHundredBoundary(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	ids := make([]uuid.UUID, 100)
	for n := range ids {
		ids[n] = commentAsset(t, w, "bob", "new_comment", item, false).Asset.ID
	}
	before := contentAssetState(t, w)
	for _, tc := range []struct{ body, field, code string }{
		{commentAssetsBody(t, "<p>Too many</p>", append(slices.Clone(ids), uuid.NewV7())...), "attachment_ids", "too_long"},
		{commentAssetsBody(t, "<p>Duplicate</p>", ids[0], ids[1], ids[0]), "attachment_ids[2]", "duplicate"},
	} {
		// Deliberately invalid requests bypass only the test's positive request
		// validator. The actual route and strict response contract still run.
		req := newRequest(t, "POST", w.base+commentsOf(item), w.tokens["bob"], []byte(tc.body))
		res, body := send(t, req)
		w.contract.CheckResponse(t, req, res)
		if res.StatusCode != 422 || !strings.Contains(string(body), `"field":"`+tc.field+`"`) || !strings.Contains(string(body), `"code":"`+tc.code+`"`) || strings.Contains(string(body), ids[0].String()) {
			t.Fatal("unstable semantic refusal", res.StatusCode, string(body))
		}
	}
	if !reflect.DeepEqual(before, contentAssetState(t, w)) {
		t.Fatal("invalid ID list consumed preparations")
	}
	var c answeredComment
	decodeAnswer(t, w.send(t, "bob", "POST", commentsOf(item), commentAssetsBody(t, "<p>One hundred</p>", ids...), 201), &c)
	page := attachmentListRead(t, w, "alice", commentPath(c.ID)+"/attachments?limit=100", 200)
	if len(page.Data) != 100 || page.NextCursor != nil {
		t.Fatal("100 attachment boundary truncated", len(page.Data))
	}
}
