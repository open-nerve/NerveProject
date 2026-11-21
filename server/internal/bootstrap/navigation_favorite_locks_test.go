package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// favoriteLockWrite is a write of TestFavoriteWritesTakeTheirTargetsLocks:
// what it is, its operation, its request and what it answers once the
// other transaction ends; in the phases of a view's row and of one's
// favorite of the target, the row the other transaction holds
// (waitingFor); and, while it waits, the rows it holds FOR SHARE in its
// transaction (shared) and those it holds no lock on (free).
type favoriteLockWrite struct {
	name, op, method, path, body string
	want                         int
	held                         uuid.UUID
	shared, free                 []lockedRow
}

// holding is w, holding the rows shared FOR SHARE and none of free while
// it waits.
func (w favoriteLockWrite) holding(shared, free []lockedRow) favoriteLockWrite {
	w.shared, w.free = shared, free
	return w
}

// waitingFor is w, which waits for the row id: a view's or a favorite's.
func (w favoriteLockWrite) waitingFor(id uuid.UUID) favoriteLockWrite {
	w.held = id
	return w
}

// lockedRow is one row a probe looks at: a table and its WHERE clause over
// the row's id.
type lockedRow struct {
	from string
	id   uuid.UUID
}

func (r lockedRow) String() string { return strings.Fields(r.from)[0] + " " + r.id.String() }

