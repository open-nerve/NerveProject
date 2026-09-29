package app_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	alice = app.AccountState{ID: uuid.NewV7(), Email: "alice@corp.com", Active: true}
	bob   = app.AccountState{ID: uuid.NewV7(), Email: "bob@corp.com", Active: true}
	carol = app.AccountState{ID: uuid.NewV7(), Email: "carol@corp.com", Active: false}
)

type createFixture struct {
	log        *callLog
	tx         *fakeTx
	accounts   *fakeAccounts
	workspaces *fakeWorkspaces
	logs       *strings.Builder // the use case's log, as text
}

func newCreate(enabled bool) (*app.CreateWorkspace, *createFixture) {
	log := &callLog{}
	f := &createFixture{log: log, tx: &fakeTx{}, accounts: &fakeAccounts{log: log, accounts: []app.AccountState{alice, bob, carol}},
		workspaces: &fakeWorkspaces{log: log}, logs: &strings.Builder{}}
	return app.NewCreateWorkspace(app.CreateWorkspaceDeps{
		Accounts: f.accounts, Workspaces: f.workspaces, Tx: f.tx, Clock: clocktest.At(now),
		Logger: slog.New(slog.NewTextHandler(f.logs, nil)), Enabled: enabled,
	}), f
}

// createdLog is the one line a creation logs: the workspace, its admin, and
// whether the API or the command line created it.
func createdLog(workspace uuid.UUID, admin app.AccountState, by string) string {
	return `level=INFO msg="workspace created" workspace_id=` + workspace.String() + " user_id=" + admin.ID.String() + " by=" + by + "\n"
}

// logged is the log without each line's time.
func logged(f *createFixture) string {
	var out strings.Builder
	for line := range strings.Lines(f.logs.String()) {
		_, rest, _ := strings.Cut(line, " ")
		out.WriteString(rest)
	}
	return out.String()
}

func as(user app.AccountState) context.Context {
	return shared.WithActor(context.Background(), shared.Actor{UserID: user.ID})
}

// The caller's account row is locked first, in the transaction, then the
// workspace and the caller's admin membership are written, all with the
// clock's time (M3 design 3.6, its global order and convention 1): for two
// callers, each with his own account. The answer is the workspace as stored,
// the caller its admin and only member; one line is logged.
func TestExecuteCreatesTheWorkspaceWithTheCallerAsAdmin(t *testing.T) {
	for _, user := range []app.AccountState{alice, bob} {
		uc, f := newCreate(true)
		size := "11-50"

		got, err := uc.Execute(as(user), domain.NewWorkspace{Name: "Acme", Slug: "acme", OrganizationSize: &size})

		if err != nil {
			t.Fatalf("%s: Execute() = %v", user.Email, err)
		}
		at := now.Format(time.RFC3339Nano)
		want := []string{
			"ShareAccount " + user.ID.String(),
			"CreateWorkspace " + got.ID.String() + ` "Acme" acme 11-50 UTC by ` + user.ID.String() + " at " + at,
			"CreateMember " + user.ID.String() + " in " + got.ID.String() + " as 20 by " + user.ID.String() + " at " + at,
		}
		if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
			t.Errorf("%s: calls = %q in %d transactions, want %q in one", user.Email, f.log.calls, f.tx.calls, want)
		}
		if got.ID == uuid.Nil() || got.Name != "Acme" || got.Slug != "acme" || *got.OrganizationSize != "11-50" || got.Timezone != "UTC" ||
			got.Role != shared.RoleAdmin || got.TotalMembers != 1 || !got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(now) {
			t.Errorf("%s: Execute() = %+v, want acme as stored, with the caller its admin and only member", user.Email, got)
		}
		if m := f.workspaces.members; len(m) != 1 || m[0].ID == uuid.Nil() || m[0].ID == got.ID {
			t.Errorf("%s: members = %+v, want one with its own id", user.Email, m)
		}
		if line, want := logged(f), createdLog(got.ID, user, "api"); line != want {
			t.Errorf("%s: log = %q, want %q", user.Email, line, want)
		}
	}
}

