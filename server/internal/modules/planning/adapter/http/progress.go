package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// GetCycleProgress serves GET /api/v0/cycles/{cycle_id}/progress: the
// frozen progress null when the use case answers none.
func (h handler) GetCycleProgress(ctx context.Context, req gen.GetCycleProgressRequestObject) (gen.GetCycleProgressResponseObject, error) {
	p, err := h.uc.GetCycleProgress.Execute(ctx, req.CycleID)
	if err != nil {
		return nil, err
	}
	out := gen.CycleProgress{Status: gen.CycleStatus(p.Status), Live: progress(p.Live), ProgressSnapshot: nullable.NewNullNullable[gen.FrozenProgress]()}
	if p.Snapshot != nil {
		out.ProgressSnapshot = nullable.NewNullableWithValue(frozen(*p.Snapshot))
	}
	return gen.GetCycleProgress200JSONResponse(out), nil
}

// GetModuleProgress serves GET /api/v0/modules/{module_id}/progress.
func (h handler) GetModuleProgress(ctx context.Context, req gen.GetModuleProgressRequestObject) (gen.GetModuleProgressResponseObject, error) {
	p, err := h.uc.GetModuleProgress.Execute(ctx, req.ModuleID)
	if err != nil {
		return nil, err
	}
	return gen.GetModuleProgress200JSONResponse(progress(p)), nil
}

// progress is p as the API shows it: its lists arrays, never null; null for
// a burndown it has not, and for an id or a day's number it has not.
func progress(p domain.Progress) gen.Progress {
	out := gen.Progress{TotalIssues: p.Total, BacklogIssues: p.Backlog, UnstartedIssues: p.Unstarted, StartedIssues: p.Started,
		CompletedIssues: p.Completed, CancelledIssues: p.Cancelled, Assignees: make([]gen.AssigneeDistribution, len(p.Assignees)),
		Labels: make([]gen.LabelDistribution, len(p.Labels)), Burndown: nullable.NewNullNullable[gen.Burndown]()}
	for i, a := range p.Assignees {
		out.Assignees[i] = gen.AssigneeDistribution{AssigneeID: orNull(a.AssigneeID), TotalIssues: a.Total, CompletedIssues: a.Completed,
			PendingIssues: a.Pending}
	}
	for i, l := range p.Labels {
		out.Labels[i] = gen.LabelDistribution{LabelID: orNull(l.LabelID), TotalIssues: l.Total, CompletedIssues: l.Completed, PendingIssues: l.Pending}
	}
	if b := p.Burndown; b != nil {
		points := make([]gen.BurndownPoint, len(b.Points))
		for i, point := range b.Points {
			points[i] = gen.BurndownPoint{Date: point.Date, Remaining: orNull(point.Remaining)}
		}
		out.Burndown = nullable.NewNullableWithValue(gen.Burndown{StartDate: b.StartDate, EndDate: b.EndDate, Scope: b.Scope, Points: points})
	}
	return out
}

// frozen is f as the API shows it: its progress as progress shows a live
// one, and when it was frozen.
func frozen(f domain.Frozen) gen.FrozenProgress {
	p := progress(f.Progress)
	return gen.FrozenProgress{TotalIssues: p.TotalIssues, BacklogIssues: p.BacklogIssues, UnstartedIssues: p.UnstartedIssues,
		StartedIssues: p.StartedIssues, CompletedIssues: p.CompletedIssues, CancelledIssues: p.CancelledIssues, Assignees: p.Assignees,
		Labels: p.Labels, Burndown: p.Burndown, FrozenAt: f.FrozenAt}
}
