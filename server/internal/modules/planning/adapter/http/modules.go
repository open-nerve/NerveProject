package httpadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

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
