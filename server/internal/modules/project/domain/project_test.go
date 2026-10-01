package domain

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func ptr[T any](v T) *T { return &v }

// CheckNewProject accepts these and returns each as it is stored: the
// identifier in upper case, the network given or public, every other field
// as given.
func TestCheckNewProjectAcceptsValidProjects(t *testing.T) {
	tests := []struct {
		p          NewProject
		identifier string
		network    Network
	}{
		{NewProject{Name: "Web", Identifier: "WEB", LeadID: ptr(uuid.MustParse("0199a2b4-0000-7000-8000-000000000001"))}, "WEB", NetworkPublic},
		{NewProject{Name: "w", Identifier: "w"}, "W", NetworkPublic},
		{NewProject{Name: strings.Repeat("项", 255), Identifier: "abcdefghij"}, "ABCDEFGHIJ", NetworkPublic},
		{NewProject{Name: "研发 Web_2 [beta] \\ /", Identifier: "çşğiöü09", Network: ptr(NetworkPrivate), Timezone: ptr("Asia/Shanghai"),
			Description: "多行\n说明", LogoProps: LogoProps{InUse: ptr(LogoIcon), Icon: &Icon{Name: ptr("home")}}}, "ÇŞĞIÖÜ09", NetworkPrivate},
		{NewProject{Name: "Ops", Identifier: "İ1", Network: ptr(NetworkPublic), LogoProps: LogoProps{InUse: ptr(LogoEmoji)}}, "İ1", NetworkPublic},
	}
	for _, tt := range tests {
		got, err := CheckNewProject(tt.p)
		want := tt.p
		want.Identifier, want.Network = tt.identifier, &tt.network
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("CheckNewProject(%+v) = %+v, %v; want %+v", tt.p, got, err, want)
		}
	}
}

