package domain

import (
	"testing"
	"uuid"
)

func TestRichTextEmptyComment(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value RichText
		empty bool
	}{
		{"empty paragraph", RichText{HTML: "<p></p>"}, true},
		{"text", RichText{Text: "你好"}, false},
		{"mention", RichText{HasMention: true}, false},
		{"image", RichText{HasImage: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.value.EmptyComment(); got != tc.empty {
				t.Fatalf("EmptyComment=%v want %v", got, tc.empty)
			}
		})
	}
}

func TestRichTextIdentityEqual(t *testing.T) {
	a := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	b := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	base := ContentIdentity{Text: "hello", MentionIDs: []uuid.UUID{a, b, a}, Links: []string{"https://example.test"}, Checked: []bool{false, true}}
	for _, tc := range []struct {
		name  string
		value ContentIdentity
		equal bool
	}{
		{"same", base, true},
		{"text", ContentIdentity{Text: "changed", MentionIDs: base.MentionIDs, Links: base.Links, Checked: base.Checked}, false},
		{"mention order", ContentIdentity{Text: base.Text, MentionIDs: []uuid.UUID{b, a, a}, Links: base.Links, Checked: base.Checked}, false},
		{"mention duplicate", ContentIdentity{Text: base.Text, MentionIDs: []uuid.UUID{a, b}, Links: base.Links, Checked: base.Checked}, false},
		{"link", ContentIdentity{Text: base.Text, MentionIDs: base.MentionIDs, Links: []string{"https://changed.test"}, Checked: base.Checked}, false},
		{"checked", ContentIdentity{Text: base.Text, MentionIDs: base.MentionIDs, Links: base.Links, Checked: []bool{true, true}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := base.Equal(tc.value); got != tc.equal {
				t.Fatalf("Equal=%v want %v", got, tc.equal)
			}
		})
	}
	if !(ContentIdentity{}).Equal(ContentIdentity{MentionIDs: []uuid.UUID{}, Links: []string{}, Checked: []bool{}}) {
		t.Fatal("empty slices changed identity")
	}
}
