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

// IssueListPage contains either Data or Groups, selected by the query.
// An empty NextCursor represents the last page.
type IssueListPage struct {
	Total          int64
	Data           []domain.Issue
	NextCursor     string
	Groups         []IssueListGroup
	SubgroupTotals []ListTotal
}

// ListTotal is the independent, deduplicated total for one lane.
type ListTotal struct {
	Key   string
	Total int64
}

// IssueListGroup contains a group or cell, with either rows or subgroups.
type IssueListGroup struct {
	Key        string
	Total      int64
	Data       []domain.Issue
	NextCursor string
	Subgroups  []IssueListGroup
}

type ListingDeps struct {
	Attachments       AttachmentCounter
	AttachmentCatalog AttachmentCountCatalog
	Rows              ListRows
	States            ListStateCatalog
	Labels            ListLabelCatalog
	Assignees         AssigneeDirectory
	Profiles          PublicProfiles
	Issues            IssueReader
	// Planning names the cycles and modules of a list ordered by them.
	Planning PlanningDirectory
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
	if q.SubGroupKey != nil && q.SubGroupBy == "" || q.SubGroupBy != "" && (q.GroupBy == "" || q.GroupBy == q.SubGroupBy || (q.GroupKey == nil) != (q.SubGroupKey == nil)) {
		return IssueListPage{}, fmt.Errorf("invalid list subgroup")
	}
	page := IssueListPage{Data: make([]domain.Issue, 0)}
	if q.GroupBy != "" && q.GroupKey == nil {
		page = IssueListPage{Groups: make([]IssueListGroup, 0)}
		if q.SubGroupBy != "" {
			page.SubgroupTotals = make([]ListTotal, 0)
		}
	}
	if len(input.Projects) == 0 {
		return page, nil
	}
	directories, err := l.directories(ctx, input.Projects)
	if err != nil {
		return IssueListPage{}, err
	}
	// The directories only an order needs are read for it alone, in the same snapshot.
	switch q.Order {
	case "attachment_count", "-attachment_count":
		directories.AttachmentCounts, err = l.attachmentCounts(ctx, input.Projects)
	case "cycle_name", "-cycle_name":
		directories.Cycles, err = planningNames(ctx, l.deps.Planning.CycleNames, input.Projects)
	case "module_name", "-module_name":
		directories.Modules, err = planningNames(ctx, l.deps.Planning.ModuleNames, input.Projects)
	}
	if err != nil {
		return IssueListPage{}, err
	}
	read := ListRead{ActorID: input.ActorID, TargetMemberID: input.TargetMemberID, Query: q, Directories: directories}
	fetch := l.deps.Rows.Flat
	if q.GroupBy != "" {
		fetch = l.deps.Rows.Groups
	}
	if q.SubGroupBy != "" {
		fetch = l.deps.Rows.Subgroups
	}
	rows, err := fetch(ctx, read)
	if err != nil {
		return IssueListPage{}, err
	}
	if len(rows) == 0 {
		if q.Cursor != nil {
			read.Query.Cursor = nil
			read.Query.Limit = 1
			totals, err := fetch(ctx, read)
			if err != nil {
				return IssueListPage{}, err
			}
			if len(totals) > 0 {
				page.Total = selectedTotal(q, totals[0])
			}
		}
		return page, nil
	}
	if q.GroupBy != "" && q.GroupKey == nil {
		if q.SubGroupBy != "" {
			return l.subgroups(ctx, input, rows)
		}
		return l.groups(ctx, input, rows)
	}
	page.Total = selectedTotal(q, rows[0])
	size := min(len(rows), q.Limit)
	page.Data, err = l.issues(ctx, input, rows[:size])
	if err != nil {
		return IssueListPage{}, err
	}
	page.NextCursor, err = pageCursor(rows, q.Limit)
	if err != nil {
		return IssueListPage{}, err
	}
	return page, nil
}

