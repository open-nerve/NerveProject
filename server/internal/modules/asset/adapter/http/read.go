package httpadapter

import (
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/localprotocol"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

var singleByteRange = regexp.MustCompile(`^bytes=([0-9]+-[0-9]*|-[0-9]+)$`)

func (t *Transfer) read(w http.ResponseWriter, r *http.Request) {
	grant, err := t.deps.ReadVerifier.VerifyRead(r.PathValue("key"), r.URL.RawQuery, r.Method)
	if err != nil || r.URL.EscapedPath() != localprotocol.Path(grant.Key) {
		transferProblem(w, http.StatusForbidden, "asset.invalid_signature")
		return
	}
	release, ok := t.claim(w)
	if !ok {
		return
	}
	defer release()
	info, reader, err := t.deps.Reader.OpenTransfer(r.Context(), grant.Key)
	if err != nil {
		t.failed(w, r, err)
		return
	}
	defer func() { _ = reader.Close() }()
	budget, err := t.begin(w, r, info.Size, false)
	if err != nil {
		t.failed(w, r, err)
		return
	}
	defer budget.close()
	r = r.WithContext(budget.ctx)
	w = &budgetWriter{ResponseWriter: w, budget: budget}
	if info.ContentType != grant.ContentType {
		t.failed(w, r, domain.ErrObjectChanged)
		return
	}
	if values := r.Header.Values("Range"); len(values) != 0 {
		value := values[0]
		if len(values) != 1 || len(value) > 128 || !singleByteRange.MatchString(value) || strings.TrimLeft(strings.TrimPrefix(value, "bytes=-"), "0") == "" {
			w.Header().Set("Content-Range", "bytes */"+strconv.FormatInt(info.Size, 10))
			transferProblem(w, http.StatusRequestedRangeNotSatisfiable, "asset.invalid_range")
			return
		}
	}
	disposition := "attachment"
	if grant.Inline {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": grant.Name}))
	w.Header().Set("Content-Type", grant.ContentType)
	w.Header().Set("ETag", info.ETag)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	remaining := max(int64(grant.ExpiresAt.Sub(t.deps.RuntimeNow())/time.Second), 0)
	w.Header().Set("Cache-Control", "private, max-age="+strconv.FormatInt(remaining, 10))
	response := &contentResponse{ResponseWriter: w}
	http.ServeContent(response, r, "", info.ModifiedAt, reader)
	if response.status >= 200 && response.status < 300 && r.Method != http.MethodHead {
		expected, err := strconv.ParseInt(response.Header().Get("Content-Length"), 10, 64)
		if err != nil || response.written != expected || response.writeErr != nil {
			// ServeContent does not report CopyN errors. A truncated response must
			// abort the connection, never acquire a JSON error tail.
			panic(http.ErrAbortHandler)
		}
	}
}

// contentResponse retains ServeContent's standard conditional/range behavior,
// while converting its pre-body errors into our fixed no-store Problem format.
type contentResponse struct {
	http.ResponseWriter
	status   int
	written  int64
	writeErr error
}

func (w *contentResponse) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *contentResponse) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	if status >= 400 {
		code := httpserver.CodeInternal
		if status == http.StatusRequestedRangeNotSatisfiable {
			code = "asset.invalid_range"
		}
		if status == http.StatusPreconditionFailed {
			code = "asset.precondition_failed"
		}
		transferProblem(w.ResponseWriter, status, code)
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *contentResponse) Write(p []byte) (int, error) {
	if w.status >= 400 {
		return len(p), nil
	}
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(p)
	w.written += int64(n)
	if err != nil {
		w.writeErr = err
	}
	return n, err
}
