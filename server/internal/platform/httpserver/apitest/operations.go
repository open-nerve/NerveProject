package apitest

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Operation is one operation of the contract, as the whole-program tests of
// bootstrap see it (M2 design 3.6, 3.11).
type Operation struct {
	Method string // upper case
	Path   string
	Public bool // security: []
	params openapi3.Parameters
	body   *openapi3.Schema
}

// Pattern is the route pattern the generated code registers, e.g.
// "POST /api/v0/auth/register".
func (o Operation) Pattern() string { return o.Method + " " + o.Path }

// HasJSONBody reports whether the operation takes an application/json body.
func (o Operation) HasJSONBody() bool { return o.body != nil }

// Target is the request target of an example call: the path with each path
// parameter, and each required query parameter, set to a valid value.
// Parameters bind before the middlewares, so a wrong one would answer 400
// before anything else runs (M2 design 3.6).
func (o Operation) Target() string {
	path, query := o.Path, url.Values{}
	for _, ref := range o.params {
		p := ref.Value
		value := fmt.Sprint(validValue(p.Schema.Value))
		switch {
		case p.In == openapi3.ParameterInPath:
			path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(value))
		case p.In == openapi3.ParameterInQuery && p.Required:
			query.Set(p.Name, value)
		}
	}
	if len(query) == 0 {
		return path
	}
	return path + "?" + query.Encode()
}

// Operations lists every operation of the contract, sorted by pattern.
func (c *Contract) Operations() []Operation {
	var ops []Operation
	for path, item := range c.doc.Paths.Map() {
		for method, op := range item.Operations() {
			o := Operation{Method: strings.ToUpper(method), Path: path, Public: !needsToken(op),
				params: slices.Concat(item.Parameters, op.Parameters)}
			if rb := op.RequestBody; rb != nil && rb.Value != nil {
				if media := rb.Value.Content.Get("application/json"); media != nil && media.Schema != nil {
					o.body = media.Schema.Value
				}
			}
			ops = append(ops, o)
		}
	}
	slices.SortFunc(ops, func(a, b Operation) int { return strings.Compare(a.Pattern(), b.Pattern()) })
	return ops
}

// FieldProblem is one entry of a problem's errors: field and code.
type FieldProblem struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

// BodyCase is a request body that breaks the operation's structure in one
// way, and what it must get: 400 bad_request with exactly Fields, in order;
// or, when Accepted, anything but 400.
type BodyCase struct {
	Name     string
	Body     []byte
	Fields   []FieldProblem
	Accepted bool
}

// unknownField is the property no schema declares.
const unknownField = "nerve_undeclared"

// BodyCases derives the cases of M2 design 3.11's fourth whole-program test
// from the operation's body schema: a valid body, broken one way at a time.
// Cases whose kind of field the schema lacks are left out.
//
//  1. an undeclared top-level property;
//  2. an undeclared property in a nested object;
//  3. null for an optional property that is not nullable;
//  4. each required property missing;
//  5. null for a nullable property: not 400;
//  6. each format property with a wrong string;
//  7. an undeclared property, a missing required property and a wrong
//     format together: every problem in one answer.
func (o Operation) BodyCases() []BodyCase {
	s := o.body
	valid := validValue(s).(map[string]any)
	with := func(change func(body map[string]any)) []byte {
		body := maps.Clone(valid)
		change(body)
		out, _ := json.Marshal(body)
		return out
	}
	var cases []BodyCase
	cases = append(cases, BodyCase{Name: "undeclared property", Body: with(func(b map[string]any) { b[unknownField] = 1 }),
		Fields: []FieldProblem{{unknownField, "not_allowed"}}})
	names := slices.Sorted(maps.Keys(s.Properties))
	for _, name := range names {
		p := s.Properties[name].Value
		if isObject(p) {
			nested := validValue(p).(map[string]any)
			nested[unknownField] = 1
			cases = append(cases, BodyCase{Name: "undeclared property in " + name, Body: with(func(b map[string]any) { b[name] = nested }),
				Fields: []FieldProblem{{name + "." + unknownField, "not_allowed"}}})
			break
		}
	}
	for _, name := range names {
		switch p := s.Properties[name].Value; {
		case isNullable(p):
			cases = append(cases, BodyCase{Name: "null for nullable " + name, Body: with(func(b map[string]any) { b[name] = nil }), Accepted: true})
		case !slices.Contains(s.Required, name):
			cases = append(cases, BodyCase{Name: "null for optional " + name, Body: with(func(b map[string]any) { b[name] = nil }),
				Fields: []FieldProblem{{name, "invalid_format"}}})
		}
	}
	for _, name := range slices.Sorted(slices.Values(s.Required)) {
		cases = append(cases, BodyCase{Name: "missing " + name, Body: with(func(b map[string]any) { delete(b, name) }),
			Fields: []FieldProblem{{name, "required"}}})
	}
	var formatted string
	for _, name := range names {
		if p := s.Properties[name].Value; checkedFormat(p) {
			formatted = name
			cases = append(cases, BodyCase{Name: "wrong " + p.Format + " in " + name, Body: with(func(b map[string]any) { b[name] = "not-a-" + p.Format }),
				Fields: []FieldProblem{{name, "invalid_format"}}})
		}
	}
	if len(s.Required) > 0 {
		missing := slices.Sorted(slices.Values(s.Required))[0]
		all := []FieldProblem{{unknownField, "not_allowed"}, {missing, "required"}}
		if formatted != "" && formatted != missing {
			all = append(all, FieldProblem{formatted, "invalid_format"})
		}
		slices.SortFunc(all, func(a, b FieldProblem) int { return strings.Compare(a.Field, b.Field) })
		cases = append(cases, BodyCase{Name: "every problem at once", Body: with(func(b map[string]any) {
			b[unknownField] = 1
			delete(b, missing)
			if formatted != "" && formatted != missing {
				b[formatted] = "not-a-format"
			}
		}), Fields: all})
	}
	return cases
}

// validValue is a value that s's structure accepts: every required
// property, the first enum value, a valid string for each checked format.
func validValue(s *openapi3.Schema) any {
	if len(s.AnyOf) > 0 {
		for _, alt := range s.AnyOf {
			if !alt.Value.Type.Is("null") {
				return validValue(alt.Value)
			}
		}
	}
	if len(s.Enum) > 0 {
		return s.Enum[0]
	}
	switch {
	case isObject(s):
		obj := map[string]any{}
		for _, name := range s.Required {
			obj[name] = validValue(s.Properties[name].Value)
		}
		return obj
	case s.Type.Includes("array"):
		return []any{}
	case s.Type.Includes("integer"), s.Type.Includes("number"):
		return 1
	case s.Type.Includes("boolean"):
		return false
	}
	switch s.Format {
	case "date-time":
		return "2026-01-01T00:00:00Z"
	case "uuid":
		return "00000000-0000-0000-0000-000000000000"
	case "email":
		return "someone@example.com"
	}
	return "x"
}

func isObject(s *openapi3.Schema) bool {
	return s.Type.Includes("object") && len(s.AnyOf) == 0
}

func isNullable(s *openapi3.Schema) bool {
	if s.Type.IncludesNull() {
		return true
	}
	for _, alt := range s.AnyOf {
		if alt.Value.Type.Is("null") {
			return true
		}
	}
	return false
}

// checkedFormat reports a string format that bodyshape checks: those the
// module template generates as Go types (M2 design 3.12).
func checkedFormat(s *openapi3.Schema) bool {
	return s.Type.Includes("string") && (s.Format == "date-time" || s.Format == "uuid")
}
