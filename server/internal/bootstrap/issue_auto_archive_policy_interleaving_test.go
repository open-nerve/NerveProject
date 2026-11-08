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
				since := time.Now().UTC().Truncate(time.Microsecond)
				at := since.Add(archiveAhead)
				if change.deleted != "" && order == "job_first" {
					// Its archive precedes a deletion on the real clock of the
					// project and workspace routes, so it dates before that clock,
					// where the app's own start-time auto-archive finds it due too:
					// that run ends first.
					waitForStartupRuns(t, w.pool)
					at = time.Now().UTC().Truncate(time.Microsecond).Add(-time.Hour)
				}
				id := archiveRaceSeed(t, w, at, 40*24*time.Hour)
				before := issueSnapshot(t, w, id)
				stable := archiveStableRow(t, w, id)
				rows := w.workItemRows(t, []uuid.UUID{w.web}, false)
				events := newArchiveRaceEvents(t, w)
				g := newGate()
				var scanned, locked *gate
				reqTx, jobTx := requestTx(w.pool), &requestProbe{TxManager: archiveJobTx(w)}
				tx := shared.TxManager(reqTx)
				switch order {
				case "after_scan":
					scanned = g
				case "job_first":
					locked = g
				case "request_first":
					tx = heldTx{tx, g}
				}
				job := archiveRaceJobOn(w, jobTx, clockAt(at), events, scanned, locked)
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
					first, second, waiter := request, archive, jobTx
					var lockedCheck, waitingCheck func()
					if order == "job_first" {
						first, second, waiter = archive, request, reqTx
						// Each probe reads its actor's own locks: a start-time run
						// locks project rows too (waitForStartupRuns).
						lockedCheck = func() {
							// N conflicts with SHARE, not KEY SHARE. No workspace or issue
							// prelock exists before the automatic UPDATE and activity INSERT.
							if requestGateRowLock(t, w.pool, w.pool, jobTx, "projects WHERE id = $1", w.web) != "FOR NO KEY UPDATE" {
								t.Error("job does not hold project N")
							}
							if requestGateRowLock(t, w.pool, w.pool, jobTx, "workspaces WHERE id = $1", w.acme) != "no lock" ||
								requestGateRowLock(t, w.pool, w.pool, jobTx, "issues WHERE id = $1", id) != "no lock" {
								t.Error("job prelocked workspace or issue")
							}
						}
						if change.deleted == "workspaces" {
							waitingCheck = func() {
								if requestRowLock(t, w.pool, w.pool, reqTx, "workspaces WHERE id = $1", w.acme) != "FOR NO KEY UPDATE" {
									t.Error("workspace deletion must hold N, compatible with the upcoming activity FK")
								}
							}
						}
					}
					archiveInterleave(t, w, g, first, second, waiter, lockedCheck, waitingCheck)
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
					w.deletedWith(t, change.deleted, parent, since, rows+want)
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
	// The stand-in below must be the first to queue on Web behind the job: a
	// real start-time run, which locks every project in turn, may otherwise be
	// there first (branch CI 42453471322).
	waitForStartupRuns(t, w.pool)
	at := time.Now().UTC().Truncate(time.Microsecond).Add(archiveAhead)
	archiveRaceSeed(t, w, at, 40*24*time.Hour)
	other := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTH")
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+other.String(), `{"archive_in":1}`, 200)
	g, second := newGate(), newGate()
	events := newArchiveRaceEvents(t, w)
	p := project.Provide(w.pool, nil)
	states := &archiveFirstStates{StateCatalog: issueProjects{projects: p.IssueProjects}, gates: []*gate{g, second}}
	jobTx := &requestProbe{TxManager: archiveJobTx(w)}
	job := issueapp.NewAutoArchive(issueapp.AutoArchiveDeps{Policies: p.ArchivePolicies, Projects: issueProjects{projects: p.IssueProjects}, States: states, Issues: issuepostgres.New(w.pool), Tx: jobTx, Clock: clockAt(at), Events: events})
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
	// Each probe reads the job's own locks: the stand-in below locks Web too, as
	// a start-time run does.
	if requestGateRowLock(t, w.pool, w.pool, jobTx, "projects WHERE id = $1", w.web) != "FOR NO KEY UPDATE" ||
		requestGateRowLock(t, w.pool, w.pool, jobTx, "projects WHERE id = $1", other) != "no lock" {
		t.Error("job holds more than its current project")
	}
	// As in main CI 42399566014, a start-time run queues on Web behind the job,
	// takes it as the job lets go, and holds it while the job holds Other.
	standIn := queuedRun(t, w.pool, requestPID(t, jobTx), "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", w.web)
	var reply exchange
	request := aliceOnWeb(http.MethodPatch, func(*testing.T, issueWorld) string { return "/api/v0/projects/" + other.String() }, `{"name":"Unblocked"}`)(t, w, postgres.NewTxManager(w.pool, 2*time.Second), &reply)
	if err := request(); err != nil {
		t.Fatal(err)
	}
	w.answered(t, reply, 200)
	close(g.open)
	held(t, pgtest.Soon(t), second, done, "second project")
	letGoOnceWaitedOn(t, w.pool, standIn())
	if requestGateRowLock(t, w.pool, w.pool, jobTx, "projects WHERE id = $1", w.web) != "no lock" ||
		requestGateRowLock(t, w.pool, w.pool, jobTx, "projects WHERE id = $1", other) != "FOR NO KEY UPDATE" {
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
