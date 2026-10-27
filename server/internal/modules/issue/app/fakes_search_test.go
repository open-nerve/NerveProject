package app_test

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

type searchRows struct {
	log  *callLog
	in   []app.SearchRead
	rows []domain.Issue
}

func (f *searchRows) Search(ctx context.Context, in app.SearchRead) ([]domain.Issue, error) {
	f.in = append(f.in, in)
	return f.rows, f.log.add(ctx, "Search")
}
func searchFixture() (*app.SearchIssues, *world, *memberListDirectory, *fakeIdentifiers, *searchRows) {
	_, w, id := newGet()
	d := &memberListDirectory{log: w.log, projects: []app.VisibleProject{{ID: web, OwnOnly: true}}}
	ids := &fakeIdentifiers{log: w.log, projects: []app.ProjectIdentifier{{ID: ops, Identifier: "I"}, {ID: web, Identifier: "İ"}}}
	rows := &searchRows{log: w.log, rows: []domain.Issue{{ID: id, WorkspaceID: acme, ProjectID: web, CreatedByID: &alice, ParentID: &bob}}}
	w.issues.others = map[uuid.UUID]domain.Issue{bob: {ID: bob, ProjectID: web, CreatedByID: &carol}}
	w.issues.counts = map[uuid.UUID]app.SubIssueCount{id: {All: 4, Own: 2}}
	u := app.NewSearchIssues(app.SearchIssuesDeps{ReadTx: fakeTx{w.log}, Authorizer: w.auth,
		Workspaces: &fakeWorkspaces{log: w.log, slug: "acme", id: acme}, Projects: d, Identifiers: ids, Issues: w.issues, Locks: w.locks, Rows: rows})
	return u, w, d, ids, rows
}
