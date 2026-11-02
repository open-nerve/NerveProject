package app

import (
	"context"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type ObjectPage struct {
	Objects []domain.ObjectInfo
	Next    string
}
type ObjectLister interface {
	List(context.Context, string, int) (ObjectPage, error)
}
type TemporaryCleaner interface {
	SweepTemporary(context.Context, time.Time, int, string) (next string, removed int, err error)
}
