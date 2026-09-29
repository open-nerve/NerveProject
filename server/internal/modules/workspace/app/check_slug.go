package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// CheckSlug answers GET /api/v0/workspace-slugs/{slug}: whether a slug can
// name a new workspace, and why not (M3 design 3.10, 5.1). It is an
// account-level operation, asking no Authorizer (M3 design 6.4); it tells
// only whether the name can be used, as Plane's check does (M3 design 8.2).
type CheckSlug struct {
	slugs SlugChecker
}

// NewCheckSlug returns the use case.
func NewCheckSlug(slugs SlugChecker) *CheckSlug {
	return &CheckSlug{slugs: slugs}
}

// Execute returns why slug cannot be used, invalid, reserved or taken, or ""
// when it can. Only a slug that could be used is looked up.
func (u *CheckSlug) Execute(ctx context.Context, slug string) (domain.SlugReason, error) {
	if reason := domain.CheckSlug(slug); reason != "" {
		return reason, nil
	}
	taken, err := u.slugs.SlugTaken(ctx, slug)
	if err != nil {
		return "", err
	}
	if taken {
		return domain.SlugTaken, nil
	}
	return "", nil
}
