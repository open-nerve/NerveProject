package app_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// intakeLister is the intake list's store as a fake: each read logged
// with its project, whether it sees only the caller's own work items and
// whether it has a cursor, and kept; it answers its pages in turn, none
// past them.
type intakeLister struct {
	log   *callLog
	pages [][]app.IntakeRecord
	reads []app.IntakeListRead
}

func (f *intakeLister) ListIntake(ctx context.Context, in app.IntakeListRead) ([]app.IntakeRecord, error) {
	f.reads = append(f.reads, in)
	if err := f.log.add(ctx, "ListIntake", in.ProjectID, in.OwnOnly, in.Query.Cursor != nil); err != nil {
		return nil, err
	}
	if len(f.reads) > len(f.pages) {
		return nil, nil
	}
	return f.pages[len(f.reads)-1], nil
}

// storeRefs is the store IssueRefs reads through, over the work items'
// fake: each read logged, it answers the work items among ids the fake
// has (fakeIssues.read), in id order, as the store reads the undeleted.
type storeRefs struct{ issues *fakeIssues }

func (f storeRefs) IssueRefRows(ctx context.Context, ids []uuid.UUID) ([]app.IssueRefRow, error) {
	if err := f.issues.log.add(ctx, "IssueRefRows", ids); err != nil {
		return nil, err
	}
	var out []app.IssueRefRow
	for _, id := range slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare) {
		if i, found := f.issues.read(id); found {
			out = append(out, app.IssueRefRow{IssueRef: app.IssueRef{ID: i.ID, ProjectID: i.ProjectID, SequenceID: i.SequenceID, Name: i.Name},
				CreatedByID: i.CreatedByID})
		}
	}
	return out, nil
}

// loggedAttachments counts each work item's attachments as counts has
// them, none for one it has not, each count logged.
type loggedAttachments struct {
	log    *callLog
	counts map[uuid.UUID]int64
}

func (f loggedAttachments) CountIssueAttachments(ctx context.Context, ids []uuid.UUID) ([]app.AttachmentCount, error) {
	if err := f.log.add(ctx, "CountIssueAttachments", ids); err != nil {
		return nil, err
	}
	rows := make([]app.AttachmentCount, len(ids))
	for n, id := range ids {
		rows[n] = app.AttachmentCount{IssueID: id, Count: f.counts[id]}
	}
	return rows, nil
}

// tickingClock answers clockNow at its first read and an hour later at
// each read after it, each read logged as "Now".
type tickingClock struct {
	log   *callLog
	reads int
}

func (c *tickingClock) Now() time.Time {
	c.log.calls = append(c.log.calls, "Now")
	c.reads++
	return clockNow.Add(time.Duration(c.reads-1) * time.Hour)
}

// loggedNow is the list's clock at clockNow, each read logged on log.
func loggedNow(log *callLog) app.Clock { return clockAt{at: clockNow, log: log} }

// intakeListWorld is web's intake list over newGet's world: Locks read the
// work items through the fake store and count their attachments through
// attachments; the list's store is rows, and IssueRefs read the duplicates'
// targets from the work items' fake.
type intakeListWorld struct {
	*world
	uc          *app.ListIntakeIssues
	rows        *intakeLister
	attachments loggedAttachments
}

func newIntakeList(clock func(*callLog) app.Clock) *intakeListWorld {
	_, w, _ := newGet()
	w.issues.others = map[uuid.UUID]domain.Issue{}
	attachments := loggedAttachments{log: w.log, counts: map[uuid.UUID]int64{}}
	w.locks = app.NewLocks(w.project, w.issues, w.auth, fakeGate{log: w.log}, fakeTx{log: w.log}, attachments, fixedAt(clockNow))
	rows := &intakeLister{log: w.log}
	uc := app.NewListIntakeIssues(app.ListIntakeIssuesDeps{ReadTx: fakeTx{log: w.log}, Locks: w.locks, Rows: rows,
		Refs: app.NewIssueRefs(storeRefs{w.issues}), Clock: clock(w.log)})
	return &intakeListWorld{world: w, uc: uc, rows: rows, attachments: attachments}
}

