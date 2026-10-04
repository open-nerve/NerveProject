package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deactivationWorld is endingWorld (ending_world_test.go) and, after it,
// through the API: gamma, carol's workspace, where dave's membership,
// which he took by accepting her invitation, ended by her removal of him;
// her invitation of bob's address to it, which he declined, no member of
// it; dave made acme's admin by alice, then removed by her, an admin whose
// membership ended; bob made beta's admin by alice, so that his roles
// differ across his memberships; Solo archived by bob; Docs, alice's
// project of acme, created after beta's Lab, which bob joined as its
// member, so that his projects' ids cross the workspaces: Web, Ops and Solo
// of acme, Lab of beta, then Docs of acme; acme's id is before beta's. url
// is the database's, for the command line.
type deactivationWorld struct {
	endingWorld
	url          string
	gamma, docs  uuid.UUID
	bobsDeclined uuid.UUID // his invitation to gamma
}

func newDeactivationWorld(t *testing.T) deactivationWorld {
	t.Helper()
	w := deactivationWorld{endingWorld: newEndingWorld(t)}
	w.url = w.pool.Config().ConnString()
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["carol"], `{"name":"gamma","slug":"gamma"}`); status !=
		http.StatusCreated {
		t.Fatalf("creating gamma = %d %s", status, body)
	}
	w.gamma = w.workspace(t, "gamma")
	answerInvitation(t, w.contract, w.base, w.tokens["dave"], "accept", invite(t, w.contract, w.base, w.tokens["carol"], "gamma", "dave@example.com"),
		http.StatusOK)
	link := invite(t, w.contract, w.base, w.tokens["carol"], "gamma", "bob@example.com")
	answerInvitation(t, w.contract, w.base, w.tokens["bob"], "decline", link, http.StatusNoContent)
	w.bobsDeclined = link.id
	w.docs = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Docs", "DOCS")
	inBeta := queryIDs(t, w.pool, `SELECT m.id FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
		WHERE s.slug = 'beta' AND m.member_id = $1`, w.ids["bob"])[0]
	inGamma := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.gamma, w.ids["dave"])[0]
	for _, step := range []struct{ token, method, path, body string }{
		{w.tokens["bob"], http.MethodPost, "/api/v0/projects/" + w.docs.String() + "/join", ""},
		{w.tokens["alice"], http.MethodPatch, "/api/v0/workspace-members/" + inBeta.String(), `{"role":20}`},
		{w.tokens["alice"], http.MethodPatch, "/api/v0/workspace-members/" + w.membership(t, "dave").String(), `{"role":20}`},
		{w.tokens["alice"], http.MethodDelete, "/api/v0/workspace-members/" + w.membership(t, "dave").String(), ""},
		{w.tokens["carol"], http.MethodDelete, "/api/v0/workspace-members/" + inGamma.String(), ""},
		{w.tokens["bob"], http.MethodPost, "/api/v0/projects/" + w.solo.String() + "/archive", ""},
	} {
		if status, body := call(t, w.contract, step.method, w.base+step.path, step.token, step.body); status >= http.StatusMultipleChoices {
			t.Fatalf("%s %s = %d %s", step.method, step.path, status, body)
		}
	}
	if acme, beta := w.workspace(t, "acme"), w.workspace(t, "beta"); acme.Compare(beta) >= 0 || w.lab.Compare(w.docs) >= 0 {
		t.Fatalf("acme %s, beta %s, Lab %s, Docs %s: want beta's id after acme's, Docs's after Lab's", acme, beta, w.lab, w.docs)
	}
	w.preconditions(t)
	return w
}

// preconditions checks the rows of deactivationWorld that decide what a
// deactivation does, each read by what makes it the one it is: dave, an
// admin of acme whose membership ended, and a member of gamma whose
// membership ended; bob, a member of acme and beta's admin; Solo archived;
// the invitation to gamma bob declined. Were one missing, a deactivation
// that counted an ended admin as another, or an ended member as another,
// wrote a role, left an archived project out, or deleted pending
// invitations alone, would pass.
func (w deactivationWorld) preconditions(t *testing.T) {
	t.Helper()
	var got string
	if err := w.pool.QueryRow(soon(t), `SELECT concat_ws('; ',
		(SELECT string_agg(s.slug || ' ' || split_part(u.email, '@', 1) || ' ' || m.role || CASE WHEN m.is_active THEN '' ELSE ' ended' END, ', '
			ORDER BY s.slug COLLATE "C", u.email COLLATE "C") FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
			JOIN users u ON u.id = m.member_id WHERE u.email IN ('bob@example.com', 'dave@example.com')),
		(SELECT 'Solo archived' FROM projects WHERE id = $1 AND archived_at IS NOT NULL),
		(SELECT 'gamma declined' FROM workspace_member_invites WHERE id = $2 AND responded_at IS NOT NULL AND NOT accepted AND deleted_at IS NULL))`,
		w.solo, w.bobsDeclined).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if want := "acme bob 15, acme dave 20 ended, beta bob 20, gamma dave 15 ended; Solo archived; gamma declined"; got != want {
		t.Fatalf("the rows a deactivation decides on: %s; want %s", got, want)
	}
}

