package app_test

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ops is acme's other project.
var ops = uuid.NewV7()

// fakeWorkspaces is the workspace directory: slug names id, each read
// logged.
type fakeWorkspaces struct {
	log  *callLog
	slug string
	id   uuid.UUID
}

func (f *fakeWorkspaces) WorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error) {
	if err := f.log.add(ctx, "WorkspaceBySlug", slug); err != nil || slug != f.slug {
		return uuid.UUID{}, false, err
	}
	return f.id, true, nil
}

// fakeIdentifiers are a workspace's projects with their identifiers, each
// read logged.
type fakeIdentifiers struct {
	log      *callLog
	projects []app.ProjectIdentifier
}

func (f *fakeIdentifiers) Identifiers(ctx context.Context, workspaceID uuid.UUID) ([]app.ProjectIdentifier, error) {
	if err := f.log.add(ctx, "Identifiers", workspaceID); err != nil {
		return nil, err
	}
	return f.projects, nil
}

// fakeNumbers is the work item id, its project's of the number, each read
// logged.
type fakeNumbers struct {
	log     *callLog
	project uuid.UUID
	number  int
	id      uuid.UUID
}

func (f *fakeNumbers) IssueByNumber(ctx context.Context, projectID uuid.UUID, number int) (uuid.UUID, bool, error) {
	if err := f.log.add(ctx, "IssueByNumber", projectID, number); err != nil || projectID != f.project || number != f.number {
		return uuid.UUID{}, false, err
	}
	return f.id, true, nil
}

// byIdentifier is getIssueByIdentifier's world: getIssue's (newGet), its
// work item web's 7, in acme, whose slug is acme, beside ops; their
// identifiers OPS and WEB.
type byIdentifier struct {
	*world
	projects *fakeIdentifiers
}

func newGetByIdentifier() (*app.GetIssueByIdentifier, byIdentifier, uuid.UUID) {
	_, w, id := newGet()
	b := byIdentifier{world: w, projects: &fakeIdentifiers{log: w.log, projects: []app.ProjectIdentifier{{ID: ops, Identifier: "OPS"},
		{ID: web, Identifier: "WEB"}}}}
	uc := app.NewGetIssueByIdentifier(app.GetIssueByIdentifierDeps{Locks: w.locks, Workspaces: &fakeWorkspaces{log: w.log, slug: "acme", id: acme},
		Projects: b.projects, Issues: &fakeNumbers{log: w.log, project: web, number: 7, id: id}})
	return uc, b, id
}

// identified are the calls of alice's read of web's work item number in
// acme, read as id: acme by its slug and the decision in it, outside any
// transaction and without a lock; acme's projects' identifiers; web's work
// item of the number; then the work item, and the decision on it, as
// getIssue reads it.
func identified(number int, id uuid.UUID) []string {
	return []string{line("WorkspaceBySlug", "acme", "outside tx"), line("Authorize", alice, domain.ActionSearch, acme, uuid.UUID{}, "outside tx"),
		line("Identifiers", acme, "outside tx"), line("IssueByNumber", web, number, "outside tx"), line("GetIssue", id, "outside tx"),
		line("Authorize", alice, domain.ActionRead, acme, web, "outside tx")}
}

// The work item, as getIssue answers it, whichever case its project's
// identifier is asked in, Turkish letters too: WEB-7 is web's 7, not
// ops's, and so is iş-7 once web's identifier is İŞ (M4 design 3.7).
func TestGetIssueByIdentifierReadsTheWorkItem(t *testing.T) {
	for _, tt := range []struct{ identifier, stored string }{{"WEB-7", "WEB"}, {"web-7", "WEB"}, {"wEb-7", "WEB"}, {"iş-7", "İŞ"}} {
		uc, b, id := newGetByIdentifier()
		b.projects.projects[1].Identifier = tt.stored
		got, err := uc.Execute(as(alice), "acme", tt.identifier)
		if want := readIssue(id); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("Execute(%s) = %+v, %v; want %+v", tt.identifier, got, err, want)
		}
		if calls := identified(7, id); !slices.Equal(b.log.calls, calls) {
			t.Errorf("%s: calls:\n%q\nwant\n%q", tt.identifier, b.log.calls, calls)
		}
	}
}

