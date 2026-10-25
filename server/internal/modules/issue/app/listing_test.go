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

// The orchestration boundary records every ctx and input, while the existing
// IssueReader fake interface panics if Listing tries to recount SQL children.
type listingPorts struct {
	app.IssueReader
	t         *testing.T
	ctx       context.Context
	calls     []string
	fail      string
	states    []app.StateRef
	labels    []app.LabelRef
	assignees []uuid.UUID
	parents   []domain.Issue
	pages     [][]app.ListRecord
	reads     []app.ListRead
	pids      []uuid.UUID
	parentIDs []uuid.UUID
}

func (f *listingPorts) called(ctx context.Context, name string) error {
	f.t.Helper()
	if ctx != f.ctx {
		f.t.Fatalf("%s escaped caller ctx", name)
	}
	f.calls = append(f.calls, name)
	if name == f.fail {
		return errDisk
	}
	return nil
}
func (f *listingPorts) ListStates(ctx context.Context, ids []uuid.UUID) ([]app.StateRef, error) {
	if !slices.Equal(ids, f.pids) {
		f.t.Fatalf("state scope %v; want %v", ids, f.pids)
	}
	return f.states, f.called(ctx, "states")
}
func (f *listingPorts) ListLabels(ctx context.Context, ids []uuid.UUID) ([]app.LabelRef, error) {
	if !slices.Equal(ids, f.pids) {
		f.t.Fatalf("label scope %v; want %v", ids, f.pids)
	}
	return f.labels, f.called(ctx, "labels")
}
func (f *listingPorts) AssigneeIDs(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	if !slices.Equal(ids, f.pids) {
		f.t.Fatalf("assignee scope %v; want %v", ids, f.pids)
	}
	return f.assignees, f.called(ctx, "assignees")
}
func (f *listingPorts) DisplayNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if !slices.Equal(ids, f.assignees) {
		f.t.Fatalf("profile IDs %v; want %v", ids, f.assignees)
	}
	return map[uuid.UUID]string{bob: "Zed", carol: "ÄB"}, f.called(ctx, "profiles")
}
func (f *listingPorts) Flat(ctx context.Context, in app.ListRead) ([]app.ListRecord, error) {
	f.reads = append(f.reads, in)
	name := "flat"
	if len(f.reads) > 1 {
		name = "total"
	}
	if err := f.called(ctx, name); err != nil {
		return nil, err
	}
	return f.pages[len(f.reads)-1], nil
}
func (f *listingPorts) GetIssues(ctx context.Context, ids []uuid.UUID) ([]domain.Issue, error) {
	f.parentIDs = slices.Clone(ids)
	return f.parents, f.called(ctx, "parents")
}
func listingFixture(t *testing.T) (*app.Listing, *listingPorts, app.ListingInput) {
	t.Helper()
	f := &listingPorts{t: t, ctx: context.WithValue(context.Background(), inTxKey{}, true), pids: []uuid.UUID{web},
		states: []app.StateRef{{ID: backlog.ID, Name: "BACKlog", Group: "backlog"}, {ID: done.ID, Name: "Unused", Group: "completed"}},
		labels: []app.LabelRef{{ID: uuid.NewV7(), Name: "ÄZ"}}, assignees: []uuid.UUID{carol, bob}, pages: [][]app.ListRecord{nil}}
	l := app.NewListing(app.ListingDeps{Rows: f, States: f, Labels: f, Assignees: f, Profiles: f, Issues: f})
	return l, f, app.ListingInput{ActorID: alice, Projects: []app.VisibleProject{{ID: web, OwnOnly: true}}, Query: domain.ListQuery{Order: "sort_order", Limit: 1, Filters: domain.ListFilters{SubIssues: true}}}
}
func listRecord(id uuid.UUID, parent *uuid.UUID) app.ListRecord {
	return app.ListRecord{Issue: domain.Issue{ID: id, ProjectID: web, ParentID: parent, SubIssuesCount: 7, AssigneeIDs: []uuid.UUID{bob}, LabelIDs: []uuid.UUID{}}, Total: 3, Position: domain.ListCursor{Order: "sort_order", Key: []byte("1"), ID: id}}
}

