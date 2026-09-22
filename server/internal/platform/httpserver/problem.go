// Package httpserver provides nerve's HTTP platform: the fixed middleware
// chain, problem+json errors, health endpoints and the server lifecycle.
package httpserver

import (
	"encoding/json"
	"net/http"
)

// ContentTypeProblem is the media type of RFC 9457 problem details.
const ContentTypeProblem = "application/problem+json"

// Codes of the problems the platform itself reports. Module codes are
// namespaced by module, e.g. "issue.state_not_in_project".
const (
	CodeNotFound   = "not_found"
	CodeBadRequest = "bad_request"
	CodeInternal   = "internal_error"
	CodeNotReady   = "not_ready"
)

// Problem is an RFC 9457 problem details body (v0 design, section 3.5).
// Code is the stable identifier clients branch on; Title is for humans.
type Problem struct {
	Status int          `json:"status"`
	Code   string       `json:"code"`
	Title  string       `json:"title"`
	Detail string       `json:"detail,omitempty"`
	Errors []FieldError `json:"errors,omitempty"`
}

// FieldError points at one invalid field of a request.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// WriteProblem sends p with status p.Status as application/problem+json.
func WriteProblem(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", ContentTypeProblem)
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p) // nothing useful to do if the client is gone
}
