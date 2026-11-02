package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// A directory stream exposes the same fixed-size boundary as os.File.ReadDir,
// with cancellation during the read rather than just before entering the API.
type inventoryStream struct {
	left, calls, total int
	cancel             context.CancelFunc
}

func (s *inventoryStream) ReadDir(n int) ([]fs.DirEntry, error) {
	s.calls++
	if n != 256 {
		return nil, errors.New("unbounded directory read")
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.left == 0 {
		return nil, io.EOF
	}
	out := make([]fs.DirEntry, 0, min(n, s.left))
	for len(out) < n && s.left > 0 {
		i := (s.total - s.left) * 7919 % s.total
		out = append(out, inventoryEntry(fmt.Sprintf("%08d", i)))
		s.left--
	}
	return out, nil
}

type inventoryEntry string

func (e inventoryEntry) Name() string      { return string(e) }
func (e inventoryEntry) IsDir() bool       { return true }
func (e inventoryEntry) Type() fs.FileMode { return fs.ModeDir }
func (e inventoryEntry) Info() (fs.FileInfo, error) {
	return nil, errors.New("not required to enumerate names")
}

func TestInventoryRetainsBoundedPage(t *testing.T) {
	s := &inventoryStream{left: 10000, total: 10000}
	names, err := directoryPage(t.Context(), s, "00004000", 3, func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if cap(names) > 4 || !slices.Equal(names, []string{"00004001", "00004002", "00004003", "00004004"}) {
		t.Fatalf("unbounded or wrong page: len=%d cap=%d", len(names), cap(names))
	}
	if s.calls != 41 {
		t.Fatalf("did not scan finite chunks: %d", s.calls)
	}
}

func TestInventoryCancellationDuringRead(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s := &inventoryStream{left: 10000, total: 10000, cancel: cancel}
	names, err := directoryPage(ctx, s, "", 3, func(string) bool { return true })
	if !errors.Is(err, context.Canceled) || len(names) != 0 || s.calls != 1 {
		t.Fatalf("continued after cancellation: names=%d reads=%d err=%v", len(names), s.calls, err)
	}
}

func TestInventoryRootRestriction(t *testing.T) {
	s := newStorageTestStore(t, t.TempDir())
	if err := os.Symlink(t.TempDir(), filepath.Join(s.cfg.Root, "elsewhere")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"elsewhere", "../"} {
		dir, err := openInventoryDirectory(s.root, path)
		if dir != nil {
			_ = dir.Close()
		}
		if err == nil {
			t.Fatalf("inventory opened outside its root: %q", path)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.List(t.Context(), "", 1); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("closed inventory = %v", err)
	}
}
