package app_test

import (
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// listing is newRelate's world with more work items: shelved, bob's,
// archived, of web; mine, alice's, of ops; hidden, carol's, of lab, a
// project alice does not see; boxed, alice's, of attic, an archived
// project. web, the work item's own project, is archived too. ops shows
// alice, its guest, only her own work items (Grant.OwnOnly): mine, not
// carol's far. relations are the work item's relations as the store
// answers them, in the order they were made, which is not the others' id
// order: far's duplicate of it; its blocked_by mine; its relates_to
// shelved; near's blocked_by it; hidden's relates_to it; boxed's
// duplicate of it.
type listing struct {
	relating
	shelved, mine, hidden, boxed domain.Issue
	lab, attic                   uuid.UUID
	relations                    []domain.Relation
}

// newListing is listIssueRelations over newListing's world, web's work
// item 7, carol's, its work item.
func newListing() (*app.ListIssueRelations, *world, *fakeRelations, uuid.UUID, listing) {
	_, w, rel, id, r := newRelate()
	l := listing{relating: r, lab: uuid.NewV7(), attic: uuid.NewV7()}
	l.shelved = domain.Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, SequenceID: 9, Name: "Shelved", CreatedByID: &bob,
		ArchivedAt: &earlier}
	l.mine = domain.Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: ops, SequenceID: 4, Name: "Mine", CreatedByID: &alice}
	l.hidden = domain.Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: l.lab, SequenceID: 1, Name: "Hidden", CreatedByID: &carol}
	l.boxed = domain.Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: l.attic, SequenceID: 2, Name: "Boxed", CreatedByID: &alice}
	for _, o := range []domain.Issue{l.shelved, l.mine, l.hidden, l.boxed} {
		w.issues.others[o.ID] = o
	}
	w.project.locked.Archived = true
	w.project.across[l.attic] = app.LockedProject{WorkspaceID: acme, Archived: true, Identifier: "ATTIC"}
	made := func(n int, p, from, to uuid.UUID, t domain.RelationType) domain.Relation {
		return domain.Relation{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: p, IssueID: from, RelatedID: to, Type: t, CreatedByID: &bob,
			CreatedAt: earlier.Add(time.Duration(n) * time.Minute)}
	}
	l.relations = []domain.Relation{made(0, ops, r.far.ID, id, domain.Duplicate), made(1, web, id, l.mine.ID, domain.BlockedBy),
		made(2, web, id, l.shelved.ID, domain.RelatesTo), made(3, web, r.near.ID, id, domain.BlockedBy),
		made(4, l.lab, l.hidden.ID, id, domain.RelatesTo), made(5, l.attic, l.boxed.ID, id, domain.Duplicate)}
	rel.relations = l.relations
	// The decisions: the work item's, then near's project, far's and lab's,
	// as the others are read, in id order, each project once.
	w.auth.answers, w.auth.grants = []error{3: shared.ErrNotVisible}, []shared.Grant{2: ownOnly}
	w.issues.counts = map[uuid.UUID]app.SubIssueCount{r.near.ID: {All: 2, Own: 1}, l.mine.ID: {All: 4, Own: 3}}
	return app.NewListIssueRelations(w.locks, rel, w.project), w, rel, id, l
}

// listed are the calls of alice's list of the work item id's relations,
// outside any transaction and without a lock (M4 design 3.4, 3.8, 3.10):
// the work item read and issue.read decided on it, as getIssue does; its
// relations; their other work items, in the relations' order; which of
// the others' projects but web, the work item's own, are archived, in id
// order: ops, lab and attic, of which attic is; issue.read decided on the
// others' projects, each once, in the others' id order, shelved, archived,
// and boxed, of attic, left out: web, ops, lab; then the others she sees
// counted.
func listed(id uuid.UUID, l listing) []string {
	calls := []string{line("GetIssue", id), line("Authorize", alice, domain.ActionRead, acme, web), line("Relations", id),
		line("GetIssues", []uuid.UUID{l.far.ID, l.mine.ID, l.shelved.ID, l.near.ID, l.hidden.ID, l.boxed.ID}),
		line("Archived", anys(sorted([]uuid.UUID{ops, l.lab, l.attic}))...)}
	for _, p := range []uuid.UUID{web, ops, l.lab} {
		calls = append(calls, line("Authorize", alice, domain.ActionRead, acme, p))
	}
	calls = append(calls, counted(l.near.ID, l.mine.ID))
	for n := range calls {
		calls[n] += " outside tx"
	}
	return calls
}

