package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"
	"uuid"

	issuepostgres "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Stale discovery policy or removal of project N must not archive after a
// policy change, project archival or deletion. Both commit orders use actual
// requests; after_scan separately changes policy after candidate discovery.
func TestAutoArchiveUsesLockedPolicy(t *testing.T) {
	for _, change := range []struct {
		name    string
		write   racingWrite
		status  int
		deleted string
	}{
		{"longer", aliceOnWeb(http.MethodPatch, webPath(""), `{"archive_in":2}`), 200, ""},
		{"disabled", aliceOnWeb(http.MethodPatch, webPath(""), `{"archive_in":0}`), 200, ""},
		{"archived_project", archivesWeb, 200, ""},
		{"deleted_project", deletesWeb, 204, "projects"},
		{"deleted_workspace", deletesAcme, 204, "workspaces"},
	} {
		for _, order := range []string{"after_scan", "request_first", "job_first"} {
			t.Run(change.name+"/"+order, func(t *testing.T) {
				w := newIssueWorld(t)
				// Project/workspace routes use the real clock, after this system time.
				at := time.Now().UTC().Truncate(time.Microsecond).Add(-time.Hour)
				id := archiveRaceSeed(t, w, at, 40*24*time.Hour)
				before := issueSnapshot(t, w, id)
				stable := archiveStableRow(t, w, id)
				rows := w.workItemRows(t, []uuid.UUID{w.web}, false)
				events := newArchiveRaceEvents(t, w)
				g := newGate()
				var scanned, locked *gate
				tx := shared.TxManager(postgres.NewTxManager(w.pool, 2*time.Second))
				switch order {
				case "after_scan":
					scanned = g
				case "job_first":
					locked = g
				case "request_first":
					tx = heldTx{tx, g}
				}
				job := archiveRaceJob(w, clockAt(at), events, scanned, locked)
				n := 0
				var reply exchange
				request := change.write(t, w, tx, &reply)
				archive := func() error { var err error; n, err = job.Execute(pgtest.Soon(t)); return err }
				if order == "after_scan" {
					done := run(archive)
					defer func() {
						select {
						case <-g.open:
						default:
							close(g.open)
						}
						if err := result(t, pgtest.Soon(t), done, "scan cleanup"); err != nil {
							t.Error(err)
						}
					}()
					held(t, pgtest.Soon(t), g, done, "candidate scan")
					if err := request(); err != nil {
						t.Fatal(err)
					}
					close(g.open)
					if err := result(t, pgtest.Soon(t), done, "archive after changed candidate"); err != nil {
						t.Fatal(err)
					}
				} else {
					first, second := request, archive
					var lockedCheck, waitingCheck func()
					if order == "job_first" {
						first, second = archive, request
						lockedCheck = func() {
							// N conflicts with SHARE, not KEY SHARE. No workspace or issue
							// prelock exists before the automatic UPDATE and activity INSERT.
							if !pgtest.RowWaits(t, w.pool, "projects", w.web, "FOR SHARE") || pgtest.RowWaits(t, w.pool, "projects", w.web, "FOR KEY SHARE") {
								t.Error("job does not hold project N")
							}
							if pgtest.RowWaits(t, w.pool, "workspaces", w.acme, "FOR UPDATE") || pgtest.RowWaits(t, w.pool, "issues", id, "FOR UPDATE") {
								t.Error("job prelocked workspace or issue")
							}
						}
						if change.deleted == "workspaces" {
							waitingCheck = func() {
								if !pgtest.RowWaits(t, w.pool, "workspaces", w.acme, "FOR SHARE") || pgtest.RowWaits(t, w.pool, "workspaces", w.acme, "FOR KEY SHARE") {
									t.Error("workspace deletion must hold N, compatible with the upcoming activity FK")
								}
							}
						}
					}
					archiveInterleave(t, w, g, first, second, lockedCheck, waitingCheck)
				}
				w.answered(t, reply, change.status)
				want := 0
				if order == "job_first" {
					want = 1
				}
				if n != want {
					t.Errorf("locked policy archived=%d want=%d", n, want)
				}
				switch change.deleted {
				case "projects":
					waitProjectDeletion(t, w.pool, w.web)
				case "workspaces":
					waitWorkspaceDeletion(t, w.pool, w.acme)
				}
				archiveRaceCheck(t, w, id, at, want, events, change.deleted != "")
				if change.deleted != "" {
					parent := w.web
					if change.deleted == "workspaces" {
						parent = w.acme
					}
					w.deletedWith(t, change.deleted, parent, at, rows+want)
				} else if want == 0 {
					if after := issueSnapshot(t, w, id); after != before {
						t.Error("policy rejection left issue/activity writes")
					}
				} else if archiveStableRow(t, w, id) != stable {
					t.Error("system archival changed unrelated issue fields")
				}
			})
		}
	}
}

