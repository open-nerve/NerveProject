package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"
	"uuid"

	assetgen "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/http/gen"
	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

type preparedAssetAnswer struct {
	Asset struct {
		ID      uuid.UUID `json:"id"`
		Purpose string    `json:"purpose"`
		Scope   struct {
			Kind string     `json:"kind"`
			ID   *uuid.UUID `json:"id"`
		} `json:"scope"`
		Name        string    `json:"name"`
		Size        int64     `json:"size"`
		ContentType string    `json:"content_type"`
		State       string    `json:"state"`
		IsBound     bool      `json:"is_bound"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Width       *int      `json:"width"`
		Height      *int      `json:"height"`
	} `json:"asset"`
	Upload struct {
		Method    string                  `json:"method"`
		URL       string                  `json:"url"`
		Headers   []assetgen.UploadHeader `json:"headers"`
		ExpiresAt time.Time               `json:"expires_at"`
	} `json:"upload"`
}

func (p preparedAssetAnswer) uploadRequest() domain.SignedRequest {
	headers := make(map[string]string, len(p.Upload.Headers))
	for _, header := range p.Upload.Headers {
		headers[header.Name] = header.Value
	}
	return domain.SignedRequest{Method: p.Upload.Method, URL: p.Upload.URL, Headers: headers, ExpiresAt: p.Upload.ExpiresAt}
}

func assetPrepareBody(purpose, scope, name, contentType string, size int64) string {
	return fmt.Sprintf(`{"purpose":%q,"scope":%s,"name":%q,"size":%d,"content_type":%q}`, purpose, scope, name, size, contentType)
}

func assetAPICall(t *testing.T, w issueWorld, actor, method, path, body string, status int) string {
	t.Helper()
	req := newRequest(t, method, w.base+path, w.tokens[actor], []byte(body))
	res, data := send(t, req)
	if res.StatusCode != status {
		t.Fatalf("%s %s = %d %s, want %d", method, path, res.StatusCode, data, status)
	}
	if res.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("asset API response is cacheable")
	}
	assertAssetSecurityHeaders(t, res)
	w.contract.CheckResponse(t, req, res)
	return string(data)
}

// Each public purpose must persist its resolved host and a usable capability.
// A fixed DTO, omitted required header, or a business-clock expiry cannot pass.
func TestAssetPrepareReadContent(t *testing.T) {
	var cfg config.Config
	w := newIssueWorldWith(t, func(c *config.Config) { c.Files.SizeLimit = 32; c.Clock.Offset = 24 * time.Hour; cfg = *c })
	item := w.newWorkItem(t)
	comment := w.newComment(t, "alice", item)
	draft := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"draft"}}`, w.web))
	store := assetClientStore(t, cfg)
	for _, tc := range []struct {
		purpose, kind string
		id            uuid.UUID
	}{
		{"USER_AVATAR", "user", uuid.Nil()}, {"USER_COVER", "user", uuid.Nil()},
		{"WORKSPACE_LOGO", "workspace", w.acme}, {"PROJECT_COVER", "project", w.web},
		{"PROJECT_COVER", "new_project", w.acme}, {"ISSUE_ATTACHMENT", "issue", item},
		{"ISSUE_DESCRIPTION", "issue", item}, {"COMMENT_DESCRIPTION", "comment", comment.ID},
		{"COMMENT_ATTACHMENT", "comment", comment.ID}, {"COMMENT_ATTACHMENT", "new_comment", item},
		{"DRAFT_ISSUE_DESCRIPTION", "draft", draft.ID}, {"DRAFT_ISSUE_ATTACHMENT", "draft", draft.ID},
	} {
		t.Run(tc.purpose+"/"+tc.kind, func(t *testing.T) {
			scope := fmt.Sprintf(`{"kind":%q}`, tc.kind)
			if tc.id != uuid.Nil() {
				scope = fmt.Sprintf(`{"kind":%q,"id":%q}`, tc.kind, tc.id)
			}
			before := time.Now()
			body := assetAPICall(t, w, "alice", "POST", "/api/v0/assets", assetPrepareBody(tc.purpose, scope, "图片.png", "image/png", 3), 201)
			after := time.Now()
			var got preparedAssetAnswer
			decodeAnswer(t, body, &got)
			row, err := assetpg.New(w.pool).FindAsset(t.Context(), got.Asset.ID)
			if err != nil {
				t.Fatal(err)
			}
			if row.Placement.UserID != w.ids["alice"] || row.Placement.Scope.Kind != domain.ScopeKind(tc.kind) || row.Placement.Scope.ID != tc.id ||
				got.Asset.ID == uuid.Nil() || got.Asset.Purpose != tc.purpose || got.Asset.Name != "图片.png" || got.Asset.Size != 3 || got.Asset.ContentType != "image/png" ||
				got.Asset.State != "pending" || got.Asset.IsBound || got.Asset.Width != nil || got.Asset.Height != nil || !got.Asset.CreatedAt.Equal(row.CreatedAt) || !got.Asset.UpdatedAt.Equal(row.UpdatedAt) {
				t.Fatalf("wrong preparation: %s row=%+v", body, row)
			}
			if !row.CreatedAt.After(before.Add(23 * time.Hour)) {
				t.Fatal("audit time did not use the shifted business clock")
			}
			if got.Upload.Method != "PUT" || len(got.Upload.Headers) != 1 || got.Upload.Headers[0].Name != "Content-Type" || got.Upload.Headers[0].Value != "image/png" ||
				got.Upload.ExpiresAt.Before(before.Add(15*time.Minute-time.Second)) || got.Upload.ExpiresAt.After(after.Add(15*time.Minute)) ||
				row.UploadExpiresAt == nil || !row.UploadExpiresAt.Equal(got.Upload.ExpiresAt) || row.Metadata.Staging == nil {
				t.Fatalf("upload capability does not match persisted expiry: %s", body)
			}
			// Three bytes at the configured rate use 1s + 15s transfer grace,
			// followed by the independently specified 30s settle grace.
			if !row.Metadata.Staging.SafeAfter.Equal(got.Upload.ExpiresAt.Add(46 * time.Second)) {
				t.Fatal("wrong staging safety deadline")
			}
			parsed, err := url.Parse(got.Upload.URL)
			if err != nil || parsed.Path != "/static/local/"+string(row.Metadata.Staging.Key) {
				t.Fatalf("upload path=%q err=%v", got.Upload.URL, err)
			}
			assetRequest(t, w.base, got.uploadRequest(), "img", 204)
			info, err := store.Inspect(t.Context(), row.Metadata.Staging.Key)
			if err != nil || info.Size != 3 || info.ContentType != "image/png" {
				t.Fatalf("real object=%+v err=%v", info, err)
			}
			meta := assetAPICall(t, w, "alice", "GET", "/api/v0/assets/"+row.ID.String(), "", 200)
			var fields map[string]json.RawMessage
			decodeAnswer(t, meta, &fields)
			if len(fields) != 12 || strings.Contains(meta, string(row.Metadata.Staging.Key)) || strings.Contains(meta, "upload") {
				t.Fatalf("metadata leaks internals: %s", meta)
			}
			refusal := assetAPICall(t, w, "alice", "GET", "/api/v0/assets/"+row.ID.String()+"/content", "", 409)
			if problemCode(t, []byte(refusal)) != "asset.not_uploaded" {
				t.Fatal(refusal)
			}
			for _, suffix := range []string{"", "/content"} {
				refusal = assetAPICall(t, w, "bob", "GET", "/api/v0/assets/"+row.ID.String()+suffix, "", 404)
				missing := assetAPICall(t, w, "bob", "GET", "/api/v0/assets/"+uuid.New().String()+suffix, "", 404)
				if refusal != missing {
					t.Fatalf("another applicant is distinguishable: %s / %s", refusal, missing)
				}
			}
		})
	}
}

