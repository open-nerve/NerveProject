package local

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

// Put consumes exactly the declared bytes plus an EOF. The source belongs to
// the caller, which must unblock its I/O on cancellation (HTTP uses socket deadlines).
func (s *Store) Put(ctx context.Context, g domain.UploadGrant, source io.Reader) (result error) {
	if err := domain.ValidateUpload(g, s.cfg.SizeLimit, s.deps.RuntimeNow()); err != nil {
		return err
	}
	if source == nil {
		return domain.ErrInvalidUpload
	}
	root, opCtx, finish, err := s.operation(ctx)
	if err != nil {
		return err
	}
	defer finish()
	destination := "objects/" + string(g.Key)
	if _, err := root.Lstat(destination); err == nil {
		return domain.ErrObjectExists
	} else if !errors.Is(err, fs.ErrNotExist) {
		return storageError("inspect upload destination", err)
	}
	temporary := "tmp/" + uuid.New().String()
	if err := root.Mkdir(temporary, 0700); err != nil {
		return storageError("create temporary object", err)
	}
	defer func() {
		result = errors.Join(result, storageError("remove temporary object", root.RemoveAll(temporary)))
	}()
	file, err := root.OpenFile(temporary+"/data", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return storageError("create upload data", err)
	}
	defer func() { _ = file.Close() }() // The successful write checks Close below.
	hash := sha256.New()
	n, err := io.CopyBuffer(io.MultiWriter(file, hash), io.LimitReader(contextReader{ctx: opCtx, reader: source}, g.Size+1), make([]byte, 64<<10))
	if err != nil {
		return storageError("write upload", err)
	}
	if n != g.Size {
		return domain.ErrInvalidUpload
	}
	if err := opCtx.Err(); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return storageError("sync upload", err)
	}
	if err := file.Close(); err != nil {
		return storageError("close upload", err)
	}
	meta := objectMetadata{Version: 1, Key: g.Key, Size: n, ContentType: g.ContentType, ETag: fmt.Sprintf(`"%x"`, hash.Sum(nil)), StoredAt: s.deps.RuntimeNow().UTC()}
	if err := writeMetadata(root, temporary+"/meta.json", meta); err != nil {
		return err
	}
	if err := opCtx.Err(); err != nil {
		return err
	}
	return publishObject(root, temporary, destination)
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if canceled := r.ctx.Err(); canceled != nil {
		return n, canceled
	}
	return n, err
}

func writeMetadata(root *os.Root, path string, meta objectMetadata) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return errors.New("invalid local object metadata")
	}
	file, err := root.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return storageError("create metadata", err)
	}
	_, writeErr := file.Write(data)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	return storageError("write metadata", errors.Join(writeErr, file.Close()))
}

func publishObject(root *os.Root, temporary, destination string) error {
	if _, err := root.Lstat(destination); err == nil {
		return domain.ErrObjectExists
	} else if !errors.Is(err, fs.ErrNotExist) {
		return storageError("inspect publish destination", err)
	}
	if err := root.Rename(temporary, destination); err != nil {
		// Every complete destination is nonempty; a directory rename cannot replace
		// the winner of another process. Never remove the winning destination.
		if _, existsErr := root.Lstat(destination); existsErr == nil {
			return domain.ErrObjectExists
		}
		return storageError("publish object", err)
	}
	return nil
}
