package domain

import (
	"errors"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestCheckProfilePatchAcceptsAValidPatch(t *testing.T) {
	// Web's THEME_OPTIONS (web/packages/constants/src/themes.ts), and the
	// CHECK of profiles.theme.
	for _, theme := range []string{"system", "light", "dark", "light-contrast", "dark-contrast"} {
		if err := CheckProfilePatch(ProfilePatch{Theme: &theme}); err != nil {
			t.Errorf("theme %q: %v", theme, err)
		}
	}
	for _, week := range []int{0, 6} {
		if err := CheckProfilePatch(ProfilePatch{Language: ptr("zh-CN"), StartOfTheWeek: &week}); err != nil {
			t.Errorf("week %d: %v", week, err)
		}
	}
}

func TestCheckProfilePatchReportsEveryField(t *testing.T) {
	err := CheckProfilePatch(ProfilePatch{Theme: ptr("custom"), Language: ptr("fr"), StartOfTheWeek: ptr(7)})

	want := []shared.FieldError{
		{Field: "theme", Code: "invalid_format", Message: "is not a known theme"},
		{Field: "language", Code: "invalid_format", Message: "is not a known language"},
		{Field: "start_of_the_week", Code: "out_of_range", Message: "must be between 0 and 6"},
	}
	var se *shared.Error
	if !errors.As(err, &se) || !slices.Equal(se.Fields, want) {
		t.Errorf("CheckProfilePatch() = %+v, want %+v", err, want)
	}
	if err := CheckProfilePatch(ProfilePatch{StartOfTheWeek: ptr(-1)}); err == nil {
		t.Error("CheckProfilePatch(week -1) = nil, want out_of_range")
	}
}

// The patch the database merges holds exactly the steps set.
func TestOnboardingStepsPatchJSON(t *testing.T) {
	tests := []struct {
		patch OnboardingStepsPatch
		want  string
	}{
		{OnboardingStepsPatch{}, `{}`},
		{OnboardingStepsPatch{ProfileComplete: ptr(true)}, `{"profile_complete":true}`},
		{OnboardingStepsPatch{WorkspaceCreate: ptr(false), WorkspaceJoin: ptr(true)}, `{"workspace_create":false,"workspace_join":true}`},
		{OnboardingStepsPatch{ProfileComplete: ptr(false), WorkspaceCreate: ptr(false), WorkspaceInvite: ptr(false), WorkspaceJoin: ptr(false)},
			`{"profile_complete":false,"workspace_create":false,"workspace_invite":false,"workspace_join":false}`},
	}
	for _, tt := range tests {
		if got := string(tt.patch.JSON()); got != tt.want {
			t.Errorf("JSON() = %s, want %s", got, tt.want)
		}
	}
}
