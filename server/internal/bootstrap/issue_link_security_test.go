package bootstrap

import (
	"errors"
	"net/http"
	"reflect"
	"testing"
	"uuid"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type linkRequest struct {
	op, method, path, body string
	want                   int
}

func linkRequests(id, link uuid.UUID) []linkRequest {
	return []linkRequest{
		{"listIssueLinks", http.MethodGet, linksOf(id), "", 200},
		{"createIssueLink", http.MethodPost, linksOf(id), `{"url":"http://169.254.169.254/security-new"}`, 201},
		{"updateIssueLink", http.MethodPatch, linkPath(link), `{"url":"http://169.254.169.254/security-changed"}`, 200},
		{"deleteIssueLink", http.MethodDelete, linkPath(link), "", 204},
	}
}

func TestIssueLinkHiddenAnswers(t *testing.T) {
	w := newIssueWorld(t)
	id, deleted := w.newWorkItem(t), w.newWorkItem(t)
	link := createLink(t, w, "alice", id, "http://169.254.169.254/hidden")
	deletedLink := createLink(t, w, "bob", deleted, "http://169.254.169.254/deleted-parent")
	own := w.newWorkItemBy(t, "bob", w.web, `{"name":"Guest owns parent"}`)
	ownLink := createLink(t, w, "alice", own, "http://169.254.169.254/own")
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+deleted.String(), "", 204)
	w.send(t, "alice", http.MethodPatch, bobsMembership(t, w), `{"role":5}`, 200)
	w.adminGuest(t)
	missing := linkRequests(uuid.NewV7(), uuid.NewV7())
	gone := linkRequests(deleted, deletedLink.ID)
	for _, who := range []string{"bob", "walt"} {
		for i, op := range linkRequests(id, link.ID) {
			t.Run(who+"/"+op.op, func(t *testing.T) {
				before := collaborationState(t, w)
				answer := func(path string) wholeAnswer {
					req := newRequest(t, op.method, w.base+path, w.tokens[who], []byte(op.body))
					w.contract.CheckRequest(t, req)
					res, body := send(t, req)
					w.contract.CheckResponse(t, req, res)
					if res.StatusCode != 404 {
						t.Fatal(res.StatusCode, string(body))
					}
					return wholeAnswerOf(res, body)
				}
				absent := answer(missing[i].path)
				for _, path := range []string{op.path, gone[i].path} {
					if actual := answer(path); actual != absent {
						t.Fatal("hidden/deleted parent differs from absent", actual, absent)
					}
				}
				if !reflect.DeepEqual(before, collaborationState(t, w)) {
					t.Fatal("hidden request changed business rows")
				}
			})
		}
	}
	// Visibility and ActionUpdate are independent: the same guest may read
	// his own parent, but only the workspace admin's grant permits a write.
	for _, op := range linkRequests(own, ownLink.ID) {
		want := 403
		if op.method == http.MethodGet {
			want = 200
		}
		w.send(t, "bob", op.method, op.path, op.body, want)
	}
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"guest_view_all_features":true}`, 200)
	for _, who := range []string{"bob", "walt"} {
		for _, op := range linkRequests(id, link.ID) {
			t.Run("all features/"+who+"/"+op.op, func(t *testing.T) {
				want := op.want
				if who == "bob" && op.method != http.MethodGet {
					want = 403
				}
				w.send(t, who, op.method, op.path, op.body, want)
			})
		}
	}
}

func testIssueLinkPrecheckResources(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	link := createLink(t, w, "alice", id, "http://169.254.169.254/precheck")
	one := poolOfOne(t, w.pool.Config().ConnString())
	store := issuepg.New(one)
	projects := issueProjects{projects: project.Provide(one).IssueProjects}
	resources := &collaborationWriteResources{}
	locks := issueapp.NewLocks(projects, store, authorizerOn(one), resources, resources)
	before := collaborationState(t, w)
	held := lockedTables(t, w.pool, issueTables, "EXCLUSIVE")
	for _, who := range []string{"walt", "wanda", "olga"} {
		ctx := shared.WithActor(pgtest.Soon(t), shared.Actor{UserID: w.ids[who]})
		for _, op := range []string{"create", "update", "delete"} {
			var err error
			switch op {
			case "create":
				_, err = issueapp.NewCreateIssueLink(issueapp.CreateIssueLinkDeps{Locks: locks, Links: store, Clock: clock.System{}, Events: discardIssueEvents{}}).Execute(ctx, id, domain.NewLink{URL: "http://169.254.169.254/refused"})
			case "update":
				title := "Refused"
				_, err = issueapp.NewUpdateIssueLink(issueapp.UpdateIssueLinkDeps{Locks: locks, Links: store, Clock: clock.System{}, Events: discardIssueEvents{}}).Execute(ctx, link.ID, domain.LinkPatch{SetTitle: true, Title: &title})
			case "delete":
				err = issueapp.NewDeleteIssueLink(issueapp.DeleteIssueLinkDeps{Locks: locks, Links: store, Clock: clock.System{}, Events: discardIssueEvents{}}).Execute(ctx, link.ID)
			}
			var want error = shared.Forbidden()
			if who == "olga" {
				want = domain.ErrIssueNotFound
				if op != "create" {
					want = domain.ErrLinkNotFound
				}
			}
			if !errors.Is(err, want) {
				t.Fatalf("%s as %s: %v, want %v", op, who, err, want)
			}
		}
	}
	release(t, held)
	if resources.gates != 0 || resources.transactions != 0 || one.Stat().AcquiredConns() != 0 {
		t.Fatal("refusal acquired write resources", resources, one.Stat().AcquiredConns())
	}
	if !reflect.DeepEqual(before, collaborationState(t, w)) {
		t.Fatal("refusal changed rows")
	}
}
