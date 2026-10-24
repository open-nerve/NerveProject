package postgresadapter_test

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The issue module's lock of a project (M4 design 3.5, 6.5). It answers the
// undeleted project's workspace, whether it is archived, its default
// assignee and its identifier, an archived one too, none for a deleted one.
// Until the transaction ends the project is held FOR NO KEY UPDATE, which a
// FOR SHARE waits for and a foreign key's FOR KEY SHARE does not, and its
// workspace's row is not locked at all: a FOR UPDATE of it does not wait.
func TestLockIssueProject(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, arch, gone := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme,
		"Gone", "GONE", alice)
	exec(t, pool, "UPDATE projects SET default_assignee_id = $2 WHERE id = $1", web, alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", arch, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)

	for name, tt := range map[string]struct {
		id    uuid.UUID
		want  app.IssueProject
		found bool
	}{
		"Web":  {web, app.IssueProject{WorkspaceID: acme, DefaultAssigneeID: &alice, Identifier: "WEB"}, true},
		"Arch": {arch, app.IssueProject{WorkspaceID: acme, Archived: true, Identifier: "ARCH"}, true},
		"Gone": {gone, app.IssueProject{}, false},
	} {
		err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			p, found, err := s.LockIssueProject(ctx, tt.id)
			if err != nil {
				return err
			}
			if found != tt.found || jsonOf(t, p) != jsonOf(t, tt.want) {
				t.Errorf("LockIssueProject(%s) = %+v, %v; want %+v, %v", name, p, found, tt.want, tt.found)
			}
			if share, keyShare := waits(t, pool, tt.id, "FOR SHARE"), waits(t, pool, tt.id, "FOR KEY SHARE"); share != tt.found || keyShare {
				t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, tt.found)
			}
			if pgtest.RowWaits(t, pool, "workspaces", acme, "FOR UPDATE") {
				t.Errorf("%s while locked: a FOR UPDATE of acme waits; want the workspace's row not locked", name)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// A project's work items are numbered 1, 2, 3, another project's from 1
// again (M4 design 3.7); a deleted project's number is an error, never a
// number.
func TestNextIssueSequence(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops, gone := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice), newProject(t, s, acme,
		"Gone", "GONE", alice)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
	ctx := context.Background()
	var got []int
	for _, p := range []uuid.UUID{web, web, ops, web} {
		n, err := s.NextIssueSequence(ctx, p)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, n)
	}
	if want := []int{1, 2, 1, 3}; !slices.Equal(got, want) {
		t.Errorf("NextIssueSequence() of Web, Web, Ops, Web = %v, want %v", got, want)
	}
	if n, err := s.NextIssueSequence(ctx, gone); err == nil || n != 0 {
		t.Errorf("NextIssueSequence() of a deleted project = %d, %v; want an error", n, err)
	}
}

// A workspace's projects' identifiers for the issue module (M4 design
// 3.7): its undeleted projects', an archived one's too, in id order; not a
// deleted project's, nor another workspace's. Web, drawn first, is the
// archived one, so its row is written after Ops's, and Ops comes first by
// name and by identifier: the id order is neither the rows' order nor that
// of the (workspace, name) or (workspace, identifier) index the planner may
// read, so that a read without its ORDER BY is seen whether the database
// scans the rows or one of those indexes.
func TestProjectIdentifiers(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops, gone := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice), newProject(t, s, acme,
		"Gone", "GONE", alice)
	newProject(t, s, beta, "Web", "BETA", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", web, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
	got, err := s.ProjectIdentifiers(context.Background(), acme)
	want := []app.IssueProjectIdentifier{{ID: web, Identifier: "WEB"}, {ID: ops, Identifier: "OPS"}}
	slices.SortFunc(want, func(a, b app.IssueProjectIdentifier) int { return a.ID.Compare(b.ID) })
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("ProjectIdentifiers(acme) = %+v, %v; want %+v", got, err, want)
	}
}

// Which projects are archived, for the issue module's lists across
// projects (M4 design 3.8): those asked for that are undeleted and
// archived, Arch; not Web, nor Gone, archived and deleted, nor one that is
// not there, nor Old, archived but not asked for.
func TestArchivedIssueProjects(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, arch, gone, old := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme,
		"Gone", "GONE", alice), newProject(t, s, acme, "Old", "OLD", alice)
	for _, p := range []uuid.UUID{arch, gone, old} {
		exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", p, now)
	}
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
	got, err := s.ArchivedIssueProjects(context.Background(), []uuid.UUID{web, arch, gone, uuid.NewV7()})
	if want := map[uuid.UUID]bool{arch: true}; err != nil || !maps.Equal(got, want) {
		t.Errorf("ArchivedIssueProjects() = %v, %v; want Arch (%s) alone", got, err, arch)
	}
}

