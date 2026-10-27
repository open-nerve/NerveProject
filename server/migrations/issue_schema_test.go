package migrations_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// issueNames are the constraints and indexes of the issue module's tables
// as TestConstraintAndIndexNames reads them: the issue module's part of its want.
var issueNames = []string{
	"draft_issues_author_created_at_idx iw",
	"draft_issues_created_by_id_fkey f c",
	"draft_issues_workspace_id_fkey f c",
	"draft_issues_project_id_fkey f c",
	"draft_issues_updated_by_id_fkey f n",
	"draft_issues_payload_check c",
	"draft_issues_pkey iu",
	"draft_issues_pkey p",
	"draft_issues_workspace_id_idx i",
	"draft_issues_project_id_idx i",
	"comment_reactions_actor_id_fkey f c",
	"comment_reactions_created_by_id_fkey f n",
	"comment_reactions_comment_id_actor_id_reaction_key iuw",
	"comment_reactions_comment_id_fkey f c",
	"comment_reactions_comment_id_idx i",
	"comment_reactions_pkey iu",
	"comment_reactions_pkey p",
	"comment_reactions_project_id_fkey f c",
	"comment_reactions_project_id_idx i",
	"comment_reactions_reaction_check c",
	"comment_reactions_updated_by_id_fkey f n",
	"comment_reactions_workspace_id_fkey f c",
	"comment_reactions_workspace_id_idx i",
	"issue_activities_actor_id_fkey f n",
	"issue_activities_created_by_id_fkey f n",
	"issue_activities_issue_id_created_at_idx i",
	"issue_activities_issue_id_fkey f c",
	"issue_activities_pkey iu",
	"issue_activities_pkey p",
	"issue_activities_project_id_fkey f c",
	"issue_activities_project_id_idx i",
	"issue_activities_updated_by_id_fkey f n",
	"issue_activities_verb_check c",
	"issue_activities_workspace_id_fkey f c",
	"issue_activities_workspace_id_idx i",
	"issue_assignees_assignee_id_fkey f c",
	"issue_assignees_assignee_id_idx iw",
	"issue_assignees_created_by_id_fkey f n",
	"issue_assignees_issue_id_assignee_id_key iuw",
	"issue_assignees_issue_id_fkey f c",
	"issue_assignees_issue_id_idx i",
	"issue_assignees_pkey iu",
	"issue_assignees_pkey p",
	"issue_assignees_project_id_fkey f c",
	"issue_assignees_project_id_idx i",
	"issue_assignees_updated_by_id_fkey f n",
	"issue_assignees_workspace_id_fkey f c",
	"issue_assignees_workspace_id_idx i",
	"issue_mentions_mention_id_fkey f c",
	"issue_mentions_mention_id_idx iw",
	"issue_mentions_created_by_id_fkey f n",
	"issue_mentions_issue_id_mention_id_key iuw",
	"issue_mentions_issue_id_fkey f c",
	"issue_mentions_issue_id_idx i",
	"issue_mentions_pkey iu",
	"issue_mentions_pkey p",
	"issue_mentions_project_id_fkey f c",
	"issue_mentions_project_id_idx i",
	"issue_mentions_updated_by_id_fkey f n",
	"issue_mentions_workspace_id_fkey f c",
	"issue_mentions_workspace_id_idx i",

	"issue_description_versions_workspace_id_fkey f c",
	"issue_description_versions_project_id_fkey f c",
	"issue_description_versions_issue_id_fkey f c",
	"issue_description_versions_owned_by_id_fkey f c",
	"issue_description_versions_created_by_id_fkey f n",
	"issue_description_versions_updated_by_id_fkey f n",
	"issue_description_versions_pkey iu",
	"issue_description_versions_pkey p",
	"issue_description_versions_issue_id_last_saved_at_idx i",
	"issue_description_versions_workspace_id_idx i",
	"issue_description_versions_project_id_idx i",
	"issue_comments_actor_id_fkey f c",
	"issue_comments_created_by_id_fkey f n",
	"issue_comments_issue_id_created_at_idx iw",
	"issue_comments_issue_id_fkey f c",
	"issue_comments_issue_id_idx i",
	"issue_comments_pkey iu",
	"issue_comments_pkey p",
	"issue_comments_project_id_fkey f c",
	"issue_comments_project_id_idx i",
	"issue_comments_updated_by_id_fkey f n",
	"issue_comments_workspace_id_fkey f c",
	"issue_comments_workspace_id_idx i",
	"issue_labels_created_by_id_fkey f n",
	"issue_labels_issue_id_fkey f c",
	"issue_labels_issue_id_idx i",
	"issue_labels_issue_id_label_id_key iuw",
	"issue_labels_label_id_fkey f c",
	"issue_labels_label_id_idx i",
	"issue_labels_pkey iu",
	"issue_labels_pkey p",
	"issue_labels_project_id_fkey f c",
	"issue_labels_project_id_idx i",
	"issue_labels_updated_by_id_fkey f n",
	"issue_labels_workspace_id_fkey f c",
	"issue_labels_workspace_id_idx i",
	"issue_links_created_by_id_fkey f n",
	"issue_links_issue_id_fkey f c",
	"issue_links_issue_id_idx i",
	"issue_links_issue_id_url_key iw",
	"issue_links_issue_id_url_key x",
	"issue_links_metadata_check c",
	"issue_links_pkey iu",
	"issue_links_pkey p",
	"issue_links_project_id_fkey f c",
	"issue_links_project_id_idx i",
	"issue_links_updated_by_id_fkey f n",
	"issue_links_url_check c",
	"issue_links_workspace_id_fkey f c",
	"issue_links_workspace_id_idx i",
	"issue_reactions_actor_id_fkey f c",
	"issue_reactions_created_by_id_fkey f n",
	"issue_reactions_issue_id_actor_id_reaction_key iuw",
	"issue_reactions_issue_id_fkey f c",
	"issue_reactions_issue_id_idx i",
	"issue_reactions_pkey iu",
	"issue_reactions_pkey p",
	"issue_reactions_project_id_fkey f c",
	"issue_reactions_project_id_idx i",
	"issue_reactions_reaction_check c",
	"issue_reactions_updated_by_id_fkey f n",
	"issue_reactions_workspace_id_fkey f c",
	"issue_reactions_workspace_id_idx i",
	"issue_relations_created_by_id_fkey f n",
	"issue_relations_issue_id_fkey f c",
	"issue_relations_issue_id_idx i",
	"issue_relations_not_self_check c",
	"issue_relations_pair_key iuw",
	"issue_relations_pkey iu",
	"issue_relations_pkey p",
	"issue_relations_project_id_fkey f c",
	"issue_relations_project_id_idx i",
	"issue_relations_related_issue_id_fkey f c",
	"issue_relations_related_issue_id_idx i",
	"issue_relations_relation_type_check c",
	"issue_relations_updated_by_id_fkey f n",
	"issue_relations_workspace_id_fkey f c",
	"issue_relations_workspace_id_idx i",
	"issue_subscribers_created_by_id_fkey f n",
	"issue_subscribers_issue_id_fkey f c",
	"issue_subscribers_issue_id_idx i",
	"issue_subscribers_issue_id_subscriber_id_key iuw",
	"issue_subscribers_pkey iu",
	"issue_subscribers_pkey p",
	"issue_subscribers_project_id_fkey f c",
	"issue_subscribers_project_id_idx i",
	"issue_subscribers_subscriber_id_fkey f c",
	"issue_subscribers_subscriber_id_idx iw",
	"issue_subscribers_updated_by_id_fkey f n",
	"issue_subscribers_workspace_id_fkey f c",
	"issue_subscribers_workspace_id_idx i",
	"issues_created_by_id_fkey f n",
	"issues_dates_check c",
	"issues_name_check c",
	"issues_name_trgm_idx iw",
	"issues_not_own_parent_check c",
	"issues_parent_id_fkey f c",
	"issues_parent_id_idx i",
	"issues_pkey iu",
	"issues_pkey p",
	"issues_priority_check c",
	"issues_project_id_fkey f c",
	"issues_project_id_idx i",
	"issues_project_id_sequence_id_key iu",
	"issues_project_id_sequence_id_key u",
	"issues_project_id_state_id_idx iw",
	"issues_sequence_id_check c",
	"issues_state_id_fkey f c",
	"issues_state_id_idx i",
	"issues_updated_by_id_fkey f n",
	"issues_workspace_id_fkey f c",
	"issues_workspace_id_idx i",
}

