package domain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestListCursor(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	cases := []struct {
		order ListOrder
		key   string
	}{
		{"sort_order", "1.0000000000000002"},
		{"created_at", `"2026-01-01T00:00:00.123456Z"`}, {"-created_at", `"0001-01-01T00:00:00.000000Z"`},
		{"updated_at", `"9999-12-31T23:59:59.999999Z"`}, {"-updated_at", `"2026-01-01T00:00:00.123457Z"`},
		{"start_date", `"0001-01-01"`}, {"-start_date", `"9999-12-31"`}, {"target_date", `"2026-01-01"`}, {"-target_date", `null`},
		{"priority", "0"}, {"-priority", "4"}, {"sub_issues_count", "9007199254740993"}, {"-sub_issues_count", "9223372036854775807"}, {"link_count", "9007199254740993"}, {"-link_count", "9223372036854775807"},
		{"state_name", `"Äbc"`}, {"-state_name", `null`}, {"label_name", `"～"`}, {"-label_name", `"😀"`}, {"assignee_name", `"İ"`}, {"-assignee_name", `""`},
	}
	for _, tc := range cases {
		t.Run(string(tc.order), func(t *testing.T) {
			c := ListCursor{Order: tc.order, Key: json.RawMessage(tc.key), ID: id}
			encoded, err := EncodeListCursor(c)
			if err != nil {
				t.Fatal(err)
			}
			want := fmt.Sprintf(`{"v":1,"p":[%q,%s,"00000000-0000-0000-0000-000000000001"]}`, tc.order, tc.key)
			raw, err := base64.RawURLEncoding.DecodeString(encoded)
			if err != nil || string(raw) != want {
				t.Fatalf("payload=%s err=%v want=%s", raw, err, want)
			}
			got, err := DecodeListCursor(encoded, tc.order)
			if err != nil || got.Order != c.Order || got.ID != id || string(got.Key) != tc.key {
				t.Fatalf("roundtrip=%+v err=%v", got, err)
			}
			_, err = DecodeListCursor(encoded, "invalid")
			assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
		})
	}
	t.Run("encode normalizes typed spelling", func(t *testing.T) {
		for _, tc := range []struct {
			order     ListOrder
			key, want string
		}{
			{"sort_order", "-0", "0"}, {"sort_order", "1.0", "1"}, {"sort_order", "1e0", "1"},
			{"created_at", `"2026-01-01T08:00:00.123456+08:00"`, `"2026-01-01T00:00:00.123456Z"`},
			{"updated_at", `"2026-01-01T00:00:00Z"`, `"2026-01-01T00:00:00.000000Z"`},
			{"state_name", `"ABCÄİ"`, `"abcÄİ"`},
		} {
			encoded, err := EncodeListCursor(ListCursor{Order: tc.order, Key: json.RawMessage(tc.key), ID: id})
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeListCursor(encoded, tc.order)
			if err != nil || string(got.Key) != tc.want {
				t.Fatalf("%s %s => %s, %v want %s", tc.order, tc.key, got.Key, err, tc.want)
			}
		}
	})
	t.Run("key validation applies to encode and decode", func(t *testing.T) {
		for _, tc := range []struct {
			order ListOrder
			key   string
		}{
			{"created_at", "null"}, {"updated_at", "null"}, {"priority", "null"}, {"sub_issues_count", "null"}, {"sort_order", "null"},
			{"created_at", `"0000-12-31T23:00:00-01:00"`}, {"created_at", `"0000-01-01T00:00:00.000000Z"`}, {"created_at", `"10000-01-01T00:00:00.000000Z"`}, {"created_at", `"0001-01-01T00:00:00+01:00"`}, {"created_at", `"9999-12-31T23:59:59-01:00"`},
			{"created_at", `"2026-01-01T00:00:00.1234560001Z"`}, {"created_at", `"2026-01-01T00:00:00.1234567Z"`}, {"created_at", `"2026-01-01"`}, {"created_at", "1"},
			{"start_date", `"2026-02-30"`}, {"target_date", `"2026-01-01T00:00:00Z"`}, {"target_date", "1"},
			{"priority", "5"}, {"priority", "-1"}, {"priority", "1.0"}, {"priority", "1e0"}, {"priority", `"1"`},
			{"sub_issues_count", "-1"}, {"sub_issues_count", "9223372036854775808"}, {"sub_issues_count", "1.5"}, {"sub_issues_count", `"1"`},
			{"sort_order", "1e309"}, {"sort_order", "NaN"}, {"sort_order", "Infinity"}, {"sort_order", `"1"`}, {"sort_order", "true"},
			{"state_name", "42"}, {"label_name", "[]"}, {"assignee_name", "{}"}, {"bad", "1"}, {"-sort_order", "1"},
		} {
			t.Run(string(tc.order)+"/"+tc.key, func(t *testing.T) {
				_, err := EncodeListCursor(ListCursor{Order: tc.order, Key: json.RawMessage(tc.key), ID: id})
				assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
				raw := fmt.Sprintf(`{"v":1,"p":[%q,%s,%q]}`, tc.order, tc.key, id.String())
				_, err = DecodeListCursor(base64.RawURLEncoding.EncodeToString([]byte(raw)), tc.order)
				assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
			})
		}
		for _, order := range []ListOrder{"start_date", "-start_date", "target_date", "-target_date", "state_name", "-state_name", "label_name", "-label_name", "assignee_name", "-assignee_name"} {
			_, err := EncodeListCursor(ListCursor{Order: order, Key: json.RawMessage("null"), ID: id})
			if err != nil {
				t.Fatalf("nullable %s: %v", order, err)
			}
		}
		for _, c := range []ListCursor{{Order: "priority", Key: json.RawMessage("1")}, {Order: "priority", ID: id}, {Order: "priority", Key: json.RawMessage("1 2"), ID: id}} {
			_, err := EncodeListCursor(c)
			assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
		}
	})
	t.Run("canonical envelope payload and order", func(t *testing.T) {
		good := `{"v":1,"p":["sort_order",1,"00000000-0000-0000-0000-000000000001"]}`
		for _, raw := range []string{
			strings.Replace(good, `"v":1`, `"v":2`, 1), strings.Replace(good, `"v":1`, `"v":1,"x":0`, 1), strings.Replace(good, `"v":1`, `"v":1,"v":1`, 1),
			" " + good, good + "null", `{"v":1,"p":null}`, `{"v":1,"p":{}}`, `{"v":1,"p":["sort_order",1]}`,
			strings.Replace(good, `,1,`, `,1.0,`, 1), strings.Replace(good, `,1,`, `,1e0,`, 1), strings.Replace(good, `,1,`, `,-0,`, 1),
			strings.Replace(good, `000000000001`, `000000000000`, 1), strings.Replace(good, `000000000001`, `bad`, 1),
			strings.Replace(good, `"]}`, `",0]}`, 1),
			`{"v":1,"p":["created_at","2026-01-01T08:00:00.123456+08:00","00000000-0000-0000-0000-000000000001"]}`,
			`{"v":1,"p":["state_name","ABC","00000000-0000-0000-0000-000000000001"]}`,
			`{"v":1,"p":["sort_order",1,"AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA"]}`,
		} {
			_, err := DecodeListCursor(base64.RawURLEncoding.EncodeToString([]byte(raw)), "sort_order")
			assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
		}
		encoded := base64.RawURLEncoding.EncodeToString([]byte(good))
		for _, bad := range []string{"", "%%%", encoded + "=", encoded + "\n", encoded[:4] + "\r" + encoded[4:]} {
			_, err := DecodeListCursor(bad, "sort_order")
			assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
		}
		_, err := DecodeListCursor(encoded, "priority")
		assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
	})
}

