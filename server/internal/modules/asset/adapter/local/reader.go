package local

import (
	"context"
	"io"
	"io/fs"
	"os"
	"sync"
)

// objectReader owns only its file; neither it nor a canceled request closes
// the store's root or another reader. Cancellation also unregisters lifecycle work.
type objectReader struct {
	file               *os.File
	ctx                context.Context
	store              *Store
	cancel             context.CancelFunc
	stopRead, stopLife func() bool
	once               sync.Once
	err                error
}

func (s *Store) ownReader(ctx context.Context, file *os.File) *objectReader {
	readCtx, cancel := context.WithCancel(ctx)
	stopLife := context.AfterFunc(s.life, cancel)
	r := &objectReader{file: file, ctx: readCtx, store: s, cancel: cancel, stopLife: stopLife}
	r.stopRead = context.AfterFunc(readCtx, func() { stopLife(); _ = r.closeFile() })
	return r
}

func (r *objectReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if r.store.closed.Load() {
		return 0, fs.ErrClosed
	}
	n, err := r.file.Read(p)
	if canceled := r.ctx.Err(); canceled != nil {
		return n, canceled
	}
	if err == io.EOF {
		return n, io.EOF
	}
	return n, storageError("read object", err)
}

func (r *objectReader) closeFile() error {
	r.once.Do(func() { r.err = storageError("close object", r.file.Close()) })
	return r.err
}
func (r *objectReader) Close() error { r.stopRead(); r.stopLife(); r.cancel(); return r.closeFile() }

func (r *objectReader) Seek(offset int64, whence int) (int64, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if r.store.closed.Load() {
		return 0, fs.ErrClosed
	}
	position, err := r.file.Seek(offset, whence)
	if canceled := r.ctx.Err(); canceled != nil {
		return position, canceled
	}
	return position, storageError("seek object", err)
}