// clears has alice join projects, as their admin, so that bob is not their
// only admin: Ops and Lab both, and his deactivation goes through.
func (w deactivationWorld) clears(t *testing.T, projects ...uuid.UUID) {
	t.Helper()
	for _, p := range projects {
		if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+p.String()+"/join", w.tokens["alice"],
			""); status != http.StatusOK {
			t.Fatalf("alice's joining %s = %d %s", p, status, body)
		}
	}
}

// deactivationPath is one path of a deactivation of name's account on w:
// deactivateMe with his bearer token, or `nerve users deactivate` of his
// address, typed in capitals between spaces, which the command normalizes
// to the one his invitations are to; as bootstrap wires each. sent starts
// it in the background, with no test on its goroutine, and returns its
// answer, which waits for its end on the test's goroutine: "" when it is
// done; else its problem's code and detail, "code: detail", the API's
// (whose status it checks against the code's) or the command's error's,
// which the command prints; "500" for a failure, which is no problem of
// the contract.
type deactivationPath struct {
	name string
	sent func(w deactivationWorld, t *testing.T, name string) (answer func() string)
}

// deactivate is p's deactivation of name's account, sent and answered.
func (p deactivationPath) deactivate(w deactivationWorld, t *testing.T, name string) string {
	t.Helper()
	return p.sent(w, t, name)()
}

// byAPI and byCommand are the deactivation's two paths (M3 design 9.3).
var (
	byAPI = deactivationPath{"deactivateMe", func(w deactivationWorld, t *testing.T, name string) func() string {
		t.Helper()
		req := newRequest(t, http.MethodPost, w.base+"/api/v0/me/deactivate", w.tokens[name], nil)
		w.contract.CheckRequest(t, req)
		answered := sendInBackground(req)
		return func() string {
			t.Helper()
			a := receiveWithin(t, answered, 10*time.Second, "the answer to deactivateMe")
			if a.err != nil {
				t.Fatal(a.err)
			}
			w.contract.CheckResponse(t, req, a.res)
			if a.res.StatusCode == http.StatusNoContent && len(a.body) == 0 {
				return ""
			}
			var p httpserver.Problem
			if err := json.Unmarshal(a.body, &p); err != nil {
				t.Fatalf("deactivateMe = %d %s, want a problem", a.res.StatusCode, a.body)
			}
			if p.Code == "internal_error" && a.res.StatusCode == http.StatusInternalServerError {
				return "500"
			}
			if want := map[string]int{"workspace.sole_admin": http.StatusConflict, "project.sole_admin": http.StatusConflict,
				"unauthorized": http.StatusUnauthorized}[p.Code]; a.res.StatusCode != want {
				t.Errorf("deactivateMe = %d %s; want %d for %s", a.res.StatusCode, a.body, want, p.Code)
			}
			return p.Code + ": " + p.Detail
		}
	}}
	byCommand = deactivationPath{"nerve users deactivate", func(w deactivationWorld, t *testing.T, name string) func() string {
		t.Helper()
		done := deactivatingUser(t.Context(), testConfig(t, w.url, false), " "+strings.ToUpper(name)+"@EXAMPLE.COM ")
		return func() string {
			t.Helper()
			run := receiveWithin(t, done, 10*time.Second, "the end of nerve users deactivate")
			var se *shared.Error
			switch {
			case run.err == nil:
				if line := regexp.MustCompile("^deactivated " + name + "@example.com: revoked [0-9]+ sessions and ended its memberships; to " +
					"bring it back, run nerve users activate, then nerve workspaces reactivate-member in each workspace\n$"); !line.MatchString(run.out) {
					t.Errorf("nerve users deactivate printed %q, want %s", run.out, line)
				}
				return ""
			case run.out != "":
				t.Errorf("nerve users deactivate printed %q, refused; want nothing", run.out)
			case errors.As(run.err, &se):
				return se.Code + ": " + run.err.Error()
			}
			return "500"
		}
	}}
	deactivationPaths = []deactivationPath{byAPI, byCommand}
)

// deactivatingUser runs `nerve users deactivate --email email` on cfg in
// the background until ctx ends, and hands over its line and error.
func deactivatingUser(ctx context.Context, cfg config.Config, email string) <-chan commandRun {
	return commandInBackground(func(logs, out io.Writer) error { return Users(ctx, cfg, logs, out, DeactivateUser(email)) })
}

// The two refusals of a deactivation as each path answers them: the
// problem's code and its detail, which the command prints as its line.
const (
	refusedWorkspaceSoleAdmin = "workspace.sole_admin: The workspace would be left without an admin: its only active admin cannot leave it, " +
		"nor can his membership end while it has other active members. It must first be given another admin, or be deleted."
	refusedProjectSoleAdmin = "project.sole_admin: The project would be left without an admin: its only active admin cannot leave it, " +
		"nor can his membership end while it has other active members. It must first be given another admin, or be deleted."
)
