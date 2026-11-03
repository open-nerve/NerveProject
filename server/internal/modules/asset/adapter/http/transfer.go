// Package httpadapter serves the asset module's signed raw transfers.
package httpadapter

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/localprotocol"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

type UploadVerifier interface {
	VerifyUpload(string, string, string) (domain.UploadGrant, error)
}
type ReadVerifier interface {
	VerifyRead(string, string, string) (domain.ReadGrant, error)
}
type Uploader interface {
	Put(context.Context, domain.UploadGrant, io.Reader) error
}

// TransferReader is the local HTTP consumer's seekable read, separate from the
// asset use cases' streaming ObjectReader (also implemented by remote backends).
type TransferReader interface {
	OpenTransfer(context.Context, domain.ObjectKey) (domain.ObjectInfo, io.ReadSeekCloser, error)
}
type ErrorWriter interface {
	Write(http.ResponseWriter, *http.Request, error)
}

type Deps struct {
	UploadVerifier UploadVerifier
	ReadVerifier   ReadVerifier
	Uploader       Uploader
	Reader         TransferReader
	Files          config.FilesConfig
	RuntimeNow     func() time.Time
	Errors         ErrorWriter
}

type Transfer struct {
	deps  Deps
	slots chan struct{}
}

func NewTransfer(deps Deps) (*Transfer, error) {
	for _, dep := range []any{deps.UploadVerifier, deps.ReadVerifier, deps.Uploader, deps.Reader, deps.Errors} {
		if dep == nil {
			return nil, errors.New("asset transfer lacks a dependency")
		}
		switch v := reflect.ValueOf(dep); v.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Interface, reflect.Chan:
			if v.IsNil() {
				return nil, errors.New("asset transfer lacks a dependency")
			}
		}
	}
	if deps.RuntimeNow == nil || deps.Files.SizeLimit < 1 || deps.Files.SizeLimit > domain.MaxObjectSize {
		return nil, errors.New("invalid asset transfer configuration")
	}
	if _, err := deps.Files.TransferBudget(deps.Files.SizeLimit); err != nil || deps.Files.TransferIdleTimeout <= 0 || deps.Files.TransferIdleTimeout > time.Minute || deps.Files.MaxConcurrentTransfers < 1 || deps.Files.MaxConcurrentTransfers > 256 {
		return nil, errors.New("invalid asset transfer budget configuration")
	}
	return &Transfer{deps: deps, slots: make(chan struct{}, deps.Files.MaxConcurrentTransfers)}, nil
}

func (t *Transfer) Register(router *httpserver.Router, middleware func(http.Handler) http.Handler) {
	RegisterStorageNamespace(router, middleware)
	router.Handle("PUT "+localprotocol.ObjectPrefix+"{key}", middleware(http.HandlerFunc(t.upload)))
	router.Handle("GET "+localprotocol.ObjectPrefix+"{key}", middleware(http.HandlerFunc(t.read)))
	router.Handle("HEAD "+localprotocol.ObjectPrefix+"{key}", middleware(http.HandlerFunc(t.read)))
}

func transferProblem(w http.ResponseWriter, status int, code string) {
	clearContentHeaders(w.Header())
	httpserver.WriteProblem(w, httpserver.Problem{Status: status, Code: code, Title: http.StatusText(status)})
}

func clearContentHeaders(header http.Header) {
	for _, key := range []string{"Content-Length", "Content-Type", "Content-Disposition", "Content-Encoding", "ETag", "Last-Modified", "Accept-Ranges"} {
		header.Del(key)
	}
	header.Set("Cache-Control", "no-store")
	header.Set("X-Content-Type-Options", "nosniff")
}

func (t *Transfer) failed(w http.ResponseWriter, r *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	var timeout net.Error
	switch {
	case errors.Is(err, domain.ErrInvalidUpload), r.Method == http.MethodPut && errors.Is(err, io.ErrUnexpectedEOF):
		transferProblem(w, http.StatusBadRequest, "asset.invalid_upload")
	case errors.Is(err, domain.ErrObjectNotFound):
		transferProblem(w, http.StatusNotFound, "asset.object_not_found")
	case errors.Is(err, domain.ErrObjectExists):
		transferProblem(w, http.StatusConflict, "asset.object_exists")
	case errors.Is(err, domain.ErrObjectChanged):
		transferProblem(w, http.StatusConflict, "asset.object_changed")
	case errors.As(err, &tooLarge):
		transferProblem(w, http.StatusRequestEntityTooLarge, httpserver.CodePayloadTooLarge)
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &timeout) && timeout.Timeout():
		transferProblem(w, http.StatusRequestTimeout, "asset.transfer_timeout")
	default:
		clearContentHeaders(w.Header())
		t.deps.Errors.Write(w, r, err)
	}
}
