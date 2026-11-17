package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// What the issue module tells the notification module about a work item (M7 P1b spec 2.6, R8), through
// issue.Provide's NotificationIssues, after the module's own writes through the API: its row FOR SHARE (IssueRowLock,
// Locks.share's lock, the fan-out's), as its row has it; its subscribers, the creation's automatic ones less one who
// unsubscribed; its own columns, its current assignees less one an update removed; a comment's plain text, none once the
// comment is deleted; and its guest rule, Issue.HiddenFrom: hidden from one who sees only his own unless he created it,
// a creator not known included, from no one else. Once the work item is deleted, it is neither locked nor read.
func TestIssueTellsTheNotificationModuleAboutAWorkItem(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	alice, bob, walt := w.ids["alice"], w.ids["bob"], w.ids["walt"]
	id := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Fix the login","state_id":%q,"assignee_ids":[%q,%q]}`, w.todo, walt, bob))
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+id.String(), fmt.Sprintf(`{"assignee_ids":[%q]}`, bob), http.StatusOK)
	w.send(t, "walt", http.MethodPost, subscriptionOf(id, "unsubscribe"), "", http.StatusNoContent)
	comment := w.newComment(t, "bob", id)
	reads := issue.Provide(w.pool, nil).NotificationIssues
	for _, q := range []struct {
		creator       *uuid.UUID
		account       uuid.UUID
		ownOnly, want bool
	}{{&alice, walt, true, true}, {&alice, alice, true, false}, {nil, alice, true, true}, {&alice, walt, false, false}, {nil, walt, false, false}} {
		if got := reads.HiddenFrom(q.creator, q.account, q.ownOnly); got != q.want {
			t.Errorf("HiddenFrom(%v, %s, %v) = %v; want %v", q.creator, q.account, q.ownOnly, got, q.want)
		}
	}
	tx := postgres.NewTxManager(w.pool, 2*time.Second)
	share := func(want issue.SharedIssue, wantFound bool) {
		t.Helper()
		if err := tx.WithinTx(t.Context(), func(ctx context.Context) error {
			got, found, err := reads.ShareIssue(ctx, id)
			if err == nil && (found != wantFound || !reflect.DeepEqual(got, want)) {
				t.Errorf("ShareIssue() = %+v, %v; want %+v, %v", got, found, want, wantFound)
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	share(issue.SharedIssue{ID: id, WorkspaceID: w.acme, ProjectID: w.web, CreatedByID: &alice}, true)
	if got, err := reads.SubscriberIDs(t.Context(), id); err != nil || !slices.Equal(got, []uuid.UUID{alice, bob}) {
		t.Errorf("SubscriberIDs() = %v, %v; want alice, bob", got, err)
	}
	got, found, err := reads.NotificationIssue(t.Context(), id)
	if want := (issue.NotificationIssue{Name: "Fix the login", SequenceID: 1, StateID: w.todo, CreatedByID: &alice, AssigneeIDs: []uuid.UUID{bob}}); err != nil ||
		!found || !reflect.DeepEqual(got, want) {
		t.Errorf("NotificationIssue() = %+v, %v, %v; want %+v", got, found, err, want)
	}
	if text, found, err := reads.CommentText(t.Context(), comment.ID); err != nil || !found || text != "Before" {
		t.Errorf("CommentText() = %q, %v, %v; want Before", text, found, err)
	}
	w.send(t, "bob", http.MethodDelete, commentPath(comment.ID), "", http.StatusNoContent)
	if text, found, err := reads.CommentText(t.Context(), comment.ID); err != nil || found || text != "" {
		t.Errorf("CommentText(deleted) = %q, %v, %v; want none", text, found, err)
	}
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+id.String(), "", http.StatusNoContent)
	share(issue.SharedIssue{}, false)
	if got, found, err := reads.NotificationIssue(t.Context(), id); err != nil || found || !reflect.DeepEqual(got, issue.NotificationIssue{}) {
		t.Errorf("NotificationIssue(deleted) = %+v, %v, %v; want none", got, found, err)
	}
}
