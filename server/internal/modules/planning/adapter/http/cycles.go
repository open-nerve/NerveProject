package httpadapter

import (
	"context"
	"uuid"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// ListCycles serves GET /api/v0/projects/{project_id}/cycles: archived
// false when left out; data an array, never null.
func (h handler) ListCycles(ctx context.Context, req gen.ListCyclesRequestObject) (gen.ListCyclesResponseObject, error) {
	archived := req.Params.Archived != nil && *req.Params.Archived
	list, err := h.uc.ListCycles.Execute(ctx, req.ProjectID, archived)
	if err != nil {
		return nil, err
	}
	out := gen.CycleList{Data: make([]gen.CycleListItem, len(list))}
	for i, c := range list {
		out.Data[i] = cycleListItem(c)
	}
	return gen.ListCycles200JSONResponse(out), nil
}

// GetCycle serves GET /api/v0/cycles/{cycle_id}.
func (h handler) GetCycle(ctx context.Context, req gen.GetCycleRequestObject) (gen.GetCycleResponseObject, error) {
	c, err := h.uc.GetCycle.Execute(ctx, req.CycleID)
	if err != nil {
		return nil, err
	}
	return gen.GetCycle200JSONResponse(cycle(c)), nil
}

// UpdateCycle serves PATCH /api/v0/cycles/{cycle_id}: the fields given go
// to the use case, a date given as null as one given as none.
func (h handler) UpdateCycle(ctx context.Context, req gen.UpdateCycleRequestObject) (gen.UpdateCycleResponseObject, error) {
	p := domain.CyclePatch{Name: req.Body.Name, Description: req.Body.Description, SortOrder: req.Body.SortOrder}
	p.SetStart, p.Start = given(req.Body.StartDate)
	p.SetEnd, p.End = given(req.Body.EndDate)
	c, err := h.uc.UpdateCycle.Execute(ctx, req.CycleID, p)
	if err != nil {
		return nil, err
	}
	return gen.UpdateCycle200JSONResponse(cycle(c)), nil
}

// DeleteCycle serves DELETE /api/v0/cycles/{cycle_id}.
func (h handler) DeleteCycle(ctx context.Context, req gen.DeleteCycleRequestObject) (gen.DeleteCycleResponseObject, error) {
	if err := h.uc.DeleteCycle.Execute(ctx, req.CycleID); err != nil {
		return nil, err
	}
	return gen.DeleteCycle204Response{}, nil
}

// ArchiveCycle serves POST /api/v0/cycles/{cycle_id}/archive.
func (h handler) ArchiveCycle(ctx context.Context, req gen.ArchiveCycleRequestObject) (gen.ArchiveCycleResponseObject, error) {
	c, err := h.uc.ArchiveCycle.Execute(ctx, req.CycleID)
	if err != nil {
		return nil, err
	}
	return gen.ArchiveCycle200JSONResponse(cycle(c)), nil
}

// UnarchiveCycle serves POST /api/v0/cycles/{cycle_id}/unarchive.
func (h handler) UnarchiveCycle(ctx context.Context, req gen.UnarchiveCycleRequestObject) (gen.UnarchiveCycleResponseObject, error) {
	c, err := h.uc.UnarchiveCycle.Execute(ctx, req.CycleID)
	if err != nil {
		return nil, err
	}
	return gen.UnarchiveCycle200JSONResponse(cycle(c)), nil
}

// TransferCycleIssues serves POST /api/v0/cycles/{cycle_id}/transfer-issues:
// the answer is how many work items moved, none too.
func (h handler) TransferCycleIssues(ctx context.Context, req gen.TransferCycleIssuesRequestObject) (gen.TransferCycleIssuesResponseObject, error) {
	n, err := h.uc.TransferCycleIssues.Execute(ctx, req.CycleID, req.Body.TargetCycleID)
	if err != nil {
		return nil, err
	}
	return gen.TransferCycleIssues200JSONResponse(gen.CycleTransferResult{TransferredIssues: int64(n)}), nil
}

// given is whether v is given, and its value, nil for null.
func given[T any](v nullable.Nullable[T]) (bool, *T) {
	switch {
	case !v.IsSpecified():
		return false, nil
	case v.IsNull():
		return true, nil
	}
	value := v.MustGet()
	return true, &value
}

// CreateCycle serves POST /api/v0/projects/{project_id}/cycles: the values
// go to the use case as given, the description empty and the dates none
// when left out.
func (h handler) CreateCycle(ctx context.Context, req gen.CreateCycleRequestObject) (gen.CreateCycleResponseObject, error) {
	in := domain.NewCycle{Name: req.Body.Name, Dates: domain.Dates{Start: req.Body.StartDate, End: req.Body.EndDate}}
	if req.Body.Description != nil {
		in.Description = *req.Body.Description
	}
	c, err := h.uc.CreateCycle.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.CreateCycle201JSONResponse(cycle(c)), nil
}

// cycle is c as the API shows it: null for each value it has not.
func cycle(c domain.ReadCycle) gen.Cycle {
	return gen.Cycle{ID: c.ID, WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID, Name: c.Name, Description: c.Description,
		StartDate: orNull(c.Dates.Start), EndDate: orNull(c.Dates.End), Status: gen.CycleStatus(c.Status), OwnedByID: orNull(c.OwnedByID),
		SortOrder: c.SortOrder, ArchivedAt: orNull(c.ArchivedAt), CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
		CreatedByID: orNull(c.CreatedByID), UpdatedByID: orNull(c.UpdatedByID)}
}

// cycleListItem is c as the list shows it: the cycle as cycle shows it,
// the counts of its work items, their assignees an array, never null, and
// its frozen counts, null for none.
func cycleListItem(c domain.CycleListItem) gen.CycleListItem {
	b := cycle(c.ReadCycle)
	assignees := c.AssigneeIDs
	if assignees == nil {
		assignees = []uuid.UUID{}
	}
	out := gen.CycleListItem{ID: b.ID, WorkspaceID: b.WorkspaceID, ProjectID: b.ProjectID, Name: b.Name, Description: b.Description,
		StartDate: b.StartDate, EndDate: b.EndDate, Status: b.Status, OwnedByID: b.OwnedByID, SortOrder: b.SortOrder, ArchivedAt: b.ArchivedAt,
		CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt, CreatedByID: b.CreatedByID, UpdatedByID: b.UpdatedByID,
		TotalIssues: c.Live.Total, BacklogIssues: c.Live.Backlog, UnstartedIssues: c.Live.Unstarted, StartedIssues: c.Live.Started,
		CompletedIssues: c.Live.Completed, CancelledIssues: c.Live.Cancelled, AssigneeIds: assignees,
		ProgressSnapshot: nullable.NewNullNullable[gen.FrozenCounts]()}
	if f := c.Snapshot; f != nil {
		out.ProgressSnapshot = nullable.NewNullableWithValue(gen.FrozenCounts{TotalIssues: f.Total, BacklogIssues: f.Backlog,
			UnstartedIssues: f.Unstarted, StartedIssues: f.Started, CompletedIssues: f.Completed, CancelledIssues: f.Cancelled,
			FrozenAt: f.FrozenAt})
	}
	return out
}

// orNull is v as a nullable field: null when v is nil.
func orNull[T any](v *T) nullable.Nullable[T] {
	if v == nil {
		return nullable.NewNullNullable[T]()
	}
	return nullable.NewNullableWithValue(*v)
}
