package app_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// counted is the call that counts the sub-items of the work items ids for
// alice, whose answer shows them (Locks.seen), as her write's transaction
// logs it; a read outside any logs it " outside tx" after.
func counted(ids ...uuid.UUID) string {
	return line("SubIssueCounts", alice, ids)
}

// counts are the sub-items alice's work item id has, three, one of them
// hers, and those of another work item, which no answer of it shows.
func counts(id uuid.UUID) map[uuid.UUID]app.SubIssueCount {
	return map[uuid.UUID]app.SubIssueCount{id: {All: 3, Own: 1}, uuid.NewV7(): {All: 5, Own: 5}}
}

// A read of a work item, getIssue's and getIssueByIdentifier's, answers the
// number of its sub-items the caller sees (M4 design 3.4, 3.9), counted
// after the decision, outside any transaction, by its grant: every one,
// or only his own when the guest rule hides others' from him
// (Grant.OwnOnly), as from alice, a guest of web who reads her own work
// item. Every one when web shows its guests every work item.
func TestAReadCountsTheSubItemsTheCallerSees(t *testing.T) {
	for _, tt := range []struct {
		name  string
		grant shared.Grant
		want  int
	}{{"an admin", shared.Grant{ProjectRole: shared.RoleAdmin}, 3}, {"a guest who sees his own", ownOnly, 1},
		{"a guest who sees every work item", everyItem, 3}} {
		get, w, id := newGet()
		w.issues.issue.By, w.issues.counts, w.auth.grants = alice, counts(id), []shared.Grant{tt.grant}
		got, err := get.Execute(as(alice), id)
		if err != nil || got.SubIssuesCount != tt.want || w.log.calls[len(w.log.calls)-1] != counted(id)+" outside tx" {
			t.Errorf("%s: getIssue = %+v, %v after %q; want %d sub-items, counted last", tt.name, got, err, w.log.calls, tt.want)
		}
		byIdentifier, b, bid := newGetByIdentifier()
		b.issues.issue.By, b.issues.counts, b.auth.grants = alice, counts(bid), []shared.Grant{{}, tt.grant}
		got, err = byIdentifier.Execute(as(alice), "acme", "WEB-7")
		if err != nil || got.SubIssuesCount != tt.want || b.log.calls[len(b.log.calls)-1] != counted(bid)+" outside tx" {
			t.Errorf("%s: getIssueByIdentifier = %+v, %v after %q; want %d sub-items, counted last", tt.name, got, err, b.log.calls, tt.want)
		}
	}
}

// An answer names the work item's parent only when the caller sees it (M4
// design 3.4, 3.9): alice reads her own work item 7, under parent. When the
// guest rule hides others' work items from her (Grant.OwnOnly), the parent
// is read after the count, as the count is, outside any transaction, in
// one read, and one carol made, or one the read does not find, is answered
// as none; her own is named. When she sees every work item it is named,
// and nothing more is read. Her change of the work item as a workspace
// admin who is only a guest of web reads the parent in the transaction,
// after the count. A failing read of the parent is the answer's error.
func TestAnAnswerNamesTheParentTheCallerSees(t *testing.T) {
	parent := uuid.NewV7()
	read := line("GetIssues", []uuid.UUID{parent})
	for _, tt := range []struct {
		name  string
		grant shared.Grant
		by    *uuid.UUID // the parent's creator; nil when the read does not find it
		want  *uuid.UUID
		reads bool
	}{
		{"carol's, to a guest who sees her own", ownOnly, &carol, nil, true},
		{"her own, to a guest who sees her own", ownOnly, &alice, &parent, true},
		{"not found, to a guest who sees her own", ownOnly, nil, nil, true},
		{"carol's, to a guest who sees every work item", everyItem, &carol, &parent, false},
	} {
		get, w, id := newGet()
		w.issues.issue.By, w.issues.issue.ParentID, w.auth.grants = alice, &parent, []shared.Grant{tt.grant}
		if tt.by != nil {
			w.issues.others = map[uuid.UUID]domain.Issue{parent: {ID: parent, WorkspaceID: acme, ProjectID: web, CreatedByID: tt.by}}
		}
		got, err := get.Execute(as(alice), id)
		want := []string{counted(id) + " outside tx"}
		if tt.reads {
			want = append(want, read+" outside tx")
		}
		if calls := w.log.calls[len(w.log.calls)-len(want):]; err != nil || !reflect.DeepEqual(got.ParentID, tt.want) ||
			!slices.Equal(calls, want) {
			t.Errorf("%s: Execute() = %v, %v, its last calls %q; want the parent %v, the last calls %q", tt.name, got.ParentID, err, calls,
				tt.want, want)
		}
	}

	_, w, id := newUpdate()
	w.issues.issue.By, w.issues.issue.ParentID = alice, &parent
	w.issues.others = map[uuid.UUID]domain.Issue{parent: {ID: parent, WorkspaceID: acme, ProjectID: web, CreatedByID: &carol}}
	adminGuest := shared.Grant{ProjectRole: shared.RoleGuest, ProjectAdmin: true, OwnOnly: true}
	w.auth.grants = []shared.Grant{adminGuest, adminGuest}
	got, err := updateOver(w).Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")})
	if calls := w.log.calls[len(w.log.calls)-2:]; err != nil || got.ParentID != nil || !slices.Equal(calls, []string{counted(id), read}) {
		t.Errorf("a change: Execute() = %v, %v, its last calls %q; want no parent, the last calls %q", got.ParentID, err, calls,
			[]string{counted(id), read})
	}

	get, w, id := newGet()
	w.issues.issue.By, w.issues.issue.ParentID, w.auth.grants, w.log.fail = alice, &parent, []shared.Grant{ownOnly}, "GetIssues"
	if got, err := get.Execute(as(alice), id); !errors.Is(err, errDisk) || !reflect.DeepEqual(got, domain.Issue{}) {
		t.Errorf("Execute() with the read of the parent failing = %+v, %v; want %v", got, err, errDisk)
	}
}

