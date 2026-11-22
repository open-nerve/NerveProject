package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ShownRecentVisit is a recent visit as the list answers it (P3d spec 2.4,
// 2.5): the visit, its target's project now, the project itself for a
// project's visit, and, a work item's, the work item now; nil for a
// project's.
type ShownRecentVisit struct {
	Visit   domain.RecentVisit
	Project ProjectTarget
	Issue   *IssueSummary
}

// RecentTargets are the reads of what recent visits name and of what a
// caller sees of it (spec 2.5, R3): the projects he reads in a workspace
// (Readable), the work items among them he sees (Issues, which applies the
// guest rule) and the projects (Projects), each read once, by ids in id
// order. A list's reads and a recording's, before its locks and under
// them, are the same reads, and recentSight decides on what they read with
// the one predicate.
type RecentTargets struct {
	Readable Readable
	Projects ProjectTargets
	Issues   Issues
}

// recentTarget is the target a recording names: its kind and its id, and
// the 404 of either kind, issue.not_found or project.not_found, which
// answers every target it may not record, the one value (spec 2.7).
type recentTarget struct {
	kind     domain.RecentVisitKind
	id       uuid.UUID
	notFound error
}

// targetOf is the target of kind with id; kind is one of the two
// (domain.CheckRecentVisitKind).
func targetOf(kind domain.RecentVisitKind, id uuid.UUID) recentTarget {
	if kind == domain.RecentVisitProject {
		return recentTarget{kind: kind, id: id, notFound: domain.ErrProjectNotFound}
	}
	return recentTarget{kind: kind, id: id, notFound: domain.ErrIssueNotFound}
}

// recentPlace is where a target is: its workspace and its project, the
// project itself for a project.
type recentPlace struct {
	workspaceID, projectID uuid.UUID
}

// place reads, without a lock, where t is (spec 2.7): a work item's
// workspace and project (Issues.IssuePlaces), a project's workspace and
// itself (Projects.ProjectTargets), archived or not; found is false for
// one not there or deleted. It decides nothing.
func (r RecentTargets) place(ctx context.Context, t recentTarget) (recentPlace, bool, error) {
	ids := []uuid.UUID{t.id}
	if t.kind == domain.RecentVisitProject {
		projects, err := r.Projects.ProjectTargets(ctx, ids)
		p, found := projects[t.id]
		return recentPlace{workspaceID: p.WorkspaceID, projectID: t.id}, found, err
	}
	issues, err := r.Issues.IssuePlaces(ctx, ids)
	p, found := issues[t.id]
	return recentPlace{workspaceID: p.WorkspaceID, projectID: p.ProjectID}, found, err
}

// recentSight is what a caller sees now, in a workspace, of the targets
// some recent visits name (spec 2.5): the projects he reads there, the work
// items among those asked that he sees, as issue answered them, and the
// projects asked that are undeleted, as one read found them.
type recentSight struct {
	grants   map[uuid.UUID]ProjectGrant
	issues   map[uuid.UUID]IssueSummary
	projects map[uuid.UUID]ProjectTarget
}

// sight reads, in the transaction ctx carries, what actor sees in the
// workspace: the projects he reads there (Readable), then those of the work
// items issues he sees by them (Issues.IssueSummaries), then the projects
// projects (ProjectTargets); none, and no read, of a kind with no ids.
func (r RecentTargets) sight(ctx context.Context, actor shared.Actor, workspaceID uuid.UUID, issues, projects []uuid.UUID) (recentSight, error) {
	grants, err := r.Readable.Readable(ctx, actor, workspaceID)
	if err != nil {
		return recentSight{}, err
	}
	s := recentSight{grants: make(map[uuid.UUID]ProjectGrant, len(grants))}
	for _, g := range grants {
		s.grants[g.ProjectID] = g
	}
	summaries := func(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]IssueSummary, error) {
		return r.Issues.IssueSummaries(ctx, actor.UserID, grants, ids)
	}
	if s.issues, err = readSome(ctx, summaries, issues); err != nil {
		return recentSight{}, err
	}
	if s.projects, err = readSome(ctx, r.Projects.ProjectTargets, projects); err != nil {
		return recentSight{}, err
	}
	return s, nil
}

// sees tells whether the caller sees the target of kind with id now (spec
// 2.5), the one predicate of a recording and of the list: a project he
// reads (Readable), archived or not, which a workspace's admin who is not
// its member does not; a work item issue answered him, undeleted, archived
// or not, in a project he reads, that the guest rule does not hide from
// him, in its project's intake or not (R5, R6).
func (s recentSight) sees(kind domain.RecentVisitKind, id uuid.UUID) bool {
	switch kind {
	case domain.RecentVisitProject:
		_, reads := s.grants[id]
		return reads
	case domain.RecentVisitIssue:
		_, seen := s.issues[id]
		return seen
	}
	return false
}

// shows is v answered with what s read of its target now, and whether the
// list lists it (spec 2.5): its target one the caller sees (sees), not a
// work item in its project's intake (R6), and its project read, undeleted.
// An archived target is listed (R5).
func (s recentSight) shows(v domain.RecentVisit) (ShownRecentVisit, bool) {
	p, found := s.projects[v.ProjectID]
	if !found || !s.sees(v.Kind, v.EntityID) {
		return ShownRecentVisit{}, false
	}
	shown := ShownRecentVisit{Visit: v, Project: p}
	if v.Kind == domain.RecentVisitIssue {
		i := s.issues[v.EntityID]
		if i.InTriage {
			return ShownRecentVisit{}, false
		}
		shown.Issue = &i
	}
	return shown, true
}

// visible are the first domain.RecentVisitCap of vs that the list lists
// (shows), in their order: those it does not list are dropped first, so a
// visit kept past them, the cap's transient 21st, takes a place (spec 2.5,
// R8).
func (s recentSight) visible(vs []domain.RecentVisit) []ShownRecentVisit {
	out := make([]ShownRecentVisit, 0, min(len(vs), domain.RecentVisitCap))
	for _, v := range vs {
		if len(out) == domain.RecentVisitCap {
			break
		}
		if shown, listed := s.shows(v); listed {
			out = append(out, shown)
		}
	}
	return out
}
