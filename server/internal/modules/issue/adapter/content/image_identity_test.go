package content

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func TestImageContentIdentity(t *testing.T) {
	const a = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	const b = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	for _, tc := range []struct {
		name, html string
		images     []string
		ids        []uuid.UUID
		hasImage   bool
	}{
		{"ordered duplicate", `<image-component src="` + b + `"></image-component><image-component src="` + a + `"></image-component><image-component src="` + b + `"></image-component>`, []string{b, a, b}, []uuid.UUID{uuid.MustParse(b), uuid.MustParse(a), uuid.MustParse(b)}, true},
		{"safe remote", `<image-component src="https://example.test/a?x=1&amp;y=2"></image-component>`, []string{"https://example.test/a?x=1&y=2"}, nil, true},
		{"upper case UUID", `<image-component src="AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA"></image-component>`, []string{"AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA"}, nil, true},
		{"noncanonical UUID", `<image-component src="aaaaaaaaaaaa4aaa8aaaaaaaaaaaaaaa"></image-component>`, []string{"aaaaaaaaaaaa4aaa8aaaaaaaaaaaaaaa"}, nil, true},
		{"missing source", `<image-component width="10"></image-component>`, []string{""}, nil, true},
		{"empty source", `<image-component width="10" src=""></image-component>`, []string{""}, nil, true},
		{"unsafe source stripped before projection", `<image-component width="10" src="javascript:alert(1)"></image-component>`, []string{""}, nil, true},
		{"raw image", `<img src="` + a + `">`, []string{}, nil, true},
		{"mixed nodes", `<img src="` + a + `"><p><image-component src="` + b + `"></image-component></p><img src="https://example.test/a">`, []string{b}, []uuid.UUID{uuid.MustParse(b)}, true},
		{"no images", `<p>caption</p>`, []string{}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := NewSanitizer().Clean(tc.html)
			if !reflect.DeepEqual(got.Identity.Images, tc.images) || got.HasImage != tc.hasImage {
				t.Fatalf("images=%q hasImage=%v; want %q %v; clean=%q", got.Identity.Images, got.HasImage, tc.images, tc.hasImage, got.HTML)
			}
			if ids := got.Identity.AssetImageIDs(); !reflect.DeepEqual(ids, tc.ids) {
				t.Fatalf("internal asset references=%v want %v", ids, tc.ids)
			}
			if tc.hasImage && got.EmptyComment() {
				t.Fatal("image-only comment lost its content")
			}
		})
	}
}

// Changing an image, its position or its multiplicity must survive a migration
// save's activity suppression; presentation-only changes keep the same identity.
func TestImageIdentitySkipActivity(t *testing.T) {
	const a = `<image-component src="aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"></image-component>`
	const b = `<image-component src="bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"></image-component>`
	sanitizer := NewSanitizer()
	for _, tc := range []struct {
		name, before, after string
		changed             bool
	}{
		{"replace", a, b, true},
		{"order", a + b, b + a, true},
		{"remove duplicate", a + a, a, true},
		{"duplicate order", a + b + a, a + a + b, true},
		{"missing source is a position", a, `<image-component width="10"></image-component>` + a, true},
		{"source removed", a, `<image-component width="10" src=""></image-component>`, true},
		{"remote source changes", `<image-component src="https://example.test/a"></image-component>`, `<image-component src="https://example.test/b"></image-component>`, true},
		{"UUID spelling changes", a, `<image-component src="AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA"></image-component>`, true},
		{"raw image remains outside internal identity", `<img src="https://example.test/a">`, `<img src="https://example.test/b">`, false},
		{"empty and missing source", `<image-component width="10" src=""></image-component>`, `<image-component width="10"></image-component>`, false},
		{"format and block IDs", `<p>caption</p>` + a, `<p data-id="aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"><em>caption</em></p><image-component width="200" alt="preview" src="aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"></image-component>`, false},
		{"source whitespace is cleaned first", a, `<image-component src=" aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa "></image-component>`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, after := sanitizer.Clean(tc.before), sanitizer.Clean(tc.after)
			change := domain.CompareDescription(before, after, true)
			if !change.Write || change.Changed != tc.changed || change.Content.HTML != after.HTML {
				t.Fatalf("migration: write=%v changed=%v want changed=%v; before=%q after=%q", change.Write, change.Changed, tc.changed, before.HTML, after.HTML)
			}
			ordinary := domain.CompareDescription(before, after, false)
			if ordinary.Changed != (before.HTML != after.HTML) || ordinary.Write != ordinary.Changed {
				t.Fatal("ordinary HTML comparison changed")
			}
		})
	}
}