// A write's answer of the work item it names, updateIssue's and
// archiveIssue's, shows the number of its sub-items the caller sees (M4
// design 3.4, 3.9), counted by the grant of the decision under the lock,
// in the transaction, last: after the work item read back, or after the
// decision when the write writes nothing: alice's own work item 7 changed,
// changed in nothing, archived, archived again, unarchived, unarchived
// again. Every one as an admin of web, or only her own as the guest rule
// has it for a workspace admin who is only a guest of web.
func TestAWritesAnswerCountsTheSubItemsTheCallerSees(t *testing.T) {
	for _, tt := range []struct {
		name    string
		prepare func(w *world)
		write   func(w *world, id uuid.UUID) (domain.Issue, error)
		writes  bool
	}{
		{"changed", func(*world) {}, func(w *world, id uuid.UUID) (domain.Issue, error) {
			return updateOver(w).Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")})
		}, true},
		{"changed in nothing", func(*world) {}, func(w *world, id uuid.UUID) (domain.Issue, error) {
			return updateOver(w).Execute(as(alice), id, domain.IssuePatch{Name: name("Fix")})
		}, false},
		{"archived", func(w *world) { w.issues.issue.StateID = done.ID }, func(w *world, id uuid.UUID) (domain.Issue, error) {
			archive, _ := archiveOver(w)
			return archive.Execute(as(alice), id)
		}, true},
		{"archived again", func(w *world) { w.issues.archivedAt = &earlier }, func(w *world, id uuid.UUID) (domain.Issue, error) {
			archive, _ := archiveOver(w)
			return archive.Execute(as(alice), id)
		}, false},
		{"unarchived", func(w *world) { w.issues.archivedAt = &earlier }, func(w *world, id uuid.UUID) (domain.Issue, error) {
			_, unarchive := archiveOver(w)
			return unarchive.Execute(as(alice), id)
		}, true},
		{"unarchived again", func(*world) {}, func(w *world, id uuid.UUID) (domain.Issue, error) {
			_, unarchive := archiveOver(w)
			return unarchive.Execute(as(alice), id)
		}, false},
	} {
		for _, view := range []struct {
			name  string
			grant shared.Grant
			want  int
		}{{"an admin", shared.Grant{ProjectRole: shared.RoleAdmin}, 3},
			{"an admin who is a guest", shared.Grant{ProjectRole: shared.RoleGuest, ProjectAdmin: true, OwnOnly: true}, 1}} {
			t.Run(tt.name+", "+view.name, func(t *testing.T) {
				_, w, id := newUpdate()
				w.issues.issue.By, w.issues.counts = alice, counts(id)
				w.auth.grants = []shared.Grant{view.grant, view.grant}
				tt.prepare(w)
				got, err := tt.write(w, id)
				before := line("GetIssue", id) // the read back
				if !tt.writes {
					before = line("Authorize", alice) // the decision under the lock
				}
				n := len(w.log.calls)
				if err != nil || got.SubIssuesCount != view.want || n < 2 || w.log.calls[n-1] != counted(id) ||
					!strings.HasPrefix(w.log.calls[n-2], before) {
					t.Errorf("Execute() = %+v, %v after %q; want %d sub-items, counted last, after %s", got, err, w.log.calls, view.want,
						before)
				}
				if !tt.writes && slices.Contains(w.log.calls, "Now") {
					t.Errorf("calls %q; want nothing written", w.log.calls)
				}
			})
		}
	}
}
