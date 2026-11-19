package domain

import (
	"encoding/base64"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// An intake cursor is [order, key, id] in the shared envelope (spec 2.3):
// a time order's key the work item's time in UTC at the microsecond, as the
// list cursor spells it; sequence_id's its number. It decodes for its own
// order alone, back to the same position.
func TestIntakeCursor(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	at := time.Date(2026, 11, 19, 9, 0, 0, 123456789, time.FixedZone("UTC+8", 8*3600))
	for _, tc := range []struct {
		cursor  IntakeCursor
		payload string
		back    IntakeCursor
	}{
		{IntakeCursor{Order: "-created_at", At: at, ID: id}, `["-created_at","2026-11-19T01:00:00.123456Z","` + id.String() + `"]`,
			IntakeCursor{Order: "-created_at", At: at.UTC().Truncate(time.Microsecond), ID: id}},
		{IntakeCursor{Order: "updated_at", At: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ID: id}, `["updated_at","2026-01-01T00:00:00.000000Z","` + id.String() + `"]`,
			IntakeCursor{Order: "updated_at", At: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ID: id}},
		{IntakeCursor{Order: "sequence_id", Seq: 12, ID: id}, `["sequence_id",12,"` + id.String() + `"]`, IntakeCursor{Order: "sequence_id", Seq: 12, ID: id}},
		{IntakeCursor{Order: "-sequence_id", Seq: 1, ID: id}, `["-sequence_id",1,"` + id.String() + `"]`, IntakeCursor{Order: "-sequence_id", Seq: 1, ID: id}},
	} {
		encoded, err := EncodeIntakeCursor(tc.cursor)
		if err != nil {
			t.Fatalf("encode %+v: %v", tc.cursor, err)
		}
		raw, err := base64.RawURLEncoding.DecodeString(encoded)
		if want := `{"v":1,"p":` + tc.payload + `}`; err != nil || string(raw) != want {
			t.Fatalf("payload = %s, %v; want %s", raw, err, want)
		}
		got, err := DecodeIntakeCursor(encoded, tc.cursor.Order)
		if err != nil || got != tc.back {
			t.Fatalf("decode = %+v, %v; want %+v", got, err, tc.back)
		}
		for _, other := range []IntakeOrder{"created_at", "-created_at", "updated_at", "-updated_at", "sequence_id", "-sequence_id"} {
			if other != tc.cursor.Order {
				_, err := DecodeIntakeCursor(encoded, other)
				assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
			}
		}
	}
}

// A payload the encoder does not write is the cursor's 400: another shape,
// an order the intake list has not, a key not of its order or not in its
// one spelling, a zero or malformed id. An encoder given no id, an order
// the list has not or a number below 1 refuses too.
func TestIntakeCursorRefusals(t *testing.T) {
	id := `"00000000-0000-0000-0000-000000000001"`
	for _, payload := range []string{
		`[]`, `["-created_at","2026-11-19T01:00:00.123456Z"]`, `["-created_at","2026-11-19T01:00:00.123456Z",` + id + `,1]`, `{}`, `null`,
		`["priority",1,` + id + `]`, `["sort_order",1,` + id + `]`, `["","2026-11-19T01:00:00.123456Z",` + id + `]`,
		`["-created_at","2026-11-19T09:00:00.123456+08:00",` + id + `]`, `["-created_at","2026-11-19T01:00:00Z",` + id + `]`,
		`["-created_at","2026-11-19T01:00:00.1234567Z",` + id + `]`, `["-created_at","2026-11-19",` + id + `]`,
		`["-created_at",null,` + id + `]`, `["-created_at",1,` + id + `]`,
		`["sequence_id",0,` + id + `]`, `["sequence_id",-1,` + id + `]`, `["sequence_id",1.5,` + id + `]`, `["sequence_id","1",` + id + `]`,
		`["sequence_id",null,` + id + `]`, `["sequence_id",1.0,` + id + `]`,
		`["sequence_id",1,"00000000-0000-0000-0000-000000000000"]`, `["sequence_id",1,"x"]`, `["sequence_id",1,1]`,
	} {
		cursor := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"p":` + payload + `}`))
		for _, order := range []IntakeOrder{"-created_at", "sequence_id"} {
			_, err := DecodeIntakeCursor(cursor, order)
			assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
		}
	}
	for _, c := range []IntakeCursor{
		{Order: "-created_at", At: time.Now()}, {Order: "priority", Seq: 1, ID: uuid.NewV7()}, {Order: "sequence_id", ID: uuid.NewV7()},
		{Order: "-sequence_id", Seq: -3, ID: uuid.NewV7()},
	} {
		if _, err := EncodeIntakeCursor(c); err == nil {
			t.Errorf("EncodeIntakeCursor(%+v) = nil error", c)
		}
	}
}
