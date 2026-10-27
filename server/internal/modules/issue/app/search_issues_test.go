package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Matching before narrowing the catalog makes the hidden uppercase I steal
// visible İ; seen outside this transaction leaks parent/count information.
func TestSearchVisibleIdentifierAndSeen(t *testing.T) {
	u, w, _, _, rows := searchFixture()
	got, err := u.Execute(as(alice), "acme", map[string][]string{"q": {"i-7"}})
	if err != nil || len(got) != 1 || got[0].ParentID != nil || got[0].SubIssuesCount != 2 {
		t.Fatal(got, err)
	}
	if len(rows.in) != 1 || rows.in[0].ExactProjectID == nil || *rows.in[0].ExactProjectID != web || rows.in[0].ActorID != alice {
		t.Fatal(rows.in)
	}
	want := []string{"Begin", line("WorkspaceBySlug", "acme"), line("Authorize", alice, domain.ActionSearch, acme, uuid.Nil()), line("Visible", acme, alice), line("Identifiers", acme), "Search", line("SubIssueCounts", alice, []uuid.UUID{got[0].ID}), line("GetIssues", []uuid.UUID{bob})}
	if !slices.Equal(w.log.calls, want) {
		t.Fatal(w.log.calls, want)
	}
}
func TestSearchIdentifierUsesExplicitScope(t *testing.T) {
	u, _, d, _, rows := searchFixture()
	d.projects = append(d.projects, app.VisibleProject{ID: ops})
	_, err := u.Execute(as(alice), "acme", map[string][]string{"q": {"i-7"}, "project_id": {web.String()}})
	if err != nil || len(rows.in) != 1 || rows.in[0].ExactProjectID == nil || *rows.in[0].ExactProjectID != web || len(rows.in[0].Projects) != 1 {
		t.Fatal(rows.in, err)
	}
}
func TestSearchSourceReadFailure(t *testing.T) {
	u, w, _, _, _ := searchFixture()
	w.log.fail = "GetIssue"
	_, err := u.Execute(as(alice), "acme", map[string][]string{"for": {"parent"}, "issue_id": {w.issues.issue.ID.String()}})
	if !errors.Is(err, errDisk) || len(w.log.calls) != 5 {
		t.Fatal(err, w.log.calls)
	}
}

func TestSearchPreflightAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name  string
		ctx   context.Context
		raw   map[string][]string
		auth  error
		slug  string
		fail  string
		want  error
		calls int
	}{
		{name: "actor first", ctx: context.Background(), raw: map[string][]string{"limit": {"bad"}}, want: shared.Unauthenticated()},
		{name: "syntax before tx", ctx: as(alice), raw: map[string][]string{"limit": {"bad"}}},
		{name: "missing workspace", ctx: as(alice), slug: "missing", want: domain.ErrWorkspaceNotFound, calls: 2},
		{name: "invisible workspace", ctx: as(alice), auth: shared.ErrNotVisible, want: domain.ErrWorkspaceNotFound, calls: 3},
		{name: "guest", ctx: as(alice), auth: shared.Forbidden(), want: shared.Forbidden(), calls: 3},
		{name: "workspace error", ctx: as(alice), fail: "WorkspaceBySlug", want: errDisk, calls: 2},
		{name: "auth error", ctx: as(alice), fail: "Authorize", want: errDisk, calls: 3},
		{name: "directory error", ctx: as(alice), fail: "Visible", want: errDisk, calls: 4},
		{name: "catalog error", ctx: as(alice), fail: "Identifiers", want: errDisk, calls: 5},
		{name: "rows error", ctx: as(alice), fail: "Search", want: errDisk, calls: 6},
		{name: "count error", ctx: as(alice), fail: "SubIssueCounts", want: errDisk, calls: 7},
		{name: "parents error", ctx: as(alice), fail: "GetIssues", want: errDisk, calls: 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, w, _, _, _ := searchFixture()
			w.auth.answers = []error{tc.auth}
			w.log.fail = tc.fail
			slug := tc.slug
			if slug == "" {
				slug = "acme"
			}
			got, err := u.Execute(tc.ctx, slug, tc.raw)
			if tc.name == "syntax before tx" {
				if err == nil {
					t.Fatal("syntax accepted")
				}
			} else if !errors.Is(err, tc.want) {
				t.Fatal(err, tc.want)
			}
			if got != nil || len(w.log.calls) != tc.calls {
				t.Fatal(got, w.log.calls)
			}
			for _, call := range w.log.calls {
				if strings.Contains(call, "outside tx") {
					t.Fatal(call)
				}
			}
		})
	}
	u := app.NewSearchIssues(app.SearchIssuesDeps{ReadTx: activityReadFailureTx{}})
	if _, err := u.Execute(as(alice), "acme", nil); !errors.Is(err, errDisk) {
		t.Fatal(err)
	}
}
