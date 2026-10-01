package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// LogoProps is a project's icon (M3 design 3.19, 5.2): a closed structure,
// every field optional; the zero value, {} as JSON, is no icon. It is the
// web app's TLogoProps (web/packages/types/src/common.ts:21-32), and
// projects.logo_props holds it as this JSON: its CHECK takes the same keys
// and value types (M3 design 4.6).
type LogoProps struct {
	InUse *string `json:"in_use,omitempty"`
	Emoji *Emoji  `json:"emoji,omitempty"`
	Icon  *Icon   `json:"icon,omitempty"`
}

// Emoji is an emoji icon: its value, or the address of a custom one, which
// the web app does not request (M3 design 8.5).
type Emoji struct {
	Value *string `json:"value,omitempty"`
	URL   *string `json:"url,omitempty"`
}

// Icon is a named icon and its colors.
type Icon struct {
	Name            *string `json:"name,omitempty"`
	Color           *string `json:"color,omitempty"`
	BackgroundColor *string `json:"background_color,omitempty"`
}

// The values of LogoProps.InUse: which of the two icons the project shows.
const (
	LogoEmoji = "emoji"
	LogoIcon  = "icon"
)

// checkLogoProps refuses an in_use other than emoji or icon, which the
// column's CHECK refuses too, and NUL in any of its texts, which jsonb
// cannot store. Its keys and their types are the contract's to hold.
func checkLogoProps(l LogoProps) []*shared.FieldError {
	var found []*shared.FieldError
	if l.InUse != nil && *l.InUse != LogoEmoji && *l.InUse != LogoIcon {
		found = append(found, &shared.FieldError{Field: "logo_props.in_use", Code: shared.FieldInvalidFormat, Message: "must be emoji or icon"})
	}
	// The texts of the emoji and the icon given, in the order their
	// problems are reported.
	type text struct {
		field string
		value *string
	}
	var texts []text
	if l.Emoji != nil {
		texts = append(texts, text{"logo_props.emoji.value", l.Emoji.Value}, text{"logo_props.emoji.url", l.Emoji.URL})
	}
	if l.Icon != nil {
		texts = append(texts, text{"logo_props.icon.name", l.Icon.Name}, text{"logo_props.icon.color", l.Icon.Color},
			text{"logo_props.icon.background_color", l.Icon.BackgroundColor})
	}
	for _, t := range texts {
		if t.value != nil {
			found = append(found, checkText(t.field, *t.value))
		}
	}
	return found
}
