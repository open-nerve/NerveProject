// Package bodyshape checks the structure of JSON request bodies at the API
// boundary, before the generated strict handler decodes them (M2 design
// 3.11): first that the body can be read one way only (no member name twice
// in an object, no string that is not valid Unicode), then JSON types,
// undeclared properties, null where the contract does not allow it, missing
// required properties, and the string formats that the generated code
// decodes into Go types. The problems of a kind are collected in one pass,
// at most maxProblems of them. Values (lengths, enums, ranges, e-mail syntax)
// are the domain's.
//
// The tables are generated per module from the API description by
// server/tools/bodyshapegen, next to the module's server.gen.go. The package
// uses only the standard library.
package bodyshape

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
	"uuid"
)

// Type is a set of JSON types.
type Type uint8

// The JSON types. Integer is a number literal that an int64 can hold; it also
// counts as a Number, a literal that a float64 can hold. These are the Go
// types bodyshapegen lets the generated code decode numbers into, so the
// check covers their whole range.
const (
	Null Type = 1 << iota
	Boolean
	Integer
	Number
	String
	Array
	Object
)

// Any accepts every JSON type: a schema without `type`.
const Any Type = 0

// Format is a string format that the generated code decodes into a Go type,
// named after that type. A wrong value cannot be held by the generated type,
// so it is checked before decoding.
type Format uint8

// The formats with a checker.
const (
	FormatNone Format = iota
	FormatTime        // time.Time (date-time)
	FormatUUID        // the standard library's uuid.UUID (uuid)
)

// Node.Extra and Node.Items hold a node index or one of these.
const (
	// Closed rejects undeclared properties: additionalProperties: false.
	Closed = -1
	// Open does not check them, or the items of an array without an item
	// schema: additionalProperties: true, or no schema.
	Open = -2
)

// Node is one schema of a request body. Child schemas are indexes into
// Table.Nodes, so recursive schemas are fine.
type Node struct {
	Types    Type
	Format   Format
	Props    map[string]int // object: declared property → node
	Required []string       // object: required properties
	Extra    int            // object: node of undeclared properties, Closed or Open
	Items    int            // array: node of the items, or Open
}

// Table holds the request-body schemas of one module.
type Table struct {
	Nodes []Node
	Roots map[string]int // route pattern, e.g. "POST /api/v0/auth/register" → body node
}

// Field codes, the subset of the closed set of api/common.yaml FieldError
// that the structure check produces.
const (
	codeRequired      = "required"
	codeInvalidFormat = "invalid_format"
	codeNotAllowed    = "not_allowed"
	codeDuplicate     = "duplicate"
)

// FieldError is one structural problem. Field is the JSON path, e.g.
// tags[1].name; the empty path is the body itself. It satisfies the field
// interface of httpserver.ProblemError by structure.
type FieldError struct {
	Field string
	Code  string
}

func (f FieldError) Error() string {
	switch f.Code {
	case codeRequired:
		return "is required"
	case codeNotAllowed:
		return "is not a property of this request"
	case codeDuplicate:
		return "appears more than once in its object"
	default:
		return "has the wrong type or format"
	}
}

// ProblemField returns the JSON path.
func (f FieldError) ProblemField() string { return f.Field }

// ProblemCode returns the field code.
func (f FieldError) ProblemCode() string { return f.Code }

// Error lists every structural problem of one request body. It satisfies
// httpserver.ProblemError by structure: 400 bad_request with the fields.
type Error struct {
	Fields []FieldError
}

func (e *Error) Error() string { return "The request body does not match the API description." }

// ProblemStatus is 400: the request breaks the contract's structure.
func (e *Error) ProblemStatus() int { return 400 }

// ProblemCode is the platform's bad_request.
func (e *Error) ProblemCode() string { return "bad_request" }

// ProblemFields returns the field errors.
func (e *Error) ProblemFields() []error {
	errs := make([]error, len(e.Fields))
	for i, f := range e.Fields {
		errs[i] = f
	}
	return errs
}

// jsonSpace is JSON's insignificant whitespace (RFC 8259 §2): space, tab, CR
// and LF. Other spaces, such as a form feed or U+00A0, are not JSON: the
// decoder rejects a body that starts with one, and so does this package.
const jsonSpace = " \t\r\n"

// ErrNotJSON is Check's answer for a body that is not one valid JSON document.
var ErrNotJSON = errors.New("the request body is not valid JSON")

