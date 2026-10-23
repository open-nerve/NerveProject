package app_test

import (
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The guest rule in Locks' decisions (M4 design 3.4, 6.7): a decision on a
// work item names its creator, and a caller who sees only his own work
// items (Grant.OwnOnly) finds none another created, before his role is
// looked at: in the precheck, and again under the lock. Each answer is
// compared whole: another's work item is the very issue.not_found of one
// that is not there, detail and all (8.4).

var (
	// ownOnly is the grant of web's guest when web does not show its
	// guests every work item: he sees only those he created.
	ownOnly = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleGuest, OwnOnly: true}
	// everyItem is the same guest's when web shows its guests every work
	// item.
	everyItem = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleGuest}
	// adminGuest is the grant of a workspace admin who is only a guest of
	// web, which does not show its guests every work item: as an admin he
	// passes each of its rules (Grant.ProjectAdmin), and as its guest he
	// sees only his own work items (Grant.OwnOnly).
	adminGuest = shared.Grant{ProjectRole: shared.RoleGuest, ProjectAdmin: true, OwnOnly: true}
)

// A guest who sees only his own work items reads his own, and not
// another's, carol's Fix or bob's, which is issue.not_found to alice as
// such a guest, the work item read and the one decision taken, its
// sub-items not counted; carol's id is after hers and bob's before it, so
// the rule tells her own by its creator, not by the ids' order. When web
// shows its guests every work item, she reads carol's. A refusal of her
// role on another's work item is issue.not_found too, the guest rule
// first; on her own, forbidden. A work item whose creator is not known, as
// one whose creator's account is purged (P9), is no guest's own:
// issue.not_found.
func TestTheGuestRuleHidesAnothersWorkItemFromAReader(t *testing.T) {
	for _, tt := range []struct {
		name    string
		creator uuid.UUID
		grant   shared.Grant
		refused error
		want    error
	}{
		{"another's", carol, ownOnly, nil, domain.ErrIssueNotFound},
		{"another's, bob's, whose id is before hers", bob, ownOnly, nil, domain.ErrIssueNotFound},
		{"her own", alice, ownOnly, nil, nil},
		{"another's, web showing its guests every work item", carol, everyItem, nil, nil},
		{"another's, her role refused", carol, ownOnly, shared.Forbidden(), domain.ErrIssueNotFound},
		{"her own, her role refused", alice, ownOnly, shared.Forbidden(), shared.Forbidden()},
		{"no known creator's", uuid.UUID{}, ownOnly, nil, domain.ErrIssueNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newGet()
			w.issues.issue.By = tt.creator
			w.auth.grants, w.auth.answers = []shared.Grant{tt.grant}, []error{tt.refused}
			got, err := uc.Execute(as(alice), id)
			if !reflect.DeepEqual(err, tt.want) || (err == nil) != (got.ID == id) {
				t.Errorf("Execute() = %+v, %v; want %v", got, err, tt.want)
			}
			calls := []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx")}
			if tt.want == nil {
				calls = append(calls, counted(id)+" outside tx")
			}
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
		})
	}
}

// A write on another's work item by a guest who sees only his own is
// issue.not_found in the precheck, his role refused or not, before the
// gate and any lock; on his own, his refused role is forbidden there. A
// write the precheck let through, whose caller became such a guest while
// it waited, is issue.not_found under the lock, the decision there final,
// and writes nothing; forbidden on his own.
func TestTheGuestRuleHidesAnothersWorkItemFromAWriter(t *testing.T) {
	member := shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleMember}
	for _, tt := range []struct {
		name    string
		creator uuid.UUID
		grants  []shared.Grant
		answers []error
		want    error
		calls   int // of locked(id)
	}{
		{"another's, in the precheck", carol, []shared.Grant{ownOnly}, []error{shared.Forbidden()}, domain.ErrIssueNotFound, 2},
		{"another's, in the precheck, his role allowed", carol, []shared.Grant{ownOnly}, nil, domain.ErrIssueNotFound, 2},
		{"his own, in the precheck", alice, []shared.Grant{ownOnly}, []error{shared.Forbidden()}, shared.Forbidden(), 2},
		{"another's, under the lock", carol, []shared.Grant{member, ownOnly}, []error{nil, shared.Forbidden()}, domain.ErrIssueNotFound, 6},
		{"his own, under the lock", alice, []shared.Grant{member, ownOnly}, []error{nil, shared.Forbidden()}, shared.Forbidden(), 6},
		{"another's, under the lock, his role allowed", carol, []shared.Grant{member, ownOnly}, nil, domain.ErrIssueNotFound, 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			_, update := gated(w, nil)
			w.issues.issue.By = tt.creator
			w.auth.grants, w.auth.answers = tt.grants, tt.answers
			if _, err := update.Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")}); !reflect.DeepEqual(err, tt.want) {
				t.Errorf("Execute() = %v, want %v", err, tt.want)
			}
			want := locked(id)[:tt.calls]
			if tt.calls > 2 {
				want = slices.Concat(locked(id)[:2], []string{line("Enter", web, "outside tx")}, locked(id)[2:], []string{line("Leave", web)})
			}
			if !slices.Equal(w.log.calls, want) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, want)
			}
		})
	}
}

// Each decision on a work item names its creator, the precheck's and the
// one under the lock, so that a rule may allow him (AllowCreator); a
// decision on a project names no one, and so does a decision on a work
// item whose creator is not known: AllowCreator takes no caller for him.
func TestADecisionOnAWorkItemNamesItsCreator(t *testing.T) {
	get, w, id := newGet()
	if _, err := get.Execute(as(alice), id); err != nil {
		t.Fatal(err)
	}
	update, uw, uid := newUpdate()
	if _, err := update.Execute(as(alice), uid, domain.IssuePatch{Name: name("Mend")}); err != nil {
		t.Fatal(err)
	}
	create, cw := newCreate()
	if _, err := create.Execute(as(alice), web, domain.NewIssue{Name: "Fix"}); err != nil {
		t.Fatal(err)
	}
	deletion, dw, did := newDelete()
	if err := deletion.Execute(as(alice), did); err != nil {
		t.Fatal(err)
	}
	unknown, nw, nid := newDelete()
	nw.issues.issue.By = uuid.UUID{}
	if err := unknown.Execute(as(alice), nid); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name    string
		targets []shared.Target
		want    []uuid.UUID
	}{
		{"a read", w.auth.targets, []uuid.UUID{carol}},
		{"an update", uw.auth.targets, []uuid.UUID{carol, carol}},
		{"a creation", cw.auth.targets, []uuid.UUID{{}, {}}},
		{"a deletion", dw.auth.targets, []uuid.UUID{carol, carol}},
		{"a deletion of a work item of no known creator", nw.auth.targets, []uuid.UUID{{}, {}}},
	} {
		var got []uuid.UUID
		for _, target := range tt.targets {
			got = append(got, target.CreatorID)
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: the decisions' creators %v, want %v", tt.name, got, tt.want)
		}
	}
}
