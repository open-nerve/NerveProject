package local

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func candidateKey(t *testing.T) domain.ObjectKey {
	t.Helper()
	key, err := domain.NewObjectKey('o', uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func readObject(t *testing.T, s *Store, info domain.ObjectInfo) string {
	t.Helper()
	reader, err := s.Open(t.Context(), info.Key, info.ETag)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	closeErr := reader.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("read: %v; close: %v", err, closeErr)
	}
	return string(data)
}

// Removing the precondition or publishing before verifying copied bytes must fail.
func TestLocalSealKeepsPublishedBytes(t *testing.T) {
	s := newStorageTestStore(t, t.TempDir())
	grant := uploadGrant(t, s, 6)
	if err := s.Put(t.Context(), grant, strings.NewReader("before")); err != nil {
		t.Fatal(err)
	}
	source, err := s.Inspect(t.Context(), grant.Key)
	if err != nil {
		t.Fatal(err)
	}
	for _, etag := range []string{"", `"wrong"`} {
		target := candidateKey(t)
		if _, err := s.Seal(t.Context(), grant.Key, target, etag); !errors.Is(err, domain.ErrObjectChanged) {
			t.Fatalf("Seal with %q = %v", etag, err)
		}
		if _, err := s.Inspect(t.Context(), target); !errors.Is(err, domain.ErrObjectNotFound) {
			t.Fatalf("failed seal published %s: %v", target, err)
		}
	}
	if _, err := s.Seal(t.Context(), grant.Key, grant.Key, source.ETag); !errors.Is(err, domain.ErrInvalidObjectKey) {
		t.Fatalf("seal to staging = %v", err)
	}
	targets := []domain.ObjectKey{candidateKey(t), candidateKey(t)}
	type result struct {
		info domain.ObjectInfo
		err  error
	}
	completed := make(chan result, 2)
	for _, target := range targets {
		go func() {
			info, err := s.Seal(t.Context(), grant.Key, target, source.ETag)
			completed <- result{info, err}
		}()
	}
	for range targets {
		got := <-completed
		if got.err != nil || got.info.Size != 6 || got.info.ContentType != grant.ContentType || got.info.Key.Kind() != 'o' {
			t.Fatalf("sealed = %+v, %v", got.info, got.err)
		}
		if data := readObject(t, s, got.info); data != "before" {
			t.Fatalf("sealed bytes = %q", data)
		}
	}
	if err := s.Put(t.Context(), grant, strings.NewReader("after!")); !errors.Is(err, domain.ErrObjectExists) {
		t.Fatalf("staging replay = %v", err)
	}
	sealedGrant := grant
	sealedGrant.Key = targets[0]
	if _, err := s.SignUpload(t.Context(), sealedGrant); !errors.Is(err, domain.ErrInvalidUpload) {
		t.Fatalf("signed sealed upload = %v", err)
	}
	if err := s.Put(t.Context(), sealedGrant, strings.NewReader("after!")); !errors.Is(err, domain.ErrInvalidUpload) {
		t.Fatalf("PUT to sealed key = %v", err)
	}
	if _, err := s.Seal(t.Context(), grant.Key, targets[0], source.ETag); !errors.Is(err, domain.ErrObjectExists) {
		t.Fatalf("seal over existing = %v", err)
	}
	first, err := s.Inspect(t.Context(), targets[0])
	if err != nil {
		t.Fatal(err)
	}
	copy, err := s.Seal(t.Context(), first.Key, candidateKey(t), first.ETag)
	if err != nil || readObject(t, s, copy) != "before" {
		t.Fatalf("sealed source copy = %+v, %v", copy, err)
	}
	// Simulate a same-size external change while deliberately retaining old metadata.
	if err := os.WriteFile(filepath.Join(s.cfg.Root, "objects", string(grant.Key), "data"), []byte("after!"), 0600); err != nil {
		t.Fatal(err)
	}
	target := candidateKey(t)
	if _, err := s.Seal(t.Context(), grant.Key, target, source.ETag); !errors.Is(err, domain.ErrObjectChanged) {
		t.Fatalf("same-size changed source = %v", err)
	}
	if _, err := s.Inspect(t.Context(), target); !errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("changed source published = %v", err)
	}
	for _, target := range targets {
		info, err := s.Inspect(t.Context(), target)
		if err != nil || readObject(t, s, info) != "before" {
			t.Fatalf("sealed bytes changed: %v", err)
		}
	}
	noTemporaryFiles(t, s.cfg.Root)
}
