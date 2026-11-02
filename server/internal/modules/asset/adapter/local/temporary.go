package local

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"time"
	"uuid"
)

func temporaryName(name string) bool {
	id, err := uuid.Parse(name)
	return err == nil && id != uuid.Nil() && id.String() == name
}

func (s *Store) SweepTemporary(ctx context.Context, before time.Time, limit int, token string) (next string, removed int, result error) {
	after, err := parseInventoryToken(token, "tmp", limit)
	if err != nil {
		return token, 0, err
	}
	if before.IsZero() {
		return token, 0, errors.New("invalid temporary cleanup cutoff")
	}
	root, ctx, finish, err := s.operation(ctx)
	if err != nil {
		return token, 0, err
	}
	defer finish()
	dir, err := openInventoryDirectory(root, "tmp")
	if err != nil {
		return token, 0, err
	}
	defer func() { _ = dir.Close() }()
	names, err := directoryPage(ctx, dir, after, limit, temporaryName)
	if err != nil {
		return token, 0, err
	}
	next = token
	for _, name := range names[:min(limit, len(names))] {
		if err := ctx.Err(); err != nil {
			return next, removed, err
		}
		deleted, err := removeTemporary(ctx, root, "tmp/"+name, before)
		if err != nil {
			return next, removed, err
		}
		if deleted {
			removed++
		}
		next = inventoryToken("tmp", name)
	}
	if err := ctx.Err(); err != nil {
		return next, removed, err
	}
	if len(names) <= limit {
		next = ""
	}
	return next, removed, nil
}

// Interrupted writes can leave no files, data alone, or data plus meta.json.
// Validate this bounded layout before removing it; RemoveAll never follows links.
func removeTemporary(ctx context.Context, root *os.Root, path string, before time.Time) (bool, error) {
	dir, err := openInventoryDirectory(root, path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} // the writer already renamed or removed it
	if err != nil {
		return false, err
	}
	defer func() { _ = dir.Close() }()
	info, err := dir.Stat()
	if err != nil {
		return false, storageError("inspect temporary age", err)
	}
	entries, err := dir.ReadDir(3)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, storageError("inspect temporary layout", err)
	}
	if len(entries) > 2 {
		return false, errors.New("invalid local temporary layout")
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if entry.Name() != "data" && entry.Name() != "meta.json" {
			return false, errors.New("invalid local temporary file")
		}
		fi, err := entry.Info()
		if err != nil {
			return false, storageError("inspect temporary file", err)
		}
		if !fi.Mode().IsRegular() {
			return false, errors.New("invalid local temporary file")
		}
	}
	if !info.ModTime().Before(before) {
		return false, nil
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	// A publishing writer may have renamed the directory after the scan. Verify
	// identity and age again before removing the path, without following a link.
	current, err := root.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, storageError("recheck temporary directory", err)
	}
	if !current.IsDir() || !os.SameFile(info, current) {
		return false, errors.New("local temporary directory changed")
	}
	if !current.ModTime().Before(before) {
		return false, nil
	}
	if err := root.RemoveAll(path); err != nil {
		return false, storageError("remove expired temporary object", err)
	}
	return true, nil
}
