package app_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The projects and the work items of the recent visits' tests (P3d spec 2.5), beside the favorites' web, ops and
// beta's site: attic, an archived project of acme alice reads, and lobby, a project of acme she is none of the members
// of. The work items: task, bob's, and chore, alice's, of web; triaged, bob's, in web's intake; relic, bob's, of attic;
// toil, bob's, of ops; and far, bob's, of beta's site.
var (
	attic, lobby                           = uuid.NewV7(), uuid.NewV7()
	task, chore, triaged, relic, toil, far = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
)

// recentIssue is a work item as the fake Issues holds it: where it is, its creator and what it is answered with.
type recentIssue struct {
	place   app.IssuePlace
	creator uuid.UUID
	summary app.IssueSummary
}

// recordedVisit is a visit RecordRecentVisit stored and its writer; trimmed, a trim TrimRecentVisits ran.
type (
	recordedVisit struct {
		visit domain.RecentVisit
		by    uuid.UUID
	}
	trimmed struct {
		user, workspace uuid.UUID
		keep            int
	}
)

// recents is the shelf with the fakes of the recent visits' ports: Issues, which mirrors issue's NavigationIssues (an
// undeleted work item's place; those in a project among the grants that the guest rule does not hide), the
// RecentVisitReader, the RecentVisitLocker and the RecentVisitWriter, each logging its calls in the world's log; the
// shelf's Readable and ProjectTargets are the projects she reads and the undeleted projects. The shelf's meanwhile runs
// once when the workspace's lock is granted.
type recents struct {
	*shelf
	// issues are the undeleted work items, of any workspace; visits, what RecentVisits answers.
	issues map[uuid.UUID]recentIssue
	visits []domain.RecentVisit
	// recorded are the visits RecordRecentVisit stored; trims, the trims run.
	recorded []recordedVisit
	trims    []trimmed
}

// newRecents is newShelf with attic, archived, and lobby, projects of acme, and the work items: alice reads web, ops and
// attic, each fully, but not lobby; she is no member of beta.
func newRecents() *recents {
	s := newShelf()
	for id, name := range map[uuid.UUID]string{attic: "attic", lobby: "lobby", task: "task", chore: "chore", triaged: "triaged", relic: "relic",
		toil: "toil", far: "far"} {
		s.names[id] = name
	}
	s.projects[attic] = app.ProjectFacts{WorkspaceID: acme}
	s.projects[lobby] = app.ProjectFacts{WorkspaceID: acme}
	s.projectTargets[web] = app.ProjectTarget{WorkspaceID: acme, Name: "Web", Identifier: "WEB", LogoProps: icon}
	s.projectTargets[ops] = app.ProjectTarget{WorkspaceID: acme, Name: "Ops", Identifier: "OPS"}
	s.projectTargets[attic] = app.ProjectTarget{WorkspaceID: acme, Name: "Attic", Identifier: "ATTIC", Archived: true}
	s.projectTargets[lobby] = app.ProjectTarget{WorkspaceID: acme, Name: "Lobby", Identifier: "LOBBY"}
	s.projectTargets[site] = app.ProjectTarget{WorkspaceID: beta, Name: "Site", Identifier: "SITE"}
	s.grants[acme] = []app.ProjectGrant{{ProjectID: web}, {ProjectID: ops}, {ProjectID: attic}}
	s.decide = func(_ shared.Action, t shared.Target, _ bool) (shared.Grant, error) {
		if t.WorkspaceID == beta {
			return shared.Grant{}, shared.ErrNotVisible
		}
		return member, nil
	}
	r := &recents{shelf: s, issues: map[uuid.UUID]recentIssue{}}
	for n, i := range []struct {
		id, project, workspace, creator uuid.UUID
		inTriage                        bool
	}{{task, web, acme, bob, false}, {chore, web, acme, alice, false}, {triaged, web, acme, bob, true}, {relic, attic, acme, bob, false},
		{toil, ops, acme, bob, false}, {far, site, beta, bob, false}} {
		r.issues[i.id] = recentIssue{place: app.IssuePlace{WorkspaceID: i.workspace, ProjectID: i.project}, creator: i.creator,
			summary: app.IssueSummary{ID: i.id, ProjectID: i.project, SequenceID: n + 1, Name: s.names[i.id], StateID: i.project, Priority: "high",
				AssigneeIDs: []uuid.UUID{bob}, InTriage: i.inTriage}}
	}
	return r
}