func TestCheckNewProjectReportsEveryField(t *testing.T) {
	valid := NewProject{Name: "Web", Identifier: "WEB"}
	with := func(change func(*NewProject)) NewProject {
		p := valid
		change(&p)
		return p
	}
	name := func(n string) NewProject { return with(func(p *NewProject) { p.Name = n }) }
	identifier := func(s string) NewProject { return with(func(p *NewProject) { p.Identifier = s }) }
	logo := func(l LogoProps) NewProject { return with(func(p *NewProject) { p.LogoProps = l }) }
	field := func(f, code, message string) []shared.FieldError {
		return []shared.FieldError{{Field: f, Code: code, Message: message}}
	}
	const forbidden = "must not contain any of & + , : ; $ ^ } { * = ? @ # | ' < > . ( ) % ! -"
	const nul = "must not contain a NUL character"
	tests := []struct {
		name string
		p    NewProject
		want []shared.FieldError
	}{
		{"empty name", name(""), field("name", "too_short", "must not be empty")},
		{"blank name", name(" \t\n"), field("name", "too_short", "must not be empty")},
		{"name of 256 characters", name(strings.Repeat("项", 256)), field("name", "too_long", "must be at most 255 characters")},
		{"name with NUL", name("W\x00eb"), field("name", "invalid_format", nul)},
		{"empty identifier", identifier(""), field("identifier", "too_short", "must not be empty")},
		{"identifier of 11 characters", identifier("abcdefghijk"), field("identifier", "too_long", "must be at most 10 characters")},
		{"identifier with -", identifier("WEB-2"), field("identifier", "invalid_format", "may hold only A-Z, 0-9 and ÇŞĞİÖÜ")},
		{"identifier with a space", identifier("WEB 2"), field("identifier", "invalid_format", "may hold only A-Z, 0-9 and ÇŞĞİÖÜ")},
		{"identifier with _", identifier("WEB_2"), field("identifier", "invalid_format", "may hold only A-Z, 0-9 and ÇŞĞİÖÜ")},
		{"identifier with another letter", identifier("ÉQUIPE"), field("identifier", "invalid_format", "may hold only A-Z, 0-9 and ÇŞĞİÖÜ")},
		{"identifier with a trailing newline", identifier("WEB\n"), field("identifier", "invalid_format", "may hold only A-Z, 0-9 and ÇŞĞİÖÜ")},
		{"description with NUL", with(func(p *NewProject) { p.Description = "a\x00" }), field("description", "invalid_format", nul)},
		{"network 1", with(func(p *NewProject) { p.Network = ptr(Network(1)) }), field("network", "invalid_format", "must be 0, private, or 2, public")},
		{"network 3", with(func(p *NewProject) { p.Network = ptr(Network(3)) }), field("network", "invalid_format", "must be 0, private, or 2, public")},
		{"unknown time zone", with(func(p *NewProject) { p.Timezone = ptr("Mars/Olympus") }),
			field("timezone", "invalid_format", "is not a known time zone")},
		{"the host's zone", with(func(p *NewProject) { p.Timezone = ptr("Local") }), field("timezone", "invalid_format", "is not a known time zone")},
		{"empty time zone", with(func(p *NewProject) { p.Timezone = ptr("") }), field("timezone", "invalid_format", "is not a known time zone")},
		{"in_use other", logo(LogoProps{InUse: ptr("other")}), field("logo_props.in_use", "invalid_format", "must be emoji or icon")},
		{"in_use empty", logo(LogoProps{InUse: ptr("")}), field("logo_props.in_use", "invalid_format", "must be emoji or icon")},
		{"emoji value with NUL", logo(LogoProps{Emoji: &Emoji{Value: ptr("\x00")}}), field("logo_props.emoji.value", "invalid_format", nul)},
		{"emoji url with NUL", logo(LogoProps{Emoji: &Emoji{URL: ptr("a\x00")}}), field("logo_props.emoji.url", "invalid_format", nul)},
		{"icon name with NUL", logo(LogoProps{Icon: &Icon{Name: ptr("\x00")}}), field("logo_props.icon.name", "invalid_format", nul)},
		{"icon color with NUL", logo(LogoProps{Icon: &Icon{Color: ptr("\x00")}}), field("logo_props.icon.color", "invalid_format", nul)},
		{"icon background with NUL", logo(LogoProps{Icon: &Icon{BackgroundColor: ptr("\x00")}}),
			field("logo_props.icon.background_color", "invalid_format", nul)},
		{"all at once", NewProject{Name: "", Identifier: "web-2", Description: "\x00", Network: ptr(Network(1)), Timezone: ptr(""),
			LogoProps: LogoProps{InUse: ptr("x"), Emoji: &Emoji{Value: ptr("\x00"), URL: ptr("\x00")},
				Icon: &Icon{Name: ptr("\x00"), Color: ptr("\x00"), BackgroundColor: ptr("\x00")}}}, []shared.FieldError{
			{Field: "name", Code: "too_short", Message: "must not be empty"},
			{Field: "identifier", Code: "invalid_format", Message: "may hold only A-Z, 0-9 and ÇŞĞİÖÜ"},
			{Field: "description", Code: "invalid_format", Message: nul},
			{Field: "network", Code: "invalid_format", Message: "must be 0, private, or 2, public"},
			{Field: "timezone", Code: "invalid_format", Message: "is not a known time zone"},
			{Field: "logo_props.in_use", Code: "invalid_format", Message: "must be emoji or icon"},
			{Field: "logo_props.emoji.value", Code: "invalid_format", Message: nul},
			{Field: "logo_props.emoji.url", Code: "invalid_format", Message: nul},
			{Field: "logo_props.icon.name", Code: "invalid_format", Message: nul},
			{Field: "logo_props.icon.color", Code: "invalid_format", Message: nul},
			{Field: "logo_props.icon.background_color", Code: "invalid_format", Message: nul},
		}},
	}
	// Each of Plane's forbidden characters, alone in a name (M3 design 3.19).
	for _, c := range "&+,:;$^}{*=?@#|'<>.()%!-" {
		tests = append(tests, struct {
			name string
			p    NewProject
			want []shared.FieldError
		}{"name with " + string(c), name("Web" + string(c) + "2"), field("name", "not_allowed", forbidden)})
	}
	for _, tt := range tests {
		_, err := CheckNewProject(tt.p)
		var e *shared.Error
		if !errors.As(err, &e) || e.Code != shared.CodeValidationFailed || !slices.Equal(e.Fields, tt.want) {
			t.Errorf("%s: CheckNewProject() = %v, want the fields %+v", tt.name, err, tt.want)
		}
	}
}

// Only a workspace's admins and members may lead a new project, not its
// guests nor a role outside the three (M3 design 3.19).
func TestCanLead(t *testing.T) {
	for role, want := range map[shared.Role]bool{shared.RoleAdmin: true, shared.RoleMember: true, shared.RoleGuest: false, 0: false, 10: false,
		25: false} {
		if got := CanLead(role); got != want {
			t.Errorf("CanLead(%d) = %v, want %v", role, got, want)
		}
	}
}
