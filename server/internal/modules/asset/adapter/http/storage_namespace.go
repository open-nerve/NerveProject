package httpadapter

import (
	"net/http"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/localprotocol"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// The reserved namespace survives switching storage drivers. Unknown transfer
// paths must not fall through to workspace pages or redirect signed queries.
func RegisterStorageNamespace(router *httpserver.Router, middleware func(http.Handler) http.Handler) {
	router.RejectPathRedirects(localprotocol.Root)
	for _, pattern := range []string{localprotocol.Root, localprotocol.Root + "/"} {
		router.Handle(pattern, middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			transferProblem(w, http.StatusNotFound, httpserver.CodeNotFound)
		})))
	}
}
