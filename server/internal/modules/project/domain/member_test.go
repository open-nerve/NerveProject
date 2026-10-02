package domain

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// One member, and a hundred, each of the three roles: accepted.
func TestCheckNewMembersAccepts(t *testing.T) {
	hundred := make([]NewMember, MaxNewMembers)
	for i := range hundred {
		hundred[i] = NewMember{MemberID: uuid.NewV7(), Role: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}[i%3]}
	}
	for _, in := range [][]NewMember{{{MemberID: uuid.NewV7(), Role: shared.RoleGuest}}, hundred} {
		if err := CheckNewMembers(in); err != nil {
			t.Errorf("CheckNewMembers(%d members) = %v, want nil", len(in), err)
		}
	}
}

// None, a hundred and one, a role outside the three, an account named
// twice: each refused, all of them in one 422 that names each by its place
// in the request, the members' problems of a list too long as well.
func TestCheckNewMembersReportsEveryProblem(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	many := make([]NewMember, MaxNewMembers+1)
	for i := range many {
		many[i] = NewMember{MemberID: uuid.NewV7(), Role: shared.RoleMember}
	}
	tests := []struct {
		name string
		in   []NewMember
		want []shared.FieldError
	}{
		{"none", nil, []shared.FieldError{{Field: "members", Code: "too_short", Message: "must name a member"}}},
		{"a hundred and one", many, []shared.FieldError{{Field: "members", Code: "too_long", Message: "must name at most 100 members"}}},
		{"a hundred and one, the last a repeat of no role", append(append([]NewMember{}, many[:MaxNewMembers]...), NewMember{MemberID: many[0].MemberID}),
			[]shared.FieldError{{Field: "members", Code: "too_long", Message: "must name at most 100 members"},
				{Field: "members[100].member_id", Code: "duplicate", Message: "is listed before"},
				{Field: "members[100].role", Code: "invalid_format", Message: "is not 5, 15 or 20"}}},
		{"roles and repeats", []NewMember{{MemberID: a, Role: 10}, {MemberID: b, Role: shared.RoleAdmin}, {MemberID: a, Role: shared.RoleGuest},
			{MemberID: b, Role: 0}}, []shared.FieldError{
			{Field: "members[0].role", Code: "invalid_format", Message: "is not 5, 15 or 20"},
			{Field: "members[2].member_id", Code: "duplicate", Message: "is listed before"},
			{Field: "members[3].member_id", Code: "duplicate", Message: "is listed before"},
			{Field: "members[3].role", Code: "invalid_format", Message: "is not 5, 15 or 20"},
		}},
	}
	for _, tt := range tests {
		var e *shared.Error
		if err := CheckNewMembers(tt.in); !errors.As(err, &e) || e.Code != "validation_failed" || !reflect.DeepEqual(e.Fields, tt.want) {
			t.Errorf("%s: CheckNewMembers() = %v; want validation_failed with %+v", tt.name, err, tt.want)
		}
	}
}

// A workspace admin is added as an admin alone, a workspace guest as a
// guest alone, a workspace member as any of the three; nobody of a role
// outside the three, as nothing outside them (M3 design 3.5).
func TestCanAdd(t *testing.T) {
	roles := []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest, 10, 25}
	allowed := map[[2]shared.Role]bool{
		{shared.RoleAdmin, shared.RoleAdmin}: true, {shared.RoleMember, shared.RoleAdmin}: true, {shared.RoleMember, shared.RoleMember}: true,
		{shared.RoleMember, shared.RoleGuest}: true, {shared.RoleGuest, shared.RoleGuest}: true,
	}
	for _, ws := range roles {
		for _, role := range roles {
			if got := CanAdd(ws, role); got != allowed[[2]shared.Role{ws, role}] {
				t.Errorf("CanAdd(%d, %d) = %v, want %v", ws, role, got, !got)
			}
		}
	}
}

// Each target is held to what was read of him: no workspace role, a
// member already, a role his workspace role does not allow; each refused
// by his place, in one 422, a member's role not looked at; the others
// pass.
func TestCheckTargets(t *testing.T) {
	role := func(r shared.Role) *shared.Role { return &r }
	target := func(r shared.Role, ws *shared.Role, member bool) Target {
		return Target{NewMember: NewMember{MemberID: uuid.NewV7(), Role: r}, WorkspaceRole: ws, Member: member}
	}
	if err := CheckTargets([]Target{target(shared.RoleAdmin, role(shared.RoleAdmin), false), target(shared.RoleGuest, role(shared.RoleMember), false),
		target(shared.RoleGuest, role(shared.RoleGuest), false)}); err != nil {
		t.Errorf("CheckTargets() of allowed targets = %v, want nil", err)
	}
	roleProblem := "is not one his workspace role allows: a workspace admin joins as an admin, a guest as a guest"
	want := []shared.FieldError{
		{Field: "members[0].member_id", Code: "not_allowed", Message: "must be an active member of the workspace"},
		{Field: "members[1].member_id", Code: "duplicate", Message: "is an active member of the project already"},
		{Field: "members[2].role", Code: "not_allowed", Message: roleProblem},
		{Field: "members[4].role", Code: "not_allowed", Message: roleProblem},
	}
	err := CheckTargets([]Target{target(shared.RoleMember, nil, false), target(shared.RoleAdmin, role(shared.RoleGuest), true),
		target(shared.RoleMember, role(shared.RoleAdmin), false), target(shared.RoleAdmin, role(shared.RoleMember), false),
		target(shared.RoleMember, role(shared.RoleGuest), false)})
	var e *shared.Error
	if !errors.As(err, &e) || e.Code != "validation_failed" || !reflect.DeepEqual(e.Fields, want) {
		t.Errorf("CheckTargets() = %v; want validation_failed with %+v", err, want)
	}
}

// A new membership takes the workspace role; an ended one the lesser of
// its role and the workspace role (M3 design 9.1's table, and the two
// equal cases). The lesser is roleOrder's, not the numbers': a role outside
// the three is below each of them, where by the numbers 25 would keep the
// ended admin's.
func TestJoinRole(t *testing.T) {
	role := func(r shared.Role) *shared.Role { return &r }
	for _, tt := range []struct {
		ended     *shared.Role
		workspace shared.Role
		want      shared.Role
	}{
		{nil, shared.RoleMember, shared.RoleMember},
		{nil, shared.RoleAdmin, shared.RoleAdmin},
		{role(shared.RoleGuest), shared.RoleMember, shared.RoleGuest},
		{role(shared.RoleAdmin), shared.RoleMember, shared.RoleMember},
		{role(shared.RoleMember), shared.RoleAdmin, shared.RoleMember},
		{role(shared.RoleAdmin), shared.RoleGuest, shared.RoleGuest},
		{role(shared.RoleMember), shared.RoleMember, shared.RoleMember},
		{role(shared.RoleAdmin), shared.RoleAdmin, shared.RoleAdmin},
		{role(shared.RoleAdmin), 25, 25},
	} {
		if got := JoinRole(tt.ended, tt.workspace); got != tt.want {
			t.Errorf("JoinRole(%s, %d) = %d, want %d", show(tt.ended), tt.workspace, got, tt.want)
		}
	}
}

// show is r as a failure prints it.
func show(r *shared.Role) string {
	if r == nil {
		return "none"
	}
	return fmt.Sprint(*r)
}
