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
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The tombstoned project can retain live issues and relations while its deletion
// is pending. Neither role nor storage direction may reveal that window.
func TestDeleteRelationDeletedEndpointPrecheck(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "stored from live", true: "stored from deleted"}[reverse], func(t *testing.T) {
			w := newIssueWorld(t)
			p := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTHER")
			x, y := w.newWorkItemIn(t, w.web, `{"name":"Live"}`), w.newWorkItemIn(t, p, `{"name":"Deleted"}`)
			from, to := x, y
			if reverse {
				from, to = y, x
			}
			r := listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(from), relating("relates_to", to), 201))[0].ID
			// Bob can read Web but cannot update after becoming its viewer.
			draftExec(t, w.pool, "UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", w.web, w.ids["bob"])
			draftExec(t, w.pool, "UPDATE projects SET guest_view_all_features=true WHERE id=$1", w.web)
			w.send(t, "bob", http.MethodGet, "/api/v0/issues/"+x.String(), "", 200)
			w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+x.String(), `{"name":"Refused"}`, 403)
			draftExec(t, w.pool, "UPDATE projects SET deleted_at=$2 WHERE id=$1", p, time.Now())
			one := poolOfOne(t, w.pool.Config().ConnString())
			realGate := newWriteGate(config.WriteGateConfig{PerProject: 1, MaxQueue: 1, MaxWait: time.Second})
			leave, err := realGate.Enter(pgtest.Soon(t), w.web, p)
			if err != nil {
				t.Fatal(err)
			}
			defer leave()
			resources := &collaborationWriteResources{}
			gate := &relationPrecheckGate{WriteGate: realGate}
			route := newIssueRouteWithGate(t, one, resources, clock.System{}, discardIssueEvents{}, gate)
			before := collaborationState(t, w)
			held := lockedTables(t, w.pool, "projects", "EXCLUSIVE")
			replies := map[string]wholeAnswer{}
			for _, actor := range []string{"alice", "bob"} {
				req, rec := route.send(http.MethodDelete, relationPath(r), w.ids[actor], "")
				w.contract.CheckResponse(t, req, rec.Result())
				if rec.Code != 404 {
					t.Errorf("%s pending reply %d %s, want 404 before occupied gates", actor, rec.Code, rec.Body.String())
				}
				replies[actor] = wholeAnswerOf(rec.Result(), rec.Body.Bytes())
				_, missing := route.send(http.MethodDelete, relationPath(uuid.NewV7()), w.ids[actor], "")
				if got := wholeAnswerOf(missing.Result(), missing.Body.Bytes()); !reflect.DeepEqual(got, replies[actor]) {
					t.Errorf("%s pending != absent: %+v / %+v", actor, replies[actor], got)
				}
			}
			requestRelease(t, held)
			if gate.calls != 0 || resources.transactions != 0 || one.Stat().AcquiredConns() != 0 {
				t.Fatal("refusal acquired resources", resources, one.Stat().AcquiredConns())
			}
			if !reflect.DeepEqual(before, collaborationState(t, w)) {
				t.Fatal("refusal wrote rows")
			}
			// Task completion removes the relation. Compare the complete normalized reply.
			draftExec(t, w.pool, "UPDATE issue_relations SET deleted_at=$2 WHERE id=$1", r, time.Now())
			for actor, want := range replies {
				_, rec := route.send(http.MethodDelete, relationPath(r), w.ids[actor], "")
				if got := wholeAnswerOf(rec.Result(), rec.Body.Bytes()); !reflect.DeepEqual(got, want) {
					t.Errorf("%s pending != completed: %+v / %+v", actor, want, got)
				}
			}
		})
	}
}

// The live deleteIssue composition must use the new ownership port too. The
// pending project's relation keeps its old audit until that project handles it.
func TestLiveIssueDeletionLeavesRelationToEarlierProject(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint(reverse), func(t *testing.T) {
			w := newIssueWorld(t)
			p := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Earlier", "EARLY")
			x, y := w.newWorkItemBy(t, "bob", w.web, `{"name":"Live"}`), w.newWorkItemIn(t, p, `{"name":"Earlier"}`)
			from, to := x, y
			if reverse {
				from, to = y, x
			}
			r := listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(from), relating("relates_to", to), 201))[0].ID
			at := time.Date(2026, 10, 28, 0, 0, 0, 123456000, time.UTC)
			draftExec(t, w.pool, "UPDATE projects SET deleted_at=$2,updated_at=$2,updated_by_id=$3 WHERE id=$1", p, at, w.ids["alice"])
			w.send(t, "bob", http.MethodDelete, "/api/v0/issues/"+x.String(), "", 204)
			if got := w.relationDeletion(t, r, at); got != "undeleted" {
				t.Fatal("live delete stole pending relation", got)
			}
			store := issuepg.New(w.pool)
			projects := issueProjects{projects: project.Provide(w.pool).IssueProjects}
			err := postgres.NewTxManager(w.pool, 2*time.Second).WithinTx(pgtest.Soon(t), func(ctx context.Context) error {
				progress, err := issueapp.NewRelationDeletions(store, projects).ForProject(ctx, issueapp.RowBatch{ProjectID: p, Limit: 2, At: at, By: w.ids["alice"]})
				if err == nil && (progress.Candidates != 1 || progress.Written != 1 || progress.After != r) {
					t.Errorf("progress %+v", progress)
				}
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := w.relationDeletion(t, r, at); got != "deleted by alice at the moment" {
				t.Fatal(got)
			}
		})
	}
}

// Observe entry without replacing the real, already occupied keyed gate.
type relationPrecheckGate struct {
	issueapp.WriteGate
	calls int
}

func (g *relationPrecheckGate) Enter(ctx context.Context, ids ...uuid.UUID) (func(), error) {
	g.calls++
	return g.WriteGate.Enter(ctx, ids...)
}