// Several candidates must not become one transaction holding all projects.
func TestAutoArchiveUsesLockedPolicyOneProjectAtATime(t *testing.T) {
	w := newIssueWorld(t)
	at := time.Now().UTC().Truncate(time.Microsecond)
	archiveRaceSeed(t, w, at, 40*24*time.Hour)
	other := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTH")
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+other.String(), `{"archive_in":1}`, 200)
	g, second := newGate(), newGate()
	events := newArchiveRaceEvents(t, w)
	p := project.Provide(w.pool)
	states := &archiveFirstStates{StateCatalog: issueProjects{projects: p.IssueProjects}, gates: []*gate{g, second}}
	job := issueapp.NewAutoArchive(issueapp.AutoArchiveDeps{Policies: p.ArchivePolicies, Projects: issueProjects{projects: p.IssueProjects}, States: states, Issues: issuepostgres.New(w.pool), Tx: newIssueJobTx(w.pool, config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: testIssueConfig}), Clock: clockAt(at), Events: events})
	count := 0
	done := run(func() error { var err error; count, err = job.Execute(pgtest.Soon(t)); return err })
	defer func() {
		for _, barrier := range []*gate{g, second} {
			select {
			case <-barrier.open:
			default:
				close(barrier.open)
			}
		}
		if err := result(t, pgtest.Soon(t), done, "single project cleanup"); err != nil {
			t.Error(err)
		}
	}()
	held(t, pgtest.Soon(t), g, done, "first project")
	if !pgtest.RowWaits(t, w.pool, "projects", w.web, "FOR SHARE") || pgtest.RowWaits(t, w.pool, "projects", other, "FOR UPDATE") {
		t.Error("job holds more than its current project")
	}
	var reply exchange
	request := aliceOnWeb(http.MethodPatch, func(*testing.T, issueWorld) string { return "/api/v0/projects/" + other.String() }, `{"name":"Unblocked"}`)(t, w, postgres.NewTxManager(w.pool, 2*time.Second), &reply)
	if err := request(); err != nil {
		t.Fatal(err)
	}
	w.answered(t, reply, 200)
	close(g.open)
	held(t, pgtest.Soon(t), second, done, "second project")
	if pgtest.RowWaits(t, w.pool, "projects", w.web, "FOR SHARE") || !pgtest.RowWaits(t, w.pool, "projects", other, "FOR SHARE") {
		t.Error("first project lock was not released while second project is held")
	}
	close(second.open)
	if err := result(t, pgtest.Soon(t), done, "both archive projects"); err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(states.ids) != 2 || states.ids[0] != w.web || states.ids[1] != other || len(events.snapshot()) != 1 || issueEventCount(t, w) != 1 {
		t.Fatalf("project observations=%v count=%d events=%d jobs=%d", states.ids, count, len(events.snapshot()), issueEventCount(t, w))
	}
}

type archiveFirstStates struct {
	issueapp.StateCatalog
	gates []*gate
	ids   []uuid.UUID
}

func (s *archiveFirstStates) States(ctx context.Context, id uuid.UUID) ([]issueapp.StateRef, error) {
	rows, err := s.StateCatalog.States(ctx, id)
	if err == nil {
		s.ids = append(s.ids, id)
		if len(s.ids) <= len(s.gates) {
			err = s.gates[len(s.ids)-1].wait(ctx)
		}
	}
	return rows, err
}
