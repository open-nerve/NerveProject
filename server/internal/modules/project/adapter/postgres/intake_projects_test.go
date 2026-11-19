package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"
)

// The issue module's pending counts ask which of the projects it sees
// have their intake turned on (M7 P2b spec 2.6, DP-20): those of the ids
// that are undeleted, unarchived and turned on, in id order, of any
// workspace; not one turned off, archived or deleted, nor an id of no
// project. It reads no membership: which projects the caller sees is
// VisibleIssueProjects'.
func TestTheIntakeProjectsAreThoseTurnedOn(t *testing.T) {
	s, pool := newStore(t)
	owner := newAccount(t, pool, "owner@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	on, also := newProject(t, s, acme, "On", "ON", owner), newProject(t, s, beta, "Also", "ALS", owner)
	off := newProject(t, s, acme, "Off", "OFF", owner)
	archived, deleted := newProject(t, s, acme, "Archived", "ARC", owner), newProject(t, s, acme, "Deleted", "DEL", owner)
	exec(t, pool, "UPDATE projects SET intake_view=true WHERE id=ANY($1::uuid[])", []uuid.UUID{on, also, archived, deleted})
	exec(t, pool, "UPDATE projects SET archived_at=$2 WHERE id=$1", archived, now)
	exec(t, pool, "UPDATE projects SET deleted_at=$2 WHERE id=$1", deleted, now)

	asked := []uuid.UUID{deleted, uuid.NewV7(), archived, off, also, on}
	got, err := s.IntakeIssueProjects(context.Background(), asked)
	if want := slices.SortedFunc(slices.Values([]uuid.UUID{on, also}), uuid.UUID.Compare); err != nil || !slices.Equal(got, want) {
		t.Errorf("IntakeIssueProjects() = %v, %v; want %v", got, err, want)
	}
	for _, ids := range [][]uuid.UUID{{off, archived, deleted}, {}, nil} {
		if got, err := s.IntakeIssueProjects(context.Background(), ids); err != nil || got == nil || len(got) != 0 {
			t.Errorf("IntakeIssueProjects(%v) = %v, %v; want none, not nil", ids, got, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.IntakeIssueProjects(ctx, asked); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("cancelled: %v, %v", got, err)
	}
}
