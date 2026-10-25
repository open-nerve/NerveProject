package domain

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestCollaborationCursors(t *testing.T) {
	const idText = "0199a2b4-0000-7000-8000-000000000001"
	id := uuid.MustParse(idText)
	at := time.Date(2026, 9, 25, 18, 0, 0, 123456000, time.FixedZone("", 8*3600))
	for _, tt := range []struct {
		name     string
		value    func(time.Time, uuid.UUID) any
		target   func() any
		position func(any) (time.Time, uuid.UUID)
	}{
		{"comments", func(at time.Time, id uuid.UUID) any { return CommentCursor{CreatedAt: at, ID: id} }, func() any { return new(CommentCursor) }, func(v any) (time.Time, uuid.UUID) { c := v.(*CommentCursor); return c.CreatedAt, c.ID }},
		{"activities", func(at time.Time, id uuid.UUID) any { return ActivityCursor{CreatedAt: at, ID: id} }, func() any { return new(ActivityCursor) }, func(v any) (time.Time, uuid.UUID) { c := v.(*ActivityCursor); return c.CreatedAt, c.ID }},
		{"description versions", func(at time.Time, id uuid.UUID) any { return DescriptionVersionCursor{LastSavedAt: at, ID: id} }, func() any { return new(DescriptionVersionCursor) }, func(v any) (time.Time, uuid.UUID) { c := v.(*DescriptionVersionCursor); return c.LastSavedAt, c.ID }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, at := range []time.Time{at, at.Truncate(time.Second)} {
				value := tt.value(at, id)
				raw, err := json.Marshal(value)
				want := `["2026-09-25T10:00:00.123456Z","` + idText + `"]`
				if at.Nanosecond() == 0 {
					want = `["2026-09-25T10:00:00Z","` + idText + `"]`
				}
				if err != nil || string(raw) != want {
					t.Fatalf("MarshalJSON = %s, %v; want %s", raw, err, want)
				}
				encoded, err := shared.EncodeCursor(value)
				if err != nil {
					t.Fatal(err)
				}
				got := tt.target()
				if err := shared.DecodeCursor(encoded, got); err != nil {
					t.Fatal(err)
				}
				gotAt, gotID := tt.position(got)
				if !gotAt.Equal(at) || gotAt.Location() != time.UTC || gotID != id {
					t.Errorf("decoded = %s, %s; want UTC %s, %s", gotAt, gotID, at.UTC(), id)
				}
				other, err := shared.EncodeCursor(tt.value(at, uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")))
				if err != nil || encoded == other {
					t.Errorf("same timestamp, different ID: %q = %q, %v", encoded, other, err)
				}
			}
			for _, payload := range []string{
				`null`, `{}`, `[]`, `["2026-09-25T10:00:00Z"]`,
				`["2026-09-25T10:00:00Z","` + idText + `","extra"]`,
				`[null,"` + idText + `"]`, `["2026-09-25T10:00:00Z",null]`,
				`[1,"` + idText + `"]`, `["not-a-time","` + idText + `"]`,
				`["2026-09-25T10:00:00Z","not-a-uuid"]`,
				`["2026-09-25T10:00:00Z","00000000-0000-0000-0000-000000000000"]`,
			} {
				if err := json.Unmarshal([]byte(payload), tt.target()); err == nil {
					t.Errorf("UnmarshalJSON(%s) accepted malformed payload", payload)
				}
				cursor := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"p":` + payload + `}`))
				assertCollaborationCursorError(t, shared.DecodeCursor(cursor, tt.target()))
			}
			for _, envelope := range []string{
				`{"v":2,"p":["2026-09-25T10:00:00Z","` + idText + `"]}`,
				`{"v":1,"p":["2026-09-25T10:00:00Z","` + idText + `"],"unknown":true}`,
				`{"v":1,"p":["2026-09-25T18:00:00+08:00","` + idText + `"]}`,
			} {
				cursor := base64.RawURLEncoding.EncodeToString([]byte(envelope))
				assertCollaborationCursorError(t, shared.DecodeCursor(cursor, tt.target()))
			}
			if _, err := shared.EncodeCursor(tt.value(at, uuid.UUID{})); err == nil {
				t.Error("EncodeCursor accepted a zero UUID")
			}
		})
	}
}

func assertCollaborationCursorError(t *testing.T, err error) {
	t.Helper()
	var problem *shared.Error
	if !errors.As(err, &problem) || problem.ProblemStatus() != 400 || !errors.Is(err, shared.InvalidCursor()) {
		t.Errorf("DecodeCursor = %v; want 400 bad_request", err)
	}
}
