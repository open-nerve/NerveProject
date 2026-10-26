package app_test

import (
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/content"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func wantCollaborationEvent(t *testing.T, p *recordedEvents, eventType, kind string, id uuid.UUID, changes []shared.Change) {
	t.Helper()
	if len(p.rows) != 1 {
		t.Fatalf("published %d events, want one %s", len(p.rows), eventType)
	}
	e := p.rows[0]
	if e.ID == (uuid.UUID{}) || e.ID == id || e.Type != eventType || e.Entity != (shared.EntityRef{Kind: kind, ID: id}) || e.WorkspaceID != acme || e.ProjectID != web || e.ActorID == nil || *e.ActorID != alice || !e.OccurredAt.Equal(clockNow) {
		t.Errorf("event envelope=%+v", e)
	}
	if !reflect.DeepEqual(e.Changes, changes) {
		t.Errorf("changes=%+v want=%+v", e.Changes, changes)
	}
}

func TestCommentEvents(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete", "same HTML"} {
		for _, fail := range []bool{false, true} {
			t.Run(line(operation, fail), func(t *testing.T) {
				_, _, w, f := newCommentWrites()
				p := &recordedEvents{t: t}
				if fail {
					p.fail = errDisk
				}
				p.before = func(shared.Event) {
					if operation == "create" && f.inserted == nil || operation == "update" && f.updated == nil || operation == "delete" && f.deleted == nil {
						t.Fatal("published before business write")
					}
					if operation == "delete" && w.log.calls[len(w.log.calls)-1] != line("DeleteCommentReactionRows", f.current.ID) {
						t.Fatal("published before child reaction deletion")
					}
				}
				clock := clockAt{at: clockNow, log: w.log}
				id := f.current.ID
				html := "<p>After</p><script>bad()</script>"
				var err error
				changes := []shared.Change{{Field: "comment_html", New: evptr("<p>After</p>")}}
				switch operation {
				case "create":
					c := app.NewCreateIssueComment(app.CreateIssueCommentDeps{Events: p, Locks: w.locks, Comments: f, Members: w.project, Sanitizer: content.NewSanitizer(), Clock: clock})
					var got domain.Comment
					got, err = c.Execute(as(alice), f.current.IssueID, html)
					id = got.ID
					if fail && f.inserted != nil {
						id = f.inserted.Comment.ID
					}
				case "update", "same HTML":
					// The event must use the lock-time HTML, not the earlier precheck value.
					w.project.waited = func() { f.current.HTML = "<p>Locked</p>" }
					if operation == "same HTML" {
						html = "<p>Locked</p><script>bad()</script>"
					}
					u := app.NewUpdateIssueComment(app.UpdateIssueCommentDeps{Events: p, Locks: w.locks, Comments: f, Members: w.project, Sanitizer: content.NewSanitizer(), Clock: clock})
					_, err = u.Execute(as(alice), id, html)
					changes[0].Old = evptr("<p>Locked</p>")
				case "delete":
					f.current.ActorID = bob
					w.auth.grants = []shared.Grant{{ProjectAdmin: true}, {ProjectAdmin: true}}
					w.project.waited = func() { f.current.HTML = "<p>Locked</p>" }
					d := app.NewDeleteIssueComment(app.DeleteIssueCommentDeps{Events: p, Locks: w.locks, Comments: f, Clock: clock})
					err = d.Execute(as(alice), id)
					changes[0] = shared.Change{Field: "comment_html", Old: evptr("<p>Locked</p>")}
				}
				if operation == "same HTML" {
					if err != nil || len(p.rows) != 0 || eventClockReads(w) != 0 {
						t.Fatal("no-op published/read clock", err, p.rows)
					}
					return
				}
				if fail {
					if !errors.Is(err, errDisk) {
						t.Fatalf("publisher error lost: %v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
				wantCollaborationEvent(t, p, "issue.comment_"+map[string]string{"create": "created", "update": "updated", "delete": "deleted"}[operation], "issue_comment", id, changes)
				if eventClockReads(w) != 1 {
					t.Fatal("event did not reuse business clock")
				}
			})
		}
	}
}

func TestReactionEvents(t *testing.T) {
	for _, comment := range []bool{false, true} {
		for _, remove := range []bool{false, true} {
			for _, fail := range []bool{false, true} {
				t.Run(line("comment", comment, "remove", remove, "fail", fail), func(t *testing.T) {
					p := &recordedEvents{t: t}
					if fail {
						p.fail = errDisk
					}
					var w *world
					var id, container uuid.UUID
					var err error
					field, kind := "issue", "issue_reaction"
					if comment {
						_, _, world, f := newCommentReaction()
						w = world
						container = f.current.ID
						id = uuid.NewV7()
						field, kind = "comment", "comment_reaction"
						key := app.ReactionKey{ResourceID: container, ActorID: alice, Reaction: "10084-65039"}
						d := app.AddCommentReactionDeps{Events: p, Locks: w.locks, Comments: f, Reactions: f, Clock: clockAt{at: clockNow, log: w.log}}
						p.before = func(shared.Event) {
							if remove && f.removed == nil || !remove && f.added == nil {
								t.Fatal("event preceded reaction write")
							}
						}
						if remove {
							f.rows[key] = domain.CommentReaction{ID: id, WorkspaceID: acme, ProjectID: web, CommentID: container, ActorID: alice, Reaction: "10084-65039"}
							err = app.NewRemoveCommentReaction(d).Execute(as(alice), container, key.Reaction)
						} else {
							_, err = app.NewAddCommentReaction(d).Execute(as(alice), container, key.Reaction)
							if f.added != nil {
								id = f.added.Reaction.ID
							}
						}
						if f.marked != nil || f.updated != nil || w.issues.activities != nil {
							t.Fatal("comment reaction touched parent audit")
						}
					} else {
						_, _, world, f, target := newReaction()
						w = world
						container = target
						id = uuid.NewV7()
						key := app.ReactionKey{ResourceID: container, ActorID: alice, Reaction: "10084-65039"}
						d := app.AddIssueReactionDeps{Events: p, Locks: w.locks, Reactions: f, Clock: clockAt{at: clockNow, log: w.log}}
						p.before = func(shared.Event) {
							if remove && f.removed == nil || !remove && f.added == nil {
								t.Fatal("event preceded reaction write")
							}
						}
						if remove {
							f.rows[key] = domain.IssueReaction{ID: id, WorkspaceID: acme, ProjectID: web, IssueID: container, ActorID: alice, Reaction: key.Reaction}
							err = app.NewRemoveIssueReaction(d).Execute(as(alice), container, key.Reaction)
						} else {
							_, err = app.NewAddIssueReaction(d).Execute(as(alice), container, key.Reaction)
							if f.added != nil {
								id = f.added.Reaction.ID
							}
						}
					}
					if id == container || id == alice || container == alice {
						t.Fatal("fixture must distinguish row/container/actor")
					}
					if fail {
						if !errors.Is(err, errDisk) {
							t.Fatalf("publisher error lost: %v", err)
						}
					} else if err != nil {
						t.Fatal(err)
					}
					eventType := "issue.reaction_added"
					changes := []shared.Change{{Field: "reaction", New: evptr("10084-65039")}, {Field: field, NewID: &container}}
					if remove {
						eventType = "issue.reaction_removed"
						changes = []shared.Change{{Field: "reaction", Old: evptr("10084-65039")}, {Field: field, OldID: &container}}
					}
					wantCollaborationEvent(t, p, eventType, kind, id, changes)
					if eventClockReads(w) != 1 {
						t.Fatal("event did not reuse business clock")
					}
				})
			}
		}
	}
}

func TestCommentEventsRefused(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete"} {
		for _, refusal := range []string{"hidden", "project archived", "issue archived", "business SQL fails", "reaction read fails", "not author"} {
			if operation == "create" && refusal == "not author" || operation != "update" && refusal == "reaction read fails" {
				continue
			}
			t.Run(line(operation, refusal), func(t *testing.T) {
				_, _, w, f := newCommentWrites()
				p := &recordedEvents{t: t}
				switch refusal {
				case "hidden":
					w.auth.answers = []error{shared.ErrNotVisible}
				case "project archived":
					w.project.locked.Archived = true
				case "issue archived":
					w.issues.archivedAt = &earlier
				case "reaction read fails":
					w.log.fail = "CommentReactions"
				case "not author":
					f.current.ActorID = bob
				case "business SQL fails":
					w.log.fail = map[string]string{"create": "InsertComment", "update": "UpdateComment", "delete": "DeleteCommentReactionRows"}[operation]
				}
				clock := clockAt{at: clockNow, log: w.log}
				var err error
				switch operation {
				case "create":
					_, err = app.NewCreateIssueComment(app.CreateIssueCommentDeps{Events: p, Locks: w.locks, Comments: f, Members: w.project, Sanitizer: content.NewSanitizer(), Clock: clock}).Execute(as(alice), f.current.IssueID, "<p>After</p>")
				case "update":
					_, err = app.NewUpdateIssueComment(app.UpdateIssueCommentDeps{Events: p, Locks: w.locks, Comments: f, Members: w.project, Sanitizer: content.NewSanitizer(), Clock: clock}).Execute(as(alice), f.current.ID, "<p>After</p>")
				case "delete":
					err = app.NewDeleteIssueComment(app.DeleteIssueCommentDeps{Events: p, Locks: w.locks, Comments: f, Clock: clock}).Execute(as(alice), f.current.ID)
				}
				if err == nil || len(p.rows) != 0 {
					t.Fatal("refused write published", err, p.rows)
				}
			})
		}
	}
}

func TestReactionEventsRefused(t *testing.T) {
	for _, comment := range []bool{false, true} {
		for _, remove := range []bool{false, true} {
			for _, refusal := range []string{"hidden", "project archived", "issue archived", "duplicate or absent", "business SQL fails"} {
				t.Run(line("comment", comment, remove, refusal), func(t *testing.T) {
					p := &recordedEvents{t: t}
					var w *world
					var err error
					configure := func(world *world) {
						w = world
						switch refusal {
						case "hidden":
							w.auth.answers = []error{shared.ErrNotVisible}
						case "project archived":
							w.project.locked.Archived = true
						case "issue archived":
							w.issues.archivedAt = &earlier
						}
					}
					seed := remove != (refusal == "duplicate or absent")
					if comment {
						_, _, world, f := newCommentReaction()
						configure(world)
						key := app.ReactionKey{ResourceID: f.current.ID, ActorID: alice, Reaction: "128077"}
						if seed {
							f.rows[key] = domain.CommentReaction{ID: uuid.NewV7(), CommentID: f.current.ID, Reaction: key.Reaction}
						}
						if refusal == "business SQL fails" {
							w.log.fail = "InsertCommentReaction"
							if remove {
								w.log.fail = "DeleteCommentReaction"
							}
						}
						d := app.AddCommentReactionDeps{Events: p, Locks: w.locks, Comments: f, Reactions: f, Clock: clockAt{at: clockNow, log: w.log}}
						if remove {
							err = app.NewRemoveCommentReaction(d).Execute(as(alice), key.ResourceID, key.Reaction)
						} else {
							_, err = app.NewAddCommentReaction(d).Execute(as(alice), key.ResourceID, key.Reaction)
						}
					} else {
						_, _, world, f, id := newReaction()
						configure(world)
						key := app.ReactionKey{ResourceID: id, ActorID: alice, Reaction: "128077"}
						if seed {
							f.rows[key] = domain.IssueReaction{ID: uuid.NewV7(), IssueID: id, Reaction: key.Reaction}
						}
						if refusal == "business SQL fails" {
							w.log.fail = "InsertIssueReaction"
							if remove {
								w.log.fail = "DeleteIssueReaction"
							}
						}
						d := app.AddIssueReactionDeps{Events: p, Locks: w.locks, Reactions: f, Clock: clockAt{at: clockNow, log: w.log}}
						if remove {
							err = app.NewRemoveIssueReaction(d).Execute(as(alice), id, key.Reaction)
						} else {
							_, err = app.NewAddIssueReaction(d).Execute(as(alice), id, key.Reaction)
						}
					}
					if err == nil || len(p.rows) != 0 {
						t.Fatal("refused reaction published", err, p.rows)
					}
				})
			}
		}
	}
}
