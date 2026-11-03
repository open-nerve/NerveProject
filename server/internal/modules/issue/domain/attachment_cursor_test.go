package domain

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"uuid"
)

func TestAttachmentCursor(t *testing.T) {
	id := uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	after := uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	encode := func(raw string) string { return base64.RawURLEncoding.EncodeToString([]byte(raw)) }
	for _, kind := range []string{"issue", "comment", "draft"} {
		t.Run(kind, func(t *testing.T) {
			raw := fmt.Sprintf(`{"v":1,"kind":%q,"id":%q,"after":%q}`, kind, id, after)
			got, err := EncodeAttachmentCursor(kind, id, after)
			if err != nil || got != encode(raw) {
				t.Fatalf("cursor=%q err=%v want=%q", got, err, encode(raw))
			}
			decoded, err := DecodeAttachmentCursor(got, kind, id)
			if err != nil || decoded != after {
				t.Fatalf("position=%s err=%v", decoded, err)
			}
			for _, bad := range []string{
				"null", "{}", strings.Replace(raw, `"v":1`, `"v":2`, 1),
				strings.Replace(raw, `"v":1`, `"v":1,"v":1`, 1),
				strings.Replace(raw, `"v":1`, `"v":1,"extra":true`, 1),
				strings.Replace(raw, `"v":1`, `"v":"1"`, 1), " " + raw,
				strings.ReplaceAll(raw, "0199a2b4", "0199A2B4"),
				strings.Replace(raw, after.String(), "00000000-0000-0000-0000-000000000000", 1),
				strings.Replace(raw, after.String(), strings.ReplaceAll(after.String(), "-", ""), 1),
				strings.Replace(raw, `"after":"`+after.String()+`"`, `"after":null`, 1),
				strings.Replace(raw, `"kind":"`+kind+`"`, `"kind":"new_comment"`, 1),
				strings.Replace(raw, id.String(), after.String(), 1), raw + "{}",
			} {
				_, err := DecodeAttachmentCursor(encode(bad), kind, id)
				assertCollaborationCursorError(t, err)
			}
			for _, bad := range []string{"", "!", got + "=", got + "\n"} {
				_, err := DecodeAttachmentCursor(bad, kind, id)
				assertCollaborationCursorError(t, err)
			}
			_, err = DecodeAttachmentCursor(got, "other", id)
			assertCollaborationCursorError(t, err)
			_, err = DecodeAttachmentCursor(got, kind, after)
			assertCollaborationCursorError(t, err)
		})
	}
	for _, q := range []struct {
		kind      string
		id, after uuid.UUID
	}{{"new_comment", id, after}, {"issue", uuid.Nil(), after}, {"issue", id, uuid.Nil()}} {
		if _, err := EncodeAttachmentCursor(q.kind, q.id, q.after); err == nil {
			t.Errorf("invalid cursor accepted: %+v", q)
		}
	}
}
