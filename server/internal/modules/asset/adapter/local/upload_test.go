package local

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func newStorageTestStore(t *testing.T, root string) *Store {
	t.Helper()
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	s, err := New(Config{Root: root, SizeLimit: 5242880}, Deps{UploadMAC: testMAC("upload-test-key"), ReadMAC: testMAC("read-test-key"), RuntimeNow: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}

func uploadGrant(t *testing.T, s *Store, size int64) domain.UploadGrant {
	t.Helper()
	return domain.UploadGrant{Key: signingKey(t, 'u'), Size: size, ContentType: "application/octet-stream", ExpiresAt: s.deps.RuntimeNow().Add(time.Minute)}
}

func noTemporaryFiles(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, "tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary objects remain: %v", entries)
	}
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestLocalUploadPublishesOnlyCompleteObjects(t *testing.T) {
	t.Run("in progress then complete", func(t *testing.T) {
		s := newStorageTestStore(t, t.TempDir())
		grant := uploadGrant(t, s, 6)
		reader, writer := io.Pipe()
		t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
		done := make(chan error, 1)
		go func() { err := s.Put(t.Context(), grant, reader); _ = reader.CloseWithError(err); done <- err }()
		if _, err := writer.Write([]byte("abc")); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Inspect(t.Context(), grant.Key); !errors.Is(err, domain.ErrObjectNotFound) {
			t.Fatalf("partial object visible: %v", err)
		}
		if _, err := writer.Write([]byte("def")); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		info, err := s.Inspect(t.Context(), grant.Key)
		if err != nil {
			t.Fatal(err)
		}
		wantETag := fmt.Sprintf(`"%x"`, sha256.Sum256([]byte("abcdef")))
		if info.Key != grant.Key || info.Size != 6 || info.ContentType != grant.ContentType || info.ETag != wantETag || !info.ModifiedAt.Equal(s.deps.RuntimeNow()) {
			t.Fatalf("metadata = %+v", info)
		}
		opened, err := s.Open(t.Context(), grant.Key, info.ETag)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(opened)
		closeErr := opened.Close()
		if err != nil || closeErr != nil || string(body) != "abcdef" {
			t.Fatalf("read = %q, %v, close %v", body, err, closeErr)
		}
		if _, err := s.Open(t.Context(), grant.Key, `"wrong"`); !errors.Is(err, domain.ErrObjectChanged) {
			t.Fatalf("wrong version = %v", err)
		}
		for _, path := range []string{"objects", "tmp", "objects/" + string(grant.Key)} {
			st, err := os.Stat(filepath.Join(s.cfg.Root, path))
			if err != nil || !st.IsDir() || st.Mode().Perm() != 0700 {
				t.Fatalf("directory %s = %v, %v", path, st, err)
			}
		}
		for _, path := range []string{"data", "meta.json"} {
			st, err := os.Stat(filepath.Join(s.cfg.Root, "objects", string(grant.Key), path))
			if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 {
				t.Fatalf("file %s = %v, %v", path, st, err)
			}
		}
		noTemporaryFiles(t, s.cfg.Root)
	})
	for _, tc := range []struct {
		name   string
		reader func() io.Reader
		size   int64
		want   error
	}{
		{"short", func() io.Reader { return strings.NewReader("ab") }, 3, domain.ErrInvalidUpload},
		{"long", func() io.Reader { return strings.NewReader("abcd") }, 3, domain.ErrInvalidUpload},
		{"reader error after exact bytes", func() io.Reader { return io.MultiReader(strings.NewReader("abc"), failingReader{io.ErrUnexpectedEOF}) }, 3, io.ErrUnexpectedEOF},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newStorageTestStore(t, t.TempDir())
			grant := uploadGrant(t, s, tc.size)
			if err := s.Put(t.Context(), grant, tc.reader()); !errors.Is(err, tc.want) {
				t.Fatalf("Put = %v; want %v", err, tc.want)
			}
			if _, err := s.Inspect(t.Context(), grant.Key); !errors.Is(err, domain.ErrObjectNotFound) {
				t.Fatalf("failed write published: %v", err)
			}
			noTemporaryFiles(t, s.cfg.Root)
		})
	}
	t.Run("canceled blocked source", func(t *testing.T) {
		s := newStorageTestStore(t, t.TempDir())
		grant := uploadGrant(t, s, 6)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		reader, writer := io.Pipe()
		defer func() { _ = reader.Close() }()
		defer func() { _ = writer.Close() }()
		done := make(chan error, 1)
		go func() { err := s.Put(ctx, grant, reader); _ = reader.CloseWithError(err); done <- err }()
		if _, err := writer.Write([]byte("abc")); err != nil {
			t.Fatal(err)
		}
		cancel()
		_ = writer.CloseWithError(context.Canceled) // The source owner unblocks its own I/O.
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("Put canceled = %v", err)
		}
		if _, err := s.Inspect(t.Context(), grant.Key); !errors.Is(err, domain.ErrObjectNotFound) {
			t.Fatalf("canceled object published: %v", err)
		}
		noTemporaryFiles(t, s.cfg.Root)
	})
	t.Run("filesystem refuses temporary directory", func(t *testing.T) {
		s := newStorageTestStore(t, t.TempDir())
		grant := uploadGrant(t, s, 3)
		tmp := filepath.Join(s.cfg.Root, "tmp")
		if err := os.Remove(tmp); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tmp, []byte("not a directory"), 0600); err != nil {
			t.Fatal(err)
		}
		err := s.Put(t.Context(), grant, bytes.NewReader([]byte("abc")))
		if err == nil || errors.Is(err, domain.ErrObjectNotFound) || strings.Contains(err.Error(), s.cfg.Root) {
			t.Fatalf("backend error lost or leaked: %v", err)
		}
		if _, err := s.Inspect(t.Context(), grant.Key); !errors.Is(err, domain.ErrObjectNotFound) {
			t.Fatalf("failed disk write published: %v", err)
		}
	})
}
