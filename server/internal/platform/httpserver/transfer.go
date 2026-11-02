package httpserver

import "net/http"

// TransferMiddleware reuses the API's client address policy and a dedicated
// bucket. Signed raw handlers own their authentication, body and socket budget.
func (a *API) TransferMiddleware(limiter Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return a.requestMeta(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			if retry, ok := limiter.Allow(RequestMetaFrom(r.Context()).IPKey); !ok {
				a.tooManyRequests(w, r, "file_transfer", retry)
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}
