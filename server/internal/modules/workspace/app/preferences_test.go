package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// prefsFixture is the preference use cases over fakes sharing one log:
// alice is a guest of acme and has settings there, bob a member of acme and
// of beta without any; carol sees neither.
type prefsFixture struct {
	log        *callLog
	tx         *fakeTx
	workspaces *fakeWorkspaces
	auth       *fakeAuthorizer
	get        *app.GetWorkspacePreferences
	update     *app.UpdateWorkspacePreferences
}

var tabbed = domain.Preferences{NavigationControl: "TABBED", NavigationProjectLimit: 3}

func newPrefs() *prefsFixture {
	log := &callLog{}
	f := &prefsFixture{log: log, tx: &fakeTx{},
		workspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta},
			prefs: map[prefsKey]domain.Preferences{{acme.ID, alice.ID}: tabbed}},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
			{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleGuest},
			{bob.ID, acme.ID}:   {WorkspaceRole: shared.RoleMember},
			{bob.ID, beta.ID}:   {WorkspaceRole: shared.RoleMember},
		}}}
	f.get = app.NewGetWorkspacePreferences(f.workspaces, f.auth)
	f.update = app.NewUpdateWorkspacePreferences(f.workspaces, f.auth, f.tx, clockAt{at: clockNow})
	return f
}

// The caller's own settings in the workspace named, read without a
// transaction after the decision: alice's, and the defaults for bob, who has
// changed none, in either workspace.
func TestGetWorkspacePreferencesReadsTheCallersOwn(t *testing.T) {
	tests := []struct {
		user app.AccountState
		w    domain.Workspace
		want domain.Preferences
	}{
		{alice, acme, tabbed},
		{bob, acme, domain.DefaultPreferences()},
		{bob, beta, domain.DefaultPreferences()},
	}
	for _, tt := range tests {
		f := newPrefs()
		got, err := f.get.Execute(as(tt.user), tt.w.Slug)
		if err != nil || got != tt.want {
			t.Errorf("%s, %s: Execute() = %+v, %v; want %+v", tt.user.Email, tt.w.Slug, got, err, tt.want)
		}
		want := []string{
			"WorkspaceBySlug " + tt.w.Slug + " outside tx",
			"Authorize " + tt.user.ID.String() + " workspace_preferences.read on " + tt.w.ID.String() + "/" + uuid.Nil().String() + " outside tx",
			"Preferences " + tt.w.ID.String() + " " + tt.user.ID.String() + " outside tx",
		}
		if !slices.Equal(f.log.calls, want) || f.tx.calls != 0 || len(f.workspaces.upserts) != 0 {
			t.Errorf("%s, %s: calls = %q, %d transactions, %d writes; want %q, none", tt.user.Email, tt.w.Slug, f.log.calls, f.tx.calls,
				len(f.workspaces.upserts), want)
		}
	}
}

// A workspace that is not there and one the caller cannot see are the same
// workspace.not_found, and the settings are not read; a failure is the
// answer, never the defaults.
func TestGetWorkspacePreferencesRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		set   func(f *prefsFixture)
		want  error
		calls int
	}{
		{"no such workspace", alice, "nothing", nil, domain.ErrNotFound, 1},
		{"not visible", carol, "acme", nil, domain.ErrNotFound, 2},
		{"alice cannot see beta", alice, "beta", nil, domain.ErrNotFound, 2},
		{"forbidden", bob, "acme", func(f *prefsFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), 2},
		{"the store failed", alice, "gamma", func(f *prefsFixture) { f.workspaces.slugErrs = map[string]error{"gamma": failure} }, failure, 1},
		{"the Authorizer failed", alice, "acme", func(f *prefsFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} },
			failure, 2},
		{"the read failed", alice, "acme", func(f *prefsFixture) { f.workspaces.prefsErr = failure }, failure, 3},
	}
	for _, tt := range tests {
		f := newPrefs()
		if tt.set != nil {
			tt.set(f)
		}
		if got, err := f.get.Execute(as(tt.user), tt.slug); !errors.Is(err, tt.want) || got != (domain.Preferences{}) {
			t.Errorf("%s: Execute() = %+v, %v; want no settings and %v", tt.name, got, err, tt.want)
		}
		if len(f.log.calls) != tt.calls {
			t.Errorf("%s: calls = %q, want %d", tt.name, f.log.calls, tt.calls)
		}
	}
	f := newPrefs()
	if _, err := f.get.Execute(context.Background(), "acme"); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and no call", err, f.log.calls)
	}
}

