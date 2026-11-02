package local

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type objectMetadata struct {
	Version     int              `json:"schema_version"`
	Key         domain.ObjectKey `json:"key"`
	Size        int64            `json:"size"`
	ContentType string           `json:"content_type"`
	ETag        string           `json:"etag"`
	StoredAt    time.Time        `json:"stored_at"`
}

func (m objectMetadata) info() domain.ObjectInfo {
	return domain.ObjectInfo{Key: m.Key, Size: m.Size, ContentType: m.ContentType, ETag: m.ETag, ModifiedAt: m.StoredAt}
}

func (s *Store) Inspect(ctx context.Context, key domain.ObjectKey) (domain.ObjectInfo, error) {
	info, file, err := s.openObject(ctx, key)
	if err != nil {
		return domain.ObjectInfo{}, err
	}
	if err := file.Close(); err != nil {
		return domain.ObjectInfo{}, storageError("close inspected object", err)
	}
	return info, nil
}

func (s *Store) Open(ctx context.Context, key domain.ObjectKey, expectedETag string) (io.ReadCloser, error) {
	info, file, err := s.openObject(ctx, key)
	if err != nil {
		return nil, err
	}
	if info.ETag != expectedETag {
		_ = file.Close()
		return nil, domain.ErrObjectChanged
	}
	if err := ctx.Err(); err != nil {
		_ = file.Close()
		return nil, err
	}
	return s.ownReader(ctx, file), nil
}

func (s *Store) openObject(ctx context.Context, key domain.ObjectKey) (domain.ObjectInfo, *os.File, error) {
	if _, err := domain.ParseObjectKey(string(key)); err != nil {
		return domain.ObjectInfo{}, nil, err
	}
	root, opCtx, finish, err := s.operation(ctx)
	if err != nil {
		return domain.ObjectInfo{}, nil, err
	}
	defer finish()
	path := "objects/" + string(key)
	entry, err := root.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return domain.ObjectInfo{}, nil, domain.ErrObjectNotFound
	}
	if err != nil {
		return domain.ObjectInfo{}, nil, storageError("inspect object", err)
	}
	if !entry.IsDir() {
		return domain.ObjectInfo{}, nil, errors.New("invalid local object directory")
	}
	dir, err := root.OpenRoot(path)
	if err != nil {
		return domain.ObjectInfo{}, nil, storageError("open object directory", err)
	}
	defer func() { _ = dir.Close() }()
	metaFile, err := regularFile(dir, "meta.json")
	if err != nil {
		return domain.ObjectInfo{}, nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(metaFile, 16385))
	closeErr := metaFile.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return domain.ObjectInfo{}, nil, storageError("read metadata", err)
	}
	var meta objectMetadata
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if len(data) > 16384 || decoder.Decode(&meta) != nil {
		return domain.ObjectInfo{}, nil, errors.New("invalid local object metadata")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF || meta.Version != 1 || meta.Key != key || meta.Size < 1 || meta.Size > domain.MaxObjectSize || !domain.ValidMediaType(meta.ContentType) || meta.StoredAt.IsZero() || !validLocalETag(meta.ETag) {
		return domain.ObjectInfo{}, nil, errors.New("invalid local object metadata")
	}
	file, err := regularFile(dir, "data")
	if err != nil {
		return domain.ObjectInfo{}, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return domain.ObjectInfo{}, nil, storageError("inspect bytes", err)
	}
	if info.Size() != meta.Size {
		_ = file.Close()
		return domain.ObjectInfo{}, nil, domain.ErrObjectChanged
	}
	if err := opCtx.Err(); err != nil {
		_ = file.Close()
		return domain.ObjectInfo{}, nil, err
	}
	return meta.info(), file, nil
}

func regularFile(root *os.Root, path string) (*os.File, error) {
	before, err := root.Lstat(path)
	if err != nil {
		return nil, storageError("inspect file", err)
	}
	if !before.Mode().IsRegular() {
		return nil, errors.New("local object data must be a regular file")
	}
	file, err := root.Open(path)
	if err != nil {
		return nil, storageError("open file", err)
	}
	after, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, storageError("inspect opened file", err)
	}
	if !after.Mode().IsRegular() || !os.SameFile(before, after) {
		_ = file.Close()
		return nil, errors.New("local object file changed while opening")
	}
	return file, nil
}

func validLocalETag(tag string) bool {
	if len(tag) != 66 || tag[0] != '"' || tag[65] != '"' || strings.TrimLeft(tag[1:65], "0123456789abcdef") != "" {
		return false
	}
	_, err := hex.DecodeString(tag[1:65])
	return err == nil
}

func (s *Store) Delete(ctx context.Context, key domain.ObjectKey) error {
	if _, err := domain.ParseObjectKey(string(key)); err != nil {
		return err
	}
	root, opCtx, finish, err := s.operation(ctx)
	if err != nil {
		return err
	}
	defer finish()
	path := "objects/" + string(key)
	info, err := root.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return storageError("inspect deletion", err)
	}
	if !info.IsDir() {
		return errors.New("invalid local object directory")
	}
	if err := opCtx.Err(); err != nil {
		return err
	}
	return storageError("delete object", root.RemoveAll(path))
}
