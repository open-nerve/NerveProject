package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DescriptionVersionPage is a page of saved descriptions, newest first.
type DescriptionVersionPage struct {
	Data       []domain.DescriptionVersion
	NextCursor string
}

// ListDescriptionVersionsDeps supplies reads and an independent snapshot.
type ListDescriptionVersionsDeps struct {
	Locks    Locks
	Versions DescriptionVersionLister
	ReadTx   shared.TxManager
}

// ListDescriptionVersions reads visibility and versions in one snapshot.
type ListDescriptionVersions struct{ d ListDescriptionVersionsDeps }

func NewListDescriptionVersions(d ListDescriptionVersionsDeps) *ListDescriptionVersions {
	return &ListDescriptionVersions{d: d}
}
func (u *ListDescriptionVersions) Execute(ctx context.Context, issueID uuid.UUID, limit *int, cursor *string) (DescriptionVersionPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return DescriptionVersionPage{}, err
	}
	var after *domain.DescriptionVersionCursor
	if cursor != nil {
		after = &domain.DescriptionVersionCursor{}
		if err := shared.DecodeCursor(*cursor, after); err != nil {
			return DescriptionVersionPage{}, err
		}
	}
	size, err := shared.PageSize(limit)
	if err != nil {
		return DescriptionVersionPage{}, err
	}
	page := DescriptionVersionPage{Data: []domain.DescriptionVersion{}}
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		if _, _, err := u.d.Locks.see(ctx, actor, issueID, domain.ActionReadDescriptionVersion); err != nil {
			return err
		}
		rows, err := u.d.Versions.DescriptionVersions(ctx, issueID, after, size+1)
		if err != nil {
			return err
		}
		if len(rows) > size {
			rows = rows[:size]
			last := rows[len(rows)-1]
			page.NextCursor, err = shared.EncodeCursor(domain.DescriptionVersionCursor{LastSavedAt: last.LastSavedAt, ID: last.ID})
			if err != nil {
				return err
			}
		}
		page.Data = append(page.Data, rows...)
		return nil
	})
	if err != nil {
		return DescriptionVersionPage{}, err
	}
	return page, nil
}

// GetDescriptionVersionDeps supplies only the reads needed by a detail.
type GetDescriptionVersionDeps struct {
	Locks    Locks
	Versions DescriptionVersionReader
}

// GetDescriptionVersion reads one live snapshot after its issue's visibility rule.
type GetDescriptionVersion struct{ d GetDescriptionVersionDeps }

func NewGetDescriptionVersion(d GetDescriptionVersionDeps) *GetDescriptionVersion {
	return &GetDescriptionVersion{d: d}
}
func (u *GetDescriptionVersion) Execute(ctx context.Context, versionID uuid.UUID) (domain.DescriptionVersion, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.DescriptionVersion{}, err
	}
	v, found, err := u.d.Versions.GetDescriptionVersion(ctx, versionID)
	if err != nil {
		return domain.DescriptionVersion{}, err
	}
	if !found {
		return domain.DescriptionVersion{}, domain.ErrDescriptionVersionNotFound
	}
	if _, _, err := u.d.Locks.seeAs(ctx, actor, v.IssueID, domain.ActionReadDescriptionVersion, domain.ErrDescriptionVersionNotFound); err != nil {
		return domain.DescriptionVersion{}, err
	}
	return v, nil
}
