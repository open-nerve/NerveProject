package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strconv"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type listOrderSample struct {
	Order string
	Rows  []struct {
		ID    uuid.UUID
		Names []string
		Key   json.RawMessage
	}
	AscendingIDs  []uuid.UUID `json:"ascending_ids"`
	DescendingIDs []uuid.UUID `json:"descending_ids"`
}

// Reversing the UUID tie break, truncating microseconds, choosing max names or
// putting descending nulls first must break these fixed, externally stored IDs.
func TestIssueListAllOrders(t *testing.T) {
	raw, err := os.ReadFile("../../../api/testdata/issue-order.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct{ Scenarios []listOrderSample }
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	orders := 0
	for _, sc := range fixture.Scenarios {
		t.Run(sc.Order, func(t *testing.T) {
			w, keys, rows := seedListOrder(t, &sc)
			path := "/api/v0/projects/" + w.web.String() + "/issues"
			for _, direction := range []string{"", "-"} {
				if sc.Order == "sort_order" && direction != "" {
					continue
				}
				orders++
				want := sc.AscendingIDs
				if direction != "" {
					want = sc.DescendingIDs
				}
				want = slices.DeleteFunc(slices.Clone(want), func(id uuid.UUID) bool { _, found := keys[id]; return !found })
				for _, shape := range []string{"flat", "groups", "subgroups", "member"} {
					t.Run(direction+sc.Order+"/"+shape, func(t *testing.T) {
						q := url.Values{"order_by": {direction + sc.Order}, "sub_issue": {"false"}, "created_by_id": {w.ids["bob"].String()}}
						endpoint := path
						if sc.Order == "link_count" {
							q.Set("limit", "2")
						}
						switch shape {
						case "groups", "subgroups":
							q.Set("group_by", "created_by_id")
							q.Set("group_key", w.ids["bob"].String())
							if shape == "subgroups" {
								q.Set("sub_group_by", "state_group")
								q.Set("sub_group_key", "backlog")
							}
						case "member":
							endpoint = memberIssuesPath(w, "bob")
							q.Del("created_by_id")
							q.Set("relation", "created")
						}
						checkListPages(t, w, "walt", endpoint, q, want, keys, rows)
					})
				}
			}
			if sc.Order == "sub_issues_count" {
				// A live child created by another person does not count for a guest.
				for n, row := range sc.Rows {
					listSeedSQL(t, w, `INSERT INTO issues(id,workspace_id,project_id,state_id,sequence_id,name,created_by_id,parent_id)
						VALUES($1,$2,$3,$4,$5,'Hidden child',$6,$7)`, uuid.NewV7(), w.acme, w.web, w.todo, 2000+n, w.ids["alice"], row.ID)
				}
				listSeedSQL(t, w, "UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", w.web, w.ids["bob"])
				for _, order := range []string{"sub_issues_count", "-sub_issues_count"} {
					t.Run("own-only/"+order, func(t *testing.T) {
						want := sc.AscendingIDs
						if order[0] == '-' {
							want = sc.DescendingIDs
						}
						checkListPages(t, w, "bob", path, url.Values{"order_by": {order}, "sub_issue": {"false"}}, want, keys, rows)
					})
				}
			}
		})
	}
	if orders != 21 {
		t.Fatalf("executed %d orders, want 21 in each of four shapes", orders)
	}
}

func seedListOrder(t *testing.T, sc *listOrderSample) (issueWorld, map[uuid.UUID]json.RawMessage, map[uuid.UUID]listRowWant) {
	t.Helper()
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	w.newWorkItemBy(t, "walt", w.web, `{"name":"A only"}`)
	w.newWorkItemBy(t, "alice", w.web, `{"name":"C only"}`)
	backlog := stateID(t, w.pool, w.web, "Backlog")
	keys, wants := map[uuid.UUID]json.RawMessage{}, map[uuid.UUID]listRowWant{}
	labels := map[string]uuid.UUID{}
	for n, r := range sc.Rows {
		if sc.Order == "state_name" && string(r.Key) == "null" {
			continue // Every admitted state has a name.
		}
		listSeedSQL(t, w, `INSERT INTO issues(id,workspace_id,project_id,state_id,sequence_id,name,created_by_id)
			VALUES($1,$2,$3,$4,$5,'Fixed order',$6)`, r.ID, w.acme, w.web, backlog, 100+n, w.ids["bob"])
		keys[r.ID] = r.Key
		want := listRowWant{}
		switch sc.Order {
		case "state_name":
			state := uuid.NewV7()
			listSeedSQL(t, w, `INSERT INTO states(id,workspace_id,project_id,name,color,"group") VALUES($1,$2,$3,$4,'#000000','backlog')`, state, w.acme, w.web, r.Names[0])
			listSeedSQL(t, w, "UPDATE issues SET state_id=$2 WHERE id=$1", r.ID, state)
		case "label_name", "assignee_name":
			for _, name := range r.Names {
				target := uuid.NewV7()
				if sc.Order == "label_name" {
					// Labels are unique by PG lower(name). Keep the fixed relative
					// order, reuse Alpha, and disambiguate the lower-case umlaut.
					if name == "Alpha" {
						name = "ALPHA"
					}
					if name == "äBC" {
						name = "äBC second"
						keys[r.ID] = json.RawMessage(`"äbc second"`)
					}
					if id, exists := labels[name]; exists {
						target = id
					} else {
						listSeedSQL(t, w, "INSERT INTO labels(id,workspace_id,project_id,name) VALUES($1,$2,$3,$4)", target, w.acme, w.web, name)
						labels[name] = target
					}
					listSeedSQL(t, w, "INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id) VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), w.acme, w.web, r.ID, target)
					want.labels = append(want.labels, target)
				} else {
					listSeedSQL(t, w, "INSERT INTO users(id,email,password,display_name) VALUES($1,$2,'x',$3)", target, target.String()+"@example.com", name)
					listSeedSQL(t, w, "INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id) VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), w.acme, w.web, r.ID, target)
					want.assignees = append(want.assignees, target)
				}
			}
		case "created_at", "updated_at", "start_date", "target_date":
			var value *string
			if err := json.Unmarshal(r.Key, &value); err != nil {
				t.Fatal(err)
			}
			// Fixed fixture column, never input from the HTTP query.
			cast := "timestamptz"
			if sc.Order == "start_date" || sc.Order == "target_date" {
				cast = "date"
			} else {
				// Retain the JSON's fixed tie/order, placing its consecutive
				// microseconds across the exact-integer boundary of float64.
				value = new(map[string]string{
					"2026-01-01T00:00:00.123456Z": "2255-06-05T23:47:34.740992Z",
					"2026-01-01T00:00:00.123457Z": "2255-06-05T23:47:34.740993Z",
				}[*value])
				keys[r.ID] = json.RawMessage(strconv.Quote(*value))
			}
			listSeedSQL(t, w, "UPDATE issues SET "+sc.Order+"=$2::text::"+cast+" WHERE id=$1", r.ID, value)
		case "priority":
			var priority int
			if err := json.Unmarshal(r.Key, &priority); err != nil {
				t.Fatal(err)
			}
			listSeedSQL(t, w, "UPDATE issues SET priority=$2 WHERE id=$1", r.ID, []string{"none", "low", "medium", "high", "urgent"}[priority])
		case "sort_order":
			var value float64
			if err := json.Unmarshal(r.Key, &value); err != nil {
				t.Fatal(err)
			}
			listSeedSQL(t, w, "UPDATE issues SET sort_order=$2 WHERE id=$1", r.ID, value)
		case "link_count":
			if err := json.Unmarshal(r.Key, &want.links); err != nil {
				t.Fatal(err)
			}
			for link := int64(0); link <= want.links; link++ {
				listSeedSQL(t, w, `INSERT INTO issue_links(id,workspace_id,project_id,issue_id,url,created_by_id,deleted_at)
                    VALUES($1,$2,$3,$4,$5,$6,CASE WHEN $7 THEN now() END)`, uuid.NewV7(), w.acme, w.web, r.ID, fmt.Sprintf("https://example.com/%s/%d", r.ID, link), w.ids["alice"], link == want.links)
			}
		case "sub_issues_count":
			want.children = []int{0, 0, 1, 2}[n]
			keys[r.ID] = json.RawMessage(strconv.Itoa(want.children)) // Huge integers are tested by the actual SQL CASE probe.
			for child := 0; child < want.children+2; child++ {
				id := uuid.NewV7()
				listSeedSQL(t, w, `INSERT INTO issues(id,workspace_id,project_id,state_id,sequence_id,name,created_by_id,parent_id,archived_at,deleted_at)
					VALUES($1,$2,$3,$4,$5,'Child',$6,$7,CASE WHEN $8 THEN now() END,CASE WHEN $9 THEN now() END)`,
					id, w.acme, w.web, backlog, 1000+n*10+child, w.ids["bob"], r.ID, child == want.children, child == want.children+1)
			}
		}
		// A deleted relation must not remove the none group or alter a name key.
		listSeedSQL(t, w, "INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id,deleted_at) VALUES($1,$2,$3,$4,$5,now())", uuid.NewV7(), w.acme, w.web, r.ID, w.bug)
		wants[r.ID] = want
	}
	t.Logf("fixed %s oracle: %d admitted rows; 4 real entry/query shapes", sc.Order, len(keys))
	return w, keys, wants
}
