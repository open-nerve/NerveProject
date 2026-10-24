package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func TestActivityVisibilityQuery(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	actor, other := newAccount(t, pool, "actor@corp.com"), newAccount(t, pool, "other@corp.com")
	ws := newWorkspace(t, pool, "history")
	targetP := newProject(t, pool, ws, "TARGET")
	full := newProject(t, pool, ws, "FULL")
	own := newProject(t, pool, ws, "OWN")
	hiddenP := newProject(t, pool, ws, "HIDDEN")
	target, neighbor, visible, owned, foreign, hidden, missing := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, targetP, target, 1, actor)
	newIssue(t, pool, targetP, neighbor, 2, actor)
	newIssue(t, pool, full, visible, 1, other)
	newIssue(t, pool, own, owned, 1, actor)
	newIssue(t, pool, own, foreign, 2, other)
	newIssue(t, pool, hiddenP, hidden, 1, actor)
	// Historical issues and projects remain readable despite archival/soft deletion.
	exec(t, pool, "UPDATE issues SET deleted_at=$2,archived_at=$2 WHERE id=ANY($1::uuid[])", []uuid.UUID{visible, owned}, now)
	exec(t, pool, "UPDATE projects SET deleted_at=$2,archived_at=$2 WHERE id=ANY($1::uuid[])", []uuid.UUID{own.id}, now)
	projects := []app.VisibleProject{{ID: full.id}, {ID: own.id, OwnOnly: true}}
	add := func(field *string, old, new *uuid.UUID) uuid.UUID {
		t.Helper()
		id := uuid.NewV7()
		err := s.InsertActivities(ctx, []app.ActivityRow{{ID: id, WorkspaceID: ws, ProjectID: targetP.id, IssueID: target, ActorID: actor, At: now, Activity: domain.Activity{Verb: "updated", Field: field, OldIdentifier: old, NewIdentifier: new}}})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	want := []uuid.UUID{}
	for _, field := range []string{"parent", "relates_to", "duplicate", "blocked_by", "blocking"} {
		for _, pair := range [][2]*uuid.UUID{{&visible, &owned}, {nil, &visible}, {&owned, nil}, {nil, nil}} {
			want = append(want, add(&field, pair[0], pair[1]))
		}
		for _, invisible := range []uuid.UUID{foreign, hidden, missing} {
			add(&field, &invisible, &visible)
			add(&field, &visible, &invisible)
		}
	}
	// State, label, account and link identifiers are not issue references.
	for _, field := range []string{"state", "labels", "assignees", "link", "description"} {
		want = append(want, add(&field, &missing, &hidden))
	}
	want = append(want, add(nil, nil, nil))
	gone := add(nil, nil, nil)
	exec(t, pool, "UPDATE issue_activities SET deleted_at=$2 WHERE id=$1", gone, now)
	if err := s.InsertActivities(ctx, []app.ActivityRow{{ID: uuid.NewV7(), WorkspaceID: ws, ProjectID: targetP.id, IssueID: neighbor, ActorID: actor, At: now, Activity: domain.Activity{Verb: "created"}}}); err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(want, func(a, b uuid.UUID) int { return a.Compare(b) })
	rows, err := s.Activities(ctx, target, actor, projects, nil, 100)
	got := []uuid.UUID{}
	for _, r := range rows {
		got = append(got, r.ID)
		if r.WorkspaceID != ws || r.ProjectID != targetP.id || r.IssueID != target || r.ActorID == nil || *r.ActorID != actor || r.Activity.Verb != "updated" || !r.CreatedAt.Equal(now) || !r.UpdatedAt.Equal(now) {
			t.Fatal("row mapping", r)
		}
	}
	if err != nil || !slices.Equal(got, want) {
		t.Fatal("historical reference filtering", got, want, err)
	}
	// Every limit-one page must advance over hidden interleaved rows before LIMIT.
	got = nil
	var after *domain.ActivityCursor
	for n := 0; n < 100; n++ {
		rows, err = s.Activities(ctx, target, actor, projects, after, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) == 0 {
			break
		}
		got = append(got, rows[0].ID)
		after = &domain.ActivityCursor{CreatedAt: rows[0].CreatedAt, ID: rows[0].ID}
	}
	if !slices.Equal(got, want) {
		t.Fatal("filter must precede limit", got, want)
	}
	// Empty directory must deny every nonnull issue reference, even on a cursor request.
	rows, err = s.Activities(ctx, target, actor, nil, &domain.ActivityCursor{CreatedAt: now, ID: want[0]}, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Activity.Field != nil && slices.Contains([]string{"parent", "relates_to", "duplicate", "blocked_by", "blocking"}, *r.Activity.Field) && (r.Activity.OldIdentifier != nil || r.Activity.NewIdentifier != nil) {
			t.Fatal("cursor expanded authorization", r)
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.Activities(cancelled, target, actor, nil, nil, 10); !errors.Is(err, context.Canceled) {
		t.Fatal("query error lost", err)
	}
}
