package httpadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// ListModules serves GET /api/v0/projects/{project_id}/modules: archived
// false when left out; data an array, never null.
func (h handler) ListModules(ctx context.Context, req gen.ListModulesRequestObject) (gen.ListModulesResponseObject, error) {
	archived := req.Params.Archived != nil && *req.Params.Archived
	list, err := h.uc.ListModules.Execute(ctx, req.ProjectID, archived)
	if err != nil {
		return nil, err
	}
	out := gen.ModuleList{Data: make([]gen.ModuleListItem, len(list))}
	for i, m := range list {
		out.Data[i] = moduleListItem(m)
	}
	return gen.ListModules200JSONResponse(out), nil
}

// GetModule serves GET /api/v0/modules/{module_id}.
func (h handler) GetModule(ctx context.Context, req gen.GetModuleRequestObject) (gen.GetModuleResponseObject, error) {
	m, err := h.uc.GetModule.Execute(ctx, req.ModuleID)
	if err != nil {
		return nil, err
	}
	return gen.GetModule200JSONResponse(module(m)), nil
}

// UpdateModule serves PATCH /api/v0/modules/{module_id}: the fields given go
// to the use case as given, a status not of the six too (its 422); a date or
// the lead given as null to clear it; member_ids, given, the whole set of
// members, empty for none.
func (h handler) UpdateModule(ctx context.Context, req gen.UpdateModuleRequestObject) (gen.UpdateModuleResponseObject, error) {
	p := domain.ModulePatch{Name: req.Body.Name, Description: req.Body.Description, MemberIDs: req.Body.MemberIds, SortOrder: req.Body.SortOrder}
	if req.Body.Status != nil {
		status := domain.ModuleStatus(*req.Body.Status)
		p.Status = &status
	}
	p.SetStartDate, p.StartDate = given(req.Body.StartDate)
	p.SetTargetDate, p.TargetDate = given(req.Body.TargetDate)
	p.SetLeadID, p.LeadID = given(req.Body.LeadID)
	m, err := h.uc.UpdateModule.Execute(ctx, req.ModuleID, p)
	if err != nil {
		return nil, err
	}
	return gen.UpdateModule200JSONResponse(module(m)), nil
}

// ArchiveModule serves POST /api/v0/modules/{module_id}/archive.
func (h handler) ArchiveModule(ctx context.Context, req gen.ArchiveModuleRequestObject) (gen.ArchiveModuleResponseObject, error) {
	m, err := h.uc.ArchiveModule.Execute(ctx, req.ModuleID)
	if err != nil {
		return nil, err
	}
	return gen.ArchiveModule200JSONResponse(module(m)), nil
}

// UnarchiveModule serves POST /api/v0/modules/{module_id}/unarchive.
func (h handler) UnarchiveModule(ctx context.Context, req gen.UnarchiveModuleRequestObject) (gen.UnarchiveModuleResponseObject, error) {
	m, err := h.uc.UnarchiveModule.Execute(ctx, req.ModuleID)
	if err != nil {
		return nil, err
	}
	return gen.UnarchiveModule200JSONResponse(module(m)), nil
}

// DeleteModule serves DELETE /api/v0/modules/{module_id}.
func (h handler) DeleteModule(ctx context.Context, req gen.DeleteModuleRequestObject) (gen.DeleteModuleResponseObject, error) {
	if err := h.uc.DeleteModule.Execute(ctx, req.ModuleID); err != nil {
		return nil, err
	}
	return gen.DeleteModule204Response{}, nil
}

// CreateModule serves POST /api/v0/projects/{project_id}/modules: the values
// go to the use case as given, a status not of the six too (its 422); the
// description empty, the dates, the status and the lead none, and no member,
// when left out; a lead given as null as none.
func (h handler) CreateModule(ctx context.Context, req gen.CreateModuleRequestObject) (gen.CreateModuleResponseObject, error) {
	in := domain.NewModule{Name: req.Body.Name, StartDate: req.Body.StartDate, TargetDate: req.Body.TargetDate}
	if req.Body.Description != nil {
		in.Description = *req.Body.Description
	}
	if req.Body.Status != nil {
		status := domain.ModuleStatus(*req.Body.Status)
		in.Status = &status
	}
	_, in.LeadID = given(req.Body.LeadID)
	if req.Body.MemberIds != nil {
		in.MemberIDs = *req.Body.MemberIds
	}
	m, err := h.uc.CreateModule.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.CreateModule201JSONResponse(module(m)), nil
}

// module is m as the API shows it: null for each value it has not, its
// members an array, never null.
func module(m domain.Module) gen.Module {
	members := m.MemberIDs
	if members == nil {
		members = []uuid.UUID{}
	}
	return gen.Module{ID: m.ID, WorkspaceID: m.WorkspaceID, ProjectID: m.ProjectID, Name: m.Name, Description: m.Description,
		StartDate: orNull(m.StartDate), TargetDate: orNull(m.TargetDate), Status: gen.ModuleStatus(m.Status), LeadID: orNull(m.LeadID),
		MemberIds: members, SortOrder: m.SortOrder, ArchivedAt: orNull(m.ArchivedAt), CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
		CreatedByID: orNull(m.CreatedByID), UpdatedByID: orNull(m.UpdatedByID)}
}

// moduleListItem is m as the list shows it: the module as module shows it,
// and the counts of its work items.
func moduleListItem(m domain.ModuleListItem) gen.ModuleListItem {
	b := module(m.Module)
	return gen.ModuleListItem{ID: b.ID, WorkspaceID: b.WorkspaceID, ProjectID: b.ProjectID, Name: b.Name, Description: b.Description,
		StartDate: b.StartDate, TargetDate: b.TargetDate, Status: b.Status, LeadID: b.LeadID, MemberIds: b.MemberIds, SortOrder: b.SortOrder,
		ArchivedAt: b.ArchivedAt, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt, CreatedByID: b.CreatedByID, UpdatedByID: b.UpdatedByID,
		TotalIssues: m.Live.Total, BacklogIssues: m.Live.Backlog, UnstartedIssues: m.Live.Unstarted, StartedIssues: m.Live.Started,
		CompletedIssues: m.Live.Completed, CancelledIssues: m.Live.Cancelled}
}
