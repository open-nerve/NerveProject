package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newUpdate is UpdateProject over newWrites' fakes, its clock logged.
func newUpdate() (*app.UpdateProject, *writeFixture) {
	f := newWrites()
	return app.NewUpdateProject(f.store, f.locks(), f.tx, clockAt{clockNow, f.log}), f
}

// updated are the calls of bob's update of web with p, which names the
// accounts assignees as its lead and default assignee: the locks, the
// decision, the assignees' memberships when it names any, the clock, the
// change, the answer.
func updated(p domain.ProjectPatch, assignees ...uuid.UUID) []string {
	patch, _ := json.Marshal(p)
	calls := lockedDecision(bob, webID, domain.ActionUpdate)
	if len(assignees) > 0 {
		calls = append(calls, fmt.Sprintf("Memberships %s %v", webID, assignees))
	}
	return append(calls, "Now", fmt.Sprintf("UpdateProject %s %s by %s at %s", webID, patch, bob, clockNow.Format(timeFormat)),
		fmt.Sprintf("GetProject %s for %s", webID, bob))
}

// UpdateProject checks the patch, then, in one transaction and in the
// order of M3 design 3.6, locks the project, decides, checks the lead and
// the default assignee it names against their memberships of the project,
// reads the clock and changes the project, the identifier upper-cased, by
// the caller; the answer is the project read back as stored, as the caller
// sees it. A patch that names no assignee, or clears them, reads no
// membership.
func TestUpdateProject(t *testing.T) {
	tests := []struct {
		name      string
		in        domain.ProjectPatch
		stored    domain.ProjectPatch
		assignees []uuid.UUID
	}{
		{"the lead a member, the default assignee the admin", domain.ProjectPatch{Name: ptr("Site"), Identifier: ptr("site"), SetLead: true,
			LeadID: &alice, SetDefaultAssignee: true, DefaultAssigneeID: &bob},
			domain.ProjectPatch{Name: ptr("Site"), Identifier: ptr("SITE"), SetLead: true, LeadID: &alice, SetDefaultAssignee: true,
				DefaultAssigneeID: &bob}, []uuid.UUID{alice, bob}},
		{"both cleared", domain.ProjectPatch{SetLead: true, SetDefaultAssignee: true, ArchiveIn: ptr(3)},
			domain.ProjectPatch{SetLead: true, SetDefaultAssignee: true, ArchiveIn: ptr(3)}, nil},
		{"nothing", domain.ProjectPatch{}, domain.ProjectPatch{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdate()

			got, err := uc.Execute(as(bob), webID, tt.in)

			if want := updated(tt.stored, tt.assignees...); err != nil || !slices.Equal(f.log.calls, want) {
				t.Errorf("Execute() = %v, calls\n%q\nwant\n%q", err, f.log.calls, want)
			}
			if got.ID != webID || got.UpdatedAt != now || got.MemberRole == nil || *got.MemberRole != shared.RoleAdmin {
				t.Errorf("Execute() = %+v; want web as stored at %v, the caller its admin", got, now)
			}
		})
	}
}

