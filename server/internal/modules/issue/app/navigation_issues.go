package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// NavigationIssuePlace is where a work item is, as a recording of a recent
// visit reads it (M7 P3d spec 2.7, R3): its workspace, which the recording
// decides on, and its project, which it locks FOR SHARE.
type NavigationIssuePlace struct {
	WorkspaceID, ProjectID uuid.UUID
}

// NavigationIssue is a work item as a recent visit is answered with it now
// (M7 P3d spec 2.4, 2.5, R3): its id, its project, its number, its name, its
// state, its priority as written, the accounts of its undeleted assignees'
// rows in id order, and whether it is in its project's intake
// (domain.Issue.InTriage), which keeps it out of the list until it is
// accepted. bootstrap converts it into navigation's value, whose fields are
// these.
type NavigationIssue struct {
	ID, ProjectID uuid.UUID
	SequenceID    int
	Name          string
	StateID       uuid.UUID
	Priority      string
	AssigneeIDs   []uuid.UUID
	InTriage      bool
}

// NavigationIssueRow is a work item as NavigationIssueReader reads it: what
// a NavigationIssue answers, with its creator, nil when not known, which
// the guest rule asks, and its undeleted intake row's status, nil for none,
// which InTriage asks.
type NavigationIssueRow struct {
	ID, ProjectID uuid.UUID
	SequenceID    int
	Name          string
	StateID       uuid.UUID
	Priority      domain.Priority
	AssigneeIDs   []uuid.UUID
	CreatedByID   *uuid.UUID
	Intake        *domain.IntakeFacts
}

// NavigationIssueReader reads the work items the navigation module's
// recent visits name (M7 P3d spec 2.5, R3): the undeleted ones among ids,
// archived or not, of any project, without a lock, in the transaction ctx
// carries or on the pool. NavigationIssuePlaces reads where each is, by
// id. NavigationIssueRows reads each as a NavigationIssueRow, in id order,
// its intake row joined as GetIssues joins it.
type NavigationIssueReader interface {
	NavigationIssuePlaces(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]NavigationIssuePlace, error)
	NavigationIssueRows(ctx context.Context, ids []uuid.UUID) ([]NavigationIssueRow, error)
}

// NavigationIssues tells the navigation module where the work items its
// recent visits name are, and which of them a caller sees now (M7 P3d spec
// 2.5, 2.7, R3): the guest rule and the intake are the module's own.
type NavigationIssues struct {
	rows NavigationIssueReader
}

// NewNavigationIssues returns the NavigationIssues that reads through rows.
func NewNavigationIssues(rows NavigationIssueReader) *NavigationIssues {
	return &NavigationIssues{rows: rows}
}

// NavigationIssuePlaces answers, by id, where each of the undeleted work
// items among ids is, archived or not, of any workspace: no decision, which
// is the caller's. It reads once, in the transaction ctx carries or on the
// pool, and not at all for no ids.
func (n *NavigationIssues) NavigationIssuePlaces(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]NavigationIssuePlace, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return n.rows.NavigationIssuePlaces(ctx, ids)
}

// NavigationIssues answers, by id, those of the work items ids that are
// undeleted, archived or not, in a project among grants, archived or not,
// and that the guest rule, domain.Issue.HiddenFrom, does not hide from
// reader by his grant there; another workspace's work item is in no project
// of grants. Each is answered with whether it is in its project's intake,
// domain.Issue.InTriage of its undeleted intake row. It reads once, in the
// transaction ctx carries or on the pool, and not at all for no ids.
func (n *NavigationIssues) NavigationIssues(ctx context.Context, reader uuid.UUID, grants []ProjectGrant, ids []uuid.UUID) (map[uuid.UUID]NavigationIssue, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := n.rows.NavigationIssueRows(ctx, ids)
	if err != nil {
		return nil, err
	}
	sees := seenByGrants(reader, grants)
	seen := make(map[uuid.UUID]NavigationIssue, len(rows))
	for _, row := range rows {
		if sees(row.ProjectID, row.CreatedByID) {
			seen[row.ID] = NavigationIssue{ID: row.ID, ProjectID: row.ProjectID, SequenceID: row.SequenceID, Name: row.Name, StateID: row.StateID,
				Priority: string(row.Priority), AssigneeIDs: row.AssigneeIDs, InTriage: domain.Issue{Intake: row.Intake}.InTriage()}
		}
	}
	return seen, nil
}