// The ids the issue tables' tests write, as SQL literals: an account, a
// workspace, two projects with a state each, a label of web, and a work
// item of each project.
const (
	issueBob  = "'0199a2b4-0000-7000-8000-000000000002'"
	issueAcme = "'0199a2b4-0000-7000-8000-000000000003'"
	issueWeb  = "'0199a2b4-0000-7000-8000-000000000004'"
	issueOps  = "'0199a2b4-0000-7000-8000-000000000005'"
	webState  = "'0199a2b4-0000-7000-8000-000000000006'"
	opsState  = "'0199a2b4-0000-7000-8000-000000000007'"
	webBug    = "'0199a2b4-0000-7000-8000-000000000008'"
	webIssue  = "'0199a2b4-0000-7000-8000-000000000009'"
	opsIssue  = "'0199a2b4-0000-7000-8000-00000000000a'"
)

// newIssueDatabase is a database with the rows of the ids above, each work
// item numbered 1, an activity row of web's, and web's work item related
// to ops'.
func newIssueDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := newPool(t, pgtest.NewDatabase(t))
	for _, stmt := range []string{
		"INSERT INTO users (id, email, password, display_name) VALUES (" + issueBob + ", 'bob@corp.com', 'x', 'bob')",
		"INSERT INTO workspaces (id, name, slug) VALUES (" + issueAcme + ", 'Acme', 'acme')",
		"INSERT INTO projects (id, workspace_id, name, identifier) VALUES (" + issueWeb + ", " + issueAcme + ", 'Web', 'WEB'), (" + issueOps +
			", " + issueAcme + ", 'Ops', 'OPS')",
		"INSERT INTO states (id, workspace_id, project_id, name, color) VALUES (" + webState + ", " + issueAcme + ", " + issueWeb +
			", 'Backlog', '#60646C'), (" + opsState + ", " + issueAcme + ", " + issueOps + ", 'Backlog', '#60646C')",
		"INSERT INTO labels (id, workspace_id, project_id, name) VALUES (" + webBug + ", " + issueAcme + ", " + issueWeb + ", 'Bug')",
		"INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id) VALUES (" + webIssue + ", " + issueAcme + ", " +
			issueWeb + ", " + webState + ", 'Work', 1), (" + opsIssue + ", " + issueAcme + ", " + issueOps + ", " + opsState + ", 'Work', 1)",
		"INSERT INTO issue_activities (id, workspace_id, project_id, issue_id) VALUES (gen_random_uuid(), " + issueAcme + ", " + issueWeb +
			", " + webIssue + ")",
		relationOf(issueWeb, webIssue, opsIssue, "relates_to"),
	} {
		if _, err := pool.Exec(context.Background(), stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	return pool
}

// relationOf is the insertion of a relation of the work item issue, of the
// project project, to the work item related, of the type kind.
func relationOf(project, issue, related, kind string) string {
	return "INSERT INTO issue_relations (id, workspace_id, project_id, issue_id, related_issue_id, relation_type) VALUES (gen_random_uuid(), " +
		issueAcme + ", " + project + ", " + issue + ", " + related + ", '" + kind + "')"
}

// The issue tables' CHECKs accept what the domain writes and reject what
// bypasses it (M4 design 4.2, 4.6, 4.11): every priority, verb and stored
// relation type, a start on its target date or with no target, a top-level
// work item, the number 1; and refuse an empty title, a priority outside
// the five, in another case too, the number 0, a work item its own parent,
// a start after its target, a verb outside the three, a relation type
// outside the three stored ones (blocking, which is stored reversed, and
// Plane's start_before among them), a work item related to itself.
func TestIssueChecksRejectCounterexamples(t *testing.T) {
	ctx := context.Background()
	pool := newIssueDatabase(t)
	for _, stmt := range []string{
		"UPDATE issues SET priority = 'urgent'", "UPDATE issues SET priority = 'high'", "UPDATE issues SET priority = 'medium'",
		"UPDATE issues SET priority = 'low'", "UPDATE issues SET priority = 'none'",
		"UPDATE issues SET start_date = '2026-10-17', target_date = '2026-10-17'",
		"UPDATE issues SET start_date = '2026-10-18', target_date = NULL",
		"UPDATE issues SET parent_id = NULL", "UPDATE issues SET sequence_id = 1",
		"UPDATE issue_activities SET verb = 'updated'", "UPDATE issue_activities SET verb = 'deleted'",
		"UPDATE issue_activities SET verb = 'created'", "UPDATE issue_relations SET relation_type = 'blocked_by'",
		"UPDATE issue_relations SET relation_type = 'duplicate'", "UPDATE issue_relations SET relation_type = 'relates_to'",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Errorf("%s: %v, want it accepted", stmt, err)
		}
	}
	for _, tt := range []struct{ name, stmt, constraint string }{
		{"empty title", "UPDATE issues SET name = ''", "issues_name_check"},
		{"unknown priority", "UPDATE issues SET priority = 'other'", "issues_priority_check"},
		{"upper-case priority", "UPDATE issues SET priority = 'Urgent'", "issues_priority_check"},
		{"number 0", "UPDATE issues SET sequence_id = 0", "issues_sequence_id_check"},
		{"its own parent", "UPDATE issues SET parent_id = id", "issues_not_own_parent_check"},
		{"a start after the target", "UPDATE issues SET start_date = '2026-10-18', target_date = '2026-10-17'", "issues_dates_check"},
		{"unknown verb", "UPDATE issue_activities SET verb = 'commented'", "issue_activities_verb_check"},
		{"blocking stored", "UPDATE issue_relations SET relation_type = 'blocking'", "issue_relations_relation_type_check"},
		{"Plane's start_before", "UPDATE issue_relations SET relation_type = 'start_before'", "issue_relations_relation_type_check"},
		{"related to itself", "UPDATE issue_relations SET related_issue_id = issue_id", "issue_relations_not_self_check"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, tt.stmt)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != tt.constraint {
				t.Errorf("%s = %v, want check_violation (23514) of %s", tt.stmt, err, tt.constraint)
			}
		})
	}
}

