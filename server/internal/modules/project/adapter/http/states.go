package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// CreateState serves POST /api/v0/projects/{project_id}/states: the group
// goes to the use case as given, triage and one outside the five too, which
// the domain refuses (422).
func (h handler) CreateState(ctx context.Context, req gen.CreateStateRequestObject) (gen.CreateStateResponseObject, error) {
	in := domain.StateCreate{Name: req.Body.Name, Color: req.Body.Color, Group: domain.StateGroup(req.Body.Group)}
	if req.Body.Description != nil {
		in.Description = *req.Body.Description
	}
	s, err := h.uc.CreateState.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.CreateState201JSONResponse(state(s)), nil
}

// state is s as the API shows it.
func state(s domain.State) gen.State {
	return gen.State{ID: s.ID, WorkspaceID: s.WorkspaceID, ProjectID: s.ProjectID, Name: s.Name, Description: s.Description, Color: s.Color,
		Group: gen.StateGroup(s.Group), Default: s.Default, Sequence: s.Sequence, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt}
}
