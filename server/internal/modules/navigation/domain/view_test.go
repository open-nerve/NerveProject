package domain_test

import (
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A view with a project is the project's, one without is the workspace's (spec 2.3).
func TestAViewsKind(t *testing.T) {
	project := uuid.NewV7()
	if k := (domain.View{ProjectID: &project}).Kind(); k != domain.ProjectView {
		t.Fatalf("with a project: %v", k)
	}
	if k := (domain.View{}).Kind(); k != domain.WorkspaceView {
		t.Fatalf("without a project: %v", k)
	}
}

// The grants the decision gives on a view's operations (shared.Grant): a project's admin, member and guest, the
// guest seeing only his own while the project does not show its guests everything, the workspace's admin who is the
// project's member or guest (ProjectAdmin, access decide.go), and the workspace's admin and member on a workspace view.
// The last two are no decision's: they pin that a view's kind decides which admin deletes it.
var (
	projectAdmin    = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleAdmin, ProjectAdmin: true}
	projectMember   = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleMember}
	guestOwnOnly    = shared.Grant{WorkspaceRole: shared.RoleGuest, ProjectRole: shared.RoleGuest, OwnOnly: true}
	guestSeesAll    = shared.Grant{WorkspaceRole: shared.RoleGuest, ProjectRole: shared.RoleGuest}
	adminAsMember   = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleMember, ProjectAdmin: true}
	adminAsGuest    = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleGuest, ProjectAdmin: true, OwnOnly: true}
	adminAsGuestAll = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleGuest, ProjectAdmin: true}
	workspaceAdmin  = shared.Grant{WorkspaceRole: shared.RoleAdmin}
	workspaceMember = shared.Grant{WorkspaceRole: shared.RoleMember}
	adminRoleAlone  = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleMember}
)

// The owner rules (spec 2.2, D29), each case another account and the owner under the same grant: only the owner
// changes a view, whoever administers it; a project view is deleted by its owner or the project's admin
// (Grant.ProjectAdmin, the workspace's admin who is the project's member or guest too), a workspace view by its owner
// or the workspace's admin; an OwnOnly grant does not see the views of others, hidden as if they were not there, and
// sees its own. A workspace view's grant is never OwnOnly. The kind decides which admin deletes: a project admin's
// grant does not delete another's workspace view, nor the workspace admin's role alone another's project view.
func TestTheOwnerRules(t *testing.T) {
	owner, other := uuid.NewV7(), uuid.NewV7()
	project := uuid.NewV7()
	projectView := domain.View{ID: uuid.NewV7(), ProjectID: &project, OwnedByID: owner}
	workspaceView := domain.View{ID: uuid.NewV7(), OwnedByID: owner}
	tests := []struct {
		name           string
		view           domain.View
		grant          shared.Grant
		hidden, delete bool
	}{
		{"project admin", projectView, projectAdmin, false, true},
		{"project member", projectView, projectMember, false, false},
		{"guest seeing his own", projectView, guestOwnOnly, true, false},
		{"guest seeing all", projectView, guestSeesAll, false, false},
		{"workspace admin as the project's member", projectView, adminAsMember, false, true},
		{"workspace admin as the project's guest", projectView, adminAsGuest, true, true},
		{"workspace admin as the project's guest seeing all", projectView, adminAsGuestAll, false, true},
		{"the workspace admin's role alone on a project view", projectView, adminRoleAlone, false, false},
		{"workspace admin", workspaceView, workspaceAdmin, false, true},
		{"workspace member", workspaceView, workspaceMember, false, false},
		{"a project admin's grant on a workspace view", workspaceView, projectAdmin, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.view.HiddenFrom(other, tt.grant); got != tt.hidden {
				t.Errorf("another's view hidden = %v; want %v", got, tt.hidden)
			}
			if tt.view.UpdatableBy(other) {
				t.Error("another updates the view")
			}
			if got := tt.view.DeletableBy(other, tt.grant); got != tt.delete {
				t.Errorf("another deletes = %v; want %v", got, tt.delete)
			}
			if tt.view.HiddenFrom(owner, tt.grant) || !tt.view.UpdatableBy(owner) || !tt.view.DeletableBy(owner, tt.grant) {
				t.Error("the owner does not see, update and delete his view")
			}
		})
	}
}

// The module's two codes (spec 2.4): every 404 on /views/{view_id} is navigation.view_not_found, and a write on a
// project's views while its toggle issue_views_view is off is 409 navigation.views_disabled. And the 404s of the
// routes on a project's and a workspace's views (spec 2.2), the codes their modules answer.
func TestTheModulesErrors(t *testing.T) {
	for _, tt := range []struct {
		err    *shared.Error
		status int
		code   string
	}{
		{domain.ErrViewNotFound, 404, "navigation.view_not_found"},
		{domain.ErrViewsDisabled, 409, "navigation.views_disabled"},
		{domain.ErrProjectNotFound, 404, "project.not_found"},
		{domain.ErrWorkspaceNotFound, 404, "workspace.not_found"},
	} {
		if tt.err.ProblemStatus() != tt.status || tt.err.ProblemCode() != tt.code || tt.err.Detail == "" {
			t.Errorf("%+v; want %d %s with a detail", tt.err, tt.status, tt.code)
		}
	}
	if errors.Is(domain.ErrViewNotFound, shared.ErrNotVisible) {
		t.Error("view_not_found is the uncoded ErrNotVisible; the use case answers the code")
	}
}
