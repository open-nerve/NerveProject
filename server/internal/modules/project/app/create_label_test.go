package app_test

import (
	"context"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newCreateLabel is CreateLabel over newLabels' fakes, its clock logged.
func newCreateLabel() (*app.CreateLabel, *writeFixture, *fakeLabels) {
	f, l := newLabels()
	return app.NewCreateLabel(f.locks(), l, f.tx, clockAt{clockNow, f.log}), f, l
}

// qa is the label the tests create: QA, at the top, a name no label of web
// or ops has, in any case; underBug is QA under web's Bug.
var (
	qa       = domain.LabelCreate{Name: "QA", Color: "#0EA5E9"}
	underBug = domain.LabelCreate{Name: "QA", Color: "#0EA5E9", ParentID: &webBug}
)

// labelCreated are the calls of user's creation of in in project at
// sortOrder: its locks and decision; the parent's read, when in gives one;
// the greatest sort order; the clock; the insert of the label, by user at
// that time.
func labelCreated(user, project uuid.UUID, in domain.LabelCreate, sortOrder float64) []string {
	calls := lockedDecision(user, project, domain.ActionLabelCreate)
	if in.ParentID != nil {
		calls = append(calls, "LabelByID "+in.ParentID.String())
	}
	return append(calls, "GreatestSortOrder "+project.String(), "Now", "CreateLabel "+labelRow(app.LabelRow{WorkspaceID: acme.ID, ProjectID: project, ParentID: in.ParentID,
		Name: in.Name, Color: in.Color, SortOrder: sortOrder, CreatedBy: user, Now: clockNow}))
}

// CreateLabel, in one transaction and in the order of M3 design 3.6, locks
// the project, decides, reads the parent given, the greatest sort order of
// its labels and the clock, then inserts the label by the caller at that
// time; it answers the label as stored, the time to the microsecond:
// web's at the top after its Feature, 95535; under its Bug, the same;
// archived ops's, as any other's (3.19), after its Docs, 75535; and ops's,
// its labels gone, at 65535. The use case makes each label's id: a second
// label's differs from the first's, and neither is the nil id, the
// project's, or a label's the fixture had.
func TestCreateLabel(t *testing.T) {
	for _, tt := range []struct {
		name      string
		project   uuid.UUID
		in        domain.LabelCreate
		set       func(l *fakeLabels)
		sortOrder float64
	}{
		{"web, at the top", webID, qa, nil, 95535},
		{"under web's Bug", webID, underBug, nil, 95535},
		{"archived ops", opsID, qa, nil, 75535},
		{"ops without labels", opsID, qa, func(l *fakeLabels) { delete(l.labels, opsDocs) }, 65535},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newCreateLabel()
			if tt.set != nil {
				tt.set(l)
			}
			before := maps.Clone(l.labels)
			got, err := uc.Execute(as(bob), tt.project, tt.in)
			want := domain.Label{ID: got.ID, WorkspaceID: acme.ID, ProjectID: tt.project, ParentID: tt.in.ParentID, Name: tt.in.Name,
				Color: tt.in.Color, SortOrder: tt.sortOrder, CreatedAt: now, UpdatedAt: now}
			if err != nil || labelJSON(t, got) != labelJSON(t, want) || labelJSON(t, l.labels[got.ID]) != labelJSON(t, want) {
				t.Errorf("Execute() = %+v, %v, stored %+v; want %+v", got, err, l.labels[got.ID], want)
			}
			if want := labelCreated(bob, tt.project, tt.in, tt.sortOrder); !slices.Equal(f.log.calls, want) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, want)
			}
			second := tt.in
			second.Name = "QA again"
			again, err := uc.Execute(as(bob), tt.project, second)
			if err != nil || again.ID == got.ID {
				t.Errorf("a second label = %s, %v; want another id than the first's, %s", again.ID, err, got.ID)
			}
			for _, id := range []uuid.UUID{got.ID, again.ID} {
				if _, had := before[id]; had || id == uuid.Nil() || id == tt.project {
					t.Errorf("a label created as %s; want a new id: not the nil id, the project's or a label's the fixture had", id)
				}
			}
		})
	}
}

