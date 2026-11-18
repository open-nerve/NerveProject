package bootstrap

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// issueRefsCall is what fakeIssueRefs was asked.
type issueRefsCall struct {
	reader uuid.UUID
	grants []issue.ProjectGrant
	ids    []uuid.UUID
}

// fakeIssueRefs answers the refs it holds and records what it was asked.
type fakeIssueRefs struct {
	refs  map[uuid.UUID]issue.IssueRef
	err   error
	asked []issueRefsCall
}

func (f *fakeIssueRefs) IssueRefs(_ context.Context, reader uuid.UUID, grants []issue.ProjectGrant, ids []uuid.UUID) (map[uuid.UUID]issue.IssueRef, error) {
	f.asked = append(f.asked, issueRefsCall{reader: reader, grants: grants, ids: ids})
	if f.err != nil {
		return nil, f.err
	}
	return f.refs, nil
}

// fakeNotificationIdentifiers answers the identifiers it holds of the
// projects it is asked and records what it was asked.
type fakeNotificationIdentifiers struct {
	identifiers map[uuid.UUID]string
	err         error
	asked       [][]uuid.UUID
}

func (f *fakeNotificationIdentifiers) NotificationIdentifiers(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	f.asked = append(f.asked, ids)
	if f.err != nil {
		return nil, f.err
	}
	out := map[uuid.UUID]string{}
	for _, id := range ids {
		if identifier, ok := f.identifiers[id]; ok {
			out[id] = identifier
		}
	}
	return out, nil
}

// notificationIssueRefs names a page's other ends in one IssueRefs and one
// read of their projects' identifiers (M7 P1c spec 2.6; pre-flight re-84
// m6): IssueRefs is asked once, with the reader, the grants and the ids as
// they came; the identifiers once, of the projects of the refs it answered,
// each once, in id order; and each ref is answered with its project's
// identifier, but one whose project has none, deleted since. No refs read
// no identifiers; a failure of either read is its error, and no answer.
func TestNotificationIssueRefsReadOnceEach(t *testing.T) {
	gail, web, ops, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	a, b, c, d := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	grants := []issue.ProjectGrant{{ProjectID: web, OwnOnly: true}, {ProjectID: ops}, {ProjectID: gone}}
	ids := []uuid.UUID{d, c, b, a, uuid.NewV7()}
	issues := &fakeIssueRefs{refs: map[uuid.UUID]issue.IssueRef{
		a: {ID: a, ProjectID: ops, SequenceID: 4, Name: "A"}, b: {ID: b, ProjectID: web, SequenceID: 7, Name: "B"},
		c: {ID: c, ProjectID: ops, SequenceID: 9, Name: "C"}, d: {ID: d, ProjectID: gone, SequenceID: 1, Name: "D"},
	}}
	identifiers := &fakeNotificationIdentifiers{identifiers: map[uuid.UUID]string{web: "WEB", ops: "OPS"}}
	refs := notificationIssueRefs{issues: issues, projects: identifiers}
	ctx := context.Background()
	got, err := refs.IssueRefs(ctx, gail, grants, ids)
	want := map[uuid.UUID]namedIssue{a: {ID: a, SequenceID: 4, Identifier: "OPS", Name: "A"}, b: {ID: b, SequenceID: 7, Identifier: "WEB", Name: "B"},
		c: {ID: c, SequenceID: 9, Identifier: "OPS", Name: "C"}}
	if err != nil || !maps.Equal(got, want) {
		t.Errorf("IssueRefs() = %+v, %v; want %+v", got, err, want)
	}
	if len(issues.asked) != 1 || issues.asked[0].reader != gail || !slices.Equal(issues.asked[0].grants, grants) || !slices.Equal(issues.asked[0].ids, ids) {
		t.Errorf("IssueRefs was asked %+v; want gail's grants and ids once", issues.asked)
	}
	if wantAsked := slices.SortedFunc(slices.Values([]uuid.UUID{web, ops, gone}), uuid.UUID.Compare); len(identifiers.asked) != 1 ||
		!slices.Equal(identifiers.asked[0], wantAsked) {
		t.Errorf("the identifiers were asked %v; want %v once", identifiers.asked, wantAsked)
	}

	issues.refs, issues.asked, identifiers.asked = map[uuid.UUID]issue.IssueRef{}, nil, nil
	if got, err := refs.IssueRefs(ctx, gail, grants, ids); err != nil || len(got) != 0 {
		t.Errorf("IssueRefs() of no refs = %+v, %v; want none", got, err)
	}
	if len(issues.asked) != 1 || len(identifiers.asked) != 0 {
		t.Errorf("of no refs, IssueRefs was asked %d times and the identifiers %v; want once and never", len(issues.asked), identifiers.asked)
	}

	failure := errors.New("connection reset")
	issues.err = failure
	if got, err := refs.IssueRefs(ctx, gail, grants, ids); !errors.Is(err, failure) || got != nil || len(identifiers.asked) != 0 {
		t.Errorf("IssueRefs() = %+v, %v, identifiers asked %v; want %v, none and no read", got, err, identifiers.asked, failure)
	}
	issues.err, issues.refs = nil, map[uuid.UUID]issue.IssueRef{a: {ID: a, ProjectID: ops, SequenceID: 4, Name: "A"}}
	identifiers.err = failure
	if got, err := refs.IssueRefs(ctx, gail, grants, ids); !errors.Is(err, failure) || got != nil {
		t.Errorf("IssueRefs() = %+v, %v; want %v and none", got, err, failure)
	}
}

