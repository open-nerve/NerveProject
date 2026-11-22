package app_test

import (
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// recordCalls are a recording's calls (spec 2.7, 2.8): place, the reads of its target's place, and see, of what she
// sees of it; before the locks, place, recent_visit.write's decision on the workspace and see; then, in the
// recording's transaction, the workspace's FOR SHARE and the decision again, the project's FOR SHARE, place and see
// again under them, the advisory lock of her visits in the workspace, its last lock, the moment, read once after it,
// the write and the trim.
func recordCalls(place, see []string, kind domain.RecentVisitKind, target string) []string {
	calls := append(append([]string(nil), place...), "authorize alice recent_visit.write acme/-")
	calls = append(calls, see...)
	calls = append(calls, "begin write", "share workspace acme", "authorize alice recent_visit.write acme/-", "share project web")
	calls = append(calls, place...)
	calls = append(calls, see...)
	return append(calls, "lock visits alice acme", "now", "record "+string(kind)+" "+target+" by alice", "trim alice acme 20", "end write")
}

// A visit of each kind is recorded, 204 (spec 2.4, 2.5): hers, in the target's workspace and project (the project
// itself for a project's), of the target, a new id, visited at the one moment read after the last lock, written by
// her; then her visits in the workspace past the cap are trimmed. Each takes its locks in spec 2.8's order and reads
// its target again under them: the workspace, the project, the advisory lock last, then the clock, the write, the
// trim.
func TestRecordARecentVisitOfEachKind(t *testing.T) {
	for _, tt := range []struct {
		name       string
		kind       domain.RecentVisitKind
		id         uuid.UUID
		place, see []string
	}{
		{"a work item", domain.RecentVisitIssue, task, []string{"issue places task"}, []string{"readable alice acme", "issue summaries task"}},
		{"a project", domain.RecentVisitProject, web, []string{"project targets web"}, []string{"readable alice acme"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := newRecents()
			if err := r.recordVisit().Execute(as(t), tt.kind, tt.id); err != nil {
				t.Fatal(err)
			}
			if len(r.recorded) != 1 {
				t.Fatalf("recorded %+v; want one", r.recorded)
			}
			v := r.recorded[0].visit
			want := domain.RecentVisit{ID: v.ID, WorkspaceID: acme, ProjectID: web, UserID: alice, Kind: tt.kind, EntityID: tt.id, VisitedAt: moment}
			if v.ID == (uuid.UUID{}) || v != want || r.recorded[0].by != alice {
				t.Errorf("recorded %+v by %s\nwant %+v by alice", v, r.name(r.recorded[0].by), want)
			}
			if !reflect.DeepEqual(r.trims, []trimmed{{alice, acme, domain.RecentVisitCap}}) {
				t.Errorf("trimmed %+v; want alice's in acme past 20", r.trims)
			}
			wantCalls(t, r.world, recordCalls(tt.place, tt.see, tt.kind, r.name(tt.id))...)
		})
	}
}

// A write or a trim that fails fails the recording, with its error, so that its transaction rolls back: a trim's
// error is not passed over for a write that went through.
func TestARecordingFailsWithItsWritesError(t *testing.T) {
	failure := errors.New("the store failed")
	for _, tt := range []struct {
		name  string
		fail  func(*recents)
		trims int
	}{
		{"the write", func(r *recents) { r.recordErr = failure }, 0},
		{"the trim", func(r *recents) { r.trimErr = failure }, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := newRecents()
			tt.fail(r)
			if err := r.recordVisit().Execute(as(t), domain.RecentVisitProject, web); !errors.Is(err, failure) {
				t.Fatalf("recording = %v; want %v", err, failure)
			}
			if len(r.trims) != tt.trims {
				t.Errorf("trims %+v after the failing %s; want %d", r.trims, tt.name, tt.trims)
			}
		})
	}
}