// UpdateWorkspacePreferences locks the workspace FOR SHARE, then decides,
// then writes the caller's row, in one transaction (M3 design 3.6): alice's
// change keeps what she set before, bob's first one starts from the
// defaults. Each write names a new row id.
func TestUpdateWorkspacePreferencesLocksThenDecidesThenWrites(t *testing.T) {
	tests := []struct {
		user app.AccountState
		w    domain.Workspace
		p    domain.PreferencesPatch
		call string
		want domain.Preferences
	}{
		{alice, acme, domain.PreferencesPatch{NavigationProjectLimit: ptr(0)}, `mode=<nil> limit="0"`, prefs("TABBED", 0)},
		{bob, acme, domain.PreferencesPatch{NavigationControl: ptr("TABBED")}, `mode="TABBED" limit=<nil>`, prefs("TABBED", 10)},
		{bob, beta, domain.PreferencesPatch{}, `mode=<nil> limit=<nil>`, domain.DefaultPreferences()},
	}
	for _, tt := range tests {
		f := newPrefs()
		got, err := f.update.Execute(as(tt.user), tt.w.Slug, tt.p)
		if err != nil || got != tt.want {
			t.Errorf("%s, %s: Execute() = %+v, %v; want %+v", tt.user.Email, tt.w.Slug, got, err, tt.want)
		}
		want := append(lockedDecision(tt.user, tt.w, "ShareWorkspaceBySlug", domain.ActionPreferencesUpdate),
			"UpsertPreferences "+tt.w.ID.String()+" "+tt.user.ID.String()+" "+tt.call+" at "+clockNow.Format(time.RFC3339Nano))
		if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
			t.Errorf("%s, %s: calls = %q in %d transactions, want %q in one", tt.user.Email, tt.w.Slug, f.log.calls, f.tx.calls, want)
		}
	}
	f := newPrefs()
	for range 2 {
		if _, err := f.update.Execute(as(bob), "beta", domain.PreferencesPatch{}); err != nil {
			t.Fatal(err)
		}
	}
	ids := f.workspaces.upserts
	if len(ids) != 2 {
		t.Fatalf("%d writes, want 2", len(ids))
	}
	if row := f.workspaces.prefIDs[prefsKey{beta.ID, bob.ID}]; ids[0].ID[6]>>4 != 7 || ids[1].ID[6]>>4 != 7 || ids[0].ID == ids[1].ID || row != ids[0].ID {
		t.Errorf("row ids %s, %s, the row inserted %s; want two different version 7 ids, the first the inserted row's", ids[0].ID, ids[1].ID, row)
	}
}

// Each refusal and failure is the answer, with nothing written: the values'
// check first, without a transaction; a workspace not there or not visible,
// workspace.not_found; a failed lock, decision or write, never a 404.
func TestUpdateWorkspacePreferencesRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	limit := domain.PreferencesPatch{NavigationProjectLimit: ptr(5)}
	shared403 := func(f *prefsFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} }
	tests := []struct {
		name   string
		user   app.AccountState
		slug   string
		p      domain.PreferencesPatch
		set    func(f *prefsFixture)
		want   error
		calls  []string
		inTxes int
	}{
		{"invalid values", alice, "acme", domain.PreferencesPatch{NavigationProjectLimit: ptr(-1)}, nil,
			shared.Invalid(shared.FieldError{Field: "navigation_project_limit", Code: shared.FieldOutOfRange, Message: "must be between 0 and 2147483647"}), nil, 0},
		{"no such workspace", alice, "nothing", limit, nil, domain.ErrNotFound, []string{"ShareWorkspaceBySlug nothing"}, 1},
		{"not visible", carol, "acme", limit, nil, domain.ErrNotFound, lockedDecision(carol, acme, "ShareWorkspaceBySlug", domain.ActionPreferencesUpdate), 1},
		{"forbidden", bob, "acme", limit, shared403, shared.Forbidden(), lockedDecision(bob, acme, "ShareWorkspaceBySlug", domain.ActionPreferencesUpdate), 1},
		{"the lock failed", alice, "acme", limit, func(f *prefsFixture) { f.workspaces.lockErrs = map[string]error{"acme": failure} }, failure,
			[]string{"ShareWorkspaceBySlug acme"}, 1},
		{"the write failed", alice, "acme", limit, func(f *prefsFixture) { f.workspaces.prefsErr = failure }, failure,
			append(lockedDecision(alice, acme, "ShareWorkspaceBySlug", domain.ActionPreferencesUpdate),
				"UpsertPreferences "+acme.ID.String()+" "+alice.ID.String()+` mode=<nil> limit="5" at `+clockNow.Format(time.RFC3339Nano)), 1},
	}
	for _, tt := range tests {
		f := newPrefs()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := f.update.Execute(as(tt.user), tt.slug, tt.p)
		if !errors.Is(err, tt.want) || got != (domain.Preferences{}) {
			t.Errorf("%s: Execute() = %+v, %v; want no settings and %v", tt.name, got, err, tt.want)
		}
		if tt.want == failure && errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: Execute() = %v, which is also workspace.not_found", tt.name, err)
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != tt.inTxes {
			t.Errorf("%s: calls = %q in %d transactions, want %q in %d", tt.name, f.log.calls, f.tx.calls, tt.calls, tt.inTxes)
		}
	}
	f := newPrefs()
	if _, err := f.update.Execute(context.Background(), "acme", limit); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and nothing", err, f.log.calls)
	}
}

// prefs is the settings mode and limit.
func prefs(mode string, limit int) domain.Preferences {
	return domain.Preferences{NavigationControl: mode, NavigationProjectLimit: limit}
}
