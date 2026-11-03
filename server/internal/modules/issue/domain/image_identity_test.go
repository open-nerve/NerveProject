package domain

import (
	"reflect"
	"testing"
	"uuid"
)

func TestImageContentIdentity(t *testing.T) {
	const a = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	const b = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	sources := []string{b, "", a, "AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA", "aaaaaaaaaaaa4aaa8aaaaaaaaaaaaaaa",
		"urn:uuid:" + a, "{" + a + "}", " " + a, a + " ", "/" + a, "https://example.test/" + a,
		a + "?x=1", "<image-component>", "invalid", "00000000-0000-0000-0000-000000000000", b}
	before := append([]string(nil), sources...)
	want := []uuid.UUID{uuid.MustParse(b), uuid.MustParse(a), uuid.Nil(), uuid.MustParse(b)}
	got := (ContentIdentity{Images: sources}).AssetImageIDs()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("internal references=%v want %v", got, want)
	}
	if !reflect.DeepEqual(sources, before) {
		t.Fatal("extracting references mutated content identity")
	}
	if got := (ContentIdentity{}).AssetImageIDs(); len(got) != 0 {
		t.Fatalf("empty identity has references: %v", got)
	}
}

func TestImageIdentitySkipActivity(t *testing.T) {
	before := RichText{HTML: "old", Identity: ContentIdentity{Images: []string{"b", "a", "b"}}}
	for _, tc := range []struct {
		name    string
		images  []string
		changed bool
	}{
		{"same", []string{"b", "a", "b"}, false},
		{"order", []string{"a", "b", "b"}, true},
		{"duplicate removed", []string{"b", "a"}, true},
		{"source changed", []string{"b", "c", "b"}, true},
		{"empty source added", []string{"b", "", "a", "b"}, true},
		{"all removed", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next := RichText{HTML: "next", Identity: ContentIdentity{Images: tc.images}}
			got := CompareDescription(before, next, true)
			if !got.Write || got.Changed != tc.changed || !reflect.DeepEqual(got.Content, next) {
				t.Fatalf("description change=%+v want changed=%v", got, tc.changed)
			}
		})
	}
	if !(ContentIdentity{}).Equal(ContentIdentity{Images: []string{}}) {
		t.Fatal("empty image slice changed identity")
	}
}
