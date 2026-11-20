package domain_test

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A view's kind chooses its actions (spec 2.2, 2.5): a project's view's are view.*, a workspace's are
// workspace_view.*, each verb its own.
func TestAViewsKindChoosesItsActions(t *testing.T) {
	for _, tt := range []struct {
		kind domain.Kind
		verb domain.Verb
		want shared.Action
	}{
		{domain.ProjectView, domain.Read, "view.read"},
		{domain.ProjectView, domain.Create, "view.create"},
		{domain.ProjectView, domain.Update, "view.update"},
		{domain.ProjectView, domain.Delete, "view.delete"},
		{domain.WorkspaceView, domain.Read, "workspace_view.read"},
		{domain.WorkspaceView, domain.Create, "workspace_view.create"},
		{domain.WorkspaceView, domain.Update, "workspace_view.update"},
		{domain.WorkspaceView, domain.Delete, "workspace_view.delete"},
	} {
		if got := domain.ActionOf(tt.kind, tt.verb); got != tt.want {
			t.Errorf("ActionOf(%v, %v) = %q; want %q", tt.kind, tt.verb, got, tt.want)
		}
	}
}