func selectedTotal(q domain.ListQuery, r ListRecord) int64 {
	if q.SubGroupKey != nil {
		return r.CellTotal
	}
	if q.GroupKey != nil {
		return r.GroupTotal
	}
	return r.Total
}
func pageCursor(rows []ListRecord, limit int) (string, error) {
	if len(rows) > limit {
		return domain.EncodeListCursor(rows[limit-1].Position)
	}
	return "", nil
}

// issues copies and prunes only emitted rows; SQL child counts are retained.
// Group pages use one batch so shared hidden parents are read only once.
func (l *Listing) issues(ctx context.Context, input ListingInput, rows []ListRecord) ([]domain.Issue, error) {
	items := make([]domain.Issue, len(rows))
	own := make(map[uuid.UUID]bool, len(input.Projects))
	for _, p := range input.Projects {
		own[p.ID] = p.OwnOnly
	}
	var guarded []int
	for n, r := range rows {
		only, ok := own[r.Issue.ProjectID]
		if !ok {
			return nil, fmt.Errorf("no grant on project %s, of work item %s", r.Issue.ProjectID, r.Issue.ID)
		}
		items[n] = r.Issue
		items[n].AssigneeIDs = append([]uuid.UUID{}, r.Issue.AssigneeIDs...)
		items[n].LabelIDs = append([]uuid.UUID{}, r.Issue.LabelIDs...)
		items[n].ModuleIDs = append([]uuid.UUID{}, r.Issue.ModuleIDs...)
		if only && r.Issue.ParentID != nil {
			guarded = append(guarded, n)
		}
	}
	if err := hideParents(ctx, l.deps.Issues, input.ActorID, items, guarded); err != nil {
		return nil, err
	}
	if err := projectAttachments(ctx, l.deps.Attachments, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (l *Listing) groups(ctx context.Context, input ListingInput, rows []ListRecord) (IssueListPage, error) {
	page := IssueListPage{Total: rows[0].Total, Groups: make([]IssueListGroup, 0)}
	var emitted []ListRecord
	var sizes []int
	for start := 0; start < len(rows); {
		end := start + 1
		for end < len(rows) && rows[end].GroupKey == rows[start].GroupKey {
			end++
		}
		part := rows[start:end]
		size := min(len(part), input.Query.Limit)
		cursor, err := pageCursor(part, input.Query.Limit)
		if err != nil {
			return IssueListPage{}, err
		}
		page.Groups = append(page.Groups, IssueListGroup{Key: part[0].GroupKey, Total: part[0].GroupTotal, NextCursor: cursor})
		emitted = append(emitted, part[:size]...)
		sizes = append(sizes, size)
		start = end
	}
	items, err := l.issues(ctx, input, emitted)
	if err != nil {
		return IssueListPage{}, err
	}
	for n, size := range sizes {
		page.Groups[n].Data = items[:size:size]
		items = items[size:]
	}
	return page, nil
}

func (l *Listing) directories(ctx context.Context, projects []VisibleProject) (ListDirectories, error) {
	ids := projectIDs(projects)
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

// planningNames reads with read the cycles' or the modules' names of the
// listed projects, those the other directories are of, and keys them as
// those are, in Go (NameOrderKey, M6 P3b spec R2).
func planningNames(ctx context.Context, read func(context.Context, []uuid.UUID) ([]PlanningName, error), projects []VisibleProject) ([]ListName, error) {
	names, err := read(ctx, projectIDs(projects))
	if err != nil {
		return nil, err
	}
	keyed := make([]ListName, len(names))
	for n, v := range names {
		keyed[n] = ListName{ID: v.ID, Key: domain.NameOrderKey(v.Name)}
	}
	return keyed, nil
}

// projectIDs are the ids of the listed projects, in their order.
func projectIDs(projects []VisibleProject) []uuid.UUID {
	ids := make([]uuid.UUID, len(projects))
	for n, p := range projects {
		ids[n] = p.ID
	}
	return ids
}
