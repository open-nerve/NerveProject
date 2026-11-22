package domain_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A recent visit is of a work item or of a project, as they are written (P3d spec 2.3, 2.4): any other entity_type,
// in the body or in the query, is one 422, validation_failed with entity_type's invalid_format, before anything is
// read (spec 2.7). Plane's other kinds (D37) and a kind in another case are among them.
func TestARecentVisitsKind(t *testing.T) {
	for _, k := range []domain.RecentVisitKind{domain.RecentVisitIssue, domain.RecentVisitProject} {
		if err := domain.CheckRecentVisitKind(k); err != nil {
			t.Fatalf("%s refused: %v", k, err)
		}
	}
	if domain.RecentVisitIssue != "issue" || domain.RecentVisitProject != "project" {
		t.Fatalf("kinds %q and %q; want issue and project", domain.RecentVisitIssue, domain.RecentVisitProject)
	}
	for _, k := range []domain.RecentVisitKind{"page", "cycle", "module", "view", "folder", "", "Issue", "PROJECT", " issue"} {
		err := domain.CheckRecentVisitKind(k)
		var problem *shared.Error
		if !errors.As(err, &problem) || problem.Code != shared.CodeValidationFailed || problem.ProblemStatus() != 422 ||
			!reflect.DeepEqual(fields(problem.Fields), []string{"entity_type invalid_format"}) {
			t.Fatalf("%q: %v; want validation_failed with entity_type invalid_format alone", k, err)
		}
	}
}

// One keeps at most 20 undeleted recent visits in a workspace, of both kinds together (spec 2.5): the recording's
// trim keeps that many.
func TestTheCapOfRecentVisits(t *testing.T) {
	if domain.RecentVisitCap != 20 {
		t.Fatalf("cap %d; want 20", domain.RecentVisitCap)
	}
}

// A recording of a work item one does not see is answered as a work item that is not there (spec 2.4, 2.7): with
// the issue module's code and words, so the answer is the one /issues/{issue_id} gives (iss/domain/errors.go).
func TestAWorkItemNotFound(t *testing.T) {
	want := shared.NewError(shared.KindNotFound, "issue.not_found", "The work item does not exist, or you cannot see it.")
	if !reflect.DeepEqual(domain.ErrIssueNotFound, want) {
		t.Fatalf("%+v; want %+v", domain.ErrIssueNotFound, want)
	}
}
