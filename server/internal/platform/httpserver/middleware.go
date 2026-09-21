package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
	"uuid"
)

// HeaderRequestID carries the request ID in requests and responses.
const HeaderRequestID = "X-Request-Id"

const maxRequestIDLen = 128

type requestIDKey struct{}

// middleware wraps h in the platform chain. The order is fixed, outermost
// first: request ID -> recover -> access log.
func middleware(h http.Handler, logger *slog.Logger) http.Handler {
	return withRequestID(withRecover(logger, withAccessLog(logger, h)))
}

// requestID returns the ID the request ID middleware assigned to the request.
func requestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// withRequestID keeps the caller's X-Request-Id when it is a safe token and
// otherwise assigns a new UUIDv7. The ID is echoed in the response.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if !validRequestID(id) {
			id = uuid.NewV7().String()
		}
		w.Header().Set(HeaderRequestID, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// validRequestID accepts 1 to 128 characters from [A-Za-z0-9._:-], which
// keeps caller-supplied IDs safe to log.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for _, c := range id {
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		case c == '-', c == '_', c == '.', c == ':':
		default:
			return false
		}
	}
	return true
}

// withRecover turns a panic into a logged 500 problem. Headers the handler set
// are dropped, except the request ID: a Set-Cookie must not leak, and a stale
// Content-Length or Content-Encoding would corrupt the problem body. If the
// response has already started, the connection is aborted instead so the
// client cannot mistake a truncated body for a complete one.
func withRecover(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w}
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler { // deliberate abort: let net/http handle it
				panic(v)
			}
			logger.ErrorContext(r.Context(), "panic serving request",
				slog.String("request_id", requestID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Any("panic", v),
				slog.String("stack", string(debug.Stack())),
			)
			if rec.status != 0 {
				panic(http.ErrAbortHandler)
			}
			header := rec.Header()
			for name := range header {
				if name != HeaderRequestID {
					delete(header, name)
				}
			}
			WriteProblem(rec, Problem{
				Status: http.StatusInternalServerError,
				Code:   CodeInternal,
				Title:  http.StatusText(http.StatusInternalServerError),
			})
		}()
		next.ServeHTTP(rec, r)
	})
}

// withAccessLog logs one line per request: method, path, status, duration
// and request ID. A request whose handler panicked is logged as a 500, the
// answer the recover middleware gives.
func withAccessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		completed := false
		defer func() {
			status := rec.status
			switch {
			case !completed:
				status = http.StatusInternalServerError
			case status == 0:
				status = http.StatusOK
			}
			logger.LogAttrs(r.Context(), slog.LevelInfo, "http request",
				slog.String("request_id", requestID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Duration("duration", time.Since(start)),
			)
		}()
		next.ServeHTTP(rec, r)
		completed = true
	})
}

// statusRecorder remembers the final status code written through it.
type statusRecorder struct {
	http.ResponseWriter
	status int // 0 until a final (non-1xx) status is written
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 && code >= http.StatusOK {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}
