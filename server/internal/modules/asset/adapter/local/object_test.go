package local

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestLocalObjectConfinement(t *testing.T) {
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(sentinel, []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"../sentinel", "../" + filepath.Base(outside) + "/sentinel", "u-%2f-sentinel", sentinel} {
		s := newStorageTestStore(t, t.TempDir())
		key := domain.ObjectKey(raw)
		if _, err := s.Inspect(t.Context(), key); !errors.Is(err, domain.ErrInvalidObjectKey) {
			t.Fatalf("Inspect invalid = %v", err)
		}
		if _, err := s.Open(t.Context(), key, "etag"); !errors.Is(err, domain.ErrInvalidObjectKey) {
			t.Fatalf("Open invalid = %v", err)
		}
		if err := s.Delete(t.Context(), key); !errors.Is(err, domain.ErrInvalidObjectKey) {
			t.Fatalf("Delete invalid = %v", err)
		}
		grant := uploadGrant(t, s, 3)
		grant.Key = key
		if err := s.Put(t.Context(), grant, strings.NewReader("bad")); !errors.Is(err, domain.ErrInvalidUpload) {
			t.Fatalf("Put invalid = %v", err)
		}
	}
	t.Run("object directory cannot be an outside symlink", func(t *testing.T) {
		s := newStorageTestStore(t, t.TempDir())
		grant := uploadGrant(t, s, 3)
		if err := os.Symlink(outside, filepath.Join(s.cfg.Root, "objects", string(grant.Key))); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Inspect(t.Context(), grant.Key); err == nil || strings.Contains(err.Error(), outside) {
			t.Fatalf("Inspect symlink = %v", err)
		}
		if err := s.Delete(t.Context(), grant.Key); err == nil || strings.Contains(err.Error(), outside) {
			t.Fatalf("Delete symlink = %v", err)
		}
		if err := s.Put(t.Context(), grant, strings.NewReader("bad")); err == nil {
			t.Fatal("upload followed symlink")
		}
	})
	for _, kind := range []string{"outside symlink", "directory", "fifo"} {
		t.Run("data "+kind, func(t *testing.T) {
			s := newStorageTestStore(t, t.TempDir())
			grant := uploadGrant(t, s, 3)
			if err := s.Put(t.Context(), grant, strings.NewReader("abc")); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(s.cfg.Root, "objects", string(grant.Key), "data")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			var err error
			switch kind {
			case "outside symlink":
				err = os.Symlink(sentinel, path)
			case "directory":
				err = os.Mkdir(path, 0700)
			case "fifo":
				err = syscall.Mkfifo(path, 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { _, err := s.Inspect(t.Context(), grant.Key); done <- err }()
			select {
			case err := <-done:
				if err == nil || errors.Is(err, domain.ErrObjectNotFound) || strings.Contains(err.Error(), outside) {
					t.Fatalf("nonregular data = %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("nonregular data blocked Inspect")
			}
		})
	}
	if got, err := os.ReadFile(sentinel); err != nil || string(got) != "keep me" {
		t.Fatalf("outside sentinel changed: %q, %v", got, err)
	}
}

func TestLocalConcurrentUploadsDoNotReplaceOrShareTemporaryPaths(t *testing.T) {
	root := t.TempDir()
	stores := []*Store{newStorageTestStore(t, root), newStorageTestStore(t, root)}
	grant := uploadGrant(t, stores[0], 6)
	writers := make([]*io.PipeWriter, 2)
	results := make([]chan error, 2)
	for i, s := range stores {
		reader, writer := io.Pipe()
		writers[i] = writer
		results[i] = make(chan error, 1)
		t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
		go func() { err := s.Put(t.Context(), grant, reader); _ = reader.CloseWithError(err); results[i] <- err }()
		if _, err := writer.Write([]byte(strings.Repeat(string(rune('a'+i)), 3))); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, "tmp"))
	if err != nil || len(entries) != 2 || entries[0].Name() == entries[1].Name() {
		t.Fatalf("independent temporary paths = %v, %v", entries, err)
	}
	for i, writer := range writers {
		if _, err := writer.Write([]byte(strings.Repeat(string(rune('a'+i)), 3))); err != nil {
			t.Fatal(err)
		}
		_ = writer.Close()
	}
	winner := -1
	for i, result := range results {
		err := <-result
		if err == nil {
			if winner >= 0 {
				t.Fatal("both uploads replaced the same object")
			}
			winner = i
		} else if !errors.Is(err, domain.ErrObjectExists) {
			t.Fatal(err)
		}
	}
	if winner < 0 {
		t.Fatal("no upload won")
	}
	info, err := stores[0].Inspect(t.Context(), grant.Key)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := stores[1].Open(t.Context(), grant.Key, info.ETag)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || string(body) != strings.Repeat(string(rune('a'+winner)), 6) {
		t.Fatalf("winner bytes = %q, %v", body, err)
	}
	if err := stores[0].Put(t.Context(), grant, strings.NewReader("cccccc")); !errors.Is(err, domain.ErrObjectExists) {
		t.Fatalf("replay = %v", err)
	}
	noTemporaryFiles(t, root)
}

func TestLocalObjectLifetimeAndFailures(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0755); err != nil {
		t.Fatal(err)
	}
	s := newStorageTestStore(t, root)
	grant := uploadGrant(t, s, 3)
	st, err := os.Stat(root)
	if err != nil || st.Mode().Perm() != 0755 {
		t.Fatalf("existing root mode changed: %v %v", st, err)
	}
	if err := s.Delete(t.Context(), grant.Key); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(t.Context(), grant, strings.NewReader("abc")); err != nil {
		t.Fatal(err)
	}
	info, err := s.Inspect(t.Context(), grant.Key)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	reader, err := s.Open(ctx, grant.Key, info.ETag)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := reader.Read(make([]byte, 1)); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled reader = %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(t.Context(), grant.Key); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(t.Context(), grant.Key); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Inspect(t.Context(), grant.Key); !errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("deleted object = %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Inspect(t.Context(), grant.Key); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("closed store = %v", err)
	}
	if err := s.Delete(t.Context(), grant.Key); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("closed deletion silently succeeded: %v", err)
	}
}

func TestLocalRootRejectsUnexpectedEntries(t *testing.T) {
	for _, name := range []string{"objects", "tmp"} {
		t.Run(name, func(t *testing.T) {
			root, outside := t.TempDir(), t.TempDir()
			if err := os.Symlink(outside, filepath.Join(root, name)); err != nil {
				t.Fatal(err)
			}
			now := time.Now()
			store, err := New(Config{Root: root, SizeLimit: 5242880}, Deps{UploadMAC: testMAC("up"), ReadMAC: testMAC("read"), RuntimeNow: func() time.Time { return now }})
			if err == nil {
				_ = store.Close()
				t.Fatal("constructor accepted a symlink as a storage directory")
			}
			if strings.Contains(err.Error(), root) || strings.Contains(err.Error(), outside) {
				t.Fatalf("constructor leaked a path: %v", err)
			}
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 0 {
				t.Fatalf("outside directory changed: %v, %v", entries, err)
			}
		})
	}
}
