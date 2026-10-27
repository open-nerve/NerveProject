package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A relation to a work item the caller may not reach takes no lock of its
// project (M4 design 3.5 convention 7, 9.3 interleaving 21, adversarial
// review M1): Lab, private, which bob does not see, and Pub, public,
// whose work items he may not read, are held FOR UPDATE by other
// transactions, which a lock of either waits for. bob relates Near, of
// Web, to Lab's and Pub's work items: refused at both places at once,
// without waiting for either. alice, who reaches Lab's, waits for it: a
// write locks the projects it reaches; released, she relates them.
func TestARelationLocksNoProjectItDoesNotReach(t *testing.T) {
	w := newObservedIssueWorld(t)
	lab := createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Lab","identifier":"LAB","network":0}`)
	pub := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Pub", "PUB")
	labs, pubs := w.newWorkItemIn(t, lab, `{"name":"Lab's"}`), w.newWorkItemIn(t, pub, `{"name":"Pub's"}`)
	near := w.newWorkItemIn(t, w.web, `{"name":"Near"}`)
	labHeld := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", lab)
	pubHeld := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", pub)

	sent := func(name, body string) (*http.Request, <-chan answer) {
		req := newRequest(t, http.MethodPost, w.base+relationsOf(near), w.tokens[name], []byte(body))
		w.contract.CheckRequest(t, req)
		w.observe(req).waitTimeout = 10 * time.Second
		return req, sendInBackground(req)
	}
	answered := func(name string, req *http.Request, ch <-chan answer) answer {
		t.Helper()
		a := receiveWithin(t, ch, 10*time.Second, name+"'s answer")
		if a.err != nil {
			t.Fatal(a.err)
		}
		w.contract.CheckResponse(t, req, a.res)
		return a
	}
	req, bobs := sent("bob", relating("relates_to", labs, pubs))
	if a, want := answered("bob", req, bobs), "validation_failed issue_ids[0] not_allowed; issue_ids[1] not_allowed"; a.res.StatusCode !=
		http.StatusUnprocessableEntity || refusalOf(t, a.body, false) != want {
		t.Errorf("bob's relation = %d %s, want 422 %s", a.res.StatusCode, a.body, want)
	}

	req, alices := sent("alice", relating("relates_to", labs))
	requestBlocked(t, w.pool, w.requests.lookup(req.Header.Get("X-Request-Id")), labHeld.Conn().PgConn().PID(), "projects")
	release(t, labHeld)
	if a := answered("alice", req, alices); a.res.StatusCode != http.StatusCreated {
		t.Errorf("alice's relation = %d %s, want 201", a.res.StatusCode, a.body)
	}
	release(t, pubHeld)
}

// relates is name's relation of the world's work item named from to the
// one named to, as kind (workItemNamed).
func relates(name, from, to, kind string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodPost, relationsOf(w.workItemNamed(t, from)),
			relating(kind, w.workItemNamed(t, to)))
	}
}

// A work item the precheck did not reach is not reached under the locks
// either, though the caller comes to see it while his write waits for
// them (M4 design 3.5). walt, acme's admin and Web's guest (adminGuest),
// sees his own work items alone in Web. He relates His to Hers, alice's,
// of Web too, which the guest rule hides from him, while alice's change
// of Web, which shows its guests every work item from then on, holds Web's
// row past its writes (serialize). His write passed the precheck without
// reaching Hers and waits for Web's row, which it locks; once her change
// commits, the decision under the lock would let him see Hers. He is still
// refused at its place, 422, and His and Hers have each of their rows as
// before. Sent again, his relation goes through.
func TestAWorkItemTheCallerComesToSeeWhileHisRelationWaitsIsRefused(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	his := w.newWorkItemBy(t, "walt", w.web, `{"name":"His"}`)
	hers := w.newWorkItemIn(t, w.web, `{"name":"Hers"}`)
	stamps := func() []string { return append(w.workItemStamps(t, his), w.workItemStamps(t, hers)...) }
	before := stamps()
	showsAll := aliceOnWeb(http.MethodPatch, webPath(""), `{"guest_view_all_features":true}`)
	var change, related exchange
	w.serialize(t, showsAll, relates("walt", "His", "Hers", "relates_to"), &change, &related)
	w.answered(t, change, http.StatusOK)
	w.refused(t, related, http.StatusUnprocessableEntity, "validation_failed issue_ids[0] not_allowed")
	if after := stamps(); !slices.Equal(after, before) {
		t.Errorf("His's and Hers' rows after the refusal:\n%q\nwant them as before it:\n%q", after, before)
	}
	w.send(t, "walt", http.MethodPost, relationsOf(his), relating("relates_to", hers), http.StatusCreated)
}

// Removing eitherUnder's existence check authorizes against a project that
// disappeared after the real precheck while LockMany was waiting for its row.
func TestDeleteRelationChecksBothLockedProjects(t *testing.T) {
	for _, deletedEnd := range []string{"first", "second"} {
		t.Run(deletedEnd, func(t *testing.T) {
			w := newIssueWorld(t)
			p := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTHER")
			x, y := w.newWorkItemIn(t, w.web, `{"name":"First"}`), w.newWorkItemIn(t, p, `{"name":"Second"}`)
			r := listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(x), relating("relates_to", y), 201))[0].ID
			gone := w.web
			if deletedEnd == "second" {
				gone = p
			}
			holder := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR NO KEY UPDATE", gone)
			if _, err := holder.Exec(pgtest.Soon(t), "UPDATE projects SET deleted_at=$2 WHERE id=$1", gone, time.Now()); err != nil {
				t.Fatal(err)
			}
			store := issuepg.New(w.pool)
			projects := issueProjects{projects: project.Provide(w.pool).IssueProjects}
			authorizer := &relationCountingAuthorizer{Authorizer: authorizerOn(w.pool)}
			tx := requestTx(w.pool)
			locks := issueapp.NewLocks(projects, store, authorizer, newWriteGate(config.WriteGateConfig{PerProject: 1, MaxQueue: 1, MaxWait: time.Second}), tx)
			remove := issueapp.NewDeleteIssueRelation(issueapp.DeleteIssueRelationDeps{Events: discardIssueEvents{}, Locks: locks, Relations: store, Clock: clock.System{}})
			ctx := shared.WithActor(pgtest.Soon(t), shared.Actor{UserID: w.ids["alice"]})
			result := make(chan error, 1)
			go func() { result <- remove.Execute(ctx, r) }()
			requestBlocked(t, w.pool, tx, holder.Conn().PgConn().PID(), "projects")
			if err := holder.Commit(pgtest.Soon(t)); err != nil {
				t.Fatal(err)
			}
			if err := receiveWithin(t, result, 3*time.Second, "relation deletion"); err != domain.ErrRelationNotFound {
				t.Fatalf("got %v want relation 404", err)
			}
			if got := authorizer.calls.Load(); got != 1 {
				t.Fatalf("authorization calls %d, want only precheck", got)
			}
			if got := w.relationDeletion(t, r, time.Time{}); got != "undeleted" {
				t.Fatal(got)
			}
		})
	}
}

type relationCountingAuthorizer struct {
	shared.Authorizer
	calls atomic.Int32
}

func (a *relationCountingAuthorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, target shared.Target) (shared.Grant, error) {
	a.calls.Add(1)
	return a.Authorizer.Authorize(ctx, actor, action, target)
}
