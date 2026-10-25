package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	issuepostgres "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestListingHistoricalAssignees(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	hidden := w.newWorkItemIn(t, w.web, `{"name":"Hidden parent"}`)
	former := w.newWorkItemBy(t, "walt", w.web, fmt.Sprintf(`{"name":"Former","parent_id":%q,"assignee_ids":%s}`, hidden, idArray(w.ids["bob"])))
	guest := w.newWorkItemBy(t, "walt", w.web, fmt.Sprintf(`{"name":"Guest","assignee_ids":%s}`, idArray(w.ids["walt"])))
	none := w.newWorkItemBy(t, "walt", w.web, `{"name":"Unassigned"}`)
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{"UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2", []any{w.web, w.ids["bob"]}},
		{"UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", []any{w.web, w.ids["walt"]}},
		{"UPDATE users SET is_active=false,first_name='Wrong source',display_name='Alpha Former' WHERE id=$1", []any{w.ids["bob"]}},
		{"UPDATE users SET display_name='Zulu Guest' WHERE id=$1", []any{w.ids["walt"]}},
	} {
		if _, err := w.pool.Exec(context.Background(), statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	// A single connection is held by the snapshot. Escaping ctx in any catalog,
	// Flat/Groups, parent read, or empty-page total query must hit the deadline.
	one := poolOfOne(t, w.pool.Config().ConnString())
	projects := issueProjects{projects: project.Provide(one).IssueProjects}
	store := issuepostgres.New(one)
	listing := issueapp.NewListing(issueapp.ListingDeps{Rows: store, States: projects, Labels: projects, Assignees: store, Profiles: issueProfiles{profiles: identity.Provide(one).PublicProfiles}, Issues: store})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := postgres.NewTxManager(one, 3*time.Second, postgres.ReadOnlySnapshot()).WithinTx(ctx, func(ctx context.Context) error {
		var isolation, readOnly string
		if err := postgres.DB(ctx, one).QueryRow(ctx, "SELECT current_setting('transaction_isolation'),current_setting('transaction_read_only')").Scan(&isolation, &readOnly); err != nil {
			return err
		}
		if isolation != "repeatable read" || readOnly != "on" {
			t.Fatalf("snapshot %s/%s", isolation, readOnly)
		}
		visible, err := projects.Visible(ctx, w.acme, w.ids["walt"])
		if err != nil {
			return err
		}
		if len(visible) != 1 || !visible[0].OwnOnly {
			t.Fatalf("guest scope %+v", visible)
		}
		for _, direction := range []string{"", "-"} {
			input := issueapp.ListingInput{ActorID: w.ids["walt"], Projects: visible, Query: issuedomain.ListQuery{Order: issuedomain.ListOrder(direction + "assignee_name"), Limit: 1, Filters: issuedomain.ListFilters{SubIssues: true}}}
			want := []uuid.UUID{former, guest, none}
			if direction == "-" {
				want = []uuid.UUID{guest, former, none}
			}
			var seen []uuid.UUID
			for n := range want {
				page, err := listing.Read(ctx, input)
				if err != nil {
					return err
				}
				if page.Total != 3 || len(page.Data) != 1 || page.Data[0].ID != want[n] {
					t.Fatalf("historical assignee page %d: %+v; want %s", n, page, want[n])
				}
				item := page.Data[0]
				if item.ID == former && (item.ParentID != nil || !slices.Equal(item.AssigneeIDs, []uuid.UUID{w.ids["bob"]})) {
					t.Fatalf("former assignee/hidden parent %+v", item)
				}
				seen = append(seen, item.ID)
				if n == len(want)-1 {
					if page.NextCursor != "" {
						t.Fatal("terminal cursor")
					}
					break
				}
				c, err := issuedomain.DecodeListCursor(page.NextCursor, input.Query.Order)
				if err != nil {
					return err
				}
				input.Query.Cursor = &c
			}
			if !slices.Equal(seen, want) {
				t.Fatalf("historical order %v", seen)
			}
			// An explicit cursor beyond the final null row forces the same snapshot
			// total query. A restrictive original filter must survive that retry.
			input.Query.Filters.Assignees = issuedomain.IDFilter{IDs: []uuid.UUID{w.ids["bob"]}}
			input.Query.Cursor = &issuedomain.ListCursor{Order: input.Query.Order, Key: []byte("null"), ID: none}
			empty, err := listing.Read(ctx, input)
			if err != nil {
				return err
			}
			if empty.Data == nil || len(empty.Data) != 0 || empty.Total != 1 || empty.NextCursor != "" {
				t.Fatalf("filtered past end %+v", empty)
			}
		}
		t.Run("group pages and empty group cursor", func(t *testing.T) {
			input := issueapp.ListingInput{ActorID: w.ids["walt"], Projects: visible, Query: issuedomain.ListQuery{Order: "assignee_name", GroupBy: "assignee_id", Limit: 1, Filters: issuedomain.ListFilters{SubIssues: true}}}
			page, err := listing.Read(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			if page.Total != 3 || page.Data != nil || len(page.Groups) != 3 {
				t.Fatalf("group page %+v", page)
			}
			for _, group := range page.Groups {
				if group.Total != 1 || len(group.Data) != 1 || group.NextCursor != "" {
					t.Fatalf("group %+v", group)
				}
				if group.Data[0].ID == former && (group.Data[0].ParentID != nil || !slices.Equal(group.Data[0].AssigneeIDs, []uuid.UUID{w.ids["bob"]})) {
					t.Fatalf("group parent/array %+v", group.Data[0])
				}
			}
			// A real multi-row group emits its own cursor; continuing it returns
			// the flat envelope with that group's unchanged total.
			input.Query.GroupBy = "created_by_id"
			page, err = listing.Read(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Groups) != 1 || page.Groups[0].Total != 3 || len(page.Groups[0].Data) != 1 || page.Groups[0].Data[0].ID != former || page.Groups[0].NextCursor == "" {
				t.Fatalf("multi-row group %+v", page)
			}
			c, err := issuedomain.DecodeListCursor(page.Groups[0].NextCursor, input.Query.Order)
			if err != nil {
				t.Fatal(err)
			}
			selected := w.ids["walt"].String()
			input.Query.GroupKey = &selected
			input.Query.Cursor = &c
			continued, err := listing.Read(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			if continued.Total != 3 || continued.Groups != nil || len(continued.Data) != 1 || continued.Data[0].ID != guest || continued.NextCursor == "" {
				t.Fatalf("continued group %+v", continued)
			}
			// The selected none-date group is intersected with the original
			// assignee filter, then retried under this same occupied connection.
			key := "none"
			input.Query.GroupBy = "target_date"
			input.Query.GroupKey = &key
			input.Query.Filters.Assignees = issuedomain.IDFilter{IDs: []uuid.UUID{w.ids["bob"]}}
			input.Query.Cursor = &issuedomain.ListCursor{Order: input.Query.Order, Key: []byte("null"), ID: none}
			empty, err := listing.Read(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			if empty.Total != 1 || empty.Groups != nil || empty.Data == nil || len(empty.Data) != 0 || empty.NextCursor != "" {
				t.Fatalf("group past end %+v", empty)
			}
			key = "2026-01-01"
			input.Query.GroupKey = &key
			empty, err = listing.Read(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			if empty.Total != 0 || len(empty.Data) != 0 {
				t.Fatalf("absent selected group %+v", empty)
			}
		})
		t.Run("cell pages and same-snapshot totals", func(t *testing.T) {
			input := issueapp.ListingInput{ActorID: w.ids["walt"], Projects: visible, Query: issuedomain.ListQuery{Order: "assignee_name", GroupBy: "priority", SubGroupBy: "target_date", Limit: 1, Filters: issuedomain.ListFilters{SubIssues: true}}}
			page, err := listing.Read(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			if page.Total != 3 || page.Data != nil || len(page.Groups) != 1 || len(page.SubgroupTotals) != 1 || page.SubgroupTotals[0].Key != "none" || page.SubgroupTotals[0].Total != 3 {
				t.Fatalf("lane envelope %+v", page)
			}
			group := page.Groups[0]
			if group.Key != "none" || group.Total != 3 || group.Data != nil || len(group.Subgroups) != 1 {
				t.Fatalf("lane group %+v", group)
			}
			cell := group.Subgroups[0]
			if cell.Key != "none" || cell.Total != 3 || len(cell.Data) != 1 || cell.Data[0].ID != former || cell.Data[0].ParentID != nil || !slices.Equal(cell.Data[0].AssigneeIDs, []uuid.UUID{w.ids["bob"]}) {
				t.Fatalf("cell parent/array %+v", cell)
			}
			c, err := issuedomain.DecodeListCursor(cell.NextCursor, input.Query.Order)
			if err != nil {
				t.Fatal(err)
			}
			key := "none"
			input.Query.GroupKey, input.Query.SubGroupKey, input.Query.Cursor = &key, &key, &c
			continued, err := listing.Read(ctx, input)
			if err != nil || continued.Total != 3 || continued.Groups != nil || continued.SubgroupTotals != nil || len(continued.Data) != 1 || continued.Data[0].ID != guest || continued.NextCursor == "" {
				t.Fatalf("continued cell %+v %v", continued, err)
			}
			input.Query.Filters.Assignees = issuedomain.IDFilter{IDs: []uuid.UUID{w.ids["bob"]}}
			input.Query.Filters.Target = []issuedomain.DateFilter{{None: true}}
			input.Query.Cursor = &issuedomain.ListCursor{Order: input.Query.Order, Key: []byte("null"), ID: none}
			empty, err := listing.Read(ctx, input)
			if err != nil || empty.Total != 1 || empty.Data == nil || len(empty.Data) != 0 || empty.NextCursor != "" || empty.Groups != nil || empty.SubgroupTotals != nil {
				t.Fatalf("filtered cell past end %+v %v", empty, err)
			}
			missing := "2026-01-01"
			input.Query.SubGroupKey = &missing
			empty, err = listing.Read(ctx, input)
			if err != nil || empty.Total != 0 || empty.Data == nil || len(empty.Data) != 0 {
				t.Fatalf("absent cell %+v %v", empty, err)
			}
		})

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
