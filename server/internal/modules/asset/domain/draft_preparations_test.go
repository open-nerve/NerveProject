package domain

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestDraftPreparationCursor(t *testing.T) {
	draft := uuid.MustParse("019a0000-0000-7000-8000-000000000001")
	after := uuid.MustParse("019a0000-0000-7000-8000-000000000002")
	raw := `{"v":1,"p":{"kind":"draft_asset_preparations","draft_id":"019a0000-0000-7000-8000-000000000001","after":"019a0000-0000-7000-8000-000000000002"}}`
	want := base64.RawURLEncoding.EncodeToString([]byte(raw))
	got, err := EncodeDraftPreparationCursor(draft, after)
	if err != nil || got != want {
		t.Fatalf("cursor=%q err=%v want=%q", got, err, want)
	}
	if got, err := DecodeDraftPreparationCursor(draft, &want); err != nil || got != after {
		t.Fatalf("after=%v err=%v", got, err)
	}
	if got, err := DecodeDraftPreparationCursor(draft, nil); err != nil || got != uuid.Nil() {
		t.Fatalf("first page=%v err=%v", got, err)
	}
	for _, bad := range []string{
		"", "not a cursor", want + "=", want + "\n",
		base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(raw, "draft_asset_preparations", "attachments", 1))),
		base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(raw, `"v":1`, `"v":2`, 1))),
		base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(raw, `"v":1`, `"v":1,"v":1`, 1))),
		base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(raw, `"p":{`, `"p":{"extra":true,`, 1))),
		base64.RawURLEncoding.EncodeToString([]byte(strings.Replace(raw, after.String(), uuid.Nil().String(), 1))),
	} {
		if got, err := DecodeDraftPreparationCursor(draft, &bad); !errors.Is(err, shared.InvalidCursor()) || got != uuid.Nil() {
			t.Errorf("invalid cursor %q became %v (%v)", bad, got, err)
		}
	}
	if _, err := DecodeDraftPreparationCursor(after, &want); !errors.Is(err, shared.InvalidCursor()) {
		t.Fatal("cross-draft cursor accepted", err)
	}
	for _, pair := range [][2]uuid.UUID{{uuid.Nil(), after}, {draft, uuid.Nil()}} {
		if _, err := EncodeDraftPreparationCursor(pair[0], pair[1]); err == nil {
			t.Fatal("encoded absent draft/position", pair)
		}
	}
}