func TestListCursorCanonicalTypedKeys(t *testing.T) {
	// Use the matching requested order: an order mismatch must not hide a
	// failure to normalize the payload before the shared canonical comparison.
	for _, tc := range []struct {
		order ListOrder
		key   string
	}{
		{"created_at", `"2026-01-01T08:00:00.123456+08:00"`},
		{"created_at", `"2026-01-01T00:00:00Z"`},
		{"updated_at", `"2026-01-01T00:00:00.123Z"`},
		{"updated_at", `"2026-01-01T00:00:00.123456000Z"`},
		{"state_name", `"ABC"`}, {"label_name", `"\u0061"`},
		{"priority", "-0"}, {"sub_issues_count", "-0"}, {"sort_order", "1e0"}, {"sort_order", "1.0"}, {"sort_order", "-0.0"},
	} {
		raw := fmt.Sprintf(`{"v":1,"p":[%q,%s,"00000000-0000-0000-0000-000000000001"]}`, tc.order, tc.key)
		_, err := DecodeListCursor(base64.RawURLEncoding.EncodeToString([]byte(raw)), tc.order)
		assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
	}
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	for _, key := range []string{"-1.7976931348623157e+308", "1.7976931348623157e+308", "5e-324", "-5e-324"} {
		encoded, err := EncodeListCursor(ListCursor{Order: "sort_order", Key: json.RawMessage(key), ID: id})
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeListCursor(encoded, "sort_order")
		if err != nil || string(got.Key) != key {
			t.Fatalf("sort boundary %s became %s (%v)", key, got.Key, err)
		}
	}
	// This envelope has one unused base64 bit pair. Changing only those bits
	// leaves decoded bytes identical, but still is not the canonical cursor.
	raw := `{"v":1,"p":["priority",0,"00000000-0000-0000-0000-000000000001"]}`
	encoded := base64.RawURLEncoding.EncodeToString([]byte(raw))
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	last := strings.IndexByte(alphabet, encoded[len(encoded)-1])
	if len(raw)%3 == 0 || last < 0 {
		t.Fatal("fixture has no unused base64 bits")
	}
	bad := encoded[:len(encoded)-1] + string(alphabet[last+1])
	_, err := DecodeListCursor(bad, "priority")
	assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
}

