package postgresadapter_test

import (
	"cmp"
	"context"
	"net/url"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// listNow is an intake list's moment in these tests: a snooze till it is
// over, one a microsecond after it is not.
var listNow = time.Date(2026, 11, 20, 9, 0, 0, 123456000, time.UTC)

// newStoreIn is newStore with its sessions in the time zone: a read that
// took a timestamp's day by the session's zone would show (R8).
func newStoreIn(t *testing.T, zone string) (*postgresadapter.Store, *pgxpool.Pool) {
	t.Helper()
	u, err := url.Parse(pgtest.NewDatabase(t))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("timezone", zone)
	u.RawQuery = q.Encode()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: u.String(), MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return postgresadapter.New(pool), pool
}

// intakeItem is a listed work item of intakeWeb: its id, number, times.
type intakeItem struct {
	id               uuid.UUID
	seq              int64
	created, updated time.Time
	status           string
}

// intakeWeb is a project of seven intake items, each status, the two sides
// of a snooze's end at listNow and of a day's two ends, and of the guest,
// Alice, Bob and no one known; and work items an intake list leaves out.
type intakeWeb struct {
	s                                   *postgresadapter.Store
	web                                 project
	guest, alice, bob, label, parent    uuid.UUID
	pend, snoozedOut, snoozedIn         intakeItem
	accepted, declined, duplicate, next intakeItem
	all                                 []intakeItem
	byName                              map[string]uuid.UUID
}

func newIntakeWeb(t *testing.T) intakeWeb {
	t.Helper()
	s, pool := newStoreIn(t, "Asia/Shanghai")
	f := intakeWeb{s: s, guest: newAccount(t, pool, "guest@corp.com"), alice: newAccount(t, pool, "alice@corp.com"),
		bob: newAccount(t, pool, "bob@corp.com")}
	ws := newWorkspace(t, pool, "acme")
	f.web = newProject(t, pool, ws, "WEB")
	ops := newProject(t, pool, ws, "OPS")
	f.label = newLabel(t, pool, f.web, "Bug")
	day18 := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	day19 := day18.AddDate(0, 0, 1)
	f.byName = map[string]uuid.UUID{"guest": f.guest, "alice": f.alice, "bob": f.bob}
	item := func(seq int64, status, by string, created, updated time.Time) intakeItem {
		i := intakeItem{id: uuid.NewV7(), seq: seq, created: created, updated: updated, status: status}
		newIssue(t, pool, f.web, i.id, int(seq), f.alice)
		var creator *uuid.UUID // An unknown creator, nobody's, is NULL.
		if id, known := f.byName[by]; known {
			creator = &id
		}
		exec(t, pool, "UPDATE issues SET created_by_id=$2, created_at=$3, updated_at=$4 WHERE id=$1", i.id, creator, created, updated)
		return i
	}
	f.pend = item(1, "pending", "guest", day19.Add(-time.Microsecond), day19)
	f.snoozedOut = item(2, "snoozed", "alice", day18, day18)
	f.snoozedIn = item(3, "snoozed", "alice", day18.Add(-time.Microsecond), day18.Add(-time.Microsecond))
	f.accepted = item(4, "accepted", "guest", now, now)
	f.declined = item(5, "declined", "nobody", now, now)
	f.duplicate = item(6, "duplicate", "bob", now, now)
	f.next = item(7, "pending", "bob", day19, day19.Add(12*time.Hour))
	f.all = []intakeItem{f.pend, f.snoozedOut, f.snoozedIn, f.accepted, f.declined, f.duplicate, f.next}
	out, in := listNow, listNow.Add(time.Microsecond)
	plain := uuid.NewV7()
	newIssue(t, pool, f.web, plain, 15, f.alice)
	for _, i := range f.all {
		var till *time.Time
		var target *uuid.UUID
		switch {
		case i.id == f.snoozedOut.id:
			till = &out
		case i.id == f.snoozedIn.id:
			till = &in
		case i.status == "duplicate":
			target = &plain
		}
		newIntakeRow(t, pool, f.web, i.id, i.status, till, target, f.alice, false)
	}
	// The pending item's assignee, label, cycle, module and link.
	exec(t, pool, "UPDATE issues SET priority='high' WHERE id=$1", f.pend.id)
	exec(t, pool, "INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id) VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), ws, f.web.id, f.pend.id, f.bob)
	exec(t, pool, "INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id) VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), ws, f.web.id, f.pend.id, f.label)
	cycle, module := newCycle(t, pool, f.web, "Sprint"), newModule(t, pool, f.web, "Auth")
	exec(t, pool, "INSERT INTO cycle_issues(id,workspace_id,project_id,cycle_id,issue_id) VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), ws, f.web.id, cycle, f.pend.id)
	exec(t, pool, "INSERT INTO module_issues(id,workspace_id,project_id,module_id,issue_id) VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), ws, f.web.id, module, f.pend.id)
	exec(t, pool, "INSERT INTO issue_links(id,workspace_id,project_id,issue_id,url) VALUES($1,$2,$3,$4,'https://example.com')", uuid.NewV7(), ws, f.web.id, f.pend.id)
	// The accepted item, the guest's, archived, under Alice's work item,
	// over her sub-item and the guest's; an archived and a deleted one.
	f.parent = uuid.NewV7()
	newIssue(t, pool, f.web, f.parent, 8, f.alice)
	exec(t, pool, "UPDATE issues SET parent_id=$2, archived_at=$3 WHERE id=$1", f.accepted.id, f.parent, now)
	for n, child := range []struct {
		by, state string
	}{{"alice", ""}, {"guest", ""}, {"guest", "archived_at"}, {"alice", "deleted_at"}} {
		id := uuid.NewV7()
		newIssue(t, pool, f.web, id, 9+n, f.byName[child.by])
		exec(t, pool, "UPDATE issues SET parent_id=$2 WHERE id=$1", id, f.accepted.id)
		if child.state != "" {
			exec(t, pool, "UPDATE issues SET "+child.state+"=$2 WHERE id=$1", id, now)
		}
	}
	// Left out: a work item whose one intake row is deleted, a deleted
	// work item's row, the pending row of another project.
	deletedRow, deletedIssue, elsewhere := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, f.web, deletedRow, 13, f.guest)
	newIntakeRow(t, pool, f.web, deletedRow, "pending", nil, nil, f.guest, true)
	newIssue(t, pool, f.web, deletedIssue, 14, f.guest)
	newIntakeRow(t, pool, f.web, deletedIssue, "pending", nil, nil, f.guest, false)
	exec(t, pool, "UPDATE issues SET deleted_at=$2 WHERE id=$1", deletedIssue, now)
	newIssue(t, pool, ops, elsewhere, 1, f.guest)
	newIntakeRow(t, pool, ops, elsewhere, "pending", nil, nil, f.guest, false)
	return f
}

// list reads the page raw asks for, as actor, own-only or not, at listNow.
func (f intakeWeb) list(t *testing.T, raw map[string][]string, actor uuid.UUID, ownOnly bool) []app.IntakeRecord {
	t.Helper()
	q, err := domain.ParseIntakeQuery(raw, actor)
	if err != nil {
		t.Fatal(err)
	}
	return f.read(t, q, actor, ownOnly)
}

func (f intakeWeb) read(t *testing.T, q domain.IntakeQuery, actor uuid.UUID, ownOnly bool) []app.IntakeRecord {
	t.Helper()
	rows, err := f.s.ListIntake(context.Background(), app.IntakeListRead{ProjectID: f.web.id, ActorID: actor, OwnOnly: ownOnly, Query: q, Now: listNow})
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// ids are the items' ids, sorted.
func ids(items ...intakeItem) []uuid.UUID {
	out := make([]uuid.UUID, len(items))
	for n, i := range items {
		out[n] = i.id
	}
	return slices.SortedFunc(slices.Values(out), uuid.UUID.Compare)
}

// The intake list's rows (M7 P2b spec 2.3): the project's undeleted work
// items with an undeleted intake row, accepted and archived too; each
// filter; the statuses effective at the request's moment, a snooze whose
// time is that moment pending; the days half-open in UTC whatever the
// session's zone, the last instant of the day in and the next day's first
// out; the own-only guest's own alone, no unknown creator; the total of
// the filters' rows on each.
func TestTheIntakeListSelectsItsRows(t *testing.T) {
	f := newIntakeWeb(t)
	p, so, si, a, d, dup, n := f.pend, f.snoozedOut, f.snoozedIn, f.accepted, f.declined, f.duplicate, f.next
	for _, tc := range []struct {
		name    string
		raw     map[string][]string
		actor   uuid.UUID
		ownOnly bool
		want    []uuid.UUID
	}{
		{"every row", nil, f.alice, false, ids(f.all...)},
		{"pending, an expired snooze too", map[string][]string{"status": {"pending"}}, f.alice, false, ids(p, so, n)},
		{"snoozed, till after now", map[string][]string{"status": {"snoozed"}}, f.alice, false, ids(si)},
		{"accepted", map[string][]string{"status": {"accepted"}}, f.alice, false, ids(a)},
		{"declined or duplicate", map[string][]string{"status": {"declined,duplicate"}}, f.alice, false, ids(d, dup)},
		{"open", map[string][]string{"status": {"pending,snoozed"}}, f.alice, false, ids(p, so, si, n)},
		{"priority", map[string][]string{"priority": {"high"}}, f.alice, false, ids(p)},
		{"no priority", map[string][]string{"priority": {"none"}}, f.alice, false, ids(so, si, a, d, dup, n)},
		{"label", map[string][]string{"label_id": {f.label.String()}}, f.alice, false, ids(p)},
		{"no label", map[string][]string{"label_id": {"none"}}, f.alice, false, ids(so, si, a, d, dup, n)},
		{"assignee me", map[string][]string{"assignee_id": {"me"}}, f.bob, false, ids(p)},
		{"no assignee", map[string][]string{"assignee_id": {"none"}}, f.alice, false, ids(so, si, a, d, dup, n)},
		{"creator", map[string][]string{"created_by_id": {f.guest.String()}}, f.alice, false, ids(p, a)},
		{"creator me", map[string][]string{"created_by_id": {"me"}}, f.bob, false, ids(dup, n)},
		{"created on a day", map[string][]string{"created_at": {"2026-11-18"}}, f.alice, false, ids(p, so)},
		{"created from a day", map[string][]string{"created_at": {"2026-11-18.."}}, f.alice, false, ids(p, so, n)},
		{"created to a day", map[string][]string{"created_at": {"..2026-11-17"}}, f.alice, false, ids(si, a, d, dup)},
		{"updated on a day", map[string][]string{"updated_at": {"2026-11-19"}}, f.alice, false, ids(p, n)},
		{"updated on the day before", map[string][]string{"updated_at": {"2026-11-18"}}, f.alice, false, ids(so)},
		{"filters together", map[string][]string{"status": {"pending"}, "priority": {"high"}}, f.alice, false, ids(p)},
		{"the own-only guest", nil, f.guest, true, ids(p, a)},
		{"own-only Alice", nil, f.alice, true, ids(so, si)},
		{"no one's but an unknown creator's", map[string][]string{"status": {"declined"}}, f.guest, true, nil},
		{"the guest who sees all", nil, f.guest, false, ids(f.all...)},
	} {
		rows := f.list(t, tc.raw, tc.actor, tc.ownOnly)
		got := make([]uuid.UUID, len(rows))
		for k, r := range rows {
			got[k] = r.Issue.ID
			if r.Total != int64(len(tc.want)) {
				t.Errorf("%s: row %s total %d, want %d", tc.name, r.Issue.ID, r.Total, len(tc.want))
			}
		}
		if slices.SortFunc(got, uuid.UUID.Compare); !slices.Equal(got, tc.want) {
			t.Errorf("%s: rows %v, want %v", tc.name, got, tc.want)
		}
	}
}

// A row is the work item as a list row has it, its intake row as stored
// (the moment's status is the use case's), its sub-items counted as the
// project list counts them: the caller's alone when he sees only his own.
func TestTheIntakeListsRows(t *testing.T) {
	f := newIntakeWeb(t)
	byID := func(rows []app.IntakeRecord) map[uuid.UUID]domain.Issue {
		m := map[uuid.UUID]domain.Issue{}
		for _, r := range rows {
			m[r.Issue.ID] = r.Issue
		}
		return m
	}
	all := byID(f.list(t, nil, f.alice, false))
	pend := all[f.pend.id]
	if pend.Priority != "high" || !slices.Equal(pend.AssigneeIDs, []uuid.UUID{f.bob}) || !slices.Equal(pend.LabelIDs, []uuid.UUID{f.label}) ||
		pend.CycleID == nil || len(pend.ModuleIDs) != 1 || pend.LinkCount != 1 || pend.SequenceID != 1 || pend.ProjectID != f.web.id ||
		pend.CreatedByID == nil || *pend.CreatedByID != f.guest || pend.DescriptionHTML != "" || pend.Intake == nil ||
		*pend.Intake != (domain.IntakeFacts{Status: domain.IntakePending}) {
		t.Errorf("the pending row = %+v", pend)
	}
	if out := all[f.snoozedOut.id]; out.Intake == nil || out.Intake.Status != domain.IntakeSnoozed || out.Intake.SnoozedTill == nil ||
		!out.Intake.SnoozedTill.Equal(listNow) || out.AssigneeIDs == nil || out.LabelIDs == nil || out.ModuleIDs == nil {
		t.Errorf("the expired snooze's row = %+v; want its row as stored, snoozed till now, and empty sets", out)
	}
	if dup := all[f.duplicate.id]; dup.Intake == nil || dup.Intake.Status != domain.IntakeDuplicate || dup.Intake.DuplicateToID == nil {
		t.Errorf("the duplicate's row = %+v", dup)
	}
	if declined := all[f.declined.id]; declined.CreatedByID != nil {
		t.Errorf("the unknown creator's row = %+v", declined)
	}
	accepted := all[f.accepted.id]
	if accepted.ArchivedAt == nil || accepted.ParentID == nil || *accepted.ParentID != f.parent || accepted.SubIssuesCount != 2 {
		t.Errorf("the accepted row = %+v; want archived, under its parent, two sub-items", accepted)
	}
	own := byID(f.list(t, nil, f.guest, true))[f.accepted.id]
	if own.SubIssuesCount != 1 || own.ParentID == nil {
		t.Errorf("the accepted row for the own-only guest = %+v; want his one sub-item, the parent as stored", own)
	}
}

// Each order pages with no row twice and none skipped, the key's equal
// values by the id the same way, and keeps its total.
func TestTheIntakeListPages(t *testing.T) {
	f := newIntakeWeb(t)
	for _, order := range []domain.IntakeOrder{"created_at", "-created_at", "updated_at", "-updated_at", "sequence_id", "-sequence_id"} {
		want := slices.Clone(f.all)
		key := func(i intakeItem) int64 {
			switch order.Key() {
			case "created_at":
				return i.created.UnixMicro()
			case "updated_at":
				return i.updated.UnixMicro()
			}
			return i.seq
		}
		slices.SortFunc(want, func(x, y intakeItem) int {
			c := cmp.Or(cmp.Compare(key(x), key(y)), x.id.Compare(y.id))
			if order.Descending() {
				return -c
			}
			return c
		})
		q := domain.IntakeQuery{Order: order, Limit: 2}
		var got []uuid.UUID
		for pages := 0; ; pages++ {
			rows := f.read(t, q, f.alice, false)
			if pages > len(f.all) || len(rows) == 0 {
				t.Fatalf("%s: page %d has %d rows", order, pages, len(rows))
			}
			for _, r := range rows {
				if r.Total != int64(len(f.all)) || r.Position.Order != order || r.Position.ID != r.Issue.ID {
					t.Errorf("%s: row %+v", order, r)
				}
			}
			size := min(len(rows), q.Limit)
			for _, r := range rows[:size] {
				got = append(got, r.Issue.ID)
			}
			if len(rows) <= q.Limit {
				break
			}
			q.Cursor = &rows[size-1].Position
		}
		wantIDs := make([]uuid.UUID, len(want))
		for n, i := range want {
			wantIDs[n] = i.id
		}
		if !slices.Equal(got, wantIDs) {
			t.Errorf("%s: pages %v, want %v", order, got, wantIDs)
		}
	}
}
