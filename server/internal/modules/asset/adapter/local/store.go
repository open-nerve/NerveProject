// Package local implements local file storage and its signed transfer grants.
package local

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type Config struct {
	Root      string
	SizeLimit int64
}

type Deps struct {
	UploadMAC, ReadMAC app.MAC
	RuntimeNow         func() time.Time
}

type Store struct {
	cfg       Config
	deps      Deps
	root      *os.Root
	life      context.Context
	cancel    context.CancelFunc
	closed    atomic.Bool
	closeOnce sync.Once
	closeErr  error
}

func New(cfg Config, deps Deps) (*Store, error) {
	if strings.TrimSpace(cfg.Root) == "" || strings.ContainsRune(cfg.Root, 0) || cfg.SizeLimit < 1 || cfg.SizeLimit > domain.MaxObjectSize {
		return nil, errors.New("invalid local file configuration")
	}
	if nilMAC(deps.UploadMAC) || nilMAC(deps.ReadMAC) || deps.RuntimeNow == nil {
		return nil, errors.New("local file storage lacks signing dependencies")
	}
	if err := os.MkdirAll(cfg.Root, 0700); err != nil {
		return nil, storageError("create root", err)
	}
	info, err := os.Lstat(cfg.Root)
	if err != nil {
		return nil, storageError("inspect root", err)
	}
	if !info.IsDir() {
		return nil, errors.New("local storage root must be a directory")
	}
	root, err := os.OpenRoot(cfg.Root)
	if err != nil {
		return nil, storageError("open root", err)
	}
	for _, name := range []string{"objects", "tmp"} {
		if err := root.Mkdir(name, 0700); err != nil && !errors.Is(err, fs.ErrExist) {
			_ = root.Close()
			return nil, storageError("create storage directory", err)
		}
		info, err := root.Lstat(name)
		if err != nil || !info.IsDir() {
			_ = root.Close()
			return nil, errors.New("invalid local storage directory")
		}
	}
	life, cancel := context.WithCancel(context.Background())
	return &Store{cfg: cfg, deps: deps, root: root, life: life, cancel: cancel}, nil
}

func (s *Store) Close() error {
	s.closeOnce.Do(func() { s.closed.Store(true); s.cancel(); s.closeErr = storageError("close root", s.root.Close()) })
	return s.closeErr
}

// Each operation owns a root handle. Closing the store cancels operations but
// cannot invalidate the handle a failing upload still needs to remove its tmp.
func (s *Store) operation(ctx context.Context) (*os.Root, context.Context, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	if s.closed.Load() {
		return nil, nil, nil, fs.ErrClosed
	}
	root, err := s.root.OpenRoot(".")
	if err != nil {
		return nil, nil, nil, storageError("open operation root", err)
	}
	opCtx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(s.life, cancel)
	return root, opCtx, func() { stop(); cancel(); _ = root.Close() }, nil
}

func storageError(operation string, err error) error {
	if err == nil {
		return nil
	}
	var pathErr *fs.PathError
	var linkErr *os.LinkError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	} else if errors.As(err, &linkErr) {
		err = linkErr.Err
	}
	return fmt.Errorf("local storage %s: %w", operation, err)
}

func nilMAC(mac app.MAC) bool {
	if mac == nil {
		return true
	}
	switch value := reflect.ValueOf(mac); value.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.Interface:
		return value.IsNil()
	default:
		return false
	}
}
