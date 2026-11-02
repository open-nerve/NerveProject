package app_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/storagetest"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
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
	// T5 replaces this transport with actual HTTP PUT against the registered handler.
	runStorageContract(t, s, s.Put, now)
}

func runStorageContract(t *testing.T, objects contractObjects, put func(context.Context, domain.UploadGrant, io.Reader) error, now time.Time) {
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
