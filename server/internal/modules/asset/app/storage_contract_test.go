package app_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/storagetest"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// This is a test consumer, not a production-wide Storage interface. Its opaque
// version assertions apply to both local and future S3 adapters.
type contractObjects interface {
	app.ObjectInspector
	app.ObjectReader
	app.ObjectSealer
	app.ObjectDeleter
}

func TestStorageContract(t *testing.T) {
	now := time.Now()
	s, err := local.New(local.Config{Root: t.TempDir(), SizeLimit: 5242880}, local.Deps{
		UploadMAC: storagetest.MAC("contract-upload"), ReadMAC: storagetest.MAC("contract-read"), RuntimeNow: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	logger := slog.New(slog.DiscardHandler)
	files := config.FilesConfig{SizeLimit: 5242880, MinTransferBytesPerSecond: 131072, TransferGrace: 15 * time.Second, TransferIdleTimeout: 30 * time.Second, MaxTransferDuration: 15 * time.Minute, MaxConcurrentTransfers: 16}
	transfer, err := httpadapter.NewTransfer(httpadapter.Deps{UploadVerifier: s, ReadVerifier: s, Uploader: s, Reader: s, RuntimeNow: func() time.Time { return now }, Files: files, Errors: httpserver.NewAPIErrors(logger)})
	if err != nil {
		t.Fatal(err)
	}
	router := httpserver.NewRouter(logger)
	transfer.Register(router, func(h http.Handler) http.Handler { return h })
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	put := func(ctx context.Context, grant domain.UploadGrant, body io.Reader) error {
		signed, err := s.SignUpload(ctx, grant)
		if err != nil {
			return err
		}
		request, err := http.NewRequestWithContext(ctx, "PUT", server.URL+signed.URL, body)
		if err != nil {
			return err
		}
		request.ContentLength = grant.Size
		for key, value := range signed.Headers {
			request.Header.Set(key, value)
		}
		response, err := server.Client().Do(request)
		if err != nil {
			return err
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != 204 {
			return fmt.Errorf("HTTP upload returned %d", response.StatusCode)
		}
		return nil
	}
	checkHTTPRead := func(info domain.ObjectInfo) {
		for _, method := range []string{"GET", "HEAD"} {
			signed, err := s.SignRead(t.Context(), domain.ReadGrant{Key: info.Key, Method: method, Name: "hello.txt", ContentType: info.ContentType, ExpiresAt: now.Add(time.Minute)})
			if err != nil {
				t.Fatal(err)
			}
			request, err := http.NewRequestWithContext(t.Context(), method, server.URL+signed.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			want := "hello"
			if method == "HEAD" {
				want = ""
			}
			if err != nil || closeErr != nil || response.StatusCode != 200 || string(data) != want || response.ContentLength != 5 || response.Header.Get("ETag") != info.ETag {
				t.Fatalf("HTTP %s = %d %q %v / %v", method, response.StatusCode, data, err, closeErr)
			}
		}
	}
	runStorageContract(t, s, put, checkHTTPRead, now)
}

func runStorageContract(t *testing.T, objects contractObjects, put func(context.Context, domain.UploadGrant, io.Reader) error, checkHTTPRead func(domain.ObjectInfo), now time.Time) {
	t.Helper()
	key := func(kind byte) domain.ObjectKey {
		k, err := domain.NewObjectKey(kind, uuid.New(), uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		return k
	}
	staging, candidate := key('u'), key('o')
	if err := objects.Delete(t.Context(), candidate); err != nil {
		t.Fatal(err)
	}
	if _, err := objects.Inspect(t.Context(), candidate); !errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("missing inspect = %v", err)
	}
	if _, err := objects.Open(t.Context(), candidate, ""); !errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("missing open = %v", err)
	}
	if _, err := objects.Seal(t.Context(), staging, candidate, ""); !errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("missing source = %v", err)
	}
	grant := domain.UploadGrant{Key: staging, Size: 5, ContentType: "text/plain", ExpiresAt: now.Add(time.Minute)}
	if err := put(t.Context(), grant, strings.NewReader("hello")); err != nil {
		t.Fatal(err)
	}
	info, err := objects.Inspect(t.Context(), staging)
	if err != nil || info.Size != 5 || info.ContentType != "text/plain" || info.ETag == "" {
		t.Fatalf("uploaded info = %+v, %v", info, err)
	}
	if _, err := objects.Seal(t.Context(), staging, candidate, "wrong-version"); !errors.Is(err, domain.ErrObjectChanged) {
		t.Fatalf("wrong version seal = %v", err)
	}
	sealed, err := objects.Seal(t.Context(), staging, candidate, info.ETag)
	if err != nil || sealed.Key != candidate || sealed.Size != 5 || sealed.ContentType != "text/plain" || sealed.ETag == "" {
		t.Fatalf("sealed info = %+v, %v", sealed, err)
	}
	if err := objects.Delete(t.Context(), staging); err != nil {
		t.Fatal(err)
	}
	checkHTTPRead(sealed)
	if _, err := objects.Open(t.Context(), candidate, "wrong-version"); !errors.Is(err, domain.ErrObjectChanged) {
		t.Fatalf("wrong version read = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	reader, err := objects.Open(ctx, candidate, sealed.ETag)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err != nil || string(data) != "hello" {
		t.Fatalf("read sealed = %q, %v", data, err)
	}
	cancel()
	if _, err := reader.Read(make([]byte, 1)); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled reader = %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := objects.Inspect(ctx, candidate); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled inspect = %v", err)
	}
	if _, err := objects.Seal(ctx, candidate, key('o'), sealed.ETag); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled seal = %v", err)
	}
	if err := objects.Delete(ctx, candidate); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled delete = %v", err)
	}
	if err := objects.Delete(t.Context(), candidate); err != nil {
		t.Fatal(err)
	}
	if err := objects.Delete(t.Context(), candidate); err != nil {
		t.Fatal(err)
	}
	if _, err := objects.Inspect(t.Context(), candidate); !errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("deleted inspect = %v", err)
	}
}
