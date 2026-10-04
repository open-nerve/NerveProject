package bootstrap

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// querier is what rowJSON reads through: a pool, or a transaction, which
// sees its own writes.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// rowJSON is the row id of table as JSON, its columns by name; nil when
// there is none.
func rowJSON(t *testing.T, q querier, table string, id uuid.UUID) map[string]any {
	t.Helper()
	var text []byte
	if err := q.QueryRow(pgtest.Soon(t), "SELECT coalesce((SELECT row_to_json(r) FROM "+table+" r WHERE r.id = $1)::text, 'null')", id).
		Scan(&text); err != nil {
		t.Fatal(err)
	}
	var row map[string]any
	if err := json.Unmarshal(text, &row); err != nil {
		t.Fatal(err)
	}
	return row
}

// rowsBut is every table's rows as tableRows has them, River's left out,
// but the rows whose ids are in ids.
func rowsBut(t *testing.T, pool *pgxpool.Pool, ids []uuid.UUID) map[string]string {
	t.Helper()
	all := map[string]string{}
	for table := range tableRows(t, pool, riversOwn) {
		var text string
		if err := pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(string_agg(t, E'\n' ORDER BY t), '') FROM (SELECT row_to_json(r)::text AS t
			FROM `+table+` r WHERE NOT to_jsonb(r) ? 'id' OR (to_jsonb(r)->>'id') <> ALL ($1::text[])) s`, uuidTexts(ids)).Scan(&text); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		all[table] = text
	}
	return all
}

// uuidTexts are ids as text.
func uuidTexts(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

// projectMemberships are the ids of user's memberships of the projects, in
// the order of projects.
func projectMemberships(t *testing.T, pool *pgxpool.Pool, user uuid.UUID, projects ...uuid.UUID) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, len(projects))
	for i, p := range projects {
		if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM project_members WHERE project_id = $1 AND member_id = $2 AND deleted_at IS NULL",
			p, user).Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	return ids
}

// An ending of name's membership of acme on the wired app: alice's removal
// of him, or his leaving.
type ending struct {
	name string
	// request is the ending's method, its path, and the access token it is
	// sent with.
	request func(w endingWorld, t *testing.T, name string) (method, path, token string)
	by      func(name string) string // the account it writes as
	// notFound is the code of its 404: the membership's, or the workspace's.
	notFound string
}

// end sends e's request on w's app: its status and body.
func (e ending) end(w endingWorld, t *testing.T, name string) (int, string) {
	t.Helper()
	method, path, token := e.request(w, t, name)
	return call(t, w.contract, method, w.base+path, token, "")
}

var endings = []ending{
	{name: "removal", request: func(w endingWorld, t *testing.T, name string) (string, string, string) {
		return http.MethodDelete, "/api/v0/workspace-members/" + w.membership(t, name).String(), w.tokens["alice"]
	}, by: func(string) string { return "alice" }, notFound: "workspace.member_not_found"},
	{name: "leaving", request: func(w endingWorld, _ *testing.T, name string) (string, string, string) {
		return http.MethodPost, "/api/v0/workspaces/acme/leave", w.tokens[name]
	}, by: func(name string) string { return name }, notFound: "workspace.not_found"},
}

// A removal and a leaving each end a membership and leave no invitation,
// in one transaction at one moment (M3 design 3.6, 3.7 rule 2, 3.8, 9.3),
// on the wired app.
//   - bob is Ops's only active admin and carol its member, erin's ended
//     membership as its admin counting for nothing: the ending of his
//     membership is 409 project.sole_admin, and no row of any table
//     changes, the pending invitation to his address, which the ending
//     would have deleted, still pending.
//   - Once alice, acme's admin, has joined Ops, as its admin, the ending
//     refused at its commit, after every statement ran, for each table it
//     writes, is 500, and no row changes: no step wrote in a transaction of
//     its own.
//   - Then it is 204, Solo, of which he is the only active member, refusing
//     nothing, nor Lab, beta's, of which he is the only admin: bob's
//     membership of acme, of Web, of Ops and of Solo ended, each row kept
//     with its role; the pending invitation to his address in acme deleted;
//     each by the ender, at one moment no earlier than the request. Each of
//     those rows was last written by dave before, so that the claim of the
//     ender's writing can fail. His membership of beta and of Lab, the
//     invitation to him there, and every other row of every table are as
//     they were.
//   - Ending dave's membership leaves his declined invitation as it was.
func TestAnEndingEndsTheMembershipsAndLeavesNoInvitation(t *testing.T) {
	for _, e := range endings {
		t.Run(e.name, func(t *testing.T) {
			w := newEndingWorld(t)
			before := tableRows(t, w.pool, riversOwn)
			if status, body := e.end(w, t, "bob"); status != http.StatusConflict || problemCode(t, []byte(body)) != "project.sole_admin" {
				t.Fatalf("ending bob's membership, Ops's only admin = %d %s, want 409 project.sole_admin", status, body)
			}
			if after := tableRows(t, w.pool, riversOwn); !maps.Equal(after, before) {
				t.Errorf("the tables after the refused ending changed:\n%v\nwant them as they were:\n%v", after, before)
			}
			w.joinsOps(t)
			before = tableRows(t, w.pool, riversOwn)
			for _, table := range []string{"workspace_member_invites", "workspace_members", "project_members"} {
				restore := refusingCommits(t, w.pool, table)
				status, body := e.end(w, t, "bob")
				restore()
				if after := tableRows(t, w.pool, riversOwn); status != http.StatusInternalServerError || !maps.Equal(after, before) {
					t.Errorf("the ending refused at its commit for %s = %d %s; want 500 and every table as it was", table, status, body)
				}
			}
			written := slices.Concat([]uuid.UUID{w.membership(t, "bob"), w.bobsInvitation},
				projectMemberships(t, w.pool, w.ids["bob"], w.web, w.ops, w.solo))
			tables := []string{"workspace_members", "workspace_member_invites", "project_members", "project_members", "project_members"}
			rowsBefore := make([]map[string]any, len(written))
			for i, id := range written {
				if tag, err := w.pool.Exec(pgtest.Soon(t), "UPDATE "+tables[i]+" SET updated_by_id = $2 WHERE id = $1", id,
					w.ids["dave"]); err != nil || tag.RowsAffected() != 1 {
					t.Fatalf("%s %s last written by dave: %v, %v", tables[i], id, tag, err)
				}
				rowsBefore[i] = rowJSON(t, w.pool, tables[i], id)
			}
			others := rowsBut(t, w.pool, written)
			started := time.Now()

			if status, body := e.end(w, t, "bob"); status != http.StatusNoContent {
				t.Fatalf("ending bob's membership = %d %s, want 204", status, body)
			}

			moment, _ := rowJSON(t, w.pool, "workspace_member_invites", w.bobsInvitation)["deleted_at"].(string)
			if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(started.Truncate(time.Microsecond)) {
				t.Errorf("the invitation deleted at %q (%v); want a moment no earlier than the request, %v", moment, err, started)
			}
			for i, id := range written {
				after := rowJSON(t, w.pool, tables[i], id)
				want := maps.Clone(rowsBefore[i])
				want["updated_at"], want["updated_by_id"] = moment, w.ids[e.by("bob")].String()
				if tables[i] == "workspace_member_invites" {
					want["deleted_at"] = moment
				} else {
					want["is_active"] = false
				}
				if !maps.Equal(after, want) {
					t.Errorf("%s %s after the ending:\n%v\nwant\n%v", tables[i], id, after, want)
				}
			}
			if after := rowsBut(t, w.pool, written); !maps.Equal(after, others) {
				t.Errorf("every other row after the ending:\n%v\nwant them as they were:\n%v", after, others)
			}
			declined := rowJSON(t, w.pool, "workspace_member_invites", w.davesDeclined)
			if status, body := e.end(w, t, "dave"); status != http.StatusNoContent {
				t.Fatalf("ending dave's membership = %d %s, want 204", status, body)
			}
			if after := rowJSON(t, w.pool, "workspace_member_invites", w.davesDeclined); !maps.Equal(after, declined) {
				t.Errorf("dave's declined invitation after the ending:\n%v\nwant it as it was:\n%v", after, declined)
			}
		})
	}
}

// A workspace's only active admin cannot leave it (M3 design 3.7 rule 1,
// 9.3): alice is refused 409 workspace.sole_admin in lone, where she is
// alone, and in acme, beside its members and dave, an admin of it whose
// membership alice's removal ended, and no row of any table changes; once
// carol is acme's admin too, alice leaves it.
func TestTheOnlyAdminCannotLeave(t *testing.T) {
	w := newEndingWorld(t)
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["alice"],
		`{"name":"lone","slug":"lone"}`); status != http.StatusCreated {
		t.Fatalf("creating lone = %d %s", status, body)
	}
	for _, step := range []struct {
		method, body string
		want         int
	}{{http.MethodPatch, `{"role":20}`, http.StatusOK}, {http.MethodDelete, "", http.StatusNoContent}} {
		if status, body := call(t, w.contract, step.method, w.base+"/api/v0/workspace-members/"+w.membership(t, "dave").String(),
			w.tokens["alice"], step.body); status != step.want {
			t.Fatalf("alice's %s of dave's membership of acme = %d %s, want %d", step.method, status, body, step.want)
		}
	}
	// dave is an admin of acme whose membership ended: were he missing, a
	// rule 1 that counted an ended admin as another would pass.
	var role int
	var active bool
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT m.role, m.is_active FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme' AND m.member_id = $1 AND m.deleted_at IS NULL`, w.ids["dave"]).Scan(&role, &active); err != nil ||
		role != int(shared.RoleAdmin) || active {
		t.Fatalf("dave's membership of acme: role %d, active %v, %v; want an admin's, ended", role, active, err)
	}
	before := tableRows(t, w.pool, riversOwn)
	for _, slug := range []string{"lone", "acme"} {
		if status, body := w.leave(t, slug, "alice"); status != http.StatusConflict || problemCode(t, []byte(body)) != "workspace.sole_admin" {
			t.Errorf("alice's leaving %s, its only admin = %d %s, want 409 workspace.sole_admin", slug, status, body)
		}
	}
	if after := tableRows(t, w.pool, riversOwn); !maps.Equal(after, before) {
		t.Errorf("the tables after the refused leavings changed:\n%v\nwant them as they were:\n%v", after, before)
	}
	if status, body := call(t, w.contract, http.MethodPatch, w.base+"/api/v0/workspace-members/"+w.membership(t, "carol").String(),
		w.tokens["alice"], `{"role":20}`); status != http.StatusOK {
		t.Fatalf("carol made acme's admin = %d %s", status, body)
	}
	if status, body := w.leave(t, "acme", "alice"); status != http.StatusNoContent {
		t.Errorf("alice's leaving acme, carol its admin too = %d %s, want 204", status, body)
	}
}
