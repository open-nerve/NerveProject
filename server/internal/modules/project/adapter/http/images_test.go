package httpadapter_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestProjectCoverImageProblems(t *testing.T) {
	for _, tc := range []struct {
		code   string
		kind   shared.Kind
		status int
	}{
		{"asset.not_found", shared.KindNotFound, 404}, {"asset.not_uploaded", shared.KindConflict, 409},
		{"asset.bound", shared.KindConflict, 409}, {"asset.scope_changed", shared.KindConflict, 409},
	} {
		t.Run(tc.code, func(t *testing.T) {
			failure := shared.NewError(tc.kind, tc.code, "Image refused.")
			h := newServer(t, fakes{create: &fakeCreate{err: failure}, update: &fakeUpdate{err: failure}})
			for _, req := range []*http.Request{
				request(http.MethodPost, "/api/v0/workspaces/acme/projects", "alice", `{"name":"New","identifier":"NEW"}`),
				request(http.MethodPatch, "/api/v0/projects/"+webID.String(), "alice", `{}`),
			} {
				res, _ := do(t, h, req)
				if res.StatusCode != tc.status {
					t.Fatalf("%s: %d", tc.code, res.StatusCode)
				}
			}
		})
	}
	h := newServer(t, fakes{update: &fakeUpdate{err: shared.ServerBusy(time.Second)}})
	res, _ := do(t, h, request(http.MethodPatch, "/api/v0/projects/"+webID.String(), "alice", `{}`))
	if res.StatusCode != 503 || res.Header.Get("Retry-After") != "1" {
		t.Fatal("gate refusal lost retry contract", res.StatusCode, res.Header)
	}
}

func TestProjectImageReadBusy(t *testing.T) {
	failure := shared.ServerBusy(time.Second)
	h := newServer(t, fakes{get: &fakeGet{err: failure}, list: &fakeList{err: failure}})
	for _, path := range []string{"/api/v0/projects/" + webID.String(), "/api/v0/workspaces/acme/projects"} {
		res, _ := do(t, h, request(http.MethodGet, path, "alice", ""))
		if res.StatusCode != 503 || res.Header.Get("Retry-After") != "1" {
			t.Fatal("read gate lost retry contract", res.StatusCode, res.Header)
		}
	}
}
