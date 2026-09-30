// Package project is the projects module (M3 design 3.3, 6.3): projects,
// their members, their states and each member's display settings. It
// carries out the workspace module's cascades on the projects
// (ProjectCascade).
package project

import (
	"context"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Cascade is what the workspace module's writes ask of the projects:
// workspace's ProjectCascade port (M3 design 3.3). Each method runs in the
// caller's transaction, which ctx carries, writes by and now into the rows
// it changes, and returns a failure as itself, so the caller's whole write
// rolls back.
type Cascade interface {
	// DeleteWorkspaceProjects soft-deletes the workspace's projects and the
	// rows under them: deleteWorkspace's last step.
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
}

// Deps are what bootstrap gives the module (M3 design 6.6, step 4).
type Deps struct {
	Pool *pgxpool.Pool
}

// Module is the wired project module.
type Module struct {
	cascade *app.Cascade
}

// New wires the module from d; nothing is registered or injected after it
// (M3 design 6.6). bootstrap builds it before workspace, which takes its
// Cascade.
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	return &Module{cascade: app.NewCascade(store)}
}

// Cascade is the module's implementation of workspace's ProjectCascade.
func (m *Module) Cascade() Cascade {
	return m.cascade
}

// Actions lists the module's actions: bootstrap's test holds the union of
// every module's actions equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}
