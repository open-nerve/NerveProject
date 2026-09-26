package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
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

type fakeChangePassword struct {
	calls int
	in    app.ChangePasswordInput
	err   error
}

func (f *fakeChangePassword) Execute(_ context.Context, in app.ChangePasswordInput) error {
	f.calls++
	f.in = in
	return f.err
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

// Each field sent reaches the use case; an absent one stays nil.
func TestUpdateMe(t *testing.T) {
	tests := []struct {
		name, body, patch string
	}{
		{"every field", `{"first_name":"Ann","last_name":"Lee","display_name":"annie","user_timezone":"Asia/Shanghai"}`,
			`{"FirstName":"Ann","LastName":"Lee","DisplayName":"annie","Timezone":"Asia/Shanghai"}`},
		{"a name and the time zone", `{"first_name":"Ann","user_timezone":"Asia/Shanghai"}`,
			`{"FirstName":"Ann","LastName":null,"DisplayName":null,"Timezone":"Asia/Shanghai"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := &fakeUpdateMe{user: domain.User{
				ID: userID, Email: "alice@corp.com", FirstName: "Ann", LastName: "Lee", DisplayName: "alice", Timezone: "Asia/Shanghai", CreatedAt: created,
			}}
			req := patchJSON("/api/v0/me", tt.body)
			apitest.Load(t).CheckRequest(t, req)

			res, body := do(t, newServer(t, fakes{updateMe: update}), req)

			want := `{"avatar_url":null,"cover_image_url":null,"created_at":"2026-09-25T10:00:00.123456Z","display_name":"alice",` +
				`"email":"alice@corp.com","first_name":"Ann","id":"` + userID.String() + `","last_name":"Lee","user_timezone":"Asia/Shanghai"}` + "\n"
			if res.StatusCode != http.StatusOK || body != want {
				t.Errorf("PATCH /me = %d %s, want 200 %s", res.StatusCode, body, want)
			}
			if got := asJSON(update.patch); got != tt.patch {
				t.Errorf("use case got %s, want %s", got, tt.patch)
			}
		})
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

func TestChangePassword(t *testing.T) {
	change := &fakeChangePassword{}
	req := withToken(postJSON("/api/v0/me/change-password", `{"current_password":"Tr0ub4dor&3","new_password":"N3w-Passw0rd!"}`))
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{change: change}), req)

	want := app.ChangePasswordInput{Current: "Tr0ub4dor&3", New: "N3w-Passw0rd!"}
	if res.StatusCode != http.StatusNoContent || body != "" || change.in != want {
		t.Errorf("POST /me/change-password = %d %q, use case got %+v; want 204 for %+v", res.StatusCode, body, change.in, want)
	}
}

// The handler exit: every error the use case returns becomes its problem.
func TestChangePasswordProblems(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		status     int
		code       string
		retryAfter string
	}{
		{"weak new password", shared.Invalid(shared.FieldError{Field: "new_password", Code: shared.FieldWeakPassword, Message: "is weak"}),
			422, "validation_failed", ""},
		{"wrong current password", domain.ErrCurrentPasswordIncorrect, 422, "identity.current_password_incorrect", ""},
		{"hashing saturated", shared.ServerBusy(time.Second), 503, "server_busy", "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, body := do(t, newServer(t, fakes{change: &fakeChangePassword{err: tt.err}}),
				withToken(postJSON("/api/v0/me/change-password", `{"current_password":"x","new_password":"y"}`)))

			if res.StatusCode != tt.status || !strings.Contains(body, `"code":"`+tt.code+`"`) || res.Header.Get("Retry-After") != tt.retryAfter {
				t.Errorf("response = %d %s Retry-After %q, want %d %s", res.StatusCode, body, res.Header.Get("Retry-After"), tt.status, tt.code)
			}
		})
	}
}
