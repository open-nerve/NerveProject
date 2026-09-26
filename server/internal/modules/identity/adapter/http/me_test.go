package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeUpdateMe struct {
	calls int
	patch domain.UserPatch
	user  domain.User
	err   error
}

func (f *fakeUpdateMe) Execute(_ context.Context, p domain.UserPatch) (domain.User, error) {
	f.calls++
	f.patch = p
	return f.user, f.err
}

// patchJSON is a PATCH with the bearer token fakeAuth accepts.
func patchJSON(path, body string) *http.Request {
	req := withToken(httptest.NewRequest(http.MethodPatch, path, strings.NewReader(body)))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// asJSON shows a patch's pointers by what they point to: null for nil.
func asJSON(v any) string {
	out, _ := json.Marshal(v)
	return string(out)
}

func TestUpdateMe(t *testing.T) {
	update := &fakeUpdateMe{user: domain.User{
		ID: userID, Email: "alice@corp.com", FirstName: "Ann", DisplayName: "alice", Timezone: "Asia/Shanghai", CreatedAt: created,
	}}
	req := patchJSON("/api/v0/me", `{"first_name":"Ann","user_timezone":"Asia/Shanghai"}`)
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{updateMe: update}), req)

	want := `{"avatar_url":null,"cover_image_url":null,"created_at":"2026-09-25T10:00:00.123456Z","display_name":"alice",` +
		`"email":"alice@corp.com","first_name":"Ann","id":"` + userID.String() + `","last_name":"","user_timezone":"Asia/Shanghai"}` + "\n"
	if res.StatusCode != http.StatusOK || body != want {
		t.Errorf("PATCH /me = %d %s, want 200 %s", res.StatusCode, body, want)
	}
	if got, want := asJSON(update.patch), `{"FirstName":"Ann","LastName":null,"DisplayName":null,"Timezone":"Asia/Shanghai"}`; got != want {
		t.Errorf("use case got %s, want %s", got, want)
	}
}

// The structure is checked before the handler (M2 design 3.11): a null name
// and the e-mail address, which only the administrator changes, are 400 and
// the use case never runs. The use case's problems pass through.
func TestUpdateMeProblems(t *testing.T) {
	tests := []struct {
		name, body string
		err        error
		status     int
		want       string
		ran        bool
	}{
		{"null name", `{"first_name":null}`, nil, 400, `"errors":[{"field":"first_name","code":"invalid_format"`, false},
		{"e-mail address", `{"email":"a@b.co"}`, nil, 400, `"errors":[{"field":"email","code":"not_allowed"`, false},
		{"invalid values", `{"display_name":""}`, shared.Invalid(shared.FieldError{Field: "display_name", Code: shared.FieldTooShort, Message: "must not be empty"}),
			422, `"errors":[{"field":"display_name","code":"too_short"`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := &fakeUpdateMe{err: tt.err}

			res, body := do(t, newServer(t, fakes{updateMe: update}), patchJSON("/api/v0/me", tt.body))

			if res.StatusCode != tt.status || !strings.Contains(body, tt.want) {
				t.Errorf("response = %d %s, want %d with %s", res.StatusCode, body, tt.status, tt.want)
			}
			if ran := update.calls == 1; ran != tt.ran {
				t.Errorf("use case ran: %v, want %v", ran, tt.ran)
			}
		})
	}
}
