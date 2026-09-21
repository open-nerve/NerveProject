package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// readinessTimeout bounds one /readyz evaluation, so a hung dependency
// answers 503 instead of hanging the probe.
const readinessTimeout = 2 * time.Second

// Check is one readiness condition, such as "the database answers".
type Check struct {
	Name string
	Run  func(ctx context.Context) error
}

// NewMux returns a router holding the platform routes:
//
//   - GET /healthz: liveness; never touches a dependency.
//   - GET /readyz: runs checks in order; the first failure answers 503.
//   - /api/: every API path no module handles answers 404 problem+json,
//     never the web UI.
//
// Modules and the web UI register their own routes on the returned mux.
func NewMux(logger *slog.Logger, checks ...Check) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeStatusOK(w)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()
		for _, c := range checks {
			if err := c.Run(ctx); err != nil {
				logger.WarnContext(ctx, "readiness check failed",
					slog.String("request_id", requestID(ctx)),
					slog.String("check", c.Name),
					slog.Any("error", err),
				)
				WriteProblem(w, Problem{
					Status: http.StatusServiceUnavailable,
					Code:   CodeNotReady,
					Title:  http.StatusText(http.StatusServiceUnavailable),
					Detail: c.Name + " is not ready",
				})
				return
			}
		}
		writeStatusOK(w)
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		WriteProblem(w, Problem{
			Status: http.StatusNotFound,
			Code:   CodeNotFound,
			Title:  http.StatusText(http.StatusNotFound),
			Detail: "no API endpoint for " + r.Method + " " + r.URL.Path,
		})
	})
	return mux
}

func writeStatusOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
