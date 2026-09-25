// Package bodyshape checks the structure of JSON request bodies at the API
// boundary, before the generated strict handler decodes them (M2 design
// 3.11): JSON types, undeclared properties, null where the contract does not
// allow it, missing required properties, and the string formats that the
// generated code decodes into Go types. Every problem is collected in one
// pass. Values (lengths, enums, ranges, e-mail syntax) are the domain's.
//
// The tables are generated per module from the API description by
// server/tools/bodyshapegen, next to the module's server.gen.go. The package
// uses only the standard library.
package bodyshape

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
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

// Check reports the structural problems of body, a syntactically valid JSON
// document, against the root of pattern, sorted by path. A pattern without a
// root has no JSON body to check.
func (t *Table) Check(pattern string, body []byte) []FieldError {
	root, ok := t.Roots[pattern]
	if !ok {
		return nil
	}
	var errs []FieldError
	// walk takes the exact bytes of one value; the whitespace around the
	// document's root value is not part of it.
	t.walk(root, bytes.Trim(body, jsonSpace), "", &errs)
	slices.SortFunc(errs, func(a, b FieldError) int {
		return cmp.Or(cmp.Compare(a.Field, b.Field), cmp.Compare(a.Code, b.Code))
	})
	return errs
}

// walk checks raw, the exact bytes of one JSON value, against node i.
func (t *Table) walk(i int, raw []byte, path string, errs *[]FieldError) {
	n := t.Nodes[i]
	kind := kindOf(raw)
	if n.Types != Any && kind&n.Types == 0 {
		*errs = append(*errs, FieldError{path, codeInvalidFormat})
		return
	}
	switch kind {
	case String:
		if n.Format != FormatNone && checkFormat(n.Format, raw) != nil {
			*errs = append(*errs, FieldError{path, codeInvalidFormat})
		}
	case Object:
		var props map[string]json.RawMessage
		_ = json.Unmarshal(raw, &props) // the whole body was valid JSON
		for name, value := range props {
			child, declared := n.Props[name]
			switch {
			case declared:
				t.walk(child, value, join(path, name), errs)
			case n.Extra == Closed:
				*errs = append(*errs, FieldError{join(path, name), codeNotAllowed})
			case n.Extra >= 0:
				t.walk(n.Extra, value, join(path, name), errs)
			}
		}
		for _, name := range n.Required {
			if _, ok := props[name]; !ok {
				*errs = append(*errs, FieldError{join(path, name), codeRequired})
			}
		}
	case Array:
		if n.Items < 0 {
			return
		}
		var items []json.RawMessage
		_ = json.Unmarshal(raw, &items)
		for j, item := range items {
			t.walk(n.Items, item, fmt.Sprintf("%s[%d]", path, j), errs)
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

func join(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}