// The catalogs of a project for the issue module (M4 design 6.5): its
// undeleted states, its triage state too, in id order, with their names,
// their groups and the default; those of the labels asked for that are its
// undeleted labels, with their names, in whatever order (a work item's
// write looks each up by its id); the roles of the accounts asked for who
// are its active members.
// Another project's states, labels and members, a deleted state or label,
// an ended or deleted membership are not in them, nor its own label or
// active member no one asked for (Feature, frank).
func TestIssueCatalogs(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	dave, erin, frank := newAccount(t, pool, "dave@corp.com"), newAccount(t, pool, "erin@corp.com"), newAccount(t, pool, "frank@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	states := seedDefaultStates(t, s, acme, web, alice)
	seedDefaultStates(t, s, acme, ops, alice)
	gone := uuid.NewV7()
	exec(t, pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "group", deleted_at) VALUES ($1, $2, $3, 'Gone', '#000', 'started', $4)`,
		gone, acme, web, now)
	bug, opsBug, oldBug := seedLabel(t, s, acme, web, nil, "Bug", alice), seedLabel(t, s, acme, ops, nil, "Bug", alice), seedLabel(t, s, acme,
		web, nil, "Old", alice)
	exec(t, pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", oldBug, now)
	seedLabel(t, s, acme, web, nil, "Feature", alice)
	ui := seedLabel(t, s, acme, web, nil, "UI", alice)
	seedMember(t, pool, acme, web, alice, 20, true)
	seedMember(t, pool, acme, web, bob, 15, true)
	seedMember(t, pool, acme, web, carol, 5, true)
	seedMember(t, pool, acme, web, dave, 15, false)
	exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", seedMember(t, pool, acme, web, erin, 15, true), now)
	seedMember(t, pool, acme, ops, erin, 20, true)
	seedMember(t, pool, acme, web, frank, 15, true)
	ctx := context.Background()

	got, err := s.IssueStates(ctx, web)
	groups := map[string]string{"Backlog": "backlog", "Todo": "unstarted", "In Progress": "started", "Done": "completed", "Cancelled": "cancelled",
		"Triage": "triage"}
	var want []app.IssueState
	for name, id := range states {
		want = append(want, app.IssueState{ID: id, Name: name, Group: groups[name], Default: name == "Backlog"})
	}
	slices.SortFunc(want, func(a, b app.IssueState) int { return a.ID.Compare(b.ID) })
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("IssueStates(Web) = %+v, %v; want %+v", got, err, want)
	}

	gotLabels, err := s.IssueLabels(ctx, web, []uuid.UUID{opsBug, ui, oldBug, bug, uuid.NewV7()})
	slices.SortFunc(gotLabels, func(a, b app.IssueLabel) int { return a.ID.Compare(b.ID) })
	if want := []app.IssueLabel{{ID: bug, Name: "Bug"}, {ID: ui, Name: "UI"}}; err != nil || !slices.Equal(gotLabels, want) {
		t.Errorf("IssueLabels(Web, …) = %+v, %v; want only Web's Bug and UI", gotLabels, err)
	}
	roles, err := s.ActiveRoles(ctx, web, []uuid.UUID{alice, bob, carol, dave, erin, uuid.NewV7()})
	if want := map[uuid.UUID]shared.Role{alice: shared.RoleAdmin, bob: shared.RoleMember, carol: shared.RoleGuest}; err != nil ||
		!maps.Equal(roles, want) {
		t.Errorf("ActiveRoles(Web, …) = %v, %v; want alice, bob and carol's roles", roles, err)
	}
}

// Mention visibility uses only active project memberships and the same project's
// guest flag. A guest's workspace role must not widen this restriction.
func TestIssueMentionMembers(t *testing.T) {
	s, pool := newStore(t)
	owner := newAccount(t, pool, "owner@corp.com")
	guest := newAccount(t, pool, "guest@corp.com")
	ended := newAccount(t, pool, "ended@corp.com")
	gone := newAccount(t, pool, "gone@corp.com")
	other := newAccount(t, pool, "other@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web := newProject(t, s, acme, "Web", "WEB", owner)
	ops := newProject(t, s, acme, "Ops", "OPS", owner)
	seedMember(t, pool, acme, web, owner, 20, true)
	seedMember(t, pool, acme, web, guest, 5, true)
	seedMember(t, pool, acme, web, ended, 15, false)
	dead := seedMember(t, pool, acme, web, gone, 15, true)
	exec(t, pool, "UPDATE project_members SET deleted_at=$2 WHERE id=$1", dead, now)
	seedMember(t, pool, acme, ops, other, 15, true)
	ids := []uuid.UUID{other, guest, ended, owner, gone}
	for _, all := range []bool{false, true} {
		exec(t, pool, "UPDATE projects SET guest_view_all_features=$2 WHERE id=$1", web, all)
		got, err := s.IssueMentionMembers(context.Background(), web, ids)
		want := []app.IssueMemberVisibility{{ID: owner, Role: shared.RoleAdmin}, {ID: guest, Role: shared.RoleGuest, OwnOnly: !all}}
		if err != nil || !slices.Equal(got, want) {
			t.Fatalf("all=%v: got %+v, %v; want %+v", all, got, err, want)
		}
	}
}
