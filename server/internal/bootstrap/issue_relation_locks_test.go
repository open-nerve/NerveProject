package bootstrap

import (
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
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
	w := newIssueWorld(t)
	var lab struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces/acme/projects", `{"name":"Lab","identifier":"LAB","network":0}`,
		http.StatusCreated), &lab)
	pub := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Pub", "PUB")
	labs, pubs := w.newWorkItemIn(t, lab.ID, `{"name":"Lab's"}`), w.newWorkItemIn(t, pub, `{"name":"Pub's"}`)
	near := w.newWorkItemIn(t, w.web, `{"name":"Near"}`)
	labHeld := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", lab.ID)
	pubHeld := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", pub)

	sent := func(name, body string) (*http.Request, <-chan answer) {
		req := newRequest(t, http.MethodPost, w.base+relationsOf(near), w.tokens[name], []byte(body))
		w.contract.CheckRequest(t, req)
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
	pgtest.WaitForLockWaitBehind(t, w.pool, labHeld.Conn().PgConn(), 10*time.Second)
	release(t, labHeld)
	if a := answered("alice", req, alices); a.res.StatusCode != http.StatusCreated {
		t.Errorf("alice's relation = %d %s, want 201", a.res.StatusCode, a.body)
	}
	release(t, pubHeld)
}
