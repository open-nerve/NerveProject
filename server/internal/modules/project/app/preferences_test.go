package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newPreferences is newWrites with carol's grant in acme too, a project
// guest's, so that a member without display settings reads and writes
// hers.
func newPreferences() *writeFixture {
	f := newWrites()
	f.auth.grants[grantKey{carol, acme.ID}] = shared.Grant{WorkspaceRole: shared.RoleGuest, ProjectRole: shared.RoleGuest}
	return f
}

// refusedAs reports whether err is want: errDisk itself, a port's failure
// come back with no *shared.Error in its chain, which the API would answer
// as a refusal; or the same refusal (sameError).
func refusedAs(err, want error) bool {
	if want == errDisk {
		var se *shared.Error
		return errors.Is(err, errDisk) && !errors.As(err, &se)
	}
	return sameError(err, want)
}

// readRefusal is a row of a read under a project's refusals (readRefusals):
// its caller and project, a port's failure, the error and the calls.
type readRefusal struct {
	name  string
	ctx   context.Context
	id    uuid.UUID
	fail  func(f *writeFixture)
	want  error
	calls []string
}

// readRefusals are the refusals of a read under a project, through
// findAndDecide, each in its place, calls being the read's calls by a user
// in a project: no caller; a project not there, and erin, who does not see
// web: 404; alice, the Authorizer's 403. Then each port's failure, come
// back as itself after the calls before it and none after: the project's,
// the decision's, and that of port, the read's own, named what.
func readRefusals(calls func(user, project uuid.UUID) []string, what, port string) []readRefusal {
	return []readRefusal{
		{"no caller", context.Background(), webID, nil, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), nil, domain.ErrNotFound, []string{"ProjectWorkspace " + uuid.Nil().String() + " outside tx"}},
		{"not seen", as(erin), webID, nil, domain.ErrNotFound, calls(erin, webID)[:2]},
		{"forbidden", as(alice), webID, nil, shared.Forbidden(), calls(alice, webID)[:2]},
		{"the project failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ProjectWorkspace": errDisk} }, errDisk,
			calls(bob, webID)[:1]},
		{"the decision failing", as(bob), webID, func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, errDisk,
			calls(bob, webID)[:2]},
		{what + " failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{port: errDisk} }, errDisk, calls(bob, webID)},
	}
}

// read are the calls of user's read of his display settings in project: the
// project's workspace, the decision, his settings; none in a transaction.
func read(user, project uuid.UUID) []string {
	return []string{"ProjectWorkspace " + project.String() + " outside tx",
		fmt.Sprintf("Authorize %s %s on %s/%s outside tx", user, domain.ActionPreferencesRead, acme.ID, project),
		fmt.Sprintf("Preferences %s for %s outside tx", project, user)}
}

// GetProjectPreferences answers the caller's display settings in the
// project after the decision on project_preferences.read, or the defaults
// while he has none; it opens no transaction and writes nothing.
func TestGetProjectPreferences(t *testing.T) {
	for _, tt := range []struct {
		user uuid.UUID
		want domain.Preferences
	}{{bob, bobsTabs}, {carol, domain.DefaultPreferences()}} {
		f := newPreferences()
		got, err := app.NewGetProjectPreferences(f.store, f.auth).Execute(as(tt.user), webID)
		if err != nil || !reflect.DeepEqual(got, tt.want) || !slices.Equal(f.log.calls, read(tt.user, webID)) {
			t.Errorf("Execute() for %s = %+v, %v, calls %q; want %+v, calls %q", tt.user, got, err, f.log.calls, tt.want, read(tt.user, webID))
		}
	}
}

// Refusals, each in its place: no caller; a project not there, or not
// visible: 404; a caller the Authorizer refuses: its 403. Every port's
// failure comes back as itself, after the calls before it and none after.
func TestGetProjectPreferencesRefuses(t *testing.T) {
	for _, tt := range readRefusals(read, "the settings", "Preferences") {
		t.Run(tt.name, func(t *testing.T) {
			f := newPreferences()
			if tt.fail != nil {
				tt.fail(f)
			}
			got, err := app.NewGetProjectPreferences(f.store, f.auth).Execute(tt.ctx, tt.id)
			if !refusedAs(err, tt.want) || !reflect.DeepEqual(got, domain.Preferences{}) || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
		})
	}
}

// newUpdatePreferences is UpdateProjectPreferences over newPreferences'
// fakes, its clock logged.
func newUpdatePreferences() (*app.UpdateProjectPreferences, *writeFixture) {
	f := newPreferences()
	return app.NewUpdateProjectPreferences(f.store, f.locks(), f.tx, clockAt{clockNow, f.log}), f
}

// changed are the calls of user's change p of his display settings in
// project: the transaction, the project's workspace, its lock, the project
// FOR SHARE, the decision, the clock, the change.
func changed(user, project uuid.UUID, p domain.PreferencesPatch) []string {
	patch, _ := json.Marshal(p)
	return append(lockedTo(project), "ShareProject "+project.String(),
		fmt.Sprintf("Authorize %s %s on %s/%s", user, domain.ActionPreferencesUpdate, acme.ID, project), "Now",
		fmt.Sprintf("UpsertPreferences %s/%s for %s %s at %s", acme.ID, project, user, patch, clockNow.Format(timeFormat)))
}