func TestAssetPrepareInputErrors(t *testing.T) {
	w := newIssueWorldWith(t, func(c *config.Config) { c.Files.SizeLimit = 32 })
	valid := assetPrepareBody("USER_AVATAR", `{"kind":"user"}`, "头像.png", "image/png", 3)
	for _, tc := range []struct {
		name, body, code string
		status           int
	}{
		{"unknown field", strings.TrimSuffix(valid, "}") + `,"user_id":"fake"}`, "bad_request", 400},
		{"duplicate field", strings.TrimSuffix(valid, "}") + `,"size":4}`, "bad_request", 400},
		{"scope wrong shape", strings.Replace(valid, `{"kind":"user"}`, `[]`, 1), "bad_request", 400},
		{"null scope", strings.Replace(valid, `{"kind":"user"}`, `null`, 1), "bad_request", 400},
		{"invalid uuid", strings.Replace(valid, `{"kind":"user"}`, `{"kind":"issue","id":"wrong"}`, 1), "bad_request", 400},
		{"purpose", assetPrepareBody("UNKNOWN", `{"kind":"user"}`, "x", "image/png", 3), "validation_failed", 422},
		{"scope mismatch", assetPrepareBody("USER_AVATAR", fmt.Sprintf(`{"kind":"workspace","id":%q}`, w.acme), "x", "image/png", 3), "validation_failed", 422},
		{"user extra id", assetPrepareBody("USER_AVATAR", fmt.Sprintf(`{"kind":"user","id":%q}`, w.ids["alice"]), "x", "image/png", 3), "validation_failed", 422},
		{"user zero id", assetPrepareBody("USER_AVATAR", `{"kind":"user","id":"00000000-0000-0000-0000-000000000000"}`, "x", "image/png", 3), "validation_failed", 422},
		{"missing host", assetPrepareBody("PROJECT_COVER", `{"kind":"project"}`, "x", "image/png", 3), "validation_failed", 422},
		{"path name", assetPrepareBody("USER_AVATAR", `{"kind":"user"}`, "a/b", "image/png", 3), "validation_failed", 422},
		{"empty", assetPrepareBody("USER_AVATAR", `{"kind":"user"}`, "x", "image/png", 0), "validation_failed", 422},
		{"effective limit", assetPrepareBody("USER_AVATAR", `{"kind":"user"}`, "x", "image/png", 33), "asset.too_large", 413},
		{"hard limit", assetPrepareBody("USER_AVATAR", `{"kind":"user"}`, "x", "image/png", 104857601), "asset.too_large", 413},
		{"svg", assetPrepareBody("USER_AVATAR", `{"kind":"user"}`, "x", "image/svg+xml", 3), "asset.type_unsupported", 415},
		{"JSON limit", assetPrepareBody("USER_AVATAR", `{"kind":"user"}`, strings.Repeat("x", 1<<20), "image/png", 3), "payload_too_large", 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := assetAPICall(t, w, "alice", "POST", "/api/v0/assets", tc.body, tc.status)
			if problemCode(t, []byte(body)) != tc.code {
				t.Fatal(body)
			}
		})
	}
	var count int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM file_assets").Scan(&count); err != nil || count != 0 {
		t.Fatalf("invalid input persisted rows=%d err=%v", count, err)
	}
}

// A deferred constraint fails COMMIT after the row and CPU signature exist.
// Returning the prepared value before commit would leak a usable capability.
func TestAssetPrepareCommitFailureReturnsNoCapability(t *testing.T) {
	w := newIssueWorld(t)
	_, err := w.pool.Exec(t.Context(), `CREATE FUNCTION reject_asset_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'asset commit rejected'; END $$;
CREATE CONSTRAINT TRIGGER reject_asset_commit AFTER INSERT ON file_assets DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_asset_commit();`)
	if err != nil {
		t.Fatal(err)
	}
	body := assetAPICall(t, w, "alice", "POST", "/api/v0/assets", assetPrepareBody("USER_AVATAR", `{"kind":"user"}`, "x.png", "image/png", 3), 500)
	if problemCode(t, []byte(body)) != "internal_error" || strings.Contains(body, "url") || strings.Contains(body, "signature") {
		t.Fatal(body)
	}
	var count int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM file_assets").Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed transaction left %d rows: %v", count, err)
	}
}
