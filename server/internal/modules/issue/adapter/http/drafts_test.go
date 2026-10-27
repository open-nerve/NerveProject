package httpadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type failedDraftList struct{ err error }

func (f failedDraftList) Execute(context.Context, string, *int, *string) (app.DraftPage, error) {
	return app.DraftPage{}, f.err
}

type failedDraftCreate struct{ err error }

func (f failedDraftCreate) Execute(context.Context, string, domain.NewDraft) (domain.Draft, error) {
	return domain.Draft{}, f.err
}

type failedDraftGet struct{ err error }

func (f failedDraftGet) Execute(context.Context, uuid.UUID) (domain.Draft, error) {
	return domain.Draft{}, f.err
}

type failedDraftUpdate struct{ err error }

func (f failedDraftUpdate) Execute(context.Context, uuid.UUID, domain.DraftPatch) (domain.Draft, error) {
	return domain.Draft{}, f.err
}

// Refusals cross the real registered HTTP boundary. Swallowing a use-case
// error, disclosing error internals or losing Retry-After fails this table.
func TestDraftHTTPRefusals(t *testing.T) {
	forbidden := shared.Forbidden()
	invalid := shared.Invalid(shared.FieldError{Field: "project_id", Code: shared.FieldNotAllowed, Message: "not a visible project"})
	cases := []struct {
		name   string
		err    error
		status int
		body   string
		ops    []string
	}{
		{"workspace", domain.ErrWorkspaceNotFound, 404, `{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`, []string{"list", "create"}},
		{"missing", domain.ErrDraftNotFound, 404, `{"status":404,"code":"issue.draft_not_found","title":"Not Found","detail":"The draft does not exist, or you cannot see it."}`, []string{"get", "update", "delete"}},
		{"forbidden", forbidden, 403, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`, []string{"list", "create", "get", "update", "delete"}},
		{"invalid", invalid, 422, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.","errors":[{"field":"project_id","code":"not_allowed","message":"not a visible project"}]}`, []string{"list", "create", "update"}},
		{"changed", domain.ErrDraftChanged, 409, `{"status":409,"code":"issue.draft_changed","title":"Conflict","detail":"The draft's project changed. Please try again."}`, []string{"update", "delete"}},
		{"busy", shared.ServerBusy(time.Second), 503, busy, []string{"create", "update", "delete"}},
		{"internal", errGone, 500, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`, []string{"list", "create", "get", "update", "delete"}},
	}
	for _, tc := range cases {
		for _, op := range tc.ops {
			t.Run(op+"/"+tc.name, func(t *testing.T) {
				uc := httpadapter.UseCases{ListDrafts: failedDraftList{tc.err}, CreateDraft: failedDraftCreate{tc.err}, GetDraft: failedDraftGet{tc.err}, UpdateDraft: failedDraftUpdate{tc.err}, DeleteDraft: &fakeDelete{err: tc.err}}
				method, path, body := "GET", "/api/v0/workspaces/acme/drafts", ""
				switch op {
				case "create":
					method, body = "POST", `{"payload":{}}`
				case "get":
					path = "/api/v0/drafts/" + webID.String()
				case "update":
					method, path, body = "PATCH", "/api/v0/drafts/"+webID.String(), `{}`
				case "delete":
					method, path = "DELETE", "/api/v0/drafts/"+webID.String()
				}
				res, got := do(t, newServer(t, uc), request(method, path, "alice", body))
				if res.StatusCode != tc.status || got != tc.body+"\n" {
					t.Fatalf("%s = %d %s, want %d %s", op, res.StatusCode, got, tc.status, tc.body)
				}
				if tc.status == 503 && res.Header.Get("Retry-After") != "1" {
					t.Fatal("busy draft lost Retry-After")
				}
			})
		}
	}
}
