package domain

import (
	"encoding/json"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// themes are the themes a profile can hold (M1-P2 handoff): web's
// THEME_OPTIONS, without custom.
var themes = []string{"system", "light", "dark", "light-contrast", "dark-contrast"}

// languages are the languages of the web UI (M1 design 6).
var languages = []string{"en", "zh-CN"}

// OnboardingSteps are the four steps of onboarding (M2 design 4.3).
type OnboardingSteps struct {
	ProfileComplete bool `json:"profile_complete"`
	WorkspaceCreate bool `json:"workspace_create"`
	WorkspaceInvite bool `json:"workspace_invite"`
	WorkspaceJoin   bool `json:"workspace_join"`
}

// Profile is an account's preferences as the API shows them.
type Profile struct {
	Theme           string
	Language        string
	StartOfTheWeek  int // 0 Sunday … 6 Saturday
	OnboardingStep  OnboardingSteps
	IsOnboarded     bool
	IsTourCompleted bool
	LastWorkspaceID *uuid.UUID // a workspace the client names; no foreign key (M2 design 3.2)
	UpdatedAt       time.Time
}

// OnboardingStepsPatch sets the steps that are not nil and leaves the
// others: the database merges it into the stored object (M2 design 3.14).
type OnboardingStepsPatch struct {
	ProfileComplete *bool `json:"profile_complete,omitempty"`
	WorkspaceCreate *bool `json:"workspace_create,omitempty"`
	WorkspaceInvite *bool `json:"workspace_invite,omitempty"`
	WorkspaceJoin   *bool `json:"workspace_join,omitempty"`
}

// JSON is the patch as the object the database merges in: only the steps
// set, {} for none.
func (p OnboardingStepsPatch) JSON() []byte {
	out, _ := json.Marshal(p) // four *bool fields: cannot fail
	return out
}

// ProfilePatch is a partial update of a profile: a nil field stays as it
// is. LastWorkspaceSet tells whether LastWorkspaceID is written; nil then
// clears it.
type ProfilePatch struct {
	Theme            *string
	Language         *string
	StartOfTheWeek   *int
	OnboardingStep   OnboardingStepsPatch
	IsOnboarded      *bool
	IsTourCompleted  *bool
	LastWorkspaceSet bool
	LastWorkspaceID  *uuid.UUID
}

// CheckProfilePatch checks p (M2 design 4.3): a theme and a language of
// their lists, a first day of the week from 0 to 6. The structure, down to
// the steps' keys and types, was checked at the boundary (M2 design 3.11).
// Every problem is reported at once, as one 422 validation_failed.
func CheckProfilePatch(p ProfilePatch) error {
	var fields []shared.FieldError
	if p.Theme != nil && !slices.Contains(themes, *p.Theme) {
		fields = append(fields, shared.FieldError{Field: "theme", Code: shared.FieldInvalidFormat, Message: "is not a known theme"})
	}
	if p.Language != nil && !slices.Contains(languages, *p.Language) {
		fields = append(fields, shared.FieldError{Field: "language", Code: shared.FieldInvalidFormat, Message: "is not a known language"})
	}
	if p.StartOfTheWeek != nil && (*p.StartOfTheWeek < 0 || *p.StartOfTheWeek > 6) {
		fields = append(fields, shared.FieldError{Field: "start_of_the_week", Code: shared.FieldOutOfRange, Message: "must be between 0 and 6"})
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}
