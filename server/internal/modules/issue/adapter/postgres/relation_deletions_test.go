package postgresadapter_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Ignoring the other project's deletion overwrites the first deletion's ownership.
func TestRelationDeletionOwnership(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "ownership-alice@corp.com"), newAccount(t, pool, "ownership-bob@corp.com")
	for _, reverseProjects := range []bool{false, true} {
		for _, reverseRelation := range []bool{false, true} {
			for _, mode := range []string{"live", "earlier", "same", "later", "internal"} {
				t.Run(fmt.Sprintf("%s/projects-reversed=%v/relation-reversed=%v", mode, reverseProjects, reverseRelation), func(t *testing.T) {
					workspace := newWorkspace(t, pool, uuid.NewV7().String())
					first, second := newProject(t, pool, workspace, "P"), newProject(t, pool, workspace, "Q")
					p, q := first, second
					if reverseProjects {
						p, q = q, p
					}
					if mode == "internal" {
						q = p
					}
					x, y, r := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
					newIssue(t, pool, p, x, 1, alice)
					newIssue(t, pool, q, y, 2, alice)
					if reverseRelation {
						newRelation(t, pool, q, y, x, r, "relates_to", alice)
					} else {
						newRelation(t, pool, p, x, y, r, "relates_to", alice)
					}
					at := now.Add(time.Hour)
					otherAt := now
					if mode == "same" {
						otherAt = at
					}
					if mode == "later" {
						otherAt = at.Add(time.Hour)
					}
					exec(t, pool, "UPDATE projects SET deleted_at=$2,updated_at=$2,updated_by_id=$3 WHERE id=$1", q.id, otherAt, alice)
					before := audits(t, pool, "issue_relations")[r]
					err := postgres.NewTxManager(pool, 5*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
						if mode == "live" {
							return app.NewRelationDeletions(s, relationDeletionProjects{deleted: map[uuid.UUID]time.Time{q.id: otherAt}}).ForIssues(ctx, p.id, app.IssueDeletion{IssueIDs: []uuid.UUID{x}, By: bob, At: at})
						}
						_, err := app.NewRelationDeletions(s, relationDeletionProjects{deleted: map[uuid.UUID]time.Time{q.id: otherAt}}).ForProject(ctx, app.RowBatch{ProjectID: p.id, Limit: 2, By: bob, At: at})
						return err
					})
					if err != nil {
						t.Fatal(err)
					}
					want := deletion(at, bob)
					if mode == "live" || mode == "earlier" {
						want = before
					}
					if got := audits(t, pool, "issue_relations")[r]; got != want {
						t.Fatalf("projectsReverse=%v relationReverse=%v got audit %s want %s", reverseProjects, reverseRelation, got, want)
					}
				})
			}
		}
	}
}

// The issue adapter receives project facts through its consumer port; actual
// project SQL and bootstrap adaptation have their own integration coverage.
type relationDeletionProjects struct {
	app.ProjectGate
	deleted map[uuid.UUID]time.Time
}

func (p relationDeletionProjects) DeletedAt(context.Context, []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	return p.deleted, nil
}

// The database cursor counts locked candidates even when ownership skips all
// of them; endpoint issues already deleted by the preceding step stay eligible.
func TestRelationDeletionCursorUsesCandidates(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "cursor@corp.com")
	workspace := newWorkspace(t, pool, "cursor")
	p, q, live := newProject(t, pool, workspace, "P"), newProject(t, pool, workspace, "Q"), newProject(t, pool, workspace, "LIVE")
	x, y, z := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, p, x, 1, by)
	newIssue(t, pool, q, y, 1, by)
	newIssue(t, pool, live, z, 1, by)
	a, b, c := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newRelation(t, pool, p, x, y, a, "relates_to", by)
	y2 := uuid.NewV7()
	newIssue(t, pool, q, y2, 2, by)
	newRelation(t, pool, q, y2, x, b, "duplicate", by)
	newRelation(t, pool, live, z, x, c, "relates_to", by)
	exec(t, pool, "UPDATE issues SET deleted_at=$2 WHERE id=ANY($1::uuid[])", []uuid.UUID{x, y}, now)
	u := app.NewRelationDeletions(s, relationDeletionProjects{deleted: map[uuid.UUID]time.Time{q.id: now}})
	batch := app.RowBatch{ProjectID: p.id, Limit: 2, At: now.Add(time.Hour), By: by}
	for _, want := range []app.BatchProgress{{After: b, Candidates: 2}, {After: c, Candidates: 1, Written: 1}, {}} {
		err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			got, err := u.ForProject(ctx, batch)
			if err == nil && got != want {
				t.Errorf("progress %+v want %+v", got, want)
			}
			batch.After = got.After
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	got := audits(t, pool, "issue_relations")
	if got[c] != deletion(batch.At, by) || got[a] == deletion(batch.At, by) || got[b] == deletion(batch.At, by) {
		t.Fatal(got)
	}
}

// Equal-time tasks compete for the same actual relation lock. The waiting
// task must re-evaluate the deleted row and preserve the first writer's audit.
func TestRelationDeletionSameTimeCompetitors(t *testing.T) {
	s, pool := newStore(t)
	by := newAccount(t, pool, "competitors@corp.com")
	workspace := newWorkspace(t, pool, "competitors")
	p, q := newProject(t, pool, workspace, "P"), newProject(t, pool, workspace, "Q")
	x, y := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, p, x, 1, by)
	newIssue(t, pool, q, y, 1, by)
	r := uuid.NewV7()
	newRelation(t, pool, q, y, x, r, "relates_to", by)
	u := app.NewRelationDeletions(s, relationDeletionProjects{deleted: map[uuid.UUID]time.Time{p.id: now, q.id: now}})
	ready, release := make(chan struct{}), make(chan struct{})
	first := make(chan error, 1)
	ctx := pgtest.Soon(t)
	go func() {
		first <- postgres.NewTxManager(pool, 5*time.Second).WithinTx(ctx, func(ctx context.Context) error {
			got, err := u.ForProject(ctx, app.RowBatch{ProjectID: p.id, Limit: 2, At: now, By: by})
			if err != nil {
				return err
			}
			if got.Written != 1 {
				return fmt.Errorf("first progress %+v", got)
			}
			close(ready)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	second := make(chan error, 1)
	go func() {
		second <- postgres.NewTxManager(pool, 5*time.Second).WithinTx(ctx, func(ctx context.Context) error {
			got, err := u.ForProject(ctx, app.RowBatch{ProjectID: q.id, Limit: 2, At: now, By: by})
			if err == nil && got != (app.BatchProgress{}) {
				return fmt.Errorf("second progress %+v", got)
			}
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "issue_relations", 3*time.Second)
	close(release)
	for _, done := range []chan error{first, second} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if got := audits(t, pool, "issue_relations")[r]; got != deletion(now, by) {
		t.Fatal(got)
	}
}
