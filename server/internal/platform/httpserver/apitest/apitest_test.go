package apitest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

const instanceJSON = `{"product":"Nerve","version":"0.1.0-dev","commit":"unknown","api_version":"v0"}`

// Component names become Go and TypeScript type names. Redocly's bundler
// renames a clash between module files to "Name-2" and only warns, so a
// clash has to fail here instead.
func TestComponentNamesAreTypeNames(t *testing.T) {
	c := Load(t)
	valid := regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	var names []string
	for name := range c.doc.Components.Schemas {
		names = append(names, "schemas/"+name)
	}
	for name := range c.doc.Components.Responses {
		names = append(names, "responses/"+name)
	}
	for name := range c.doc.Components.Parameters {
		names = append(names, "parameters/"+name)
	}
	for name := range c.doc.Components.RequestBodies {
		names = append(names, "requestBodies/"+name)
	}
	if len(names) == 0 {
		t.Fatal("the contract has no components")
	}
	for _, name := range names {
		if _, base, _ := strings.Cut(name, "/"); !valid.MatchString(base) {
			t.Errorf("component %s is not a PascalCase type name; two module files may define it differently", name)
		}
	}
}

func TestValidateResponse(t *testing.T) {
	c := Load(t)
	tests := []struct {
		name, method, path string
		status             int
		contentType, body  string
		valid              bool
	}{
		{"documented 200", "GET", "/api/v0/instance", 200, "application/json", instanceJSON, true},
		{"problem as default", "GET", "/api/v0/instance", 500, "application/problem+json", `{"status":500,"code":"internal_error","title":"Internal Server Error"}`, true},
		{"missing field", "GET", "/api/v0/instance", 200, "application/json", `{"product":"Nerve","version":"0.1.0-dev","api_version":"v0"}`, false},
		{"value outside the enum", "GET", "/api/v0/instance", 200, "application/json", strings.Replace(instanceJSON, `"v0"`, `"v1"`, 1), false},
		{"undocumented field", "GET", "/api/v0/instance", 200, "application/json", strings.Replace(instanceJSON, `}`, `,"extra":1}`, 1), false},
		{"undocumented content type", "GET", "/api/v0/instance", 200, "text/plain", "Nerve", false},
		{"problem without code", "GET", "/api/v0/instance", 500, "application/problem+json", `{"status":500,"title":"Internal Server Error"}`, false},
		{"undocumented path", "GET", "/api/v0/nope", 404, "application/problem+json", `{"status":404,"code":"not_found","title":"Not Found"}`, false},
		{"undocumented method", "POST", "/api/v0/instance", 404, "application/problem+json", `{"status":404,"code":"not_found","title":"Not Found"}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			header := http.Header{"Content-Type": {tt.contentType}}
			err := c.validateResponse(req, tt.status, header, []byte(tt.body))
			if (err == nil) != tt.valid {
				t.Errorf("validateResponse() = %v, want valid = %v", err, tt.valid)
			}
		})
	}
}

func TestCheckResponseLeavesTheBodyReadable(t *testing.T) {
	c := Load(t)
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	rec.WriteHeader(http.StatusOK)
	_, _ = rec.WriteString(instanceJSON)
	res := rec.Result()

	c.CheckResponse(t, httptest.NewRequest(http.MethodGet, "/api/v0/instance", nil), res)

	body, err := io.ReadAll(res.Body)
	if err != nil || string(body) != instanceJSON {
		t.Errorf("body after CheckResponse = %q, %v; want %s", body, err, instanceJSON)
	}
}

func TestValidateSchema(t *testing.T) {
	c := Load(t)
	tests := []struct {
		name, schema, body string
		valid              bool
	}{
		{"problem", "Problem", `{"status":404,"code":"not_found","title":"Not Found","detail":"no API endpoint for GET /api/v0/nope"}`, true},
		{"problem with field errors", "Problem", `{"status":422,"code":"issue.invalid","title":"Unprocessable Content","errors":[{"field":"name","message":"is required"}]}`, true},
		{"problem without code", "Problem", `{"status":404,"title":"Not Found"}`, false},
		{"problem with an unknown member", "Problem", `{"status":404,"code":"not_found","title":"Not Found","instance":"/x"}`, false},
		{"unknown schema", "Nope", `{}`, false},
		{"not JSON", "Problem", `not json`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.validateSchema(tt.schema, []byte(tt.body))
			if (err == nil) != tt.valid {
				t.Errorf("validateSchema() = %v, want valid = %v", err, tt.valid)
			}
		})
	}
}
