package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakePlanningProjects is project's PlanningProjects answering what it holds,
// recording what it was asked.
type fakePlanningProjects struct {
	facts project.PlanningFacts
	found bool
	roles map[uuid.UUID]shared.Role
	err   error
	asked []string
}

func (f *fakePlanningProjects) PlanningFacts(_ context.Context, id uuid.UUID) (project.PlanningFacts, bool, error) {
	f.asked = append(f.asked, "facts "+id.String())
	return f.facts, f.found, f.err
}

func (f *fakePlanningProjects) LockPlanning(_ context.Context, id uuid.UUID) (project.PlanningFacts, bool, error) {
	f.asked = append(f.asked, "lock "+id.String())
	return f.facts, f.found, f.err
}

func (f *fakePlanningProjects) ActiveRoles(_ context.Context, id uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	f.asked = append(f.asked, fmt.Sprint("roles ", id, " ", ids))
	return f.roles, f.err
}

// planningProjects hands planning project's answers to the same questions:
// the facts, read or locked, converted, every field; the active roles of the
// accounts asked for in the project asked for (M6 P2 spec R5), as they came;
// found and the error as they came.
func TestPlanningProjectsConvertsProjectsAnswers(t *testing.T) {
	web, acme, ann, bob := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	fake := &fakePlanningProjects{facts: project.PlanningFacts{WorkspaceID: acme, Archived: true, Timezone: "Asia/Shanghai", CycleView: true, ModuleView: true},
		found: true, roles: map[uuid.UUID]shared.Role{ann: shared.RoleAdmin, bob: shared.RoleGuest}}
	p := planningProjects{projects: fake}
	ctx := context.Background()
	want := planning.ProjectFacts{WorkspaceID: acme, Archived: true, Timezone: "Asia/Shanghai", CycleView: true, ModuleView: true}

	if f, found, err := p.Facts(ctx, web); err != nil || !found || f != want {
		t.Errorf("Facts() = %+v, %v, %v; want %+v, found", f, found, err, want)
	}
	if f, found, err := p.Lock(ctx, web); err != nil || !found || f != want {
		t.Errorf("Lock() = %+v, %v, %v; want %+v, found", f, found, err, want)
	}
	if roles, err := p.Roles(ctx, web, []uuid.UUID{ann, bob}); err != nil || !maps.Equal(roles, fake.roles) {
		t.Errorf("Roles() = %v, %v; want %v", roles, err, fake.roles)
	}
	if want := []string{"facts " + web.String(), "lock " + web.String(), fmt.Sprint("roles ", web, " ", []uuid.UUID{ann, bob})}; !slices.Equal(fake.asked, want) {
		t.Errorf("asked %q, want %q", fake.asked, want)
	}

	boom := errors.New("boom")
	fake.found, fake.err = false, boom
	if _, found, err := p.Lock(ctx, web); !errors.Is(err, boom) || found {
		t.Errorf("Lock() = %v, %v; want the error, not found", found, err)
	}
	if _, err := p.Roles(ctx, web, []uuid.UUID{ann}); !errors.Is(err, boom) {
		t.Errorf("Roles() = %v; want the error", err)
	}
}
