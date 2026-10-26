package app_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func (f *listingPorts) Subgroups(ctx context.Context, in app.ListRead) ([]app.ListRecord, error) {
	f.reads = append(f.reads, in)
	name := "subgroups"
	if len(f.reads) > 1 {
		name = "cell total"
	}
	if err := f.called(ctx, name); err != nil {
		return nil, err
	}
	return f.pages[len(f.reads)-1], nil
}

func cellRecord(id uuid.UUID, group, lane string, parent *uuid.UUID) app.ListRecord {
	r := groupRecord(id, group, 3, parent)
	r.SubGroupKey, r.SubgroupTotal, r.CellTotal = lane, 4, 2
	return r
}

func TestIssueSubgroupsListing(t *testing.T) {
	l, f, in := listingFixture(t)
	in.Query.GroupBy, in.Query.SubGroupBy = "label_id", "priority"
	first, second, parent, lookParent := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	f.pages = [][]app.ListRecord{{
		cellRecord(first, "a", "low", &parent), cellRecord(second, "a", "low", &lookParent),
		cellRecord(second, "b", "urgent", &parent), cellRecord(first, "b", "urgent", nil),
		cellRecord(first, "b", "low", &parent),
	}}
	page, err := l.Read(f.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 5 || page.Data != nil || page.NextCursor != "" || len(page.Groups) != 2 || !reflect.DeepEqual(page.SubgroupTotals, []app.ListTotal{{Key: "urgent", Total: 4}, {Key: "low", Total: 4}}) {
		t.Fatalf("lane envelope %+v", page)
	}
	for n, group := range page.Groups {
		if group.Total != 3 || group.Data != nil || group.NextCursor != "" || len(group.Subgroups) != n+1 {
			t.Fatalf("group %+v", group)
		}
		for _, cell := range group.Subgroups {
			if cell.Total != 2 || len(cell.Data) != 1 || cell.Data[0].ParentID != nil || cell.Data[0].SubIssuesCount != 7 || cell.Subgroups != nil {
				t.Fatalf("cell %+v", cell)
			}
		}
	}
	for n, id := range []uuid.UUID{first, second} {
		c, err := domain.DecodeListCursor(page.Groups[n].Subgroups[0].NextCursor, in.Query.Order)
		if err != nil || c.ID != id {
			t.Fatalf("independent cursor %+v %v", c, err)
		}
	}
	if page.Groups[1].Subgroups[1].NextCursor != "" || !slices.Equal(f.parentIDs, []uuid.UUID{parent}) || !slices.Equal(f.calls, []string{"states", "labels", "assignees", "profiles", "subgroups", "parents"}) {
		t.Fatalf("output-only parent read / cursors: %+v %v", page, f.calls)
	}
	page.Groups[0].Subgroups[0].Data[0].AssigneeIDs[0] = carol
	if f.pages[0][0].Issue.ParentID == nil || f.pages[0][0].Issue.AssigneeIDs[0] != bob {
		t.Fatal("mutated repository row")
	}
}

func TestIssueCellContinuation(t *testing.T) {
	for _, mode := range []string{"first", "past end", "absent", "empty first"} {
		t.Run(mode, func(t *testing.T) {
			l, f, in := listingFixture(t)
			group, lane := "none", "high"
			in.Query.GroupBy, in.Query.SubGroupBy = "label_id", "priority"
			in.Query.GroupKey, in.Query.SubGroupKey = &group, &lane
			in.Query.Filters.Labels.IDs = []uuid.UUID{uuid.NewV7()}
			in.Query.Filters.Assignees.IDs = []uuid.UUID{bob}
			r := cellRecord(uuid.NewV7(), group, lane, nil)
			f.pages = [][]app.ListRecord{{r}}
			want := int64(2)
			if mode != "first" {
				f.pages = [][]app.ListRecord{nil, {r}}
				in.Query.Cursor = &r.Position
			}
			if mode == "absent" || mode == "empty first" {
				f.pages = [][]app.ListRecord{nil, nil}
				want = 0
			}
			if mode == "empty first" {
				in.Query.Cursor = nil
			}
			before := in.Query
			page, err := l.Read(f.ctx, in)
			if err != nil || page.Total != want || page.Data == nil || page.Groups != nil || page.SubgroupTotals != nil || page.NextCursor != "" {
				t.Fatalf("selected cell %+v %v", page, err)
			}
			if !reflect.DeepEqual(before, in.Query) {
				t.Fatal("input query mutated")
			}
			if in.Query.Cursor != nil {
				wantRead := f.reads[0]
				wantRead.Query.Cursor, wantRead.Query.Limit = nil, 1
				if len(f.reads) != 2 || !reflect.DeepEqual(f.reads[1], wantRead) {
					t.Fatalf("retry lost selection/filter/catalog %+v", f.reads)
				}
			} else if len(f.reads) != 1 {
				t.Fatalf("unnecessary query %v", f.calls)
			}
		})
	}
}

func TestIssueSubgroupsFailuresAndEmpty(t *testing.T) {
	for _, failure := range []string{"subgroups", "cell total", "parents"} {
		l, f, in := listingFixture(t)
		in.Query.GroupBy, in.Query.SubGroupBy = "label_id", "assignee_id"
		f.fail = failure
		parent, key := uuid.NewV7(), "none"
		r := cellRecord(uuid.NewV7(), key, key, &parent)
		f.pages = [][]app.ListRecord{{r}}
		if failure == "cell total" {
			in.Query.GroupKey, in.Query.SubGroupKey = &key, &key
			in.Query.Cursor = &r.Position
			f.pages = [][]app.ListRecord{nil, {r}}
		}
		page, err := l.Read(f.ctx, in)
		if err != errDisk || !reflect.DeepEqual(page, app.IssueListPage{}) {
			t.Fatalf("%s failure %+v %v", failure, page, err)
		}
	}
	for _, emptyScope := range []bool{false, true} {
		l, f, in := listingFixture(t)
		in.Query.GroupBy, in.Query.SubGroupBy = "label_id", "assignee_id"
		if emptyScope {
			in.Projects = nil
		}
		page, err := l.Read(f.ctx, in)
		if err != nil || page.Total != 0 || page.Data != nil || page.Groups == nil || len(page.Groups) != 0 || page.SubgroupTotals == nil || len(page.SubgroupTotals) != 0 {
			t.Fatalf("empty lanes %+v %v", page, err)
		}
	}
}
