package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeLinks is each of the module links' use cases: each call is recorded as
// "operation caller id", with what it got; it answers link and list, or
// err.
type fakeLinks struct {
	calls   []string
	created []domain.NewModuleLink
	patched []domain.ModuleLinkPatch
	link    domain.ModuleLink
	list    []domain.ModuleLink
	err     error
}

func (f *fakeLinks) record(ctx context.Context, op string, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	f.calls = append(f.calls, op+" "+actor.UserID.String()+" "+id.String())
	return nil
}

type fakeCreateLink struct{ *fakeLinks }

func (f fakeCreateLink) Execute(ctx context.Context, module uuid.UUID, in domain.NewModuleLink) (domain.ModuleLink, error) {
	if err := f.record(ctx, "create", module); err != nil {
		return domain.ModuleLink{}, err
	}
	f.created = append(f.created, in)
	return f.link, f.err
}

type fakeUpdateLink struct{ *fakeLinks }

func (f fakeUpdateLink) Execute(ctx context.Context, id uuid.UUID, p domain.ModuleLinkPatch) (domain.ModuleLink, error) {
	if err := f.record(ctx, "update", id); err != nil {
		return domain.ModuleLink{}, err
	}
	f.patched = append(f.patched, p)
	return f.link, f.err
}

type fakeDeleteLink struct{ *fakeLinks }

func (f fakeDeleteLink) Execute(ctx context.Context, id uuid.UUID) error {
	if err := f.record(ctx, "delete", id); err != nil {
		return err
	}
	return f.err
}

type fakeListLinks struct{ *fakeLinks }

func (f fakeListLinks) Execute(ctx context.Context, module uuid.UUID) ([]domain.ModuleLink, error) {
	if err := f.record(ctx, "list", module); err != nil {
		return nil, err
	}
	return f.list, f.err
}

// linkServer serves the module links' operations over f.
func linkServer(t *testing.T, f *fakeLinks) http.Handler {
	return newServer(t, httpadapter.UseCases{CreateModuleLink: fakeCreateLink{f}, UpdateModuleLink: fakeUpdateLink{f}, DeleteModuleLink: fakeDeleteLink{f},
		ListModuleLinks: fakeListLinks{f}})
}

var specID = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000e1")

func ptrTo[T any](v T) *T { return &v }

// specLinks are Billing's links as the use cases answer them: Spec, with a
// title and its preview; Bare, without either, whose creator is gone.
func specLinks() (spec, bare domain.ModuleLink) {
	title, favicon := "Spec", "https://nerve.dev/favicon.ico"
	failed := "failed"
	spec = domain.ModuleLink{ID: specID, ModuleID: billingID, ProjectID: webID, WorkspaceID: acmeID, URL: "https://nerve.dev/spec", Title: &title,
		Metadata: shared.LinkMetadata{"title": &title, "favicon": nil, "url": nil, "favicon_url": &favicon, "error": &failed}, CreatedByID: &aliceID,
		CreatedAt: created, UpdatedAt: created.Add(1)}
	bare = domain.ModuleLink{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000e2"), ModuleID: billingID, URL: "http://nerve.dev",
		Metadata: shared.LinkMetadata{}, CreatedAt: created, UpdatedAt: created}
	return spec, bare
}

const (
	specJSON = `{"created_at":"2026-11-07T10:00:00.123456Z","created_by_id":"0199a2b4-0000-7000-8000-000000000001",` +
		`"id":"0199a2b4-0000-7000-8000-0000000000e1","metadata":{"error":"failed","favicon":null,"favicon_url":"https://nerve.dev/favicon.ico",` +
		`"title":"Spec","url":null},"module_id":"0199a2b4-0000-7000-8000-0000000000d2","title":"Spec",` +
		`"updated_at":"2026-11-07T10:00:00.123456001Z","url":"https://nerve.dev/spec"}`
	bareJSON = `{"created_at":"2026-11-07T10:00:00.123456Z","created_by_id":null,"id":"0199a2b4-0000-7000-8000-0000000000e2","metadata":{},` +
		`"module_id":"0199a2b4-0000-7000-8000-0000000000d2","title":null,"updated_at":"2026-11-07T10:00:00.123456Z","url":"http://nerve.dev"}`
)

