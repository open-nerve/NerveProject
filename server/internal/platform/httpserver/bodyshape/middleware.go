package bodyshape

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// ErrNotJSON is answered for a body that is not one valid JSON document.
var ErrNotJSON = errors.New("the request body is not valid JSON")

// Middleware checks the body of every request whose route has a root in t,
// before the generated strict handler decodes it. It reads the whole body
// (already bounded by the platform's body limit) and puts it back unchanged.
//
// onError answers a body that could not be read (the read error, e.g.
// *http.MaxBytesError), that is not JSON (ErrNotJSON) or that breaks the
// structure (*Error). An empty body, or one of only JSON whitespace, is passed
// on: the generated decoder answers it.
func Middleware(t *Table, onError func(http.ResponseWriter, *http.Request, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := t.Roots[r.Pattern]; !ok || r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				onError(w, r, err)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			if len(bytes.Trim(body, jsonSpace)) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			if !json.Valid(body) {
				onError(w, r, ErrNotJSON)
				return
			}
			if fields := t.Check(r.Pattern, body); len(fields) > 0 {
				onError(w, r, &Error{Fields: fields})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
