package domain

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The list's cursor (spec 2.3) is the last row's [created_at, id], the
// list's order, created_at in UTC with its nanoseconds, in the shared
// envelope. A cursor is a position, so an id no row has is read; any other
// shape or spelling is 400 bad_request on cursor.
func TestListCursor(t *testing.T) {
	const idText = "0199a2b4-0000-7000-8000-000000000099"
	id := uuid.MustParse(idText)
	at := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	encoded, err := EncodeListCursor(ListCursor{CreatedAt: at.In(time.FixedZone("CST", 8*3600)), ID: id})
	if err != nil {
		t.Fatal(err)
	}
	const envelope = `{"v":1,"p":["2026-01-02T03:04:05.123456Z","` + idText + `"]}`
	if encoded != base64.RawURLEncoding.EncodeToString([]byte(envelope)) {
		t.Fatalf("cursor = %q, want %s", encoded, envelope)
	}
	got, err := DecodeListCursor(encoded)
	if err != nil || !got.CreatedAt.Equal(at) || got.CreatedAt.Location() != time.UTC || got.ID != id {
		t.Fatalf("roundtrip = %+v, %v", got, err)
	}
	if _, err := EncodeListCursor(ListCursor{CreatedAt: at}); err == nil {
		t.Fatal("a zero id encoded")
	}
	for _, raw := range []string{
		`{"v":1,"p":null}`,
		`{"v":1,"p":[]}`,
		`{"v":1,"p":["2026-01-02T03:04:05.123456Z"]}`,
		`{"v":1,"p":["` + idText + `","2026-01-02T03:04:05.123456Z"]}`,
		`{"v":1,"p":["2026-01-02T03:04:05.123456Z","` + idText + `","extra"]}`,
		`{"v":1,"p":["2026-01-02T03:04:05.123456Z","00000000-0000-0000-0000-000000000000"]}`,
		`{"v":1,"p":["2026-01-02","` + idText + `"]}`,
		`{"v":1,"p":["2026-01-02T03:04:05.123456Z","not-an-id"]}`,
		`{"v":1,"p":{"created_at":"2026-01-02T03:04:05.123456Z","id":"` + idText + `"}}`,
		`{"v":2,"p":["2026-01-02T03:04:05.123456Z","` + idText + `"]}`,
		`{"v":1,"p":["2026-01-02T03:04:05.123456Z","` + idText + `"],"extra":true}`,
		`{"v":1,"p":["2026-01-02T11:04:05.123456+08:00","` + idText + `"]}`,
		`{"v":1,"p":["2026-01-02T03:04:05.1234560Z","` + idText + `"]}`,
		`{"v":1, "p":["2026-01-02T03:04:05.123456Z","` + idText + `"]}`,
	} {
		_, err := DecodeListCursor(base64.RawURLEncoding.EncodeToString([]byte(raw)))
		assertInvalidCursor(t, raw, err)
	}
	for _, raw := range []string{encoded + "=", encoded + "\n", "not-base64", ""} {
		_, err := DecodeListCursor(raw)
		assertInvalidCursor(t, raw, err)
	}
}

func assertInvalidCursor(t *testing.T, raw string, err error) {
	t.Helper()
	var se *shared.Error
	if !errors.As(err, &se) || se.ProblemStatus() != 400 || !errors.Is(err, shared.InvalidCursor()) {
		t.Errorf("DecodeListCursor(%s) = %v, want 400 bad_request", raw, err)
	}
}