// Refusals, each in its place, and no label stored: no caller, and values
// the domain refuses, before the transaction; a project that is not there,
// a workspace or project deleted while its lock waited, a project moved
// meanwhile, and a caller who does not see the project, each
// project.not_found; a member, the Authorizer's 403. A parent that is no
// label, another project's, or one under another, after the decision,
// each 422 parent_id not_allowed. A name another label of the project has,
// in another case, is the store's project.label_name_taken, after the
// insert. The parent read for another id, and the label answered for
// another id, are each the write's own error.
func TestCreateLabelRefuses(t *testing.T) {
	upTo := func(n int) []string { return lockedDecision(bob, webID, domain.ActionLabelCreate)[:n] }
	under := func(parent uuid.UUID) domain.LabelCreate { return domain.LabelCreate{Name: "QA", ParentID: &parent} }
	parentRefused := shared.Invalid(shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed})
	none := uuid.NewV7()
	taken := domain.LabelCreate{Name: "bug"}
	for _, tt := range []struct {
		name    string
		ctx     context.Context
		project uuid.UUID
		in      domain.LabelCreate
		set     func(f *writeFixture, l *fakeLabels)
		want    error
		calls   []string
	}{
		{"no caller", context.Background(), webID, qa, nil, shared.Unauthenticated(), nil},
		{"a blank name", as(bob), webID, domain.LabelCreate{Name: " "}, nil,
			shared.Invalid(shared.FieldError{Field: "name", Code: shared.FieldTooShort}), nil},
		{"no project", as(bob), uuid.Nil(), qa, nil, domain.ErrNotFound, noProject},
		{"acme deleted while its lock waited", as(bob), webID, qa, func(f *writeFixture, _ *fakeLabels) { f.workspaces.gone = true },
			domain.ErrNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webID, qa, func(f *writeFixture, _ *fakeLabels) { f.store.deleted = true },
			domain.ErrNotFound, upTo(4)},
		{"web moved to another workspace", as(bob), webID, qa, func(f *writeFixture, _ *fakeLabels) { f.store.moved = uuid.NewV7() },
			domain.ErrNotFound, upTo(4)},
		{"a caller who does not see web", as(erin), webID, qa, nil, domain.ErrNotFound,
			lockedDecision(erin, webID, domain.ActionLabelCreate)},
		{"a member", as(alice), webID, under(webBug), nil, shared.Forbidden(), lockedDecision(alice, webID, domain.ActionLabelCreate)},
		{"a parent that is no label", as(bob), webID, under(none), nil, parentRefused,
			append(lockedDecision(bob, webID, domain.ActionLabelCreate), "LabelByID "+none.String())},
		{"ops's Docs as the parent", as(bob), webID, under(opsDocs), nil, parentRefused,
			append(lockedDecision(bob, webID, domain.ActionLabelCreate), "LabelByID "+opsDocs.String())},
		{"web's UI, under Bug, as the parent", as(bob), webID, under(webUI), nil, parentRefused,
			append(lockedDecision(bob, webID, domain.ActionLabelCreate), "LabelByID "+webUI.String())},
		{"the parent read for another id", as(bob), webID, under(webBug), func(_ *writeFixture, l *fakeLabels) { l.answersAs = webFeature },
			nil, append(lockedDecision(bob, webID, domain.ActionLabelCreate), "LabelByID "+webBug.String())},
		{"a name web's Bug has, in another case", as(bob), webID, taken, nil, domain.ErrLabelNameTaken,
			labelCreated(bob, webID, taken, 95535)},
		{"the label answered for another id", as(bob), webID, qa, func(_ *writeFixture, l *fakeLabels) { l.changedAs = uuid.NewV7() },
			nil, labelCreated(bob, webID, qa, 95535)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newCreateLabel()
			if tt.set != nil {
				tt.set(f, l)
			}
			before := maps.Clone(l.labels)
			_, err := uc.Execute(tt.ctx, tt.project, tt.in)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if tt.name != "the label answered for another id" && !maps.Equal(l.labels, before) {
				t.Errorf("the labels after the refusal: %v; want them as they were, %v", l.labels, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestCreateLabelReturnsEachFailure(t *testing.T) {
	all := labelCreated(bob, webID, underBug, 95535)
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the creation's calls ran
	}{
		{"the project's workspace", fail("ProjectWorkspace"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 5},
		{"the parent", fail("LabelByID"), 6},
		{"the greatest sort order", fail("GreatestSortOrder"), 7},
		{"the insert", fail("CreateLabel"), 9},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, _ := newCreateLabel()
			tt.fail(f)
			_, err := uc.Execute(as(bob), webID, underBug)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}
