// Package planning is the cycles and modules module (M6 design 3, 4): it owns
// the cycles' tables and brings creating, reading, listing, changing,
// archiving, unarchiving and deleting a cycle of a project.
package planning

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Actions lists the module's actions: bootstrap's test holds the union of
// every module's actions equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}
