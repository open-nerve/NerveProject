package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

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

// rawMark stands for a value that json.Marshal cannot write, until BodyCases
// puts the JSON text in its place.
const rawMark = "nerve_raw_value"

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
//  7. the first property twice: duplicate (a body read two ways);
//  8. the first property of the nested object twice, in it;
//  9. bytes that are not UTF-8 in the first string property: invalid_format;
//  10. every kind of 1–4, 6 and 11 that the schema has, each on a property
//     of its own, together: every problem in one answer. Left out when the
//     schema has only the first kind. 7–9 and 12 are not in it: a body read
//     two ways gets only those answers, before its structure is checked, and
//     12 would take the list 11 takes;
//  11. an undeclared property in the item of the first array of objects
//     (M3 design 5.2), which the valid body lists once;
//  12. each required property of that item missing.
func (o Operation) BodyCases() []BodyCase {
	s := o.body
	valid := validValue(s).(map[string]any)
	with := func(change func(body map[string]any)) []byte {
		body := maps.Clone(valid)
		change(body)
		out, _ := json.Marshal(body)
		return out
	}
	names := slices.Sorted(maps.Keys(s.Properties))
	var nested, listed string
	var optional, required, formatted []string
	for _, name := range names {
		p := s.Properties[name].Value
		if nested == "" && isObject(p) {
			nested = name
		}
		if listed == "" && isList(p) {
			listed = name
		}
		if !isNullable(p) && !slices.Contains(s.Required, name) {
			optional = append(optional, name)
		}
		if checkedFormat(p) {
			formatted = append(formatted, name)
		}
	}
	required = slices.Sorted(slices.Values(s.Required))
	undeclaredIn := func(name string) map[string]any {
		v := validValue(s.Properties[name].Value).(map[string]any)
		v[unknownField] = 1
		return v
	}
	// item is the valid item of the list with change made to it, listed as
	// the list's only item.
	item := func(change func(item map[string]any)) []any {
		v := validValue(s.Properties[listed].Value.Items.Value).(map[string]any)
		change(v)
		return []any{v}
	}
	wrong := func(name string) string { return "not-a-" + s.Properties[name].Value.Format }
	// raw is the valid body with name's value written as text, which json.Marshal would not write: a
	// second member of the same name, bytes that are not UTF-8. The mark must be the body's only one, or
	// the text would land elsewhere, or nowhere, and the case would test something else.
	raw := func(name, text string) []byte {
		body := maps.Clone(valid)
		body[name] = rawMark
		out, _ := json.Marshal(body)
		mark := []byte(`"` + rawMark + `"`)
		if n := bytes.Count(out, mark); n != 1 {
			panic(fmt.Sprintf("apitest: the body %s has the raw mark %d times, want once", out, n))
		}
		return bytes.Replace(out, mark, []byte(text), 1)
	}
	// twice is the text of name's value v, then of a second member name: v.
	twice := func(name string, v any) string {
		value, _ := json.Marshal(v)
		return string(value) + `,"` + name + `":` + string(value)
	}
	// valueOf is name's value in values, else a valid one for its schema.
	valueOf := func(schema *openapi3.Schema, name string, values map[string]any) any {
		if v, ok := values[name]; ok {
			return v
		}
		return validValue(schema.Properties[name].Value)
	}

	cases := []BodyCase{{Name: "undeclared property", Body: with(func(b map[string]any) { b[unknownField] = 1 }),
		Fields: []FieldProblem{{unknownField, "not_allowed"}}}}
	if nested != "" {
		cases = append(cases, BodyCase{Name: "undeclared property in " + nested, Body: with(func(b map[string]any) { b[nested] = undeclaredIn(nested) }),
			Fields: []FieldProblem{{nested + "." + unknownField, "not_allowed"}}})
	}
	for _, name := range names {
		switch {
		case isNullable(s.Properties[name].Value):
			cases = append(cases, BodyCase{Name: "null for nullable " + name, Body: with(func(b map[string]any) { b[name] = nil }), Accepted: true})
		case slices.Contains(optional, name):
			cases = append(cases, BodyCase{Name: "null for optional " + name, Body: with(func(b map[string]any) { b[name] = nil }),
				Fields: []FieldProblem{{name, "invalid_format"}}})
		}
	}
	for _, name := range required {
		cases = append(cases, BodyCase{Name: "missing " + name, Body: with(func(b map[string]any) { delete(b, name) }),
			Fields: []FieldProblem{{name, "required"}}})
	}
	for _, name := range formatted {
		cases = append(cases, BodyCase{Name: "wrong " + s.Properties[name].Value.Format + " in " + name,
			Body: with(func(b map[string]any) { b[name] = wrong(name) }), Fields: []FieldProblem{{name, "invalid_format"}}})
	}
	if len(names) > 0 {
		name := names[0]
		cases = append(cases, BodyCase{Name: name + " twice", Body: raw(name, twice(name, valueOf(s, name, valid))),
			Fields: []FieldProblem{{name, "duplicate"}}})
	}
	if nested != "" {
		n := s.Properties[nested].Value
		if inner := slices.Sorted(maps.Keys(n.Properties)); len(inner) > 0 {
			text := `{"` + inner[0] + `":` + twice(inner[0], valueOf(n, inner[0], nil)) + `}`
			cases = append(cases, BodyCase{Name: nested + "." + inner[0] + " twice", Body: raw(nested, text),
				Fields: []FieldProblem{{nested + "." + inner[0], "duplicate"}}})
		}
	}
	if i := slices.IndexFunc(names, func(name string) bool { return s.Properties[name].Value.Type.Includes("string") }); i >= 0 {
		cases = append(cases, BodyCase{Name: "not UTF-8 in " + names[i], Body: raw(names[i], "\"\xff\""),
			Fields: []FieldProblem{{names[i], "invalid_format"}}})
	}
	if listed != "" {
		cases = append(cases, BodyCase{Name: "undeclared property in an item of " + listed,
			Body:   with(func(b map[string]any) { b[listed] = item(func(i map[string]any) { i[unknownField] = 1 }) }),
			Fields: []FieldProblem{{listed + "[0]." + unknownField, "not_allowed"}}})
		for _, name := range slices.Sorted(slices.Values(s.Properties[listed].Value.Items.Value.Required)) {
			cases = append(cases, BodyCase{Name: "missing " + name + " in an item of " + listed,
				Body:   with(func(b map[string]any) { b[listed] = item(func(i map[string]any) { delete(i, name) }) }),
				Fields: []FieldProblem{{listed + "[0]." + name, "required"}}})
		}
	}

	// Case 10: each kind takes the first property no earlier kind took.
	body := maps.Clone(valid)
	body[unknownField] = 1
	all := []FieldProblem{{unknownField, "not_allowed"}}
	used := map[string]bool{}
	first := func(names []string) string {
		for _, name := range names {
			if !used[name] {
				used[name] = true
				return name
			}
		}
		return ""
	}
	if name := first([]string{nested}); name != "" {
		body[name] = undeclaredIn(name)
		all = append(all, FieldProblem{name + "." + unknownField, "not_allowed"})
	}
	if name := first([]string{listed}); name != "" {
		body[name] = item(func(i map[string]any) { i[unknownField] = 1 })
		all = append(all, FieldProblem{name + "[0]." + unknownField, "not_allowed"})
	}
	if name := first(optional); name != "" {
		body[name] = nil
		all = append(all, FieldProblem{name, "invalid_format"})
	}
	if name := first(required); name != "" {
		delete(body, name)
		all = append(all, FieldProblem{name, "required"})
	}
	if name := first(formatted); name != "" {
		body[name] = wrong(name)
		all = append(all, FieldProblem{name, "invalid_format"})
	}
	if len(all) >= 2 {
		slices.SortFunc(all, func(a, b FieldProblem) int { return strings.Compare(a.Field, b.Field) })
		out, _ := json.Marshal(body)
		cases = append(cases, BodyCase{Name: "every problem at once", Body: out, Fields: all})
	}
	return cases
}
