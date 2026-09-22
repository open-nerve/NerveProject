// Package apitest checks HTTP responses against nerve's OpenAPI contract,
// api/dist/openapi.yaml, with kin-openapi. Only tests import it (architecture
// rule 8), so kin-openapi never reaches the nerve binary.
package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
)

// Contract is the loaded and validated OpenAPI document.
type Contract struct {
	doc    *openapi3.T
	router routers.Router
}

// Load reads api/dist/openapi.yaml and fails t unless it is a valid OpenAPI
// document.
func Load(t testing.TB) *Contract {
	t.Helper()
	c, err := load(contractPath())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// CheckResponse fails t unless res, the response to req, is documented by
// req's operation: the status, the Content-Type and the body schema. res.Body
// stays readable for the caller.
func (c *Contract) CheckResponse(t testing.TB, req *http.Request, res *http.Response) {
	t.Helper()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	res.Body = io.NopCloser(bytes.NewReader(body))
	if err := c.validateResponse(req, res.StatusCode, res.Header, body); err != nil {
		t.Errorf("%s %s answered %d %s: %v", req.Method, req.URL.Path, res.StatusCode, body, err)
	}
}

// CheckSchema fails t unless body is a JSON value valid against the schema
// components.schemas[name], e.g. "Problem".
func (c *Contract) CheckSchema(t testing.TB, name string, body []byte) {
	t.Helper()
	if err := c.validateSchema(name, body); err != nil {
		t.Errorf("%s: %v", body, err)
	}
}

// contractPath locates api/dist/openapi.yaml from this file, five directories
// below the repository root (tests run without -trimpath).
func contractPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "api", "dist", "openapi.yaml")
}

func load(path string) (*Contract, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", path, err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate %s: %w", path, err)
	}
	router, err := legacy.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("route %s: %w", path, err)
	}
	return &Contract{doc: doc, router: router}, nil
}

func (c *Contract) validateResponse(req *http.Request, status int, header http.Header, body []byte) error {
	route, pathParams, err := c.router.FindRoute(req)
	if err != nil {
		return fmt.Errorf("no documented operation: %w", err)
	}
	in := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, PathParams: pathParams, Route: route},
		Status:                 status,
		Header:                 header,
		Options:                &openapi3filter.Options{IncludeResponseStatus: true, MultiError: true},
	}
	in.SetBodyBytes(body)
	return openapi3filter.ValidateResponse(context.Background(), in)
}

func (c *Contract) validateSchema(name string, body []byte) error {
	schema, ok := c.doc.Components.Schemas[name]
	if !ok {
		return fmt.Errorf("no schema %q in the contract", name)
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return fmt.Errorf("not JSON: %w", err)
	}
	opts := []openapi3.SchemaValidationOption{openapi3.MultiErrors()}
	if c.doc.IsOpenAPI31OrLater() {
		opts = append(opts, openapi3.EnableJSONSchema2020())
	}
	return schema.Value.VisitJSON(value, opts...)
}
