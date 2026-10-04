package workspace

import (
	"context"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
)

// Deactivator ends a deactivated account's memberships: identity's
// MembershipDeactivator (M3 design 3.9), which bootstrap wires into
// identity.New and identity.NewAdmin. It runs in the deactivation's
// transaction, which ctx carries, under the account row's lock.
type Deactivator interface {
	// DeactivateMemberships ends every workspace and project membership of
	// account userID and deletes every invitation to email, and the pending
	// ones of a workspace he leaves with no active member; its refusal,
	// workspace.sole_admin or project.sole_admin, or its failure comes back
	// as itself.
	DeactivateMemberships(ctx context.Context, userID uuid.UUID, email string) error
}

// DeactivatorDeps are what the Deactivator needs: the pool, the clock and
// the project module's cascade, and no Authorizer, signing key or jobs
// client (M3 design 6.6).
type DeactivatorDeps struct {
	Pool  *pgxpool.Pool
	Clock app.Clock
	// Projects is the project module's Cascade: project.New's, or the
	// command line's project.NewCascade.
	Projects app.ProjectCascade
}

// NewDeactivator builds the Deactivator alone, on the pool: for the
// command line, whose `nerve users deactivate` ends memberships through it
// and builds no HTTP side (M3 design 6.6). New builds its own through it.
func NewDeactivator(d DeactivatorDeps) Deactivator {
	return app.NewDeactivator(postgresadapter.New(d.Pool), d.Projects, d.Clock)
}