// intakeRecord is a row of web's intake list as the store reads it: the
// work item id, made by by, under parent, nil for none, four sub-items
// counted, with its intake row as stored, facts, among seven rows, at its
// position in -created_at.
func intakeRecord(id, by uuid.UUID, parent *uuid.UUID, facts *domain.IntakeFacts) app.IntakeRecord {
	return app.IntakeRecord{Total: 7, Position: domain.IntakeCursor{Order: "-created_at", At: clockNow, ID: id},
		Issue: domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, CreatedByID: &by, ParentID: parent, SubIssuesCount: 4,
			AssigneeIDs: []uuid.UUID{bob}, LabelIDs: []uuid.UUID{}, ModuleIDs: []uuid.UUID{}, Intake: facts}}
}

// listedRow is the record r as the list answers it: its parent parent,
// count attachments, its intake row facts.
func listedRow(r app.IntakeRecord, parent *uuid.UUID, count int64, facts *domain.IntakeFacts) domain.Issue {
	i := r.Issue
	i.ParentID, i.AttachmentCount, i.Intake = parent, count, facts
	return i
}

// A page of web's intake list (M7 P2b spec 2.3, R10), in one read
// snapshot, its moment read once: the project and the decision on
// intake.read; the clock; the page and the row after it, read at that
// moment with the caller's grant; then the page's rows trimmed as the
// project list's are, the row after them not read again: the parents the
// guest rule hides from an own-only guest none (one read of them; none
// for a member, who sees every parent), the attachments counted; and each
// row's intake row as its detail answers it: its effective status at the
// moment, a snooze not yet run out with its time, and the work item it
// duplicates, read once for the page, only when the caller sees it. The
// cursor is the last row's position; the total, the store's.
func TestListIntakeIssuesReadsAPageAsTheProjectListTrimsIt(t *testing.T) {
	parentHidden, parentOwn, targetHidden, targetOwn, after, afterParent, afterTarget := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(),
		uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	a, b, c, d := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	pending, snoozed := intakeOf(domain.IntakePending, nil, nil), intakeOf(domain.IntakeSnoozed, &soon, nil)
	for _, tt := range []struct {
		name    string
		grant   shared.Grant
		parents []uuid.UUID // read for the guest rule
		want    func(rows []app.IntakeRecord) []domain.Issue
	}{
		{"an own-only guest", ownOnly, []uuid.UUID{parentHidden, parentOwn}, func(rows []app.IntakeRecord) []domain.Issue {
			return []domain.Issue{listedRow(rows[0], nil, 2, pending), listedRow(rows[1], &parentOwn, 0, snoozed),
				listedRow(rows[2], nil, 0, intakeOf(domain.IntakeDuplicate, nil, nil)), listedRow(rows[3], nil, 0, intakeOf(domain.IntakeDuplicate, nil, &targetOwn))}
		}},
		{"a member", shared.Grant{ProjectRole: shared.RoleMember}, nil, func(rows []app.IntakeRecord) []domain.Issue {
			return []domain.Issue{listedRow(rows[0], &parentHidden, 2, pending), listedRow(rows[1], &parentOwn, 0, snoozed),
				listedRow(rows[2], nil, 0, intakeOf(domain.IntakeDuplicate, nil, &targetHidden)), listedRow(rows[3], nil, 0, intakeOf(domain.IntakeDuplicate, nil, &targetOwn))}
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newIntakeList(loggedNow)
			w.auth.grants = []shared.Grant{tt.grant}
			for id, by := range map[uuid.UUID]uuid.UUID{parentHidden: carol, parentOwn: alice, targetHidden: carol, targetOwn: alice, afterParent: carol, afterTarget: carol} {
				w.issues.others[id] = domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, CreatedByID: &by}
			}
			w.attachments.counts[a] = 2
			rows := []app.IntakeRecord{intakeRecord(a, alice, &parentHidden, pending), intakeRecord(b, alice, &parentOwn, snoozed),
				intakeRecord(c, alice, nil, intakeOf(domain.IntakeDuplicate, nil, &targetHidden)), intakeRecord(d, alice, nil, intakeOf(domain.IntakeDuplicate, nil, &targetOwn)),
				intakeRecord(after, alice, &afterParent, intakeOf(domain.IntakeDuplicate, nil, &afterTarget))}
			w.rows.pages = [][]app.IntakeRecord{rows}
			raw := map[string][]string{"limit": {"4"}, "status": {"pending,snoozed,duplicate"}}
			page, err := w.uc.Execute(as(alice), web, raw)
			if err != nil {
				t.Fatal(err)
			}
			if want := tt.want(rows); page.Total != 7 || !reflect.DeepEqual(page.Data, want) {
				t.Errorf("Execute() = total %d, rows\n%+v\nwant 7,\n%+v", page.Total, page.Data, want)
			}
			if position, err := domain.DecodeIntakeCursor(page.NextCursor, "-created_at"); err != nil || position.ID != d {
				t.Errorf("the next cursor %q (%+v, %v), want the fourth row's position", page.NextCursor, position, err)
			}
			calls := []string{"Begin", line("Find", web), line("Authorize", alice, domain.ActionReadIntake, acme, web), "Now", line("ListIntake", web, tt.grant.OwnOnly, false)}
			if tt.parents != nil {
				calls = append(calls, line("GetIssues", ascendingIDs(tt.parents)))
			}
			calls = append(calls, line("CountIssueAttachments", ascendingIDs([]uuid.UUID{a, b, c, d})), line("IssueRefRows", ascendingIDs([]uuid.UUID{targetHidden, targetOwn})))
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			q, _ := domain.ParseIntakeQuery(raw, alice)
			if want := []app.IntakeListRead{{ProjectID: web, ActorID: alice, OwnOnly: tt.grant.OwnOnly, Query: q, Now: clockNow}}; !reflect.DeepEqual(w.rows.reads, want) {
				t.Errorf("the reads %+v, want %+v", w.rows.reads, want)
			}
		})
	}
}

