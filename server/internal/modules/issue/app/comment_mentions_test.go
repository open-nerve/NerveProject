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

// mentionReads are the MentionMembers calls of w's log.
func mentionReads(w *world) []string {
	var calls []string
	for _, c := range w.log.calls {
		if strings.HasPrefix(c, "MentionMembers") {
			calls = append(calls, c)
		}
	}
	return calls
}

// A comment's creation event carries every member its body mentions, gus,
// a guest who sees only his own, among them, out, no member, not, in id
// order, between its HTML and its work item (M7 P1a spec 2.2). It
// subscribes, as M4's, the caller and the mentioned who can see the item.
func TestCommentCreatedEventMentions(t *testing.T) {
	_, w, f, id := newComment()
	w.project.mentionVisibility = []app.MemberVisibility{{ID: gus, Role: shared.RoleGuest, OwnOnly: true}, {ID: bob, Role: shared.RoleMember}}
	p := &recordedEvents{t: t}
	uc := app.NewCreateIssueComment(app.CreateIssueCommentDeps{Events: p, Locks: w.locks, Comments: f, Members: w.project, Sanitizer: content.NewSanitizer(), Clock: clockAt{at: clockNow, log: w.log}})
	html := "<p>Hi</p>" + mentioning(out, gus, bob, bob)
	got, err := uc.Execute(as(alice), id, domain.NewComment{HTML: html})
	if err != nil {
		t.Fatal(err)
	}
	clean := content.NewSanitizer().Clean(html).HTML
	want := slices.Concat([]shared.Change{{Field: "comment_html", New: &clean}}, mentioned(bob, gus), []shared.Change{{Field: "issue", NewID: &id}})
	wantCollaborationEvent(t, p, "issue.comment_created", "issue_comment", got.ID, want)
	if f.subscribed == nil || !slices.Equal(f.subscribed.Targets, []uuid.UUID{bob, alice}) {
		t.Fatalf("subscriptions=%+v", f.subscribed)
	}
}

// An edit's event carries only the members its body newly mentions: those
// the body it replaces, read again under the project's lock and cleaned
// again, does not; out, no member, is not among them (spec 2.2). It
// subscribes the caller and, of those it newly mentions, the ones who can
// see the work item (R7): bob, still mentioned, who may have unsubscribed,
// is not subscribed again; gus, hidden, is not either.
func TestCommentUpdatedEventMentions(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		precheck, locked, next      []uuid.UUID
		asked, want, subscribedOnes []uuid.UUID
	}{
		{"adds one", []uuid.UUID{bob}, []uuid.UUID{bob}, []uuid.UUID{carol, bob}, []uuid.UUID{carol}, []uuid.UUID{carol}, []uuid.UUID{alice, carol}},
		{"removes one", []uuid.UUID{bob, carol}, []uuid.UUID{bob, carol}, []uuid.UUID{bob}, nil, nil, []uuid.UUID{alice}},
		{"lock-time body", []uuid.UUID{bob}, []uuid.UUID{bob, carol}, []uuid.UUID{bob, dave, carol}, []uuid.UUID{dave}, []uuid.UUID{dave}, []uuid.UUID{alice, dave}},
		{"no member, hidden guest", nil, nil, []uuid.UUID{out, gus, bob}, []uuid.UUID{bob, gus, out}, []uuid.UUID{bob, gus}, []uuid.UUID{bob, alice}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, w, f := newCommentWrites()
			w.project.mentionVisibility = []app.MemberVisibility{{ID: dave, Role: shared.RoleMember}, {ID: gus, Role: shared.RoleGuest, OwnOnly: true}, {ID: carol, Role: shared.RoleAdmin}, {ID: bob, Role: shared.RoleMember}}
			f.current.HTML = "<p>Before</p>" + mentioning(tc.precheck...)
			locked := "<p>Locked</p>" + mentioning(tc.locked...)
			w.project.waited = func() { f.current.HTML = locked }
			p := &recordedEvents{t: t}
			u := app.NewUpdateIssueComment(app.UpdateIssueCommentDeps{Events: p, Locks: w.locks, Comments: f, Members: w.project, Sanitizer: content.NewSanitizer(), Clock: clockAt{at: clockNow, log: w.log}})
			html := "<p>After</p>" + mentioning(tc.next...)
			if _, err := u.Execute(as(alice), f.current.ID, html); err != nil {
				t.Fatal(err)
			}
			clean := content.NewSanitizer().Clean(html).HTML
			want := slices.Concat([]shared.Change{{Field: "comment_html", Old: &locked, New: &clean}}, mentioned(tc.want...), []shared.Change{{Field: "issue", NewID: &f.current.IssueID}})
			wantCollaborationEvent(t, p, "issue.comment_updated", "issue_comment", f.current.ID, want)
			var reads []string
			if tc.asked != nil {
				reads = []string{line("MentionMembers", web, tc.asked)}
			}
			if got := mentionReads(w); !reflect.DeepEqual(got, reads) {
				t.Fatalf("member reads=%q want %q", got, reads)
			}
			if f.subscribed == nil || !slices.Equal(f.subscribed.Targets, tc.subscribedOnes) {
				t.Fatalf("subscriptions=%+v want %v", f.subscribed, tc.subscribedOnes)
			}
		})
	}
}
