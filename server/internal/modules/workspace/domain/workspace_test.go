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
		{"name that is not UTF-8", name("Ac\xffme"), field("name", "invalid_format", "must be valid UTF-8")},
		{"name of symbols only", name("-_________-"), field("name", "invalid_format", "must contain a letter or a digit")},
		{"name of spaces only", name("   "), field("name", "invalid_format", "must contain a letter or a digit")},
		{"name with a web address", name("Acme www.acme.io"), field("name", "contains_url", "must not contain a web address")},
		{"name with a dotted host", name("acme.io"), field("name", "contains_url", "must not contain a web address")},
		{"empty slug", slug(""), field("slug", "too_short", "must not be empty")},
		{"slug of 49 characters", slug(strings.Repeat("x", 49)), field("slug", "too_long", "must be at most 48 characters")},
		{"upper-case slug", slug("Acme"), field("slug", "invalid_format", "may hold only lower-case letters, digits, - and _")},
		{"slug with a dot", slug("acme.io"), field("slug", "invalid_format", "may hold only lower-case letters, digits, - and _")},
		{"slug with a space", slug("my team"), field("slug", "invalid_format", "may hold only lower-case letters, digits, - and _")},
		{"slug with a trailing newline", slug("acme\n"), field("slug", "invalid_format", "may hold only lower-case letters, digits, - and _")},
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

// A patch is checked by the rules of a new workspace, field by field: a
// field left nil is not checked, so the empty patch passes.
func TestCheckWorkspacePatch(t *testing.T) {
	for _, p := range []WorkspacePatch{
		{},
		{Name: ptr("研发部")},
		{Name: ptr(strings.Repeat("工", 80)), OrganizationSize: ptr("500+"), Timezone: ptr("Asia/Shanghai")},
		{OrganizationSize: ptr("Just myself")},
		{Timezone: ptr("UTC")},
	} {
		if err := CheckWorkspacePatch(p); err != nil {
			t.Errorf("CheckWorkspacePatch(%+v) = %v, want nil", p, err)
		}
	}
	tests := []struct {
		name string
		p    WorkspacePatch
		want []shared.FieldError
	}{
		{"empty name", WorkspacePatch{Name: ptr("")}, []shared.FieldError{{Field: "name", Code: "too_short", Message: "must not be empty"}}},
		{"name with a web address", WorkspacePatch{Name: ptr("acme.io"), Timezone: ptr("UTC")},
			[]shared.FieldError{{Field: "name", Code: "contains_url", Message: "must not contain a web address"}}},
		{"unknown organization size", WorkspacePatch{OrganizationSize: ptr("1000+")},
			[]shared.FieldError{{Field: "organization_size", Code: "invalid_format", Message: "is not a known organization size"}}},
		{"the host's zone", WorkspacePatch{Name: ptr("Acme"), Timezone: ptr("Local")},
			[]shared.FieldError{{Field: "timezone", Code: "invalid_format", Message: "is not a known time zone"}}},
		{"all at once", WorkspacePatch{Name: ptr("-"), OrganizationSize: ptr(""), Timezone: ptr("")}, []shared.FieldError{
			{Field: "name", Code: "invalid_format", Message: "must contain a letter or a digit"},
			{Field: "organization_size", Code: "invalid_format", Message: "is not a known organization size"},
			{Field: "timezone", Code: "invalid_format", Message: "is not a known time zone"},
		}},
	}
	for _, tt := range tests {
		err := CheckWorkspacePatch(tt.p)
		var se *shared.Error
		if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
			t.Errorf("%s: CheckWorkspacePatch(%+v) = %#v, want validation_failed with %v", tt.name, tt.p, err, tt.want)
		}
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
		"acme\n":                 SlugInvalid,
		"sign-up":                SlugReserved,
		"readyz":                 SlugReserved,
		"static":                 SlugReserved,
		"login":                  "", // not a route: /login is a workspace's address (M3 design 3.10)
		"workspace-invitations2": "",
	} {
		if got := CheckSlug(slug); got != want {
			t.Errorf("CheckSlug(%q) = %q, want %q", slug, got, want)
		}
	}
	// Plane's product words that RESTRICTED_URLS still held after M1/P3 are
	// not reserved (the M1/P3 handoff to M3): each can name a workspace.
	for _, word := range []string{"one", "business", "pro", "license", "licenses", "initiatives", "initiative",
		"workflow", "workflows", "story", "disco", "drive", "channels"} {
		if got := CheckSlug(word); got != "" {
			t.Errorf("CheckSlug(%q) = %q, want it usable", word, got)
		}
	}
}
