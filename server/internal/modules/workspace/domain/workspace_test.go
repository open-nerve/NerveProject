package domain

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func ptr[T any](v T) *T { return &v }

func TestCheckNewWorkspaceAcceptsValidWorkspaces(t *testing.T) {
	for _, w := range []NewWorkspace{
		{Name: "Acme", Slug: "acme"},
		{Name: "a", Slug: "a"},
		{Name: strings.Repeat("工", 80), Slug: strings.Repeat("x", 48)},
		{Name: "研发部", Slug: "rd_team-2", OrganizationSize: ptr("Just myself"), Timezone: ptr("Asia/Shanghai")},
		{Name: "٣", Slug: "0", OrganizationSize: ptr("500+"), Timezone: ptr("UTC")},
		{Name: "-_ Team 7 _-", Slug: "-_-"},
	} {
		if err := CheckNewWorkspace(w); err != nil {
			t.Errorf("CheckNewWorkspace(%+v) = %v, want nil", w, err)
		}
	}
	for _, size := range organizationSizes {
		if err := CheckNewWorkspace(NewWorkspace{Name: "Acme", Slug: "acme", OrganizationSize: &size}); err != nil {
			t.Errorf("organization size %q: %v, want nil", size, err)
		}
	}
}

func TestCheckNewWorkspaceReportsEveryField(t *testing.T) {
	valid := NewWorkspace{Name: "Acme", Slug: "acme"}
	with := func(change func(*NewWorkspace)) NewWorkspace {
		w := valid
		change(&w)
		return w
	}
	name := func(n string) NewWorkspace { return with(func(w *NewWorkspace) { w.Name = n }) }
	slug := func(s string) NewWorkspace { return with(func(w *NewWorkspace) { w.Slug = s }) }
	field := func(f, code, message string) []shared.FieldError {
		return []shared.FieldError{{Field: f, Code: code, Message: message}}
	}
	tests := []struct {
		name string
		w    NewWorkspace
		want []shared.FieldError
	}{
		{"empty name", name(""), field("name", "too_short", "must not be empty")},
		{"name of 81 characters", name(strings.Repeat("工", 81)), field("name", "too_long", "must be at most 80 characters")},
		{"name with NUL", name("Ac\x00me"), field("name", "invalid_format", "must not contain a NUL character")},
		{"name of symbols only", name("-_________-"), field("name", "invalid_format", "must contain a letter or a digit")},
		{"name of spaces only", name("   "), field("name", "invalid_format", "must contain a letter or a digit")},
		{"name with a web address", name("Acme www.acme.io"), field("name", "contains_url", "must not contain a web address")},
		{"name with a dotted host", name("acme.io"), field("name", "contains_url", "must not contain a web address")},
		{"empty slug", slug(""), field("slug", "too_short", "must not be empty")},
		{"slug of 49 characters", slug(strings.Repeat("x", 49)), field("slug", "too_long", "must be at most 48 characters")},
		{"upper-case slug", slug("Acme"), field("slug", "invalid_format", "may hold only lower-case letters, digits, - and _")},
		{"slug with a dot", slug("acme.io"), field("slug", "invalid_format", "may hold only lower-case letters, digits, - and _")},
		{"slug with a space", slug("my team"), field("slug", "invalid_format", "may hold only lower-case letters, digits, - and _")},
		{"slug with a non-ASCII letter", slug("équipe"), field("slug", "invalid_format", "may hold only lower-case letters, digits, - and _")},
		{"slug of the app", slug("create-workspace"), field("slug", "not_allowed", "is reserved")},
		{"slug of the app's public directory", slug("icons"), field("slug", "not_allowed", "is reserved")},
		{"slug of the server", slug("api"), field("slug", "not_allowed", "is reserved")},
		{"slug held for later", slug("admin"), field("slug", "not_allowed", "is reserved")},
		{"unknown organization size", with(func(w *NewWorkspace) { w.OrganizationSize = ptr("1000+") }),
			field("organization_size", "invalid_format", "is not a known organization size")},
		{"unknown time zone", with(func(w *NewWorkspace) { w.Timezone = ptr("Mars/Olympus") }),
			field("timezone", "invalid_format", "is not a known time zone")},
		{"the host's zone", with(func(w *NewWorkspace) { w.Timezone = ptr("Local") }),
			field("timezone", "invalid_format", "is not a known time zone")},
		{"all at once", NewWorkspace{Name: "", Slug: "API", OrganizationSize: ptr(""), Timezone: ptr("")}, []shared.FieldError{
			{Field: "name", Code: "too_short", Message: "must not be empty"},
			{Field: "slug", Code: "invalid_format", Message: "may hold only lower-case letters, digits, - and _"},
			{Field: "organization_size", Code: "invalid_format", Message: "is not a known organization size"},
			{Field: "timezone", Code: "invalid_format", Message: "is not a known time zone"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckNewWorkspace(tt.w)
			var se *shared.Error
			if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
				t.Errorf("CheckNewWorkspace(%+v) = %#v, want validation_failed with %v", tt.w, err, tt.want)
			}
		})
	}
}

func TestCheckSlug(t *testing.T) {
	for slug, want := range map[string]SlugReason{
		"acme":                   "",
		"a":                      "",
		strings.Repeat("x", 48):  "",
		"":                       SlugInvalid,
		strings.Repeat("x", 49):  SlugInvalid,
		"Acme":                   SlugInvalid,
		"acme/x":                 SlugInvalid,
		"sign-up":                SlugReserved,
		"readyz":                 SlugReserved,
		"static":                 SlugReserved,
		"login":                  "", // not a route: /login is a workspace's address (M3 design 3.10)
		"one":                    "", // Plane's product words are not reserved
		"workspace-invitations2": "",
	} {
		if got := CheckSlug(slug); got != want {
			t.Errorf("CheckSlug(%q) = %q, want %q", slug, got, want)
		}
	}
}
