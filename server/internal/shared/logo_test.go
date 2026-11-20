package shared_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The icon's rules, which a project's and a view's share (M7/P3b spec R5):
// in_use is emoji or icon, and no text holds NUL; every problem is
// reported, in the order of the fields.
func TestLogoPropsCheck(t *testing.T) {
	s := func(v string) *string { return &v }
	nul := "must not contain a NUL character"
	for _, tc := range []struct {
		name string
		logo shared.LogoProps
		want []shared.FieldError
	}{
		{"no icon", shared.LogoProps{}, nil},
		{"emoji", shared.LogoProps{InUse: s(shared.LogoInUseEmoji), Emoji: &shared.LogoEmoji{Value: s("128640"), URL: s("https://example.com/e.png")}}, nil},
		{"icon", shared.LogoProps{InUse: s(shared.LogoInUseIcon), Icon: &shared.LogoIcon{Name: s("home"), Color: s("#6d7b8a"), BackgroundColor: s("")}}, nil},
		{"in_use other", shared.LogoProps{InUse: s("other")}, []shared.FieldError{{Field: "logo_props.in_use", Code: "invalid_format", Message: "must be emoji or icon"}}},
		{"everything at once", shared.LogoProps{InUse: s(""), Emoji: &shared.LogoEmoji{Value: s("\x00"), URL: s("a\x00")},
			Icon: &shared.LogoIcon{Name: s("\x00"), Color: s("\x00"), BackgroundColor: s("\x00b")}}, []shared.FieldError{
			{Field: "logo_props.in_use", Code: "invalid_format", Message: "must be emoji or icon"},
			{Field: "logo_props.emoji.value", Code: "invalid_format", Message: nul},
			{Field: "logo_props.emoji.url", Code: "invalid_format", Message: nul},
			{Field: "logo_props.icon.name", Code: "invalid_format", Message: nul},
			{Field: "logo_props.icon.color", Code: "invalid_format", Message: nul},
			{Field: "logo_props.icon.background_color", Code: "invalid_format", Message: nul},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []shared.FieldError
			for _, f := range tc.logo.Check() {
				got = append(got, *f)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Check = %+v; want %+v", got, tc.want)
			}
		})
	}
	// The zero value is no icon, {} as its column holds it.
	if b, err := json.Marshal(shared.LogoProps{}); err != nil || string(b) != "{}" {
		t.Errorf("json of no icon = %s, %v; want {}", b, err)
	}
}
