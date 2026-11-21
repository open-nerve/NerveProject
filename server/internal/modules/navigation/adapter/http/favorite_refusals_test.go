package httpadapter_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// favoriteRefusals are every refusal the favorites' operations declare, by
// name, and a failure: the views' (refusals) and the favorites' own.
func favoriteRefusals(t *testing.T) map[string]refusal {
	t.Helper()
	all := refusals(t)
	all["not allowed"] = refusal{shared.Invalid(shared.FieldError{Field: "entity_id", Code: shared.FieldNotAllowed, Message: "is no target you may favor"}),
		http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity",` +
			`"detail":"The request has invalid values.","errors":[{"field":"entity_id","code":"not_allowed","message":"is no target you may favor"}]}`}
	all["no favorite"] = refusal{domain.ErrFavoriteNotFound, http.StatusNotFound,
		`{"status":404,"code":"navigation.favorite_not_found","title":"Not Found","detail":"The favorite does not exist, or you cannot see it."}`}
	all["name taken"] = refusal{domain.ErrFavoriteFolderNameTaken, http.StatusConflict, `{"status":409,"code":"navigation.favorite_folder_name_taken",` +
		`"title":"Conflict","detail":"You have a folder of this name in this workspace already."}`}
	return all
}

// Each favorites' operation answers each refusal its x-problem-codes
// declare, and a failure, as the use case gives it: the problem, its status
// and its code. A favorite the caller may not have is 404, never 403.
func TestTheRefusalsOfEachFavoriteOperation(t *testing.T) {
	all := favoriteRefusals(t)
	for _, op := range []struct {
		method, path, body string
		refusals           []string
	}{
		{http.MethodGet, favoritesPath, "", []string{"no workspace", "forbidden", "failure"}},
		{http.MethodPost, favoritesPath, `{"entity_type":"folder","name":"Pinned"}`, []string{"not allowed", "no workspace", "forbidden", "name taken", "failure"}},
		{http.MethodPatch, favoritePath, `{"name":"Pinned"}`, []string{"not allowed", "no favorite", "forbidden", "name taken", "failure"}},
		{http.MethodDelete, favoritePath, "", []string{"no favorite", "forbidden", "failure"}},
	} {
		for _, name := range op.refusals {
			tt := all[name]
			f := &favoriteFakes{err: tt.err}
			res, body := do(t, newServer(t, f.useCases()), request(op.method, op.path, op.body))
			if res.StatusCode != tt.status || body != tt.want+"\n" {
				t.Errorf("%s %s, %s: %d %s, want %d %s", op.method, op.path, name, res.StatusCode, body, tt.status, tt.want)
			}
			if len(f.calls) != 1 {
				t.Errorf("%s %s, %s: the use cases got %+v, want one call", op.method, op.path, name, f.calls)
			}
		}
	}
}

// Every navigation.favorite_not_found on /favorites/{favorite_id} is one
// answer, value for value (P3c spec 2.7): its status, its headers and its
// body alike, whichever the method and the id. The use cases give the one
// value for every favorite the caller may not have, whatever the reason;
// the handlers answer it as it is.
func TestEveryFavoriteNotFoundIsOneAnswer(t *testing.T) {
	f := &favoriteFakes{err: domain.ErrFavoriteNotFound}
	h := newServer(t, f.useCases())
	type answer struct {
		status int
		header http.Header
		body   string
	}
	var first *answer
	for _, req := range []*http.Request{request(http.MethodPatch, favoritePath, `{}`), request(http.MethodDelete, favoritePath, ""),
		request(http.MethodPatch, "/api/v0/favorites/0199a2b4-0000-7000-8000-0000000000f7", `{"sequence":1}`),
		request(http.MethodDelete, "/api/v0/favorites/0199a2b4-0000-7000-8000-0000000000e1", "")} {
		req.Header.Set("X-Request-Id", "req-1")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		res := rec.Result()
		body, _ := io.ReadAll(res.Body)
		got := answer{res.StatusCode, res.Header, string(body)}
		if first == nil {
			first = &got
			if got.status != http.StatusNotFound || problemCode(t, got.body) != "navigation.favorite_not_found" {
				t.Fatalf("%s %s = %d %s, want 404 navigation.favorite_not_found", req.Method, req.URL.Path, got.status, got.body)
			}
			continue
		}
		if !reflect.DeepEqual(got, *first) {
			t.Errorf("%s %s = %+v, want %+v", req.Method, req.URL.Path, got, *first)
		}
	}
}

// A body that is not the creation's or the change's structure answers 400
// bad_request before any use case: not an object, a field it does not have
// (the project, which the target gives, Plane's is_folder and
// entity_identifier, the other body's fields), null for any field but a
// change's parent_id, a value of another type, an id that is no UUID, a
// number no float64 holds, and a creation without its kind. Which kinds
// there are and what each takes are the use case's (422).
func TestAFavoritesBodyHeldToItsStructure(t *testing.T) {
	const board = `"0199a2b4-0000-7000-8000-0000000000e1"`
	f := &favoriteFakes{}
	h := newServer(t, f.useCases())
	for _, body := range []string{`[]`, `"folder"`, `{}`, `{"name":"Pinned"}`, `{"entity_type":null}`, `{"entity_type":1}`,
		`{"entity_type":["folder"]}`, `{"entity_type":"project","entity_id":"web"}`, `{"entity_type":"project","entity_id":null}`,
		`{"entity_type":"folder","name":null}`, `{"entity_type":"folder","name":1}`, `{"entity_type":"view","entity_id":` + board + `,"parent_id":null}`,
		`{"entity_type":"view","entity_id":` + board + `,"parent_id":"pinned"}`, `{"entity_type":"view","entity_id":` + board + `,"project_id":` + board + `}`,
		`{"entity_type":"folder","name":"Pinned","is_folder":true}`, `{"entity_type":"view","entity_identifier":` + board + `}`,
		`{"entity_type":"folder","name":"Pinned","sequence":1}`} {
		if res, got := do(t, h, request(http.MethodPost, favoritesPath, body)); res.StatusCode != http.StatusBadRequest || problemCode(t, got) != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	for _, body := range []string{`[]`, `null`, `{"parent_id":"pinned"}`, `{"parent_id":1}`, `{"sequence":null}`, `{"sequence":"1"}`,
		`{"sequence":1e400}`, `{"name":null}`, `{"name":1}`, `{"entity_type":"folder"}`, `{"entity_id":` + board + `}`,
		`{"workspace_id":` + board + `}`, `{"logo_props":{}}`} {
		if res, got := do(t, h, request(http.MethodPatch, favoritePath, body)); res.StatusCode != http.StatusBadRequest || problemCode(t, got) != "bad_request" {
			t.Errorf("PATCH %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(f.calls) != 0 {
		t.Errorf("the use cases got %+v, want none", f.calls)
	}
}

// A favorite's id in the path that is no UUID answers 400 bad_request
// before any use case.
func TestAFavoritesIDThatIsNoUUIDAnswers400(t *testing.T) {
	f := &favoriteFakes{}
	h := newServer(t, f.useCases())
	for _, req := range []*http.Request{request(http.MethodPatch, "/api/v0/favorites/pinned", `{}`), request(http.MethodDelete, favoritePath+"x", "")} {
		if res, body := do(t, h, req); res.StatusCode != http.StatusBadRequest || problemCode(t, body) != "bad_request" {
			t.Errorf("%s %s = %d %s, want 400 bad_request", req.Method, req.URL.Path, res.StatusCode, body)
		}
	}
	if len(f.calls) != 0 {
		t.Errorf("the use cases got %+v, want none", f.calls)
	}
}
