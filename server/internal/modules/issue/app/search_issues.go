package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type SearchIssuesDeps struct {
	ReadTx      shared.TxManager
	Authorizer  shared.Authorizer
	Workspaces  WorkspaceDirectory
	Projects    VisibleProjects
	Identifiers ProjectIdentifiers
	Issues      IssueReader
	Locks       Locks
	Rows        SearchRows
}

// SearchIssues keeps authorization, candidate reads and their visible shape in
// one read snapshot. Pure syntax and actor checks precede the transaction.
type SearchIssues struct{ d SearchIssuesDeps }

func NewSearchIssues(d SearchIssuesDeps) *SearchIssues { return &SearchIssues{d: d} }
func (u *SearchIssues) Execute(ctx context.Context, slug string, raw map[string][]string) ([]domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	query, err := domain.ParseSearchQuery(raw)
	if err != nil {
		return nil, err
	}
	var out []domain.Issue
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		wid, found, err := u.d.Workspaces.WorkspaceBySlug(ctx, slug)
		if err != nil {
			return err
		}
		if !found {
			return domain.ErrWorkspaceNotFound
		}
		_, err = u.d.Authorizer.Authorize(ctx, actor, domain.ActionSearch, shared.Target{WorkspaceID: wid})
		if errors.Is(err, shared.ErrNotVisible) {
			return domain.ErrWorkspaceNotFound
		}
		if err != nil {
			return err
		}
		visible, err := u.d.Projects.Visible(ctx, wid, actor.UserID)
		if err != nil {
			return err
		}
		g := make(grants, len(visible))
		for _, p := range visible {
			g[p.ID] = shared.Grant{OwnOnly: p.OwnOnly}
		}
		if query.ProjectID != nil {
			if _, ok := g[*query.ProjectID]; !ok {
				return domain.ErrProjectNotFound
			}
		}
		scope := visible
		if query.IssueID != nil {
			source, found, err := u.d.Issues.GetIssue(ctx, *query.IssueID)
			if err != nil {
				return err
			}
			grant, ok := g[source.ProjectID]
			if !found || source.WorkspaceID != wid || !ok || source.ArchivedAt != nil || source.HiddenFrom(actor.UserID, grant.OwnOnly) {
				return domain.ErrIssueNotFound
			}
			if query.For == "parent" || query.For == "sub_issue" {
				if query.ProjectID != nil && *query.ProjectID != source.ProjectID {
					return shared.Invalid(shared.FieldError{Field: "project_id", Code: shared.FieldNotAllowed, Message: "must match the source issue's project"})
				}
				query.ProjectID = &source.ProjectID
			}
		}
		if query.ProjectID != nil {
			scope = nil
			for _, p := range visible {
				if p.ID == *query.ProjectID {
					scope = append(scope, p)
				}
			}
		}
		identifiers, err := u.d.Identifiers.Identifiers(ctx, wid)
		if err != nil {
			return err
		}
		// Match only the candidate scope; hidden or unselected folded duplicates
		// must never change which visible project's number is requested.
		var names []string
		var projectIDs []uuid.UUID
		for _, id := range identifiers {
			for _, p := range scope {
				if id.ID == p.ID {
					names = append(names, id.Identifier)
					projectIDs = append(projectIDs, id.ID)
					break
				}
			}
		}
		in := SearchRead{ActorID: actor.UserID, Projects: scope, Query: query}
		if index, ok := domain.MatchIdentifier(query.Identifier, names); ok && query.IdentifierNumber != nil {
			in.ExactProjectID = &projectIDs[index]
		}
		rows, err := u.d.Rows.Search(ctx, in)
		if err != nil {
			return err
		}
		out, err = u.d.Locks.seen(ctx, actor.UserID, g, rows...)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
