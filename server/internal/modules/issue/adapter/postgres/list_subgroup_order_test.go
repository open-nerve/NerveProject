package postgresadapter_test

import (
	"context"
	"math"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Only the cross-module catalog boundary is supplied here. Listing and its
// Store, generated SQL, row mapping, trimming and lane sorting are real.
type subgroupCatalog struct{ states []app.StateRef }

func (c subgroupCatalog) ListStates(context.Context, []uuid.UUID) ([]app.StateRef, error) {
	return c.states, nil
}
func (subgroupCatalog) ListLabels(context.Context, []uuid.UUID) ([]app.LabelRef, error) {
	return nil, nil
}
func (subgroupCatalog) DisplayNames(context.Context, []uuid.UUID) (map[uuid.UUID]string, error) {
	return nil, nil
}

func TestIssueSubgroupsLaneOrder(t *testing.T) {
	s, pool := newStore(t)
	a := newAccount(t, pool, "lane-order")
	w := newWorkspace(t, pool, "lane-order")
	p := newProject(t, pool, w, "LORDER")
	labels := []uuid.UUID{newLabel(t, pool, p, "first"), newLabel(t, pool, p, "later")}
	slices.SortFunc(labels, func(a, b uuid.UUID) int { return a.Compare(b) })
	people := []uuid.UUID{a, newAccount(t, pool, "lane-b"), newAccount(t, pool, "lane-c"), newAccount(t, pool, "lane-d")}
	slices.SortFunc(people, func(a, b uuid.UUID) int { return a.Compare(b) })
	priorities := []string{"urgent", "high", "medium", "low", "none"}
	groups := []string{"backlog", "unstarted", "started", "completed", "cancelled"}
	states := make([]uuid.UUID, 5)
	for n := range states {
		states[n] = uuid.NewV7()
	}
	slices.SortFunc(states, func(a, b uuid.UUID) int { return a.Compare(b) })
	catalog := subgroupCatalog{}
	for n, state := range states {
		exec(t, pool, "INSERT INTO states(id,workspace_id,project_id,name,color,\"group\") VALUES($1,$2,$3,$4,'#000000',$4)", state, w, p.id, groups[n])
		catalog.states = append(catalog.states, app.StateRef{ID: state, Name: groups[n], Group: domain.StateGroup(groups[n])})
		id := uuid.NewV7()
		newIssue(t, pool, p, id, n+1, a)
		var creator any
		if n < 4 {
			creator = people[n]
			if err := s.InsertAssignees(context.Background(), linkRows(p, id, a, now, people[n])); err != nil {
				t.Fatal(err)
			}
			label := labels[1]
			if n == 3 {
				label = labels[0]
			}
			if err := s.InsertLabels(context.Background(), linkRows(p, id, a, now, label)); err != nil {
				t.Fatal(err)
			}
		}
		exec(t, pool, "UPDATE issues SET state_id=$2,priority=$3,target_date=$4,created_by_id=$5 WHERE id=$1", id, state, priorities[n], []any{"2026-01-01", "2026-01-02", "2026-01-03", "2026-01-04", nil}[n], creator)
	}
	listing := app.NewListing(app.ListingDeps{Rows: s, States: catalog, Labels: catalog, Assignees: s, Profiles: catalog, Issues: s})
	stateKeys, personKeys := make([]string, 5), make([]string, 0, 5)
	for n, state := range states {
		stateKeys[n] = state.String()
	}
	for _, person := range people {
		personKeys = append(personKeys, person.String())
	}
	personKeys = append(personKeys, "none")
	for _, tc := range []struct {
		field domain.ListGroup
		want  []string
	}{
		{"priority", priorities}, {"state_group", groups}, {"state_id", stateKeys},
		{"target_date", []string{"2026-01-01", "2026-01-02", "2026-01-03", "2026-01-04", "none"}},
		{"assignee_id", personKeys}, {"created_by_id", personKeys},
		{"label_id", []string{labels[0].String(), labels[1].String(), "none"}},
	} {
		t.Run(string(tc.field), func(t *testing.T) {
			input := app.ListingInput{ActorID: a, Projects: []app.VisibleProject{{ID: p.id}}, Query: domain.ListQuery{Order: "sort_order", Limit: 1, GroupBy: "label_id", SubGroupBy: tc.field, Filters: domain.ListFilters{SubIssues: true}}}
			if tc.field == "label_id" {
				input.Query.GroupBy = "priority"
			}
			page, err := listing.Read(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			if page.Total != 5 || len(page.SubgroupTotals) != len(tc.want) || len(page.Groups) == 0 {
				t.Fatalf("lane envelope %+v", page)
			}
			// The earliest lane is deliberately missing from the first group.
			if page.Groups[0].Subgroups[0].Key == tc.want[0] {
				t.Fatal("fixture does not distinguish first-seen from global lane order")
			}
			for n, want := range tc.want {
				total := int64(1)
				if tc.field == "label_id" && n == 1 {
					total = 3
				}
				if page.SubgroupTotals[n].Key != want || page.SubgroupTotals[n].Total != total {
					t.Fatalf("lane %d %+v want %s/%d", n, page.SubgroupTotals[n], want, total)
				}
			}
			// A group containing three early lanes must also use the field's
			// fixed order in SQL, independently of the top-level mapper sort.
			if tc.field != "label_id" {
				cells := page.Groups[1].Subgroups
				if len(cells) != 3 {
					t.Fatalf("early cells %+v", cells)
				}
				for n, cell := range cells {
					if cell.Key != tc.want[n] || cell.Total != 1 || len(cell.Data) != 1 || cell.NextCursor != "" {
						t.Fatalf("cell %d %+v want key %s", n, cell, tc.want[n])
					}
				}
			}
		})
	}
}

func TestIssueSubgroupsPrecision(t *testing.T) {
	s, pool := newStore(t)
	a := newAccount(t, pool, "cell-precision")
	w := newWorkspace(t, pool, "cell-precision")
	p := newProject(t, pool, w, "CPREC")
	ids := []uuid.UUID{uuid.MustParse("00000000-0000-4000-8000-000000000003"), uuid.MustParse("00000000-0000-4000-8000-000000000002"), uuid.MustParse("00000000-0000-4000-8000-000000000001")}
	for n, id := range ids {
		newIssue(t, pool, p, id, n+1, a)
		exec(t, pool, "UPDATE issues SET sort_order=$2,created_at=$3::timestamptz,updated_at=$3::timestamptz,target_date=$4::date WHERE id=$1", id, []float64{1, math.Nextafter(1, 2), math.Nextafter(math.Nextafter(1, 2), 2)}[n], []string{"2255-06-05 23:47:34.740992Z", "2255-06-05 23:47:34.740993Z", "2255-06-05 23:47:34.740994Z"}[n], []any{"2026-01-01", "2026-01-02", nil}[n])
	}
	key := "none"
	in := app.ListRead{ActorID: a, Directories: app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}}, States: []app.ListState{{ID: p.state, Group: "backlog", Key: "backlog"}}}, Query: domain.ListQuery{GroupBy: "label_id", GroupKey: &key, SubGroupBy: "assignee_id", SubGroupKey: &key, Limit: 1, Filters: domain.ListFilters{SubIssues: true}}}
	for _, tc := range []struct {
		order domain.ListOrder
		want  []uuid.UUID
	}{
		{"sort_order", ids}, {"created_at", ids}, {"updated_at", ids},
		{"-created_at", []uuid.UUID{ids[2], ids[1], ids[0]}}, {"-updated_at", []uuid.UUID{ids[2], ids[1], ids[0]}},
		{"target_date", ids}, {"-target_date", []uuid.UUID{ids[1], ids[0], ids[2]}},
	} {
		t.Run(string(tc.order), func(t *testing.T) {
			in.Query.Order = tc.order
			checkCellPages(t, s, in, tc.want)
		})
	}
}
