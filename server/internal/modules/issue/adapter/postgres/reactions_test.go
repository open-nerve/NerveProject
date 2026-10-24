package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func TestIssueReactions(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	w := newWorkspace(t, pool, "acme")
	p := newProject(t, pool, w, "WEB")
	one, two := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, p, one, 1, alice)
	newIssue(t, pool, p, two, 2, alice)
	ids := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	ctx := context.Background()
	for _, n := range []int{4, 3, 2, 1, 0} {
		issue, actor, value := one, alice, "128077"
		if n == 1 {
			actor = bob
		}
		if n == 2 {
			issue = two
		}
		if n == 3 {
			value = "10084-65039"
		}
		if n == 4 {
			value = "1"
		}
		r := domain.IssueReaction{ID: ids[n], WorkspaceID: w, ProjectID: p.id, IssueID: issue, ActorID: actor, Reaction: value}
		if err := s.InsertIssueReaction(ctx, app.IssueReactionWrite{Reaction: r, By: actor, At: now}); err != nil {
			t.Fatal(err)
		}
	}
	exec(t, pool, "UPDATE issue_reactions SET deleted_at=$2,updated_at=$2 WHERE id=$1", ids[4], now.Add(-time.Hour))
	rows, err := s.IssueReactions(ctx, one)
	if err != nil {
		t.Fatal(err)
	}
	got := []uuid.UUID{}
	for _, r := range rows {
		got = append(got, r.ID)
		if r.WorkspaceID != w || r.ProjectID != p.id || r.IssueID != one || !r.CreatedAt.Equal(now) {
			t.Fatal(r)
		}
	}
	if !slices.Equal(got, []uuid.UUID{ids[0], ids[1], ids[3]}) {
		t.Fatal(got)
	}
	for _, tt := range []struct {
		k     app.ReactionKey
		id    uuid.UUID
		found bool
	}{{app.ReactionKey{ResourceID: one, ActorID: alice, Reaction: "128077"}, ids[0], true}, {app.ReactionKey{ResourceID: one, ActorID: bob, Reaction: "128077"}, ids[1], true}, {app.ReactionKey{ResourceID: two, ActorID: alice, Reaction: "128077"}, ids[2], true}, {app.ReactionKey{ResourceID: one, ActorID: alice, Reaction: "1"}, uuid.UUID{}, false}} {
		r, found, err := s.IssueReaction(ctx, tt.k)
		if err != nil || found != tt.found || r.ID != tt.id {
			t.Fatal(tt, r, found, err)
		}
	}
	before := audits(t, pool, "issue_reactions")
	content := rowsAsText(t, pool, "issue_reactions")
	at := now.Add(time.Hour)
	key := app.ReactionKey{ResourceID: one, ActorID: alice, Reaction: "128077"}
	if err := s.DeleteIssueReaction(ctx, app.ReactionDeletion{Key: key, By: alice, At: at}); err != nil {
		t.Fatal(err)
	}
	want := maps.Clone(before)
	want[ids[0]] = deletion(at, alice)
	if after := audits(t, pool, "issue_reactions"); !maps.Equal(after, want) {
		t.Fatal("delete scope", after, want)
	}
	if !maps.Equal(rowsAsText(t, pool, "issue_reactions"), content) {
		t.Fatal("changed reaction content")
	}
	// Deleting again must preserve the first deletion, including its writer.
	if err := s.DeleteIssueReaction(ctx, app.ReactionDeletion{Key: key, By: bob, At: at.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(audits(t, pool, "issue_reactions"), want) {
		t.Fatal("rewrote deleted stamp")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.IssueReactions(cancelled, one); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, _, err := s.IssueReaction(cancelled, key); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.DeleteIssueReaction(cancelled, app.ReactionDeletion{Key: key}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.InsertIssueReaction(cancelled, app.IssueReactionWrite{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
