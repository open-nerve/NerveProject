package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// columns are the row id of table, each column's value as JSON text.
func columns(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) map[string]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT key, value::text FROM "+table+" r, jsonb_each(to_jsonb(r)) WHERE r.id = $1", id)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			t.Fatal(err)
		}
		out[k] = v
	}
	if err := rows.Err(); err != nil || len(out) == 0 {
		t.Fatalf("the row %s of %s: %v, %d columns", id, table, err, len(out))
	}
	return out
}

// changed is before with the columns of change changed, as JSON text.
func changed(before map[string]string, change map[string]string) map[string]string {
	out := maps.Clone(before)
	maps.Copy(out, change)
	return out
}

// LockProject reads the undeleted project's workspace and whether it is
// archived, and holds it FOR NO KEY UPDATE until the transaction ends: a
// FOR SHARE of it waits, a foreign key's FOR KEY SHARE does not, and no
// other project is held. A deleted project, and no project, are not found.
func TestLockProject(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops, old := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, beta, "Ops", "OPS", alice), newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", old, now)
	for _, tt := range []struct {
		id    uuid.UUID
		want  app.LockedProject
		found bool
	}{{web, app.LockedProject{WorkspaceID: acme}, true}, {ops, app.LockedProject{WorkspaceID: beta, Archived: true}, true},
		{old, app.LockedProject{}, false}, {uuid.NewV7(), app.LockedProject{}, false}} {
		if got, found, err := s.LockProject(context.Background(), tt.id); err != nil || found != tt.found || got != tt.want {
			t.Errorf("LockProject(%s) = %+v, %v, %v; want %+v, %v", tt.id, got, found, err, tt.want, tt.found)
		}
	}
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if _, _, err := s.LockProject(ctx, web); err != nil {
			return err
		}
		if !waits(t, pool, web, "FOR SHARE") || waits(t, pool, web, "FOR KEY SHARE") || waits(t, pool, ops, "FOR SHARE") {
			t.Error("while the lock is held: want a FOR SHARE of web to wait, and neither its FOR KEY SHARE nor a FOR SHARE of ops")
		}
		return nil
	})
	if err != nil || waits(t, pool, web, "FOR SHARE") {
		t.Errorf("after the transaction: %v; want web free", err)
	}
}

// A project deleted while LockProject waited for its lock is not found:
// after the wait, Postgres evaluates deleted_at IS NULL again on the row's
// newest version (M3 design 3.6 convention 2).
func TestTheProjectLockSeesADeletionItWaitedFor(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, s, newWorkspace(t, pool, "acme"), "Web", "WEB", alice)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "UPDATE projects SET deleted_at = $2 WHERE id = $1", web, now); err != nil {
		t.Fatal(err)
	}
	type answer struct {
		found bool
		err   error
	}
	done := make(chan answer, 1)
	go func() {
		var a answer
		a.err = postgres.NewTxManager(pool, 5*time.Second).WithinTx(ctx, func(ctx context.Context) error {
			var err error
			_, a.found, err = s.LockProject(ctx, web)
			return err
		})
		done <- a
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 5*time.Second)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-done:
		if a.err != nil || a.found {
			t.Errorf("LockProject after the deletion committed = found %v, %v; want not found", a.found, a.err)
		}
	case <-ctx.Done():
		t.Fatal("LockProject did not end within 10s")
	}
}