// Refusals, each in its place, and nothing changed:
//   - the request's values, and no caller, before the transaction;
//   - a project not there, its workspace deleted while its lock waited, the
//     project deleted while its own lock waited (never 403), the project of
//     another workspace by the time it is locked, or not visible: 404,
//     before the decision or after it, whatever the patch names;
//   - a project member: the Authorizer's 403, an invalid lead too;
//   - an archived project: 409, after the decision, before the assignees;
//   - a lead or a default assignee who is not an active member of the
//     project, or is its guest: one 422 naming each, after the decision.
func TestUpdateProjectRefuses(t *testing.T) {
	leads := func(lead *uuid.UUID, assignee *uuid.UUID) domain.ProjectPatch {
		return domain.ProjectPatch{SetLead: lead != nil, LeadID: lead, SetDefaultAssignee: assignee != nil, DefaultAssigneeID: assignee}
	}
	decided := func(user, project uuid.UUID) []string { return lockedDecision(user, project, domain.ActionUpdate) }
	memberships := func(accounts ...uuid.UUID) string { return fmt.Sprintf("Memberships %s %v", webID, accounts) }
	unassignable := func(fields ...string) error {
		var problems []shared.FieldError
		for _, f := range fields {
			problems = append(problems, domain.Unassignable(f))
		}
		return shared.Invalid(problems...)
	}
	tests := []struct {
		name    string
		ctx     context.Context
		project uuid.UUID
		in      domain.ProjectPatch
		want    error
		calls   []string
	}{
		{"archive_in 13", as(bob), webID, domain.ProjectPatch{ArchiveIn: ptr(13)},
			shared.Invalid(shared.FieldError{Field: "archive_in", Code: "out_of_range"}), nil},
		{"no caller", context.Background(), webID, domain.ProjectPatch{}, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), domain.ProjectPatch{}, domain.ErrNotFound, noProject},
		{"the workspace gone", as(bob), webID, leads(&dave, nil), domain.ErrNotFound, lockedTo(webID)},
		{"moved to another workspace", as(bob), webID, leads(&dave, nil), domain.ErrNotFound,
			append(lockedTo(webID), "LockProject "+webID.String())},
		{"deleted while its lock waited", as(bob), webID, leads(&dave, nil), domain.ErrNotFound,
			append(lockedTo(webID), "LockProject "+webID.String())},
		{"not seen", as(erin), webID, leads(&dave, nil), domain.ErrNotFound, decided(erin, webID)},
		{"a project member", as(alice), webID, leads(&dave, nil), shared.Forbidden(), decided(alice, webID)},
		{"archived", as(bob), opsID, leads(&dave, nil), domain.ErrArchived, decided(bob, opsID)},
		{"a guest as the lead", as(bob), webID, leads(&carol, nil), unassignable("project_lead_id"), append(decided(bob, webID), memberships(carol))},
		{"an ended member as the lead", as(bob), webID, leads(&dave, nil), unassignable("project_lead_id"),
			append(decided(bob, webID), memberships(dave))},
		{"no member as the lead", as(bob), webID, leads(&erin, nil), unassignable("project_lead_id"), append(decided(bob, webID), memberships(erin))},
		{"a guest as the default assignee", as(bob), webID, leads(nil, &carol), unassignable("default_assignee_id"),
			append(decided(bob, webID), memberships(carol))},
		{"both", as(bob), webID, leads(&erin, &dave), unassignable("project_lead_id", "default_assignee_id"),
			append(decided(bob, webID), memberships(erin, dave))},
		{"a member lead, a guest default assignee", as(bob), webID, leads(&alice, &carol), unassignable("default_assignee_id"),
			append(decided(bob, webID), memberships(alice, carol))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdate()
			f.workspaces.gone = tt.name == "the workspace gone"
			f.store.deleted = tt.name == "deleted while its lock waited"
			if tt.name == "moved to another workspace" {
				f.store.moved = uuid.NewV7()
			}

			got, err := uc.Execute(tt.ctx, tt.project, tt.in)

			if !sameError(err, tt.want) || got.ID != (uuid.UUID{}) || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after; a project the store cannot read back
// after the change is an internal error, not a 404.
func TestUpdateProjectReturnsEachFailure(t *testing.T) {
	failure := errors.New("disk full")
	in := domain.ProjectPatch{SetLead: true, LeadID: &alice}
	all := updated(in, alice)
	tests := []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of all's ran
	}{
		{"the project's workspace", func(f *writeFixture) { f.store.errs = map[string]error{"ProjectWorkspace": failure} }, 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = failure }, 3},
		{"the lock", func(f *writeFixture) { f.store.errs = map[string]error{"LockProject": failure} }, 4},
		{"the decision", func(f *writeFixture) { f.auth.errs = map[grantKey]error{{bob, acme.ID}: failure} }, 5},
		{"the memberships", func(f *writeFixture) { f.store.errs = map[string]error{"Memberships": failure} }, 6},
		{"the change", func(f *writeFixture) { f.store.errs = map[string]error{"UpdateProject": failure} }, 8},
		{"the answer", func(f *writeFixture) { f.store.errs = map[string]error{"GetProject": failure} }, 9},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = failure }, 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdate()
			tt.fail(f)
			got, err := uc.Execute(as(bob), webID, in)
			if !errors.Is(err, failure) || got.ID != (uuid.UUID{}) || !slices.Equal(f.log.calls, all[:tt.calls]) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v after %q", got, err, f.log.calls, failure, all[:tt.calls])
			}
		})
	}
	uc, f := newUpdate()
	f.store.missing = true
	var e *shared.Error
	if got, err := uc.Execute(as(bob), webID, in); err == nil || errors.As(err, &e) || got.ID != (uuid.UUID{}) {
		t.Errorf("Execute() with the project gone = %+v, %v; want an internal error", got, err)
	}
}

// The store's conflicts come back as themselves: 409 of the identifier or
// the name.
func TestUpdateProjectAnswersTheStoresConflicts(t *testing.T) {
	for _, conflict := range []error{domain.ErrIdentifierTaken, domain.ErrNameTaken} {
		uc, f := newUpdate()
		f.store.errs = map[string]error{"UpdateProject": conflict}
		if _, err := uc.Execute(as(bob), webID, domain.ProjectPatch{Identifier: ptr("OPS")}); !errors.Is(err, conflict) {
			t.Errorf("Execute() = %v, want %v", err, conflict)
		}
	}
}
