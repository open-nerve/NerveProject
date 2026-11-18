// Package access decides what a caller may do in a workspace or a project
// (M3 design 3.3, 3.4, 6.4). It has no table and no operation: it reads the
// memberships through ports that the modules owning them implement, and
// every module asks it through shared.Authorizer; and, for many accounts at
// once, through its Decider (M7 P1b spec 2.7); and which projects of a
// workspace a caller may read, through its ReadableProjects (M7 P1c spec
// 2.5).
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
	// ProjectAccess is project's ProjectAccess, converted
	// (bootstrap/access_ports.go).
	ProjectAccess app.ProjectAccess
}

// ProjectFacts are the facts the ProjectAccess port hands over: bootstrap
// converts project's into them (M3 design 6.5).
type ProjectFacts = app.ProjectFacts

// New returns the Authorizer (M3 design 6.6, step 3).
func New(d Deps) shared.Authorizer {
	return app.NewAuthorizer(d.WorkspaceRoles, d.ProjectAccess)
}

// DeciderDeps are the ports DecideMany reads many accounts' facts through
// (M7 P1b spec 2.7); bootstrap takes them from the other modules' Provide.
type DeciderDeps struct {
	WorkspaceRoles app.WorkspaceRolesMany
	// ProjectAccess is project's ProjectAccessMany, converted
	// (bootstrap/access_ports.go).
	ProjectAccess app.ProjectAccessMany
}

// Decider decides an action for many accounts at once with Authorize's
// rule (DecideMany, M7 P1b spec 2.7); Decision is its answer for one
// account, ProjectFactsMany the facts the ProjectAccessMany port hands
// over, which bootstrap converts project's into.
type (
	Decider          = app.Decider
	Decision         = app.Decision
	ProjectFactsMany = app.ProjectFactsMany
)

// NewDecider returns the Decider over d's ports.
func NewDecider(d DeciderDeps) *Decider {
	return app.NewDecider(d.WorkspaceRoles, d.ProjectAccess)
}

// ReadableDeps are the ports Readable reads a caller's facts through (M7
// P1c spec 2.5); bootstrap takes them from the other modules' Provide.
type ReadableDeps struct {
	WorkspaceRoles app.WorkspaceRoles
	// Projects is project's MemberProjectAccess, converted
	// (bootstrap/access_ports.go).
	Projects app.MemberProjectAccess
}

// ReadableProjects answers the projects of a workspace a caller may read
// the work items of, with Authorize's rule of issue.read (Readable, M7 P1c
// spec 2.5); Grant is its answer for one project, MemberProjectFacts the
// facts the MemberProjectAccess port hands over, which bootstrap converts
// project's into.
type (
	ReadableProjects   = app.ReadableProjects
	Grant              = app.Grant
	MemberProjectFacts = app.MemberProjectFacts
)

// NewReadableProjects returns the ReadableProjects over d's ports.
func NewReadableProjects(d ReadableDeps) *ReadableProjects {
	return app.NewReadableProjects(d.WorkspaceRoles, d.Projects)
}

// RuleKeys lists the actions the rule table has a row for; bootstrap's test
// holds them equal to the union of the modules' Actions() (M3 design 3.4).
func RuleKeys() []shared.Action {
	return domain.RuleKeys()
}
