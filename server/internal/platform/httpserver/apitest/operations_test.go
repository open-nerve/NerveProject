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
      tags: [things]
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
	if ops[1].ID != "getThing" || !slices.Equal(ops[1].Tags, []string{"things"}) || ops[0].ID != "listThings" || ops[0].Tags != nil {
		t.Errorf("IDs and tags = %q %q, %q %q; want each operation's", ops[0].ID, ops[0].Tags, ops[1].ID, ops[1].Tags)
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

// Only the parameters whose Go type rejects some strings get a wrong value:
// the uuid path parameter and the integer, not the enum or the free string.
// Each required query parameter gets a case without it, the enum too; the
// optional one and the path parameter get none. The others keep their
// example values.
func TestParamCases(t *testing.T) {
	ops := contractFrom(t, bodiesContract).Operations()

	got := ops[1].ParamCases()

	const thing = "/api/v0/things/00000000-0000-0000-0000-000000000000"
	want := []ParamCase{
		{"wrong thing_id", "/api/v0/things/not-a-uuid?limit=1&view=full", "thing_id", "invalid_format"},
		{"wrong limit", thing + "?limit=not-a-number&view=full", "limit", "invalid_format"},
		{"missing limit", thing + "?view=full", "limit", "required"},
		{"missing view", thing + "?limit=1", "view", "required"},
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
		{"wrong flag", "/api/v0/x?flag=not-a-boolean", "flag", "invalid_format"},
		{"wrong at", "/api/v0/x?at=not-a-date-time", "at", "invalid_format"},
	}
	if got := op.ParamCases(); !slices.Equal(got, want) || op.Target() != "/api/v0/x" {
		t.Errorf("ParamCases() = %q, Target() = %q; want %q and no query", got, op.Target(), want)
	}
}