// A time zone given is stored as given.
func TestExecuteStoresTheTimeZoneGiven(t *testing.T) {
	uc, f := newCreate(true)
	zone := "Asia/Shanghai"
	got, err := uc.Execute(as(alice), domain.NewWorkspace{Name: "研发部", Slug: "rd", Timezone: &zone})
	if err != nil || got.Timezone != zone || got.OrganizationSize != nil {
		t.Fatalf("Execute() = %+v, %v; want the time zone %s and no size", got, err, zone)
	}
	if call := f.log.calls[1]; call != "CreateWorkspace "+got.ID.String()+` "研发部" rd <nil> Asia/Shanghai by `+alice.ID.String()+" at "+now.Format(time.RFC3339Nano) {
		t.Errorf("CreateWorkspace call = %q", call)
	}
}

// While creation is off, the use case answers workspace.creation_disabled
// before it looks at anything, the values and the caller included (Plane
// views/workspace/base.py:83-96); in the API, authentication and the body's
// shape come first.
func TestExecuteWhileCreationIsDisabled(t *testing.T) {
	uc, f := newCreate(false)
	for _, w := range []domain.NewWorkspace{{Name: "Acme", Slug: "acme"}, {Name: "", Slug: "API"}} {
		if _, err := uc.Execute(as(alice), w); !errors.Is(err, domain.ErrCreationDisabled) {
			t.Errorf("Execute(%+v) = %v, want workspace.creation_disabled", w, err)
		}
	}
	if _, err := uc.Execute(context.Background(), domain.NewWorkspace{Name: "Acme", Slug: "acme"}); !errors.Is(err, domain.ErrCreationDisabled) {
		t.Errorf("Execute() without an actor = %v, want workspace.creation_disabled", err)
	}
	if len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("calls = %q in %d transactions, want none", f.log.calls, f.tx.calls)
	}
}

// Invalid values are 422 before any lock or write.
func TestExecuteRefusesInvalidValuesBeforeTheTransaction(t *testing.T) {
	uc, f := newCreate(true)
	_, err := uc.Execute(as(alice), domain.NewWorkspace{Name: "Acme", Slug: "api"})
	var se *shared.Error
	if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || se.Fields[0].Field != "slug" {
		t.Errorf("Execute() = %v, want validation_failed on slug", err)
	}
	if len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("calls = %q in %d transactions, want none", f.log.calls, f.tx.calls)
	}
}

// An account that is deactivated, as read under the lock, or gone, gets 401
// and writes nothing (M3 design 3.6 convention 6).
func TestExecuteRefusesAnAccountNoLongerActive(t *testing.T) {
	gone := app.AccountState{ID: uuid.NewV7()}
	for _, user := range []app.AccountState{carol, gone} {
		uc, f := newCreate(true)
		_, err := uc.Execute(as(user), domain.NewWorkspace{Name: "Acme", Slug: "acme"})
		if !errors.Is(err, shared.Unauthenticated()) {
			t.Errorf("%s: Execute() = %v, want 401 unauthorized", user.ID, err)
		}
		if want := []string{"ShareAccount " + user.ID.String()}; !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", user.ID, f.log.calls, want)
		}
		if f.logs.Len() != 0 {
			t.Errorf("%s: log = %q, want nothing", user.ID, f.logs)
		}
	}
}

// A taken slug is the store's answer, and nothing more is written.
func TestExecuteSlugTaken(t *testing.T) {
	uc, f := newCreate(true)
	f.workspaces.createErr = domain.ErrSlugTaken
	if _, err := uc.Execute(as(alice), domain.NewWorkspace{Name: "Acme", Slug: "acme"}); !errors.Is(err, domain.ErrSlugTaken) {
		t.Errorf("Execute() = %v, want workspace.slug_taken", err)
	}
	if len(f.log.calls) != 2 || len(f.workspaces.members) != 0 || f.logs.Len() != 0 {
		t.Errorf("calls = %q, log %q; want the lock and the insert only, and no log", f.log.calls, f.logs)
	}
}

// The lock's failure is the use case's, before any write.
func TestExecuteReturnsTheLocksError(t *testing.T) {
	uc, f := newCreate(true)
	failure := errors.New("connection reset")
	f.accounts.err = failure
	if _, err := uc.Execute(as(alice), domain.NewWorkspace{Name: "Acme", Slug: "acme"}); !errors.Is(err, failure) {
		t.Errorf("Execute() = %v, want %v", err, failure)
	}
	if len(f.log.calls) != 1 {
		t.Errorf("calls = %q, want the lock only", f.log.calls)
	}
}

