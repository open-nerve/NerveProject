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

// newDeleteLabel is DeleteLabel over newLabels' fakes, its clock logged.
func newDeleteLabel() (*app.DeleteLabel, *writeFixture, *fakeLabels) {
	f, l := newLabels()
	return app.NewDeleteLabel(f.locks(), l, f.tx, clockAt{clockNow, f.log}), f, l
}

// labelDeleted are the calls of user's deletion of the label id of
// project: its locks and decision, the clock, the deletion by user at that
// time.
func labelDeleted(user, id, project uuid.UUID) []string {
	return append(labelLocked(id, project, user, domain.ActionLabelDelete), "Now",
		fmt.Sprintf("DeleteLabel %s by %s at %s", id, user, clockNow.Format(timeFormat)))
}

// DeleteLabel, in one transaction and in the order of M3 design 3.6 and
// 3.16, locks the label's project, decides, reads the clock, then deletes
// the label and the labels under it by the caller at that time: Bug takes
// UI with it; UI, under Bug, and Feature go alone; and ops's Docs, of an
// archived project, goes as any other (3.19).
func TestDeleteLabel(t *testing.T) {
	for _, tt := range []struct {
		name    string
		id      uuid.UUID
		project uuid.UUID
		gone    []uuid.UUID // the labels the deletion takes
	}{
		{"Bug, with UI under it", webBug, webID, []uuid.UUID{webBug, webUI}},
		{"UI, under Bug", webUI, webID, []uuid.UUID{webUI}},
		{"Feature", webFeature, webID, []uuid.UUID{webFeature}},
		{"archived ops's Docs", opsDocs, opsID, []uuid.UUID{opsDocs}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newDeleteLabel()
			want := maps.Clone(l.labels)
			for _, id := range tt.gone {
				delete(want, id)
			}
			err := uc.Execute(as(bob), tt.id)
			if err != nil || !maps.Equal(l.labels, want) {
				t.Errorf("Execute() = %v, the labels after it %v; want nil, %v", err, l.labels, want)
			}
			if calls := labelDeleted(bob, tt.id, tt.project); !slices.Equal(f.log.calls, calls) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, calls)
			}
		})
	}
}

// Refusals, each in its place, and no label deleted: no caller, before the
// transaction; a label that is not there, a workspace or project deleted
// while its lock waited, a label deleted or moved to ops meanwhile, and a
// caller who does not see web, each project.label_not_found; a member, the
// Authorizer's 403. The label read for another id is the shared path's,
// pinned once in lock_test.go.
func TestDeleteLabelRefuses(t *testing.T) {
	upTo := func(n int) []string { return labelLocked(webBug, webID, bob, domain.ActionLabelDelete)[:n] }
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		set   func(f *writeFixture)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webBug, nil, shared.Unauthenticated(), nil},
		{"no label", as(bob), uuid.Nil(), nil, domain.ErrLabelNotFound, []string{"Begin", "LabelByID " + uuid.Nil().String()}},
		{"acme deleted while its lock waited", as(bob), webBug, func(f *writeFixture) { f.workspaces.gone = true },
			domain.ErrLabelNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webBug, func(f *writeFixture) { f.store.deleted = true },
			domain.ErrLabelNotFound, upTo(4)},
		{"the label deleted while the locks waited", as(bob), webBug, func(f *writeFixture) { f.store.reread.gone = true },
			domain.ErrLabelNotFound, upTo(5)},
		{"the label moved to ops", as(bob), webBug, func(f *writeFixture) { f.store.reread.project = opsID },
			domain.ErrLabelNotFound, upTo(5)},
		{"a caller who does not see web", as(erin), webBug, nil, domain.ErrLabelNotFound,
			labelLocked(webBug, webID, erin, domain.ActionLabelDelete)},
		{"a member", as(alice), webBug, nil, shared.Forbidden(), labelLocked(webBug, webID, alice, domain.ActionLabelDelete)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newDeleteLabel()
			if tt.set != nil {
				tt.set(f)
			}
			before := maps.Clone(l.labels)
			err := uc.Execute(tt.ctx, tt.id)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if !maps.Equal(l.labels, before) {
				t.Errorf("the labels after the refusal: %v; want them as they were, %v", l.labels, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestDeleteLabelReturnsEachFailure(t *testing.T) {
	all := labelDeleted(bob, webBug, webID)
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the deletion's calls ran
	}{
		{"the label's read", fail("LabelByID"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the read under the locks", func(f *writeFixture) { f.store.reread.err = errDisk }, 5},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 6},
		{"the deletion", fail("DeleteLabel"), 8},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, _ := newDeleteLabel()
			tt.fail(f)
			err := uc.Execute(as(bob), webBug)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}