// The page and its answer are at the one moment the clock gives once: a
// snooze the store selected as snoozed at it is answered snoozed, with its
// time, though the clock, were it read again, would be past it.
func TestListIntakeIssuesAnswersAtTheMomentItReadsAt(t *testing.T) {
	w := newIntakeList(func(log *callLog) app.Clock { return &tickingClock{log: log} })
	id, till := uuid.NewV7(), clockNow.Add(30*time.Minute)
	w.rows.pages = [][]app.IntakeRecord{{intakeRecord(id, alice, nil, intakeOf(domain.IntakeSnoozed, &till, nil))}}
	page, err := w.uc.Execute(as(alice), web, nil)
	if err != nil || len(page.Data) != 1 || !reflect.DeepEqual(page.Data[0].Intake, intakeOf(domain.IntakeSnoozed, &till, nil)) {
		t.Fatalf("Execute() = %+v, %v; want the row snoozed till %s", page, err, till)
	}
	if len(w.rows.reads) != 1 || !w.rows.reads[0].Now.Equal(clockNow) {
		t.Errorf("the reads %+v, want one at %s", w.rows.reads, clockNow)
	}
	if n := slices.Index(w.log.calls, "Now"); n < 0 || slices.Contains(w.log.calls[n+1:], "Now") {
		t.Errorf("calls %q, want the clock read once", w.log.calls)
	}
}