// Check checks body, a whole request body, against the root of pattern. It
// returns nil when there is nothing to check: a pattern without a root has no
// JSON body, and an empty body, or one of only JSON whitespace, is left to the
// generated decoder. A body that is not one valid JSON document is ErrNotJSON.
// One that can be read two ways (ambiguity.go) is an *Error with its
// ambiguities: its structure is checked once it has one reading. One that
// breaks the structure is an *Error with its problems. Either lists at most
// maxProblems, sorted by path.
func (t *Table) Check(pattern string, body []byte) error {
	root, ok := t.Roots[pattern]
	// walk takes the exact bytes of one value; the whitespace around the
	// document's root value is not part of it.
	value := bytes.Trim(body, jsonSpace)
	switch {
	case !ok, len(value) == 0:
		return nil
	case !json.Valid(value):
		return ErrNotJSON
	}
	var p problems
	ambiguities(value, &p)
	if len(p.errs) == 0 {
		t.walk(root, value, &p)
	}
	if len(p.errs) == 0 {
		return nil
	}
	slices.SortFunc(p.errs, func(a, b FieldError) int {
		return cmp.Or(cmp.Compare(a.Field, b.Field), cmp.Compare(a.Code, b.Code))
	})
	return &Error{Fields: p.errs}
}

// maxProblems bounds the problems of one body. A problem's path can be as
// long as the body, and a body can have a problem every few bytes: listing
// them all could cost the square of the body (M2 design 3.11).
const maxProblems = 16

// problems collects the problems of one body while it is read. The path of
// the value being read is a stack of segments, written out only for a
// problem: a string per value would cost the square of the depth.
type problems struct {
	path []segment
	errs []FieldError
}

// segment is one step of a path: a member's name, or an item's index.
type segment struct {
	name  string
	index int // -1 for a name
}

func (p *problems) enter(name string) { p.path = append(p.path, segment{name, -1}) }

func (p *problems) enterItem(i int) { p.path = append(p.path, segment{"", i}) }

func (p *problems) leave() { p.path = p.path[:len(p.path)-1] }

// full reports whether maxProblems have been reported: the rest is not read.
func (p *problems) full() bool { return len(p.errs) >= maxProblems }

// report adds a problem at the current path, unless p is full.
func (p *problems) report(code string) {
	if !p.full() {
		p.errs = append(p.errs, FieldError{p.at(), code})
	}
}

// at writes out the current path: names joined by dots, indexes in brackets,
// e.g. tags[1].name; the body itself is the empty path.
func (p *problems) at() string {
	var b strings.Builder
	for _, s := range p.path {
		switch {
		case s.index >= 0:
			b.WriteString("[" + strconv.Itoa(s.index) + "]")
		case b.Len() > 0:
			b.WriteString("." + s.name)
		default:
			b.WriteString(s.name)
		}
	}
	return b.String()
}

// walk checks raw, the exact bytes of one JSON value, against node i. It
// goes only where the schema goes: declared properties, the values of an
// open map, the items of an array with an item schema. It reads the members
// of an object in name order, so a body with more than maxProblems problems
// gets the same ones every time.
func (t *Table) walk(i int, raw []byte, p *problems) {
	if p.full() {
		return
	}
	n := t.Nodes[i]
	kind := kindOf(raw)
	if n.Types != Any && kind&n.Types == 0 {
		p.report(codeInvalidFormat)
		return
	}
	switch kind {
	case String:
		if n.Format != FormatNone && checkFormat(n.Format, raw) != nil {
			p.report(codeInvalidFormat)
		}
	case Object:
		var props map[string]json.RawMessage
		_ = json.Unmarshal(raw, &props) // the whole body was valid JSON
		for _, name := range slices.Sorted(maps.Keys(props)) {
			p.enter(name)
			child, declared := n.Props[name]
			switch {
			case declared:
				t.walk(child, props[name], p)
			case n.Extra == Closed:
				p.report(codeNotAllowed)
			case n.Extra >= 0:
				t.walk(n.Extra, props[name], p)
			}
			p.leave()
		}
		for _, name := range n.Required {
			if _, ok := props[name]; !ok {
				p.enter(name)
				p.report(codeRequired)
				p.leave()
			}
		}
	case Array:
		if n.Items < 0 {
			return
		}
		var items []json.RawMessage
		_ = json.Unmarshal(raw, &items)
		for j, item := range items {
			p.enterItem(j)
			t.walk(n.Items, item, p)
			p.leave()
		}
	}
}

// kindOf returns the JSON type of a valid JSON value. A number literal that
// not even a float64 holds, such as 1e400, has no type: the generated decoder
// could not decode it into any field bodyshapegen allows.
func kindOf(raw []byte) Type {
	switch raw[0] {
	case 'n':
		return Null
	case 't', 'f':
		return Boolean
	case '"':
		return String
	case '[':
		return Array
	case '{':
		return Object
	}
	if _, err := strconv.ParseInt(string(raw), 10, 64); err == nil {
		return Integer | Number
	}
	if _, err := strconv.ParseFloat(string(raw), 64); err == nil {
		return Number
	}
	return 0
}

// checkFormat decodes the value's own bytes into the format's Go type with
// encoding/json: exactly what the generated decoder does with this field, so
// the two accept the same values by construction (M2 design 3.11).
func checkFormat(f Format, raw []byte) error {
	switch f {
	case FormatTime:
		return json.Unmarshal(raw, new(time.Time))
	case FormatUUID:
		return json.Unmarshal(raw, new(uuid.UUID))
	}
	return fmt.Errorf("bodyshape: unknown format %d", f)
}
