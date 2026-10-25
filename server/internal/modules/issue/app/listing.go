package app

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListingInput contains the authorized project scope and validated query.
// Actor controls visibility; the independent target controls member relations.
type ListingInput struct {
	ActorID        uuid.UUID
	TargetMemberID *uuid.UUID
	Projects       []VisibleProject
	Query          domain.ListQuery
}

// IssueListPage is a flat page. An empty NextCursor represents the last page.
type IssueListPage struct {
	Total      int64
	Data       []domain.Issue
	NextCursor string
}

type ListingDeps struct {
	Rows      ListRows
	States    ListStateCatalog
	Labels    ListLabelCatalog
	Assignees AssigneeDirectory
	Profiles  PublicProfiles
	Issues    IssueReader
}

// Listing assembles a list inside the read snapshot provided by its caller.
// It never opens or upgrades a transaction.
type Listing struct{ deps ListingDeps }

func NewListing(d ListingDeps) *Listing { return &Listing{deps: d} }

func (l *Listing) Read(ctx context.Context, input ListingInput) (IssueListPage, error) {
	q := input.Query
	relation := q.Relation == "assigned" || q.Relation == "created" || q.Relation == "subscribed"
	if input.TargetMemberID == nil && q.Relation != "" || input.TargetMemberID != nil && !relation {
		return IssueListPage{}, fmt.Errorf("list relation and target member do not match")
	}
	if q.Limit < 1 || q.Limit > shared.MaxPageSize {
		return IssueListPage{}, fmt.Errorf("invalid list page size")
	}
	page := IssueListPage{Data: make([]domain.Issue, 0)}
	if len(input.Projects) == 0 {
		return page, nil
	}
	directories, err := l.directories(ctx, input.Projects)
	if err != nil {
		return IssueListPage{}, err
	}
	read := ListRead{ActorID: input.ActorID, TargetMemberID: input.TargetMemberID, Query: q, Directories: directories}
	rows, err := l.deps.Rows.Flat(ctx, read)
	if err != nil {
		return IssueListPage{}, err
	}
	if len(rows) == 0 {
		if q.Cursor != nil {
			read.Query.Cursor = nil
			read.Query.Limit = 1
			totals, err := l.deps.Rows.Flat(ctx, read)
			if err != nil {
				return IssueListPage{}, err
			}
			if len(totals) > 0 {
				page.Total = totals[0].Total
			}
		}
		return page, nil
	}
	page.Total = rows[0].Total
	size := min(len(rows), q.Limit)
	page.Data = make([]domain.Issue, size)
	own := make(map[uuid.UUID]bool, len(input.Projects))
	for _, p := range input.Projects {
		own[p.ID] = p.OwnOnly
	}
	var guarded []int
	for n, r := range rows[:size] {
		only, ok := own[r.Issue.ProjectID]
		if !ok {
			return IssueListPage{}, fmt.Errorf("no grant on project %s, of work item %s", r.Issue.ProjectID, r.Issue.ID)
		}
		page.Data[n] = r.Issue
		page.Data[n].AssigneeIDs = append([]uuid.UUID{}, r.Issue.AssigneeIDs...)
		page.Data[n].LabelIDs = append([]uuid.UUID{}, r.Issue.LabelIDs...)
		if only && r.Issue.ParentID != nil {
			guarded = append(guarded, n)
		}
	}
	if err := hideParents(ctx, l.deps.Issues, input.ActorID, page.Data, guarded); err != nil {
		return IssueListPage{}, err
	}
	if len(rows) > q.Limit {
		page.NextCursor, err = domain.EncodeListCursor(rows[size-1].Position)
		if err != nil {
			return IssueListPage{}, err
		}
	}
	return page, nil
}

func (l *Listing) directories(ctx context.Context, projects []VisibleProject) (ListDirectories, error) {
	ids := make([]uuid.UUID, len(projects))
	for n, p := range projects {
		ids[n] = p.ID
	}
	states, err := l.deps.States.ListStates(ctx, ids)
	if err != nil {
		return ListDirectories{}, err
	}
	labels, err := l.deps.Labels.ListLabels(ctx, ids)
	if err != nil {
		return ListDirectories{}, err
	}
	assignees, err := l.deps.Assignees.AssigneeIDs(ctx, ids)
	if err != nil {
		return ListDirectories{}, err
	}
	names, err := l.deps.Profiles.DisplayNames(ctx, assignees)
	if err != nil {
		return ListDirectories{}, err
	}
	dirs := ListDirectories{Projects: slices.Clone(projects), States: make([]ListState, 0, len(states)), Labels: make([]ListName, 0, len(labels)), Assignees: make([]ListName, 0, len(assignees))}
	for _, s := range states {
		dirs.States = append(dirs.States, ListState{ID: s.ID, Group: s.Group, Key: domain.NameOrderKey(s.Name)})
	}
	for _, v := range labels {
		dirs.Labels = append(dirs.Labels, ListName{ID: v.ID, Key: domain.NameOrderKey(v.Name)})
	}
	for _, id := range assignees {
		if name, ok := names[id]; ok {
			dirs.Assignees = append(dirs.Assignees, ListName{ID: id, Key: domain.NameOrderKey(name)})
		}
	}
	return dirs, nil
}
