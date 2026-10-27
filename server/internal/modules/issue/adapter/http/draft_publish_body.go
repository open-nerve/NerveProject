package httpadapter

import (
	"errors"
	"io"
	"net/http"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// draftPublishBody is inside the API authentication/limit chain. No request
// bytes may override or supplement the stored payload, even with unknown length.
func draftPublishBody(api *httpserver.API) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.Pattern == "POST /api/v0/drafts/{draft_id}/publish" {
				var first [1]byte
				n, err := r.Body.Read(first[:])
				if n != 0 || !errors.Is(err, io.EOF) {
					if err == nil || n != 0 {
						err = errors.New("publish accepts no request body")
					}
					api.Errors.BodyError(w, r, err)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
