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

func (f *listingPorts) Groups(ctx context.Context, in app.ListRead) ([]app.ListRecord, error) {
	f.reads = append(f.reads, in)
	name := "groups"
	if len(f.reads) > 1 {
		name = "group total"
	}
	if err := f.called(ctx, name); err != nil {
		return nil, err
	}
	return f.pages[len(f.reads)-1], nil
}
func groupRecord(id uuid.UUID, key string, total int64, parent *uuid.UUID) app.ListRecord {
	r := listRecord(id, parent)
	r.Total = 5
	r.GroupKey = key
	r.GroupTotal = total
	return r
}

func TestIssueGroupListing(t *testing.T) {
	l, f, in := listingFixture(t)
	in.Query.GroupBy = "label_id"
	first, second, third, parent := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	f.pages = [][]app.ListRecord{{groupRecord(first, "a", 3, &parent), groupRecord(second, "a", 3, nil), groupRecord(third, "none", 2, &parent)}}
	page, err := l.Read(f.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 5 || page.Data != nil || len(page.Groups) != 2 || page.NextCursor != "" {
		t.Fatalf("group envelope %+v", page)
	}
	if page.Groups[0].Key != "a" || page.Groups[0].Total != 3 || len(page.Groups[0].Data) != 1 || page.Groups[0].Data[0].ID != first || page.Groups[0].Data[0].ParentID != nil || page.Groups[0].Data[0].SubIssuesCount != 7 {
		t.Fatalf("first group %+v", page.Groups[0])
	}
	c, err := domain.DecodeListCursor(page.Groups[0].NextCursor, in.Query.Order)
	if err != nil || c.ID != first {
		t.Fatalf("cursor %+v %v", c, err)
	}
	if page.Groups[1].Key != "none" || page.Groups[1].Total != 2 || page.Groups[1].NextCursor != "" || page.Groups[1].Data[0].ID != third || page.Groups[1].Data[0].ParentID != nil {
		t.Fatalf("last group %+v", page.Groups[1])
	}
	if !slices.Equal(f.calls, []string{"states", "labels", "assignees", "profiles", "groups", "parents"}) {
		t.Fatalf("calls %v", f.calls)
	}
	page.Groups[0].Data[0].AssigneeIDs[0] = carol
	if f.pages[0][0].Issue.AssigneeIDs[0] != bob || f.pages[0][0].Issue.ParentID == nil {
		t.Fatal("mutated repository row")
	}
}
func TestIssueGroupContinuation(t *testing.T) {
	for _, empty := range []bool{false, true} {
		for _, cursor := range []bool{false, true} {
			t.Run(map[bool]string{true: "empty", false: "rows"}[empty]+map[bool]string{true: " cursor", false: " first"}[cursor], func(t *testing.T) {
				l, f, in := listingFixture(t)
				key := "none"
				in.Query.GroupBy = "target_date"
				in.Query.GroupKey = &key
				in.Query.Filters.Labels.IDs = []uuid.UUID{uuid.NewV7()}
				in.Query.Filters.Priorities = []domain.Priority{"high"}
				r := groupRecord(uuid.NewV7(), key, 3, nil)
				r.Total = 8
				f.pages = [][]app.ListRecord{{r}}
				if empty {
					f.pages = [][]app.ListRecord{nil, {r}}
				}
				if cursor {
					c := r.Position
					in.Query.Cursor = &c
				}
				before := in.Query
				page, err := l.Read(f.ctx, in)
				if err != nil {
					t.Fatal(err)
				}
				want := int64(3)
				if empty && !cursor {
					want = 0
				}
				if page.Total != want || page.Data == nil || page.Groups != nil {
					t.Fatalf("selected envelope %+v", page)
				}
				if !reflect.DeepEqual(in.Query, before) {
					t.Fatal("query mutated")
				}
				if empty && cursor {
					wantRead := f.reads[0]
					wantRead.Query.Cursor = nil
					wantRead.Query.Limit = 1
					if len(f.reads) != 2 || !reflect.DeepEqual(f.reads[1], wantRead) {
						t.Fatalf("total read %+v", f.reads)
					}
				} else if len(f.reads) != 1 {
					t.Fatalf("extra reads %v", f.calls)
				}
			})
		}
	}
}
func TestIssueGroupFailuresAndEmpty(t *testing.T) {
	for _, failure := range []string{"groups", "group total", "parents"} {
		t.Run(failure, func(t *testing.T) {
			l, f, in := listingFixture(t)
			in.Query.GroupBy = "label_id"
			f.fail = failure
			parent := uuid.NewV7()
			r := groupRecord(uuid.NewV7(), "none", 1, &parent)
			f.pages = [][]app.ListRecord{{r}}
			if failure == "group total" {
				key := "none"
				in.Query.GroupKey = &key
				in.Query.Cursor = &r.Position
				f.pages = [][]app.ListRecord{nil, {r}}
			}
			page, err := l.Read(f.ctx, in)
			if err != errDisk || !reflect.DeepEqual(page, app.IssueListPage{}) {
				t.Fatalf("failure %+v %v", page, err)
			}
		})
	}
	for _, scope := range []bool{false, true} {
		l, f, in := listingFixture(t)
		in.Query.GroupBy = "label_id"
		if scope {
			in.Projects = nil
		}
		page, err := l.Read(f.ctx, in)
		if err != nil || page.Data != nil || page.Groups == nil || len(page.Groups) != 0 || page.Total != 0 {
			t.Fatalf("empty group %+v %v", page, err)
		}
	}
}
