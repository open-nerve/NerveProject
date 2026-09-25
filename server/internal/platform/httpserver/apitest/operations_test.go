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
		`every problem at once {"nerve_undeclared":1,"owner_id":"not-a-format"}`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("BodyCases() =\n%q\nwant\n%q", got, want)
	}
	last := post.BodyCases()[len(want)-1]
	if wantFields := []FieldProblem{{"name", "required"}, {"nerve_undeclared", "not_allowed"}, {"owner_id", "invalid_format"}}; !slices.Equal(last.Fields, wantFields) {
		t.Errorf("every problem at once expects %v, want %v", last.Fields, wantFields)
	}
}
