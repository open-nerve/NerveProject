package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type listGroupWant struct {
	field   string
	keys    []string
	members map[string][]uuid.UUID
}

type listGroupFixture struct {
	w      issueWorld
	ids    []uuid.UUID
	groups map[string]listGroupWant
	keys   map[uuid.UUID]json.RawMessage
	rows   map[uuid.UUID]listRowWant
	a, c   uuid.UUID
}

func seedListGroups(t *testing.T) listGroupFixture {
	t.Helper()
	w := newIssueWorld(t)
	relations := memberRelations(t, w) // Includes genuine A-only/C-only auto-subscriptions.
	for _, id := range relations[3:] {
		listSeedSQL(t, w, "UPDATE issues SET start_date='2030-01-01',target_date=NULL WHERE id=$1", id)
	}
	var label struct{ ID uuid.UUID }
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/labels", `{"name":"C"}`, 201), &label)
	other := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTHER")
	w.makeMember(t, "walt", other, shared.RoleMember)
	w.makeMember(t, "bob", other, shared.RoleMember)
	backlog := stateID(t, w.pool, w.web, "Backlog")
	f := listGroupFixture{w: w, groups: map[string]listGroupWant{}, keys: map[uuid.UUID]json.RawMessage{}, rows: map[uuid.UUID]listRowWant{}, a: w.bug, c: label.ID}
	for n := 0; n < 10; n++ {
		kind := n / 2
		project, state := w.web, []uuid.UUID{w.todo, w.done, backlog, w.todo, stateID(t, w.pool, other, "Done")}[kind]
		if kind == 4 {
			project = other
		}
		actor := []string{"alice", "bob", "alice", "walt", "alice"}[kind]
		labels := [][]uuid.UUID{{f.a, f.c}, {f.c}, {}, {f.a}, {}}[kind]
		assignees := [][]uuid.UUID{{w.ids["alice"], w.ids["bob"]}, {w.ids["bob"]}, {}, {w.ids["alice"]}, {}}[kind]
		body := fmt.Sprintf(`{"name":"Matrix %d","state_id":%q,"label_ids":%s,"assignee_ids":%s,"start_date":"2030-01-01","priority":%q}`, n, state, idArray(labels...), idArray(assignees...), []string{"urgent", "low", "none", "high", "low"}[kind])
		id := w.newWorkItemBy(t, actor, project, body)
		w.send(t, "bob", http.MethodPost, commentsOf(id), `{"comment_html":"<p>Subscribe</p>"}`, 201)
		stamp := fmt.Sprintf("2030-02-01T00:00:00.%06dZ", n)
		listSeedSQL(t, w, "UPDATE issues SET created_at=$2::text::timestamptz,target_date=$3::text::date,created_by_id=CASE WHEN $4 THEN NULL ELSE created_by_id END WHERE id=$1", id, stamp, []*string{new("2030-03-02"), nil, new("2030-03-01"), nil, nil}[kind], kind == 2)
		f.ids = append(f.ids, id)
		f.keys[id] = json.RawMessage(strconv.Quote(stamp))
		f.rows[id] = listRowWant{labels: labels, assignees: assignees}
	}
	for _, column := range []string{"archived_at", "deleted_at"} {
		id := w.newWorkItemBy(t, "bob", w.web, fmt.Sprintf(`{"name":"Excluded","start_date":"2030-01-01","label_ids":%s,"assignee_ids":%s}`, idArray(f.a, f.c), idArray(w.ids["bob"])))
		listSeedSQL(t, w, "UPDATE issues SET "+column+"=now() WHERE id=$1", id)
	}
	// The hidden parent's ID must be pruned for OwnOnly; counts ignore the
	// list's date filter and exclude archived/deleted children.
	listSeedSQL(t, w, "UPDATE issues SET parent_id=$2 WHERE id=$1", f.ids[6], f.ids[0])
	for n, actor := range []string{"walt", "bob", "walt", "walt"} {
		child := w.newWorkItemBy(t, actor, w.web, fmt.Sprintf(`{"name":"Child","parent_id":%q}`, f.ids[6]))
		if n == 2 {
			listSeedSQL(t, w, "UPDATE issues SET archived_at=now() WHERE id=$1", child)
		}
		if n == 3 {
			listSeedSQL(t, w, "UPDATE issues SET deleted_at=now() WHERE id=$1", child)
		}
	}
	r := f.rows[f.ids[0]]
	r.children = 1 // Matrix row 6 is a direct child of row 0.
	f.rows[f.ids[0]] = r
	r = f.rows[f.ids[6]]
	r.children = 2
	f.rows[f.ids[6]] = r
	// A third invisible project also has B's relationship and matching date.
	unseen := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Unseen", "UNSEEN")
	w.makeMember(t, "bob", unseen, shared.RoleMember)
	w.newWorkItemBy(t, "bob", unseen, `{"name":"Unseen","start_date":"2030-01-01"}`)
	// Explicit, independently reviewed membership table; never derive it from
	// returned rows, production grouping code, or the seed's kind switch.
	add := func(field string, keys []string, indexes [][]int) {
		g := listGroupWant{field: field, keys: slices.Clone(keys), members: map[string][]uuid.UUID{}}
		for n, key := range keys {
			for _, index := range indexes[n] {
				g.members[key] = append(g.members[key], f.ids[index])
			}
		}
		if field != "priority" && field != "state_group" {
			slices.Sort(g.keys)
			if slices.Contains(g.keys, "none") {
				g.keys = append(slices.DeleteFunc(g.keys, func(s string) bool { return s == "none" }), "none")
			}
		}
		f.groups[field] = g
	}
	add("state_id", []string{w.todo.String(), w.done.String(), backlog.String()}, [][]int{{0, 1, 6, 7}, {2, 3}, {4, 5}})
	add("state_group", []string{"backlog", "unstarted", "completed"}, [][]int{{4, 5}, {0, 1, 6, 7}, {2, 3, 8, 9}})
	add("priority", []string{"urgent", "high", "low", "none"}, [][]int{{0, 1}, {6, 7}, {2, 3, 8, 9}, {4, 5}})
	add("label_id", []string{f.a.String(), f.c.String(), "none"}, [][]int{{0, 1, 6, 7}, {0, 1, 2, 3}, {4, 5, 8, 9}})
	add("assignee_id", []string{w.ids["alice"].String(), w.ids["bob"].String(), "none"}, [][]int{{0, 1, 6, 7}, {0, 1, 2, 3}, {4, 5, 8, 9}})
	add("created_by_id", []string{w.ids["alice"].String(), w.ids["bob"].String(), w.ids["walt"].String(), "none"}, [][]int{{0, 1, 8, 9}, {2, 3}, {6, 7}, {4, 5}})
	add("target_date", []string{"2030-03-01", "2030-03-02", "none"}, [][]int{{4, 5}, {0, 1}, {2, 3, 6, 7, 8, 9}})
	add("project_id", []string{w.web.String(), other.String()}, [][]int{{0, 1, 2, 3, 4, 5, 6, 7}, {8, 9}})
	return f
}
