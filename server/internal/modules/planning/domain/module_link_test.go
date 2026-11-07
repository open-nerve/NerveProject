package domain_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// A module's link is checked as a work item's (M6 design 3.2, shared.CheckLinkURL and CheckLinkTitle): the address
// loses its surrounding white space and gains http:// when it has no scheme, and is otherwise kept as given; one that
// is no HTTP or HTTPS address is refused at url; a title may be none or empty, and is refused at title past 255
// characters or with NUL; both problems at once.
func TestCheckNewModuleLink(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   domain.NewModuleLink
		want domain.NewModuleLink
	}{
		{"no title", domain.NewModuleLink{URL: " nerve.dev/Docs "}, domain.NewModuleLink{URL: "http://nerve.dev/Docs"}},
		{"an empty title", domain.NewModuleLink{URL: "HTTPS://nerve.dev", Title: ptr("")}, domain.NewModuleLink{URL: "HTTPS://nerve.dev", Title: ptr("")}},
		{"a title of 255 characters", domain.NewModuleLink{URL: "https://nerve.dev", Title: ptr(strings.Repeat("链", 255))},
			domain.NewModuleLink{URL: "https://nerve.dev", Title: ptr(strings.Repeat("链", 255))}},
	} {
		if got, err := domain.CheckNewModuleLink(tt.in); err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
	for _, tt := range []struct {
		name string
		in   domain.NewModuleLink
		want []string
	}{
		{"another scheme", domain.NewModuleLink{URL: "ftp://nerve.dev"}, []string{"url not_allowed"}},
		{"credentials", domain.NewModuleLink{URL: "https://u:p@nerve.dev"}, []string{"url not_allowed"}},
		{"empty", domain.NewModuleLink{URL: "  "}, []string{"url not_allowed"}},
		{"a long title", domain.NewModuleLink{URL: "https://nerve.dev", Title: ptr(strings.Repeat("x", 256))}, []string{"title too_long"}},
		{"a NUL title", domain.NewModuleLink{URL: "https://nerve.dev", Title: ptr("a\x00")}, []string{"title invalid_format"}},
		{"both", domain.NewModuleLink{URL: "javascript://x", Title: ptr("\x00")}, []string{"url not_allowed", "title invalid_format"}},
	} {
		got, err := domain.CheckNewModuleLink(tt.in)
		if fields := fieldsOf(t, err); !reflect.DeepEqual(fields, tt.want) || !reflect.DeepEqual(got, domain.NewModuleLink{}) {
			t.Errorf("%s: %q, %+v; want %q", tt.name, fields, got, tt.want)
		}
	}
}

// A change of a link checks only what it gives, by a creation's rules: the title alone may be given, without the
// address (M6 P2 spec 2.3), and given null it clears the title; given or not is kept.
func TestCheckModuleLinkPatch(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   domain.ModuleLinkPatch
		want domain.ModuleLinkPatch
	}{
		{"nothing", domain.ModuleLinkPatch{}, domain.ModuleLinkPatch{}},
		{"the title alone", domain.ModuleLinkPatch{SetTitle: true, Title: ptr("Spec")}, domain.ModuleLinkPatch{SetTitle: true, Title: ptr("Spec")}},
		{"the title cleared", domain.ModuleLinkPatch{SetTitle: true}, domain.ModuleLinkPatch{SetTitle: true}},
		{"the address alone", domain.ModuleLinkPatch{URL: ptr(" nerve.dev ")}, domain.ModuleLinkPatch{URL: ptr("http://nerve.dev")}},
		{"both", domain.ModuleLinkPatch{URL: ptr("https://nerve.dev/a"), SetTitle: true, Title: ptr("")},
			domain.ModuleLinkPatch{URL: ptr("https://nerve.dev/a"), SetTitle: true, Title: ptr("")}},
	} {
		if got, err := domain.CheckModuleLinkPatch(tt.in); err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
	for _, tt := range []struct {
		name string
		in   domain.ModuleLinkPatch
		want []string
	}{
		{"a bad address", domain.ModuleLinkPatch{URL: ptr("mailto://x")}, []string{"url not_allowed"}},
		{"a long title", domain.ModuleLinkPatch{SetTitle: true, Title: ptr(strings.Repeat("x", 256))}, []string{"title too_long"}},
		{"both", domain.ModuleLinkPatch{URL: ptr(""), SetTitle: true, Title: ptr("\x00")}, []string{"url not_allowed", "title invalid_format"}},
	} {
		got, err := domain.CheckModuleLinkPatch(tt.in)
		if fields := fieldsOf(t, err); !reflect.DeepEqual(fields, tt.want) || !reflect.DeepEqual(got, domain.ModuleLinkPatch{}) {
			t.Errorf("%s: %q, %+v; want %q", tt.name, fields, got, tt.want)
		}
	}
	// A title not given is not checked: an absent title is not a null one.
	if got, err := domain.CheckModuleLinkPatch(domain.ModuleLinkPatch{Title: ptr(strings.Repeat("x", 300))}); err != nil || got.SetTitle {
		t.Errorf("a title not given: %+v, %v", got, err)
	}
}
