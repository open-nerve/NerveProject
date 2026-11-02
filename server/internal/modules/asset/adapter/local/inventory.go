package local

import (
	"container/heap"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"io/fs"
	"os"
	"slices"
	"strings"
)

// The cursor is adapter-owned and versioned. It contains only a validated flat
// name, and can never change which directory the caller scans.
func inventoryToken(kind, after string) string {
	return base64.RawURLEncoding.EncodeToString([]byte("1:" + kind + ":" + after))
}

func parseInventoryToken(token, kind string, limit int) (string, error) {
	invalid := errors.New("invalid local inventory cursor or limit")
	if limit < 1 || limit > 1000 {
		return "", invalid
	}
	if token == "" {
		return "", nil
	}
	if len(token) > 128 {
		return "", invalid
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil {
		return "", invalid
	}
	after, ok := strings.CutPrefix(string(raw), "1:"+kind+":")
	if !ok || (kind == "objects" && !objectName(after)) || (kind == "tmp" && !temporaryName(after)) || inventoryToken(kind, after) != token {
		return "", invalid
	}
	return after, nil
}

func openInventoryDirectory(root *os.Root, path string) (*os.File, error) {
	before, err := root.Lstat(path)
	if err != nil {
		return nil, storageError("inspect inventory directory", err)
	}
	if !before.IsDir() {
		return nil, errors.New("invalid local inventory directory")
	}
	dir, err := root.Open(path)
	if err != nil {
		return nil, storageError("open inventory directory", err)
	}
	after, err := dir.Stat()
	if err != nil || !after.IsDir() || !os.SameFile(before, after) {
		_ = dir.Close()
		if err != nil {
			return nil, storageError("inspect opened inventory directory", err)
		}
		return nil, errors.New("local inventory directory changed while opening")
	}
	return dir, nil
}

type directoryReader interface {
	ReadDir(int) ([]fs.DirEntry, error)
}

// Scan flat names in fixed chunks, retaining only the smallest limit+1 names.
// Each page costs O(N) directory enumeration and O(limit) retained memory.
// Metadata is read only for the page being returned; this is not a snapshot.
func directoryPage(ctx context.Context, dir directoryReader, after string, limit int, validName func(string) bool) ([]string, error) {
	names := make(nameWindow, 0, limit+1)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		entries, err := dir.ReadDir(256)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, storageError("enumerate inventory", err)
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if !entry.IsDir() || !validName(entry.Name()) {
				return nil, errors.New("invalid local inventory entry")
			}
			name := entry.Name()
			if name <= after {
				continue
			}
			if len(names) < limit+1 {
				heap.Push(&names, name)
			} else if name < names[0] {
				names[0] = name
				heap.Fix(&names, 0)
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	slices.Sort(names)
	return names, nil
}

// A max-heap makes the largest retained name the first one replaced.
type nameWindow []string

func (h nameWindow) Len() int           { return len(h) }
func (h nameWindow) Less(i, j int) bool { return h[i] > h[j] }
func (h nameWindow) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *nameWindow) Push(v any)        { *h = append(*h, v.(string)) }
func (h *nameWindow) Pop() any          { old := *h; v := old[len(old)-1]; *h = old[:len(old)-1]; return v }
