package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// createIssue as bootstrap wires it makes every row it writes at the time
// of its request, one moment, from the clock issue.New takes, and as its
// caller (M4 design 3.7): alice creates a work item in Web of the world,
// in Todo, assigned to bob and labelled Bug. The work item, its assignee,
// its label, its two subscribers, alice and bob, and its two activities,
// in both of which alice acts, are seven rows, each made and last changed
// by alice at one moment within the request. The second activity names
// bob by his display name as identity has it (bootstrap/ports.go's
// issueProfiles).
func TestCreatingAWorkItemStampsItsRows(t *testing.T) {
	w := newIssueWorld(t)
	before := time.Now().Truncate(time.Microsecond)
	item := w.newWorkItem(t)
	after := time.Now()

	var made time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT created_at FROM issues WHERE id = $1", item).Scan(&made); err != nil {
		t.Fatal(err)
	}
	if made.Before(before) || made.After(after) {
		t.Errorf("the work item made at %v; want a moment within %v to %v", made, before, after)
	}
	activity := "issue_activities made by alice made, changed by alice made, of alice"
	want := []string{activity, activity, "issue_assignees made by alice made, changed by alice made, of bob",
		"issue_description_versions made by alice made, changed by alice made, of alice",
		"issue_labels made by alice made, changed by alice made", "issue_subscribers made by alice made, changed by alice made, of alice",
		"issue_subscribers made by alice made, changed by alice made, of bob", "issues made by alice made, changed by alice made"}
	if got := w.workItemStamps(t, item); !slices.Equal(got, want) {
		t.Errorf("the work item's rows:\n%q\nwant\n%q", got, want)
	}

	var named, displayName string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT a.new_value, u.display_name FROM issue_activities a, users u
		WHERE a.issue_id = $1 AND a.field = 'assignees' AND a.new_identifier = $2 AND u.id = $2`, item, w.ids["bob"]).Scan(&named,
		&displayName); err != nil {
		t.Fatal(err)
	}
	if named != displayName {
		t.Errorf("the activity assigning bob names him %q; want his display name %q", named, displayName)
	}
}

// testIssueConfig is the work-item side's configuration of the tests'
// apps: the write gate's defaults, and a lock timeout under testConfig's
// request timeout. They are test-only values, which configuration loading
// would refuse, max_wait and the lock timeout together not being under
// the request timeout; the test worlds' configurations are never
// validated.
var testIssueConfig = config.IssueConfig{WriteGate: config.WriteGateConfig{PerProject: 2, MaxWait: 2 * time.Second, MaxQueue: 64},
	WriteLockTimeout: 4 * time.Second}

// newIssueRoute is the issue module (issue.New) on pool, with tx its
// transactions, the write gate of testIssueConfig, the Authorizer on pool
// and project's, workspace's and identity's ports converted, all as
// bootstrap wires them.
func newIssueRoute(t *testing.T, pool *pgxpool.Pool, tx shared.TxManager) moduleRoute {
	t.Helper()
	return newRoute(t, issue.New(issue.Deps{Pool: pool, Tx: tx,
		ReadTx:    postgres.NewTxManager(pool, 2*time.Second, postgres.ReadOnlySnapshot()),
		WriteGate: newWriteGate(testIssueConfig.WriteGate), Clock: clock.System{},
		Authorizer: authorizerOn(pool), Projects: issueProjects{projects: project.Provide(pool).IssueProjects},
		Workspaces: issueWorkspaces{directory: workspace.Provide(pool).WorkspaceDirectory},
		Profiles:   issueProfiles{profiles: identity.Provide(pool).PublicProfiles}}).Register)
}

// Each write on the work-item side runs every statement of its transaction
// on that transaction's connection (M4 design 3.5, 6.7): its lock, the
// work item read again under it, the decision under it, its checks of the
// states, the members and the labels, its number, its writes, the display
// names, and its answer. The issue module and the Authorizer are wired as
// bootstrap wires them, on a pool of one connection (poolOfOne): a
// statement sent through the pool rather than the transaction would wait
// for a second connection that never comes, and its request fail at the
// request's deadline. On Web of the world, alice creates a work item in
// Todo, assigned to bob and labelled Bug; with bob made Web's default
// assignee, one without assignees, which he takes; one labelled with a
// label no project has, which the check under the lock refuses (422); walt,
// who sees Web, is refused before the lock (403). She changes the first:
// into Done, assigned to herself instead of bob, without Bug; then its name
// as it is, which writes nothing; then with a label no project has (422),
// and walt is refused (403); and she reads it, without a transaction. walt,
// made acme's admin and Web's guest (adminGuest), changes His, his own,
// which she has put under the first, hidden from him: the first is read
// under the lock with the count of the answer's sub-items, both on the
// transaction's connection, and the answer names no parent. She makes Part
// under the first, its parent read under the lock, then moves Part
// under the second, the second's ancestry read under it, and puts Low
// under Part. She relates the first to Far, of Ops, as blocking it, both
// projects locked at once, and deletes the relation. She archives the
// first, in Done, and unarchives it; she deletes the second, and Part and
// Low under it. With Web archived, her creation and her change are
// refused under the lock (409).
func TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection(t *testing.T) {
	w := newIssueWorld(t)
	one := poolOfOne(t, w.pool.Config().ConnString())
	route := newIssueRoute(t, one, postgres.NewTxManager(one, 2*time.Second))
	path := "/api/v0/projects/" + w.web.String() + "/issues"
	alice := w.ids["alice"]
	send := func(caller uuid.UUID, body string, want int) string {
		t.Helper()
		return route.answerWithin(t, w.contract, http.MethodPost, path, caller, body, want)
	}

	commentID := w.newComment(t, "alice", w.newWorkItem(t)).ID
	route.answerWithin(t, w.contract, http.MethodPost, commentReactionsOf(commentID), alice, `{"reaction":"128077"}`, http.StatusCreated)
	route.answerWithin(t, w.contract, http.MethodDelete, commentReactionsOf(commentID)+"/128077", alice, "", http.StatusNoContent)
	reactionID := w.newWorkItem(t)
	route.answerWithin(t, w.contract, http.MethodPost, reactionsOf(reactionID), alice, `{"reaction":"128077"}`, http.StatusCreated)
	route.answerWithin(t, w.contract, http.MethodGet, reactionsOf(reactionID), alice, "", http.StatusOK)
	route.answerWithin(t, w.contract, http.MethodDelete, reactionsOf(reactionID)+"/128077", alice, "", http.StatusNoContent)
	var comment answeredComment
	decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, commentsOf(w.newWorkItem(t)), alice, `{"comment_html":"<p>Hi</p>"}`, http.StatusCreated), &comment)
	route.answerWithin(t, w.contract, http.MethodPatch, commentPath(comment.ID), alice, `{"comment_html":"<p>Changed</p>"}`, http.StatusOK)
	route.answerWithin(t, w.contract, http.MethodDelete, commentPath(comment.ID), alice, "", http.StatusNoContent)
	var first struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, send(alice, strings.TrimSuffix(w.fullIssue(), "}")+fmt.Sprintf(`,"description_html":%q}`, mentionHTML(w.ids["bob"])), http.StatusCreated), &first)
	route.answerWithin(t, w.contract, http.MethodPatch, "/api/v0/issues/"+first.ID.String(), alice, fmt.Sprintf(`{"description_html":%q}`, mentionHTML(w.ids["alice"])), 200)
	if _, err := w.pool.Exec(context.Background(), "UPDATE projects SET default_assignee_id = $2 WHERE id = $1", w.web, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	var defaulted struct {
		ID          uuid.UUID   `json:"id"`
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
	}
	if decodeAnswer(t, send(alice, `{"name":"Defaulted"}`, http.StatusCreated), &defaulted); !slices.Equal(defaulted.AssigneeIDs,
		[]uuid.UUID{w.ids["bob"]}) {
		t.Errorf("a work item without assignees is assigned to %v; want bob, Web's default assignee", defaulted.AssigneeIDs)
	}
	send(alice, `{"name":"Fix","label_ids":["`+uuid.NewV7().String()+`"]}`, http.StatusUnprocessableEntity)
	send(w.ids["walt"], `{"name":"Fix"}`, http.StatusForbidden)
	change := func(caller uuid.UUID, body string, want int) string {
		t.Helper()
		return route.answerWithin(t, w.contract, http.MethodPatch, "/api/v0/issues/"+first.ID.String(), caller, body, want)
	}
	var changed struct {
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
		LabelIDs    []uuid.UUID `json:"label_ids"`
	}
	if decodeAnswer(t, change(alice, `{"state_id":"`+w.done.String()+`","assignee_ids":["`+alice.String()+`"],"label_ids":[]}`,
		http.StatusOK), &changed); !slices.Equal(changed.AssigneeIDs, []uuid.UUID{alice}) || len(changed.LabelIDs) != 0 {
		t.Errorf("the work item changed is assigned to %v, labelled %v; want alice alone, no label", changed.AssigneeIDs, changed.LabelIDs)
	}
	change(alice, `{"name":"Fix"}`, http.StatusOK)
	change(alice, `{"label_ids":["`+uuid.NewV7().String()+`"]}`, http.StatusUnprocessableEntity)
	change(w.ids["walt"], `{"name":"Mend"}`, http.StatusForbidden)
	route.answerWithin(t, w.contract, http.MethodGet, "/api/v0/issues/"+first.ID.String(), alice, "", http.StatusOK)
	w.adminGuest(t)
	var his storedIssue
	decodeAnswer(t, send(w.ids["walt"], `{"name":"His"}`, http.StatusCreated), &his)
	hisPath := "/api/v0/issues/" + his.ID.String()
	route.answerWithin(t, w.contract, http.MethodPatch, hisPath, alice, `{"parent_id":"`+first.ID.String()+`"}`, http.StatusOK)
	var mended storedIssue
	if decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPatch, hisPath, w.ids["walt"], `{"name":"Mended"}`, http.StatusOK),
		&mended); mended.ParentID != nil {
		t.Errorf("walt's His, under the first, is answered under %s; want no parent, the first being hidden from him", mended.ParentID)
	}
	var part struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, send(alice, `{"name":"Part","parent_id":"`+first.ID.String()+`"}`, http.StatusCreated), &part)
	route.answerWithin(t, w.contract, http.MethodPatch, "/api/v0/issues/"+part.ID.String(), alice, `{"parent_id":"`+defaulted.ID.String()+`"}`,
		http.StatusOK)
	var low struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, send(alice, `{"name":"Low"}`, http.StatusCreated), &low)
	route.answerWithin(t, w.contract, http.MethodPost, subIssues(part.ID), alice, subIssueIDs(low.ID), http.StatusOK)
	far := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS"), `{"name":"Far"}`)
	related := listIn[answeredRelation](t, route.answerWithin(t, w.contract, http.MethodPost, relationsOf(first.ID), alice, relating("blocking", far),
		http.StatusCreated))
	route.answerWithin(t, w.contract, http.MethodDelete, relationPath(related[0].ID), alice, "", http.StatusNoContent)
	for _, verb := range []string{"archive", "unarchive"} {
		route.answerWithin(t, w.contract, http.MethodPost, "/api/v0/issues/"+first.ID.String()+"/"+verb, alice, "", http.StatusOK)
	}
	route.answerWithin(t, w.contract, http.MethodDelete, "/api/v0/issues/"+defaulted.ID.String(), alice, "", http.StatusNoContent)
	if _, err := w.pool.Exec(context.Background(), "UPDATE projects SET archived_at = now() WHERE id = $1", w.web); err != nil {
		t.Fatal(err)
	}
	send(alice, `{"name":"Fix"}`, http.StatusConflict)
	change(alice, `{"name":"Mend"}`, http.StatusConflict)
}
