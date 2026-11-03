package app

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type draftAssetRows struct {
	*contentRows
	pages   [][]uuid.UUID
	listErr error
}

func (f *draftAssetRows) DraftAssetIDs(_ context.Context, draft, after uuid.UUID, limit int) ([]uuid.UUID, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	var ids []uuid.UUID
	for id, a := range f.rows {
		if a.Placement.DraftID == draft && a.DeletedAt == nil && id.Compare(after) > 0 {
			ids = append(ids, id)
		}
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	if len(ids) > limit {
		ids = ids[:limit]
	}
	f.pages = append(f.pages, slices.Clone(ids))
	return ids, nil
}

func (f *draftAssetRows) MoveDraftAsset(_ context.Context, in DraftAssetWrite) (domain.Asset, error) {
	if f.writeErr != nil {
		return domain.Asset{}, f.writeErr
	}
	a := f.rows[in.AssetID]
	if a.StorageRevision != in.Revision {
		return domain.Asset{}, domain.ErrAssetScopeChanged
	}
	a.Placement.ProjectID = uuid.Nil()
	if in.ProjectID != nil {
		a.Placement.ProjectID = *in.ProjectID
	}
	if in.IssueID != uuid.Nil() {
		a.Placement.IssueID, a.Placement.DraftID = in.IssueID, uuid.Nil()
		a.Placement.Scope = domain.Scope{Kind: domain.ScopeIssue, ID: in.IssueID}
		if a.Purpose == domain.DraftIssueDescription {
			a.Purpose = domain.IssueDescription
		} else {
			a.Purpose = domain.IssueAttachment
		}
	}
	if in.Bind && a.BoundAt == nil {
		a.BoundAt = &in.At
	}
	a.UpdatedAt, a.UpdatedByID = in.At, &in.By
	a.StorageRevision++
	f.rows[a.ID] = a
	f.writes = append(f.writes, a.ID)
	return a, nil
}

func draftAssetFixture(t *testing.T) (*DraftAssetMover, *draftAssetRows, DraftAssetRequest, time.Time) {
	t.Helper()
	_, rows, content, at := contentFixture(t, domain.ScopeIssue)
	f := &draftAssetRows{contentRows: rows}
	q := DraftAssetRequest{ActorID: content.ActorID, WorkspaceID: content.Host.WorkspaceID, DraftID: accessID(40),
		OldProjectID: imgPtr(content.Host.ProjectID), NewProjectID: imgPtr(accessID(41)), ImageIDs: []uuid.UUID{accessID(8)}}
	for id, a := range f.rows {
		a.Placement.Scope = domain.Scope{Kind: domain.ScopeDraft, ID: q.DraftID}
		a.Placement.DraftID, a.Placement.IssueID = q.DraftID, uuid.Nil()
		if a.Purpose == domain.IssueDescription {
			a.Purpose = domain.DraftIssueDescription
		} else {
			a.Purpose = domain.DraftIssueAttachment
		}
		f.rows[id] = a
	}
	for _, id := range []uuid.UUID{accessID(9), accessID(10)} {
		a := f.rows[accessID(8)]
		a.ID = id
		key, err := domain.NewObjectKey('o', id, accessID(90))
		if err != nil {
			t.Fatal(err)
		}
		final := *a.Metadata.Final
		final.Key = key
		a.Metadata.Final = &final
		if id == accessID(9) {
			past := at.Add(-30 * 24 * time.Hour)
			a.BoundAt, a.CreatedAt = &past, past
		}
		f.rows[id] = a
	}
	u := NewDraftAssetTransfer(DraftAssetDeps{Locks: f, Reader: f, Writer: f, Clock: imageClockFunc(func() time.Time {
		f.clockLocks = append(f.clockLocks, len(f.locks))
		return at
	})})
	return u, f, q, at
}
