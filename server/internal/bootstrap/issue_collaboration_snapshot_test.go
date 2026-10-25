package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// collaborationSnapshotRows verifies both primary reads and page-sized batch reads.
type collaborationSnapshotRows struct {
	*issuepg.Store
	inspect func(context.Context)
}

func (s collaborationSnapshotRows) Comments(ctx context.Context, id uuid.UUID, c *domain.CommentCursor, n int) ([]domain.Comment, error) {
	s.inspect(ctx)
	return s.Store.Comments(ctx, id, c, n)
}
func (s collaborationSnapshotRows) CommentReactions(ctx context.Context, ids []uuid.UUID) ([]domain.CommentReaction, error) {
	s.inspect(ctx)
	return s.Store.CommentReactions(ctx, ids)
}
func (s collaborationSnapshotRows) DescriptionVersions(ctx context.Context, id uuid.UUID, c *domain.DescriptionVersionCursor, n int) ([]domain.DescriptionVersion, error) {
	s.inspect(ctx)
	return s.Store.DescriptionVersions(ctx, id, c, n)
}

func TestCollaborationSnapshot(t *testing.T) {
	for _, kind := range []string{"comments", "activities", "versions"} {
		t.Run(kind, func(t *testing.T) {
			w := newIssueWorld(t)
			id := w.newWorkItemIn(t, w.web, `{"name":"Snapshot target","description_html":"<p>Original</p>"}`)
			store := issuepg.New(w.pool)
			ctx := shared.WithActor(pgtest.Soon(t), shared.Actor{UserID: w.ids["bob"]})
			path := commentsOf(id)
			comment := w.newComment(t, "alice", id)
			oldReaction := addCommentReaction(t, w, "alice", comment.ID, "128077")
			version := savedVersion(t, w, id)
			// Historical reference membership changes independently of the target's
			// normal visibility. A title remains when the reference activity is hidden.
			ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Snapshot reference", "SNAP")
			w.makeMember(t, "bob", ops, 15)
			ref := w.newWorkItemIn(t, ops, `{"name":"Historical reference"}`)
			w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+id.String()+"/relations", fmt.Sprintf(`{"relation_type":"relates_to","issue_ids":[%q]}`, ref), 201)
			w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+id.String(), `{"name":"Keep visible title"}`, 200)
			checks := 0
			var invalidSnapshots []string
			inspect := func(ctx context.Context) {
				checks++
				var ro, isolation string
				if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT current_setting('transaction_read_only'), current_setting('transaction_isolation')").Scan(&ro, &isolation); err != nil || ro != "on" || isolation != "repeatable read" {
					invalidSnapshots = append(invalidSnapshots, fmt.Sprintf("read %d: %s/%s: %v", checks, ro, isolation, err))
				}
			}
			projects := activitySnapshotProjects{issueProjects: issueProjects{projects: project.Provide(w.pool).IssueProjects}, inspect: inspect}
			plainLocks := issueapp.NewLocks(projects, store, authorizerOn(w.pool), versionReadOnly{}, versionReadOnly{})
			readTx := postgres.NewTxManager(w.pool, 2*time.Second, postgres.ReadOnlySnapshot())
			rows := collaborationSnapshotRows{Store: store, inspect: inspect}
			execute := func(locks issueapp.Locks) (any, error) {
				switch kind {
				case "comments":
					return issueapp.NewListIssueComments(issueapp.ListIssueCommentsDeps{Locks: locks, Comments: rows, ReadTx: readTx}).Execute(ctx, id, nil, nil)
				case "activities":
					return issueapp.NewListIssueActivities(issueapp.ListIssueActivitiesDeps{Locks: locks, Projects: projects, Activities: activitySnapshotRows{Store: store, inspect: inspect}, ReadTx: readTx}).Execute(ctx, id, nil, nil)
				default:
					return issueapp.NewListDescriptionVersions(issueapp.ListDescriptionVersionsDeps{Locks: locks, Versions: rows, ReadTx: readTx}).Execute(ctx, id, nil, nil)
				}
			}
			before, err := execute(plainLocks)
			if err != nil {
				t.Fatal(err)
			}
			checks = 0
			changed := false
			wrapped := versionSnapshotStore{Store: store, inspect: inspect, afterIssue: func() {
				if changed {
					t.Fatal("target unexpectedly reread")
				}
				changed = true
				// This transaction is acquired from the pool while the snapshot is open;
				// it cannot reuse or nest within the read-only transaction's connection.
				tx, err := w.pool.Begin(pgtest.Soon(t))
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = tx.Rollback(context.Background()) }()
				type statement struct {
					sql  string
					args []any
				}
				statements := []statement{
					{"UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", []any{w.web, w.ids["bob"]}},
					{"UPDATE projects SET guest_view_all_features=false WHERE id=$1", []any{w.web}},
				}
				switch kind {
				case "comments":
					statements = append(statements, statement{"UPDATE issue_comments SET comment_html='<p>Committed edit</p>',comment_stripped='Committed edit' WHERE id=$1", []any{comment.ID}}, statement{"UPDATE comment_reactions SET deleted_at=now() WHERE id=$1", []any{oldReaction.ID}})
				case "activities":
					statements = append(statements, statement{"UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2", []any{ops, w.ids["bob"]}}, statement{"UPDATE issue_activities SET deleted_at=now() WHERE issue_id=$1 AND verb='created'", []any{id}})
				case "versions":
					statements = append(statements, statement{"UPDATE issue_description_versions SET deleted_at=now() WHERE id=$1", []any{version.ID}})
				}
				for _, s := range statements {
					if _, err := tx.Exec(pgtest.Soon(t), s.sql, s.args...); err != nil {
						t.Fatal(err)
					}
				}
				if err := tx.Commit(pgtest.Soon(t)); err != nil {
					t.Fatal(err)
				}
			}}
			locks := issueapp.NewLocks(projects, wrapped, authorizerOn(w.pool), versionReadOnly{}, versionReadOnly{})
			during, err := execute(locks)
			expectedChecks := 2
			if kind == "comments" || kind == "activities" {
				expectedChecks = 3
			}
			if err != nil || !changed || checks != expectedChecks {
				t.Fatal("snapshot did not include first target and every downstream read", kind, checks, err)
			}
			if len(invalidSnapshots) > 0 {
				t.Fatal("first authorization/page escaped independent read snapshot", invalidSnapshots)
			}
			if !reflect.DeepEqual(before, during) {
				t.Fatal("mixed snapshots", before, during)
			}
			if kind == "activities" {
				path = activitiesOf(id)
			}
			if kind == "versions" {
				path = versionsOf(id)
			}
			// The next request sees the new target permission even though its historical
			// reference catalog could still contain other readable projects.
			w.send(t, "bob", http.MethodGet, path, "", 404)
			if _, err := execute(plainLocks); err != domain.ErrIssueNotFound {
				t.Fatal("next application request kept old authorization", err)
			}
			if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE project_members SET role=15 WHERE project_id=$1 AND member_id=$2", w.web, w.ids["bob"]); err != nil {
				t.Fatal(err)
			}
			after, err := execute(plainLocks)
			if err != nil {
				t.Fatal(err)
			}
			if reflect.DeepEqual(after, before) {
				t.Fatal("next request kept old snapshot", after)
			}
			switch kind {
			case "comments":
				got := listIn[answeredComment](t, w.send(t, "bob", http.MethodGet, path, "", 200))
				if len(got) != 1 || got[0].HTML != "<p>Committed edit</p>" || len(got[0].Reactions) != 0 {
					t.Fatal("next page did not see committed comment and reaction", got)
				}
			case "activities":
				got := activityPage(t, w, "bob", path)
				if len(got.Data) != 1 || got.Data[0].Field == nil || *got.Data[0].Field != "name" {
					t.Fatal("next page did not apply both row deletion and historical directory", got)
				}
			case "versions":
				got := listIn[answeredVersion](t, w.send(t, "bob", http.MethodGet, path, "", 200))
				if len(got) != 0 {
					t.Fatal("next page kept deleted version", got)
				}
			}
		})
	}
}
