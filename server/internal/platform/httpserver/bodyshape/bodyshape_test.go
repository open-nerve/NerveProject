package bodyshape

import (
	"encoding/json"
	"slices"
	"testing"
	"time"
	"uuid"
)

const pattern = "POST /api/v0/things"

// things is what bodyshapegen writes for a schema with every supported
// construct: required and optional properties, a nullable object and
// nullable scalars, nesting, an array of objects, an open map, formats.
func things() *Table {
	return &Table{
		Nodes: []Node{
			/* 0 */ {Types: Object, Extra: Closed, Items: Open, Required: []string{"name", "nested"},
				Props: map[string]int{"count": 2, "id": 4, "labels": 10, "maybe": 5, "name": 1, "nested": 6, "note": 11, "tags": 8, "when": 3}},
			/* 1 */ {Types: String, Extra: Open, Items: Open},
			/* 2 */ {Types: Integer, Extra: Open, Items: Open},
			/* 3 */ {Types: String, Format: FormatTime, Extra: Open, Items: Open},
			/* 4 */ {Types: String | Null, Format: FormatUUID, Extra: Open, Items: Open},
			/* 5 */ {Types: Object | Null, Extra: Closed, Items: Open, Props: map[string]int{"a": 1}, Required: []string{"a"}},
			/* 6 */ {Types: Object, Extra: Closed, Items: Open, Props: map[string]int{"a": 1, "b": 7}, Required: []string{"a"}},
			/* 7 */ {Types: Integer | Null, Extra: Open, Items: Open},
			/* 8 */ {Types: Array, Extra: Open, Items: 9},
			/* 9 */ {Types: Object, Extra: Closed, Items: Open, Props: map[string]int{"name": 1}, Required: []string{"name"}},
			/* 10 */ {Types: Object, Extra: 1, Items: Open, Props: map[string]int{}},
			/* 11 */ {Types: Number, Extra: Open, Items: Open},
		},
		Roots: map[string]int{pattern: 0},
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []FieldError
	}{
		{"valid", `{"name":"a","count":3,"when":"2026-09-25T10:00:00Z","id":"0199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70",` +
			`"maybe":{"a":"x"},"nested":{"a":"y","b":2},"tags":[{"name":"t"}],"labels":{"k":"v"},"note":1.5}`, nil},
		{"only the required properties", `{"name":"a","nested":{"a":"y"}}`, nil},
		{"unknown top-level property", `{"name":"a","nested":{"a":"y"},"extra":1}`,
			[]FieldError{{"extra", "not_allowed"}}},
		{"unknown nested property", `{"name":"a","nested":{"a":"y","x":true}}`,
			[]FieldError{{"nested.x", "not_allowed"}}},
		{"null for an optional non-nullable property", `{"name":"a","nested":{"a":"y"},"count":null}`,
			[]FieldError{{"count", "invalid_format"}}},
		{"null for a required non-nullable property", `{"name":null,"nested":{"a":"y"}}`,
			[]FieldError{{"name", "invalid_format"}}},
		{"null where the contract allows it", `{"name":"a","nested":{"a":"y","b":null},"maybe":null,"id":null}`, nil},
		{"missing required properties", `{"nested":{}}`,
			[]FieldError{{"name", "required"}, {"nested.a", "required"}}},
		{"wrong types", `{"name":5,"nested":[],"count":"3"}`,
			[]FieldError{{"count", "invalid_format"}, {"name", "invalid_format"}, {"nested", "invalid_format"}}},
		{"a decimal is not an integer", `{"name":"a","nested":{"a":"y"},"count":1.5}`,
			[]FieldError{{"count", "invalid_format"}}},
		{"an exponent is not an integer", `{"name":"a","nested":{"a":"y"},"count":1e2}`,
			[]FieldError{{"count", "invalid_format"}}},
		{"an integer is a number", `{"name":"a","nested":{"a":"y"},"note":7}`, nil},
		{"array items", `{"name":"a","nested":{"a":"y"},"tags":[{"name":"t"},{},{"name":"u","x":1}]}`,
			[]FieldError{{"tags[1].name", "required"}, {"tags[2].x", "not_allowed"}}},
		{"open map values", `{"name":"a","nested":{"a":"y"},"labels":{"k":5,"j":"ok"}}`,
			[]FieldError{{"labels.k", "invalid_format"}}},
		{"wrong date-time", `{"name":"a","nested":{"a":"y"},"when":"2026-09-25 10:00:00Z"}`,
			[]FieldError{{"when", "invalid_format"}}},
		{"date-time with an escaped Z", `{"name":"a","nested":{"a":"y"},"when":"2026-09-25T10:00:00\u005a"}`, nil},
		{"wrong uuid", `{"name":"a","nested":{"a":"y"},"id":"xyz"}`,
			[]FieldError{{"id", "invalid_format"}}},
		{"the body is not an object", `["name"]`, []FieldError{{"", "invalid_format"}}},
		{"a space before the body", " " + `{"name":"a","nested":{"a":"y"}}`, nil},
		{"a newline before the body", "\n" + `{"name":"a","nested":{"a":"y"}}`, nil},
		{"a tab before the body", "\t" + `{"name":"a","nested":{"a":"y"}}`, nil},
		{"CRLF before the body", "\r\n" + `{"name":"a","nested":{"a":"y"}}`, nil},
		{"whitespace after the body", `{"name":"a","nested":{"a":"y"}}` + " \t\r\n", nil},
		{"a form feed is not JSON whitespace", "\f" + `{"name":"a","nested":{"a":"y"}}`,
			[]FieldError{{"", "invalid_format"}}},
		{"every problem at once, sorted by path", `{"when":"yesterday","zzz":1,"nested":{"a":"y","b":"2"}}`,
			[]FieldError{{"name", "required"}, {"nested.b", "invalid_format"}, {"when", "invalid_format"}, {"zzz", "not_allowed"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := things().Check(pattern, []byte(tt.body)); !slices.Equal(got, tt.want) {
				t.Errorf("Check(%s) = %v, want %v", tt.body, got, tt.want)
			}
		})
	}
}

func TestCheckOfARouteWithoutABody(t *testing.T) {
	if got := things().Check("GET /api/v0/things", []byte(`{"x":1}`)); got != nil {
		t.Errorf("Check() = %v, want nil for a route without a root", got)
	}
}

// The checker must accept exactly what the generated decoder accepts: it
// decodes the field's own bytes into the same Go type (M2 design 3.11).
func TestFormatsMatchTheDecoder(t *testing.T) {
	type decoded struct {
		When time.Time `json:"when"`
		ID   uuid.UUID `json:"id"`
	}
	tests := []struct {
		field  string
		format Format
		raw    string
	}{
		{"when", FormatTime, `"2026-09-25T10:00:00Z"`},
		{"when", FormatTime, `"2026-09-25T10:00:00+08:00"`},
		{"when", FormatTime, `"2026-09-25 10:00:00Z"`},
		{"when", FormatTime, `"2026-09-25"`},
		{"when", FormatTime, `"2026-09-25T10:00:00\u005a"`},
		{"when", FormatTime, `"not a time"`},
		{"id", FormatUUID, `"0199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70"`},
		{"id", FormatUUID, `"0199A2B4-7C3E-7D2A-9F10-2B3C4D5E6F70"`},
		{"id", FormatUUID, `"0199a2b47c3e7d2a9f102b3c4d5e6f70"`},
		{"id", FormatUUID, `"{0199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70}"`},
		{"id", FormatUUID, `"urn:uuid:0199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70"`},
		{"id", FormatUUID, `"\u0030199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70"`},
		{"id", FormatUUID, `"xyz"`},
	}
	accepted := 0
	for _, tt := range tests {
		got := checkFormat(tt.format, []byte(tt.raw))
		want := json.Unmarshal([]byte(`{"`+tt.field+`":`+tt.raw+`}`), new(decoded))
		if (got == nil) != (want == nil) {
			t.Errorf("%s %s: checker %v, decoder %v", tt.field, tt.raw, got, want)
		}
		if got == nil {
			accepted++
		}
	}
	// Both outcomes occur, so the comparison is not trivially one-sided.
	if accepted == 0 || accepted == len(tests) {
		t.Errorf("%d of %d values accepted, want some of each", accepted, len(tests))
	}
}

func TestErrorIsAProblem(t *testing.T) {
	err := &Error{Fields: []FieldError{{"name", "required"}, {"extra", "not_allowed"}}}

	if err.ProblemStatus() != 400 || err.ProblemCode() != "bad_request" {
		t.Errorf("problem = %d %s, want 400 bad_request", err.ProblemStatus(), err.ProblemCode())
	}
	fields := err.ProblemFields()
	if len(fields) != 2 {
		t.Fatalf("ProblemFields() = %v, want 2", fields)
	}
	f, ok := fields[0].(interface {
		ProblemField() string
		ProblemCode() string
	})
	if !ok || f.ProblemField() != "name" || f.ProblemCode() != "required" || fields[0].Error() != "is required" {
		t.Errorf("fields[0] = %v, want name required with a message", fields[0])
	}
}
