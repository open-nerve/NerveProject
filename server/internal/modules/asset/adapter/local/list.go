package local

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (s *Store) List(ctx context.Context, token string, limit int) (app.ObjectPage, error) {
	after, err := parseInventoryToken(token, "objects", limit)
	if err != nil {
		return app.ObjectPage{}, err
	}
	root, ctx, finish, err := s.operation(ctx)
	if err != nil {
		return app.ObjectPage{}, err
	}
	defer finish()
	dir, err := openInventoryDirectory(root, "objects")
	if err != nil {
		return app.ObjectPage{}, err
	}
	defer func() { _ = dir.Close() }()
	names, err := directoryPage(ctx, dir, after, limit, objectName)
	if err != nil {
		return app.ObjectPage{}, err
	}
	page := app.ObjectPage{Objects: make([]domain.ObjectInfo, 0, min(limit, len(names)))}
	for _, name := range names[:min(limit, len(names))] {
		if err := ctx.Err(); err != nil {
			return app.ObjectPage{}, err
		}
		if err := validObjectLayout(root, "objects/"+name); err != nil {
			return app.ObjectPage{}, err
		}
		info, err := s.Inspect(ctx, domain.ObjectKey(name))
		if err != nil {
			return app.ObjectPage{}, err
		}
		page.Objects = append(page.Objects, info)
	}
	if len(names) > limit {
		page.Next = inventoryToken("objects", names[limit-1])
	}
	if err := ctx.Err(); err != nil {
		return app.ObjectPage{}, err
	}
	return page, nil
}

func validObjectLayout(root *os.Root, path string) error {
	dir, err := openInventoryDirectory(root, path)
	if err != nil {
		return err
	}
	defer func() { _ = dir.Close() }()
	entries, err := dir.ReadDir(3)
	if err != nil && !errors.Is(err, io.EOF) {
		return storageError("inspect object layout", err)
	}
	if len(entries) != 2 {
		return errors.New("invalid local object layout")
	}
	for _, entry := range entries {
		if entry.Name() != "data" && entry.Name() != "meta.json" {
			return errors.New("invalid local object file")
		}
	}
	return nil
}

func objectName(name string) bool {
	_, err := domain.ParseObjectKey(name)
	return err == nil
}
