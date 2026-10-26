package postgresadapter_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Consume fixed expected IDs, never derive expected order with a comparator.
// Astronomical child counts belong to expression probes, not row seeding.
func TestFlatIssueListOrders(t *testing.T) {
	raw, err := os.ReadFile("../../../../../../api/testdata/issue-order.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Scenarios []struct {
			Order string
			Rows  []struct {
				ID    uuid.UUID
				Names []string
				Key   json.RawMessage
			}
			AscendingIDs  []uuid.UUID `json:"ascending_ids"`
			DescendingIDs []uuid.UUID `json:"descending_ids"`
		}
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, sc := range fixture.Scenarios {
		t.Run(sc.Order, func(t *testing.T) {
			if sc.Order == "sub_issues_count" {
				t.Skip("huge count keys are numeric probes; bounded counts tested separately")
			}
			s, pool := newStore(t)
			actor := newAccount(t, pool, "order")
			wid := newWorkspace(t, pool, "orders")
			p := newProject(t, pool, wid, "ORDER")
			dirs := app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}}, States: []app.ListState{{ID: p.state, Group: "backlog", Key: "backlog"}}}
			var absent uuid.UUID
			for n, r := range sc.Rows {
				if sc.Order == "state_name" && string(r.Key) == "null" {
					absent = r.ID
					continue
				} // An admitted state always has a name.
				if sc.Order == "label_name" {
					p = newProject(t, pool, wid, fmt.Sprintf("L%d", n))
					dirs.Projects = append(dirs.Projects, app.VisibleProject{ID: p.id})
					dirs.States = append(dirs.States, app.ListState{ID: p.state, Group: "backlog", Key: "backlog"})
				}
				newIssue(t, pool, p, r.ID, n+1, actor)
				switch sc.Order {
				case "state_name":
					state := uuid.NewV7()
					exec(t, pool, "INSERT INTO states(id,workspace_id,project_id,name,color) VALUES($1,$2,$3,$4,'#000000')", state, wid, p.id, r.Names[0])
					exec(t, pool, "UPDATE issues SET state_id=$2 WHERE id=$1", r.ID, state)
					dirs.States = append(dirs.States, app.ListState{ID: state, Group: "backlog", Key: domain.NameOrderKey(r.Names[0])})
				case "label_name", "assignee_name":
					for _, name := range r.Names {
						var target uuid.UUID
						table, column := "issue_labels", "label_id"
						if sc.Order == "label_name" {
							target = newLabel(t, pool, p, name)
							dirs.Labels = append(dirs.Labels, app.ListName{ID: target, Key: domain.NameOrderKey(name)})
						} else {
							target = newAccount(t, pool, uuid.NewV7().String())
							table, column = "issue_assignees", "assignee_id"
							dirs.Assignees = append(dirs.Assignees, app.ListName{ID: target, Key: domain.NameOrderKey(name)})
						}
						exec(t, pool, "INSERT INTO "+table+"(id,workspace_id,project_id,issue_id,"+column+") VALUES($1,$2,$3,$4,$5)", uuid.NewV7(), wid, p.id, r.ID, target)
					}
				case "created_at", "updated_at":
					var value string
					if err := json.Unmarshal(r.Key, &value); err != nil {
						t.Fatal(err)
					}
					at, err := time.Parse(time.RFC3339Nano, value)
					if err != nil {
						t.Fatal(err)
					}
					exec(t, pool, "UPDATE issues SET "+sc.Order+"=$2 WHERE id=$1", r.ID, at)
				case "start_date", "target_date":
					var value *string
					if err := json.Unmarshal(r.Key, &value); err != nil {
						t.Fatal(err)
					}
					exec(t, pool, "UPDATE issues SET "+sc.Order+"=$2::text::date WHERE id=$1", r.ID, value)
				case "priority":
					var value int
					if err := json.Unmarshal(r.Key, &value); err != nil {
						t.Fatal(err)
					}
					exec(t, pool, "UPDATE issues SET priority=$2 WHERE id=$1", r.ID, []string{"none", "low", "medium", "high", "urgent"}[value])
				case "sort_order":
					var value float64
					if err := json.Unmarshal(r.Key, &value); err != nil {
						t.Fatal(err)
					}
					exec(t, pool, "UPDATE issues SET sort_order=$2 WHERE id=$1", r.ID, value)
				}
			}
			for _, direction := range []string{"", "-"} {
				if direction == "-" && sc.Order == "sort_order" {
					continue
				}
				t.Run(direction+sc.Order, func(t *testing.T) {
					want := sc.AscendingIDs
					if direction == "-" {
						want = sc.DescendingIDs
					}
					want = slices.DeleteFunc(slices.Clone(want), func(id uuid.UUID) bool { return id == absent })
					in := app.ListRead{ActorID: actor, Directories: dirs, Query: domain.ListQuery{Order: domain.ListOrder(direction + sc.Order), Limit: 100, Filters: domain.ListFilters{SubIssues: true}}}
					got, err := s.Flat(context.Background(), in)
					if err != nil {
						t.Fatal(err)
					}
					checkFlatIDs(t, got, want)
					checkFlatPages(t, s, in, want)
				})
			}
		})
	}
}

func TestFlatIssueListCountOrder(t *testing.T) {
	s, pool := newStore(t)
	a := newAccount(t, pool, "counts")
	wid := newWorkspace(t, pool, "count-order")
	p := newProject(t, pool, wid, "COUNT")
	ids := []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000003"), uuid.MustParse("00000000-0000-0000-0000-000000000002"), uuid.MustParse("00000000-0000-0000-0000-000000000001")}
	for n, id := range ids {
		newIssue(t, pool, p, id, n+1, a)
		for child := 0; child < n; child++ {
			cid := uuid.NewV7()
			newIssue(t, pool, p, cid, 10+n*10+child, a)
			exec(t, pool, "UPDATE issues SET parent_id=$2 WHERE id=$1", cid, id)
		}
	}
	for _, order := range []domain.ListOrder{"sub_issues_count", "-sub_issues_count"} {
		in := app.ListRead{ActorID: a, Directories: app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}}, States: []app.ListState{{ID: p.state, Group: "backlog", Key: "b"}}}, Query: domain.ListQuery{Order: order, Limit: 100}}
		got, err := s.Flat(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		want := slices.Clone(ids)
		if order[0] == '-' {
			slices.Reverse(want)
		}
		checkFlatIDs(t, got, want)
		checkFlatPages(t, s, in, want)
	}
}
