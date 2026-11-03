package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

type issueCountProbe struct {
	t     *testing.T
	ctx   context.Context
	calls [][]uuid.UUID
	rows  []app.AttachmentCount
	err   error
}

func (p *issueCountProbe) CountIssueAttachments(ctx context.Context, ids []uuid.UUID) ([]app.AttachmentCount, error) {
	p.t.Helper()
	if ctx != p.ctx {
		p.t.Fatal("attachment count escaped caller snapshot")
	}
	p.calls = append(p.calls, slices.Clone(ids))
	return slices.Clone(p.rows), p.err
}
func listingWithCounts(f *listingPorts, counts app.AttachmentCounter) *app.Listing {
	return app.NewListing(app.ListingDeps{Attachments: counts, Rows: f, States: f, Labels: f, Assignees: f, Profiles: f, Issues: f})
}
func TestIssueAttachmentProjectionBatch(t *testing.T) {
	for _, shape := range []string{"flat", "groups", "subgroups", "empty"} {
		t.Run(shape, func(t *testing.T) {
			_, f, in := listingFixture(t)
			first, second, look := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			counts := &issueCountProbe{t: t, ctx: f.ctx, rows: []app.AttachmentCount{{IssueID: first, Count: 9}, {IssueID: second, Count: 0}}}
			switch shape {
			case "flat":
				in.Query.Limit = 2
				f.pages = [][]app.ListRecord{{listRecord(first, nil), listRecord(second, nil), listRecord(look, nil)}}
			case "groups":
				in.Query.GroupBy = "label_id"
				f.pages = [][]app.ListRecord{{groupRecord(first, "a", 2, nil), groupRecord(look, "a", 2, nil), groupRecord(second, "b", 1, nil), groupRecord(first, "c", 1, nil)}}
			case "subgroups":
				in.Query.GroupBy = "label_id"
				in.Query.SubGroupBy = "priority"
				f.pages = [][]app.ListRecord{{cellRecord(first, "a", "low", nil), cellRecord(look, "a", "low", nil), cellRecord(second, "b", "high", nil), cellRecord(first, "b", "low", nil)}}
			case "empty":
				f.pages = [][]app.ListRecord{nil}
			}
			page, err := listingWithCounts(f, counts).Read(f.ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if shape == "empty" {
				if len(counts.calls) != 0 {
					t.Fatal("empty output queried assets")
				}
				return
			}
			wantIDs := sorted([]uuid.UUID{first, second})
			if len(counts.calls) != 1 || !slices.Equal(counts.calls[0], wantIDs) {
				t.Fatalf("calls %v want one unique output batch %v", counts.calls, wantIDs)
			}
			items := slices.Clone(page.Data)
			for _, g := range page.Groups {
				items = append(items, g.Data...)
				for _, c := range g.Subgroups {
					items = append(items, c.Data...)
				}
			}
			if len(items) < 2 {
				t.Fatal("empty projection fixture")
			}
			for _, i := range items {
				want, ok := map[uuid.UUID]int64{first: 9, second: 0}[i.ID]
				if !ok || i.AttachmentCount != want {
					t.Fatalf("projection %+v", i)
				}
			}
			for _, r := range f.pages[0] {
				if r.Issue.AttachmentCount != 0 {
					t.Fatal("mutated repository projection")
				}
			}
		})
	}
}
func TestIssueAttachmentProjectionRefusesBrokenCounter(t *testing.T) {
	for _, fault := range []string{"missing", "duplicate", "extra", "negative", "failure", "unauthorized"} {
		t.Run(fault, func(t *testing.T) {
			_, f, in := listingFixture(t)
			id := uuid.NewV7()
			f.pages = [][]app.ListRecord{{listRecord(id, nil)}}
			p := &issueCountProbe{t: t, ctx: f.ctx, rows: []app.AttachmentCount{{IssueID: id, Count: 4}}}
			switch fault {
			case "missing":
				p.rows = nil
			case "duplicate":
				p.rows = append(p.rows, p.rows[0])
			case "extra":
				p.rows = append(p.rows, app.AttachmentCount{IssueID: uuid.NewV7(), Count: 1})
			case "negative":
				p.rows[0].Count = -1
			case "failure":
				p.err = errDisk
			case "unauthorized":
				f.pages[0][0].Issue.ProjectID = uuid.NewV7()
			}
			got, err := listingWithCounts(f, p).Read(f.ctx, in)
			if err == nil || !reflect.DeepEqual(got, app.IssueListPage{}) {
				t.Fatalf("bad count returned successful page: %+v %v", got, err)
			}
			if fault == "failure" && !errors.Is(err, errDisk) {
				t.Fatal("swallowed source failure", err)
			}
			if fault == "unauthorized" && len(p.calls) != 0 {
				t.Fatal("counted unauthorized row")
			}
		})
	}
}