// What she may record is what she sees now (spec 2.5, R5, R6), the one predicate of the list: an archived project and
// a work item of one, a work item in its project's intake, her own work item where she sees only her own are recorded.
// Every other target is its own 404, issue.not_found or project.not_found, the one value, before any lock: another's
// work item where she sees only her own, as a project's guest or as the workspace's guest (404, not 403); a project
// she does not read, the workspace's admin who is none of its members, and its work items; a target not there or
// deleted; one of a workspace she is no member of, never workspace.not_found.
func TestRecordingARecentVisitSheSees(t *testing.T) {
	ownOnly := func(r *recents) { r.grants[acme] = []app.ProjectGrant{{ProjectID: web, OwnOnly: true}} }
	for _, tt := range []struct {
		name    string
		kind    domain.RecentVisitKind
		id      uuid.UUID
		prepare func(r *recents)
		want    error
	}{
		{"an archived project", domain.RecentVisitProject, attic, nil, nil},
		{"a work item of an archived project", domain.RecentVisitIssue, relic, nil, nil},
		{"a work item in its project's intake", domain.RecentVisitIssue, triaged, nil, nil},
		{"her own work item where she sees only her own", domain.RecentVisitIssue, chore, ownOnly, nil},
		{"another's work item where she sees only her own", domain.RecentVisitIssue, task, ownOnly, domain.ErrIssueNotFound},
		{"another's work item, as the workspace's guest who sees only her own", domain.RecentVisitIssue, task, func(r *recents) {
			ownOnly(r)
			r.decide = func(shared.Action, shared.Target, bool) (shared.Grant, error) { return workspaceGuest, nil }
		}, domain.ErrIssueNotFound},
		{"a project she does not read, the workspace's admin", domain.RecentVisitProject, lobby, func(r *recents) {
			r.decide = func(shared.Action, shared.Target, bool) (shared.Grant, error) { return workspaceAdmin, nil }
		}, domain.ErrProjectNotFound},
		{"a work item of a project she no longer reads", domain.RecentVisitIssue, toil, func(r *recents) {
			r.grants[acme] = []app.ProjectGrant{{ProjectID: web}}
		}, domain.ErrIssueNotFound},
		{"a work item not there", domain.RecentVisitIssue, uuid.NewV7(), nil, domain.ErrIssueNotFound},
		{"a project not there", domain.RecentVisitProject, uuid.NewV7(), nil, domain.ErrProjectNotFound},
		{"a deleted work item", domain.RecentVisitIssue, task, func(r *recents) { delete(r.issues, task) }, domain.ErrIssueNotFound},
		{"a deleted project", domain.RecentVisitProject, ops, func(r *recents) {
			delete(r.projectTargets, ops)
			r.grants[acme] = []app.ProjectGrant{{ProjectID: web}}
		}, domain.ErrProjectNotFound},
		{"a work item of a workspace she is no member of", domain.RecentVisitIssue, far, nil, domain.ErrIssueNotFound},
		{"a project of a workspace she is no member of", domain.RecentVisitProject, site, nil, domain.ErrProjectNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := newRecents()
			if tt.prepare != nil {
				tt.prepare(r)
			}
			err := r.recordVisit().Execute(as(t), tt.kind, tt.id)
			wantSame(t, err, tt.want)
			if tt.want == nil {
				if len(r.recorded) != 1 || r.recorded[0].visit.EntityID != tt.id {
					t.Errorf("recorded %+v; want her visit of %s", r.recorded, r.name(tt.id))
				}
				return
			}
			wroteNoVisit(t, r)
			for _, call := range r.calls {
				if call == "begin write" {
					t.Errorf("began the recording's transaction; want its check to refuse before any lock")
				}
			}
		})
	}
}

// The kind is refused first, before any read (spec 2.7): one not of the two, also of a target not there and as the
// workspace's guest, is a 422 at entity_type, and nothing is read; no actor is unauthenticated, before it.
func TestTheRefusalOrderOfARecording(t *testing.T) {
	for _, tt := range []struct {
		name string
		kind domain.RecentVisitKind
		id   uuid.UUID
	}{
		{"a page", "page", task},
		{"a work item's kind in another case", "Issue", task},
		{"a page not there", "page", uuid.NewV7()},
	} {
		r := newRecents()
		r.decide = func(shared.Action, shared.Target, bool) (shared.Grant, error) { return workspaceGuest, nil }
		err := r.recordVisit().Execute(as(t), tt.kind, tt.id)
		if fields := fieldsOf(t, err); len(fields) != 1 || fields[0].Field != "entity_type" || fields[0].Code != shared.FieldInvalidFormat {
			t.Errorf("%s: fields %+v; want entity_type's invalid_format alone", tt.name, fields)
		}
		wroteNoVisit(t, r)
		wantCalls(t, r.world)
	}
	r := newRecents()
	if err := r.recordVisit().Execute(t.Context(), "page", task); err == nil || len(r.calls) != 0 {
		t.Errorf("no actor: %v after %q; want refused before anything", err, r.calls)
	}
}
