package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A read that fails answers its error, never a plausible answer: not "not a
// member", which the Authorizer would turn into workspace.not_found; not
// "free", "none" or app.ErrNotFound. Each read runs on a cancelled context
// against a workspace alice administers and has display settings in, so
// that the right answer is none of the zero values.
func TestAFailedReadIsAnErrorNotAnAnswer(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	tabbed := "TABBED"
	upsert(t, s, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: w.ID, UserID: alice,
		Patch: domain.PreferencesPatch{NavigationControl: &tabbed}, Now: now})
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	failed := func(err error) bool { return errors.Is(err, context.Canceled) }

	if role, ok, err := s.ActiveRole(cancelled, w.ID, alice); !failed(err) || ok || role != 0 {
		t.Errorf("ActiveRole() = %d, %v, %v; want context.Canceled, not a member", role, ok, err)
	}
	if taken, err := s.SlugTaken(cancelled, "acme"); !failed(err) || taken {
		t.Errorf("SlugTaken() = %v, %v; want context.Canceled", taken, err)
	}
	if list, err := s.ListWorkspaces(cancelled, alice); !failed(err) || list != nil {
		t.Errorf("ListWorkspaces() = %v, %v; want context.Canceled, no list", list, err)
	}
	if got, err := s.WorkspaceBySlug(cancelled, "acme"); !failed(err) || errors.Is(err, app.ErrNotFound) || !sameWorkspace(got, domain.Workspace{}) {
		t.Errorf("WorkspaceBySlug() = %+v, %v; want context.Canceled, not app.ErrNotFound", got, err)
	}
	if p, found, err := s.Preferences(cancelled, w.ID, alice); !failed(err) || found || p != (domain.Preferences{}) {
		t.Errorf("Preferences() = %+v, %v, %v; want context.Canceled, not no row", p, found, err)
	}
	if list, err := s.ListMembers(cancelled, w.ID); !failed(err) || list != nil {
		t.Errorf("ListMembers() = %v, %v; want context.Canceled, no list", list, err)
	}
	bob := joinAt(t, s, w.ID, newAccount(t, pool, "bob@corp.com"), shared.RoleMember, now)
	if got, err := s.MemberByID(cancelled, bob.ID); !failed(err) || errors.Is(err, app.ErrNotFound) || got != (domain.Membership{}) {
		t.Errorf("MemberByID() = %+v, %v; want context.Canceled, not app.ErrNotFound", got, err)
	}
	if other, err := s.HasOtherAdmin(cancelled, w.ID, bob.MemberID); !failed(err) || other {
		t.Errorf("HasOtherAdmin() = %v, %v; want context.Canceled, not another admin", other, err)
	}
	if ids, err := s.LockMemberWorkspaces(cancelled, alice); !failed(err) || ids != nil {
		t.Errorf("LockMemberWorkspaces() = %v, %v; want context.Canceled, no workspace", ids, err)
	}
	if sole, err := s.SoleAdmin(cancelled, []uuid.UUID{w.ID}, alice); !failed(err) || sole {
		t.Errorf("SoleAdmin() = %v, %v; want context.Canceled, not the only admin", sole, err)
	}
	invite(t, s, w.ID, "carol@corp.com", shared.RoleGuest, alice)
	if list, err := s.ListInvitations(cancelled, w.ID); !failed(err) || list != nil {
		t.Errorf("ListInvitations() = %v, %v; want context.Canceled, no list", list, err)
	}
}

// A write that fails answers its error, never nil, which a use case would
// take for done, and never a row. Each write runs on a cancelled context
// against a workspace alice administers.
func TestAFailedWriteIsAnError(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	failed := func(err error) bool { return errors.Is(err, context.Canceled) }

	if got, err := s.UpdateWorkspace(cancelled, w.ID, domain.WorkspacePatch{Name: ptr("Renamed")}, alice, now); !failed(err) ||
		!sameWorkspace(got, domain.Workspace{}) {
		t.Errorf("UpdateWorkspace() = %+v, %v; want context.Canceled", got, err)
	}
	if got, err := s.UpsertPreferences(cancelled, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: w.ID, UserID: alice, Now: now}); !failed(err) ||
		got != (domain.Preferences{}) {
		t.Errorf("UpsertPreferences() = %+v, %v; want context.Canceled", got, err)
	}
	for name, write := range map[string]func(context.Context, uuid.UUID, uuid.UUID, time.Time) error{
		"DeleteWorkspace": s.DeleteWorkspace, "DeleteWorkspaceInvitations": s.DeleteWorkspaceInvitations,
		"DeleteWorkspaceMembers": s.DeleteWorkspaceMembers, "DeleteWorkspacePreferences": s.DeleteWorkspacePreferences,
	} {
		if err := write(cancelled, w.ID, alice, now); !failed(err) {
			t.Errorf("%s() = %v; want context.Canceled", name, err)
		}
	}
	bob := joinAt(t, s, w.ID, newAccount(t, pool, "bob@corp.com"), shared.RoleMember, now)
	if got, err := s.UpdateMemberRole(cancelled, bob.ID, shared.RoleGuest, alice, now); !failed(err) || got != (domain.Membership{}) {
		t.Errorf("UpdateMemberRole() = %+v, %v; want context.Canceled", got, err)
	}
	if err := s.EndMember(cancelled, w.ID, bob.MemberID, alice, now); !failed(err) {
		t.Errorf("EndMember() = %v; want context.Canceled", err)
	}
	if err := s.ReactivateMember(cancelled, w.ID, bob.MemberID, now); !failed(err) {
		t.Errorf("ReactivateMember() = %v; want context.Canceled, not no membership", err)
	}
	if err := s.DeletePendingInvitations(cancelled, w.ID, "carol@corp.com", alice, now); !failed(err) {
		t.Errorf("DeletePendingInvitations() = %v; want context.Canceled", err)
	}
	if err := s.DeleteInvitationsTo(cancelled, "carol@corp.com", alice, now); !failed(err) {
		t.Errorf("DeleteInvitationsTo() = %v; want context.Canceled", err)
	}
	if err := s.DeleteInvitationsOfWorkspacesLeftEmpty(cancelled, []uuid.UUID{w.ID}, bob.MemberID, bob.MemberID, now); !failed(err) {
		t.Errorf("DeleteInvitationsOfWorkspacesLeftEmpty() = %v; want context.Canceled", err)
	}
	if err := s.EndWorkspaceMemberships(cancelled, []uuid.UUID{w.ID}, bob.MemberID, bob.MemberID, now); !failed(err) {
		t.Errorf("EndWorkspaceMemberships() = %v; want context.Canceled", err)
	}
	var dup *app.DuplicateInvitation
	if got, err := s.CreateInvitations(cancelled, []app.InvitationRow{
		{ID: uuid.NewV7(), WorkspaceID: w.ID, Email: "carol@corp.com", Role: shared.RoleGuest, CreatedBy: alice, Now: now},
	}); !failed(err) || errors.As(err, &dup) || got != nil {
		t.Errorf("CreateInvitations() = %+v, %v; want context.Canceled, not a duplicate", got, err)
	}
}