// A page past the last row has none, and the total read again without
// the cursor, one row asked for, as the project list does (Listing.Read);
// a first page without a row reads nothing more, its total none.
func TestListIntakeIssuesPastTheLastRow(t *testing.T) {
	cursor, _ := domain.EncodeIntakeCursor(domain.IntakeCursor{Order: "-created_at", At: clockNow, ID: bob})
	for _, tt := range []struct {
		name  string
		raw   map[string][]string
		total int64
		reads int
	}{
		{"a cursor past the last row", map[string][]string{"cursor": {cursor}, "limit": {"3"}}, 7, 2},
		{"a first page", nil, 0, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newIntakeList(loggedNow)
			w.rows.pages = [][]app.IntakeRecord{nil, {intakeRecord(bob, alice, nil, intakeOf(domain.IntakePending, nil, nil))}}
			page, err := w.uc.Execute(as(alice), web, tt.raw)
			if err != nil || page.Total != tt.total || page.Data == nil || len(page.Data) != 0 || page.NextCursor != "" {
				t.Fatalf("Execute() = %+v, %v; want no row of %d", page, err, tt.total)
			}
			if len(w.rows.reads) != tt.reads {
				t.Fatalf("the reads %+v, want %d", w.rows.reads, tt.reads)
			}
			if tt.reads == 2 {
				first, again := w.rows.reads[0], w.rows.reads[1]
				first.Query.Cursor, first.Query.Limit = nil, 1
				if w.rows.reads[0].Query.Cursor == nil || !reflect.DeepEqual(again, first) {
					t.Errorf("the reads %+v, want the second without the cursor, one row", w.rows.reads)
				}
			}
		})
	}
}

// A query the list does not take is refused before any read (400, 422);
// then a project not there or not seen is project.not_found, a role the
// rule does not allow forbidden; a request without a caller is
// unauthenticated. A failing port ends the list with its failure, as
// itself. None answers a page.
func TestListIntakeIssuesRefusals(t *testing.T) {
	decided := []string{"Begin", line("Find", web), line("Authorize", alice, domain.ActionReadIntake, acme, web)}
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		raw   map[string][]string
		set   func(w *intakeListWorld)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), nil, func(*intakeListWorld) {}, shared.Unauthenticated(), nil},
		{"a status not listed", as(alice), map[string][]string{"status": {"open"}}, func(w *intakeListWorld) { w.project.found = false },
			shared.Invalid(shared.FieldError{Field: "status", Code: "not_allowed", Message: "is not allowed for this list"}), nil},
		{"no project", as(alice), nil, func(w *intakeListWorld) { w.project.found = false }, domain.ErrProjectNotFound, decided[:2]},
		{"a project not seen", as(alice), nil, func(w *intakeListWorld) { w.auth.answers = []error{shared.ErrNotVisible} }, domain.ErrProjectNotFound, decided},
		{"a role not allowed", as(alice), nil, func(w *intakeListWorld) { w.auth.answers = []error{shared.Forbidden()} }, shared.Forbidden(), decided},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newIntakeList(loggedNow)
			tt.set(w)
			page, err := w.uc.Execute(tt.ctx, web, tt.raw)
			if !reflect.DeepEqual(err, tt.want) || !reflect.DeepEqual(page, app.IntakeIssuePage{}) || !slices.Equal(w.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", page, err, w.log.calls, tt.want, tt.calls)
			}
		})
	}
	cursor, _ := domain.EncodeIntakeCursor(domain.IntakeCursor{Order: "-created_at", At: clockNow, ID: bob})
	for _, tt := range []struct {
		fail string // a port's name, or its call (wantStoppedOn)
		raw  map[string][]string
	}{
		{"Find", nil}, {"Authorize", nil}, {"ListIntake", nil}, {"GetIssues", nil}, {"CountIssueAttachments", nil}, {"IssueRefRows", nil},
		{line("ListIntake", web, true, false), map[string][]string{"cursor": {cursor}}},
	} {
		t.Run(tt.fail, func(t *testing.T) {
			w := newIntakeList(loggedNow)
			w.auth.grants = []shared.Grant{ownOnly}
			parent, target := uuid.NewV7(), uuid.NewV7()
			w.rows.pages = [][]app.IntakeRecord{{intakeRecord(bob, alice, &parent, intakeOf(domain.IntakeDuplicate, nil, &target))}}
			if tt.raw != nil {
				w.rows.pages = [][]app.IntakeRecord{nil}
			}
			w.log.fail = tt.fail
			page, err := w.uc.Execute(as(alice), web, tt.raw)
			if !reflect.DeepEqual(page, app.IntakeIssuePage{}) {
				t.Errorf("Execute() = %+v, want no page", page)
			}
			if tt.raw != nil {
				wantStoppedOn(t, w.world, err, tt.fail)
				return
			}
			wantStoppedAt(t, w.world, err, tt.fail)
		})
	}
}
