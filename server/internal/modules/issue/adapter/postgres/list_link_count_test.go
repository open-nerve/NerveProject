package postgresadapter_test

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Both pages must use the last emitted row (not the lookahead) as the public
// position. The equal-count pair deliberately opposes insertion/UUID order.
func checkLinkCountPages(t *testing.T, shape string) {
	t.Helper()
	s, pool := newStore(t)
	actor := newAccount(t, pool, "link-count")
	workspace := newWorkspace(t, pool, "link-count")
	p := newProject(t, pool, workspace, "LINKCOUNT")
	ids := []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000004"), uuid.MustParse("00000000-0000-0000-0000-000000000003"), uuid.MustParse("00000000-0000-0000-0000-000000000002"), uuid.MustParse("00000000-0000-0000-0000-000000000001")}
	counts := []int64{0, 1, 2, 2}
	for n, id := range ids {
		newIssue(t, pool, p, id, n+1, actor)
		for link := int64(0); link <= counts[n]; link++ {
			exec(t, pool, `INSERT INTO issue_links(id,workspace_id,project_id,issue_id,url,deleted_at) VALUES($1,$2,$3,$4,$5,CASE WHEN $6 THEN now() END)`, uuid.NewV7(), workspace, p.id, id, fmt.Sprintf("https://example.com/%s/%d", id, link), link == counts[n])
		}
	}
	in := app.ListRead{ActorID: actor, Directories: app.ListDirectories{Projects: []app.VisibleProject{{ID: p.id}}, States: []app.ListState{{ID: p.state, Group: "backlog", Key: "backlog"}}}, Query: domain.ListQuery{Limit: 2}}
	read := s.Flat
	if shape != "flat" {
		in.Query.GroupBy = "label_id"
		read = s.Groups
	}
	if shape == "subgroups" || shape == "cell" {
		in.Query.SubGroupBy = "assignee_id"
		read = s.Subgroups
	}
	for _, order := range []domain.ListOrder{"link_count", "-link_count"} {
		t.Run(string(order), func(t *testing.T) {
			in.Query.Order = order
			in.Query.Cursor = nil
			in.Query.GroupKey = nil
			in.Query.SubGroupKey = nil
			want := []uuid.UUID{ids[0], ids[1], ids[3], ids[2]}
			if order[0] == '-' {
				slices.Reverse(want)
			}
			// First request exercises the nested, unselected group/cell projection.
			var seen []uuid.UUID
			for start := 0; start <= len(want); start += 2 {
				rows, err := read(context.Background(), in)
				if err != nil {
					t.Fatal(err)
				}
				if len(rows) != min(3, len(want)-start) {
					t.Fatalf("%s page %d rows=%d", shape, start, len(rows))
				}
				var next string
				if shape == "cell" {
					next = checkLinkCellEnvelope(t, s, in, want[start:], ids, counts)
				}
				if start == len(want) {
					break
				}
				take := min(2, len(rows))
				checkFlatIDs(t, rows[:take], want[start:start+take])
				for _, r := range rows[:take] {
					if r.Total != 4 || (shape != "flat" && (r.GroupTotal != 4 || r.GroupKey != "none")) || ((shape == "subgroups" || shape == "cell") && (r.SubgroupTotal != 4 || r.CellTotal != 4 || r.SubGroupKey != "none")) {
						t.Fatal("lost totals/selection", r)
					}
					n := slices.Index(ids, r.Issue.ID)
					if r.Issue.LinkCount != counts[n] || string(r.Position.Key) != fmt.Sprint(counts[n]) {
						t.Fatalf("%s position %+v want %d", shape, r.Position, counts[n])
					}
					seen = append(seen, r.Issue.ID)
				}
				encoded, err := domain.EncodeListCursor(rows[take-1].Position)
				if err != nil {
					t.Fatal(err)
				}
				if next != "" {
					encoded = next
				}
				cursor, err := domain.DecodeListCursor(encoded, order)
				if err != nil {
					t.Fatal(err)
				}
				in.Query.Cursor = &cursor
				if shape != "flat" {
					in.Query.GroupKey = new("none")
				}
				if shape == "subgroups" || shape == "cell" {
					in.Query.SubGroupKey = new("none")
				}
			}
			if !slices.Equal(seen, want) {
				t.Fatal("pagination duplicated or omitted IDs", seen, want)
			}
			// The terminal cursor must retain its integer spelling across JSON.
			if len(in.Query.Cursor.Key) == 0 || !json.Valid(in.Query.Cursor.Key) {
				t.Fatal("missing terminal key")
			}
		})
	}
}

// Consume the initial nested cell's real cursor, then the selected cell's
// envelope; a row lookahead must never become the public continuation position.
func checkLinkCellEnvelope(t *testing.T, s *postgresadapter.Store, in app.ListRead, want, ids []uuid.UUID, counts []int64) string {
	t.Helper()
	catalog := subgroupCatalog{}
	for _, state := range in.Directories.States {
		catalog.states = append(catalog.states, app.StateRef{ID: state.ID, Name: state.Key, Group: state.Group})
	}
	listing := app.NewListing(app.ListingDeps{Rows: s, States: catalog, Labels: catalog, Assignees: s, Profiles: catalog, Issues: s})
	page, err := listing.Read(context.Background(), app.ListingInput{ActorID: in.ActorID, Projects: in.Directories.Projects, Query: in.Query})
	if err != nil || page.Total != 4 {
		t.Fatal("cell page total", page, err)
	}
	data, next := page.Data, page.NextCursor
	if in.Query.GroupKey == nil {
		if len(page.Groups) != 1 || page.Groups[0].Key != "none" || page.Groups[0].Total != 4 || len(page.Groups[0].Subgroups) != 1 || len(page.SubgroupTotals) != 1 || page.SubgroupTotals[0].Key != "none" || page.SubgroupTotals[0].Total != 4 {
			t.Fatal("nested cell", page)
		}
		cell := page.Groups[0].Subgroups[0]
		if cell.Key != "none" || cell.Total != 4 {
			t.Fatal("cell totals", cell)
		}
		data, next = cell.Data, cell.NextCursor
	} else if page.Groups != nil || page.SubgroupTotals != nil {
		t.Fatal("selected cell envelope", page)
	}
	if len(data) != min(2, len(want)) || (next != "") != (len(want) > 2) {
		t.Fatal("cell data/lookahead", page)
	}
	for n, row := range data {
		if row.ID != want[n] || row.LinkCount != counts[slices.Index(ids, row.ID)] {
			t.Fatal("cell row", row, want)
		}
	}
	if next != "" {
		cursor, err := domain.DecodeListCursor(next, in.Query.Order)
		last := data[len(data)-1]
		if err != nil || cursor.ID != last.ID || string(cursor.Key) != fmt.Sprint(counts[slices.Index(ids, last.ID)]) {
			t.Fatal("cell cursor", cursor, err)
		}
	}
	return next
}
