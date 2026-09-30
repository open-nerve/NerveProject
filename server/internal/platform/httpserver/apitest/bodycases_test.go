package apitest

import (
	"slices"
	"strings"
	"testing"
)

func TestBodyCases(t *testing.T) {
	post := contractFrom(t, bodiesContract).Operations()[2]

	var got []string
	for _, c := range post.BodyCases() {
		got = append(got, c.Name+" "+string(c.Body))
		if c.Accepted != (len(c.Fields) == 0) {
			t.Errorf("case %s: accepted %v with fields %v", c.Name, c.Accepted, c.Fields)
		}
	}
	const owner = `"owner_id":"00000000-0000-0000-0000-000000000000"`
	valid := `"name":"x",` + owner
	want := []string{
		`undeclared property {"name":"x","nerve_undeclared":1,` + owner + `}`,
		`undeclared property in settings {` + valid + `,"settings":{"nerve_undeclared":1}}`,
		`null for optional count {"count":null,` + valid + `}`,
		`null for optional kind {"kind":null,` + valid + `}`,
		`null for nullable note {"name":"x","note":null,` + owner + `}`,
		`null for optional settings {` + valid + `,"settings":null}`,
		`missing name {"owner_id":"00000000-0000-0000-0000-000000000000"}`,
		`missing owner_id {"name":"x"}`,
		`wrong uuid in owner_id {"name":"x","owner_id":"not-a-uuid"}`,
		`count twice {"count":1,"count":1,` + valid + `}`,
		`settings.notify twice {` + valid + `,"settings":{"notify":false,"notify":false}}`,
		"not UTF-8 in kind {\"kind\":\"\xff\"," + valid + `}`,
		`every problem at once {"count":null,"nerve_undeclared":1,"owner_id":"not-a-uuid","settings":{"nerve_undeclared":1}}`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("BodyCases() =\n%q\nwant\n%q", got, want)
	}
	last := post.BodyCases()[len(want)-1]
	wantFields := []FieldProblem{{"count", "invalid_format"}, {"name", "required"}, {"nerve_undeclared", "not_allowed"},
		{"owner_id", "invalid_format"}, {"settings.nerve_undeclared", "not_allowed"}}
	if !slices.Equal(last.Fields, wantFields) {
		t.Errorf("every problem at once expects %v, want %v", last.Fields, wantFields)
	}
}

// The case with every problem at once combines whichever kinds the schema
// has, each on a property no other kind took, as soon as there are two; a
// schema with only undeclared properties to offer has no such case.
func TestBodyCasesCombineEveryKindTheSchemaHas(t *testing.T) {
	tests := []struct {
		name, schema string
		want         string // the body of "every problem at once", or "" for none
		fields       []FieldProblem
	}{
		{"no required property", "{type: object, properties: {label: {type: string}, due: {type: [string, 'null'], format: date-time}}}",
			`{"due":"not-a-date-time","label":null,"nerve_undeclared":1}`,
			[]FieldProblem{{"due", "invalid_format"}, {"label", "invalid_format"}, {"nerve_undeclared", "not_allowed"}}},
		{"only a nested object", "{type: object, properties: {step: {type: object, properties: {a: {type: boolean}}}}}",
			`{"nerve_undeclared":1,"step":{"nerve_undeclared":1}}`,
			[]FieldProblem{{"nerve_undeclared", "not_allowed"}, {"step.nerve_undeclared", "not_allowed"}}},
		{"the required property is the only formatted one", "{type: object, required: [id], properties: {id: {type: string, format: uuid}}}",
			`{"nerve_undeclared":1}`, []FieldProblem{{"id", "required"}, {"nerve_undeclared", "not_allowed"}}},
		{"only nullable properties", "{type: object, properties: {note: {type: [string, 'null']}}}", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cases := bodyOperation(t, tt.schema).BodyCases()
			var got *BodyCase
			for i := range cases {
				if cases[i].Name == "every problem at once" {
					got = &cases[i]
				}
			}
			switch {
			case tt.want == "" && got != nil:
				t.Errorf("every problem at once = %s, want no such case", got.Body)
			case tt.want != "" && (got == nil || string(got.Body) != tt.want || !slices.Equal(got.Fields, tt.fields)):
				t.Errorf("every problem at once = %+v, want %s with %v", got, tt.want, tt.fields)
			}
		})
	}
}

