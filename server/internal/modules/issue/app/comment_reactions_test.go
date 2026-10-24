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

func TestCommentPageIncludesReactions(t *testing.T) {
	_, w, f, id := newComment()
	f.listed = []domain.Comment{{ID: bob}, {ID: alice}}
	f.reactions = []domain.CommentReaction{{ID: carol, CommentID: bob, ActorID: alice, Reaction: "128077"}, {ID: dave, CommentID: bob, ActorID: bob, Reaction: "128078"}}
	limit := 1
	u := app.NewListIssueComments(app.ListIssueCommentsDeps{Locks: w.locks, Comments: f, ReadTx: fakeTx{log: w.log}})
	page, err := u.Execute(as(alice), id, &limit, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 1 || !reflect.DeepEqual(page.Data[0].Reactions, f.reactions) {
		t.Fatal("embedded reactions", page)
	}
	if !reflect.DeepEqual(f.reactionIDs, []uuid.UUID{bob}) {
		t.Fatal("must batch only page ids", f.reactionIDs)
	}
}
func TestCommentPatchIncludesReactions(t *testing.T) {
	for _, html := range []string{"<p>Before</p>", "<p>After</p>"} {
		t.Run(html, func(t *testing.T) {
			u, _, _, f := newCommentWrites()
			f.reactions = []domain.CommentReaction{{ID: bob, CommentID: f.current.ID, ActorID: alice, Reaction: "128077"}, {ID: carol, CommentID: f.current.ID, ActorID: bob, Reaction: "128078"}}
			got, err := u.Execute(as(alice), f.current.ID, html)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Reactions, f.reactions) {
				t.Fatal("PATCH lost live reactions", got.Reactions)
			}
			if html == "<p>Before</p>" && (f.updated != nil || f.marked != nil) {
				t.Fatal("no-op wrote audit")
			}
		})
	}
}

type commentReactionRows struct {
	*writableComments
	rows    map[app.ReactionKey]domain.CommentReaction
	added   *app.CommentReactionWrite
	removed *app.ReactionDeletion
}