// POST goes to the use case for the caller and the path's module with the
// address and the title as given, null or left out for none, empty kept;
// the answer is 201 with the link, its metadata only with the keys it has.
// GET answers the use case's list, an empty one as an empty array.
func TestCreateAndListModuleLinks(t *testing.T) {
	spec, bare := specLinks()
	path := "/api/v0/modules/" + billingID.String() + "/links"
	empty := ""
	for _, tt := range []struct {
		body string
		in   domain.NewModuleLink
		link domain.ModuleLink
		want string
	}{
		{`{"url":" nerve.dev/spec ","title":"Spec"}`, domain.NewModuleLink{URL: " nerve.dev/spec ", Title: ptrTo("Spec")}, spec, specJSON},
		{`{"url":"http://nerve.dev"}`, domain.NewModuleLink{URL: "http://nerve.dev"}, bare, bareJSON},
		{`{"url":"http://nerve.dev","title":null}`, domain.NewModuleLink{URL: "http://nerve.dev"}, bare, bareJSON},
		{`{"url":"http://nerve.dev","title":""}`, domain.NewModuleLink{URL: "http://nerve.dev", Title: &empty}, bare, bareJSON},
	} {
		f := &fakeLinks{link: tt.link}
		res, body := do(t, linkServer(t, f), request(http.MethodPost, path, "alice", tt.body))
		if res.StatusCode != http.StatusCreated || body != tt.want+"\n" {
			t.Errorf("POST %s = %d %s, want 201 %s", tt.body, res.StatusCode, body, tt.want)
		}
		if want := []string{"create " + aliceID.String() + " " + billingID.String()}; !slices.Equal(f.calls, want) ||
			!reflect.DeepEqual(f.created, []domain.NewModuleLink{tt.in}) {
			t.Errorf("POST %s: the use case got %q %+v, want %q %+v", tt.body, f.calls, f.created, want, tt.in)
		}
	}
	for _, tt := range []struct {
		list []domain.ModuleLink
		want string
	}{
		{[]domain.ModuleLink{spec, bare}, `{"data":[` + specJSON + `,` + bareJSON + `]}`},
		{nil, `{"data":[]}`},
	} {
		f := &fakeLinks{list: tt.list}
		res, body := do(t, linkServer(t, f), request(http.MethodGet, path, "alice", ""))
		if want := []string{"list " + aliceID.String() + " " + billingID.String()}; res.StatusCode != http.StatusOK || body != tt.want+"\n" ||
			!slices.Equal(f.calls, want) {
			t.Errorf("GET = %d %s, calls %q; want 200 %s", res.StatusCode, body, f.calls, tt.want)
		}
	}
}

// PATCH goes to the use case with what is given: the address, the title,
// null to clear it, empty kept, left out unchanged; the answer is 200 with
// the link as changed. DELETE answers 204.
func TestUpdateAndDeleteModuleLink(t *testing.T) {
	spec, _ := specLinks()
	path := "/api/v0/module-links/" + specID.String()
	empty := ""
	for _, tt := range []struct {
		body  string
		patch domain.ModuleLinkPatch
	}{
		{`{"url":"nerve.dev/other"}`, domain.ModuleLinkPatch{URL: ptrTo("nerve.dev/other")}},
		{`{"title":"Spec"}`, domain.ModuleLinkPatch{SetTitle: true, Title: ptrTo("Spec")}},
		{`{"title":null}`, domain.ModuleLinkPatch{SetTitle: true}},
		{`{"url":"http://nerve.dev","title":""}`, domain.ModuleLinkPatch{URL: ptrTo("http://nerve.dev"), SetTitle: true, Title: &empty}},
		{`{}`, domain.ModuleLinkPatch{}},
	} {
		f := &fakeLinks{link: spec}
		res, body := do(t, linkServer(t, f), request(http.MethodPatch, path, "alice", tt.body))
		if res.StatusCode != http.StatusOK || body != specJSON+"\n" || !reflect.DeepEqual(f.patched, []domain.ModuleLinkPatch{tt.patch}) ||
			!slices.Equal(f.calls, []string{"update " + aliceID.String() + " " + specID.String()}) {
			t.Errorf("PATCH %s = %d %s, the use case got %q %+v; want 200, %+v", tt.body, res.StatusCode, body, f.calls, f.patched, tt.patch)
		}
	}
	f := &fakeLinks{}
	if res, body := do(t, linkServer(t, f), request(http.MethodDelete, path, "alice", "")); res.StatusCode != http.StatusNoContent || body != "" ||
		!slices.Equal(f.calls, []string{"delete " + aliceID.String() + " " + specID.String()}) {
		t.Errorf("DELETE = %d %s, calls %q; want 204", res.StatusCode, body, f.calls)
	}
}

