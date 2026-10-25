package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

func TestWriteDescriptionPreservesAudit(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice := newAccount(t, pool, "alice@corp.com")
	p := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	older, id, other := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for n, key := range []uuid.UUID{older, id, other} {
		newIssue(t, pool, p, key, n+1, alice)
	}
	exec(t, pool, "UPDATE issues SET parent_id=$2 WHERE id=$1", id, older)
	var beforeFields string
	if err := pool.QueryRow(ctx, "SELECT (to_jsonb(i) - 'description_html' - 'description_stripped')::text FROM issues i WHERE id=$1", id).Scan(&beforeFields); err != nil {
		t.Fatal(err)
	}
	before, stamps := rowsAsText(t, pool, "issues"), audits(t, pool, "issues")
	write := app.DescriptionWrite{IssueID: id, HTML: "<p>新的描述</p>", Text: "新的描述"}
	if err := s.WriteDescription(ctx, write); err != nil {
		t.Fatal(err)
	}
	var html string
	var text *string
	if err := pool.QueryRow(ctx, "SELECT description_html,description_stripped FROM issues WHERE id=$1", id).Scan(&html, &text); err != nil {
		t.Fatal(err)
	}
	if html != write.HTML || text == nil || *text != write.Text {
		t.Fatalf("description not stored: %q %v", html, text)
	}
	var afterFields string
	if err := pool.QueryRow(ctx, "SELECT (to_jsonb(i) - 'description_html' - 'description_stripped')::text FROM issues i WHERE id=$1", id).Scan(&afterFields); err != nil {
		t.Fatal(err)
	}
	if afterFields != beforeFields {
		t.Fatal("description write changed other issue columns")
	}
	after := rowsAsText(t, pool, "issues")
	delete(after, id)
	delete(before, id)
	if !maps.Equal(after, before) || !maps.Equal(audits(t, pool, "issues"), stamps) {
		t.Fatal("description write changed neighbor or audit columns")
	}
	// Every internal reader must carry the two columns, including sub-items.
	one, found, err := s.GetIssue(ctx, id)
	if err != nil || !found || one.DescriptionHTML != html || !reflect.DeepEqual(one.DescriptionStripped, text) {
		t.Fatal("GetIssue lost description", one, err)
	}
	many, err := s.GetIssues(ctx, []uuid.UUID{id})
	if err != nil || len(many) != 1 || !reflect.DeepEqual(many[0], one) {
		t.Fatal("GetIssues lost description", many, err)
	}
	subs, err := s.SubIssues(ctx, older, nil)
	if err != nil || len(subs) != 1 || !reflect.DeepEqual(subs[0], one) {
		t.Fatal("SubIssues lost description", subs, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.WriteDescription(cancelled, write); !errors.Is(err, context.Canceled) {
		t.Fatalf("write error lost: %v", err)
	}
}
