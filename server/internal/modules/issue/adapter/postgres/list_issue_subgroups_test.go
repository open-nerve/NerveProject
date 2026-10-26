package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Y=A/P, X1=A+C/P+Q, Z=C/Q, X2=A+C/P+Q, N=none/none.
// Counting expanded rows would give 11 overall, 5 per nonempty group/lane.
func subgroupFixture(t *testing.T) (*postgresadapter.Store, *pgxpool.Pool, app.ListRead, []uuid.UUID, []uuid.UUID, []uuid.UUID) {
	t.Helper()
	s, pool := newStore(t)
	a, b := newAccount(t, pool, "cell-a"), newAccount(t, pool, "cell-b")
	w := newWorkspace(t, pool, "cells")
	p := newProject(t, pool, w, "CELL")
	labels := []uuid.UUID{newLabel(t, pool, p, "A"), newLabel(t, pool, p, "C")}
	people := []uuid.UUID{a, b}
	slices.SortFunc(labels, func(a, b uuid.UUID) int { return a.Compare(b) })
	slices.SortFunc(people, func(a, b uuid.UUID) int { return a.Compare(b) })
	ids := make([]uuid.UUID, 5)
	for n := range ids {
		ids[n] = uuid.NewV7()
		newIssue(t, pool, p, ids[n], n+1, a)
		exec(t, pool, "UPDATE issues SET sort_order=$2 WHERE id=$1", ids[n], (n+1)*5)
		ls, ps := labels, people
		switch n {
		case 0:
			ls, ps = labels[:1], people[:1]
		case 2:
			ls, ps = labels[1:], people[1:]
		case 4:
			ls, ps = nil, nil
		}
		if err := s.InsertLabels(context.Background(), linkRows(p, ids[n], a, now, ls...)); err != nil {
			t.Fatal(err)
		}
		if err := s.InsertAssignees(context.Background(), linkRows(p, ids[n], a, now, ps...)); err != nil {
			t.Fatal(err)
		}
	}
	// A deleted relationship must not suppress none or create another cell.
	exec(t, pool, "INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id,deleted_at) VALUES($1,$2,$3,$4,$5,now())", uuid.NewV7(), w, p.id, ids[4], labels[0])
	exec(t, pool, "INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id,deleted_at) VALUES($1,$2,$3,$4,$5,now())", uuid.NewV7(), w, p.id, ids[4], people[0])
	in := app.ListRead{ActorID: a, Directories: app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}}, States: []app.ListState{{ID: p.state, Group: "backlog", Key: "backlog"}}}, Query: domain.ListQuery{Order: "sort_order", Limit: 1, GroupBy: "label_id", SubGroupBy: "assignee_id", Filters: domain.ListFilters{SubIssues: true}}}
	return s, pool, in, ids, labels, people
}

