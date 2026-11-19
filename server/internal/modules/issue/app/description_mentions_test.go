package app_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/content"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// mentioning is ids mentioned, in their order, as the editor writes them.
func mentioning(ids ...uuid.UUID) string {
	html := ""
	for _, id := range ids {
		html += `<mention-component entity_name="user_mention" entity_identifier="` + id.String() + `"></mention-component>`
	}
	return html
}

// mentioned are a write's changes for the accounts ids, one each.
func mentioned(ids ...uuid.UUID) []shared.Change {
	var changes []shared.Change
	for _, id := range ids {
		changes = append(changes, shared.Change{Field: "mention", NewID: &id})
	}
	return changes
}

// mentionCreate and mentionUpdate are eventCreate and eventUpdate with the
// real sanitizer, which reads the mentions out of a description.
func mentionCreate(w *world, p shared.EventPublisher) *app.CreateIssue {
	return app.NewCreateIssue(app.CreateIssueDeps{Locks: w.locks, Creation: app.NewIssueCreation(app.IssueCreationDeps{Attachments: noIssueAttachments{}, Events: p, Issues: w.issues, Sequences: w.project, States: w.project, Labels: w.project, Members: w.project, Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}, Sanitizer: content.NewSanitizer()})})
}
func mentionUpdate(w *world, p shared.EventPublisher) *app.UpdateIssue {
	return app.NewUpdateIssue(app.UpdateIssueDeps{Locks: w.locks, Clock: clockAt{at: clockNow, log: w.log}, Update: app.NewIssueUpdate(app.IssueUpdateDeps{Events: p, Locks: w.locks, Issues: w.issues, States: w.project, Labels: w.project, Members: w.project, Profiles: w.profiles, Sanitizer: content.NewSanitizer()})})
}

// subscribed are the InsertSubscribers calls of w's log, each with the
// accounts it subscribes.
func subscribed(w *world) []string {
	var calls []string
	for _, c := range w.log.calls {
		if strings.HasPrefix(c, "InsertSubscribers") {
			calls = append(calls, c)
		}
	}
	return calls
}

// A creation's event ends with every member its description mentions, a
// guest who sees only his own among them, in id order; one who is no member
// (out) is not among them (M7 P1a spec 2.2). Its subscriptions are M4's: the
// caller and the mentioned who can see the work item.
func TestIssueCreatedEventMentions(t *testing.T) {
	_, w := newCreate()
	w.project.mentionVisibility = []app.MemberVisibility{{ID: gus, Role: shared.RoleGuest, OwnOnly: true}, {ID: carol, Role: shared.RoleAdmin}, {ID: bob, Role: shared.RoleMember}}
	p := &recordedEvents{t: t}
	html := "<p>Fix</p>" + mentioning(out, gus, carol, bob, bob)
	got, err := mentionCreate(w, p).Execute(as(alice), web, domain.NewIssue{Name: "Fix", DescriptionHTML: &html})
	if err != nil {
		t.Fatal(err)
	}
	clean := content.NewSanitizer().Clean(html).HTML
	want := append([]shared.Change{
		{Field: "name", New: evptr("Fix")}, {Field: "description", New: &clean},
		{Field: "state", New: evptr("Backlog"), NewID: &backlog.ID}, {Field: "priority", New: evptr("none")},
		{Field: "start_date"}, {Field: "target_date"}, {Field: "parent"}, {Field: "sort_order", New: evptr("65535")},
		{Field: "sequence_id", New: evptr("7")}, {Field: "completed_at"},
	}, mentioned(bob, carol, gus)...)
	wantIssueEvent(t, p, "issue.created", got.ID, want)
	if calls := subscribed(w); !slices.Equal(calls, []string{line("InsertSubscribers", []uuid.UUID{bob, alice, carol})}) {
		t.Fatalf("subscriptions=%q", calls)
	}
}