// A list of objects (M3 design 5.2) has one valid item in the valid body,
// and its cases: an undeclared property in the item, each required
// property of the item missing, and the first in the case with every
// problem at once. An array of strings has none, and none but the first
// list of objects is taken.
func TestBodyCasesOfAListOfObjects(t *testing.T) {
	const item = "{type: object, additionalProperties: false, required: [role, email], " +
		"properties: {email: {type: string, format: email}, role: {type: integer, enum: [5, 15, 20]}, note: {type: string}}}"
	op := bodyOperation(t, "{type: object, required: [invitations, tags], properties: {invitations: {type: array, items: "+item+
		"}, tags: {type: array, items: {type: string}}, zones: {type: array, items: "+item+"}}}")

	byName := map[string]BodyCase{}
	for _, c := range op.BodyCases() {
		byName[c.Name] = c
	}
	const listed = `"invitations":[{"email":"someone@example.com","role":5}]`
	for _, want := range []BodyCase{
		{Name: "undeclared property in an item of invitations", Body: []byte(`{"invitations":[{"email":"someone@example.com",` +
			`"nerve_undeclared":1,"role":5}],"tags":[]}`), Fields: []FieldProblem{{"invitations[0].nerve_undeclared", "not_allowed"}}},
		{Name: "missing email in an item of invitations", Body: []byte(`{"invitations":[{"role":5}],"tags":[]}`),
			Fields: []FieldProblem{{"invitations[0].email", "required"}}},
		{Name: "missing role in an item of invitations", Body: []byte(`{"invitations":[{"email":"someone@example.com"}],"tags":[]}`),
			Fields: []FieldProblem{{"invitations[0].role", "required"}}},
		{Name: "missing tags", Body: []byte("{" + listed + "}"), Fields: []FieldProblem{{"tags", "required"}}},
		{Name: "every problem at once", Body: []byte(`{"invitations":[{"email":"someone@example.com","nerve_undeclared":1,"role":5}],` +
			`"nerve_undeclared":1,"zones":null}`), Fields: []FieldProblem{{"invitations[0].nerve_undeclared", "not_allowed"},
			{"nerve_undeclared", "not_allowed"}, {"tags", "required"}, {"zones", "invalid_format"}}},
	} {
		got, ok := byName[want.Name]
		if !ok || string(got.Body) != string(want.Body) || !slices.Equal(got.Fields, want.Fields) {
			t.Errorf("%s = %s %v, want %s %v", want.Name, got.Body, got.Fields, want.Body, want.Fields)
		}
	}
	for name := range byName {
		if strings.Contains(name, "item of tags") || strings.Contains(name, "item of zones") {
			t.Errorf("case %q: only the first list of objects has item cases", name)
		}
	}
}

// A case written as text puts the text in place of the one mark it wrote
// into the body. A valid value that is the mark as well would make the case
// test something else, so BodyCases fails loudly.
func TestBodyCasesPanicWhenAValidValueIsTheRawMark(t *testing.T) {
	op := bodyOperation(t, "{type: object, required: [kind], properties: {a: {type: string}, kind: {type: string, enum: [nerve_raw_value]}}}")
	defer func() {
		if recover() == nil {
			t.Error("BodyCases() returned, want a panic: the body holds the raw mark twice")
		}
	}()
	op.BodyCases()
}

// bodyOperation returns the one operation of a contract whose JSON body has
// the given schema.
func bodyOperation(t *testing.T, schema string) Operation {
	t.Helper()
	src := "openapi: 3.1.0\ninfo: {title: body, version: v0}\npaths:\n  /api/v0/x:\n    post:\n" +
		"      requestBody:\n        content:\n          application/json:\n            schema: " + schema + "\n" +
		"      responses: {'204': {description: none}}\n"
	return contractFrom(t, src).Operations()[0]
}
