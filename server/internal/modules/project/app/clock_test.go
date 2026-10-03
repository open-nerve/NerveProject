package app_test

import (
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// createProject only inserts, so it reads the clock once, before its
// transaction and its locks (P3's ruling (c)): no row it reads under them
// has a time its rows must follow. Every row it writes has that one time.
func TestCreatingAProjectReadsTheClockBeforeItsTransaction(t *testing.T) {
	uc, f := newCreate()
	if _, err := uc.Execute(as(alice), "acme", domain.NewProject{Name: "Web", Identifier: "WEB", LeadID: &bob}); err != nil {
		t.Fatal(err)
	}
	if len(f.log.calls) < 2 || f.log.calls[0] != "Now" || f.log.calls[1] != "Begin" || slices.Contains(f.log.calls[1:], "Now") {
		t.Errorf("calls %q; want the clock read once, first, before the transaction", f.log.calls)
	}
	for _, r := range f.projects.members {
		if r.Now != clockNow {
			t.Errorf("a membership at %v, want %v", r.Now, clockNow)
		}
	}
	for _, r := range f.projects.prefs {
		if r.Now != clockNow {
			t.Errorf("display settings at %v, want %v", r.Now, clockNow)
		}
	}
	for _, r := range f.projects.states {
		if r.Now != clockNow {
			t.Errorf("a state at %v, want %v", r.Now, clockNow)
		}
	}
}

// Each write that changes an existing row reads the clock once, in its
// transaction, after its locks (its workspace's FOR SHARE first, then its
// project's), its decision and its checks, just before it writes (P2 spec
// 2.6, M3 design 3.3): a write that queued behind another on either lock
// never stamps an earlier time than the one it waited for. The clock logs
// its read among the fakes' calls.
func TestEachWriteReadsTheClockUnderItsLock(t *testing.T) {
	tests := []struct {
		name string
		run  func() (calls []string, err error)
		want []string
	}{
		{"updateProject", func() ([]string, error) {
			uc, f := newUpdate()
			_, err := uc.Execute(as(bob), webID, domain.ProjectPatch{SetLead: true, LeadID: &alice})
			return f.log.calls, err
		}, updated(domain.ProjectPatch{SetLead: true, LeadID: &alice}, alice)},
		{"archiveProject", func() ([]string, error) {
			uc, f := newArchive(true)
			_, err := uc.Execute(as(bob), webID)
			return f.log.calls, err
		}, archived(webID, true)},
		{"unarchiveProject", func() ([]string, error) {
			uc, f := newArchive(false)
			_, err := uc.Execute(as(bob), opsID)
			return f.log.calls, err
		}, archived(opsID, false)},
		{"deleteProject", func() ([]string, error) {
			uc, f := newDelete()
			err := uc.Execute(as(bob), webID)
			return f.log.calls, err
		}, deleted(webID)},
		{"updateProjectPreferences", func() ([]string, error) {
			uc, f := newUpdatePreferences()
			_, err := uc.Execute(as(bob), webID, domain.PreferencesPatch{SortOrder: ptr(1.0)})
			return f.log.calls, err
		}, changed(bob, webID, domain.PreferencesPatch{SortOrder: ptr(1.0)})},
		{"addProjectMembers", func() ([]string, error) {
			uc, f := newAdd()
			_, err := uc.Execute(as(bob), webID, []domain.NewMember{{MemberID: ivy, Role: shared.RoleGuest}})
			return f.log.calls, err
		}, slices.Concat(beforeTargets(bob, webID, []domain.NewMember{{MemberID: ivy, Role: shared.RoleGuest}}), []string{"Now",
			"LowestSortOrder " + acme.ID.String() + " " + ivy.String()}, grown(ivy, nil, shared.RoleGuest, 65535, bob),
			[]string{"ListMembers " + webID.String()})},
		{"joinProject", func() ([]string, error) {
			uc, f := newJoin()
			_, err := uc.Execute(as(hank), webID)
			return f.log.calls, err
		}, slices.Concat(beforeJoin(hank, webID), []string{"Now"}, grown(hank, nil, shared.RoleMember, 65535, hank),
			[]string{"GetProject " + webID.String() + " for " + hank.String()})},
		{"updateProjectMember", func() ([]string, error) {
			uc, f := newUpdateMember()
			_, err := uc.Execute(as(bob), aliceInWeb, shared.RoleGuest)
			return f.log.calls, err
		}, roleChanged(newMemberWrites(), bob, alice, shared.RoleGuest)},
		{"removeProjectMember", func() ([]string, error) {
			uc, f := newRemoveMember()
			err := uc.Execute(as(bob), aliceInWeb)
			return f.log.calls, err
		}, removed(bob, alice)},
		{"leaveProject", func() ([]string, error) {
			uc, f := newLeave()
			err := uc.Execute(as(bob), webID)
			return f.log.calls, err
		}, left(bob, webID, true)},
	}
	for _, tt := range tests {
		if calls, err := tt.run(); err != nil || !slices.Equal(calls, tt.want) {
			t.Errorf("%s: calls = %q, %v; want %q", tt.name, calls, err, tt.want)
		}
	}
}
