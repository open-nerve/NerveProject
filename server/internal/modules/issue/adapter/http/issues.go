package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// CreateIssue serves POST /api/v0/projects/{project_id}/issues: the state
// and the priority nil and the lists empty when not given; a value given
// goes to the use case as it came, which checks it.
func (h handler) CreateIssue(ctx context.Context, req gen.CreateIssueRequestObject) (gen.CreateIssueResponseObject, error) {
	in := domain.NewIssue{Name: req.Body.Name, StateID: req.Body.StateID, Priority: (*domain.Priority)(req.Body.Priority)}
	if req.Body.AssigneeIds != nil {
		in.AssigneeIDs = *req.Body.AssigneeIds
	}
	if req.Body.LabelIds != nil {
		in.LabelIDs = *req.Body.LabelIds
	}
	i, err := h.uc.CreateIssue.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.CreateIssue201JSONResponse(issueDetail(i)), nil
}

// GetIssue serves GET /api/v0/issues/{issue_id}.
func (h handler) GetIssue(ctx context.Context, req gen.GetIssueRequestObject) (gen.GetIssueResponseObject, error) {
	i, err := h.uc.GetIssue.Execute(ctx, req.IssueID)
	if err != nil {
		return nil, err
	}
	return gen.GetIssue200JSONResponse(issueDetail(i)), nil
}

// GetIssueByIdentifier serves GET
// /api/v0/workspaces/{slug}/issues/{identifier}: the identifier goes to
// the use case as it came, which reads it.
func (h handler) GetIssueByIdentifier(ctx context.Context, req gen.GetIssueByIdentifierRequestObject) (gen.GetIssueByIdentifierResponseObject,
	error) {
	i, err := h.uc.GetIssueByIdentifier.Execute(ctx, req.Slug, req.Identifier)
	if err != nil {
		return nil, err
	}
	return gen.GetIssueByIdentifier200JSONResponse(issueDetail(i)), nil
}

// UpdateIssue serves PATCH /api/v0/issues/{issue_id}: each field nil when
// not given; a value given goes to the use case as it came, which checks
// it.
func (h handler) UpdateIssue(ctx context.Context, req gen.UpdateIssueRequestObject) (gen.UpdateIssueResponseObject, error) {
	p := domain.IssuePatch{Name: req.Body.Name, StateID: req.Body.StateID, Priority: (*domain.Priority)(req.Body.Priority),
		AssigneeIDs: req.Body.AssigneeIds, LabelIDs: req.Body.LabelIds}
	i, err := h.uc.UpdateIssue.Execute(ctx, req.IssueID, p)
	if err != nil {
		return nil, err
	}
	return gen.UpdateIssue200JSONResponse(issueDetail(i)), nil
}

// DeleteIssue serves DELETE /api/v0/issues/{issue_id}.
func (h handler) DeleteIssue(ctx context.Context, req gen.DeleteIssueRequestObject) (gen.DeleteIssueResponseObject, error) {
	if err := h.uc.DeleteIssue.Execute(ctx, req.IssueID); err != nil {
		return nil, err
	}
	return gen.DeleteIssue204Response{}, nil
}

// ArchiveIssue serves POST /api/v0/issues/{issue_id}/archive.
func (h handler) ArchiveIssue(ctx context.Context, req gen.ArchiveIssueRequestObject) (gen.ArchiveIssueResponseObject, error) {
	i, err := h.uc.ArchiveIssue.Execute(ctx, req.IssueID)
	if err != nil {
		return nil, err
	}
	return gen.ArchiveIssue200JSONResponse(issueDetail(i)), nil
}

// UnarchiveIssue serves POST /api/v0/issues/{issue_id}/unarchive.
func (h handler) UnarchiveIssue(ctx context.Context, req gen.UnarchiveIssueRequestObject) (gen.UnarchiveIssueResponseObject, error) {
	i, err := h.uc.UnarchiveIssue.Execute(ctx, req.IssueID)
	if err != nil {
		return nil, err
	}
	return gen.UnarchiveIssue200JSONResponse(issueDetail(i)), nil
}

// issueDetail is i as the API shows it: each field that may be null null
// when it is nil, each list [] when it is empty.
func issueDetail(i domain.Issue) gen.IssueDetail {
	return gen.IssueDetail{ID: i.ID, WorkspaceID: i.WorkspaceID, ProjectID: i.ProjectID, SequenceID: i.SequenceID, Name: i.Name,
		StateID: i.StateID, Priority: gen.IssuePriority(i.Priority), ParentID: orNull(i.ParentID), SortOrder: i.SortOrder,
		CompletedAt: orNull(i.CompletedAt), ArchivedAt: orNull(i.ArchivedAt), AssigneeIds: orEmpty(i.AssigneeIDs), LabelIds: orEmpty(i.LabelIDs),
		CreatedByID: orNull(i.CreatedByID), UpdatedByID: orNull(i.UpdatedByID), CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt}
}

// orNull is v as a nullable field: null for nil.
func orNull[T any](v *T) nullable.Nullable[T] {
	if v == nil {
		return nullable.NewNullNullable[T]()
	}
	return nullable.NewNullableWithValue(*v)
}

// orEmpty is s, an empty list for nil: a list is never null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