// ProjectWorkspace reads the undeleted project's workspace, archived or
// not, and takes no lock: while the read's transaction is open, a FOR
// UPDATE of the project does not wait. A deleted project, and no project,
// are not found. Every write on a project reads it first, to lock the
// workspace before the project (M3 design 3.6 convention 2).
func TestProjectWorkspace(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops, old := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, beta, "Ops", "OPS", alice), newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", old, now)
	for _, tt := range []struct {
		id, workspace uuid.UUID
		found         bool
	}{{web, acme, true}, {ops, beta, true}, {old, uuid.UUID{}, false}, {uuid.NewV7(), uuid.UUID{}, false}} {
		if got, found, err := s.ProjectWorkspace(context.Background(), tt.id); err != nil || found != tt.found || got != tt.workspace {
			t.Errorf("ProjectWorkspace(%s) = %s, %v, %v; want %s, %v", tt.id, got, found, err, tt.workspace, tt.found)
		}
	}
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if _, _, err := s.ProjectWorkspace(ctx, web); err != nil {
			return err
		}
		if waits(t, pool, web, "FOR UPDATE") {
			t.Error("while the read's transaction is open, a FOR UPDATE of web waits; want no lock taken")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// UpdateProject changes exactly the fields the patch gives, and the audit
// columns to the moment and the account given; every other column of the
// project keeps its value, and every other project keeps every column: one
// of the same workspace, archived, one of another. A patch that gives
// nothing changes the audit columns alone; the lead and the default
// assignee change only when their flags are set, to none too.
func TestUpdateProject(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web := newProject(t, s, acme, "Web", "WEB", alice)
	// Every field is left out once while it holds another value than its default and than the other projects': the
	// cycles switch, which the third change sets, starts on.
	exec(t, pool, "UPDATE projects SET project_lead_id = $2, default_assignee_id = $3, cycle_view = true WHERE id = $1", web, alice, bob)
	ops := newProject(t, s, acme, "Ops", "OPS", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2, project_lead_id = $3 WHERE id = $1", ops, now, bob)
	newProject(t, s, beta, "Web", "WEB", alice)
	later := now.Add(time.Hour)
	update := func(p domain.ProjectPatch, by uuid.UUID, at time.Time) {
		t.Helper()
		if err := s.UpdateProject(context.Background(), web, p, by, at); err != nil {
			t.Fatal(err)
		}
	}
	others := tableRows(t, pool, "projects", web)
	audit := func(by uuid.UUID, at time.Time) map[string]string {
		return map[string]string{"updated_by_id": `"` + by.String() + `"`, "updated_at": `"` + at.Format("2006-01-02T15:04:05.999999") + `+00:00"`}
	}

	before := columns(t, pool, "projects", web)
	update(domain.ProjectPatch{}, carol, later)
	if got, want := columns(t, pool, "projects", web), changed(before, audit(carol, later)); !maps.Equal(got, want) {
		t.Errorf("an empty patch: %v\nwant %v", got, want)
	}

	before = columns(t, pool, "projects", web)
	update(domain.ProjectPatch{Name: ptr("研发 Web"), Description: ptr("The app"), Identifier: ptr("WEBÇ"), Network: ptr(domain.NetworkPrivate),
		SetLead: true, LeadID: &carol, CycleView: ptr(true), ModuleView: ptr(true), IssueViewsView: ptr(true), IntakeView: ptr(true),
		GuestViewAllFeatures: ptr(true), ArchiveIn: ptr(12), LogoProps: &domain.LogoProps{InUse: ptr("icon"), Icon: &domain.Icon{Name: ptr("home")}},
		Timezone: ptr("Asia/Shanghai")}, bob, later.Add(time.Hour))
	want := changed(before, audit(bob, later.Add(time.Hour)))
	maps.Copy(want, map[string]string{"name": `"研发 Web"`, "description": `"The app"`, "identifier": `"WEBÇ"`, "network": "0",
		"project_lead_id": `"` + carol.String() + `"`, "cycle_view": "true", "module_view": "true", "issue_views_view": "true",
		"intake_view": "true", "guest_view_all_features": "true", "archive_in": "12", "logo_props": `{"icon": {"name": "home"}, "in_use": "icon"}`,
		"timezone": `"Asia/Shanghai"`})
	if got := columns(t, pool, "projects", web); !maps.Equal(got, want) {
		t.Errorf("every field: %v\nwant %v", got, want)
	}

	before = columns(t, pool, "projects", web)
	update(domain.ProjectPatch{SetDefaultAssignee: true, CycleView: ptr(false)}, alice, later)
	if got, want := columns(t, pool, "projects", web), changed(before, changed(audit(alice, later),
		map[string]string{"default_assignee_id": "null", "cycle_view": "false"})); !maps.Equal(got, want) {
		t.Errorf("the default assignee cleared: %v\nwant %v", got, want)
	}
	before = columns(t, pool, "projects", web)
	update(domain.ProjectPatch{SetLead: true}, alice, later)
	if got, want := columns(t, pool, "projects", web), changed(before, changed(audit(alice, later),
		map[string]string{"project_lead_id": "null"})); !maps.Equal(got, want) {
		t.Errorf("the lead cleared: %v\nwant %v", got, want)
	}

	// The five switches, which the steps above give together and keep together, each given alone: on while every other
	// is off, it changes its own column alone; then left out with the others, each keeps its own value, not another's.
	switches := []struct {
		column string
		set    func(p *domain.ProjectPatch, on *bool)
	}{
		{"cycle_view", func(p *domain.ProjectPatch, on *bool) { p.CycleView = on }},
		{"module_view", func(p *domain.ProjectPatch, on *bool) { p.ModuleView = on }},
		{"issue_views_view", func(p *domain.ProjectPatch, on *bool) { p.IssueViewsView = on }},
		{"intake_view", func(p *domain.ProjectPatch, on *bool) { p.IntakeView = on }},
		{"guest_view_all_features", func(p *domain.ProjectPatch, on *bool) { p.GuestViewAllFeatures = on }},
	}
	var off domain.ProjectPatch
	for _, s := range switches {
		s.set(&off, ptr(false))
	}
	for _, s := range switches {
		update(off, carol, later)
		var alone domain.ProjectPatch
		s.set(&alone, ptr(true))
		before = columns(t, pool, "projects", web)
		update(alone, bob, later)
		if got, want := columns(t, pool, "projects", web), changed(before, changed(audit(bob, later),
			map[string]string{s.column: "true"})); !maps.Equal(got, want) {
			t.Errorf("%s alone: %v\nwant %v", s.column, got, want)
		}
		before = columns(t, pool, "projects", web)
		update(domain.ProjectPatch{}, alice, later)
		if got, want := columns(t, pool, "projects", web), changed(before, audit(alice, later)); !maps.Equal(got, want) {
			t.Errorf("%s on, the others off, none given: %v\nwant %v", s.column, got, want)
		}
	}
	if after := tableRows(t, pool, "projects", web); after != others {
		t.Errorf("the other projects:\n%s\nwant\n%s", after, others)
	}
}

// An identifier or a name is taken by another undeleted project of the
// workspace only, archived or not: the project's own, another
// workspace's and a deleted project's are free (M3 design 3.19). A
// refused change changes nothing.
func TestUpdateProjectIdentifierOrNameTaken(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	newProject(t, s, beta, "Docs", "DOCS", alice)
	old := newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", old, now)
	update := func(name, identifier string) error {
		return s.UpdateProject(context.Background(), web, domain.ProjectPatch{Name: &name, Identifier: &identifier}, alice, now)
	}
	before := columns(t, pool, "projects", web)
	if err := update("Web", "OPS"); !errors.Is(err, domain.ErrIdentifierTaken) {
		t.Errorf("ops's identifier: %v, want project.identifier_taken", err)
	}
	if err := update("Ops", "WEB"); !errors.Is(err, domain.ErrNameTaken) {
		t.Errorf("ops's name: %v, want project.name_taken", err)
	}
	if got := columns(t, pool, "projects", web); !maps.Equal(got, before) {
		t.Errorf("after the refusals: %v, want %v", got, before)
	}
	for _, free := range [][2]string{{"Web", "WEB"}, {"Docs", "DOCS"}, {"Old", "OLD"}} {
		if err := update(free[0], free[1]); err != nil {
			t.Errorf("%s %s: %v, want it free", free[0], free[1], err)
		}
	}
}

// A change that breaks another constraint than the two unique keys is the
// store's error, not a taken identifier or name: the domain checks every
// value, so it is a bug (500).
func TestUpdateProjectBreakingAnotherConstraintIsInternal(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, s, newWorkspace(t, pool, "acme"), "Web", "WEB", alice)
	err := s.UpdateProject(context.Background(), web, domain.ProjectPatch{ArchiveIn: ptr(13)}, alice, now)
	var se *shared.Error
	var pgErr *pgconn.PgError
	if errors.As(err, &se) || !errors.As(err, &pgErr) || pgErr.ConstraintName != "projects_archive_in_check" {
		t.Errorf("archive_in 13: UpdateProject() = %v; want the violation of projects_archive_in_check, not a domain error", err)
	}
}
