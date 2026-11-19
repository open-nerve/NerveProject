package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// The issue module's locks of a project carry whether its intake is
// turned on (M7 P2a spec 2.4): a submission and a triage read the toggle
// under the lock alone. Web's is on, Ops' off, its default; each lock
// answers its own, the lock of one project and of several alike.
func TestTheIssueLocksReadTheIntakeToggle(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	exec(t, pool, "UPDATE projects SET intake_view = true WHERE id = $1", web)

	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		for _, tt := range []struct {
			name string
			id   uuid.UUID
			on   bool
		}{{"Web", web, true}, {"Ops", ops, false}} {
			p, found, err := s.LockIssueProject(ctx, tt.id)
			if err != nil || !found || p.IntakeView != tt.on {
				t.Errorf("LockIssueProject(%s) = %+v, %v, %v; want intake_view %v", tt.name, p, found, err, tt.on)
			}
		}
		locked, err := s.LockIssueProjects(ctx, []uuid.UUID{web, ops})
		if err != nil || len(locked) != 2 || !locked[web].IntakeView || locked[ops].IntakeView {
			t.Errorf("LockIssueProjects() = %+v, %v; want Web's intake_view on and Ops' off", locked, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
