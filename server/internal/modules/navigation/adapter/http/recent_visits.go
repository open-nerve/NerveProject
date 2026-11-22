package httpadapter

import (
	"context"
	"uuid"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apigen"
)

// RecordRecentVisitUseCase is app.RecordRecentVisit.
type RecordRecentVisitUseCase interface {
	Execute(ctx context.Context, kind domain.RecentVisitKind, id uuid.UUID) error
}

// ListRecentVisitsUseCase is app.ListRecentVisits: kind is nil for both
// kinds.
type ListRecentVisitsUseCase interface {
	Execute(ctx context.Context, slug string, kind *domain.RecentVisitKind) ([]app.ShownRecentVisit, error)
}

// RecordRecentVisit serves POST /api/v0/me/recent-visits: the body's kind
// goes to the use case as given, which checks it, with the target's id; a
// new visit and a revisit alike answer 204 (P3d spec 2.4, R7).
func (h handler) RecordRecentVisit(ctx context.Context, req gen.RecordRecentVisitRequestObject) (gen.RecordRecentVisitResponseObject, error) {
	if err := h.uc.RecordRecentVisit.Execute(ctx, domain.RecentVisitKind(req.Body.EntityType), req.Body.EntityID); err != nil {
		return nil, err
	}
	return gen.RecordRecentVisit204Response{}, nil
}

// ListRecentVisits serves GET /api/v0/workspaces/{slug}/recent-visits: the
// query's entity_type goes to the use case as given, which checks it, nil
// when it is left out (spec 2.4, R8).
func (h handler) ListRecentVisits(ctx context.Context, req gen.ListRecentVisitsRequestObject) (gen.ListRecentVisitsResponseObject, error) {
	var kind *domain.RecentVisitKind
	if k := req.Params.EntityType; k != nil {
		of := domain.RecentVisitKind(*k)
		kind = &of
	}
	list, err := h.uc.ListRecentVisits.Execute(ctx, req.Slug, kind)
	if err != nil {
		return nil, err
	}
	out := gen.RecentVisitList{Data: make([]gen.RecentVisit, len(list))}
	for i, v := range list {
		out.Data[i] = recentVisitOf(v)
	}
	return gen.ListRecentVisits200JSONResponse(out), nil
}

// recentVisitOf is v as the API shows it (spec 2.4): its project now, its
// icon {} for none; and its work item now, a work item's visit's, its
// assignees [] for none, null for a project's.
func recentVisitOf(v app.ShownRecentVisit) gen.RecentVisit {
	p := v.Project
	out := gen.RecentVisit{ID: v.Visit.ID, EntityType: gen.RecentVisitKind(v.Visit.Kind), EntityID: v.Visit.EntityID, VisitedAt: v.Visit.VisitedAt,
		Project: gen.RecentVisitProject{ID: v.Visit.ProjectID, Name: p.Name, Identifier: p.Identifier, LogoProps: logoOut(p.LogoProps)},
		Issue:   nullable.NewNullNullable[gen.RecentVisitIssue]()}
	if i := v.Issue; i != nil {
		assignees := i.AssigneeIDs
		if assignees == nil {
			assignees = []uuid.UUID{}
		}
		out.Issue = nullable.NewNullableWithValue(gen.RecentVisitIssue{ID: i.ID, SequenceID: i.SequenceID, Name: i.Name, StateID: i.StateID,
			Priority: apigen.IssuePriority(i.Priority), AssigneeIds: assignees})
	}
	return out
}
