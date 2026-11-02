// Package local implements local file storage and its signed transfer grants.
package local

import (
	"errors"
	"reflect"
	"strings"
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
	cfg  Config
	deps Deps
}

func New(cfg Config, deps Deps) (*Store, error) {
	if strings.TrimSpace(cfg.Root) == "" || strings.ContainsRune(cfg.Root, 0) || cfg.SizeLimit < 1 || cfg.SizeLimit > domain.MaxObjectSize {
		return nil, errors.New("invalid local file configuration")
	}
	if nilMAC(deps.UploadMAC) || nilMAC(deps.ReadMAC) || deps.RuntimeNow == nil {
		return nil, errors.New("local file storage lacks signing dependencies")
	}
	return &Store{cfg: cfg, deps: deps}, nil
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
