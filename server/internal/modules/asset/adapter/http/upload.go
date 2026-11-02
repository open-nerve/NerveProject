package httpadapter

import (
	"net/http"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/localprotocol"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

func (t *Transfer) upload(w http.ResponseWriter, r *http.Request) {
	grant, err := t.deps.UploadVerifier.VerifyUpload(r.PathValue("key"), r.URL.RawQuery, r.Method)
	if err != nil || r.URL.EscapedPath() != localprotocol.Path(grant.Key) {
		transferProblem(w, http.StatusForbidden, "asset.invalid_signature")
		return
	}
	if grant.Size > t.deps.Files.SizeLimit || r.ContentLength > t.deps.Files.SizeLimit {
		transferProblem(w, http.StatusRequestEntityTooLarge, httpserver.CodePayloadTooLarge)
		return
	}
	if r.ContentLength != grant.Size || len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("Content-Type") != grant.ContentType {
		transferProblem(w, http.StatusBadRequest, "asset.invalid_upload")
		return
	}
	release, ok := t.claim(w)
	if !ok {
		return
	}
	defer release()
	budget, err := t.begin(w, r, grant.Size, true)
	if err != nil {
		t.failed(w, r, err)
		return
	}
	defer budget.close()
	r = r.WithContext(budget.ctx)
	body := budgetReader{reader: http.MaxBytesReader(w, r.Body, grant.Size), budget: budget}
	if err := t.deps.Uploader.Put(r.Context(), grant, body); err != nil {
		t.failed(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNoContent)
}
