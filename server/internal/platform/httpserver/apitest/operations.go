package apitest

import (
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
	ID     string // operationId
	Tags   []string
	Method string // upper case
	Path   string
	Public bool // security: []
	// ProblemHeaders are the headers its default response, the problem,
	// declares, sorted.
	ProblemHeaders []string
	params         openapi3.Parameters
	body           *openapi3.Schema
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
	return o.target("", "")
}

// target is Target with parameter name set to value, when name is not "".
func (o Operation) target(name, value string) string {
	path, query := o.Path, url.Values{}
	for _, ref := range o.params {
		p := ref.Value
		v, set := fmt.Sprint(validValue(p.Schema.Value)), p.Required
		if p.Name == name {
			v, set = value, true
		}
		switch {
		case p.In == openapi3.ParameterInPath:
			path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(v))
		case p.In == openapi3.ParameterInQuery && set:
			query.Set(p.Name, v)
		}
	}
	if len(query) == 0 {
		return path
	}
	return path + "?" + query.Encode()
}

// ParamCase is a request target with one parameter that cannot bind, and
// the parameter and field code the 400 must name.
type ParamCase struct {
	Name   string
	Target string
	Field  string
	Code   string // invalid_format, or required for a parameter left out
}

// ParamCases derives the cases of the parameter binding whole-program test
// (M2 design 3.11): for each path or query parameter whose Go type rejects
// some strings, a number, a boolean, or a string whose format is generated
// as a Go type, the example target with that parameter wrong; for each
// required query parameter, the example target without it. Other strings,
// enums too, bind whatever they are.
func (o Operation) ParamCases() []ParamCase {
	var cases []ParamCase
	for _, ref := range o.params {
		p := ref.Value
		if p.In != openapi3.ParameterInPath && p.In != openapi3.ParameterInQuery {
			continue
		}
		wrong := ""
		switch s := p.Schema.Value; {
		case s.Type.Includes("integer"), s.Type.Includes("number"):
			wrong = "not-a-number"
		case s.Type.Includes("boolean"):
			wrong = "not-a-boolean"
		case checkedFormat(s):
			wrong = "not-a-" + s.Format
		}
		if wrong != "" {
			cases = append(cases, ParamCase{Name: "wrong " + p.Name, Target: o.target(p.Name, wrong), Field: p.Name, Code: "invalid_format"})
		}
		if p.In == openapi3.ParameterInQuery && p.Required {
			cases = append(cases, ParamCase{Name: "missing " + p.Name, Target: o.without(p.Name), Field: p.Name, Code: "required"})
		}
	}
	return cases
}

// without is Target without the query parameter name.
func (o Operation) without(name string) string {
	path, query, _ := strings.Cut(o.Target(), "?")
	values, _ := url.ParseQuery(query)
	values.Del(name)
	if len(values) == 0 {
		return path
	}
	return path + "?" + values.Encode()
}

// Operations lists every operation of the contract, sorted by pattern.
func (c *Contract) Operations() []Operation {
	var ops []Operation
	for path, item := range c.doc.Paths.Map() {
		for method, op := range item.Operations() {
			o := Operation{ID: op.OperationID, Tags: op.Tags, Method: strings.ToUpper(method), Path: path, Public: !needsToken(op),
				params: slices.Concat(item.Parameters, op.Parameters)}
			if rb := op.RequestBody; rb != nil && rb.Value != nil {
				if media := rb.Value.Content.Get("application/json"); media != nil && media.Schema != nil {
					o.body = media.Schema.Value
				}
			}
			if problem := op.Responses.Default(); problem != nil && problem.Value != nil {
				o.ProblemHeaders = slices.Sorted(maps.Keys(problem.Value.Headers))
			}
			ops = append(ops, o)
		}
	}
	slices.SortFunc(ops, func(a, b Operation) int { return strings.Compare(a.Pattern(), b.Pattern()) })
	return ops
}

// validValue is a value that s's structure accepts: every required
// property, the first enum value, a valid string for each checked format,
// one valid item in a list of objects and none in another array.
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
	case isList(s):
		return []any{validValue(s.Items.Value)}
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

// isList reports whether s is an array of objects.
func isList(s *openapi3.Schema) bool {
	return s.Type.Includes("array") && s.Items != nil && s.Items.Value != nil && isObject(s.Items.Value)
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
