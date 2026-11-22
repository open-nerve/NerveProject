package app_test

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// shownOf is v as the list answers it, with r's project of it and, a work item's, r's work item.
func shownOf(r *recents, v domain.RecentVisit) app.ShownRecentVisit {
	shown := app.ShownRecentVisit{Visit: v, Project: r.projectTargets[v.ProjectID]}
	if v.Kind == domain.RecentVisitIssue {
		i := r.issues[v.EntityID].summary
		shown.Issue = &i
	}
	return shown
}

// The list answers, in the store's order, her visits whose targets she sees now and that are listed (spec 2.4, 2.5,
// R5, R6), each with its project and its work item now, in one read-only snapshot: the workspace and
// recent_visit.read's decision, her visits there, then the projects she reads, the work items she sees and the
// projects, each once. Listed: a work item and a project she reads, an archived project and its work item. Kept, not
// listed: a work item in its project's intake; a work item and a project she no longer reads (she left ops), or does
// not, the workspace's admin who is none of lobby's members; a work item deleted, and a project deleted with its work
// items. Where she sees only her own, another's work item is not listed, her own is.
func TestListRecentVisitsSheSees(t *testing.T) {
	gone := uuid.NewV7()
	for _, tt := range []struct {
		name   string
		grants []app.ProjectGrant
		listed []int
	}{
		{"she reads web and attic", []app.ProjectGrant{{ProjectID: web}, {ProjectID: attic}}, []int{0, 1, 2, 4, 5}},
		{"she sees only her own in web", []app.ProjectGrant{{ProjectID: web, OwnOnly: true}, {ProjectID: attic}}, []int{1, 2, 4, 5}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := newRecents()
			r.grants[acme] = tt.grants
			r.names[gone] = "gone"
			r.decide = func(shared.Action, shared.Target, bool) (shared.Grant, error) { return workspaceAdmin, nil }
			r.visits = []domain.RecentVisit{visitOf(domain.RecentVisitIssue, task, web, 1), visitOf(domain.RecentVisitProject, web, web, 2),
				visitOf(domain.RecentVisitIssue, chore, web, 3), visitOf(domain.RecentVisitIssue, triaged, web, 4),
				visitOf(domain.RecentVisitIssue, relic, attic, 5), visitOf(domain.RecentVisitProject, attic, attic, 6),
				visitOf(domain.RecentVisitIssue, toil, ops, 7), visitOf(domain.RecentVisitProject, ops, ops, 8),
				visitOf(domain.RecentVisitProject, lobby, lobby, 9), visitOf(domain.RecentVisitIssue, gone, web, 10),
				visitOf(domain.RecentVisitProject, gone, gone, 11)}
			got, err := r.listVisits().Execute(as(t), "acme", nil)
			if err != nil {
				t.Fatal(err)
			}
			want := make([]app.ShownRecentVisit, len(tt.listed))
			for n, i := range tt.listed {
				want[n] = shownOf(r, r.visits[i])
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("listed\n %+v\nwant\n %+v", got, want)
			}
			wantCalls(t, r.world, "begin read", "slug acme", "authorize alice recent_visit.read acme/-", "visits alice acme any",
				"readable alice acme", "issue summaries "+r.idsOf([]uuid.UUID{task, chore, triaged, relic, toil, gone}),
				"project targets "+r.idsOf([]uuid.UUID{web, attic, ops, lobby, gone}), "end read")
			wroteNoVisit(t, r)
		})
	}
}

