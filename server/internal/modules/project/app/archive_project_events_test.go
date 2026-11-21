package app_test

import (
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// wantProjectArchive checks that published is one event, bob's archive of
// project, of acme, at at, the moment written into its archived_at; with no
// changes (M7 P3c spec 2.6, R3).
func wantProjectArchive(t *testing.T, published []shared.Event, project uuid.UUID, at time.Time) {
	t.Helper()
	if len(published) != 1 {
		t.Fatalf("published %+v, want the one archive", published)
	}
	e := published[0]
	if e.ID == (uuid.UUID{}) || e.Type != "project.archived" || e.Entity != (shared.EntityRef{Kind: "project", ID: project}) || e.WorkspaceID != acme.ID ||
		e.ProjectID != project || e.ActorID == nil || *e.ActorID != bob || !e.OccurredAt.Equal(at) || len(e.Changes) != 0 {
		t.Errorf("published %+v; want bob's archive of %s in %s at %v", e, project, acme.ID, at)
	}
}

// tickingClock is a second later after each read, each read logged as "Now":
// a use case that read it again for its event, or wrote another reading than
// the one after its locks, would publish another moment than the one it
// wrote.
type tickingClock struct {
	at  time.Time
	log *callLog
}

func (c *tickingClock) Now() time.Time {
	c.log.calls = append(c.log.calls, "Now")
	read := c.at
	c.at = c.at.Add(time.Second)
	return read
}

// An archive's event carries the moment its write stamped into archived_at:
// the clock's reading after the locks, the second, and not the first, which
// only checks the caller's credential before them (the account's lock, the
// transaction's first); the workspace is the one the project's lock read. An
// archived project archived again takes the new moment and publishes it
// (M7 P3c spec 2.6, R3; M3 design 3.19).
func TestAProjectsArchiveIsPublishedAtTheMomentItWrites(t *testing.T) {
	written := now.Add(time.Second)
	for name, project := range map[string]uuid.UUID{"web": webID, "ops, archived already": opsID} {
		t.Run(name, func(t *testing.T) {
			f := newWrites()
			clock := &tickingClock{at: now, log: f.log}
			uc := app.NewArchiveProject(app.ArchiveProjectDeps{Projects: f.store, Locks: f.locks(), Tx: f.tx, Clock: clock, Caller: imageCaller{f.log},
				Images: noProjectImages{}, Events: f.events})

			got, err := uc.Execute(as(bob), project)

			write := fmt.Sprintf("SetArchived %s true by %s at %s", project, bob, written.Format(timeFormat))
			if err != nil || got.ArchivedAt == nil || !got.ArchivedAt.Equal(written) || !slices.Contains(f.log.calls, write) {
				t.Fatalf("Execute() = %+v, %v, calls %q; want %q, archived at %v", got, err, f.log.calls, write, written)
			}
			wantProjectArchive(t, f.events.events, project, written)
		})
	}
}