func TestListingDirectoriesAndPage(t *testing.T) {
	l, f, in := listingFixture(t)
	first, lookahead, parent, unseen := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	f.pages = [][]app.ListRecord{{listRecord(first, &parent), listRecord(lookahead, &unseen)}}
	page, err := l.Read(f.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || len(page.Data) != 1 || page.Data[0].ID != first || page.Data[0].ParentID != nil || page.Data[0].SubIssuesCount != 7 {
		t.Fatalf("page %+v", page)
	}
	position, err := domain.DecodeListCursor(page.NextCursor, in.Query.Order)
	if err != nil || position.ID != first {
		t.Fatalf("cursor %+v, %v", position, err)
	}
	want := app.ListDirectories{Projects: in.Projects, States: []app.ListState{{ID: backlog.ID, Group: "backlog", Key: "backlog"}, {ID: done.ID, Group: "completed", Key: "unused"}}, Labels: []app.ListName{{ID: f.labels[0].ID, Key: "Äz"}}, Assignees: []app.ListName{{ID: carol, Key: "Äb"}, {ID: bob, Key: "zed"}}}
	if !reflect.DeepEqual(f.reads[0].Directories, want) || !reflect.DeepEqual(f.reads[0].Query, in.Query) {
		t.Fatalf("read %+v; want dirs %+v", f.reads[0], want)
	}
	if !slices.Equal(f.calls, []string{"states", "labels", "assignees", "profiles", "flat", "parents"}) {
		t.Fatalf("calls %v", f.calls)
	}
	if !slices.Equal(f.parentIDs, []uuid.UUID{parent}) {
		t.Fatalf("read lookahead parent: %v", f.parentIDs)
	}
	// Parent pruning and response array edits must not mutate repository records.
	page.Data[0].AssigneeIDs[0] = carol
	if f.pages[0][0].Issue.ParentID == nil || f.pages[0][0].Issue.AssigneeIDs[0] != bob {
		t.Fatal("mutated Rows storage")
	}
}

func TestListingParents(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		own                    bool
		creator                *uuid.UUID
		found, archived, shown bool
	}{
		{"hidden", true, &bob, true, false, false}, {"unknown creator", true, nil, true, false, false},
		{"missing", true, &alice, false, false, false}, {"own archived", true, &alice, true, true, true},
		{"all visible", false, &bob, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l, f, in := listingFixture(t)
			parent := uuid.NewV7()
			in.Projects[0].OwnOnly = tc.own
			f.pages = [][]app.ListRecord{{listRecord(uuid.NewV7(), &parent)}}
			if tc.found {
				p := domain.Issue{ID: parent, CreatedByID: tc.creator}
				if tc.archived {
					p.ArchivedAt = &earlier
				}
				f.parents = []domain.Issue{p}
			}
			got, err := l.Read(f.ctx, in)
			if err != nil || (got.Data[0].ParentID != nil) != tc.shown || got.NextCursor != "" || got.Data[0].SubIssuesCount != 7 {
				t.Fatalf("page %+v, %v", got, err)
			}
			if tc.own && !slices.Equal(f.parentIDs, []uuid.UUID{parent}) {
				t.Fatalf("parents %v", f.parentIDs)
			}
			if !tc.own && slices.Contains(f.calls, "parents") {
				t.Fatal("unnecessary parent read")
			}
		})
	}
}

func TestListingEmptyCursorTotal(t *testing.T) {
	for _, hasCursor := range []bool{false, true} {
		t.Run(map[bool]string{false: "no cursor", true: "past end"}[hasCursor], func(t *testing.T) {
			l, f, in := listingFixture(t)
			if hasCursor {
				c := listRecord(uuid.NewV7(), nil).Position
				in.Query.Cursor = &c
			}
			key := "none"
			// Only fake Rows accepts these fields today. This proves the orchestration
			// copy, not real group/cell SQL (which is introduced in later tasks).
			in.Query.GroupBy = "label_id"
			in.Query.SubGroupBy = "assignee_id"
			in.Query.GroupKey = &key
			in.Query.SubGroupKey = &key
			in.Query.Filters.Labels.IDs = []uuid.UUID{f.labels[0].ID}
			in.Query.Filters.Priorities = []domain.Priority{"high"}
			f.pages = [][]app.ListRecord{nil, {listRecord(uuid.NewV7(), nil)}}
			before := in.Query
			got, err := l.Read(f.ctx, in)
			if err != nil || got.Data == nil || len(got.Data) != 0 || got.NextCursor != "" {
				t.Fatalf("empty page %+v, %v", got, err)
			}
			if !reflect.DeepEqual(in.Query, before) {
				t.Fatal("mutated original query")
			}
			if !hasCursor {
				if len(f.reads) != 1 || got.Total != 0 {
					t.Fatalf("true empty: %+v, %d reads", got, len(f.reads))
				}
				return
			}
			want := f.reads[0]
			want.Query.Cursor = nil
			want.Query.Limit = 1
			if len(f.reads) != 2 || !reflect.DeepEqual(f.reads[1], want) || got.Total != 3 {
				t.Fatalf("total page %+v; reads %+v", got, f.reads)
			}
			if !slices.Equal(f.calls, []string{"states", "labels", "assignees", "profiles", "flat", "total"}) {
				t.Fatalf("calls %v", f.calls)
			}
		})
	}
}