// A list of relations (M4 design 3.4, 3.8, 3.10), with listed's calls,
// answers the relations whose other work item alice sees, in the order
// they were made, each as the work item reads it, each other as she sees
// it in its project: mine, which it is blocked by, counted as she sees it
// in ops, her own sub-items; then near, which it blocks, every one of its
// sub-items in web, which is archived but the work item's own. Shelved,
// archived, is not shown, nor boxed, of another archived project, nor
// far, which the guest rule hides from her in ops, nor hidden, of a
// project she does not see.
func TestListIssueRelationsListsWhatTheCallerSees(t *testing.T) {
	uc, w, _, id, l := newListing()
	got, err := uc.Execute(as(alice), id)
	mine, near := l.mine, l.near
	mine.SubIssuesCount, near.SubIssuesCount = 3, 2
	want := []app.IssueRelation{{ID: l.relations[1].ID, Type: domain.BlockedBy, Issue: mine, CreatedAt: l.relations[1].CreatedAt},
		{ID: l.relations[3].ID, Type: domain.Blocking, Issue: near, CreatedAt: l.relations[3].CreatedAt}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	if calls := listed(id, l); !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
}

// What refuses a list of relations is what refuses getIssue, before any
// relation is read: a work item not there, or whose project the caller
// does not see, or that the guest rule hides from him, issue.not_found; a
// role that may not read it, forbidden. A read, a decision or a count
// that fails ends it with its failure; a work item without relations has
// none.
func TestListIssueRelationsRefusals(t *testing.T) {
	for _, tt := range []struct {
		name  string
		set   func(w *world, rel *fakeRelations, id uuid.UUID) uuid.UUID // the work item asked for
		want  error
		calls int // of listed's
	}{
		{"no work item", func(*world, *fakeRelations, uuid.UUID) uuid.UUID { return uuid.NewV7() }, domain.ErrIssueNotFound, 1},
		{"a project not seen", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID {
			w.auth.answers = []error{shared.ErrNotVisible}
			return id
		}, domain.ErrIssueNotFound, 2},
		{"another's, to a guest who sees only his own", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID {
			w.auth.grants = []shared.Grant{ownOnly}
			return id
		}, domain.ErrIssueNotFound, 2},
		{"a role not allowed", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID {
			w.auth.answers = []error{shared.Forbidden()}
			return id
		}, shared.Forbidden(), 2},
		{"the relations' read fails", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID { w.log.fail = "Relations"; return id }, errDisk, 3},
		{"the others' read fails", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID { w.log.fail = "GetIssues"; return id }, errDisk, 4},
		{"the projects' read fails", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID { w.log.fail = "Archived"; return id }, errDisk, 5},
		{"a decision on another project fails", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID {
			w.log.fail = line("Authorize", alice, domain.ActionRead, acme, ops)
			return id
		}, errDisk, 7},
		{"the count fails", func(w *world, _ *fakeRelations, id uuid.UUID) uuid.UUID { w.log.fail = "SubIssueCounts"; return id }, errDisk, 9},
		{"no relations", func(_ *world, rel *fakeRelations, id uuid.UUID) uuid.UUID { rel.relations = nil; return id }, nil, 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, rel, id, l := newListing()
			asked := tt.set(w, rel, id)
			got, err := uc.Execute(as(alice), asked)
			if !reflect.DeepEqual(err, tt.want) || len(got) != 0 {
				t.Errorf("Execute() = %+v, %v; want none and %v", got, err, tt.want)
			}
			calls := listed(id, l)[:tt.calls]
			calls[0] = line("GetIssue", asked, "outside tx")
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
		})
	}
}
