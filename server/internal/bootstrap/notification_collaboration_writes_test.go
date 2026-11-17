package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"uuid"

	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	notificationdomain "github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fieldAt is the position of the job's one change of field.
func fieldAt(t *testing.T, job notificationriver.FanoutArgs, field string) int16 {
	t.Helper()
	at := -1
	for n, c := range job.Changes {
		if c.Field == field {
			if at >= 0 {
				t.Fatalf("%s has two %s changes: %+v", job.Type, field, job.Changes)
			}
			at = n
		}
	}
	if at < 0 {
		t.Fatalf("%s has no %s change: %+v", job.Type, field, job.Changes)
	}
	return int16(at)
}

// draftPublished is the work item alice publishes from a draft in Web's Todo, named Published and assigned to bob,
// with two files attached, files, and a description showing an image; a third file was attached and deleted before.
func (w issueWorld) draftPublished(t *testing.T) (storedIssue, []preparedAssetAnswer) {
	t.Helper()
	payload := map[string]any{"name": "Published", "state_id": w.todo.String(), "assignee_ids": []string{w.ids["bob"].String()}}
	body, err := json.Marshal(map[string]any{"project_id": w.web.String(), "payload": payload})
	if err != nil {
		t.Fatal(err)
	}
	draft := createDraftHTTP(t, w, "alice", string(body))
	files := []preparedAssetAnswer{draftAsset(t, w, draft.ID, false), draftAsset(t, w, draft.ID, false)}
	image, dropped := draftAsset(t, w, draft.ID, true), draftAsset(t, w, draft.ID, false)
	assetAPICall(t, w, "alice", http.MethodDelete, "/api/v0/assets/"+dropped.Asset.ID.String(), "", http.StatusNoContent)
	payload["description_html"] = issueImageHTML(image.Asset.ID)
	draftRequest(t, w, "alice", http.MethodPatch, draftPath(draft.ID), draftPayloadBody(t, payload), http.StatusOK)
	var published storedIssue
	decodeAnswer(t, draftRequest(t, w, "alice", http.MethodPost, publishPath(draft.ID), "", http.StatusCreated), &published)
	return published, files
}

