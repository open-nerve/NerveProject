// Package domain holds the project module's rules (M3 design 6.3): pure
// functions and values.
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// Actions lists the module's actions: the keys of its rows in the access
// module's rule table (M3 design 3.4). bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys. Each operation adds
// its action here with its row.
func Actions() []shared.Action {
	return nil
}
