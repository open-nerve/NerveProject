package postgresadapter_test

import (
	"context"
	"fmt"
	"math"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

type attachmentListCatalog struct {
	subgroupCatalog
	counts   []app.AttachmentCount
	projects []uuid.UUID
	calls    int
}

func (c *attachmentListCatalog) ProjectAttachmentCounts(_ context.Context, ids []uuid.UUID, after uuid.UUID, size int) ([]app.AttachmentCount, error) {
	if !slices.Equal(ids, c.projects) || size != 1000 {
		return nil, fmt.Errorf("wrong catalog scope or size")
	}
	c.calls++
	index := slices.IndexFunc(c.counts, func(row app.AttachmentCount) bool { return row.IssueID.Compare(after) > 0 })
	if index < 0 {
		return nil, nil
	}
	return c.counts[index:min(index+size, len(c.counts))], nil
}

func (c *attachmentListCatalog) CountIssueAttachments(_ context.Context, ids []uuid.UUID) ([]app.AttachmentCount, error) {
	rows := make([]app.AttachmentCount, len(ids))
	for n, id := range ids {
		rows[n].IssueID = id
		for _, count := range c.counts {
			if count.IssueID == id {
				rows[n].Count = count.Count
				break
			}
		}
	}
	return rows, nil
}

// Only the asset catalog is supplied. Real Listing, SQL, row projection and
// public cursors consume 1001 entries; enormous counts need no imaginary files.
func TestListAttachmentCountFullDirectory(t *testing.T) {
	s, pool := newStore(t)
	actor := newAccount(t, pool, "attachment-list")
	wid := newWorkspace(t, pool, "attachment-list")
	p := newProject(t, pool, wid, "ATTACHLIST")
	catalog := &attachmentListCatalog{subgroupCatalog: subgroupCatalog{states: []app.StateRef{{ID: p.state, Group: "backlog", Name: "Backlog"}}}, projects: []uuid.UUID{p.id}}
	var ids []uuid.UUID
	for n := 1; n <= 1002; n++ {
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012x", n))
		ids = append(ids, id)
		newIssue(t, pool, p, id, n, actor)
		if n == 1 {
			continue
		}
		count := int64(1)
		switch n {
		case 998:
			count = 2
		case 999:
			count = 9007199254740992
		case 1000, 1001:
			count = 9007199254740993
		case 1002:
			count = math.MaxInt64
		}
		catalog.counts = append(catalog.counts, app.AttachmentCount{IssueID: id, Count: count})
	}
	listing := app.NewListing(app.ListingDeps{Attachments: catalog, AttachmentCatalog: catalog, Rows: s, States: catalog, Labels: catalog, Assignees: s, Profiles: catalog, Issues: s})
	for _, shape := range []int{0, 1, 2} {
		for _, order := range []domain.ListOrder{"attachment_count", "-attachment_count"} {
			t.Run(fmt.Sprint(shape)+string(order), func(t *testing.T) {
				in := app.ListingInput{ActorID: actor, Projects: []app.VisibleProject{{ID: p.id}}, Query: domain.ListQuery{Order: order, Limit: 2}}
				if shape > 0 {
					in.Query.GroupBy = "label_id"
					in.Query.GroupKey = new("none")
				}
				if shape > 1 {
					in.Query.SubGroupBy = "assignee_id"
					in.Query.SubGroupKey = new("none")
				}
				want := slices.Clone(ids)
				if order[0] == '-' {
					slices.Reverse(want)
				}
				for start := 0; start < 6; start += 2 {
					before := catalog.calls
					page, err := listing.Read(t.Context(), in)
					if err != nil || catalog.calls-before != 2 || page.Total != 1002 || len(page.Data) != 2 || page.NextCursor == "" {
						t.Fatalf("full catalog page %+v %v calls=%d", page, err, catalog.calls-before)
					}
					for n, row := range page.Data {
						if row.ID != want[start+n] {
							t.Fatalf("ranked page %d row %s want %s", start, row.ID, want[start+n])
						}
					}
					cursor, err := domain.DecodeListCursor(page.NextCursor, order)
					last := page.Data[1]
					if err != nil || cursor.ID != last.ID || string(cursor.Key) != fmt.Sprint(last.AttachmentCount) {
						t.Fatalf("rounded public cursor %+v %v", cursor, err)
					}
					in.Query.Cursor = &cursor
				}
			})
		}
	}
}