// NUL cannot be represented by PostgreSQL text; canonical JSON alone is not
// enough to make a name position a valid public cursor.
func TestListCursorRejectsNULNames(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	for _, order := range []ListOrder{"state_name", "-state_name", "label_name", "-label_name", "assignee_name", "-assignee_name"} {
		t.Run(string(order), func(t *testing.T) {
			for _, key := range []string{`"\u0000"`, `"a\u0000名"`} {
				_, err := EncodeListCursor(ListCursor{Order: order, Key: json.RawMessage(key), ID: id})
				assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
				raw := fmt.Sprintf(`{"v":1,"p":[%q,%s,%q]}`, order, key, id.String())
				_, err = DecodeListCursor(base64.RawURLEncoding.EncodeToString([]byte(raw)), order)
				assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
			}
			for _, key := range []string{`null`, `""`, `"Ä名😀\n"`} {
				value, err := EncodeListCursor(ListCursor{Order: order, Key: json.RawMessage(key), ID: id})
				if err != nil {
					t.Fatal(err)
				}
				got, err := DecodeListCursor(value, order)
				if err != nil || string(got.Key) != key {
					t.Fatalf("valid name %s: %s / %v", key, got.Key, err)
				}
			}
		})
	}
}

// Count positions must remain exact integers, independently of sort_order's float key.
func TestListCountCursorBoundaries(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	for _, order := range []ListOrder{"sub_issues_count", "-sub_issues_count", "link_count", "-link_count"} {
		for _, key := range []string{"0", "9007199254740991", "9007199254740992", "9007199254740993", "9223372036854775807"} {
			encoded, err := EncodeListCursor(ListCursor{Order: order, Key: json.RawMessage(key), ID: id})
			if err != nil {
				t.Fatal(order, key, err)
			}
			got, err := DecodeListCursor(encoded, order)
			if err != nil || string(got.Key) != key {
				t.Fatalf("%s %s => %s %v", order, key, got.Key, err)
			}
		}
		for _, key := range []string{"-1", "null", "1.5", "1.0", "1e0", `"1"`, "9223372036854775808", "-0"} {
			if key != "-0" {
				_, err := EncodeListCursor(ListCursor{Order: order, Key: json.RawMessage(key), ID: id})
				assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
			}
			raw := fmt.Sprintf(`{"v":1,"p":[%q,%s,%q]}`, order, key, id.String())
			_, err := DecodeListCursor(base64.RawURLEncoding.EncodeToString([]byte(raw)), order)
			assertListProblem(t, err, shared.KindBadRequest, "cursor", shared.FieldInvalidFormat)
		}
	}
}
