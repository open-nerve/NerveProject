package domain_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fieldsOf are err's field problems as "field code", or nil when err is no
// 422 validation_failed.
func fieldsOf(err error) []string {
	var e *shared.Error
	if !errors.As(err, &e) || !errors.Is(err, shared.Invalid()) {
		return nil
	}
	var out []string
	for _, f := range e.Fields {
		out = append(out, f.Field+" "+f.Code)
	}
	return out
}

// A valid batch comes back in the request's order, each address
// normalized as registration does it (M3 design 3.13), each role as sent.
func TestCheckInvitationsNormalizes(t *testing.T) {
	batch := []domain.NewInvitation{{" Zoe@Corp.COM\t", shared.RoleAdmin}, {"elodie@exämple.com", shared.RoleGuest}, {"bob@corp.com", shared.RoleMember}}

	got, err := domain.CheckInvitations(batch)

	want := []domain.NewInvitation{{"zoe@corp.com", shared.RoleAdmin}, {"elodie@exämple.com", shared.RoleGuest}, {"bob@corp.com", shared.RoleMember}}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("CheckInvitations() = %+v, %v; want %+v", got, err, want)
	}
}

// Every problem the request alone shows is in one answer, each on its
// invitation's index in the request: an invalid address, one listed again
// after normalization (the later ones), a role outside the three, compared
// by set. The limits of the batch come first, alone.
func TestCheckInvitationsRefuses(t *testing.T) {
	many := make([]domain.NewInvitation, domain.MaxInvitations+1)
	for i := range many {
		many[i] = domain.NewInvitation{Email: "someone" + strings.Repeat("x", i) + "@corp.com", Role: shared.RoleGuest}
	}
	tests := []struct {
		name  string
		batch []domain.NewInvitation
		want  []string
	}{
		{"no invitation", nil, []string{"invitations too_short"}},
		{"one too many", many, []string{"invitations too_long"}},
		{"an invalid address", []domain.NewInvitation{{"bob@corp.com", 5}, {"not an address", 5}}, []string{"invitations[1].email invalid_format"}},
		{"an empty address", []domain.NewInvitation{{" ", 15}}, []string{"invitations[0].email invalid_format"}},
		{"an address too long", []domain.NewInvitation{{strings.Repeat("a", 250) + "@corp.com", 15}}, []string{"invitations[0].email invalid_format"}},
		{"an address twice, once in upper case", []domain.NewInvitation{{"bob@corp.com", 5}, {"carol@corp.com", 5}, {" BOB@corp.com", 15},
			{"bob@corp.com", 20}}, []string{"invitations[2].email duplicate", "invitations[3].email duplicate"}},
		{"role 10, between two roles", []domain.NewInvitation{{"bob@corp.com", 10}}, []string{"invitations[0].role invalid_format"}},
		{"role 0", []domain.NewInvitation{{"bob@corp.com", 0}}, []string{"invitations[0].role invalid_format"}},
		{"role 25, above the admin", []domain.NewInvitation{{"bob@corp.com", 25}}, []string{"invitations[0].role invalid_format"}},
		{"every problem at once", []domain.NewInvitation{{"x", 1}, {"bob@corp.com", 5}, {"Bob@corp.com", 21}},
			[]string{"invitations[0].email invalid_format", "invitations[0].role invalid_format", "invitations[2].email duplicate",
				"invitations[2].role invalid_format"}},
	}
	for _, tt := range tests {
		got, err := domain.CheckInvitations(tt.batch)
		if got != nil || !slices.Equal(fieldsOf(err), tt.want) {
			t.Errorf("%s: CheckInvitations() = %+v, %v (%q); want %q", tt.name, got, err, fieldsOf(err), tt.want)
		}
	}
	if _, err := domain.CheckInvitations(many[:domain.MaxInvitations]); err != nil {
		t.Errorf("CheckInvitations() of %d invitations = %v, want them allowed", domain.MaxInvitations, err)
	}
}

// The problems found after the request is read name the request's index.
func TestTheProblemsOfAnAddressNameItsIndex(t *testing.T) {
	for _, tt := range []struct {
		got  shared.FieldError
		want string
	}{
		{domain.MemberAddress(3), "invitations[3].email not_allowed"},
		{domain.InvitedAddress(0), "invitations[0].email duplicate"},
		{domain.InvitedAddress(12), "invitations[12].email duplicate"},
	} {
		if got := tt.got.Field + " " + tt.got.Code; got != tt.want {
			t.Errorf("%s, want %s", got, tt.want)
		}
	}
}
