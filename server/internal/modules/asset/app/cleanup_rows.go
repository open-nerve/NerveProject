package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (u *Cleanup) cleanRow(ctx context.Context, snapshot domain.Asset, facts map[ParentKey]DeletionStamp, cutoff, until time.Time) (int, bool, error) {
	var keys []domain.ObjectKey
	err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		rows, err := u.d.Locks.LockAssets(ctx, []uuid.UUID{snapshot.ID})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		a := rows[0]
		if a.Placement != snapshot.Placement || a.StorageRevision != snapshot.StorageRevision {
			return nil
		}
		changed := applyDeletionFacts(&a, facts, cutoff)
		claimed, altered := claimObjects(&a, cutoff, u.d.RuntimeNow())
		keys = claimed
		changed = changed || altered
		if changed {
			_, err = u.d.Writer.SaveAsset(ctx, a)
		}
		return err
	})
	if err != nil {
		return 0, false, err
	}
	removed := 0
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return removed, false, err
		}
		remaining := time.Until(until)
		if remaining <= 0 {
			return removed, false, context.DeadlineExceeded
		}
		operation, cancel := context.WithTimeout(ctx, min(remaining, u.d.OperationTimeout))
		err = u.d.Objects.Delete(operation, key)
		cancel()
		// A failed object remains reclaiming. Advancing the row lets the next
		// attempt visit its neighbours; a fresh cycle retries this bounded key set.
		if err != nil {
			if ctx.Err() != nil {
				return removed, false, ctx.Err()
			}
			return removed, true, err
		}
		err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
			rows, err := u.d.Locks.LockAssets(ctx, []uuid.UUID{snapshot.ID})
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				return nil
			}
			a := rows[0]
			if !removeReclaimed(&a.Metadata, key) {
				return nil
			}
			_, err = u.d.Writer.SaveAsset(ctx, a)
			return err
		})
		if err != nil {
			return removed, false, err
		}
		removed++
	}
	return removed, true, nil
}

func applyDeletionFacts(a *domain.Asset, facts map[ParentKey]DeletionStamp, now time.Time) bool {
	var selected *DeletionStamp
	p := a.Placement
	for _, parent := range []ParentKey{{domain.ScopeWorkspace, p.WorkspaceID}, {domain.ScopeProject, p.ProjectID}, {domain.ScopeIssue, p.IssueID}, {domain.ScopeComment, p.CommentID}, {domain.ScopeDraft, p.DraftID}} {
		stamp, ok := facts[parent]
		if ok && (selected == nil || stamp.At.Before(selected.At)) {
			copy := stamp
			selected = &copy
		}
	}
	if selected != nil {
		return markDeleted(a, *selected)
	}
	if a.DeletedAt == nil && a.BoundAt == nil && a.CreatedAt.Before(now.Add(-7*24*time.Hour)) {
		return markDeleted(a, DeletionStamp{At: now})
	}
	return false
}
func markDeleted(a *domain.Asset, stamp DeletionStamp) bool {
	if a.DeletedAt != nil && !stamp.At.Before(*a.DeletedAt) {
		return false
	}
	a.DeletedAt = &stamp.At
	a.DeletedByID = stamp.By
	a.UpdatedAt = stamp.At
	a.UpdatedByID = stamp.By
	return true
}
func claimObjects(a *domain.Asset, businessNow, runtimeNow time.Time) ([]domain.ObjectKey, bool) {
	var keys []domain.ObjectKey
	changed := false
	claim := func(record *domain.ObjectRecord, allowed bool) {
		if record == nil || !allowed || !record.SafeAfter.Before(runtimeNow) {
			return
		}
		keys = append(keys, record.Key)
		if record.Phase != domain.ObjectReclaiming {
			record.Phase = domain.ObjectReclaiming
			changed = true
		}
	}
	claim(a.Metadata.Staging, a.DeletedAt != nil || a.Uploaded)
	claim(a.Metadata.Final, a.DeletedAt != nil && a.DeletedAt.Before(businessNow.Add(-60*24*time.Hour)))
	for i := range a.Metadata.Candidates {
		claim(&a.Metadata.Candidates[i].ObjectRecord, true)
	}
	return keys, changed
}
func removeReclaimed(m *domain.StorageMetadata, key domain.ObjectKey) bool {
	if m.Staging != nil && m.Staging.Key == key && m.Staging.Phase == domain.ObjectReclaiming {
		m.Staging = nil
		return true
	}
	if m.Final != nil && m.Final.Key == key && m.Final.Phase == domain.ObjectReclaiming {
		m.Final = nil
		return true
	}
	for i, c := range m.Candidates {
		if c.Key == key && c.Phase == domain.ObjectReclaiming {
			m.Candidates = append(m.Candidates[:i], m.Candidates[i+1:]...)
			return true
		}
	}
	return false
}