// A body that is not a link's — without its address, with a field it may
// not have, of another type, null for the address — is refused as
// bad_request before the use case.
func TestModuleLinkBodiesHeldToTheirStructure(t *testing.T) {
	f := &fakeLinks{}
	h := linkServer(t, f)
	for _, tt := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v0/modules/" + billingID.String() + "/links", `{}`},
		{http.MethodPost, "/api/v0/modules/" + billingID.String() + "/links", `{"title":"A"}`},
		{http.MethodPost, "/api/v0/modules/" + billingID.String() + "/links", `{"url":1}`},
		{http.MethodPost, "/api/v0/modules/" + billingID.String() + "/links", `{"url":"http://a","metadata":{}}`},
		{http.MethodPost, "/api/v0/modules/" + billingID.String() + "/links", `{"url":"http://a","title":1}`},
		{http.MethodPatch, "/api/v0/module-links/" + specID.String(), `{"url":null}`},
		{http.MethodPatch, "/api/v0/module-links/" + specID.String(), `{"module_id":"` + billingID.String() + `"}`},
		{http.MethodPatch, "/api/v0/module-links/" + specID.String(), `{"title":[]}`},
	} {
		res, got := do(t, h, request(tt.method, tt.path, "alice", tt.body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("%s %s = %d %s, want 400 bad_request", tt.method, tt.body, res.StatusCode, got)
		}
	}
	if len(f.calls) != 0 {
		t.Errorf("the use cases got %q, want nothing", f.calls)
	}
}

// The use cases' refusals, as the contract declares them, and a failure.
func TestModuleLinkRefusals(t *testing.T) {
	_, badURL := domain.CheckNewModuleLink(domain.NewModuleLink{URL: "ftp://nerve.dev"})
	invalid := `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
		`"errors":[{"field":"url","code":"not_allowed","message":"must be an HTTP or HTTPS URL without credentials, zones or control characters, of at most 2048 characters"}]}`
	refusals := map[string]struct {
		err    error
		status int
		want   string
	}{
		"invalid":       {badURL, http.StatusUnprocessableEntity, invalid},
		"no module":     {domain.ErrModuleNotFound, http.StatusNotFound, `{"status":404,"code":"planning.module_not_found","title":"Not Found","detail":"The module does not exist, or you cannot see it."}`},
		"no link":       {domain.ErrModuleLinkNotFound, http.StatusNotFound, `{"status":404,"code":"planning.module_link_not_found","title":"Not Found","detail":"The link does not exist, or you cannot see it."}`},
		"forbidden":     {shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		"project":       {domain.ErrProjectArchived, http.StatusConflict, `{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		"modules off":   {domain.ErrModulesDisabled, http.StatusConflict, `{"status":409,"code":"planning.modules_disabled","title":"Conflict","detail":"Modules are turned off in this project; turn them on to change them."}`},
		"archived":      {domain.ErrArchived, http.StatusConflict, `{"status":409,"code":"planning.archived","title":"Conflict","detail":"The cycle or module is archived; unarchive it to change it."}`},
		"taken address": {domain.ErrModuleLinkExists, http.StatusConflict, `{"status":409,"code":"planning.module_link_exists","title":"Conflict","detail":"This module already has this link."}`},
		"busy":          {shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		"failure":       {errGone, http.StatusInternalServerError, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	}
	links, link := "/api/v0/modules/"+billingID.String()+"/links", "/api/v0/module-links/"+specID.String()
	for _, tt := range []struct {
		method, path, body string
		refusals           []string
	}{
		{http.MethodGet, links, "", []string{"no module", "forbidden", "failure"}},
		{http.MethodPost, links, `{"url":"a"}`, []string{"invalid", "no module", "forbidden", "project", "modules off", "archived", "taken address", "busy", "failure"}},
		{http.MethodPatch, link, `{"url":"a"}`, []string{"invalid", "no link", "forbidden", "project", "modules off", "archived", "taken address", "busy", "failure"}},
		{http.MethodDelete, link, "", []string{"no link", "forbidden", "project", "modules off", "archived", "busy", "failure"}},
	} {
		for _, name := range tt.refusals {
			r := refusals[name]
			res, body := do(t, linkServer(t, &fakeLinks{err: r.err}), request(tt.method, tt.path, "alice", tt.body))
			if res.StatusCode != r.status || body != r.want+"\n" || r.want == busy && res.Header.Get("Retry-After") != "1" {
				t.Errorf("%s %s, %s = %d %s; want %d %s", tt.method, tt.path, name, res.StatusCode, body, r.status, r.want)
			}
		}
	}
}
