package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"uuid"

	assetgen "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/http/gen"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type preparationPageHTTP struct {
	Data       []assetgen.Asset `json:"data"`
	NextCursor *string          `json:"next_cursor"`
}

func preparationsPath(id uuid.UUID) string { return draftPath(id) + "/asset-preparations" }

func preparationsHTTP(t *testing.T, w issueWorld, who string, id uuid.UUID, query string) preparationPageHTTP {
	t.Helper()
	raw := assetAPICall(t, w, who, "GET", preparationsPath(id)+query, "", 200)
	listKeys(t, []byte(raw), "data", "next_cursor")
	var page preparationPageHTTP
	decodeAnswer(t, raw, &page)
	if page.Data == nil {
		t.Fatal("empty preparation list must be []")
	}
	return page
}

func TestDraftAssetPreparationsHTTP(t *testing.T) {
	w := newIssueWorld(t)
	for _, who := range []string{"alice", "bob", "walt"} {
		// Walt is a workspace member but has not joined this project.
		for _, project := range []string{"null", fmt.Sprintf("%q", w.web)} {
			t.Run(who+"/"+project, func(t *testing.T) {
				d := createDraftHTTP(t, w, who, `{"project_id":`+project+`,"payload":{}}`)
				if page := preparationsHTTP(t, w, who, d.ID, ""); len(page.Data) != 0 || page.NextCursor != nil {
					t.Fatal(page)
				}
				for i := 0; i < 3; i++ {
					prepareAssetHTTP(t, w, who, "DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", d.ID), fmt.Sprintf("private-%d.txt", i), "text/plain", i+1)
				}
				before := draftAssetSnapshot(t, w)
				page := preparationsHTTP(t, w, who, d.ID, "?limit=2")
				if len(page.Data) != 2 || page.NextCursor == nil {
					t.Fatalf("lost preparations or pagination: %+v", page)
				}
				for i, a := range page.Data {
					if a.Name != fmt.Sprintf("private-%d.txt", i) || a.State != "pending" || a.IsBound || a.Scope.ID == nil || *a.Scope.ID != d.ID || a.Size != int64(i+1) {
						t.Fatal(a)
					}
					raw, err := json.Marshal(a)
					if err != nil {
						t.Fatal(err)
					}
					listKeys(t, raw, "id", "purpose", "scope", "name", "size", "content_type", "state", "is_bound", "created_at", "updated_at", "width", "height")
				}
				next := preparationsHTTP(t, w, who, d.ID, "?limit=2&cursor="+url.QueryEscape(*page.NextCursor))
				if len(next.Data) != 1 || next.NextCursor != nil || next.Data[0].Name != "private-2.txt" {
					t.Fatal(next)
				}
				if !reflect.DeepEqual(before, draftAssetSnapshot(t, w)) {
					t.Fatal("reading preparations changed the graph")
				}
			})
		}
	}
}

func TestDraftAssetPreparationsParameters(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
	cursor, err := assetdomain.EncodeDraftPreparationCursor(uuid.NewV7(), uuid.NewV7())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		suffix, code string
		status       int
	}{
		{"?limit=0", "validation_failed", 422}, {"?limit=101", "validation_failed", 422},
		{"?limit=x", "bad_request", 400}, {"?limit=1.5", "bad_request", 400}, {"?limit=", "bad_request", 400},
		{"?limit=1&limit=2", "bad_request", 400}, {"?cursor=", "bad_request", 400}, {"?cursor=bad", "bad_request", 400},
		{"?cursor=" + cursor, "bad_request", 400}, {"?cursor=a&cursor=b", "bad_request", 400},
		{"?unknown=1", "bad_request", 400}, {"?cursor=%GG", "bad_request", 400}, {"?limit=1;cursor=a", "bad_request", 400},
	} {
		t.Run(tc.suffix, func(t *testing.T) {
			for _, id := range []uuid.UUID{d.ID, uuid.NewV7()} {
				raw := assetAPICall(t, w, "alice", "GET", preparationsPath(id)+tc.suffix, "", tc.status)
				if problemCode(t, []byte(raw)) != tc.code {
					t.Fatal(raw)
				}
			}
		})
	}
	assetAPICall(t, w, "alice", "GET", "/api/v0/drafts/not-a-uuid/asset-preparations", "", 400)
	assetAPICall(t, w, "", "GET", preparationsPath(d.ID), "", 401)
}

