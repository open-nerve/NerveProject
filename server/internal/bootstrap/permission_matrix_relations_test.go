package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"uuid"
)

// The rows of the permission matrix of a work item's relations (M4 design
// 3.10, 9.2), naming the work item by its id: its column's work item
// (issueOf). A row of 9.2's table has the issue table's columns; a variant
// that asks what is checked after the decision, the project level's, as
// updateIssue's; one on the archived project, its table's.

// relatesTo is the request of a row whose callers each relate their
// column's work item to the one other gives, as blocking it.
func relatesTo(other func(c caller, s seeded) uuid.UUID) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return toWorkItem(http.MethodPost, "/relations", `{"relation_type":"blocking","issue_ids":["`+other(c, s).String()+`"]}`)(c, s)
	}
}

// beforesItem is the work item of another project that a column's cells
// relate their column's work item to: acme's private project's of the
// member before, which its admins and members see (PA, PM and PM+WA, who
// answer alike in both projects).
func beforesItem(_ caller, s seeded) uuid.UUID {
	return s.issue("acme/private", issueOf(callerBefore))
}

// archivedItem is the work item of acme's archived project, which may be
// related to none.
func archivedItem(_ caller, s seeded) uuid.UUID {
	return s.issue("acme/archived", matrixIssue)
}

func relationMatrixRows() []matrixRow {
	// The workspace admin who is only a guest of the project may relate his
	// own work item, as its admin (Grant.ProjectAdmin), but not to PM's,
	// which the guest rule hides from him: refused at its place (M4 design
	// 3.4).
	relates := onAWorkItem(cellCreated, cellCreated, cellCreated, cellForbidden)
	relates[callerAdminGuestOwn] = cellValidationFailed
	return []matrixRow{
		// The project's admins and members, the member who made it too, not
		// its guests, as updateIssue (M4 design 3.4, 9.2).
		{op: "createIssueRelations", write: true, columns: issueColumns, request: relatesTo(otherItemOf), cells: relates,
			refusal: "issue_ids[0] not_allowed", check: relatedTo(otherItemOf)},
		// An archived project's work items relate to none (M4 design 3.8),
		// after the decision.
		{op: "createIssueRelations", variant: "archived", write: true, columns: archivedColumns, request: relatesTo(otherItemOf),
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor do its archived work items: the project's 409 comes first.
		{op: "createIssueRelations", variant: "archived, its work item archived", write: true, columns: archivedColumns,
			prepare: inDone(true), request: relatesTo(otherItemOf), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor does an archived work item, after the decision, before the work
		// items asked for are looked at.
		{op: "createIssueRelations", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true),
			request: relatesTo(otherItemOf), cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellForbidden)},
		// A work item of another project of the workspace, which the caller
		// sees: related (M4 design 3.10).
		{op: "createIssueRelations", variant: "another project's work item", write: true, columns: projectColumns,
			request: relatesTo(beforesItem), cells: issueRows.of(cellCreated, cellCreated, cellIssueNotFound, cellCreated, cellForbidden, cellForbidden),
			check: relatedTo(beforesItem)},
		// The archived project's work item, after the decision: refused at
		// its place.
		{op: "createIssueRelations", variant: "an archived project's work item", write: true, columns: projectColumns,
			request: relatesTo(archivedItem), refusal: "issue_ids[0] not_allowed",
			cells: issueRows.of(cellValidationFailed, cellValidationFailed, cellIssueNotFound, cellValidationFailed, cellForbidden, cellForbidden)},
	}
}

// relatedTo is the check of a row whose callers relate their column's work
// item to the one other gives (relatesTo): the answer is that relation
// alone, blocking it as the column's work item reads it, the other work
// item changed last by the caller.
func relatedTo(other func(c caller, s seeded) uuid.UUID) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var got struct {
			Data []struct {
				RelationType string `json:"relation_type"`
				Issue        struct {
					ID          uuid.UUID  `json:"id"`
					UpdatedByID *uuid.UUID `json:"updated_by_id"`
				} `json:"issue"`
			} `json:"data"`
		}
		decodeAnswer(t, answer, &got)
		me := s.account(accountOf(c))
		if len(got.Data) != 1 || got.Data[0].RelationType != "blocking" || got.Data[0].Issue.ID != other(c, s) ||
			!reflect.DeepEqual(got.Data[0].Issue.UpdatedByID, &me) {
			t.Errorf("%s relates %s; want blocking %s alone, changed last by the caller", c, answer, other(c, s))
		}
	}
}