func TestIssueSubgroups(t *testing.T) {
	s, _, in, ids, labels, people := subgroupFixture(t)
	for _, limit := range []int{1, 100} {
		in.Query.Limit = limit
		rows, err := s.Subgroups(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		cells := []struct {
			group, lane string
			ids         []uuid.UUID
			gt, lt      int64
		}{
			{labels[0].String(), people[0].String(), []uuid.UUID{ids[0], ids[1], ids[3]}, 3, 3},
			{labels[0].String(), people[1].String(), []uuid.UUID{ids[1], ids[3]}, 3, 3},
			{labels[1].String(), people[0].String(), []uuid.UUID{ids[1], ids[3]}, 3, 3},
			{labels[1].String(), people[1].String(), []uuid.UUID{ids[1], ids[2], ids[3]}, 3, 3},
			{"none", "none", []uuid.UUID{ids[4]}, 1, 1},
		}
		wantRows := 11
		if limit == 1 {
			wantRows = 9
		}
		if len(rows) != wantRows {
			t.Fatalf("per-cell lookahead rows=%d want=%d", len(rows), wantRows)
		}
		for _, cell := range cells {
			size := min(len(cell.ids), limit+1)
			part := rows[:size]
			rows = rows[size:]
			checkFlatIDs(t, part, cell.ids[:size])
			for _, r := range part {
				if r.GroupKey != cell.group || r.SubGroupKey != cell.lane || r.Total != 5 || r.GroupTotal != cell.gt || r.SubgroupTotal != cell.lt || r.CellTotal != int64(len(cell.ids)) {
					t.Fatalf("cell counts %+v; want %+v", r, cell)
				}
				if r.Issue.LabelIDs == nil || r.Issue.AssigneeIDs == nil {
					t.Fatal("nil row arrays")
				}
				if (r.Issue.ID == ids[1] || r.Issue.ID == ids[3]) && (!slices.Equal(r.Issue.LabelIDs, labels) || !slices.Equal(r.Issue.AssigneeIDs, people)) {
					t.Fatalf("incomplete arrays %+v", r.Issue)
				}
			}
		}
	}
}

func TestIssueCellContinuation(t *testing.T) {
	s, _, in, ids, labels, people := subgroupFixture(t)
	for _, cell := range []struct {
		group, lane string
		want        []uuid.UUID
	}{
		{labels[0].String(), people[0].String(), []uuid.UUID{ids[0], ids[1], ids[3]}},
		{labels[0].String(), people[1].String(), []uuid.UUID{ids[1], ids[3]}},
		{labels[1].String(), people[0].String(), []uuid.UUID{ids[1], ids[3]}},
		{labels[1].String(), people[1].String(), []uuid.UUID{ids[1], ids[2], ids[3]}},
		{"none", "none", []uuid.UUID{ids[4]}},
		{labels[0].String(), "none", nil},
	} {
		in.Query.GroupKey, in.Query.SubGroupKey = &cell.group, &cell.lane
		checkCellPages(t, s, in, cell.want)
	}
	group, lane := labels[1].String(), people[1].String()
	in.Query.GroupKey, in.Query.SubGroupKey = &group, &lane
	for _, tc := range []struct {
		name              string
		labels, assignees []uuid.UUID
	}{{"label intersection", labels[:1], nil}, {"assignee intersection", nil, people[:1]}, {"both intersections", labels[:1], people[:1]}} {
		t.Run(tc.name, func(t *testing.T) {
			in.Query.Filters.Labels.IDs, in.Query.Filters.Assignees.IDs = tc.labels, tc.assignees
			checkCellPages(t, s, in, []uuid.UUID{ids[1], ids[3]})
		})
	}
}

func checkCellPages(t *testing.T, s *postgresadapter.Store, in app.ListRead, want []uuid.UUID) {
	t.Helper()
	in.Query.Limit, in.Query.Cursor = 1, nil
	catalog := subgroupCatalog{}
	for _, state := range in.Directories.States {
		catalog.states = append(catalog.states, app.StateRef{ID: state.ID, Name: state.Key, Group: state.Group})
	}
	listing := app.NewListing(app.ListingDeps{Rows: s, States: catalog, Labels: catalog, Assignees: s, Profiles: catalog, Issues: s})
	for n := 0; n <= len(want); n++ {
		rows, err := s.Subgroups(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != min(2, len(want)-n) {
			t.Fatalf("cell page %d rows=%d", n, len(rows))
		}
		page, err := listing.Read(context.Background(), app.ListingInput{ActorID: in.ActorID, Projects: in.Directories.Projects, Query: in.Query})
		if err != nil || page.Total != int64(len(want)) || page.Data == nil || len(page.Data) != min(1, len(want)-n) || page.Groups != nil || page.SubgroupTotals != nil {
			t.Fatalf("cell envelope page %d %+v %v", n, page, err)
		}
		if n == len(want) {
			if page.NextCursor != "" {
				t.Fatal("past-end cursor")
			}
			break
		}
		if page.Data[0].ID != want[n] || (page.NextCursor != "") != (n+1 < len(want)) {
			t.Fatalf("cell continuation page %d %+v", n, page)
		}
		if rows[0].Issue.ID != want[n] || rows[0].CellTotal != int64(len(want)) {
			t.Fatalf("cell page %d %+v; want %s total%d", n, rows[0], want[n], len(want))
		}
		encoded, err := domain.EncodeListCursor(rows[0].Position)
		if err != nil {
			t.Fatal(err)
		}
		c, err := domain.DecodeListCursor(encoded, in.Query.Order)
		if err != nil {
			t.Fatal(err)
		}
		in.Query.Cursor = &c
	}
}
