package bootstrap

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"uuid"
)

// getIssueByIdentifier as bootstrap wires it (M4 design 3.7). In acme of
// the world, beside Web, alice makes the projects IŞ, as she gives "iş",
// which createProject upper-cases letter for letter, İŞ, İI, Iİ, ŞI, Şİ,
// deleted since, and Sec, which is private; Web's work items 1, 2, deleted
// since, and 3, archived in Done (in the database); IŞ's, İŞ's, İI's,
// Iİ's, ŞI's and Sec's 1; and walt Web's guest, whom Web shows only his
// own. Then she makes the workspace beta, and in it a project WEB too,
// after acme's, and İŞ, each with its work item 1. A work item is found by
// its identifier among its workspace's projects alone, in any case,
// Turkish letters too: of IŞ and İŞ, which fold alike, and of İI and Iİ,
// the one that is the identifier asked upper-cased letter for letter; an
// archived one as any other. A project deleted since whose identifier
// folds as a live one's and is the identifier asked upper-cased, Şİ, names
// nothing: ŞI's work item is found; a workspace's one dotted project,
// beta's İŞ, is found by a dotless identifier, iş. An identifier that is
// not one, one that names no project of acme, one that folds as İI and Iİ
// do and is neither of them upper-cased, a number of no undeleted work
// item, a project its caller does not see, and another's work item to a
// guest who sees only his own are one answer, issue.not_found, body for
// body. wanda, who sees Web and is none of its members, may not read it;
// olga, no member of acme, finds no workspace, and nor does a slug of
// none, body for body.
func TestAWorkItemIsFoundByItsIdentifier(t *testing.T) {
	w := newIssueWorld(t)
	is := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Is", "iş")
	dotted := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Dotted", "İŞ")
	dotFirst := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Dot first", "İI")
	dotLast := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Dot last", "Iİ")
	shi := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Shi", "ŞI")
	gone := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Shi dotted", "Şİ")
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+gone.String(), "", http.StatusNoContent)
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces/acme/projects", `{"name":"Sec","identifier":"SEC","network":0}`,
		http.StatusCreated), &created)
	web1, web2, web3 := w.newWorkItem(t), w.newWorkItem(t), w.newWorkItem(t)
	isOne, dottedOne := w.newWorkItemIn(t, is, `{"name":"Fix"}`), w.newWorkItemIn(t, dotted, `{"name":"Fix"}`)
	secOne := w.newWorkItemIn(t, created.ID, `{"name":"Fix"}`)
	dotFirstOne, dotLastOne := w.newWorkItemIn(t, dotFirst, `{"name":"Fix"}`), w.newWorkItemIn(t, dotLast, `{"name":"Fix"}`)
	shiOne := w.newWorkItemIn(t, shi, `{"name":"Fix"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+web2.String(), "", http.StatusNoContent)
	if _, err := w.pool.Exec(context.Background(), "UPDATE issues SET state_id = $2, completed_at = updated_at, archived_at = updated_at WHERE id = $1",
		web3, w.done); err != nil {
		t.Fatal(err)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["walt"].String()+`","role":5}]}`, http.StatusCreated)
	w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Beta","slug":"beta"}`, http.StatusCreated)
	betaOne := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "beta", "Web", "WEB"), `{"name":"Fix"}`)
	betaDotted := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "beta", "Dotted", "İŞ"), `{"name":"Fix"}`)
	read := func(name, slug, identifier string) (int, string) {
		return call(t, w.contract, http.MethodGet, w.base+"/api/v0/workspaces/"+slug+"/issues/"+url.PathEscape(identifier), w.tokens[name], "")
	}

	for _, tt := range []struct {
		slug, identifier string
		want             uuid.UUID
		archived         bool
	}{{"acme", "WEB-1", web1, false}, {"acme", "web-1", web1, false}, {"acme", "wEb-1", web1, false}, {"acme", "WEB-3", web3, true},
		{"acme", "iş-1", isOne, false}, {"acme", "ış-1", isOne, false}, {"acme", "IŞ-1", isOne, false}, {"acme", "İş-1", dottedOne, false},
		{"acme", "İŞ-1", dottedOne, false}, {"acme", "İi-1", dotFirstOne, false}, {"acme", "iİ-1", dotLastOne, false},
		{"acme", "sec-1", secOne, false}, {"beta", "web-1", betaOne, false}, {"acme", "Şİ-1", shiOne, false},
		{"beta", "iş-1", betaDotted, false}} {
		status, body := read("alice", tt.slug, tt.identifier)
		var got struct {
			ID         uuid.UUID `json:"id"`
			ArchivedAt *string   `json:"archived_at"`
		}
		if status == http.StatusOK {
			decodeAnswer(t, body, &got)
		}
		if status != http.StatusOK || got.ID != tt.want || (got.ArchivedAt != nil) != tt.archived {
			t.Errorf("%s in %s = %d %s; want 200, the work item %s, archived %v", tt.identifier, tt.slug, status, body, tt.want, tt.archived)
		}
	}

	refusals := map[string]string{}
	for _, tt := range []struct{ name, slug, identifier, code string }{
		{"alice", "acme", "WEB1", "issue.not_found"}, {"alice", "acme", "WEB-0", "issue.not_found"},
		{"alice", "acme", "WEB-01", "issue.not_found"}, {"alice", "acme", "API-1", "issue.not_found"},
		{"alice", "acme", "IS-1", "issue.not_found"}, {"alice", "acme", "ii-1", "issue.not_found"},
		{"alice", "acme", "WEB-4", "issue.not_found"}, {"alice", "acme", "WEB-2", "issue.not_found"},
		{"walt", "acme", "SEC-1", "issue.not_found"}, {"walt", "acme", "WEB-1", "issue.not_found"},
		{"wanda", "acme", "WEB-1", "forbidden"}, {"olga", "acme", "WEB-1", "workspace.not_found"},
		{"alice", "acne", "WEB-1", "workspace.not_found"},
	} {
		status, body := read(tt.name, tt.slug, tt.identifier)
		if refusals[tt.code] == "" {
			refusals[tt.code] = body
		}
		want := http.StatusNotFound
		if tt.code == "forbidden" {
			want = http.StatusForbidden
		}
		if status != want || !strings.Contains(body, `"code":"`+tt.code+`"`) || body != refusals[tt.code] {
			t.Errorf("%s as %s in %s = %d %s; want %d %s, as each such refusal: %s", tt.identifier, tt.name, tt.slug, status, body, want, tt.code,
				refusals[tt.code])
		}
	}
}