// Each write of one's favorites, createFavorite, updateFavorite and
// deleteFavorite, takes its locks in the one order of M7 P3c spec 2.8 (R6),
// as bootstrap wires it: its workspace's row FOR SHARE first, in its
// transaction; then its target's project's row FOR SHARE, a cycle's and a
// module's as a project's, for their writes lock the project's row; then a
// view's row FOR SHARE; then the favorites' rows in their one total order,
// a folder's before an item's. A favorite is a write of the workspace's, no
// write on a project (writesOnAProject): its matrix rows are of the
// workspace level's columns alone, and this test stands for the project's
// lock table. In acme, whose admin alice made Web (issueWorld), with Web's
// cycles and views turned on, alice makes for each phase a folder of hers,
// the targets its writes name, cycles and views of Web and views of acme's
// own, and the favorites they change, delete or create again; once the
// start-time jobs have run (waitForStartupRuns), and with no fan-out
// pending (waitForFanouts), another transaction holds a row and each write
// of the phase is sent:
//   - The workspace first: the other transaction holds acme's row FOR NO
//     KEY UPDATE, as a change of a role in it or its deletion does. A
//     creation of Web's view's favorite in her folder, a move of her
//     favorite of a cycle of Web into that folder and a deletion of
//     another wait for acme's row, and meanwhile hold neither Web's row,
//     nor the view's, nor any favorite's row.
//   - The project after its workspace: the other transaction holds Web's
//     row FOR NO KEY UPDATE, as an archiving, a deletion or a cycle's
//     write does. A creation in her folder of a favorite of Web, of a
//     cycle and of a view of Web, a move into it of her favorite of
//     another view of Web and a deletion of her favorite of another cycle
//     wait for Web's row, and meanwhile hold acme's row FOR SHARE, and no
//     lock on the view's row or on any favorite's.
//   - The view after its project: the other transaction holds a view of
//     Web's row FOR UPDATE, as its change or its deletion does. A creation
//     in her folder of a favorite of it, a move into the folder of her
//     favorite of another and a deletion of her favorite of a third wait
//     for the view's row, each for its own, and meanwhile hold acme's and
//     Web's rows FOR SHARE, and no lock on any favorite's row.
//   - A folder's before its favorites: the other transaction holds her
//     folder's row FOR UPDATE, then the favorites it holds FOR UPDATE in
//     id order, as its deletion does. A creation of a favorite of a view of
//     acme's own in it, a move of her favorite of another into it and a
//     change of her favorite of a third that it holds, which names it its
//     parent again, wait for the folder's row in the statement that shares
//     it (ShareFolder), not in the one that locks their own (LockFavorite),
//     and meanwhile hold acme's row and their view's FOR SHARE, the one
//     moved in no lock on its own row.
//   - One's favorite of the target last: the other transaction holds her
//     favorite of a view of Web FOR UPDATE, as its deletion does, then her
//     favorite of a cycle of Web. A creation of a favorite of each again,
//     the view's in her folder, waits for the one she has in the statement
//     that shares it (ShareFavoriteOfTarget), its last lock before its
//     write, and meanwhile holds acme's and Web's rows FOR SHARE, and the
//     view's and the folder's when it names them; it answers the one she
//     has as it is, 200.
//
// Each wait is the request's own, its PID's, and caused by the other
// transaction (requestBlocked), in the statement of the lock the phase is
// about; each row it holds or does not is its own transaction's lock
// (requestRowLock); and meanwhile it has written no table, a creation no
// row of its own yet (requestWrittenTables). Once the other transaction
// ends, the write answers as it would alone. Each write in each phase is a
// subtest, which names it when it fails; the first that fails ends the
// test, as a write left waiting could hold what the next one's probes look
// at.
func TestFavoriteWritesTakeTheirTargetsLocks(t *testing.T) {
	dbURL := pgtest.NewDatabase(t)
	w := newIssueWorldOnRequests(t, dbURL, openPool(t, dbURL), func(*config.Config) {}, &requestQueries{})
	w.turnCyclesOn(t, w.web)
	w.changeProject(t, w.web, `{"issue_views_view":true}`)
	cycle := func(name string) uuid.UUID { return w.newCycle(t, "alice", `{"name":"`+name+`"}`) }
	view := func(path, name string) uuid.UUID {
		v, _ := w.newView(t, "alice", path, `{"name":"`+name+`"}`)
		return v.ID
	}
	webView := func(name string) uuid.UUID { return view(projectViewsPath(w.web), name) }
	acmeView := func(name string) uuid.UUID { return view(workspaceViewsPath("acme"), name) }
	favor := func(body string) uuid.UUID {
		var f favoriteAnswer
		decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces/acme/favorites", body, http.StatusCreated), &f)
		return f.ID
	}
	folder := func(name string) uuid.UUID { return favor(`{"entity_type":"folder","name":"` + name + `"}`) }
	of := func(kind string, id uuid.UUID) string {
		return fmt.Sprintf(`{"entity_type":%q,"entity_id":%q}`, kind, id)
	}
	in := func(kind string, id, parent uuid.UUID) string {
		return fmt.Sprintf(`{"entity_type":%q,"entity_id":%q,"parent_id":%q}`, kind, id, parent)
	}
	into := func(parent uuid.UUID) string { return fmt.Sprintf(`{"parent_id":%q}`, parent) }
	create := func(name, body string) favoriteLockWrite {
		return favoriteLockWrite{name: name, op: "createFavorite", method: http.MethodPost, path: "/api/v0/workspaces/acme/favorites",
			body: body, want: http.StatusCreated}
	}
	again := func(name, body string) favoriteLockWrite {
		c := create(name, body)
		c.want = http.StatusOK
		return c
	}
	update := func(name string, id uuid.UUID, body string) favoriteLockWrite {
		return favoriteLockWrite{name: name, op: "updateFavorite", method: http.MethodPatch, path: "/api/v0/favorites/" + id.String(),
			body: body, want: http.StatusOK}
	}
	remove := func(name string, id uuid.UUID) favoriteLockWrite {
		return favoriteLockWrite{name: name, op: "deleteFavorite", method: http.MethodDelete, path: "/api/v0/favorites/" + id.String(),
			want: http.StatusNoContent}
	}
	acme, web := lockedRow{"workspaces WHERE id = $1", w.acme}, lockedRow{"projects WHERE id = $1", w.web}
	viewRow := func(id uuid.UUID) lockedRow { return lockedRow{"issue_views WHERE id = $1", id} }
	favoriteRow := func(id uuid.UUID) lockedRow { return lockedRow{"user_favorites WHERE id = $1", id} }
	rows := func(rs ...lockedRow) []lockedRow { return rs }

	fA, vA := folder("A"), webView("A")
	uA, dA := favor(of("cycle", cycle("A1"))), favor(of("cycle", cycle("A2")))
	fB, cB, vB, vB4 := folder("B"), cycle("B"), webView("B"), webView("B4")
	uB, dB := favor(of("view", vB4)), favor(of("cycle", cycle("B5")))
	fC, vC1, vC2, vC3 := folder("C"), webView("C1"), webView("C2"), webView("C3")
	uC, dC := favor(of("view", vC2)), favor(of("view", vC3))
	fD, wD1, wD2, wD3 := folder("D"), acmeView("D1"), acmeView("D2"), acmeView("D3")
	xD, kD := favor(of("view", wD2)), favor(in("view", wD3, fD))
	fE, vE, cE := folder("E"), webView("E"), cycle("E")
	hvE, hcE := favor(of("view", vE)), favor(of("cycle", cE))
	phases := []struct {
		name, table, statement string
		hold                   func(t *testing.T, write favoriteLockWrite) pgx.Tx
		writes                 []favoriteLockWrite
	}{
		{"its workspace first", "workspaces", "ShareDirectoryWorkspaceByID", func(t *testing.T, _ favoriteLockWrite) pgx.Tx {
			return holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", w.acme)
		}, []favoriteLockWrite{
			create("a view of Web's in her folder", in("view", vA, fA)).holding(nil, rows(web, viewRow(vA), favoriteRow(fA))),
			update("a cycle's into her folder", uA, into(fA)).holding(nil, rows(web, favoriteRow(fA), favoriteRow(uA))),
			remove("a cycle's", dA).holding(nil, rows(web, favoriteRow(dA))),
		}},
		{"its project after its workspace", "projects", "ShareNavigationProject", func(t *testing.T, _ favoriteLockWrite) pgx.Tx {
			return holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", w.web)
		}, []favoriteLockWrite{
			create("Web in her folder", in("project", w.web, fB)).holding(rows(acme), rows(favoriteRow(fB))),
			create("a cycle of Web's in her folder", in("cycle", cB, fB)).holding(rows(acme), rows(favoriteRow(fB))),
			create("a view of Web's in her folder", in("view", vB, fB)).holding(rows(acme), rows(viewRow(vB), favoriteRow(fB))),
			update("a view's into her folder", uB, into(fB)).holding(rows(acme), rows(viewRow(vB4), favoriteRow(fB), favoriteRow(uB))),
			remove("a cycle's", dB).holding(rows(acme), rows(favoriteRow(dB))),
		}},
		{"its view after its project", "issue_views", "ShareView", func(t *testing.T, write favoriteLockWrite) pgx.Tx {
			return holding(t, w.pool, "SELECT 1 FROM issue_views WHERE id = $1 FOR UPDATE", write.held)
		}, []favoriteLockWrite{
			create("the view's in her folder", in("view", vC1, fC)).waitingFor(vC1).holding(rows(acme, web), rows(favoriteRow(fC))),
			update("the view's into her folder", uC, into(fC)).waitingFor(vC2).holding(rows(acme, web), rows(favoriteRow(fC), favoriteRow(uC))),
			remove("the view's", dC).waitingFor(vC3).holding(rows(acme, web), rows(favoriteRow(dC))),
		}},
		{"a folder's before its favorites", "user_favorites", "ShareFolder", func(t *testing.T, _ favoriteLockWrite) pgx.Tx {
			other := holding(t, w.pool, "SELECT 1 FROM user_favorites WHERE id = $1 FOR UPDATE", fD)
			if _, err := other.Exec(pgtest.Soon(t), "SELECT id FROM user_favorites WHERE parent_id = $1 AND deleted_at IS NULL ORDER BY id FOR UPDATE",
				fD); err != nil {
				t.Fatal(err)
			}
			return other
		}, []favoriteLockWrite{
			create("a view of acme's in the folder", in("view", wD1, fD)).holding(rows(acme, viewRow(wD1)), nil),
			update("a view's into the folder", xD, into(fD)).holding(rows(acme, viewRow(wD2)), rows(favoriteRow(xD))),
			update("a view's in the folder, the folder named again", kD, fmt.Sprintf(`{"parent_id":%q,"sequence":1}`, fD)).
				holding(rows(acme, viewRow(wD3)), nil),
		}},
		{"one's favorite of its target last", "user_favorites", "ShareFavoriteOfTarget", func(t *testing.T, write favoriteLockWrite) pgx.Tx {
			return holding(t, w.pool, "SELECT 1 FROM user_favorites WHERE id = $1 FOR UPDATE", write.held)
		}, []favoriteLockWrite{
			again("a view of Web's she has, in her folder", in("view", vE, fE)).waitingFor(hvE).holding(rows(acme, web, viewRow(vE), favoriteRow(fE)), nil),
			again("a cycle of Web's she has", of("cycle", cE)).waitingFor(hcE).holding(rows(acme, web), nil),
		}},
	}
	waitForStartupRuns(t, w.pool)
	for _, phase := range phases {
		for _, write := range phase.writes {
			if !t.Run(phase.name+"/"+write.op+", "+write.name, func(t *testing.T) {
				req := newRequest(t, write.method, w.base+write.path, w.tokens["alice"], []byte(write.body))
				w.contract.CheckRequest(t, req)
				waitForFanouts(t, w.pool)
				other := phase.hold(t, write)
				probe := w.requests.observe(req)
				answered := sendInBackground(req)
				pid := requestBlocked(t, w.pool, probe, other.Conn().PgConn().PID(), phase.table)
				sample, waiting := readRequestLock(t, pgtest.Soon(t), w.pool, probe)
				if !waiting || !strings.Contains(sample.query, "-- name: "+phase.statement+" ") {
					t.Errorf("%s waits in %q; want its %s", write.op, strings.Join(strings.Fields(sample.query), " "), phase.statement)
				}
				for _, r := range write.shared {
					if lock := requestRowLock(t, w.pool, w.pool, probe, r.from, r.id); lock != "FOR SHARE" {
						t.Errorf("%s holds %s %s while it waits for %s; want FOR SHARE", write.op, r, lock, phase.table)
					}
				}
				for _, r := range write.free {
					if lock := requestRowLock(t, w.pool, w.pool, probe, r.from, r.id); lock != "no lock" {
						t.Errorf("%s holds %s %s while it waits for %s; want no lock", write.op, r, lock, phase.table)
					}
				}
				if written := requestWrittenTables(t, w.pool, pid); len(written) != 0 {
					t.Errorf("%s has written %v while it waits for %s; want nothing written", write.op, written, phase.table)
				}
				if err := other.Rollback(context.Background()); err != nil {
					t.Fatal(err)
				}
				a := receiveWithin(t, answered, 10*time.Second, "the answer to "+write.op)
				if a.err != nil {
					t.Fatal(a.err)
				}
				w.contract.CheckResponse(t, req, a.res)
				if a.res.StatusCode != write.want {
					t.Errorf("%s = %d %s, want %d", write.op, a.res.StatusCode, a.body, write.want)
				}
			}) {
				t.FailNow()
			}
		}
	}
}