// The other ends a page of gail's notifications in acme names, as the
// notification module names them to her (M7 P1c spec 2.6, R4), through
// issue.Provide's IssueRefs and project.Provide's NotificationIdentifiers
// composed, with the grants access's Readable answers her: she is acme's
// and beta's member, a guest of acme's Web, which does not show its guests
// every work item, a member of acme's archived Old and of beta's Site, and
// of acme's Secret not at all. Named are Web's work items she created, an
// archived one too, each with WEB, and Old's that alice created, with OLD;
// not Web's that alice created, nor one whose creator is not known, nor
// her deleted one, nor Secret's, nor Site's, which is beta's, nor an id of
// no work item.
func TestNotificationIssueRefsNameWhatTheReaderSees(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	ctx := context.Background()
	gail, alice := uuid.NewV7(), uuid.NewV7()
	acme, beta := uuid.NewV7(), uuid.NewV7()
	web, old, secret, site := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	states := map[uuid.UUID]uuid.UUID{}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	for id, name := range map[uuid.UUID]string{gail: "gail", alice: "alice"} {
		exec("INSERT INTO users (id, email, password, display_name) VALUES ($1, $2, 'x', $3)", id, name+"@example.com", name)
	}
	for _, w := range []struct {
		id   uuid.UUID
		slug string
	}{{acme, "acme"}, {beta, "beta"}} {
		exec("INSERT INTO workspaces (id, name, slug) VALUES ($1, $2, $2)", w.id, w.slug)
		exec("INSERT INTO workspace_members (id, workspace_id, member_id, role) VALUES ($1, $2, $3, $4), ($5, $2, $6, $7)",
			uuid.NewV7(), w.id, alice, int(shared.RoleAdmin), uuid.NewV7(), gail, int(shared.RoleMember))
	}
	for _, p := range []struct {
		id, workspace uuid.UUID
		identifier    string
		gail          shared.Role
	}{{web, acme, "WEB", shared.RoleGuest}, {old, acme, "OLD", shared.RoleMember}, {secret, acme, "SEC", 0}, {site, beta, "SITE", shared.RoleMember}} {
		states[p.id] = uuid.NewV7()
		exec("INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, $3, $3)", p.id, p.workspace, p.identifier)
		exec(`INSERT INTO states (id, workspace_id, project_id, name, color, "default") VALUES ($1, $2, $3, 'Backlog', '#60646C', true)`,
			states[p.id], p.workspace, p.id)
		exec("INSERT INTO project_members (id, workspace_id, project_id, member_id, role) VALUES ($1, $2, $3, $4, $5)",
			uuid.NewV7(), p.workspace, p.id, alice, int(shared.RoleAdmin))
		if p.gail != 0 {
			exec("INSERT INTO project_members (id, workspace_id, project_id, member_id, role) VALUES ($1, $2, $3, $4, $5)",
				uuid.NewV7(), p.workspace, p.id, gail, int(p.gail))
		}
	}
	exec("UPDATE projects SET archived_at = now() WHERE id = $1", old)
	item := func(p, workspace uuid.UUID, sequence int, name string, creator *uuid.UUID) uuid.UUID {
		t.Helper()
		id := uuid.NewV7()
		exec(`INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id, created_by_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			id, workspace, p, states[p], name, sequence, creator)
		return id
	}
	hers, shelved, alices, unknown, deleted := item(web, acme, 1, "Hers", &gail), item(web, acme, 2, "Shelved", &gail),
		item(web, acme, 3, "Alice's", &alice), item(web, acme, 4, "Unknown", nil), item(web, acme, 5, "Deleted", &gail)
	olds, secrets, far := item(old, acme, 1, "Old's", &alice), item(secret, acme, 1, "Secret's", &gail), item(site, beta, 1, "Far", &gail)
	exec("UPDATE issues SET archived_at = now() WHERE id = $1", shelved)
	exec("UPDATE issues SET deleted_at = now() WHERE id = $1", deleted)

	wp, pp := workspace.Provide(pool, nil), project.Provide(pool, nil)
	readable := access.NewReadableProjects(access.ReadableDeps{WorkspaceRoles: wp.WorkspaceRoles,
		Projects: accessMemberProjects{projects: pp.MemberProjectAccess}})
	answered, err := readable.Readable(ctx, shared.Actor{UserID: gail}, acme)
	ownOnly := map[uuid.UUID]bool{}
	grants := make([]issue.ProjectGrant, len(answered))
	for i, g := range answered {
		ownOnly[g.ProjectID] = g.OwnOnly
		grants[i] = issue.ProjectGrant(g)
	}
	if want := map[uuid.UUID]bool{web: true, old: false}; err != nil || len(answered) != len(want) || !maps.Equal(ownOnly, want) {
		t.Fatalf("Readable(gail, acme) = %+v, %v; want Web own-only and Old", answered, err)
	}
	refs := notificationIssueRefs{issues: issue.Provide(pool, nil).IssueRefs, projects: pp.NotificationIdentifiers}
	got, err := refs.IssueRefs(ctx, gail, grants, []uuid.UUID{far, secrets, olds, deleted, unknown, alices, shelved, hers, uuid.NewV7()})
	want := map[uuid.UUID]namedIssue{hers: {ID: hers, SequenceID: 1, Identifier: "WEB", Name: "Hers"},
		shelved: {ID: shelved, SequenceID: 2, Identifier: "WEB", Name: "Shelved"}, olds: {ID: olds, SequenceID: 1, Identifier: "OLD", Name: "Old's"}}
	if err != nil || !maps.Equal(got, want) {
		t.Errorf("IssueRefs(gail, acme) = %+v, %v; want %+v", got, err, want)
	}
}
