package local

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func temporaryFixture(t *testing.T, s *Store, n int, at time.Time) string {
	t.Helper()
	path := filepath.Join(s.cfg.Root, "tmp", fmt.Sprintf("00000000-0000-4000-8000-%012x", n))
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "data"), []byte("unfinished"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTemporaryCleanupIsolation(t *testing.T) {
	s := newStorageTestStore(t, t.TempDir())
	other := newStorageTestStore(t, t.TempDir())
	// Deliberately unrelated to the signing clock: only the supplied runtime
	// cutoff and physical directory age may decide reclamation.
	cutoff := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	old := temporaryFixture(t, s, 4, cutoff.Add(-time.Hour))
	equal := temporaryFixture(t, s, 2, cutoff)
	fresh := temporaryFixture(t, s, 3, cutoff.Add(time.Hour))
	first := temporaryFixture(t, s, 1, cutoff.Add(-time.Hour))
	elsewhere := temporaryFixture(t, other, 1, cutoff.Add(-time.Hour))
	key := inventoryObject(t, s, 0)
	token, removed, err := s.SweepTemporary(t.Context(), cutoff, 2, "")
	if err != nil || token == "" || removed != 1 {
		t.Fatalf("first page: %q %d %v", token, removed, err)
	}
	next, removed, err := s.SweepTemporary(t.Context(), cutoff, 2, token)
	if err != nil || next != "" || removed != 1 {
		t.Fatalf("second page: %q %d %v", next, removed, err)
	}
	for _, path := range []string{first, old} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("old temporary remains: %v", err)
		}
	}
	for _, path := range []string{equal, fresh, elsewhere, filepath.Join(s.cfg.Root, "objects", string(key))} {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("protected path lost: %s %v", path, err)
		}
	}
	if _, err := s.List(t.Context(), token, 1); err == nil {
		t.Fatal("temporary cursor accepted for objects")
	}
	if _, _, err := s.SweepTemporary(t.Context(), time.Time{}, 1, ""); err == nil {
		t.Fatal("zero cutoff accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := s.SweepTemporary(ctx, cutoff, 1, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
}

func TestTemporaryCleanupRefusesUnknownLayout(t *testing.T) {
	for _, kind := range []string{"unknown name", "unknown file", "linked directory", "linked file", "nested directory", "tmp symlink"} {
		t.Run(kind, func(t *testing.T) {
			s := newStorageTestStore(t, t.TempDir())
			other := newStorageTestStore(t, t.TempDir())
			cutoff := time.Now().Add(-24 * time.Hour)
			path := temporaryFixture(t, s, 1, cutoff.Add(-time.Hour))
			protected := temporaryFixture(t, other, 1, cutoff.Add(-time.Hour))
			var err error
			switch kind {
			case "unknown name":
				err = os.Rename(path, filepath.Join(s.cfg.Root, "tmp", "unknown"))
			case "unknown file":
				err = os.WriteFile(filepath.Join(path, "unknown"), []byte("keep"), 0600)
			case "linked directory":
				err = os.RemoveAll(path)
				if err == nil {
					err = os.Symlink(protected, path)
				}
			case "linked file":
				err = os.Symlink(filepath.Join(protected, "data"), filepath.Join(path, "meta.json"))
			case "nested directory":
				err = os.Mkdir(filepath.Join(path, "meta.json"), 0700)
			case "tmp symlink":
				err = os.RemoveAll(filepath.Join(s.cfg.Root, "tmp"))
				if err == nil {
					err = os.Symlink(filepath.Join(other.cfg.Root, "tmp"), filepath.Join(s.cfg.Root, "tmp"))
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if kind != "unknown name" && kind != "linked directory" && kind != "tmp symlink" {
				if err := os.Chtimes(path, cutoff.Add(-time.Hour), cutoff.Add(-time.Hour)); err != nil {
					t.Fatal(err)
				}
			}
			if _, n, err := s.SweepTemporary(t.Context(), cutoff, 10, ""); err == nil || n != 0 {
				t.Fatalf("unsafe temporary cleanup: %d %v", n, err)
			}
			if _, err := os.Stat(filepath.Join(protected, "data")); err != nil {
				t.Fatalf("outside root changed: %v", err)
			}
		})
	}
}

func TestTemporaryCleanupReturnsCommittedProgressOnError(t *testing.T) {
	s := newStorageTestStore(t, t.TempDir())
	cutoff := time.Now().Add(-24 * time.Hour)
	first := temporaryFixture(t, s, 1, cutoff.Add(-time.Hour))
	bad := temporaryFixture(t, s, 2, cutoff.Add(-time.Hour))
	if err := os.WriteFile(filepath.Join(bad, "keep"), []byte("unknown"), 0600); err != nil {
		t.Fatal(err)
	}
	next, n, err := s.SweepTemporary(t.Context(), cutoff, 2, "")
	if err == nil || n != 1 || next == "" {
		t.Fatalf("lost partial progress: %q %d %v", next, n, err)
	}
	if _, err := os.Lstat(first); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("first deletion not committed")
	}
	if _, err := os.Lstat(filepath.Join(bad, "keep")); err != nil {
		t.Fatal("unknown file removed")
	}
	if _, n, err := s.SweepTemporary(t.Context(), cutoff, 2, next); err == nil || n != 0 {
		t.Fatalf("failed item was skipped: %d %v", n, err)
	}
}
