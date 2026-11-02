package local

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

// Create the real published layout without fsync: these are inventory fixtures,
// not evidence that uploads have committed. Upload durability has its own tests.
func inventoryObject(t *testing.T, s *Store, i int) domain.ObjectKey {
	t.Helper()
	k, err := domain.ParseObjectKey(fmt.Sprintf("o-%032x-%032x", i+1, 10001-i))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(s.cfg.Root, "objects", string(k))
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	meta := objectMetadata{Version: 1, Key: k, Size: 1, ContentType: "application/octet-stream", ETag: fmt.Sprintf(`"%x"`, sha256.Sum256([]byte("x"))), StoredAt: s.deps.RuntimeNow()}
	b, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"data": []byte("x"), "meta.json": b} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return k
}

func TestLocalInventoryPages(t *testing.T) {
	s := newStorageTestStore(t, t.TempDir())
	const count = 10000
	want := make([]domain.ObjectKey, count)
	// A coprime stride creates every ID, with neither insertion nor enumeration
	// order matching the expected lexical cursor order.
	for n := range count {
		i := n * 7919 % count
		want[i] = inventoryObject(t, s, i)
	}
	slices.Sort(want)
	var got []domain.ObjectKey
	token, pages := "", 0
	start := time.Now()
	for {
		pageStart := time.Now()
		page, err := s.List(t.Context(), token, 1000)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Objects) != 1000 {
			t.Fatalf("page length %d", len(page.Objects))
		}
		for _, obj := range page.Objects {
			if obj.Size != 1 || obj.ContentType != "application/octet-stream" || !obj.ModifiedAt.Equal(s.deps.RuntimeNow()) || obj.ETag == "" {
				t.Fatalf("bad object %+v", obj)
			}
			got = append(got, obj.Key)
		}
		pages++
		t.Logf("10000 directories: page %d, 1000 results, %s", pages, time.Since(pageStart))
		if page.Next == "" {
			break
		}
		if page.Next == token || pages > 10 {
			t.Fatal("cursor did not advance")
		}
		token = page.Next
	}
	if pages != 10 || !slices.Equal(got, want) {
		t.Fatalf("coverage: pages=%d results=%d", pages, len(got))
	}
	t.Logf("full 10000 object inventory: %s", time.Since(start))
}

func TestLocalInventoryRejectsBadInputsAndLayout(t *testing.T) {
	for _, tc := range []string{"unknown", "symlink", "empty", "bad metadata", "linked data", "unknown file", "objects symlink"} {
		t.Run(tc, func(t *testing.T) {
			s := newStorageTestStore(t, t.TempDir())
			key := inventoryObject(t, s, 0)
			path := filepath.Join(s.cfg.Root, "objects", string(key))
			outside := t.TempDir()
			marker := filepath.Join(outside, "keep")
			if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			var err error
			switch tc {
			case "unknown":
				err = os.Mkdir(filepath.Join(s.cfg.Root, "objects", "unknown"), 0700)
			case "symlink":
				err = os.RemoveAll(path)
				if err == nil {
					err = os.Symlink(outside, path)
				}
			case "empty":
				err = os.Remove(filepath.Join(path, "meta.json"))
			case "bad metadata":
				err = os.WriteFile(filepath.Join(path, "meta.json"), []byte(`{"schema_version":2}`), 0600)
			case "unknown file":
				err = os.WriteFile(filepath.Join(path, "keep"), []byte("unknown"), 0600)
			case "linked data":
				err = os.Remove(filepath.Join(path, "data"))
				if err == nil {
					err = os.Symlink(marker, filepath.Join(path, "data"))
				}
			case "objects symlink":
				err = os.Rename(filepath.Join(s.cfg.Root, "objects"), filepath.Join(s.cfg.Root, "saved"))
				if err == nil {
					err = os.Symlink(outside, filepath.Join(s.cfg.Root, "objects"))
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if page, err := s.List(t.Context(), "", 2); err == nil || len(page.Objects) != 0 {
				t.Fatalf("damaged layout returned success or data: %+v %v", page, err)
			}
			if b, err := os.ReadFile(marker); err != nil || string(b) != "keep" {
				t.Fatalf("external file changed: %q %v", b, err)
			}
		})
	}
	s := newStorageTestStore(t, t.TempDir())
	for _, token := range []string{"garbage", "../objects", strings.Repeat("x", 4096)} {
		if _, err := s.List(t.Context(), token, 1); err == nil {
			t.Fatal("invalid cursor accepted")
		}
	}
	for _, limit := range []int{-1, 0, 1001, int(^uint(0) >> 1)} {
		if _, err := s.List(t.Context(), "", limit); err == nil {
			t.Fatal("invalid limit accepted")
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.List(ctx, "", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
	if page, err := s.List(t.Context(), "", 1); err != nil || len(page.Objects) != 0 || page.Next != "" {
		t.Fatalf("empty root = %+v %v", page, err)
	}
}
