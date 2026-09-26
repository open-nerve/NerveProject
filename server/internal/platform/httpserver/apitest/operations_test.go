package apitest

import (
	"slices"
	"testing"
)

const bodiesContract = `
openapi: 3.1.0
info: {title: bodies, version: v0}
x-problem-codes: [bad_request]
paths:
  /api/v0/things:
    post:
      operationId: createThing
      security: [{bearer: []}]
      x-problem-codes: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              required: [name, owner_id]
              properties:
                name: {type: string}
                owner_id: {type: string, format: uuid}
                note: {type: [string, 'null']}
                count: {type: integer}
                kind: {type: string, enum: [a, b]}
                settings:
                  type: object
                  additionalProperties: false
                  properties:
                    notify: {type: boolean}
      responses:
        '204': {description: created}
    get:
      operationId: listThings
      security: []
      x-problem-codes: []
      responses:
        '204': {description: none}
        default:
          description: problem
          headers:
            WWW-Authenticate: {schema: {type: string}}
            Retry-After: {schema: {type: integer}}
  /api/v0/things/{thing_id}:
    parameters:
      - {name: thing_id, in: path, required: true, schema: {type: string, format: uuid}}
    get:
      operationId: getThing
      security: [{bearer: []}]
      x-problem-codes: []
      parameters:
        - {name: limit, in: query, required: true, schema: {type: integer}}
        - {name: view, in: query, required: true, schema: {type: string, enum: [full, short]}}
        - {name: q, in: query, schema: {type: string}}
      responses:
        '204': {description: none}
components:
  securitySchemes:
    bearer: {type: http, scheme: bearer}
`

func TestOperations(t *testing.T) {
	ops := contractFrom(t, bodiesContract).Operations()

	if len(ops) != 3 || ops[0].Pattern() != "GET /api/v0/things" || !ops[0].Public || ops[0].HasJSONBody() ||
		ops[1].Pattern() != "GET /api/v0/things/{thing_id}" || ops[1].Public || ops[1].HasJSONBody() ||
		ops[2].Pattern() != "POST /api/v0/things" || ops[2].Public || !ops[2].HasJSONBody() {
		t.Errorf("Operations() = %+v", ops)
	}
	if !slices.Equal(ops[0].ProblemHeaders, []string{"Retry-After", "WWW-Authenticate"}) || ops[1].ProblemHeaders != nil {
		t.Errorf("ProblemHeaders = %q, %q; want the default response's, sorted, and none without one", ops[0].ProblemHeaders, ops[1].ProblemHeaders)
	}
}

func TestTarget(t *testing.T) {
	ops := contractFrom(t, bodiesContract).Operations()

	if got := ops[0].Target(); got != "/api/v0/things" {
		t.Errorf("Target() without parameters = %q", got)
	}
	// The path-level parameter is filled; only the required query parameters
	// are set.
	if got, want := ops[1].Target(), "/api/v0/things/00000000-0000-0000-0000-000000000000?limit=1&view=full"; got != want {
		t.Errorf("Target() = %q, want %q", got, want)
	}
}

// Only the parameters whose Go type rejects some strings get a case: the
// uuid path parameter and the integer, not the enum or the free string. The
// others keep their example values.
func TestParamCases(t *testing.T) {
	ops := contractFrom(t, bodiesContract).Operations()

	got := ops[1].ParamCases()

	want := []ParamCase{
		{"wrong thing_id", "/api/v0/things/not-a-uuid?limit=1&view=full", "thing_id"},
		{"wrong limit", "/api/v0/things/00000000-0000-0000-0000-000000000000?limit=not-a-number&view=full", "limit"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("ParamCases() =\n%q\nwant\n%q", got, want)
	}
	if got := ops[0].ParamCases(); got != nil {
		t.Errorf("ParamCases() without parameters = %q, want none", got)
	}
}

// An optional parameter gets its case too, set only there. A header
// parameter gets none: a target cannot carry it.
func TestParamCasesOfAnOptionalParameter(t *testing.T) {
	op := contractFrom(t, `
openapi: 3.1.0
info: {title: params, version: v0}
paths:
  /api/v0/x:
    get:
      parameters:
        - {name: flag, in: query, schema: {type: boolean}}
        - {name: at, in: query, schema: {type: string, format: date-time}}
        - {name: X-Page, in: header, schema: {type: integer}}
      responses: {'204': {description: none}}
`).Operations()[0]

	want := []ParamCase{
		{"wrong flag", "/api/v0/x?flag=not-a-boolean", "flag"},
		{"wrong at", "/api/v0/x?at=not-a-date-time", "at"},
	}
	if got := op.ParamCases(); !slices.Equal(got, want) || op.Target() != "/api/v0/x" {
		t.Errorf("ParamCases() = %q, Target() = %q; want %q and no query", got, op.Target(), want)
	}
}

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

// bodyOperation returns the one operation of a contract whose JSON body has
// the given schema.
func bodyOperation(t *testing.T, schema string) Operation {
	t.Helper()
	src := "openapi: 3.1.0\ninfo: {title: body, version: v0}\npaths:\n  /api/v0/x:\n    post:\n" +
		"      requestBody:\n        content:\n          application/json:\n            schema: " + schema + "\n" +
		"      responses: {'204': {description: none}}\n"
	return contractFrom(t, src).Operations()[0]
}