func (f *commentReactionRows) CommentReaction(ctx context.Context, k app.ReactionKey) (domain.CommentReaction, bool, error) {
	err := f.log.add(ctx, "CommentReaction", k)
	r, ok := f.rows[k]
	return r, ok, err
}
func (f *commentReactionRows) InsertCommentReaction(ctx context.Context, w app.CommentReactionWrite) error {
	if err := f.log.add(ctx, "InsertCommentReaction", w.Reaction.CommentID); err != nil {
		return err
	}
	f.added = &w
	f.rows[app.ReactionKey{ResourceID: w.Reaction.CommentID, ActorID: w.Reaction.ActorID, Reaction: w.Reaction.Reaction}] = w.Reaction
	return nil
}
func (f *commentReactionRows) DeleteCommentReaction(ctx context.Context, d app.ReactionDeletion) error {
	if err := f.log.add(ctx, "DeleteCommentReaction", d.Key); err != nil {
		return err
	}
	f.removed = &d
	delete(f.rows, d.Key)
	return nil
}
func newCommentReaction() (*app.AddCommentReaction, *app.RemoveCommentReaction, *world, *commentReactionRows) {
	_, _, w, c := newCommentWrites()
	c.current.ActorID = bob
	f := &commentReactionRows{writableComments: c, rows: map[app.ReactionKey]domain.CommentReaction{}}
	d := app.AddCommentReactionDeps{Locks: w.locks, Comments: f, Reactions: f, Clock: clockAt{at: clockNow, log: w.log}}
	return app.NewAddCommentReaction(d), app.NewRemoveCommentReaction(d), w, f
}
func TestCommentReactions(t *testing.T) {
	a, r, w, f := newCommentReaction()
	before := f.current
	got, err := a.Execute(as(alice), f.current.ID, "10084-65039")
	if err != nil {
		t.Fatal(err)
	}
	if got.ActorID != alice || got.CommentID != f.current.ID || got.ID == (uuid.UUID{}) || !got.CreatedAt.Equal(clockNow) {
		t.Fatal(got)
	}
	if f.added == nil || f.added.By != alice || !f.added.At.Equal(clockNow) {
		t.Fatal("reaction audit")
	}
	_, err = a.Execute(as(alice), f.current.ID, "10084-65039")
	wantRefused(t, err, domain.ErrReactionExists)
	wantRefused(t, r.Execute(as(bob), f.current.ID, "10084-65039"), domain.ErrReactionNotFound)
	if _, err = a.Execute(as(bob), f.current.ID, "10084-65039"); err != nil {
		t.Fatal(err)
	}
	if err = r.Execute(as(alice), f.current.ID, "10084-65039"); err != nil {
		t.Fatal(err)
	}
	if len(f.rows) != 1 || f.removed.Key.ActorID != alice || !f.removed.At.Equal(clockNow) {
		t.Fatal("caller scope")
	}
	if !reflect.DeepEqual(before, f.current) || f.marked != nil || f.updated != nil || w.issues.activities != nil || f.subscribed != nil {
		t.Fatal("reaction touched parent audit")
	}
	for _, remove := range []bool{false, true} {
		for _, kind := range []string{"hidden", "missing", "moved under lock", "deleted under lock", "invalid", "project archived", "issue archived"} {
			t.Run(line(kind, remove), func(t *testing.T) {
				a, r, w, f := newCommentReaction()
				value := "128077"
				want := error(domain.ErrCommentNotFound)
				switch kind {
				case "hidden":
					w.auth.grants = []shared.Grant{ownOnly}
					w.auth.answers = []error{shared.Forbidden()}
				case "missing":
					f.found = false
				case "moved under lock":
					w.project.waited = func() { f.current.IssueID = uuid.NewV7() }
				case "deleted under lock":
					w.project.waited = func() { f.found = false }
				case "invalid":
					value = "-1"
				case "project archived":
					w.project.locked.Archived = true
					want = domain.ErrProjectArchived
				case "issue archived":
					w.issues.archivedAt = &earlier
					want = domain.ErrIssueArchived
				}
				var err error
				if remove {
					err = r.Execute(as(alice), f.current.ID, value)
				} else {
					_, err = a.Execute(as(alice), f.current.ID, value)
				}
				if kind == "invalid" {
					if err == nil || len(w.log.calls) != 0 {
						t.Fatal(err, w.log.calls)
					}
				} else {
					wantRefused(t, err, want)
				}
				if kind == "hidden" || kind == "missing" {
					noCommentResources(t, w)
				}
				if f.added != nil || f.removed != nil || slices.Contains(w.log.calls, "Now") {
					t.Fatal("refusal wrote or read clock", w.log.calls)
				}
			})
		}
	}
	for _, port := range []string{"GetComment", "GetIssue", "Authorize", "Lock", "CommentReaction", "InsertCommentReaction", "DeleteCommentReaction"} {
		t.Run(port, func(t *testing.T) {
			a, r, w, f := newCommentReaction()
			w.log.fail = port
			var err error
			if port == "DeleteCommentReaction" {
				f.rows[app.ReactionKey{ResourceID: f.current.ID, ActorID: alice, Reaction: "1"}] = domain.CommentReaction{ID: bob}
				err = r.Execute(as(alice), f.current.ID, "1")
			} else {
				_, err = a.Execute(as(alice), f.current.ID, "1")
			}
			if err != errDisk {
				t.Fatal("port error changed", err)
			}
		})
	}
}
func TestCommentReactionReadFailures(t *testing.T) {
	for _, html := range []string{"<p>Before</p>", "<p>After</p>"} {
		u, _, w, f := newCommentWrites()
		w.log.fail = "CommentReactions"
		if _, err := u.Execute(as(alice), f.current.ID, html); err != errDisk {
			t.Fatal("PATCH lost reader error", err)
		}
	}
	_, w, f, id := newComment()
	f.listed = []domain.Comment{{ID: bob}}
	w.log.fail = "CommentReactions"
	u := app.NewListIssueComments(app.ListIssueCommentsDeps{Locks: w.locks, Comments: f, ReadTx: fakeTx{log: w.log}})
	if _, err := u.Execute(as(alice), id, nil, nil); err != errDisk {
		t.Fatal("page lost reader error", err)
	}
	_, w, f, id = newComment()
	u = app.NewListIssueComments(app.ListIssueCommentsDeps{Locks: w.locks, Comments: f, ReadTx: fakeTx{log: w.log}})
	if _, err := u.Execute(as(alice), id, nil, nil); err != nil || f.reactionIDs != nil {
		t.Fatal("empty page read reactions", err, f.reactionIDs)
	}
}
