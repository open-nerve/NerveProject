package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/things.golden")

const (
	thingsSpec = "testdata/things.yaml"
	thingsConf = "testdata/oapi-codegen.yaml"
)

// The golden file is the reviewed output: every supported construct, the
// type-mapping (uuid checked, email left to the domain), a reference into
// another file, and one root per operation with a JSON body.
func TestGenerateMatchesTheGoldenFile(t *testing.T) {
	got, err := generate(thingsSpec, thingsConf)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "things.golden")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("generate() differs from %s (go test -run Golden -update rewrites it):\n%s", golden, got)
	}
}

// Maps are iterated in random order: numbering must not depend on it, or
// make gen-check reports a difference on an unchanged description.
func TestGenerateIsDeterministic(t *testing.T) {
	first, err := generate(thingsSpec, thingsConf)
	if err != nil {
		t.Fatal(err)
	}
	for range 10 {
		again, err := generate(thingsSpec, thingsConf)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(again, first) {
			t.Fatal("two runs of generate() on the same input differ")
		}
	}
}

// Without the module template's type-mapping, uuid is generated as
// openapi_types.UUID, which has no checker: the table follows the mapping.
func TestGenerateFollowsTheTypeMapping(t *testing.T) {
	conf := filepath.Join(t.TempDir(), "oapi-codegen.yaml")
	if err := os.WriteFile(conf, []byte("package: gen\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := generate(thingsSpec, conf)

	if err == nil || !strings.Contains(err.Error(), `format "email" is generated as openapi_types.Email`) {
		t.Errorf("generate() = %v, want the default email mapping rejected", err)
	}
}

func TestGenerateRejectsWhatItCannotCheck(t *testing.T) {
	tests := []struct{ name, property, want string }{
		{"date", "{type: string, format: date}", `due: format "date" is generated as openapi_types.Date`},
		{"byte", "{type: string, format: byte}", `due: format "byte" is generated as []byte`},
		{"x-go-type", "{type: string, x-go-type: civil.Date}", "due: x-go-type bypasses the type-mapping"},
		{"x-go-type-import", "{type: string, x-go-type-import: {path: time}}", "due: x-go-type-import bypasses the type-mapping"},
		{"oneOf", "{oneOf: [{type: string}, {type: integer}]}", "due: allOf, oneOf and not are not supported"},
		{"allOf", "{allOf: [{type: string}]}", "due: allOf, oneOf and not are not supported"},
		{"not", "{not: {type: string}}", "due: allOf, oneOf and not are not supported"},
		{"anyOf of two types", "{anyOf: [{type: string}, {type: integer}]}", "due: anyOf is supported only as [X, {type: 'null'}]"},
		{"nullable", "{type: string, nullable: true}", "due: nullable is OpenAPI 3.0"},
		{"nested", "{type: object, properties: {at: {type: array, items: {type: string, format: date}}}}", `due.at[]: format "date"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := filepath.Join(t.TempDir(), "m.yaml")
			src := "openapi: 3.1.0\ninfo: {title: t, version: v0}\npaths:\n  /api/v0/x:\n    post:\n" +
				"      requestBody:\n        content:\n          application/json:\n            schema:\n" +
				"              type: object\n              properties:\n                due: " + tt.property + "\n" +
				"      responses: {'204': {description: none}}\n"
			if err := os.WriteFile(spec, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := generate(spec, thingsConf)

			if err == nil || !strings.Contains(err.Error(), "POST /api/v0/x: "+tt.want) {
				t.Errorf("generate() = %v, want an error containing %q", err, "POST /api/v0/x: "+tt.want)
			}
		})
	}
}
