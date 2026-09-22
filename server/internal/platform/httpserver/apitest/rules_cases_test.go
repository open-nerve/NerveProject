package apitest

import (
	"slices"
	"strings"
	"testing"
)

// ruleCasesBase follows every authoring rule; each case of
// TestAuthoringRulesReportViolations breaks exactly one.
const ruleCasesBase = `
openapi: 3.1.0
info: {title: rule cases, version: v0}
tags:
  - name: things
paths:
  /api/v0/things:
    get:
      operationId: listThings
      tags: [things]
      parameters:
        - {name: kind, in: query, schema: {type: string, enum: [a, b]}}
      responses:
        '200':
          description: The things.
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Thing'}
        default:
          $ref: '#/components/responses/Problem'
components:
  responses:
    Problem:
      description: Error.
      content:
        application/problem+json:
          schema: {$ref: '#/components/schemas/Problem'}
  schemas:
    Problem:
      type: object
      additionalProperties: false
      properties:
        code: {type: string}
    Thing:
      type: object
      additionalProperties: false
      properties:
        name: {type: string}
        note: {type: [string, 'null']}
        labels: {type: array, items: {type: string}}
    Target:
      oneOf:
        - $ref: '#/components/schemas/Thing'
        - {type: 'null'}
    Named:
      type: object
      allOf:
        - $ref: '#/components/schemas/Thing'
        - {required: [name]}
`

// TestAuthoringRulesReportViolations proves each check of
// authoringViolations on a hand-built document, since the real contract
// passes them all.
func TestAuthoringRulesReportViolations(t *testing.T) {
	if got := authoringViolations(parse(t, ruleCasesBase)); len(got) != 0 {
		t.Fatalf("the base document breaks rules: %q", got)
	}
	const (
		noProblem = "GET /api/v0/things: has no default response whose application/problem+json schema is the Problem component"
		nullable  = ": uses nullable, the OpenAPI 3.0 keyword; write type: [T, 'null']"
		constant  = ": uses const, which oapi-codegen turns into interface{}; write a single-value enum"
		nullEnum  = `: has null in enum, which adds a "<nil>" Go constant; write oneOf: [{$ref: …}, {type: 'null'}]`
		open      = ": object schema does not set additionalProperties: false"
		thing     = "    Thing:\n      type: object\n      additionalProperties: false\n"
	)
	tests := []struct {
		name, old, new, want string
	}{
		{"path outside /api/v0/", "  /api/v0/things:", "  /things:", "/things: does not start with /api/v0/"},
		{"operationId not lower camelCase", "operationId: listThings", "operationId: ListThings",
			`GET /api/v0/things: operationId "ListThings" is not lower camelCase`},
		{"no operationId", "      operationId: listThings\n", "", `GET /api/v0/things: operationId "" is not lower camelCase`},
		{"no tag", "      tags: [things]\n", "", "GET /api/v0/things: has no tag"},
		{"undeclared tag", "tags: [things]", "tags: [stuff]", `GET /api/v0/things: tag "stuff" is not declared in the top-level tags`},
		{"no default response", "        default:\n          $ref: '#/components/responses/Problem'\n", "", noProblem},
		{"default with another schema", "schema: {$ref: '#/components/schemas/Problem'}", "schema: {$ref: '#/components/schemas/Thing'}", noProblem},
		{"default as application/json", "        application/problem+json:\n", "        application/json:\n", noProblem},
		{"nullable", "note: {type: [string, 'null']}", "note: {type: string, nullable: true}", "components/schemas/Thing/properties/note" + nullable},
		{"nullable: false", "note: {type: [string, 'null']}", "note: {type: string, nullable: false}", "components/schemas/Thing/properties/note" + nullable},
		{"const", "name: {type: string}", "name: {type: string, const: thing}", "components/schemas/Thing/properties/name" + constant},
		{"const: null", "name: {type: string}", "name: {const: null}", "components/schemas/Thing/properties/name" + constant},
		{"null in enum", "note: {type: [string, 'null']}", "note: {type: [string, 'null'], enum: [a, null]}", "components/schemas/Thing/properties/note" + nullEnum},
		{"in items", "items: {type: string}", "items: {type: string, nullable: true}", "components/schemas/Thing/properties/labels/items" + nullable},
		{"in oneOf", "- {type: 'null'}", "- {const: null}", "components/schemas/Target/oneOf/1" + constant},
		{"in allOf", "- {required: [name]}", "- {required: [name], nullable: true}", "components/schemas/Named/allOf/1" + nullable},
		{"in a parameter", "enum: [a, b]", "enum: [a, null]", "GET /api/v0/things parameters/kind" + nullEnum},
		{"in an inline response schema", "schema: {$ref: '#/components/schemas/Thing'}", "schema: {type: string, const: x}",
			"GET /api/v0/things responses/200 application/json" + constant},
		{"open object", thing, "    Thing:\n      type: object\n", "components/schemas/Thing" + open},
		{"additionalProperties: true", thing, "    Thing:\n      type: object\n      additionalProperties: true\n", "components/schemas/Thing" + open},
		{"composition with properties of its own", "    Named:\n      type: object\n",
			"    Named:\n      type: object\n      properties: {id: {type: string}}\n", "components/schemas/Named" + open},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(ruleCasesBase, tt.old) {
				t.Fatalf("the base document has no %q", tt.old)
			}
			doc := parse(t, strings.Replace(ruleCasesBase, tt.old, tt.new, 1))
			if got := authoringViolations(doc); !slices.Equal(got, []string{tt.want}) {
				t.Errorf("violations = %q, want %q", got, tt.want)
			}
		})
	}
}