// A work item's number is unique in its project among every work item,
// deleted ones too (M4 design 3.7: a number is never used again): a second
// one is refused before and after the first is soft-deleted; another
// project holds its own 1. The assignees', labels' and subscribers' keys
// hold among undeleted rows only (4.3–4.5): a second undeleted row of the
// pair is refused, and soft-deleting the first frees the pair; another
// work item holds the same pair of its own. A pair of work items has at
// most one undeleted relation, whatever its type and direction (3.10,
// 4.6): beside web's work item related to ops', ops' is refused a relation
// to web's of another type, and web's one to ops' of another type; web's
// relation to another work item of web, and that one's to ops' work item,
// each sharing one end with the first, are accepted; soft-deleting the
// first frees the pair for ops' relation to web's.
func TestIssueUniqueKeys(t *testing.T) {
	ctx := context.Background()
	pool := newIssueDatabase(t)
	refused := func(t *testing.T, stmt, index string) {
		t.Helper()
		_, err := pool.Exec(ctx, stmt)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != index {
			t.Errorf("%s = %v, want unique_violation (23505) of %s", stmt, err, index)
		}
	}
	second := "INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id) VALUES (gen_random_uuid(), " + issueAcme + ", " +
		issueWeb + ", " + webState + ", 'Again', 1)"
	refused(t, second, "issues_project_id_sequence_id_key")
	if _, err := pool.Exec(ctx, "UPDATE issues SET deleted_at = now() WHERE id = "+webIssue); err != nil {
		t.Fatal(err)
	}
	refused(t, second, "issues_project_id_sequence_id_key")

	for _, k := range []struct{ table, column, value, index string }{
		{"issue_assignees", "assignee_id", issueBob, "issue_assignees_issue_id_assignee_id_key"},
		{"issue_mentions", "mention_id", issueBob, "issue_mentions_issue_id_mention_id_key"},
		{"issue_labels", "label_id", webBug, "issue_labels_issue_id_label_id_key"},
		{"issue_subscribers", "subscriber_id", issueBob, "issue_subscribers_issue_id_subscriber_id_key"},
	} {
		t.Run(k.table, func(t *testing.T) {
			insert := func(project, issue string) string {
				return "INSERT INTO " + k.table + " (id, workspace_id, project_id, issue_id, " + k.column + ") VALUES (gen_random_uuid(), " +
					issueAcme + ", " + project + ", " + issue + ", " + k.value + ")"
			}
			for _, stmt := range []string{insert(issueWeb, webIssue), insert(issueOps, opsIssue)} {
				if _, err := pool.Exec(ctx, stmt); err != nil {
					t.Fatalf("%s: %v", stmt, err)
				}
			}
			refused(t, insert(issueWeb, webIssue), k.index)
			if _, err := pool.Exec(ctx, "UPDATE "+k.table+" SET deleted_at = now() WHERE issue_id = "+webIssue); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, insert(issueWeb, webIssue)); err != nil {
				t.Errorf("after the first is deleted, %s = %v; want the pair free", insert(issueWeb, webIssue), err)
			}
		})
	}

	t.Run("issue_relations", func(t *testing.T) {
		other := "'0199a2b4-0000-7000-8000-00000000000b'"
		refused(t, relationOf(issueOps, opsIssue, webIssue, "blocked_by"), "issue_relations_pair_key")
		refused(t, relationOf(issueWeb, webIssue, opsIssue, "duplicate"), "issue_relations_pair_key")
		for _, stmt := range []string{
			"INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id) VALUES (" + other + ", " + issueAcme + ", " + issueWeb +
				", " + webState + ", 'Other', 2)",
			relationOf(issueWeb, webIssue, other, "relates_to"), relationOf(issueWeb, other, opsIssue, "blocked_by"),
			"UPDATE issue_relations SET deleted_at = now() WHERE issue_id = " + webIssue + " AND related_issue_id = " + opsIssue,
			relationOf(issueOps, opsIssue, webIssue, "blocked_by"),
		} {
			if _, err := pool.Exec(ctx, stmt); err != nil {
				t.Errorf("%s: %v, want it accepted", stmt, err)
			}
		}
	})
}