// An update's event ends, after sort_order, with the members its description
// newly mentions, those it has no mention row of, a guest who sees only his
// own among them, in id order; bob, mentioned before, is not. Only those of
// them who can see the work item subscribe (R7): dave, not bob again.
func TestIssueUpdatedEventMentions(t *testing.T) {
	_, w, id := newUpdate()
	w.issues.issue.DescriptionHTML = "<p>Before</p>" + mentioning(bob)
	w.issues.mentions = []uuid.UUID{bob}
	w.project.mentionVisibility = []app.MemberVisibility{{ID: dave, Role: shared.RoleMember}, {ID: gus, Role: shared.RoleGuest, OwnOnly: true}, {ID: bob, Role: shared.RoleMember}}
	p := &recordedEvents{t: t}
	old, html := w.issues.issue.DescriptionHTML, "<p>After</p>"+mentioning(out, dave, gus, bob)
	if _, err := mentionUpdate(w, p).Execute(as(alice), id, domain.IssuePatch{Name: evptr("Mend"), DescriptionHTML: &html, SortOrder: evptr(0.0)}); err != nil {
		t.Fatal(err)
	}
	clean := content.NewSanitizer().Clean(html).HTML
	wantIssueEvent(t, p, "issue.updated", id, append([]shared.Change{
		{Field: "name", Old: evptr("Fix"), New: evptr("Mend")}, {Field: "description", Old: &old, New: &clean},
		{Field: "sort_order", Old: evptr("65535"), New: evptr("0")},
	}, mentioned(gus, dave)...))
	if calls := subscribed(w); !slices.Equal(calls, []string{line("InsertSubscribers", []uuid.UUID{dave})}) {
		t.Fatalf("subscriptions=%q, want only the newly mentioned who sees the work item", calls)
	}
	if !slices.Equal(w.issues.mentions, []uuid.UUID{bob, gus, dave}) {
		t.Fatalf("mention rows=%v", w.issues.mentions)
	}
}

// A save that mentions only those mentioned before subscribes no one again
// (R7) and its event has no mention; neither has a patch without a
// description, though the stored one mentions someone with no mention row;
// a save written but unchanged (skip_activity) publishes nothing (spec 2.2).
func TestIssueUpdatedEventWithoutNewMentions(t *testing.T) {
	for _, tc := range []string{"same mentions", "no description", "written unchanged"} {
		t.Run(tc, func(t *testing.T) {
			_, w, id := newUpdate()
			w.issues.issue.DescriptionHTML = "<p>Old</p>" + mentioning(bob)
			w.issues.mentions = []uuid.UUID{bob}
			w.project.mentionVisibility = []app.MemberVisibility{{ID: bob, Role: shared.RoleMember}}
			p := &recordedEvents{t: t}
			old := w.issues.issue.DescriptionHTML
			patch := domain.IssuePatch{DescriptionHTML: evptr("<p>New</p>" + mentioning(bob, bob))}
			switch tc {
			case "no description":
				w.issues.mentions = nil
				patch = domain.IssuePatch{Name: evptr("Mend")}
			case "written unchanged":
				patch = domain.IssuePatch{DescriptionHTML: evptr(`<p data-id="block">Old</p>` + mentioning(bob)), SkipActivity: evptr(true)}
			}
			if _, err := mentionUpdate(w, p).Execute(as(alice), id, patch); err != nil {
				t.Fatal(err)
			}
			var want []string
			switch tc {
			case "same mentions":
				clean := content.NewSanitizer().Clean(*patch.DescriptionHTML).HTML
				wantIssueEvent(t, p, "issue.updated", id, []shared.Change{{Field: "description", Old: &old, New: &clean}})
			case "no description":
				wantIssueEvent(t, p, "issue.updated", id, []shared.Change{{Field: "name", Old: evptr("Fix"), New: evptr("Mend")}})
			case "written unchanged":
				if len(p.rows) != 0 || w.issues.descriptionWrites != 1 {
					t.Fatalf("events=%+v description writes=%d", p.rows, w.issues.descriptionWrites)
				}
			}
			if calls := subscribed(w); !reflect.DeepEqual(calls, want) {
				t.Fatalf("subscriptions=%q want %q", calls, want)
			}
		})
	}
}
