// Package access decides what a caller may do in a workspace or a project
// (M3 design 3.3, 3.4, 6.4). It has no table and no operation: it reads the
// memberships through ports that the modules owning them implement, and
// every module asks it through shared.Authorizer.
package access

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/access/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Deps are the ports the Authorizer reads through; bootstrap takes them from
// the other modules' Provide (M3 design 6.6).
type Deps struct {
	WorkspaceRoles app.WorkspaceRoles
}

// New returns the Authorizer (M3 design 6.6, step 3).
func New(d Deps) shared.Authorizer {
	return app.NewAuthorizer(d.WorkspaceRoles)
}

// RuleKeys lists the actions the rule table has a row for; bootstrap's test
// holds them equal to the union of the modules' Actions() (M3 design 3.4).
func RuleKeys() []shared.Action {
	return domain.RuleKeys()
}
