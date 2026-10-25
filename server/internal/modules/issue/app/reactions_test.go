package app_test

import (
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func newReaction() (*app.AddIssueReaction, *app.RemoveIssueReaction, *world, *fakeReactions, uuid.UUID) {
	_, w, id := newUpdate()
	f := &fakeReactions{fakeRelations: &fakeRelations{fakeIssues: w.issues}, rows: map[app.ReactionKey]domain.IssueReaction{}}
	d := app.AddIssueReactionDeps{Locks: w.locks, Reactions: f, Clock: clockAt{at: clockNow, log: w.log}}
	return app.NewAddIssueReaction(d), app.NewRemoveIssueReaction(d), w, f, id
}
func TestIssueReactions(t *testing.T) {
	t.Run("lifecycle and caller scope", func(t *testing.T) {
		a, r, w, f, id := newReaction()
		got, e := a.Execute(as(alice), id, "10084-65039")
		if e != nil {
			t.Fatal(e)
		}
		if got.ID == (uuid.UUID{}) || got.ActorID != alice || got.Reaction != "10084-65039" || got.IssueID != id || !got.CreatedAt.Equal(clockNow) {
			t.Fatal("wrong reaction", got)
		}
		if f.marked == nil || f.marked.By != alice || !f.marked.At.Equal(clockNow) || f.added == nil || f.added.By != alice || !f.added.At.Equal(clockNow) {
			t.Fatal("audit")
		}
		if w.issues.activities != nil {
			t.Fatal("activity")
		}
		_, e = a.Execute(as(alice), id, "10084-65039")
		wantCode(t, e, "issue.reaction_exists")
		_, e = a.Execute(as(bob), id, "10084-65039")
		if e != nil {
			t.Fatal(e)
		}
		wantRefused(t, r.Execute(as(carol), id, "10084-65039"), domain.ErrReactionNotFound)
		if e = r.Execute(as(alice), id, "10084-65039"); e != nil {
			t.Fatal(e)
		}
		if f.removed == nil || f.removed.Key.ActorID != alice || !f.removed.At.Equal(clockNow) {
			t.Fatal("delete stamp")
		}
		again, e := a.Execute(as(alice), id, "10084-65039")
		if e != nil || again.ID == got.ID {
			t.Fatal("re-add", again, e)
		}
	})
	t.Run("order", func(t *testing.T) {
		a, _, w, _, id := newReaction()
		if _, e := a.Execute(as(alice), id, "128077"); e != nil {
			t.Fatal(e)
		}
		key := app.ReactionKey{ResourceID: id, ActorID: alice, Reaction: "128077"}
		want := append(lockedAs(shared.Action("issue_reaction.write"), id), line("IssueReaction", key), "Now", line("MarkChanged", []uuid.UUID{id}), line("InsertIssueReaction", id))
		if !slices.Equal(w.log.calls, want) {
			t.Fatal(w.log.calls, want)
		}
	})
	t.Run("snapshot starts before visibility", func(t *testing.T) {
		_, _, w, f, id := newReaction()
		u := app.NewListIssueReactions(app.ListIssueReactionsDeps{Locks: w.locks, Reactions: f, ReadTx: fakeTx{log: w.log}})
		got, e := u.Execute(as(alice), id)
		if e != nil || got == nil || len(got) != 0 {
			t.Fatal(got, e)
		}
		if !slices.Equal(w.log.calls, []string{"Begin", line("GetIssue", id), line("Authorize", alice, domain.ActionRead, acme, web), line("IssueReactions", id)}) {
			t.Fatal(w.log.calls)
		}
	})
	for _, port := range []string{"GetIssue", "Authorize", "Lock", "IssueReaction", "MarkChanged", "InsertIssueReaction", "DeleteIssueReaction", "IssueReactions"} {
		t.Run("port "+port, func(t *testing.T) {
			a, r, w, f, id := newReaction()
			if port == "DeleteIssueReaction" {
				f.rows[app.ReactionKey{ResourceID: id, ActorID: alice, Reaction: "1"}] = domain.IssueReaction{ID: bob}
			}
			w.log.fail = port
			var e error
			switch port {
			case "DeleteIssueReaction":
				e = r.Execute(as(alice), id, "1")
			case "IssueReactions":
				_, e = app.NewListIssueReactions(app.ListIssueReactionsDeps{Locks: w.locks, Reactions: f, ReadTx: fakeTx{log: w.log}}).Execute(as(alice), id)
			default:
				_, e = a.Execute(as(alice), id, "1")
			}
			wantStoppedAt(t, w, e, port)
		})
	}
	for _, remove := range []bool{false, true} {
		t.Run(line("hidden", remove), func(t *testing.T) {
			_, _, w, f, id := newReaction()
			w.auth.grants = []shared.Grant{ownOnly}
			w.auth.answers = []error{shared.Forbidden()}
			locks := app.NewLocks(w.project, w.issues, w.auth, fakeGate{log: w.log}, fakeTx{log: w.log})
			d := app.AddIssueReactionDeps{Locks: locks, Reactions: f, Clock: clockAt{at: clockNow, log: w.log}}
			a, r := app.NewAddIssueReaction(d), app.NewRemoveIssueReaction(d)
			var e error
			if remove {
				e = r.Execute(as(alice), id, "1")
			} else {
				_, e = a.Execute(as(alice), id, "1")
			}
			wantRefused(t, e, domain.ErrIssueNotFound)
			if len(w.log.calls) != 2 || f.marked != nil {
				t.Fatal(w.log.calls)
			}
		})
	}
}
func wantCode(t *testing.T, e error, code string) {
	t.Helper()
	var se *shared.Error
	if !errors.As(e, &se) || se.Code != code {
		t.Fatalf("error = %v, want %s", e, code)
	}
}

func TestIssueReactionsRejectBeforeWriting(t *testing.T) {
	for _, remove := range []bool{false, true} {
		for _, test := range []string{"invalid", "project archived", "issue archived", "duplicate or missing"} {
			t.Run(line(test, remove), func(t *testing.T) {
				a, r, w, f, id := newReaction()
				value := "1"
				var want error
				switch test {
				case "invalid":
					value = "-1"
				case "project archived":
					w.project.locked.Archived = true
					want = domain.ErrProjectArchived
				case "issue archived":
					w.issues.archivedAt = &earlier
					want = domain.ErrIssueArchived
				case "duplicate or missing":
					if !remove {
						f.rows[app.ReactionKey{ResourceID: id, ActorID: alice, Reaction: value}] = domain.IssueReaction{ID: bob}
						want = domain.ErrReactionExists
					} else {
						want = domain.ErrReactionNotFound
					}
				}
				var err error
				if remove {
					err = r.Execute(as(alice), id, value)
				} else {
					_, err = a.Execute(as(alice), id, value)
				}
				if test == "invalid" {
					if err == nil || len(w.log.calls) != 0 {
						t.Fatal(err, w.log.calls)
					}
				} else {
					wantRefused(t, err, want)
				}
				if slices.Contains(w.log.calls, "Now") || f.marked != nil || f.added != nil || f.removed != nil {
					t.Fatal("refusal wrote", w.log.calls)
				}
			})
		}
	}
}