// A workspace that is not there, or of which the caller is no member, is
// workspace.not_found, before any work item is looked for; a role outside
// issue.search, forbidden. Then an identifier that is no identifier, of no
// project of acme, of a number no work item of web has, of a work item
// whose project the caller does not see, or that the guest rule hides
// from him, is issue.not_found, one answer for each, detail and all; a
// role that may not read it, forbidden. A read or a decision that fails,
// its failure as itself; a request without a caller, unauthenticated
// before any read. Each answer is compared whole, not by its code alone.
func TestGetIssueByIdentifierRefusals(t *testing.T) {
	for _, tt := range []struct {
		name       string
		set        func(b byIdentifier)
		ctx        context.Context
		slug       string
		identifier string
		want       error
		calls      int // of identified's
	}{
		{"no workspace", func(byIdentifier) {}, as(alice), "acne", "WEB-7", domain.ErrWorkspaceNotFound, 1},
		{"a workspace he is no member of", func(b byIdentifier) { b.auth.answers = []error{shared.ErrNotVisible} }, as(alice), "acme", "WEB-7",
			domain.ErrWorkspaceNotFound, 2},
		{"a workspace role outside the rule", func(b byIdentifier) { b.auth.answers = []error{shared.Forbidden()} }, as(alice), "acme", "WEB-7",
			shared.Forbidden(), 2},
		{"no identifier", func(byIdentifier) {}, as(alice), "acme", "WEB7", domain.ErrIssueNotFound, 2},
		{"no project's identifier", func(byIdentifier) {}, as(alice), "acme", "WEBS-7", domain.ErrIssueNotFound, 3},
		{"no work item of the number", func(byIdentifier) {}, as(alice), "acme", "WEB-8", domain.ErrIssueNotFound, 4},
		{"a project not seen", func(b byIdentifier) { b.auth.answers = []error{nil, shared.ErrNotVisible} }, as(alice), "acme", "WEB-7",
			domain.ErrIssueNotFound, 6},
		{"another's, to a guest who sees only his own", func(b byIdentifier) { b.auth.grants = []shared.Grant{{}, ownOnly} }, as(alice), "acme",
			"WEB-7", domain.ErrIssueNotFound, 6},
		{"a role not allowed", func(b byIdentifier) { b.auth.answers = []error{nil, shared.Forbidden()} }, as(alice), "acme", "WEB-7",
			shared.Forbidden(), 6},
		{"the workspace's read fails", func(b byIdentifier) { b.log.fail = "WorkspaceBySlug" }, as(alice), "acme", "WEB-7", errDisk, 1},
		{"the decision in the workspace fails", func(b byIdentifier) { b.log.fail = "Authorize" }, as(alice), "acme", "WEB-7", errDisk, 2},
		{"the identifiers' read fails", func(b byIdentifier) { b.log.fail = "Identifiers" }, as(alice), "acme", "WEB-7", errDisk, 3},
		{"the number's read fails", func(b byIdentifier) { b.log.fail = "IssueByNumber" }, as(alice), "acme", "WEB-7", errDisk, 4},
		{"the work item's read fails", func(b byIdentifier) { b.log.fail = "GetIssue" }, as(alice), "acme", "WEB-7", errDisk, 5},
		{"no caller", func(byIdentifier) {}, context.Background(), "acme", "WEB-7", shared.Unauthenticated(), 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, b, id := newGetByIdentifier()
			tt.set(b)
			got, err := uc.Execute(tt.ctx, tt.slug, tt.identifier)
			if !reflect.DeepEqual(err, tt.want) || got.ID != (uuid.UUID{}) {
				t.Errorf("Execute(%s, %s) = %+v, %v; want no work item and %v", tt.slug, tt.identifier, got, err, tt.want)
			}
			number := 7
			if tt.identifier == "WEB-8" {
				number = 8
			}
			calls := identified(number, id)[:tt.calls]
			if tt.calls > 0 {
				calls[0] = line("WorkspaceBySlug", tt.slug, "outside tx")
			}
			if !slices.Equal(b.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", b.log.calls, calls)
			}
		})
	}
}
