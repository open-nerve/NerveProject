package content

import (
	"reflect"
	"testing"
	"uuid"
)

func TestContentIdentity(t *testing.T) {
	sanitizer := NewSanitizer()
	before := sanitizer.Clean(`<h2>Title</h2><ul><li><p><strong>Task</strong> <a href="https://example.test/a">link</a><input type="checkbox"></p></li></ul>`)
	withIDs := sanitizer.Clean(`<h2 data-id="fixed-heading">Title</h2><ul data-id="list" data-tight="true"><li data-id="item"><p data-id="p"><em>Task</em> <a href="https://example.test/a">link</a><input type="checkbox"></p></li></ul>`)
	if !before.Identity.Equal(withIDs.Identity) {
		t.Fatal("block ids or formatting changed identity")
	}
	changedHref := sanitizer.Clean(`<h2>Title</h2><ul><li><p><strong>Task</strong> <a href="https://example.test/b">link</a><input type="checkbox"></p></li></ul>`)
	if before.Identity.Equal(changedHref.Identity) {
		t.Fatal("href change hidden")
	}
	changedChecked := sanitizer.Clean(`<h2>Title</h2><ul><li><p><strong>Task</strong> <a href="https://example.test/a">link</a><input type="checkbox" checked></p></li></ul>`)
	if before.Identity.Equal(changedChecked.Identity) {
		t.Fatal("checked change hidden")
	}
	if before.Text != "TitleTask link" {
		t.Fatalf("text=%q", before.Text)
	}
	if got := sanitizer.Clean("<p></p>"); got.Text != "" || !got.EmptyComment() {
		t.Fatalf("empty paragraph=%+v", got)
	}
}

func TestContentIdentityMentions(t *testing.T) {
	input := `<p>` +
		`<mention-component entity_name="user_mention" entity_identifier="11111111-1111-4111-8111-111111111111"></mention-component>` +
		`<mention-component entity_name="project_mention" entity_identifier="22222222-2222-4222-8222-222222222222"></mention-component>` +
		`<mention-component entity_name="user_mention" entity_identifier="invalid"></mention-component>` +
		`<mention-component entity_name="user_mention" entity_identifier="22222222-2222-4222-8222-222222222222"></mention-component>` +
		`<mention-component entity_name="user_mention" entity_identifier="11111111-1111-4111-8111-111111111111"></mention-component></p>`
	got := NewSanitizer().Clean(input)
	want := []uuid.UUID{uuid.MustParse("11111111-1111-4111-8111-111111111111"), uuid.MustParse("22222222-2222-4222-8222-222222222222"), uuid.MustParse("11111111-1111-4111-8111-111111111111")}
	if !reflect.DeepEqual(got.Identity.MentionIDs, want) {
		t.Fatalf("mentions=%v", got.Identity.MentionIDs)
	}
	if !got.HasMention || got.EmptyComment() {
		t.Fatalf("mention content lost: %+v", got)
	}
	for _, html := range []string{`<image-component src="https://example.test/image"></image-component>`, `<img src="https://example.test/image">`} {
		if got := NewSanitizer().Clean(html); !got.HasImage || got.EmptyComment() {
			t.Fatalf("image content lost: %+v", got)
		}
	}
	invalid := NewSanitizer().Clean(`<mention-component entity_name="user_mention" entity_identifier="invalid"></mention-component><mention-component entity_name="project_mention" entity_identifier="11111111-1111-4111-8111-111111111111"></mention-component>`)
	if invalid.HasMention || !invalid.EmptyComment() {
		t.Fatalf("non-user mention treated as content: %+v", invalid)
	}
}