// The collaboration writes through the API send what M7 P1b spec 2.3–2.5 say, read back from the table once the app's
// fan-outs have run; each notification of a change at its position in its event, in the work item's workspace and
// project, at the event's moment, by its actor, of the work item as it is then. alice's work items have no subscriber
// but her until the creations' fan-outs have run. bob's comments on one send alice, its creator, each comment, by its
// id: its creation and its change with the first 200 characters of its stored text, not bytes (a 240-character text of
// two- and three-byte characters is cut at 200), and its deletion with none. alice's link on one bob subscribed to sends
// him its address created, changed, then deleted; her change of its title alone enqueues no fan-out and sends nothing.
// Her relation of a work item bob subscribed to blocking one walt subscribed to sends bob, on the end it was requested
// on, blocking, created, with the other by its id, not walt; its removal through the relation's id sends walt, on the
// end stored as its issue, blocked_by, deleted, with the other by its id, not bob (R11). Her attachment of a file to
// one bob subscribed to sends him the attachment, created, then deleted. The publication of her draft assigned to bob
// with two files sends bob its creation and his assignment, as its creation does, and each file attached as its own
// attachment, as its events say (R5); not the image its description shows, nor the file deleted from the draft.
func TestCollaborationWritesNotifyThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob, walt := w.ids["alice"], w.ids["bob"], w.ids["walt"]
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	todo := webState{w.todo, "Todo", "unstarted"}
	talk, linked, attached := w.itemMade(t, "alice", "Talk", todo), w.itemMade(t, "alice", "Links", todo), w.itemMade(t, "alice", "Files", todo)
	blocker, blocked := w.itemMade(t, "alice", "Blocker", todo), w.itemMade(t, "alice", "Blocked", todo)
	// A fan-out reads the work item's subscribers when it runs: the creations' have run before anyone else subscribes.
	waitForFanouts(t, w.pool)
	for _, s := range []struct {
		watcher string
		item    uuid.UUID
	}{{"bob", linked.ID}, {"bob", attached.ID}, {"bob", blocker.ID}, {"walt", blocked.ID}} {
		w.send(t, s.watcher, http.MethodPost, "/api/v0/issues/"+s.item.String()+"/subscribe", "", http.StatusNoContent)
	}
	long := strings.Repeat("评论éa", 60)
	comments := make([]answeredComment, 3)
	for n, html := range []string{"<p>" + long + "</p>", "<p>Looks good</p>", "<p>Remove me</p>"} {
		decodeAnswer(t, w.send(t, "bob", http.MethodPost, commentsOf(talk.ID), fmt.Sprintf(`{"comment_html":%q}`, html), http.StatusCreated), &comments[n])
	}
	first, second := "http://169.254.169.254/notes", "http://169.254.169.254/notes-2"
	link := createLink(t, w, "alice", linked.ID, first)
	relation := listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(blocker.ID), relating("blocking", blocked.ID),
		http.StatusCreated))[0]
	file := uploadedAssetHTTP(t, w, "alice", "ISSUE_ATTACHMENT", attachmentScope("issue", attached.ID), "notes.txt", "text/plain", []byte("abc"))
	published, files := w.draftPublished(t)
	// A comment's fan-out reads its text when it runs: the comments' creations' have run before they change.
	waitForFanouts(t, w.pool)

	w.send(t, "bob", http.MethodPatch, commentPath(comments[1].ID), `{"comment_html":"<p>Looks good now</p>"}`, http.StatusOK)
	w.send(t, "bob", http.MethodDelete, commentPath(comments[2].ID), "", http.StatusNoContent)
	w.send(t, "alice", http.MethodPatch, linkPath(link.ID), fmt.Sprintf(`{"url":%q}`, second), http.StatusOK)
	w.send(t, "alice", http.MethodPatch, linkPath(link.ID), `{"title":"Notes"}`, http.StatusOK)
	w.send(t, "alice", http.MethodDelete, linkPath(link.ID), "", http.StatusNoContent)
	w.send(t, "alice", http.MethodDelete, relationPath(relation.ID), "", http.StatusNoContent)
	assetAPICall(t, w, "alice", http.MethodDelete, "/api/v0/assets/"+file.Asset.ID.String(), "", http.StatusNoContent)
	waitForFanouts(t, w.pool)

	v := eventValue[string]
	t.Run("comments", func(t *testing.T) {
		var events []uuid.UUID
		var want []sentNotification
		comment := func(job notificationriver.FanoutArgs, verb string, id uuid.UUID, excerpt *string) {
			events = append(events, job.EventID)
			want = append(want, w.sentOf(job, &bob, &alice, talk,
				sending{alice, "created", 0, notificationdomain.ChangeSnapshot{Field: "comment", Verb: notificationdomain.Verb(verb), CommentID: &id,
					Excerpt: excerpt}})...)
		}
		for n, excerpt := range []string{strings.Repeat("评论éa", 50), "Looks good", "Remove me"} {
			comment(fanoutOf(t, w.pool, "issue.comment_created", comments[n].ID), "created", comments[n].ID, &excerpt)
		}
		comment(fanoutOf(t, w.pool, "issue.comment_updated", comments[1].ID), "updated", comments[1].ID, v("Looks good now"))
		comment(fanoutOf(t, w.pool, "issue.comment_deleted", comments[2].ID), "deleted", comments[2].ID, nil)
		assertSent(t, w.pool, talk.ID, events, want)
	})
	t.Run("links", func(t *testing.T) {
		made, moved, gone := fanoutOf(t, w.pool, "issue.link_created", link.ID), fanoutOf(t, w.pool, "issue.link_updated", link.ID),
			fanoutOf(t, w.pool, "issue.link_deleted", link.ID)
		address := func(verb string, before, after *string) sending {
			return sending{bob, "subscribed", 0, notificationdomain.ChangeSnapshot{Field: "link", Verb: notificationdomain.Verb(verb), OldValue: before,
				NewValue: after}}
		}
		assertSent(t, w.pool, linked.ID, []uuid.UUID{made.EventID, moved.EventID, gone.EventID}, slices.Concat(
			w.sentOf(made, &alice, &alice, linked, address("created", nil, &first)),
			w.sentOf(moved, &alice, &alice, linked, address("updated", &first, &second)),
			w.sentOf(gone, &alice, &alice, linked, address("deleted", &second, nil))))
	})
	t.Run("relations", func(t *testing.T) {
		added, removed := fanoutOf(t, w.pool, "issue.relation_added", relation.ID), fanoutOf(t, w.pool, "issue.relation_removed", relation.ID)
		assertSent(t, w.pool, blocker.ID, []uuid.UUID{added.EventID}, w.sentOf(added, &alice, &alice, blocker,
			sending{bob, "subscribed", fieldAt(t, added, "requested_on"), notificationdomain.ChangeSnapshot{Field: "blocking", Verb: "created",
				NewID: &blocked.ID}}))
		assertSent(t, w.pool, blocked.ID, []uuid.UUID{removed.EventID}, w.sentOf(removed, &alice, &alice, blocked,
			sending{walt, "subscribed", fieldAt(t, removed, "requested_on"), notificationdomain.ChangeSnapshot{Field: "blocked_by", Verb: "deleted",
				OldID: &blocker.ID}}))
	})
	t.Run("attachments", func(t *testing.T) {
		added, deleted := fanoutOf(t, w.pool, "asset.attachment_added", file.Asset.ID), fanoutOf(t, w.pool, "asset.attachment_deleted", file.Asset.ID)
		assertSent(t, w.pool, attached.ID, []uuid.UUID{added.EventID, deleted.EventID}, slices.Concat(
			w.sentOf(added, &alice, &alice, attached, sending{bob, "subscribed", 0, notificationdomain.ChangeSnapshot{Field: "attachment", Verb: "created"}}),
			w.sentOf(deleted, &alice, &alice, attached, sending{bob, "subscribed", 0, notificationdomain.ChangeSnapshot{Field: "attachment", Verb: "deleted"}})))
	})
	t.Run("a draft's publication with files", func(t *testing.T) {
		item := notificationdomain.IssueSnapshot{ID: published.ID, Name: "Published", SequenceID: published.SequenceID, Identifier: "WEB",
			StateName: todo.name, StateGroup: todo.group}
		created, bobName := fanoutOf(t, w.pool, "issue.created", published.ID), displayNames(t, w, bob)[0]
		events := []uuid.UUID{created.EventID}
		want := w.sentOf(created, &alice, &alice, item, sending{bob, "assigned", -1, madeIt},
			sending{bob, "assigned", assignedAt(t, created, bob), assigning(&bobName, &bob)})
		for _, f := range files {
			job := fanoutOf(t, w.pool, "asset.attachment_added", f.Asset.ID)
			events = append(events, job.EventID)
			want = append(want, w.sentOf(job, &alice, &alice, item, sending{bob, "assigned", 0,
				notificationdomain.ChangeSnapshot{Field: "attachment", Verb: "created"}})...)
		}
		assertSent(t, w.pool, published.ID, events, want)
	})
}