// The admin membership's failure is the use case's, from inside the
// transaction, after the lock and both inserts: no workspace is answered and
// nothing is logged.
func TestExecuteReturnsTheMembershipsError(t *testing.T) {
	uc, f := newCreate(true)
	failure := errors.New("connection reset")
	f.workspaces.memberErr = failure
	got, err := uc.Execute(as(alice), domain.NewWorkspace{Name: "Acme", Slug: "acme"})
	if !errors.Is(err, failure) || got != (domain.Workspace{}) {
		t.Errorf("Execute() = %+v, %v; want no workspace and %v", got, err, failure)
	}
	if len(f.log.calls) != 3 || !strings.HasPrefix(f.log.calls[2], "CreateMember ") || f.tx.calls != 1 {
		t.Errorf("calls = %q in %d transactions, want the lock and both inserts in one", f.log.calls, f.tx.calls)
	}
	if f.logs.Len() != 0 {
		t.Errorf("log = %q, want nothing", f.logs)
	}
}

// Without a caller, Execute is 401 and calls nothing: the use case refuses
// on its own, whoever calls it.
func TestExecuteWithoutACaller(t *testing.T) {
	uc, f := newCreate(true)
	if _, err := uc.Execute(context.Background(), domain.NewWorkspace{Name: "Acme", Slug: "acme"}); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("Execute() without an actor = %v, want 401 unauthorized", err)
	}
	if len(f.log.calls) != 0 {
		t.Errorf("calls = %q, want none", f.log.calls)
	}
}

// The server's administrator creates a workspace for the account of an
// address, normalized, whatever workspace.creation_enabled says (M3 design
// 3.11).
func TestExecuteForAdminCreatesForTheAccountOfTheAddress(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		uc, f := newCreate(enabled)

		got, err := uc.ExecuteForAdmin(context.Background(), "  Bob@Corp.COM ", domain.NewWorkspace{Name: "Acme", Slug: "acme"})

		if err != nil || got.Role != shared.RoleAdmin || got.TotalMembers != 1 {
			t.Fatalf("enabled %v: ExecuteForAdmin() = %+v, %v", enabled, got, err)
		}
		at := now.Format(time.RFC3339Nano)
		want := []string{
			"ShareAccountByEmail bob@corp.com",
			"CreateWorkspace " + got.ID.String() + ` "Acme" acme <nil> UTC by ` + bob.ID.String() + " at " + at,
			"CreateMember " + bob.ID.String() + " in " + got.ID.String() + " as 20 by " + bob.ID.String() + " at " + at,
		}
		if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
			t.Errorf("enabled %v: calls = %q in %d transactions, want %q in one", enabled, f.log.calls, f.tx.calls, want)
		}
		if line, want := logged(f), createdLog(got.ID, bob, "cli"); line != want {
			t.Errorf("enabled %v: log = %q, want %q", enabled, line, want)
		}
	}
}

// No account, a deactivated one read under the lock, invalid values: the
// command's errors, and nothing written.
func TestExecuteForAdminRefuses(t *testing.T) {
	tests := []struct {
		name  string
		email string
		w     domain.NewWorkspace
		want  error
		calls int
	}{
		{"no account", "dave@corp.com", domain.NewWorkspace{Name: "Acme", Slug: "acme"}, domain.ErrAccountNotFound, 1},
		{"a deactivated account", "carol@corp.com", domain.NewWorkspace{Name: "Acme", Slug: "acme"}, domain.ErrAccountDeactivated, 1},
		{"a reserved slug", "alice@corp.com", domain.NewWorkspace{Name: "Acme", Slug: "healthz"}, shared.Invalid(), 0},
	}
	for _, tt := range tests {
		uc, f := newCreate(true)
		if _, err := uc.ExecuteForAdmin(context.Background(), tt.email, tt.w); !errors.Is(err, tt.want) {
			t.Errorf("%s: ExecuteForAdmin() = %v, want %v", tt.name, err, tt.want)
		}
		if len(f.log.calls) != tt.calls || len(f.workspaces.members) != 0 || f.logs.Len() != 0 {
			t.Errorf("%s: calls = %q, log %q; want %d, no write and no log", tt.name, f.log.calls, f.logs, tt.calls)
		}
	}
}