func preparationRefusal(t *testing.T, w issueWorld, who string, id uuid.UUID) (string, http.Header) {
	t.Helper()
	req := newRequest(t, "GET", w.base+preparationsPath(id), w.tokens[who], nil)
	res, raw := send(t, req)
	w.contract.CheckResponse(t, req, res)
	assertAssetSecurityHeaders(t, res)
	if res.StatusCode != 404 || problemCode(t, raw) != "asset.not_found" || res.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("private list refusal: %d %s", res.StatusCode, raw)
	}
	headers := res.Header.Clone()
	// Transport-generated values legitimately differ between requests.
	headers.Del("X-Request-Id")
	headers.Del("Date")
	return string(raw), headers
}

func TestDraftAssetPreparationsPrivate(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
	prepareAssetHTTP(t, w, "alice", "DRAFT_ISSUE_ATTACHMENT", attachmentScope("draft", d.ID), "private-name.txt", "text/plain", 1)
	for _, who := range []string{"bob", "wanda", "olga"} {
		got, h := preparationRefusal(t, w, who, d.ID)
		absent, ah := preparationRefusal(t, w, who, uuid.NewV7())
		if got != absent || !reflect.DeepEqual(h, ah) || strings.Contains(got, "private-name") {
			t.Fatalf("distinguishable refusal: %s %s %+v %+v", got, absent, h, ah)
		}
	}
	var members struct {
		Data []struct {
			ID     uuid.UUID `json:"id"`
			Member struct {
				ID uuid.UUID `json:"id"`
			} `json:"member"`
		} `json:"data"`
	}
	decodeAnswer(t, w.send(t, "alice", "GET", "/api/v0/workspaces/acme/members", "", 200), &members)
	var walt uuid.UUID
	for _, m := range members.Data {
		if m.Member.ID == w.ids["walt"] {
			walt = m.ID
		}
	}
	if walt == uuid.Nil() {
		t.Fatal("missing workspace membership")
	}
	w.send(t, "alice", "PATCH", "/api/v0/workspace-members/"+walt.String(), `{"role":5}`, 200)
	got, h := preparationRefusal(t, w, "walt", d.ID)
	absent, ah := preparationRefusal(t, w, "walt", uuid.NewV7())
	if got != absent || !reflect.DeepEqual(h, ah) {
		t.Fatal("guest can distinguish draft existence", got, absent)
	}
	draftRequest(t, w, "walt", "GET", draftPath(d.ID), "", 403)
	draftRequest(t, w, "alice", "DELETE", draftPath(d.ID), "", 204)
	got, h = preparationRefusal(t, w, "alice", d.ID)
	absent, ah = preparationRefusal(t, w, "alice", uuid.NewV7())
	if got != absent || !reflect.DeepEqual(h, ah) {
		t.Fatal("deleted draft differs", got, absent)
	}
}

func TestDraftAssetPreparationsDatabaseFailure(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
	// Fault injection in this isolated database: an owner/query failure is 500,
	// never an empty page or a resource-concealing 404.
	if _, err := w.pool.Exec(t.Context(), "ALTER TABLE file_assets RENAME TO unavailable_assets"); err != nil {
		t.Fatal(err)
	}
	raw := assetAPICall(t, w, "alice", "GET", preparationsPath(d.ID), "", 500)
	if problemCode(t, []byte(raw)) != "internal_error" {
		t.Fatal(raw)
	}
}
