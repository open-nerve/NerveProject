package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"
)

// Removing any candidate predicate or ID ordering must change this answer.
func TestArchivePolicies(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	// The normal schema rejects out-of-range policies. Only this isolated fixture
	// removes the check to prove the scan remains bounded for legacy/corrupt values.
	exec(t, pool, "ALTER TABLE projects DROP CONSTRAINT projects_archive_in_check")
	var want []uuid.UUID
	for i, policy := range []int{-1, 0, 1, 6, 12, 13, 6, 6} {
		id := newProject(t, s, acme, string(rune('A'+i)), string(rune('A'+i)), alice)
		exec(t, pool, "UPDATE projects SET archive_in = $2 WHERE id = $1", id, policy)
		switch i {
		case 2, 3, 4:
			want = append(want, id)
		case 6:
			exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", id, now)
		case 7:
			exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", id, now)
		}
	}
	// Rewriting the first eligible row makes physical order differ from ID order.
	exec(t, pool, "UPDATE projects SET name = 'Rewritten' WHERE id = $1", want[0])
	got, err := s.ArchiveProjects(context.Background())
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("ArchiveProjects() = %v, %v; want %v", got, err, want)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.ArchiveProjects(ctx); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("cancelled scan = %v, %v", got, err)
	}
	exec(t, pool, "UPDATE projects SET archive_in = 0")
	if got, err := s.ArchiveProjects(context.Background()); err != nil || len(got) != 0 {
		t.Errorf("disabled scan = %v, %v", got, err)
	}
}
