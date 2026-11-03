package httpadapter

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestMain(m *testing.M) { apitest.Main(m, "asset") }

type businessAuth struct{}

func (businessAuth) Authenticate(ctx context.Context, _ string) (context.Context, string, error) {
	return shared.WithActor(ctx, shared.Actor{UserID: uuid.New(), SessionID: uuid.New()}), "test-session", nil
}

type prepareError struct{ err error }

func (f prepareError) Execute(context.Context, domain.NewAsset) (app.PreparedAsset, error) {
	return app.PreparedAsset{}, f.err
}

type getError struct{ err error }

func (f getError) Execute(context.Context, uuid.UUID) (domain.Asset, error) {
	return domain.Asset{}, f.err
}

type contentError struct{ err error }

func (f contentError) Execute(context.Context, uuid.UUID) (domain.SignedRequest, error) {
	return domain.SignedRequest{}, f.err
}

type duplicateError struct{ err error }

func (f duplicateError) Execute(context.Context, uuid.UUID, domain.DuplicateAsset) (domain.Asset, error) {
	return domain.Asset{}, f.err
}

type deleteError struct{ err error }

func (f deleteError) Execute(context.Context, uuid.UUID) error { return f.err }

// The adapter is the real system under test here; only use case results are
// supplied. Host authorization itself is covered by the full application.
func TestAssetBusinessProblemMapping(t *testing.T) {
	contract := apitest.Load(t)
	for _, tc := range []struct {
		err    error
		code   string
		status int
		paths  []string
	}{
		{domain.ErrAssetNotFound, "asset.not_found", 404, []string{"prepare", "get", "content", "complete", "duplicate", "delete"}},
		{domain.ErrAssetScopeChanged, "asset.scope_changed", 409, []string{"prepare", "get", "content", "complete", "duplicate", "delete"}},
		{shared.Forbidden(), "forbidden", 403, []string{"prepare", "get", "content", "complete", "duplicate", "delete"}},
		{domain.ErrAssetNotUploaded, "asset.not_uploaded", 409, []string{"content", "duplicate"}},
		{domain.ErrAssetTooLarge, "asset.too_large", 413, []string{"prepare", "complete", "duplicate"}},
		{domain.ErrAssetTypeUnsupported, "asset.type_unsupported", 415, []string{"prepare", "complete", "duplicate"}},
		{shared.Invalid(shared.FieldError{Field: "name", Code: shared.FieldInvalidFormat, Message: "is invalid"}), "validation_failed", 422, []string{"prepare", "duplicate"}},
		{shared.ServerBusy(time.Second), "server_busy", 503, []string{"prepare", "content", "complete", "duplicate", "delete"}},
		{shared.NewError(shared.KindForbidden, "issue.not_author", "Only the author may edit."), "issue.not_author", 403, []string{"prepare", "complete", "duplicate", "delete"}},
		{shared.NewError(shared.KindConflict, "project.archived", "The project is archived."), "project.archived", 409, []string{"prepare", "complete", "duplicate", "delete"}},
		{shared.NewError(shared.KindConflict, "issue.archived", "The issue is archived."), "issue.archived", 409, []string{"prepare", "complete", "duplicate", "delete"}},
		{domain.ErrAssetObjectMissing, "asset.object_missing", 409, []string{"complete", "duplicate"}},
		{domain.ErrAssetObjectChanged, "asset.object_changed", 409, []string{"complete", "duplicate"}},
		{domain.ErrAssetContentInvalid, "asset.content_invalid", 422, []string{"complete", "duplicate"}},
		{domain.AssetBusy(), "asset.busy", 503, []string{"complete"}},
		{domain.AssetStorageUnavailable(), "asset.storage_unavailable", 503, []string{"complete", "duplicate"}},
		{domain.ErrAssetBound, "asset.bound", 409, []string{"delete"}},
	} {
		for _, operation := range tc.paths {
			t.Run(tc.code+"/"+operation, func(t *testing.T) {
				logger := slog.New(slog.DiscardHandler)
				router := httpserver.NewRouter(logger)
				limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
				api, err := httpserver.NewAPI(httpserver.APIConfig{Logger: logger, Authenticator: businessAuth{}, MaxBodyBytes: 1 << 20, RequestTimeout: time.Second, IPv6PrefixLen: 64, Anonymous: limit, Authenticated: limit, AuthFailure: limit})
				if err != nil {
					t.Fatal(err)
				}
				Register(router, api, UseCases{Duplicate: duplicateError{tc.err}, Delete: deleteError{tc.err}, Prepare: prepareError{tc.err}, Get: getError{tc.err}, Complete: getError{tc.err}, ResolveContent: contentError{tc.err}})
				method, path, body := "GET", "/api/v0/assets/"+uuid.New().String(), ""
				if operation == "prepare" {
					method, path, body = "POST", "/api/v0/assets", `{"purpose":"USER_AVATAR","scope":{"kind":"user"},"name":"x.png","size":3,"content_type":"image/png"}`
				}
				if operation == "duplicate" {
					method, body = "POST", `{"purpose":"USER_AVATAR","scope":{"kind":"user"}}`
					path += "/duplicate"
				}
				if operation == "delete" {
					method = "DELETE"
				}
				if operation == "complete" {
					method, body = "POST", `{}`
					path += "/complete"
				}
				if operation == "content" {
					path += "/content"
				}
				req := httptest.NewRequest(method, path, strings.NewReader(body))
				req.Header.Set("Authorization", "Bearer test")
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				res := rec.Result()
				defer func() { _ = res.Body.Close() }()
				contract.CheckResponse(t, req, res)
				var problem httpserver.Problem
				if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
					t.Fatal(err)
				}
				if rec.Code != tc.status || problem.Code != tc.code {
					t.Fatalf("status=%d problem=%s", rec.Code, rec.Body)
				}
				if tc.status == http.StatusServiceUnavailable && res.Header.Get("Retry-After") != "1" {
					t.Fatal("missing retry delay")
				}
			})
		}
	}
}
