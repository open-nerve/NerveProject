package shared

// LogoProps is an icon, a project's (M3 design 3.19, 5.2) or a view's
// (M7/P3b spec R5): a closed structure, every field optional; the zero
// value, {} as JSON, is no icon. It is the web app's TLogoProps
// (web/packages/types/src/common.ts:21-32) and the contract's LogoProps
// (api/common.yaml). A jsonb column holds it as this JSON, and its CHECK
// takes the same keys and value types (projects.logo_props, M3 design 4.6).
type LogoProps struct {
	InUse *string    `json:"in_use,omitempty"`
	Emoji *LogoEmoji `json:"emoji,omitempty"`
	Icon  *LogoIcon  `json:"icon,omitempty"`
}

// LogoEmoji is an emoji icon: its value, or the address of a custom one,
// which the web app does not request (M3 design 8.5).
type LogoEmoji struct {
	Value *string `json:"value,omitempty"`
	URL   *string `json:"url,omitempty"`
}

// LogoIcon is a named icon and its colors.
type LogoIcon struct {
	Name            *string `json:"name,omitempty"`
	Color           *string `json:"color,omitempty"`
	BackgroundColor *string `json:"background_color,omitempty"`
}

// The values of LogoProps.InUse: which of the two icons is shown.
const (
	LogoInUseEmoji = "emoji"
	LogoInUseIcon  = "icon"
)

// Check refuses an in_use other than emoji or icon, which the column's
// CHECK refuses too, and NUL in any of its texts (CheckText), which jsonb
// cannot store. Its keys and their types are the contract's to hold. It
// returns the problems in the order they are reported under logo_props,
// none when there is none.
func (l LogoProps) Check() []*FieldError {
	var found []*FieldError
	if l.InUse != nil && *l.InUse != LogoInUseEmoji && *l.InUse != LogoInUseIcon {
		found = append(found, &FieldError{Field: "logo_props.in_use", Code: FieldInvalidFormat, Message: "must be emoji or icon"})
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
		if t.value == nil {
			continue
		}
		if f := CheckText(t.field, *t.value); f != nil {
			found = append(found, f)
		}
	}
	return found
}
