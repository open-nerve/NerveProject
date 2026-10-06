package app_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newUpdateLabel is UpdateLabel over newLabels' fakes, its clock logged.
func newUpdateLabel() (*app.UpdateLabel, *writeFixture, *fakeLabels) {
	f, l := newLabels()
	return app.NewUpdateLabel(f.locks(), l, f.tx, clockAt{clockNow, f.log}), f, l
}

// labelLocked are the calls of a write by caller on the label id of
// project up to its decision on action: the transaction, the label read,
// acme's row FOR SHARE, the project FOR NO KEY UPDATE, the label read
// again, the decision.
func labelLocked(id, project, caller uuid.UUID, action shared.Action) []string {
	return rowLocked("LabelByID", id, project, caller, action)
}

// labelUpdated are the calls of user's change p of the label id of
// project: its locks and decision; whether it has labels under it and the
// parent's read, when p gives a parent; the clock; the change by user at
// that time.
func labelUpdated(user, id, project uuid.UUID, p domain.LabelPatch) []string {
	calls := labelLocked(id, project, user, domain.ActionLabelUpdate)
	if p.SetParent && p.ParentID != nil {
		calls = append(calls, "HasChildren "+id.String(), "LabelByID "+p.ParentID.String())
	}
	return append(calls, "Now", fmt.Sprintf("UpdateLabel %s %s by %s at %s", id, labelPatch(p), user, clockNow.Format(timeFormat)))
}

// UpdateLabel, in one transaction and in the order of M3 design 3.6, locks
// the label's project, decides, reads whether the label has labels under
// it and the parent when a parent is given, reads the clock, then changes
// the fields given, by the caller at that time; it answers the label as
// stored, the time to the microsecond. Feature is renamed and its color
// set to none, moved under Bug, and given a sort order; Bug is renamed in
// another case, its own name; UI goes to the top, which reads neither its
// children nor a parent; and ops's Docs, of an archived project, is renamed
// and recolored as any other (3.19).
func TestUpdateLabel(t *testing.T) {
	for _, tt := range []struct {
		name    string
		id      uuid.UUID
		p       domain.LabelPatch
		changes func(l *domain.Label) // what p changes of the label as it was
	}{
		{"Feature renamed and recolored", webFeature, domain.LabelPatch{Name: ptr("Story"), Color: ptr("")},
			func(l *domain.Label) { l.Name, l.Color = "Story", "" }},
		{"Feature under Bug", webFeature, domain.LabelPatch{SetParent: true, ParentID: &webBug},
			func(l *domain.Label) { l.ParentID = &webBug }},
		{"Feature given a sort order", webFeature, domain.LabelPatch{SortOrder: ptr(-0.5)}, func(l *domain.Label) { l.SortOrder = -0.5 }},
		{"Bug renamed in another case", webBug, domain.LabelPatch{Name: ptr("BUG")}, func(l *domain.Label) { l.Name = "BUG" }},
		{"UI to the top", webUI, domain.LabelPatch{SetParent: true}, func(l *domain.Label) { l.ParentID = nil }},
		{"archived ops's Docs", opsDocs, domain.LabelPatch{Name: ptr("Guides"), Color: ptr("#FF0000")},
			func(l *domain.Label) { l.Name, l.Color = "Guides", "#FF0000" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newUpdateLabel()
			before := l.labels[tt.id]
			got, err := uc.Execute(as(bob), tt.id, tt.p)
			want := before
			tt.changes(&want)
			want.UpdatedAt = now
			if err != nil || labelJSON(t, got) != labelJSON(t, want) || labelJSON(t, l.labels[tt.id]) != labelJSON(t, want) {
				t.Errorf("Execute() = %+v, %v, stored %+v; want %+v", got, err, l.labels[tt.id], want)
			}
			if calls := labelUpdated(bob, tt.id, before.ProjectID, tt.p); !slices.Equal(f.log.calls, calls) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, calls)
			}
		})
	}
}

