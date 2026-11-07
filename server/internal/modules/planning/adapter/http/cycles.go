package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

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

// orNull is v as a nullable field: null when v is nil.
func orNull[T any](v *T) nullable.Nullable[T] {
	if v == nil {
		return nullable.NewNullNullable[T]()
	}
	return nullable.NewNullableWithValue(*v)
}
