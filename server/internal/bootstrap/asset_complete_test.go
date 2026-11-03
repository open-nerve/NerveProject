package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func prepareAssetHTTP(t *testing.T, w issueWorld, actor, purpose, scope, name, contentType string, size int) preparedAssetAnswer {
	t.Helper()
	var got preparedAssetAnswer
	decodeAnswer(t, assetAPICall(t, w, actor, "POST", "/api/v0/assets", assetPrepareBody(purpose, scope, name, contentType, int64(size)), 201), &got)
	return got
}
func putAssetHTTP(t *testing.T, w issueWorld, prepared preparedAssetAnswer, data []byte, status int) {
	t.Helper()
	assetRequest(t, w.base, prepared.uploadRequest(), string(data), status)
}
func readAssetHTTP(t *testing.T, w issueWorld, actor string, id uuid.UUID) (*http.Response, []byte) {
	t.Helper()
	var content struct {
		URL string `json:"url"`
	}
	decodeAnswer(t, assetAPICall(t, w, actor, "GET", "/api/v0/assets/"+id.String()+"/content", "", 200), &content)
	res, data := send(t, newRequest(t, "GET", w.base+content.URL, "", nil))
	if res.StatusCode != 200 {
		t.Fatalf("signed GET = %d %s", res.StatusCode, data)
	}
	return res, data
}
func completeAssetPath(id uuid.UUID) string { return "/api/v0/assets/" + id.String() + "/complete" }

func assetTestPNG(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	im := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	im.SetNRGBA(1, 2, color.NRGBA{R: 30, G: 50, B: 70, A: 255})
	if err := png.Encode(&out, im); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// A response-only implementation cannot satisfy the signed GET, database
// proof, repeated-call audit and old PUT assertions against real local bytes.
func TestAssetCompleteRealBytes(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	for _, tc := range []struct {
		name, purpose, scope, contentType, disposition string
		data                                           []byte
		image                                          bool
	}{
		{"attachment", "ISSUE_ATTACHMENT", fmt.Sprintf(`{"kind":"issue","id":%q}`, item), "text/plain", "attachment;", []byte("hello immutable world"), false},
		{"avatar", "USER_AVATAR", `{"kind":"user"}`, "image/png", "inline;", assetTestPNG(t), true},
		{"image attachment", "ISSUE_ATTACHMENT", fmt.Sprintf(`{"kind":"issue","id":%q}`, item), "image/png", "attachment;", assetTestPNG(t), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prepared := prepareAssetHTTP(t, w, "alice", tc.purpose, tc.scope, "private-file", tc.contentType, len(tc.data))
			putAssetHTTP(t, w, prepared, tc.data, 204)
			body := assetAPICall(t, w, "alice", "POST", completeAssetPath(prepared.Asset.ID), `{}`, 200)
			got := prepared.Asset
			decodeAnswer(t, body, &got)
			row, err := assetpg.New(w.pool).FindAsset(t.Context(), got.ID)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(tc.data)
			if got.ID != prepared.Asset.ID || got.State != "uploaded" || got.IsBound || !row.Uploaded || row.Metadata.Final == nil || row.Metadata.Staging == nil || len(row.Metadata.Candidates) != 0 ||
				row.Metadata.Final.Key.Kind() != 'o' || row.Metadata.Final.Key.AssetID() != row.ID || row.Metadata.Final.SHA256 != hex.EncodeToString(digest[:]) || row.Metadata.Final.ETag == "" {
				t.Fatalf("not a verified immutable result: API=%s row=%+v", body, row)
			}
			if tc.image {
				if got.Width == nil || got.Height == nil || *got.Width != 2 || *got.Height != 3 {
					t.Fatalf("wrong dimensions: %s", body)
				}
			} else if got.Width != nil || got.Height != nil {
				t.Fatalf("attachment unexpectedly image-validated: %s", body)
			}
			response, data := readAssetHTTP(t, w, "alice", got.ID)
			if !bytes.Equal(data, tc.data) || response.Header.Get("Content-Type") != tc.contentType || !strings.HasPrefix(response.Header.Get("Content-Disposition"), tc.disposition) {
				t.Fatalf("read bytes/disposition differ: %q %v", data, response.Header)
			}
			if again := assetAPICall(t, w, "alice", "POST", completeAssetPath(got.ID), `{}`, 200); again != body {
				t.Fatalf("repeat changed public result: %s / %s", body, again)
			}
			after, err := assetpg.New(w.pool).FindAsset(t.Context(), got.ID)
			if err != nil || !reflect.DeepEqual(row, after) {
				t.Fatalf("repeat changed final, candidates or audit: before=%+v after=%+v err=%v", row, after, err)
			}
			putAssetHTTP(t, w, prepared, bytes.Repeat([]byte{'x'}, len(tc.data)), 409)
			_, data = readAssetHTTP(t, w, "alice", got.ID)
			if !bytes.Equal(data, tc.data) {
				t.Fatal("old upload changed the published object")
			}
		})
	}
}

func TestAssetCompleteRefusalsPreserveRecovery(t *testing.T) {
	w := newIssueWorld(t)
	for _, tc := range []struct {
		name   string
		data   []byte
		status int
		code   string
	}{
		{"missing staging", nil, 409, "asset.object_missing"},
		{"fake png", []byte("not an image"), 422, "asset.content_invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prepared := prepareAssetHTTP(t, w, "alice", "USER_AVATAR", `{"kind":"user"}`, "x.png", "image/png", max(1, len(tc.data)))
			if tc.data != nil {
				putAssetHTTP(t, w, prepared, tc.data, 204)
			}
			for range 3 {
				body := assetAPICall(t, w, "alice", "POST", completeAssetPath(prepared.Asset.ID), `{}`, tc.status)
				if problemCode(t, []byte(body)) != tc.code {
					t.Fatal(body)
				}
			}
			row, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
			if err != nil || row.Uploaded || row.Metadata.Final != nil || row.Metadata.Staging == nil || len(row.Metadata.Candidates) != 3 {
				t.Fatalf("failed validation lost recovery: %+v %v", row, err)
			}
			seen := map[domain.ObjectKey]bool{}
			for _, candidate := range row.Metadata.Candidates {
				if seen[candidate.Key] || candidate.Key.AssetID() != row.ID || candidate.Key.Kind() != 'o' {
					t.Fatal("candidate key reused or belongs to another asset")
				}
				seen[candidate.Key] = true
			}
			req := newRequest(t, "POST", w.base+completeAssetPath(row.ID), w.tokens["alice"], []byte(`{}`))
			res, body := send(t, req)
			w.contract.CheckResponse(t, req, res)
			if res.StatusCode != 503 || res.Header.Get("Retry-After") != "1" || problemCode(t, body) != "asset.busy" {
				t.Fatalf("full candidate set: %d %s %v", res.StatusCode, body, res.Header)
			}
			if refusal := assetAPICall(t, w, "bob", "POST", completeAssetPath(row.ID), `{}`, 404); problemCode(t, []byte(refusal)) != "asset.not_found" {
				t.Fatal(refusal)
			}
			for _, body := range []string{`{"unexpected":true}`, `[]`, `null`, ""} {
				assetAPICall(t, w, "alice", "POST", completeAssetPath(row.ID), body, 400)
			}
		})
	}
}