// UpdateProjectPreferences checks the change, then, in one transaction,
// takes the project's locks, the project FOR SHARE, decides, reads the clock and changes the
// caller's settings, or makes them from the defaults while he has none; an
// archived project's change too. The answer is the settings as stored.
func TestUpdateProjectPreferences(t *testing.T) {
	cycles := domain.Navigation{DefaultTab: "cycles", HideInMoreMenu: []string{"intake"}}
	tests := []struct {
		name    string
		user    uuid.UUID
		project uuid.UUID
		in      domain.PreferencesPatch
		want    domain.Preferences
	}{
		{"bob's tabs", bob, webID, domain.PreferencesPatch{Navigation: &cycles}, domain.Preferences{Navigation: cycles, SortOrder: 10}},
		{"carol's place, from the defaults", carol, webID, domain.PreferencesPatch{SortOrder: ptr(-2.5)},
			domain.Preferences{Navigation: domain.DefaultPreferences().Navigation, SortOrder: -2.5}},
		{"in the archived project", bob, opsID, domain.PreferencesPatch{SortOrder: ptr(3.0)},
			domain.Preferences{Navigation: domain.DefaultPreferences().Navigation, SortOrder: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdatePreferences()
			got, err := uc.Execute(as(tt.user), tt.project, tt.in)
			if want := changed(tt.user, tt.project, tt.in); err != nil || !reflect.DeepEqual(got, tt.want) || !slices.Equal(f.log.calls, want) {
				t.Errorf("Execute() = %+v, %v, calls\n%q\nwant %+v, calls\n%q", got, err, f.log.calls, tt.want, want)
			}
		})
	}
}

// Refusals, each in its place, and nothing changed: an unknown tab, and no
// caller, before the transaction; a project not there, deleted while its
// FOR SHARE waited, of another workspace by the time it is shared, or not
// visible: 404; a caller the Authorizer refuses: its 403. Every port's
// failure comes back as itself, the commit's too, after the calls before it
// and none after.
func TestUpdateProjectPreferencesRefuses(t *testing.T) {
	in := domain.PreferencesPatch{SortOrder: ptr(1.0)}
	all := changed(bob, webID, in)
	moved := func(f *writeFixture) { f.store.moved = uuid.NewV7() }
	deleted := func(f *writeFixture) { f.store.deleted = true }
	tests := []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		in    domain.PreferencesPatch
		setup func(f *writeFixture) // the project's state, when not as newWrites has it
		fail  func(f *writeFixture) // a port's failure
		want  error
		calls []string
	}{
		{"an unknown tab", as(bob), webID, domain.PreferencesPatch{Navigation: &domain.Navigation{DefaultTab: "pages"}}, nil, nil,
			shared.Invalid(shared.FieldError{Field: "navigation.default_tab", Code: "invalid_format"}), nil},
		{"no caller", context.Background(), webID, in, nil, nil, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), in, nil, nil, domain.ErrNotFound, noProject},
		{"moved to another workspace", as(bob), webID, in, moved, nil, domain.ErrNotFound,
			append(lockedTo(webID), "ShareProject "+webID.String())},
		{"deleted while its lock waited", as(bob), webID, in, deleted, nil, domain.ErrNotFound,
			append(lockedTo(webID), "ShareProject "+webID.String())},
		{"not seen", as(erin), webID, in, nil, nil, domain.ErrNotFound, changed(erin, webID, in)[:5]},
		{"forbidden", as(alice), webID, in, nil, nil, shared.Forbidden(), changed(alice, webID, in)[:5]},
		{"the lock failing", as(bob), webID, in, nil, func(f *writeFixture) { f.store.errs = map[string]error{"ShareProject": errDisk} }, errDisk,
			all[:4]},
		{"the decision failing", as(bob), webID, in, nil, func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, errDisk,
			all[:5]},
		{"the change failing", as(bob), webID, in, nil, func(f *writeFixture) { f.store.errs = map[string]error{"UpsertPreferences": errDisk} },
			errDisk, all},
		{"the commit failing", as(bob), webID, in, nil, func(f *writeFixture) { f.tx.commitErr = errDisk }, errDisk, all},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newUpdatePreferences()
			if tt.setup != nil {
				tt.setup(f)
			}
			if tt.fail != nil {
				tt.fail(f)
			}
			got, err := uc.Execute(tt.ctx, tt.id, tt.in)
			if !refusedAs(err, tt.want) || !reflect.DeepEqual(got, domain.Preferences{}) || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
			if tt.fail == nil && !reflect.DeepEqual(f.store.projects[webID].prefs, map[uuid.UUID]domain.Preferences{bob: bobsTabs}) {
				t.Errorf("the settings after the refusal: %+v, want bob's alone, as they were", f.store.projects[webID].prefs)
			}
		})
	}
}
