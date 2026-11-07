package domain_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The URL and title rules are shared.CheckLinkURL and CheckLinkTitle
// (internal/shared/link_test.go). These pin what the work item's checks add:
// the normalized URL replaces the given one, a patch without changing the
// caller's string, and a refusal reaches both checks.
func TestLinkURLNormalization(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"case", "  HTTP://Example.com/P?q=A  ", "HTTP://Example.com/P?q=A"},
		{"bare", " example.com/p?next=ftp://x ", "http://example.com/p?next=ftp://x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.CheckNewLink(domain.NewLink{URL: tc.input})
			if err != nil || got.URL != tc.want {
				t.Fatalf("normalized URL = %q, %v; want %q", got.URL, err, tc.want)
			}
			original := tc.input
			patch, err := domain.CheckLinkPatch(domain.LinkPatch{URL: &tc.input})
			if err != nil || patch.URL == nil || *patch.URL != tc.want || tc.input != original {
				t.Fatalf("normalized patch = %#v, %v; want %q", patch, err, tc.want)
			}
		})
	}
	input := "ftp://example.com"
	_, err := domain.CheckNewLink(domain.NewLink{URL: input})
	wantLinkFields(t, err, [2]string{"url", shared.FieldNotAllowed})
	_, err = domain.CheckLinkPatch(domain.LinkPatch{URL: &input})
	wantLinkFields(t, err, [2]string{"url", shared.FieldNotAllowed})
}

// Losing SetTitle conflates omission with an explicit null; trimming title
// or counting its bytes also changes a caller-visible value.
func TestLinkTitlePatch(t *testing.T) {
	for _, title := range []*string{nil, linkString(""), linkString("  Keep me  "), linkString(strings.Repeat("界", 255))} {
		got, err := domain.CheckNewLink(domain.NewLink{URL: "example.com", Title: title})
		if err != nil || !reflect.DeepEqual(got.Title, title) {
			t.Fatalf("create title = %v, %v; want %v", got.Title, err, title)
		}
		for _, set := range []bool{false, true} {
			input := domain.LinkPatch{SetTitle: set, Title: title}
			got, err := domain.CheckLinkPatch(input)
			if err != nil || !reflect.DeepEqual(got, input) {
				t.Fatalf("patch = %#v, %v; want %#v", got, err, input)
			}
		}
	}
	for _, tc := range []struct{ title, code string }{{strings.Repeat("界", 256), shared.FieldTooLong}, {"a\x00b", shared.FieldInvalidFormat}} {
		_, err := domain.CheckNewLink(domain.NewLink{URL: "example.com", Title: &tc.title})
		wantLinkFields(t, err, [2]string{"title", tc.code})
		_, err = domain.CheckLinkPatch(domain.LinkPatch{SetTitle: true, Title: &tc.title})
		wantLinkFields(t, err, [2]string{"title", tc.code})
	}
	badURL, badTitle := "ftp://example.com", strings.Repeat("界", 256)
	_, err := domain.CheckNewLink(domain.NewLink{URL: badURL, Title: &badTitle})
	wantLinkFields(t, err, [2]string{"url", shared.FieldNotAllowed}, [2]string{"title", shared.FieldTooLong})
	_, err = domain.CheckLinkPatch(domain.LinkPatch{URL: &badURL, SetTitle: true, Title: &badTitle})
	wantLinkFields(t, err, [2]string{"url", shared.FieldNotAllowed}, [2]string{"title", shared.FieldTooLong})
}

func linkString(s string) *string { return &s }

func wantLinkFields(t *testing.T, err error, want ...[2]string) {
	t.Helper()
	var problem *shared.Error
	if !errors.As(err, &problem) || problem.Kind != shared.KindInvalid || problem.Code != shared.CodeValidationFailed || len(problem.Fields) != len(want) {
		t.Fatalf("problem = %#v (%v); want fields %v", problem, err, want)
	}
	for n, field := range problem.Fields {
		if field.Field != want[n][0] || field.Code != want[n][1] {
			t.Fatalf("field = %#v; want %v", field, want[n])
		}
	}
}