func (r *recents) IssuePlaces(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]app.IssuePlace, error) {
	r.log("issue places %s", r.idsOf(ids))
	out := map[uuid.UUID]app.IssuePlace{}
	for _, id := range ids {
		if i, found := r.issues[id]; found {
			out[id] = i.place
		}
	}
	return out, nil
}

// IssueSummaries mirrors issue's NavigationIssues: the undeleted work items among ids in a project among grants that the
// guest rule, by the grant there, does not hide from reader.
func (r *recents) IssueSummaries(_ context.Context, reader uuid.UUID, grants []app.ProjectGrant, ids []uuid.UUID) (map[uuid.UUID]app.IssueSummary, error) {
	r.log("issue summaries %s", r.idsOf(ids))
	out := map[uuid.UUID]app.IssueSummary{}
	for _, id := range ids {
		i, found := r.issues[id]
		at := slices.IndexFunc(grants, func(g app.ProjectGrant) bool { return g.ProjectID == i.place.ProjectID })
		if found && at >= 0 && (!grants[at].OwnOnly || i.creator == reader) {
			out[id] = i.summary
		}
	}
	return out, nil
}

func (r *recents) RecentVisits(_ context.Context, user, workspace uuid.UUID, kind *domain.RecentVisitKind) ([]domain.RecentVisit, error) {
	of := "any"
	if kind != nil {
		of = string(*kind)
	}
	r.log("visits %s %s %s", r.name(user), r.name(workspace), of)
	return r.visits, nil
}

func (r *recents) LockRecentVisits(_ context.Context, user, workspace uuid.UUID) error {
	r.log("lock visits %s %s", r.name(user), r.name(workspace))
	return nil
}

func (r *recents) RecordRecentVisit(_ context.Context, v domain.RecentVisit, by uuid.UUID) error {
	r.log("record %s %s by %s", v.Kind, r.name(v.EntityID), r.name(by))
	r.recorded = append(r.recorded, recordedVisit{v, by})
	return nil
}

func (r *recents) TrimRecentVisits(_ context.Context, user, workspace uuid.UUID, keep int) (int, error) {
	r.log("trim %s %s %d", r.name(user), r.name(workspace), keep)
	r.trims = append(r.trims, trimmed{user, workspace, keep})
	return 0, nil
}

// targets are the recents' reads of targets.
func (r *recents) targets() app.RecentTargets {
	return app.RecentTargets{Readable: r, Projects: r, Issues: r}
}

// recordVisit is the recents' recording, in its "write" transactions; listVisits, its list, in its "read" ones.
func (r *recents) recordVisit() *app.RecordRecentVisit {
	return app.NewRecordRecentVisit(app.RecordRecentVisitDeps{Locks: app.NewRecentVisitLocks(r, r, r.targets(), r, r, tx{r.world, "write"}),
		Visits: r, Clock: r})
}

func (r *recents) listVisits() *app.ListRecentVisits {
	return app.NewListRecentVisits(app.ListRecentVisitsDeps{Workspaces: r, Authorizer: r, Visits: r, Targets: r.targets(), Tx: tx{r.world, "read"}})
}

// visitOf is alice's stored visit in acme of the target of kind with id, in project, visited n minutes before moment.
func visitOf(kind domain.RecentVisitKind, id, project uuid.UUID, n int) domain.RecentVisit {
	return domain.RecentVisit{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: project, UserID: alice, Kind: kind, EntityID: id,
		VisitedAt: moment.Add(-time.Duration(n) * time.Minute)}
}

// wroteNoVisit fails t unless the recents recorded and trimmed nothing, took no advisory lock and read no clock.
func wroteNoVisit(t *testing.T, r *recents) {
	t.Helper()
	if len(r.recorded)+len(r.trims) != 0 {
		t.Errorf("wrote: recorded %+v, trimmed %+v; want nothing", r.recorded, r.trims)
	}
	for _, call := range r.calls {
		if call == "now" || strings.HasPrefix(call, "lock visits") {
			t.Errorf("called %q; want no advisory lock and no clock", call)
		}
	}
}
