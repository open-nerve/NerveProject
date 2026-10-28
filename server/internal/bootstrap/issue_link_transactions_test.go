package bootstrap

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestIssueLinkWritesLockAndRecheck(t *testing.T) {
	t.Run("project first", func(t *testing.T) {
		w := newIssueWorld(t)
		for _, iw := range issueWrites {
			if strings.HasSuffix(iw.op, "IssueLink") {
				t.Run(iw.op, func(t *testing.T) { iw.checkProjectLocks(t, w) })
			}
		}
	})
	t.Run("precheck uses no write resources", testIssueLinkPrecheckResources)
	for _, operation := range []string{"createIssueLink", "updateIssueLink", "deleteIssueLink"} {
		for _, own := range []bool{false, true} {
			for _, removed := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/own=%t/removed=%t", operation, own, removed), func(t *testing.T) {
					w := newIssueWorld(t)
					w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"network":0}`, 200)
					creator := "alice"
					if own {
						creator = "bob"
					}
					id := w.newWorkItemBy(t, creator, w.web, `{"name":"Link membership wait"}`)
					link := createLink(t, w, "alice", id, "http://169.254.169.254/member")
					waitUniqueLinkCrawl(t, w, link)
					var op linkRequest
					for _, candidate := range linkRequests(id, link.ID) {
						if candidate.op == operation {
							op = candidate
						}
					}
					before := collaborationState(t, w)
					change, status := demotesBob, 200
					if removed {
						change, status = removesBob, 204
					}
					var changed, written exchange
					w.serializeAtObserved(t, "projects", change, collaborationAt("bob", op.method, op.path, op.body, clock.System{}), &changed, &written, func(probe *requestProbe) {
						if lock := requestGateRowLock(t, w.pool, w.pool, probe, "projects WHERE id = $1", w.web); lock != "FOR NO KEY UPDATE" {
							t.Fatal("membership holder lost project N lock", lock)
						}
					})
					w.answered(t, changed, status)
					want, code := 404, "issue.link_not_found"
					if operation == "createIssueLink" {
						code = "issue.not_found"
					}
					if own && !removed {
						want, code = 403, "forbidden"
					}
					w.refused(t, written, want, code)
					if !reflect.DeepEqual(before, collaborationState(t, w)) {
						t.Fatal("authorization changed while waiting, but write changed rows")
					}
					if want == 404 {
						missing := uuid.NewV7()
						path := linksOf(missing)
						if operation != "createIssueLink" {
							path = linkPath(missing)
						}
						req, rec := newIssueRoute(t, w.pool, nil).send(op.method, path, w.ids["bob"], op.body)
						w.contract.CheckResponse(t, req, rec.Result())
						if wholeAnswerOf(written.rec.Result(), written.rec.Body.Bytes()) != wholeAnswerOf(rec.Result(), rec.Body.Bytes()) {
							t.Fatal("rechecked hidden answer differs from absent")
						}
					}
				})
			}
		}
	}
	for _, update := range []bool{false, true} {
		for _, firstActor := range []string{"alice", "bob"} {
			t.Run(fmt.Sprintf("duplicate/update=%t/first=%s", update, firstActor), func(t *testing.T) {
				w := newIssueWorld(t)
				id := w.newWorkItem(t)
				method, firstPath, secondPath := http.MethodPost, linksOf(id), linksOf(id)
				if update {
					firstPath = linkPath(createLink(t, w, "bob", id, "http://169.254.169.254/first").ID)
					secondPath = linkPath(createLink(t, w, "alice", id, "http://169.254.169.254/second").ID)
					method = http.MethodPatch
				}
				secondActor := "bob"
				if firstActor == "bob" {
					secondActor = "alice"
				}
				body := `{"url":"http://169.254.169.254/contended"}`
				var first, second exchange
				w.serializeAtAfter(t, "projects", collaborationAt(firstActor, method, firstPath, body, clock.System{}), collaborationAt(secondActor, method, secondPath, body, clock.System{}), &first, &second, func() {
					assertLinkProjectLock(t, w)
				})
				want := 201
				if update {
					want = 200
				}
				w.answered(t, first, want)
				w.refused(t, second, 409, "issue.link_exists")
				if got := queryIDs(t, w.pool, "SELECT id FROM issue_links WHERE issue_id=$1 AND url=$2 AND deleted_at IS NULL", id, "http://169.254.169.254/contended"); len(got) != 1 {
					t.Fatal("duplicate serialization lost unique winner", got)
				}
			})
		}
	}
	for _, operation := range []string{"createIssueLink", "updateIssueLink", "deleteIssueLink"} {
		t.Run("clock/"+operation, func(t *testing.T) {
			w := newIssueWorld(t)
			id := w.newWorkItem(t)
			link := createLink(t, w, "alice", id, "http://169.254.169.254/clock")
			c := clockAt(time.Date(2044, 7, 8, 9, 10, 11, 0, time.UTC))
			tracked := &eventCountingClock{base: c}
			for _, op := range linkRequests(id, link.ID) {
				if op.op != operation {
					continue
				}
				var holder, writer exchange
				w.serializeAtAfter(t, "projects", collaborationAt("alice", http.MethodPatch, "/api/v0/issues/"+id.String(), `{"name":"Clock holder"}`, clock.System{}), collaborationAt("bob", op.method, op.path, op.body, tracked), &holder, &writer, func() {
					assertLinkProjectLock(t, w)
					if tracked.calls.Load() != 0 {
						t.Error("clock read before project lock")
					}
					c.advance(time.Hour)
				})
				w.answered(t, holder, 200)
				w.answered(t, writer, op.want)
				if tracked.calls.Load() != 1 {
					t.Fatal("write must read clock once", tracked.calls.Load())
				}
				for _, table := range []string{"issues", "issue_links", "issue_activities"} {
					condition := "issue_id=$1"
					if table == "issues" {
						condition = "id=$1"
					}
					var n int
					if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM "+table+" WHERE "+condition+" AND updated_at=$2 AND updated_by_id=$3", id, c.Now(), w.ids["bob"]).Scan(&n); err != nil || n != 1 {
						t.Fatal("write lost lock-time audit", table, n, err)
					}
				}
			}
		})
	}
}

func assertLinkProjectLock(t *testing.T, w issueWorld) {
	t.Helper()
	if !pgtest.RowWaits(t, w.pool, "projects", w.web, "FOR SHARE") || pgtest.RowWaits(t, w.pool, "projects", w.web, "FOR KEY SHARE") {
		t.Fatal("holder must own project FOR NO KEY UPDATE")
	}
	if pgtest.RowWaits(t, w.pool, "workspaces", w.acme, "FOR NO KEY UPDATE") {
		t.Fatal("link write holds conflicting workspace lock")
	}
}

func TestIssueLinkWriteRollback(t *testing.T) {
	for _, operation := range []string{"createIssueLink", "updateIssueLink", "deleteIssueLink"} {
		t.Run(operation, func(t *testing.T) {
			w := newIssueWorld(t)
			id := w.newWorkItemBy(t, "alice", w.web, w.fullIssue())
			neighbor := w.newWorkItemBy(t, "bob", w.web, w.fullIssue())
			link := createLink(t, w, "bob", id, "http://169.254.169.254/rollback")
			neighborLink := createLink(t, w, "alice", neighbor, "http://169.254.169.254/neighbor")
			waitUniqueLinkCrawl(t, w, link)
			waitUniqueLinkCrawl(t, w, neighborLink)
			if _, err := w.pool.Exec(pgtest.Soon(t), `UPDATE issue_links SET metadata='{"title":"Existing preview"}' WHERE id=$1`, link.ID); err != nil {
				t.Fatal(err)
			}
			store := issuepg.New(w.pool)
			projects := issueProjects{projects: project.Provide(w.pool).IssueProjects}
			locks := issueapp.NewLocks(projects, store, authorizerOn(w.pool), newWriteGate(testIssueConfig.WriteGate), postgres.NewTxManager(w.pool, 2*time.Second))
			ctx := shared.WithActor(pgtest.Soon(t), shared.Actor{UserID: w.ids["alice"]})
			writePort := map[string]string{"createIssueLink": "InsertLink", "updateIssueLink": "UpdateLink", "deleteIssueLink": "DeleteLink"}[operation]
			for _, port := range []string{"MarkChanged", writePort, "InsertActivities", "Publish"} {
				t.Run(port, func(t *testing.T) {
					before := collaborationState(t, w)
					failure := errors.New("failed after real link stage: " + port)
					wrapped := collaborationFaultStore{Store: store, port: port, failure: failure}
					var err error
					if port == "Publish" {
						var events []shared.Event
						route := issueEventRouteFor(t, w, clock.System{}, &events, &failure, issue.EventTypes(), nil)
						for _, op := range linkRequests(id, link.ID) {
							if op.op == operation {
								route.answerWithin(t, w.contract, op.method, op.path, w.ids["alice"], op.body, 500)
							}
						}
						if len(events) != 1 || issueEventCount(t, w) != 0 {
							t.Fatal("publisher stage not reached or probe job committed", events)
						}
					} else {
						switch operation {
						case "createIssueLink":
							_, err = issueapp.NewCreateIssueLink(issueapp.CreateIssueLinkDeps{Locks: locks, Links: wrapped, Clock: clock.System{}, Events: discardIssueEvents{}}).Execute(ctx, id, domain.NewLink{URL: "http://169.254.169.254/rollback-new"})
						case "updateIssueLink":
							url := "http://169.254.169.254/rollback-updated"
							_, err = issueapp.NewUpdateIssueLink(issueapp.UpdateIssueLinkDeps{Locks: locks, Links: wrapped, Clock: clock.System{}, Events: discardIssueEvents{}}).Execute(ctx, link.ID, domain.LinkPatch{URL: &url})
						case "deleteIssueLink":
							err = issueapp.NewDeleteIssueLink(issueapp.DeleteIssueLinkDeps{Locks: locks, Links: wrapped, Clock: clock.System{}, Events: discardIssueEvents{}}).Execute(ctx, link.ID)
						}
						if err != failure {
							t.Fatal("failure identity lost", err, failure)
						}
					}
					if after := collaborationState(t, w); !reflect.DeepEqual(before, after) {
						for table, rows := range before {
							if rows != after[table] {
								t.Error("failed link write committed business/activity rows", table, rows, after[table])
							}
						}
					}
				})
			}
		})
	}
}