// Comments require a human actor and default to an unedited empty document.
func TestCommentDefaultsAndRequiredActor(t *testing.T) {
	pool := newIssueDatabase(t)
	ctx := context.Background()
	insert := "INSERT INTO issue_comments (id,workspace_id,project_id,issue_id,actor_id) VALUES (gen_random_uuid()," + issueAcme + "," + issueWeb + "," + webIssue + ","
	var html, text string
	var unedited bool
	if err := pool.QueryRow(ctx, insert+issueBob+") RETURNING comment_html,comment_stripped,edited_at IS NULL").Scan(&html, &text, &unedited); err != nil || html != "<p></p>" || text != "" || !unedited {
		t.Fatal(html, text, unedited, err)
	}
	_, err := pool.Exec(ctx, insert+"NULL)")
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23502" || pe.ColumnName != "actor_id" {
		t.Fatalf("null actor: %v", err)
	}
}

// The database enforces the same decimal shape and partial uniqueness as the domain.
func TestIssueReactionConstraints(t *testing.T) {
	pool := newIssueDatabase(t)
	ctx := context.Background()
	insert := "INSERT INTO issue_reactions(id,workspace_id,project_id,issue_id,actor_id,reaction) VALUES(gen_random_uuid()," + issueAcme + "," + issueWeb + "," + webIssue + "," + issueBob + ",$1)"
	for _, value := range []string{"128077", "10084-65039", strings.Repeat("1234567-", 15) + "1234567"} {
		if _, err := pool.Exec(ctx, insert, value); err != nil {
			t.Fatal(value, err)
		}
	}
	for _, value := range []string{"", "-1", "12345678", "1--2", strings.Repeat("1-", 16) + "1", strings.Repeat("1", 129)} {
		_, err := pool.Exec(ctx, insert, value)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "23514" || pe.ConstraintName != "issue_reactions_reaction_check" {
			t.Fatalf("invalid %q: %v", value, err)
		}
	}
	_, err := pool.Exec(ctx, insert, "128077")
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23505" || pe.ConstraintName != "issue_reactions_issue_id_actor_id_reaction_key" {
		t.Fatal("duplicate", err)
	}
	if _, err := pool.Exec(ctx, "UPDATE issue_reactions SET deleted_at=now() WHERE reaction='128077'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, insert, "128077"); err != nil {
		t.Fatal("re-add", err)
	}
	for _, column := range []string{"actor_id", "reaction"} {
		_, err := pool.Exec(ctx, "UPDATE issue_reactions SET "+column+"=NULL WHERE reaction='128077'")
		if !errors.As(err, &pe) || pe.Code != "23502" || pe.ColumnName != column {
			t.Fatal(column, err)
		}
	}
}
