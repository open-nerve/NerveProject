package domain

import (
	"fmt"
	"math"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Preferences are an account's display settings in a workspace (M3 design
// 3.18, 5.2): how the sidebar shows the projects, and how many before
// "more".
type Preferences struct {
	NavigationControl      string // one of navigationControls
	NavigationProjectLimit int    // 0 shows every project
}

// PreferencesPatch is a partial update of Preferences: a nil field stays as
// it is.
type PreferencesPatch struct {
	NavigationControl      *string
	NavigationProjectLimit *int
}

// DefaultPreferences are an account's settings in a workspace until it first
// changes them: the columns' defaults (M3 design 4.5), which the store's
// test holds equal.
func DefaultPreferences() Preferences {
	return Preferences{NavigationControl: "ACCORDION", NavigationProjectLimit: 10}
}

// navigationControls are the web app's two modes (TProjectNavigationMode),
// which the column's CHECK holds too.
var navigationControls = []string{"ACCORDION", "TABBED"}

// Apply returns p with the fields patch sets changed.
func (p Preferences) Apply(patch PreferencesPatch) Preferences {
	if patch.NavigationControl != nil {
		p.NavigationControl = *patch.NavigationControl
	}
	if patch.NavigationProjectLimit != nil {
		p.NavigationProjectLimit = *patch.NavigationProjectLimit
	}
	return p
}

// CheckPreferencesPatch checks the fields p sets: a mode of the two, and a
// limit from 0 to the column's integer maximum. Every problem is reported at
// once, as one 422 validation_failed.
func CheckPreferencesPatch(p PreferencesPatch) error {
	var control, limit *shared.FieldError
	if p.NavigationControl != nil && !slices.Contains(navigationControls, *p.NavigationControl) {
		control = &shared.FieldError{Field: "navigation_control_preference", Code: shared.FieldInvalidFormat, Message: "is not ACCORDION or TABBED"}
	}
	if p.NavigationProjectLimit != nil && (*p.NavigationProjectLimit < 0 || *p.NavigationProjectLimit > math.MaxInt32) {
		limit = &shared.FieldError{Field: "navigation_project_limit", Code: shared.FieldOutOfRange,
			Message: fmt.Sprintf("must be between 0 and %d", math.MaxInt32)}
	}
	return invalid(control, limit)
}
