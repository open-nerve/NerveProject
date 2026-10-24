package migrations_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestCommentReactionConstraints(t *testing.T) {
	pool := newIssueDatabase(t)
	ctx := context.Background()
	comment := "'0199a2b4-0000-7000-8000-00000000000b'"
	if _, err := pool.Exec(ctx, "INSERT INTO issue_comments(id,workspace_id,project_id,issue_id,actor_id) VALUES("+comment+","+issueAcme+","+issueWeb+","+webIssue+","+issueBob+")"); err != nil {
		t.Fatal(err)
	}
	insert := "INSERT INTO comment_reactions(id,workspace_id,project_id,comment_id,actor_id,reaction) VALUES(gen_random_uuid()," + issueAcme + "," + issueWeb + "," + comment + "," + issueBob + ",$1)"
	for _, v := range []string{"128077", "10084-65039", strings.Repeat("1234567-", 15) + "1234567"} {
		if _, err := pool.Exec(ctx, insert, v); err != nil {
			t.Fatal(v, err)
		}
	}
	for _, v := range []string{"", "-1", "12345678", "1--2", strings.Repeat("1-", 16) + "1", strings.Repeat("1", 129)} {
		_, err := pool.Exec(ctx, insert, v)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "23514" || pe.ConstraintName != "comment_reactions_reaction_check" {
			t.Fatal("invalid reaction accepted", v, err)
		}
	}
	_, err := pool.Exec(ctx, insert, "128077")
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23505" || pe.ConstraintName != "comment_reactions_comment_id_actor_id_reaction_key" {
		t.Fatal("duplicate", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE comment_reactions SET deleted_at=now() WHERE reaction='128077'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, insert, "128077"); err != nil {
		t.Fatal("readd", err)
	}
	for _, col := range []string{"comment_id", "actor_id", "reaction"} {
		_, err := pool.Exec(ctx, "UPDATE comment_reactions SET "+col+"=NULL WHERE reaction='128077'")
		if !errors.As(err, &pe) || pe.Code != "23502" || pe.ColumnName != col {
			t.Fatal(col, err)
		}
	}
}

// Removing a work item physically reaches reactions through its comment FK.
func TestCommentReactionTwoLevelCascade(t *testing.T) {
	for _, target := range []string{"issue", "comment"} {
		t.Run(target, func(t *testing.T) {
			pool := newIssueDatabase(t)
			ctx := context.Background()
			for _, pair := range []struct{ comment, issue string }{{"'0199a2b4-0000-7000-8000-00000000000b'", webIssue}, {"'0199a2b4-0000-7000-8000-00000000000c'", opsIssue}} {
				if _, err := pool.Exec(ctx, "INSERT INTO issue_comments(id,workspace_id,project_id,issue_id,actor_id) VALUES("+pair.comment+","+issueAcme+","+issueWeb+","+pair.issue+","+issueBob+")"); err != nil {
					t.Fatal(err)
				}
				if _, err := pool.Exec(ctx, "INSERT INTO comment_reactions(id,workspace_id,project_id,comment_id,actor_id,reaction) VALUES(gen_random_uuid(),"+issueAcme+","+issueWeb+","+pair.comment+","+issueBob+",'1')"); err != nil {
					t.Fatal(err)
				}
			}
			stmt := "DELETE FROM issues WHERE id=" + webIssue
			if target == "comment" {
				stmt = "DELETE FROM issue_comments WHERE issue_id=" + webIssue
			}
			if _, err := pool.Exec(ctx, stmt); err != nil {
				t.Fatal("two-level physical deletion failed", err)
			}
			var n int
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM comment_reactions r JOIN issue_comments c ON c.id=r.comment_id WHERE c.issue_id="+opsIssue).Scan(&n); err != nil || n != 1 {
				t.Fatal("neighbor lost", n, err)
			}
			if err := pool.QueryRow(ctx, "SELECT count(*) FROM comment_reactions").Scan(&n); err != nil || n != 1 {
				t.Fatal("reaction FK did not cascade", n, err)
			}
		})
	}
}
