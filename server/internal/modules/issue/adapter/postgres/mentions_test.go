package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

func TestMentions(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	a, b := newAccount(t, pool, "a@corp.com"), newAccount(t, pool, "b@corp.com")
	p := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	id, neighbor := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, p, id, 1, a)
	newIssue(t, pool, p, neighbor, 2, a)
	row := app.LinkRows{WorkspaceID: p.workspace, ProjectID: p.id, IssueID: id, IDs: []uuid.UUID{uuid.NewV7(), uuid.NewV7()}, Targets: []uuid.UUID{b, a}, By: a, At: now}
	if err := s.InsertMentions(ctx, row); err != nil {
		t.Fatal(err)
	}
	want := []uuid.UUID{a, b}
	slices.SortFunc(want, uuid.UUID.Compare)
	if got, err := s.MentionIDs(ctx, id); err != nil || !slices.Equal(got, want) {
		t.Fatal(got, err)
	}
	before := rowsAsText(t, pool, "issue_mentions")
	if err := s.RemoveMentions(ctx, app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{b}, By: a, At: now.Add(1)}); err != nil {
		t.Fatal(err)
	}
	if got, err := s.MentionIDs(ctx, id); err != nil || !slices.Equal(got, []uuid.UUID{a}) {
		t.Fatal(got, err)
	}
	if after := rowsAsText(t, pool, "issue_mentions"); after[row.IDs[1]] != before[row.IDs[1]] {
		t.Fatal("retained row rewritten")
	}
	row.IDs, row.Targets = []uuid.UUID{uuid.NewV7()}, []uuid.UUID{b}
	if err := s.InsertMentions(ctx, row); err != nil {
		t.Fatal("soft deletion prevented readd", err)
	}
	if got, err := s.MentionIDs(ctx, neighbor); err != nil || len(got) != 0 {
		t.Fatal("neighbor mentions", got, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.MentionIDs(cancelled, id); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.InsertMentions(cancelled, row); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := s.RemoveMentions(cancelled, app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{a}, By: a, At: now}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
