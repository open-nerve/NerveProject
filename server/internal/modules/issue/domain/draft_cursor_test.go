package domain

import (
	"encoding/base64"
	"testing"
	"time"
	"uuid"
)

// A cursor is a position, so an unknown id is accepted. Shape or canonical
// spelling changes must still be rejected by the shared cursor envelope.
func TestDraftCursor(t *testing.T) {
	const idText = "0199a2b4-0000-7000-8000-000000000099"
	id := uuid.MustParse(idText)
	at := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	encoded, err := EncodeDraftCursor(DraftCursor{CreatedAt: at, ID: id})
	if err != nil {
		t.Fatal(err)
	}
	const envelope = `{"v":1,"p":["2026-01-02T03:04:05.123456Z","` + idText + `"]}`
	if encoded != base64.RawURLEncoding.EncodeToString([]byte(envelope)) {
		t.Fatalf("cursor = %q", encoded)
	}
	got, err := DecodeDraftCursor(encoded)
	if err != nil || !got.CreatedAt.Equal(at) || got.CreatedAt.Location() != time.UTC || got.ID != id {
		t.Fatalf("roundtrip = %+v, %v", got, err)
	}
	if _, err := EncodeDraftCursor(DraftCursor{CreatedAt: at}); err == nil {
		t.Fatal("zero UUID encoded")
	}
	for _, raw := range []string{
		`{"v":1,"p":null}`, `{"v":1,"p":["2026-01-02T03:04:05.123456Z"]}`,
		`{"v":1,"p":["2026-01-02T03:04:05.123456Z","` + idText + `","extra"]}`,
		`{"v":1,"p":["2026-01-02T03:04:05.123456Z","00000000-0000-0000-0000-000000000000"]}`,
		`{"v":2,"p":["2026-01-02T03:04:05.123456Z","` + idText + `"]}`,
		`{"v":1,"p":["2026-01-02T03:04:05.123456Z","` + idText + `"],"extra":true}`,
		`{"v":1,"p":["2026-01-02T11:04:05.123456+08:00","` + idText + `"]}`,
		`{"v":1,"p":["2026-01-02T03:04:05.1234560Z","` + idText + `"]}`,
		`{"v":1, "p":["2026-01-02T03:04:05.123456Z","` + idText + `"]}`,
	} {
		_, err := DecodeDraftCursor(base64.RawURLEncoding.EncodeToString([]byte(raw)))
		assertCollaborationCursorError(t, err)
	}
	for _, raw := range []string{encoded + "=", encoded + "\n", "not-base64"} {
		_, err := DecodeDraftCursor(raw)
		assertCollaborationCursorError(t, err)
	}
}