func TestListingFailuresAndScope(t *testing.T) {
	for _, failure := range []string{"states", "labels", "assignees", "profiles", "flat", "parents", "total"} {
		t.Run(failure, func(t *testing.T) {
			l, f, in := listingFixture(t)
			f.fail = failure
			parent := uuid.NewV7()
			f.pages = [][]app.ListRecord{{listRecord(uuid.NewV7(), &parent)}}
			if failure == "total" {
				c := listRecord(uuid.NewV7(), nil).Position
				in.Query.Cursor = &c
				f.pages = [][]app.ListRecord{nil, nil}
			}
			got, err := l.Read(f.ctx, in)
			if err != errDisk || !reflect.DeepEqual(got, app.IssueListPage{}) {
				t.Fatalf("failure: %+v, %v", got, err)
			}
			if f.calls[len(f.calls)-1] != failure {
				t.Fatalf("continued after error: %v", f.calls)
			}
		})
	}
	t.Run("empty scope", func(t *testing.T) {
		l, f, in := listingFixture(t)
		in.Projects = nil
		got, err := l.Read(f.ctx, in)
		if err != nil || got.Data == nil || len(got.Data) != 0 || got.Total != 0 || len(f.calls) != 0 {
			t.Fatalf("empty scope %+v, %v, calls %v", got, err, f.calls)
		}
	})
	t.Run("missing project grant", func(t *testing.T) {
		l, f, in := listingFixture(t)
		r := listRecord(uuid.NewV7(), nil)
		r.Issue.ProjectID = uuid.NewV7()
		f.pages = [][]app.ListRecord{{r}}
		if _, err := l.Read(f.ctx, in); err == nil {
			t.Fatal("unscoped issue accepted")
		}
	})
	for _, relation := range []string{"", "assigned", "created", "subscribed", "other"} {
		for _, target := range []*uuid.UUID{nil, &bob} {
			t.Run(relation+"/"+map[bool]string{true: "target", false: "project"}[target != nil], func(t *testing.T) {
				l, f, in := listingFixture(t)
				in.TargetMemberID = target
				in.Query.Relation = relation
				_, err := l.Read(f.ctx, in)
				valid := target == nil && relation == "" || target != nil && (relation == "assigned" || relation == "created" || relation == "subscribed")
				if !valid {
					if err == nil || len(f.calls) != 0 {
						t.Fatalf("bad pair read: %v, %v", err, f.calls)
					}
					return
				}
				if err != nil || f.reads[0].ActorID != alice || !reflect.DeepEqual(f.reads[0].TargetMemberID, target) {
					t.Fatalf("actor/target: %+v, %v", f.reads, err)
				}
			})
		}
	}
}

func TestListingPageBoundaries(t *testing.T) {
	t.Run("last emitted position", func(t *testing.T) {
		l, f, in := listingFixture(t)
		in.Query.Limit = 2
		first, second, third := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
		f.pages = [][]app.ListRecord{{listRecord(first, nil), listRecord(second, nil), listRecord(third, nil)}}
		got, err := l.Read(f.ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		cursor, err := domain.DecodeListCursor(got.NextCursor, in.Query.Order)
		if err != nil || len(got.Data) != 2 || got.Data[1].ID != second || cursor.ID != second {
			t.Fatalf("page %+v cursor %+v, %v", got, cursor, err)
		}
	})
	t.Run("invalid emitted position", func(t *testing.T) {
		l, f, in := listingFixture(t)
		record := listRecord(uuid.NewV7(), nil)
		record.Position.Key = []byte("null")
		f.pages = [][]app.ListRecord{{record, listRecord(uuid.NewV7(), nil)}}
		if got, err := l.Read(f.ctx, in); err == nil || !reflect.DeepEqual(got, app.IssueListPage{}) {
			t.Fatalf("bad emitted cursor %+v, %v", got, err)
		}
	})
	t.Run("empty cursor and empty catalogs", func(t *testing.T) {
		l, f, in := listingFixture(t)
		f.states = nil
		f.labels = nil
		f.assignees = nil
		c := listRecord(uuid.NewV7(), nil).Position
		in.Query.Cursor = &c
		f.pages = [][]app.ListRecord{nil, nil}
		got, err := l.Read(f.ctx, in)
		if err != nil || got.Data == nil || len(got.Data) != 0 || got.Total != 0 || len(f.reads) != 2 {
			t.Fatalf("empty cursor %+v, %v", got, err)
		}
		dirs := f.reads[0].Directories
		if dirs.States == nil || dirs.Labels == nil || dirs.Assignees == nil {
			t.Fatalf("nil catalogs %+v", dirs)
		}
	})
	for _, limit := range []int{0, 101} {
		t.Run("bad limit", func(t *testing.T) {
			l, f, in := listingFixture(t)
			in.Query.Limit = limit
			if _, err := l.Read(f.ctx, in); err == nil || len(f.calls) != 0 {
				t.Fatalf("invalid limit %d: %v, %v", limit, err, f.calls)
			}
		})
	}
}
