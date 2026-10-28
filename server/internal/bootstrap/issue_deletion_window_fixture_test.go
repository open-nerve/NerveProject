package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type deletionWindowFixture struct {
	w                             issueWorld
	x                             deletionWindowTargets
	q                             uuid.UUID
	kept                          []uuid.UUID
	memberGone, noProject, qDraft uuid.UUID
	relations                     []uuid.UUID
}

// All business rows are made through their API. Storage is only observed for
// the real worker barrier; NULL creator's deliberate corruption is separate.
func newDeletionWindowFixture(t *testing.T) deletionWindowFixture {
	t.Helper()
	w := newIssueWorld(t)
	f := deletionWindowFixture{w: w}
	f.q = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Kept", "KEEP")
	w.makeMember(t, "bob", f.q, shared.RoleMember)
	for i := 0; i < 3; i++ {
		f.kept = append(f.kept, w.newWorkItemBy(t, "bob", f.q, `{"name":"Window keep","priority":"high","assignee_ids":["`+w.ids["bob"].String()+`"]}`))
	}
	w.newWorkItemIn(t, f.q, `{"name":"Foreign creator"}`)
	f.memberGone = w.newWorkItemBy(t, "bob", w.web, `{"name":"Window gone","assignee_ids":["`+w.ids["bob"].String()+`"]}`)
	f.x = deletionWindowTargets{project: w.web, user: w.ids["bob"], slug: "acme"}
	f.x.issue = w.newWorkItemIn(t, w.web, `{"name":"Target","state_id":"`+w.done.String()+`","description_html":"<p>Original</p>"}`)
	f.x.other = w.newWorkItem(t)
	f.x.archived = w.doneWorkItem(t)
	w.send(t, "alice", "POST", "/api/v0/issues/"+f.x.archived.String()+"/archive", "", 200)
	var item storedIssue
	decodeAnswer(t, w.send(t, "alice", "GET", "/api/v0/issues/"+f.x.issue.String(), "", 200), &item)
	f.x.identifier = fmt.Sprintf("WEB-%d", item.SequenceID)
	f.x.comment = w.newComment(t, "alice", f.x.issue).ID
	w.newComment(t, "bob", f.x.issue)
	w.send(t, "alice", "POST", reactionsOf(f.x.issue), `{"reaction":"128077"}`, 201)
	w.send(t, "alice", "POST", commentReactionsOf(f.x.comment), `{"reaction":"128077"}`, 201)
	link := createLink(t, w, "alice", f.x.issue, "http://169.254.169.254/window")
	waitUniqueLinkCrawl(t, w, link)
	f.x.link = link.ID
	// A different author creates a second version without waiting or storage seeding.
	patchDescription(t, w, "bob", f.x.issue, "<p>Second author</p>", false)
	var versions answeredVersionPage
	decodeAnswer(t, w.send(t, "alice", "GET", versionsOf(f.x.issue), "", 200), &versions)
	if len(versions.Data) != 2 {
		t.Fatal("API description did not preserve both authors' versions")
	}
	f.x.version = versions.Data[1].ID
	f.x.relation = listIn[answeredRelation](t, w.send(t, "alice", "POST", relationsOf(f.x.issue), relating("relates_to", f.x.other), 201))[0].ID
	// Every stored relation type, in both directions, is distinguishable from
	// the one live relation. P's UUID is before Q; endpoints occur both ways.
	for _, kind := range []string{"relates_to", "duplicate", "blocked_by"} {
		for _, reverse := range []bool{false, true} {
			peer := w.newWorkItemIn(t, w.web, `{"name":"Hidden reference"}`)
			from, to := f.kept[0], peer
			if reverse {
				from, to = to, from
			}
			r := listIn[answeredRelation](t, w.send(t, "alice", "POST", relationsOf(from), relating(kind, to), 201))[0]
			f.relations = append(f.relations, r.ID)
		}
	}
	w.send(t, "alice", "POST", relationsOf(f.kept[0]), relating("relates_to", f.kept[1]), 201)
	f.qDraft = createDraftHTTP(t, w, "alice", `{"project_id":"`+f.q.String()+`","payload":{"name":"Kept"}}`).ID
	f.noProject = createDraftHTTP(t, w, "alice", `{"payload":{"name":"Unassigned"}}`).ID
	f.x.draft = createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Old project"}}`).ID
	return f
}

func (f deletionWindowFixture) suspend(t *testing.T, workspace bool) func() {
	t.Helper()
	w := f.w
	holders := []pgx.Tx{holdProjectDeletion(t, w.pool, w.web)}
	projects := []uuid.UUID{w.web}
	if workspace {
		holders = append(holders, holdProjectDeletion(t, w.pool, f.q), holding(t, w.pool, "SELECT id FROM draft_issues WHERE id=$1 FOR UPDATE", f.noProject))
		projects = append(projects, f.q)
		w.send(t, "wanda", http.MethodDelete, "/api/v0/workspaces/acme", "", 204)
	} else {
		w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+w.web.String(), "", 204)
	}
	var jobs []deletionJob
	for _, p := range projects {
		jobs = append(jobs, projectDeletionJob(t, w.pool, p))
	}
	if workspace {
		jobs = append(jobs, parentDeletionJob(t, w.pool, "workspace_drafts", w.acme))
	}
	for i, job := range jobs {
		deletionWorkerBlocked(t, w.pool, holders[i], job, "draft_issues")
	}
	var live bool
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at IS NULL FROM issues WHERE id=$1", f.x.issue).Scan(&live); err != nil || !live {
		t.Fatal("no pending issue window", live, err)
	}
	for _, id := range []uuid.UUID{f.x.draft, f.noProject} {
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at IS NULL FROM draft_issues WHERE id=$1", id).Scan(&live); err != nil || !live {
			t.Fatal("no pending draft window", id, live, err)
		}
	}
	return func() {
		for _, holder := range holders {
			requestRelease(t, holder)
		}
		for _, job := range jobs {
			waitDeletionJob(t, w.pool, job, "completed")
		}
	}
}
