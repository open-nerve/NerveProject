package httpadapter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeGetProfile struct{ profile domain.Profile }

func (f *fakeGetProfile) Execute(context.Context) (domain.Profile, error) { return f.profile, nil }

type fakeUpdateProfile struct {
	calls   int
	patch   domain.ProfilePatch
	profile domain.Profile
	err     error
}

func (f *fakeUpdateProfile) Execute(_ context.Context, p domain.ProfilePatch) (domain.Profile, error) {
	f.calls++
	f.patch = p
	return f.profile, f.err
}

var workspaceID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000007")

func TestGetProfile(t *testing.T) {
	get := &fakeGetProfile{profile: domain.Profile{
		Theme: "system", Language: "en", OnboardingStep: domain.OnboardingSteps{ProfileComplete: true}, UpdatedAt: created,
	}}
	req := withToken(httptest.NewRequest(http.MethodGet, "/api/v0/me/profile", nil))
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{getProfile: get}), req)

	want := `{"is_onboarded":false,"is_tour_completed":false,"language":"en","last_workspace_id":null,` +
		`"onboarding_step":{"profile_complete":true,"workspace_create":false,"workspace_invite":false,"workspace_join":false},` +
		`"start_of_the_week":0,"theme":"system","updated_at":"2026-09-25T10:00:00.123456Z"}` + "\n"
	if res.StatusCode != http.StatusOK || body != want {
		t.Errorf("GET /me/profile = %d %s, want 200 %s", res.StatusCode, body, want)
	}
}

// Each field sent reaches the use case; an absent one stays nil, and the
// workspace is written only when sent, null clearing it.
func TestUpdateProfile(t *testing.T) {
	tests := []struct {
		name, body, patch string
	}{
		{"every field",
			`{"theme":"dark","language":"zh-CN","start_of_the_week":1,"onboarding_step":{"workspace_join":true},` +
				`"is_onboarded":true,"is_tour_completed":false,"last_workspace_id":"` + workspaceID.String() + `"}`,
			`{"Theme":"dark","Language":"zh-CN","StartOfTheWeek":1,"OnboardingStep":{"workspace_join":true},` +
				`"IsOnboarded":true,"IsTourCompleted":false,"LastWorkspaceSet":true,"LastWorkspaceID":"` + workspaceID.String() + `"}`},
		{"one step, the workspace cleared", `{"onboarding_step":{"profile_complete":true},"last_workspace_id":null}`,
			`{"Theme":null,"Language":null,"StartOfTheWeek":null,"OnboardingStep":{"profile_complete":true},` +
				`"IsOnboarded":null,"IsTourCompleted":null,"LastWorkspaceSet":true,"LastWorkspaceID":null}`},
		{"nothing", `{}`,
			`{"Theme":null,"Language":null,"StartOfTheWeek":null,"OnboardingStep":{},` +
				`"IsOnboarded":null,"IsTourCompleted":null,"LastWorkspaceSet":false,"LastWorkspaceID":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := &fakeUpdateProfile{profile: domain.Profile{
				Theme: "dark", Language: "zh-CN", StartOfTheWeek: 1, OnboardingStep: domain.OnboardingSteps{WorkspaceJoin: true},
				IsOnboarded: true, LastWorkspaceID: &workspaceID, UpdatedAt: created,
			}}
			req := patchJSON("/api/v0/me/profile", tt.body)
			apitest.Load(t).CheckRequest(t, req)

			res, body := do(t, newServer(t, fakes{updateProfile: update}), req)

			want := `{"is_onboarded":true,"is_tour_completed":false,"language":"zh-CN","last_workspace_id":"` + workspaceID.String() + `",` +
				`"onboarding_step":{"profile_complete":false,"workspace_create":false,"workspace_invite":false,"workspace_join":true},` +
				`"start_of_the_week":1,"theme":"dark","updated_at":"2026-09-25T10:00:00.123456Z"}` + "\n"
			if res.StatusCode != http.StatusOK || body != want {
				t.Errorf("PATCH /me/profile = %d %s, want 200 %s", res.StatusCode, body, want)
			}
			if got := asJSON(update.patch); got != tt.patch {
				t.Errorf("use case got %s, want %s", got, tt.patch)
			}
		})
	}
}

// The structure, down to the steps' keys, is checked before the handler
// (M2 design 3.11): the use case never runs. The use case's problems pass
// through.
func TestUpdateProfileProblems(t *testing.T) {
	tests := []struct {
		name, body string
		err        error
		status     int
		want       string
		ran        bool
	}{
		{"unknown step", `{"onboarding_step":{"profile_completed":true}}`, nil, 400,
			`"errors":[{"field":"onboarding_step.profile_completed","code":"not_allowed"`, false},
		{"null theme", `{"theme":null}`, nil, 400, `"errors":[{"field":"theme","code":"invalid_format"`, false},
		{"workspace not a uuid", `{"last_workspace_id":"x"}`, nil, 400, `"errors":[{"field":"last_workspace_id","code":"invalid_format"`, false},
		{"invalid values", `{"theme":"neon"}`, shared.Invalid(shared.FieldError{Field: "theme", Code: shared.FieldInvalidFormat, Message: "is not a known theme"}),
			422, `"errors":[{"field":"theme","code":"invalid_format"`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := &fakeUpdateProfile{err: tt.err}

			res, body := do(t, newServer(t, fakes{updateProfile: update}), patchJSON("/api/v0/me/profile", tt.body))

			if res.StatusCode != tt.status || !strings.Contains(body, tt.want) {
				t.Errorf("response = %d %s, want %d with %s", res.StatusCode, body, tt.status, tt.want)
			}
			if ran := update.calls == 1; ran != tt.ran {
				t.Errorf("use case ran: %v, want %v", ran, tt.ran)
			}
		})
	}
}