// The list drops what it does not list before it takes the cap (spec 2.5, R8): of 21 undeleted visits of web's work
// items, the cap's transient 21st among them, the 5th is not listed (its work item is in its project's intake), so the
// list answers the other 20, the 21st too. Taking 20 first would answer 19.
func TestListRecentVisitsFiltersBeforeItTakesTheCap(t *testing.T) {
	r := newRecents()
	for n := range 21 {
		id := uuid.NewV7()
		r.issues[id] = recentIssue{place: app.IssuePlace{WorkspaceID: acme, ProjectID: web}, creator: bob,
			summary: app.IssueSummary{ID: id, ProjectID: web, SequenceID: 100 + n, Name: "Work", StateID: web, Priority: "none", InTriage: n == 4}}
		r.visits = append(r.visits, visitOf(domain.RecentVisitIssue, id, web, n))
	}
	got, err := r.listVisits().Execute(as(t), "acme", nil)
	if err != nil {
		t.Fatal(err)
	}
	var want []app.ShownRecentVisit
	for n, v := range r.visits {
		if n != 4 {
			want = append(want, shownOf(r, v))
		}
	}
	if len(got) != domain.RecentVisitCap || !reflect.DeepEqual(got, want) {
		t.Errorf("listed %d: %+v\nwant the 20 but the 5th, the 21st among them", len(got), got)
	}
}

// The list of one kind asks the store for that kind alone, and answers what it reads (spec 2.4); with no visit it reads
// no target.
func TestListRecentVisitsOfOneKind(t *testing.T) {
	for _, kind := range []domain.RecentVisitKind{domain.RecentVisitIssue, domain.RecentVisitProject} {
		r := newRecents()
		if got, err := r.listVisits().Execute(as(t), "acme", &kind); err != nil || len(got) != 0 {
			t.Errorf("%s: listed %+v, %v; want none", kind, got, err)
		}
		wantCalls(t, r.world, "begin read", "slug acme", "authorize alice recent_visit.read acme/-", "visits alice acme "+string(kind), "end read")
	}
}

// The refusals of the list, in order (spec 2.7): a kind not one of the two is a 422 at entity_type before anything is
// read, also on a slug of no workspace and as the workspace's guest; then a slug of none and a workspace she is not a
// member of are workspace.not_found, the one value, and no visit is read. The workspace's guest lists hers.
func TestTheRefusalsOfAListOfRecentVisits(t *testing.T) {
	guest := func(shared.Action, shared.Target, bool) (shared.Grant, error) { return workspaceGuest, nil }
	for _, slug := range []string{"acme", "nowhere"} {
		r := newRecents()
		r.decide = guest
		page := domain.RecentVisitKind("page")
		_, err := r.listVisits().Execute(as(t), slug, &page)
		if fields := fieldsOf(t, err); len(fields) != 1 || fields[0].Field != "entity_type" || fields[0].Code != shared.FieldInvalidFormat {
			t.Errorf("%s: fields %+v; want entity_type's invalid_format alone", slug, fields)
		}
		wantCalls(t, r.world)
	}
	for _, tt := range []struct {
		name, slug string
		decide     func(shared.Action, shared.Target, bool) (shared.Grant, error)
		calls      []string
	}{
		{"a slug of none", "nowhere", nil, []string{"begin read", "slug nowhere", "end read"}},
		{"a workspace she is no member of", "acme", func(shared.Action, shared.Target, bool) (shared.Grant, error) {
			return shared.Grant{}, shared.ErrNotVisible
		}, []string{"begin read", "slug acme", "authorize alice recent_visit.read acme/-", "end read"}},
	} {
		r := newRecents()
		if tt.decide != nil {
			r.decide = tt.decide
		}
		got, err := r.listVisits().Execute(as(t), tt.slug, nil)
		if got != nil {
			t.Errorf("%s: listed %+v; want none", tt.name, got)
		}
		wantSame(t, err, domain.ErrWorkspaceNotFound)
		wantCalls(t, r.world, tt.calls...)
	}
	r := newRecents()
	r.decide = guest
	r.visits = []domain.RecentVisit{visitOf(domain.RecentVisitProject, web, web, 1)}
	if got, err := r.listVisits().Execute(as(t), "acme", nil); err != nil || len(got) != 1 {
		t.Errorf("the workspace's guest: listed %+v, %v; want her one visit", got, err)
	}
}
