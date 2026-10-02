package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// tableRows is every row of table as JSON, one a line by id: a row whose id
// is in written without the columns cols, which the write under test sets.
// Compared before and after a write, it shows that the write changed those
// columns of those rows alone; the caller checks the columns themselves.
func tableRows(t *testing.T, pool *pgxpool.Pool, table string, written []uuid.UUID, cols ...string) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(CASE WHEN r.id = ANY ($1) THEN (to_jsonb(r) - $2::text[])::text
		ELSE to_jsonb(r)::text END, E'\n' ORDER BY r.id), '') FROM `+table+` r`, written, cols).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// stamp is what an ending writes of a row, as text: whether a membership
// (workspace_members) is active, or when an invitation
// (workspace_member_invites) was deleted, then its updated_at and
// updated_by_id.
func stamp(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) string {
	t.Helper()
	state := "CASE WHEN is_active THEN 'active' ELSE 'ended' END"
	if table == "workspace_member_invites" {
		state = "coalesce('deleted at ' || deleted_at::text, 'undeleted')"
	}
	var got string
	if err := pool.QueryRow(context.Background(), "SELECT "+state+" || ' at ' || updated_at::text || ' by ' || updated_by_id::text FROM "+
		table+" WHERE id = $1", id).Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

// at is tm as PostgreSQL shows a timestamptz, as stamp does.
func at(t *testing.T, pool *pgxpool.Pool, tm time.Time) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), "SELECT $1::timestamptz::text", tm).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// EndMember ends the user's active, undeleted membership of the workspace,
// by the account and at the time given, and changes nothing else (M3
// design 3.6, 4.3). bob's membership of acme was last written by himself.
// His deleted membership of acme, stored before his live one or after it,
// his membership of beta and carol's of acme keep every column, and his
// ended one its other columns. A pair with no active, undeleted membership
// is an error that is not app.ErrNotFound, and changes nothing: carol's in
// beta, of which there is none, bob's in gamma, deleted, and bob's in
// acme, ended by then, whose ender and moment stay.
func TestEndMember(t *testing.T) {
	for _, deletedFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("the deleted membership stored first %v", deletedFirst), func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
			acme, beta, gamma := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice),
				newWorkspace(t, s, "Gamma", "gamma", alice)
			deleted := func(workspace uuid.UUID) {
				exec(t, pool, `INSERT INTO workspace_members (id, workspace_id, member_id, role, created_by_id, updated_by_id, created_at,
					updated_at, deleted_at) VALUES ($1, $2, $3, 20, $3, $3, $4, $4, $4)`, uuid.NewV7(), workspace, bob, now.Add(-time.Hour))
			}
			if deletedFirst {
				deleted(acme.ID)
			}
			bobIn := joinAt(t, s, acme.ID, bob, shared.RoleMember, now)
			if !deletedFirst {
				deleted(acme.ID)
			}
			joinAt(t, s, acme.ID, carol, shared.RoleGuest, now)
			joinAt(t, s, beta.ID, bob, shared.RoleAdmin, now)
			deleted(gamma.ID)
			if got, want := stamp(t, pool, "workspace_members", bobIn.ID), fmt.Sprintf("active at %s by %s", at(t, pool, now), bob); got != want {
				t.Fatalf("bob's membership of acme before: %s, want %s", got, want)
			}
			before := tableRows(t, pool, "workspace_members", []uuid.UUID{bobIn.ID}, "is_active", "updated_at", "updated_by_id")
			later := now.Add(time.Hour)

			if err := s.EndMember(context.Background(), acme.ID, bob, alice, later); err != nil {
				t.Fatal(err)
			}

			if got, want := stamp(t, pool, "workspace_members", bobIn.ID), fmt.Sprintf("ended at %s by %s", at(t, pool, later), alice); got != want {
				t.Errorf("bob's membership of acme: %s, want %s", got, want)
			}
			if after := tableRows(t, pool, "workspace_members", []uuid.UUID{bobIn.ID}, "is_active", "updated_at", "updated_by_id"); after != before {
				t.Errorf("the memberships, bob's in acme without the columns written:\n%s\nwant them as they were:\n%s", after, before)
			}
			before = tableRows(t, pool, "workspace_members", nil)
			for _, pair := range []struct {
				name            string
				workspace, user uuid.UUID
			}{{"carol in beta", beta.ID, carol}, {"bob in gamma", gamma.ID, bob}, {"bob in acme, ended", acme.ID, bob}} {
				if err := s.EndMember(context.Background(), pair.workspace, pair.user, alice, later.Add(time.Hour)); err == nil ||
					errors.Is(err, app.ErrNotFound) {
					t.Errorf("EndMember() of %s = %v, want an error that is not app.ErrNotFound", pair.name, err)
				}
			}
			if after := tableRows(t, pool, "workspace_members", nil); after != before {
				t.Errorf("the memberships after ending none:\n%s\nwant them as they were:\n%s", after, before)
			}
		})
	}
}

// DeletePendingInvitations soft-deletes the workspace's pending invitation
// to the address, by the account and at the time given, and changes nothing
// else (M3 design 3.8). In acme bob@corp.com has a pending invitation,
// written by alice, and two earlier ones, one accepted, one deleted while
// pending, each deleted then, stored before it or after; carol@corp.com has
// a pending one; beta has bob's pending one, gamma his declined one. Ending
// in acme deletes his pending one there; ending in gamma leaves the
// declined one; every other row keeps every column, and the earlier ones
// their moments.
func TestDeletePendingInvitations(t *testing.T) {
	for _, earlierFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("the earlier invitation stored first %v", earlierFirst), func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
			acme, beta, gamma := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice),
				newWorkspace(t, s, "Gamma", "gamma", alice)
			earlier := func() {
				exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, accepted, responded_at, created_by_id,
					updated_by_id, created_at, updated_at, deleted_at) VALUES ($1, $2, 'bob@corp.com', 15, true, $4, $3, $3, $4, $4, $4)`,
					uuid.NewV7(), acme.ID, alice, now.Add(-time.Hour))
				exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at,
					updated_at, deleted_at) VALUES ($1, $2, 'bob@corp.com', 5, $3, $3, $4, $4, $4)`, uuid.NewV7(), acme.ID, alice, now.Add(-time.Hour))
			}
			if earlierFirst {
				earlier()
			}
			pending := invite(t, s, acme.ID, "bob@corp.com", shared.RoleGuest, alice)
			if !earlierFirst {
				earlier()
			}
			invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
			invite(t, s, beta.ID, "bob@corp.com", shared.RoleMember, alice)
			declined := invite(t, s, gamma.ID, "bob@corp.com", shared.RoleMember, alice)
			exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $2 WHERE id = $1", declined.ID, now)
			if got, want := stamp(t, pool, "workspace_member_invites", pending.ID), fmt.Sprintf("undeleted at %s by %s", at(t, pool, now), alice); got != want {
				t.Fatalf("bob's pending invitation to acme before: %s, want %s", got, want)
			}
			before := tableRows(t, pool, "workspace_member_invites", []uuid.UUID{pending.ID}, "deleted_at", "updated_at", "updated_by_id")
			later := now.Add(time.Hour)

			for _, w := range []uuid.UUID{acme.ID, gamma.ID} {
				if err := s.DeletePendingInvitations(context.Background(), w, "bob@corp.com", bob, later); err != nil {
					t.Fatal(err)
				}
			}

			if got, want := stamp(t, pool, "workspace_member_invites", pending.ID), fmt.Sprintf("deleted at %[1]s at %[1]s by %[2]s",
				at(t, pool, later), bob); got != want {
				t.Errorf("bob's pending invitation to acme: %s, want %s", got, want)
			}
			after := tableRows(t, pool, "workspace_member_invites", []uuid.UUID{pending.ID}, "deleted_at", "updated_at", "updated_by_id")
			if after != before {
				t.Errorf("the invitations, bob's pending one to acme without the columns written:\n%s\nwant them as they were:\n%s", after, before)
			}
		})
	}
}

// HasOtherAdmin reports whether the workspace has an active admin other
// than the user (M3 design 3.7 rule 1): alice is the admin of each
// workspace; bob, in each case, stands to it as the case says. Only an
// active, undeleted admin's membership of that workspace is another admin.
// The user asked about is alice, but where the case asks about bob, an
// active member: alice is another admin then, the user left out by who he
// is, not as one admin of a count.
func TestHasOtherAdmin(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	set := func(sql string) func(uuid.UUID) {
		return func(w uuid.UUID) { exec(t, pool, sql, w, bob) }
	}
	tests := []struct {
		name string
		bob  func(workspace uuid.UUID)
		user uuid.UUID // asked about
		want bool
	}{
		{"an active admin", func(w uuid.UUID) { join(t, s, w, bob, shared.RoleAdmin) }, alice, true},
		{"an active member", func(w uuid.UUID) { join(t, s, w, bob, shared.RoleMember) }, alice, false},
		{"an active member, asked about", func(w uuid.UUID) { join(t, s, w, bob, shared.RoleMember) }, bob, true},
		{"an admin whose membership ended", func(w uuid.UUID) {
			join(t, s, w, bob, shared.RoleAdmin)
			set("UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2")(w)
		}, alice, false},
		{"an admin whose membership is deleted", func(w uuid.UUID) {
			join(t, s, w, bob, shared.RoleAdmin)
			set("UPDATE workspace_members SET deleted_at = now() WHERE workspace_id = $1 AND member_id = $2")(w)
		}, alice, false},
		{"an admin of another workspace", func(uuid.UUID) { join(t, s, newWorkspace(t, s, "Other", "other", alice).ID, bob, shared.RoleAdmin) },
			alice, false},
		{"none: alice is alone", func(uuid.UUID) {}, alice, false},
	}
	for i, tt := range tests {
		w := newWorkspace(t, s, "Acme", fmt.Sprintf("acme-%d", i), alice)
		tt.bob(w.ID)
		if got, err := s.HasOtherAdmin(context.Background(), w.ID, tt.user); err != nil || got != tt.want {
			t.Errorf("bob %s: HasOtherAdmin() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}