// Refusals, each in its place, and no label changed: no caller, and values
// the domain refuses, before the transaction; a label that is not there, a
// workspace or project deleted while its lock waited, a label deleted or
// moved to ops meanwhile, and a caller who does not see web, each
// project.label_not_found; a member, the Authorizer's 403. Then, after the
// decision, each a 422 parent_id not_allowed: Feature under itself, under
// UI, which is under Bug, under ops's Docs, or under a label that is not
// there; and Bug, which has UI under it, under Feature. Feature renamed to
// Bug's name in another case is the store's project.label_name_taken. The
// change answered for another label is the write's own error; the label
// read for another id is the shared path's, pinned once in lock_test.go.
func TestUpdateLabelRefuses(t *testing.T) {
	upTo := func(n int) []string { return labelLocked(webFeature, webID, bob, domain.ActionLabelUpdate)[:n] }
	rename := domain.LabelPatch{Name: ptr("Story")}
	under := func(parent uuid.UUID) domain.LabelPatch { return domain.LabelPatch{SetParent: true, ParentID: &parent} }
	// parentRead are the calls of the move of id under parent up to the parent's read.
	parentRead := func(id, parent uuid.UUID) []string { return labelUpdated(bob, id, webID, under(parent))[:8] }
	parentRefused := shared.Invalid(shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed})
	none := uuid.NewV7()
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		p     domain.LabelPatch
		set   func(f *writeFixture, l *fakeLabels)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webFeature, rename, nil, shared.Unauthenticated(), nil},
		{"a blank name", as(bob), webFeature, domain.LabelPatch{Name: ptr("")}, nil,
			shared.Invalid(shared.FieldError{Field: "name", Code: shared.FieldTooShort}), nil},
		{"no label", as(bob), uuid.Nil(), rename, nil, domain.ErrLabelNotFound, []string{"Begin", "LabelByID " + uuid.Nil().String()}},
		{"acme deleted while its lock waited", as(bob), webFeature, rename, func(f *writeFixture, _ *fakeLabels) { f.workspaces.gone = true },
			domain.ErrLabelNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webFeature, rename, func(f *writeFixture, _ *fakeLabels) { f.store.deleted = true },
			domain.ErrLabelNotFound, upTo(4)},
		{"the label deleted while the locks waited", as(bob), webFeature, rename,
			func(f *writeFixture, _ *fakeLabels) { f.store.reread.gone = true }, domain.ErrLabelNotFound, upTo(5)},
		{"the label moved to ops", as(bob), webFeature, rename, func(f *writeFixture, _ *fakeLabels) { f.store.reread.project = opsID },
			domain.ErrLabelNotFound, upTo(5)},
		{"a caller who does not see web", as(erin), webFeature, rename, nil, domain.ErrLabelNotFound,
			labelLocked(webFeature, webID, erin, domain.ActionLabelUpdate)},
		{"a member", as(alice), webFeature, rename, nil, shared.Forbidden(), labelLocked(webFeature, webID, alice, domain.ActionLabelUpdate)},
		{"Feature under itself", as(bob), webFeature, under(webFeature), nil, parentRefused, parentRead(webFeature, webFeature)},
		{"Feature under UI, which is under Bug", as(bob), webFeature, under(webUI), nil, parentRefused, parentRead(webFeature, webUI)},
		{"Feature under ops's Docs", as(bob), webFeature, under(opsDocs), nil, parentRefused, parentRead(webFeature, opsDocs)},
		{"Feature under a label that is not there", as(bob), webFeature, under(none), nil, parentRefused, parentRead(webFeature, none)},
		{"Bug, with UI under it, under Feature", as(bob), webBug, under(webFeature), nil, parentRefused, parentRead(webBug, webFeature)},
		{"Feature renamed to Bug's name in another case", as(bob), webFeature, domain.LabelPatch{Name: ptr("bug")}, nil,
			domain.ErrLabelNameTaken, labelUpdated(bob, webFeature, webID, domain.LabelPatch{Name: ptr("bug")})},
		{"the change answered for another label", as(bob), webFeature, rename,
			func(_ *writeFixture, l *fakeLabels) { l.changedAs = uuid.NewV7() }, nil, labelUpdated(bob, webFeature, webID, rename)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newUpdateLabel()
			if tt.set != nil {
				tt.set(f, l)
			}
			before := maps.Clone(l.labels)
			_, err := uc.Execute(tt.ctx, tt.id, tt.p)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if tt.name != "the change answered for another label" && !maps.Equal(l.labels, before) {
				t.Errorf("the labels after the refusal: %v; want them as they were, %v", l.labels, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after: Feature moved under Bug, so that the
// children and the parent are read.
func TestUpdateLabelReturnsEachFailure(t *testing.T) {
	move := domain.LabelPatch{SetParent: true, ParentID: &webBug}
	all := labelUpdated(bob, webFeature, webID, move)
	fail := func(method string) func(f *writeFixture, _ *fakeLabels) {
		return func(f *writeFixture, _ *fakeLabels) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture, l *fakeLabels)
		calls int // how many of the change's calls ran
	}{
		{"the label's read", fail("LabelByID"), 2},
		{"the workspace's lock", func(f *writeFixture, _ *fakeLabels) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the read under the locks", func(f *writeFixture, _ *fakeLabels) { f.store.reread.err = errDisk }, 5},
		{"the decision", func(f *writeFixture, _ *fakeLabels) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 6},
		{"the children", fail("HasChildren"), 7},
		{"the parent", func(_ *writeFixture, l *fakeLabels) { l.failsFor = webBug }, 8},
		{"the change", fail("UpdateLabel"), 10},
		{"the commit", func(f *writeFixture, _ *fakeLabels) { f.tx.commitErr = errDisk }, 10},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newUpdateLabel()
			tt.fail(f, l)
			_, err := uc.Execute(as(bob), webFeature, move)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}
