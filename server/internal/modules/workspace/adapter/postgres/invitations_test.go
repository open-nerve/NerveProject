package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// invite stores a pending invitation of email to workspace with role, by
// inviter at now, and returns it as stored.
func invite(t *testing.T, s *postgresadapter.Store, workspace uuid.UUID, email string, role shared.Role, inviter uuid.UUID) domain.Invitation {
	t.Helper()
	got, err := s.CreateInvitations(context.Background(), []app.InvitationRow{
		{ID: uuid.NewV7(), WorkspaceID: workspace, Email: email, Role: role, CreatedBy: inviter, Now: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	return got[0]
}

// pendingEmails are the addresses of workspace's undeleted invitations, sorted.
func pendingEmails(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID) []string {
	t.Helper()
	var emails []string
	if err := pool.QueryRow(context.Background(), `SELECT coalesce(array_agg(email ORDER BY email), '{}') FROM workspace_member_invites
		WHERE workspace_id = $1 AND deleted_at IS NULL`, workspace).Scan(&emails); err != nil {
		t.Fatal(err)
	}
	return emails
}

// CreateInvitations stores each row with the use case's values and answers
// them as stored, in the order given: pending, the time the clock's, the
// inviter in both audit columns. Two workspaces, two inviters.
func TestCreateInvitationsStoresTheRows(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", bob)
	rows := []app.InvitationRow{
		{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: "erin@corp.com", Role: shared.RoleMember, CreatedBy: alice, Now: now},
		{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: "dave@corp.com", Role: shared.RoleGuest, CreatedBy: alice, Now: now},
		{ID: uuid.NewV7(), WorkspaceID: beta.ID, Email: "dave@corp.com", Role: shared.RoleAdmin, CreatedBy: bob, Now: now.Add(time.Hour)},
	}

	got, err := s.CreateInvitations(context.Background(), rows)

	if err != nil || len(got) != len(rows) {
		t.Fatalf("CreateInvitations() = %+v, %v; want the %d rows", got, err, len(rows))
	}
	for i, r := range rows {
		want := domain.Invitation{ID: r.ID, WorkspaceID: r.WorkspaceID, Email: r.Email, Role: r.Role, CreatedAt: r.Now, CreatedByID: &r.CreatedBy}
		if !sameInvitation(got[i], want) || got[i].CreatedAt.Location() != time.UTC {
			t.Errorf("row %d: %+v, want %+v", i, got[i], want)
		}
		var createdBy, updatedBy uuid.UUID
		var updated time.Time
		var responded, deleted *time.Time
		if err := pool.QueryRow(context.Background(), `SELECT created_by_id, updated_by_id, updated_at, responded_at, deleted_at
			FROM workspace_member_invites WHERE id = $1`, r.ID).Scan(&createdBy, &updatedBy, &updated, &responded, &deleted); err != nil {
			t.Fatal(err)
		}
		if createdBy != r.CreatedBy || updatedBy != r.CreatedBy || !updated.Equal(r.Now) || responded != nil || deleted != nil {
			t.Errorf("row %d: by %s, %s at %v, responded %v, deleted %v; want by the inviter at %v, pending", i, createdBy, updatedBy, updated,
				responded, deleted, r.Now)
		}
	}
}

// sameInvitation compares two invitations, the pointers by value.
func sameInvitation(a, b domain.Invitation) bool {
	sameTime := (a.RespondedAt == nil) == (b.RespondedAt == nil) && (a.RespondedAt == nil || a.RespondedAt.Equal(*b.RespondedAt))
	sameBy := (a.CreatedByID == nil) == (b.CreatedByID == nil) && (a.CreatedByID == nil || *a.CreatedByID == *b.CreatedByID)
	a.RespondedAt, b.RespondedAt, a.CreatedByID, b.CreatedByID = nil, nil, nil, nil
	return sameTime && sameBy && a.ID == b.ID && a.WorkspaceID == b.WorkspaceID && a.Email == b.Email && a.Role == b.Role &&
		a.Accepted == b.Accepted && a.CreatedAt.Equal(b.CreatedAt)
}

// An address with an undeleted invitation of the workspace, pending or
// declined, refuses the row that repeats it: *app.DuplicateInvitation
// naming it, and nothing after it is inserted; in a transaction, the rows
// before it roll back with it (M3 design 3.8). An address invited to
// another workspace, or whose invitation is deleted, is free.
func TestCreateInvitationsRefusesAnAddressTaken(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	declined := invite(t, s, acme.ID, "dave@corp.com", shared.RoleMember, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $1 WHERE id = $2", now, declined.ID)
	deleted := invite(t, s, acme.ID, "erin@corp.com", shared.RoleMember, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET deleted_at = $1 WHERE id = $2", now, deleted.ID)
	invite(t, s, beta.ID, "frank@corp.com", shared.RoleMember, alice)
	row := func(email string) app.InvitationRow {
		return app.InvitationRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: email, Role: shared.RoleGuest, CreatedBy: alice, Now: now}
	}
	tx := postgres.NewTxManager(pool, 2*time.Second)

	for _, taken := range []string{"carol@corp.com", "dave@corp.com"} {
		var got []domain.Invitation
		err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			got, err = s.CreateInvitations(ctx, []app.InvitationRow{row("erin@corp.com"), row(taken), row("zoe@corp.com")})
			return err
		})
		var dup *app.DuplicateInvitation
		if !errors.As(err, &dup) || dup.Email != taken || got != nil {
			t.Errorf("a batch repeating %s: %+v, %v; want *app.DuplicateInvitation of it", taken, got, err)
		}
		if want := []string{"carol@corp.com", "dave@corp.com"}; !slices.Equal(pendingEmails(t, pool, acme.ID), want) {
			t.Errorf("after the batch repeating %s, acme's invitations are %q, want %q", taken, pendingEmails(t, pool, acme.ID), want)
		}
	}
	if _, err := s.CreateInvitations(context.Background(), []app.InvitationRow{row("erin@corp.com"), row("frank@corp.com")}); err != nil {
		t.Errorf("addresses free in acme: %v", err)
	}
	if want := []string{"carol@corp.com", "dave@corp.com", "erin@corp.com", "frank@corp.com"}; !slices.Equal(pendingEmails(t, pool, acme.ID), want) {
		t.Errorf("acme's invitations are %q, want %q", pendingEmails(t, pool, acme.ID), want)
	}
}
