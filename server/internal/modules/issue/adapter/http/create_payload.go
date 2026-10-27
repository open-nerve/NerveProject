package httpadapter

import (
	"encoding/json"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// IssueCreatePayloadDecoder applies the create contract to stored JSON bytes.
// Business validation stays with the application after structural validation.
type IssueCreatePayloadDecoder struct{}

// DecodeCreatePayload checks the original create body structure before decoding.
func (IssueCreatePayloadDecoder) DecodeCreatePayload(raw []byte) (domain.NewIssue, error) {
	if err := gen.BodyShapes().Check("POST /api/v0/projects/{project_id}/issues", raw); err != nil {
		return domain.NewIssue{}, err
	}
	var body gen.IssueCreate
	if err := json.Unmarshal(raw, &body); err != nil {
		return domain.NewIssue{}, err
	}
	return newIssueInput(body), nil
}

// newIssueInput preserves omitted fields and explicitly empty lists alike.
func newIssueInput(body gen.IssueCreate) domain.NewIssue {
	in := domain.NewIssue{DescriptionHTML: body.DescriptionHTML, Name: body.Name, StateID: body.StateID, Priority: (*domain.Priority)(body.Priority), StartDate: body.StartDate, TargetDate: body.TargetDate, ParentID: body.ParentID}
	if body.AssigneeIds != nil {
		in.AssigneeIDs = *body.AssigneeIds
	}
	if body.LabelIds != nil {
		in.LabelIDs = *body.LabelIds
	}
	return in
}
