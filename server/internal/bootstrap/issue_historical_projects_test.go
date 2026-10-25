package bootstrap

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Conversion preserves every field, argument and context, and never turns a
// directory failure into an empty successful result.
func TestHistoricalProjectsConversion(t *testing.T) {
	workspaceID, actorID := uuid.NewV7(), uuid.NewV7()
	first, second := uuid.NewV7(), uuid.NewV7()
	f := &fakeIssueProjects{historical: []project.HistoricalIssueProject{
		{ID: first, Role: shared.RoleGuest, OwnOnly: true},
		{ID: second, Role: shared.RoleMember},
	}}
	var p issue.ProjectPorts = issueProjects{projects: f}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	got, err := p.Historical(ctx, workspaceID, actorID)
	want := []issue.VisibleProject{{ID: first, Role: shared.RoleGuest, OwnOnly: true}, {ID: second, Role: shared.RoleMember}}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("historical conversion: got %+v, %v; want %+v", got, err, want)
	}
	if f.historicalContext != ctx || f.historicalWorkspace != workspaceID || f.historicalActor != actorID {
		t.Fatalf("historical arguments/context changed: %+v", f)
	}
	failure := errors.New("historical directory unavailable")
	f.err = failure
	got, err = p.Historical(ctx, workspaceID, actorID)
	if !errors.Is(err, failure) || got != nil {
		t.Fatalf("historical failure: got %+v, %v; want nil and original error", got, err)
	}
}

// The real provider and converter keep a workspace admin's project guest scope,
// while the same actor's project member scope remains unrestricted.
func TestHistoricalProjectsComposition(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	other := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTHER")
	w.makeMember(t, "wanda", other, shared.RoleAdmin)
	// Membership creation follows workspace role policy. Historical rows can
	// retain a narrower project role after the workspace role changes.
	if _, err := w.pool.Exec(context.Background(), "UPDATE project_members SET role=CASE WHEN project_id=$1 THEN 5 ELSE 15 END WHERE member_id=$2",
		w.web, w.ids["wanda"]); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(context.Background(), "UPDATE projects SET guest_view_all_features=false WHERE id=ANY($1::uuid[])",
		[]uuid.UUID{w.web, other}); err != nil {
		t.Fatal(err)
	}
	var p issue.ProjectPorts = issueProjects{projects: project.Provide(w.pool).IssueProjects}
	got, err := p.Historical(context.Background(), w.acme, w.ids["wanda"])
	want := []issue.VisibleProject{{ID: w.web, Role: shared.RoleGuest, OwnOnly: true}, {ID: other, Role: shared.RoleMember}}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("admin guest project scope: got %+v, %v; want %+v", got, err, want)
	}
}
