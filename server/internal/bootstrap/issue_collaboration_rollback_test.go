package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/content"
	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// collaborationFaultStore runs the real statement before failing that port, so
// each assertion observes rollback of actual SQL, including the failing write.
type collaborationFaultStore struct {
	*issuepg.Store
	port    string
	failure error
}

func (s collaborationFaultStore) after(port string, err error) error {
	if err != nil {
		return err
	}
	if s.port == port {
		return s.failure
	}
	return nil
}
func (s collaborationFaultStore) MarkChanged(ctx context.Context, row issueapp.ChangedRows) error {
	return s.after("MarkChanged", s.Store.MarkChanged(ctx, row))
}
func (s collaborationFaultStore) InsertSubscribers(ctx context.Context, row issueapp.LinkRows) error {
	return s.after("InsertSubscribers", s.Store.InsertSubscribers(ctx, row))
}
func (s collaborationFaultStore) InsertComment(ctx context.Context, row issueapp.CommentWrite) error {
	return s.after("InsertComment", s.Store.InsertComment(ctx, row))
}
func (s collaborationFaultStore) UpdateComment(ctx context.Context, row issueapp.CommentWrite) error {
	return s.after("UpdateComment", s.Store.UpdateComment(ctx, row))
}
func (s collaborationFaultStore) DeleteComment(ctx context.Context, row issueapp.CommentDeletion) error {
	return s.after("DeleteComment", s.Store.DeleteComment(ctx, row))
}
func (s collaborationFaultStore) DeleteCommentReactionRows(ctx context.Context, row issueapp.CommentDeletion) error {
	return s.after("DeleteCommentReactionRows", s.Store.DeleteCommentReactionRows(ctx, row))
}
func (s collaborationFaultStore) InsertIssueReaction(ctx context.Context, row issueapp.IssueReactionWrite) error {
	return s.after("InsertIssueReaction", s.Store.InsertIssueReaction(ctx, row))
}
func (s collaborationFaultStore) DeleteIssueReaction(ctx context.Context, row issueapp.ReactionDeletion) error {
	return s.after("DeleteIssueReaction", s.Store.DeleteIssueReaction(ctx, row))
}
func (s collaborationFaultStore) InsertCommentReaction(ctx context.Context, row issueapp.CommentReactionWrite) error {
	return s.after("InsertCommentReaction", s.Store.InsertCommentReaction(ctx, row))
}
func (s collaborationFaultStore) DeleteCommentReaction(ctx context.Context, row issueapp.ReactionDeletion) error {
	return s.after("DeleteCommentReaction", s.Store.DeleteCommentReaction(ctx, row))
}
func (s collaborationFaultStore) CommentReactions(ctx context.Context, ids []uuid.UUID) ([]domain.CommentReaction, error) {
	rows, err := s.Store.CommentReactions(ctx, ids)
	return rows, s.after("CommentReactions", err)
}

func TestCollaborationRollsBackPortFailures(t *testing.T) {
	for _, operation := range []struct {
		name  string
		ports []string
	}{
		{"createIssueComment", []string{"MarkChanged", "InsertSubscribers", "InsertComment"}},
		{"updateIssueComment", []string{"MarkChanged", "InsertSubscribers", "UpdateComment", "CommentReactions"}},
		{"deleteIssueComment", []string{"MarkChanged", "DeleteComment", "DeleteCommentReactionRows"}},
		{"addIssueReaction", []string{"MarkChanged", "InsertIssueReaction"}},
		{"removeIssueReaction", []string{"MarkChanged", "DeleteIssueReaction"}},
		{"addCommentReaction", []string{"InsertCommentReaction"}},
		{"removeCommentReaction", []string{"DeleteCommentReaction"}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			w := newIssueWorld(t)
			w.makeMember(t, "walt", w.web, 15)
			id := w.newWorkItemIn(t, w.web, `{"name":"Atomic collaboration"}`)
			comment := w.newComment(t, "bob", id)
			if operation.name == "removeIssueReaction" {
				w.send(t, "bob", http.MethodPost, reactionsOf(id), `{"reaction":"128077"}`, 201)
			}
			if operation.name == "removeCommentReaction" || operation.name == "deleteIssueComment" {
				addCommentReaction(t, w, "bob", comment.ID, "128077")
			}
			store := issuepg.New(w.pool)
			projects := issueProjects{projects: project.Provide(w.pool).IssueProjects}
			locks := issueapp.NewLocks(projects, store, authorizerOn(w.pool), newWriteGate(testIssueConfig.WriteGate), postgres.NewTxManager(w.pool, 2*time.Second))
			ctx := shared.WithActor(pgtest.Soon(t), shared.Actor{UserID: w.ids["bob"]})
			for _, port := range operation.ports {
				t.Run(port, func(t *testing.T) {
					sentinel := errors.New("failed after real SQL: " + port)
					wrapped := collaborationFaultStore{Store: store, port: port, failure: sentinel}
					before := collaborationState(t, w)
					var err error
					switch operation.name {
					case "createIssueComment":
						_, err = issueapp.NewCreateIssueComment(issueapp.CreateIssueCommentDeps{Events: discardIssueEvents{}, Locks: locks, Comments: wrapped, Members: projects, Sanitizer: content.NewSanitizer(), Clock: clock.System{}}).Execute(ctx, id, mentionHTML(w.ids["walt"]))
					case "updateIssueComment":
						_, err = issueapp.NewUpdateIssueComment(issueapp.UpdateIssueCommentDeps{Events: discardIssueEvents{}, Locks: locks, Comments: wrapped, Members: projects, Sanitizer: content.NewSanitizer(), Clock: clock.System{}}).Execute(ctx, comment.ID, mentionHTML(w.ids["walt"]))
					case "deleteIssueComment":
						err = issueapp.NewDeleteIssueComment(issueapp.DeleteIssueCommentDeps{Events: discardIssueEvents{}, Locks: locks, Comments: wrapped, Clock: clock.System{}}).Execute(ctx, comment.ID)
					case "addIssueReaction":
						_, err = issueapp.NewAddIssueReaction(issueapp.AddIssueReactionDeps{Events: discardIssueEvents{}, Locks: locks, Reactions: wrapped, Clock: clock.System{}}).Execute(ctx, id, "128077")
					case "removeIssueReaction":
						err = issueapp.NewRemoveIssueReaction(issueapp.RemoveIssueReactionDeps{Events: discardIssueEvents{}, Locks: locks, Reactions: wrapped, Clock: clock.System{}}).Execute(ctx, id, "128077")
					case "addCommentReaction":
						_, err = issueapp.NewAddCommentReaction(issueapp.AddCommentReactionDeps{Events: discardIssueEvents{}, Locks: locks, Comments: store, Reactions: wrapped, Clock: clock.System{}}).Execute(ctx, comment.ID, "128077")
					case "removeCommentReaction":
						err = issueapp.NewRemoveCommentReaction(issueapp.RemoveCommentReactionDeps{Events: discardIssueEvents{}, Locks: locks, Comments: store, Reactions: wrapped, Clock: clock.System{}}).Execute(ctx, comment.ID, "128077")
					default:
						t.Fatal("unknown operation", operation.name)
					}
					if err != sentinel {
						t.Fatal("port error identity changed", err, sentinel)
					}
					if !reflect.DeepEqual(before, collaborationState(t, w)) {
						t.Fatal("failed port committed collaboration state", operation.name, port)
					}
				})
			}
		})
	}
}
