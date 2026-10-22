package app

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssueRelation is a relation as one of its work items reads it (M4 design
// 3.10, 5.2): its id, its type from the work item, the other work item as
// the caller sees it, and when it was made.
type IssueRelation struct {
	ID        uuid.UUID
	Type      domain.RelationType
	Issue     domain.Issue
	CreatedAt time.Time
}

// CreateIssueRelationsDeps are what createIssueRelations reads and writes
// through.
type CreateIssueRelationsDeps struct {
	Locks     Locks
	Relations RelationCreator
	Clock     Clock
}

// CreateIssueRelations relates a work item to others: POST
// /api/v0/issues/{issue_id}/relations (M4 design 3.5, 3.10, 5.1).
type CreateIssueRelations struct {
	d CreateIssueRelationsDeps
}

// NewCreateIssueRelations returns the use case.
func NewCreateIssueRelations(d CreateIssueRelationsDeps) *CreateIssueRelations {
	return &CreateIssueRelations{d: d}
}

// Execute checks t and ids (domain.CheckNewRelations), before any read;
// then Locks runs the write on the work item id across projects (M4 design
// 3.5, 11.11, convention 7): its precheck, the decision on issue.update on
// it, the guest rule first, then each of ids read and decided on, on
// issue.read, the guest rule first, before any lock, so that one the
// caller may not reach takes no lock; then the write gates of the work
// item's project and of the others' it reaches, all at once in ascending
// id order, which refuse a write that may wait no longer (503
// server_busy); then in one transaction those projects FOR NO KEY UPDATE,
// in ascending id order, the work item and each other read and decided on
// again, and under them, in this order: an archived project refuses it
// (409 project.archived), then an archived work item (409 issue.archived);
// each of ids reached, not archived, of a project not archived, and not
// related to the work item yet (check), every problem in one 422; then the
// clock; the relations, in the others' id order, each as stored
// (domain.NewRelation); the work item and each other last changed by the
// caller; for each relation a row of the work item's activity and one of
// the other's, each of its own project, the type as its work item reads
// it, the other work item by its identifier (rows 20 and 21 of 3.13). All
// of it is the caller's at that moment. The answer is the relations as the
// work item reads them, in the others' id order, each other read back in
// the transaction, as the caller sees it (Locks.seen).
func (u *CreateIssueRelations) Execute(ctx context.Context, id uuid.UUID, t domain.RelationType, ids []uuid.UUID) ([]IssueRelation, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if err := domain.CheckNewRelations(t, ids); err != nil {
		return nil, err
	}
	w := write{issue: &id, action: domain.ActionUpdate, notFound: domain.ErrIssueNotFound, reach: ids}
	var created []IssueRelation
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		switch {
		case h.project.Archived:
			return domain.ErrProjectArchived
		case h.issue.ArchivedAt != nil:
			return domain.ErrIssueArchived
		}
		others, err := u.check(ctx, h)
		if err != nil {
			return err
		}
		relations, err := u.write(ctx, h, actor.UserID, t, others)
		if err != nil {
			return err
		}
		created, err = u.answer(ctx, actor.UserID, h, relations, others)
		return err
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// check holds the work items the caller's addition relates the work item h
// holds to, as h reached them under the locks (M4 design 3.4, 3.8, 3.10):
// each reached, not archived, of a project that is not archived, and not
// related to the work item yet, of any type, either way. Every problem is
// reported at once, in one 422, each at its place in the addition's work
// items. It answers them, in id order.
func (u *CreateIssueRelations) check(ctx context.Context, h held) ([]end, error) {
	var reached []uuid.UUID
	for _, e := range h.reached {
		if e.ok {
			reached = append(reached, e.issue.ID)
		}
	}
	var related []uuid.UUID
	if len(reached) > 0 {
		var err error
		if related, err = u.d.Relations.Related(ctx, h.issue.ID, slices.SortedFunc(slices.Values(reached), uuid.UUID.Compare)); err != nil {
			return nil, err
		}
	}
	var problems []shared.FieldError
	for n, e := range h.reached {
		switch {
		case !e.ok || e.issue.ArchivedAt != nil || e.project.Archived:
			problems = append(problems, domain.RelatedNotAllowed(n))
		case slices.Contains(related, e.issue.ID):
			problems = append(problems, domain.RelatedAlready(n))
		}
	}
	if len(problems) > 0 {
		return nil, shared.Invalid(problems...)
	}
	others := slices.Clone(h.reached)
	slices.SortFunc(others, func(a, b end) int { return a.issue.ID.Compare(b.issue.ID) })
	return others, nil
}

// write relates the work item h holds to others, in their order, the
// caller by's at the clock's moment (M4 design 3.7, 3.10, 3.13): the
// relations, as stored (domain.NewRelation); the work item and each other
// last changed by him, in id order; for each relation a row of the work
// item's activity and one of the other's, each of its own project, the
// type as its work item reads it, the other by its identifier. It answers
// the relations.
func (u *CreateIssueRelations) write(ctx context.Context, h held, by uuid.UUID, t domain.RelationType, others []end) ([]domain.Relation,
	error) {
	at := u.d.Clock.Now()
	from := end{issue: h.issue, project: h.project}
	relations := make([]domain.Relation, len(others))
	changed := []uuid.UUID{h.issue.ID}
	var rows []ActivityRow
	for n, o := range others {
		relations[n] = domain.NewRelation(uuid.NewV7(), h.issue, o.issue, t, by, at)
		changed = append(changed, o.issue.ID)
		rows = append(rows, from.stamp(by, at).activity([]domain.Activity{domain.RelationAdded(t, *refOf(o.project, o.issue))})...)
		rows = append(rows, o.stamp(by, at).activity([]domain.Activity{domain.RelationAdded(t.Reversed(), *refOf(h.project, h.issue))})...)
	}
	if err := u.d.Relations.InsertRelations(ctx, relations); err != nil {
		return nil, err
	}
	slices.SortFunc(changed, uuid.UUID.Compare)
	if err := u.d.Relations.MarkChanged(ctx, ChangedRows{IDs: changed, By: by, At: at}); err != nil {
		return nil, err
	}
	if err := u.d.Relations.InsertActivities(ctx, rows); err != nil {
		return nil, err
	}
	return relations, nil
}

// answer is the relations as the work item h holds reads them, each with
// its other end, of others, read back in the transaction as the caller
// actor sees it (Locks.seen), by his grants under the locks. One not there
// to read back is an error: the write found each undeleted under the
// locks.
func (u *CreateIssueRelations) answer(ctx context.Context, actor uuid.UUID, h held, relations []domain.Relation, others []end) (
	[]IssueRelation, error) {
	ids, g := make([]uuid.UUID, len(others)), grants{h.issue.ProjectID: h.grant}
	for n, o := range others {
		ids[n], g[o.issue.ProjectID] = o.issue.ID, o.grant
	}
	stored, err := u.d.Relations.GetIssues(ctx, ids)
	switch {
	case err != nil:
		return nil, err
	case len(stored) != len(ids):
		return nil, fmt.Errorf("work items %v are not all there after their write", ids)
	}
	seen, err := u.d.Locks.seen(ctx, actor, g, stored...)
	if err != nil {
		return nil, err
	}
	out := make([]IssueRelation, len(relations))
	for n, r := range relations {
		out[n] = IssueRelation{ID: r.ID, Type: r.TypeFrom(h.issue.ID), Issue: seen[n], CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// stamp is where the rows of e's work item go, the caller by's at the
// moment at.
func (e end) stamp(by uuid.UUID, at time.Time) stamp {
	return stamp{workspace: e.issue.WorkspaceID, project: e.issue.ProjectID, issue: e.issue.ID, by: by, at: at}
}
