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
	// Flat, parent read, or empty-page total query must hit the deadline.
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
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
