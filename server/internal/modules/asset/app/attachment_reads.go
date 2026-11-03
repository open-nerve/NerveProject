package app

import (
	"context"
	"errors"
	"slices"
	"uuid"
)

type AttachmentReads struct{ rows AttachmentReadRows }

func NewAttachmentReads(rows AttachmentReadRows) *AttachmentReads {
	return &AttachmentReads{rows: rows}
}
func (r *AttachmentReads) ListAttachments(ctx context.Context, q AttachmentRead) ([]Attachment, error) {
	return r.rows.ListAttachments(ctx, q)
}
func attachmentIDs(ids []uuid.UUID) ([]uuid.UUID, error) {
	ids = slices.Clone(ids)
	slices.SortFunc(ids, uuid.UUID.Compare)
	ids = slices.Compact(ids)
	if len(ids) > 0 && ids[0] == uuid.Nil() {
		return nil, errors.New("attachment read received a zero ID")
	}
	return ids, nil
}
func validAttachmentCounts(rows []AttachmentCount, after uuid.UUID, limit int) error {
	if len(rows) > limit {
		return errors.New("attachment count query exceeded its bound")
	}
	for _, row := range rows {
		if row.IssueID.Compare(after) <= 0 || row.Count <= 0 {
			return errors.New("attachment count query returned an invalid directory")
		}
		after = row.IssueID
	}
	return nil
}
func (r *AttachmentReads) CountIssueAttachments(ctx context.Context, ids []uuid.UUID) ([]AttachmentCount, error) {
	ids, err := attachmentIDs(ids)
	if err != nil {
		return nil, err
	}
	out := make([]AttachmentCount, 0, len(ids))
	for batch := range slices.Chunk(ids, 1000) {
		rows, err := r.rows.AttachmentCounts(ctx, batch)
		if err != nil {
			return nil, err
		}
		if err := validAttachmentCounts(rows, uuid.Nil(), len(batch)); err != nil {
			return nil, err
		}
		counts := make(map[uuid.UUID]int64, len(rows))
		for _, row := range rows {
			if _, found := slices.BinarySearchFunc(batch, row.IssueID, uuid.UUID.Compare); !found {
				return nil, errors.New("attachment count query returned an unrequested issue")
			}
			counts[row.IssueID] = row.Count
		}
		for _, id := range batch {
			out = append(out, AttachmentCount{IssueID: id, Count: counts[id]})
		}
	}
	return out, nil
}

// The first N rows of each disjoint project batch contain the global first N.
// Merge these prefixes before advancing the global issue cursor. Project order
// must never decide that cursor: later project batches can contain smaller IDs.
func (r *AttachmentReads) ProjectAttachmentCounts(ctx context.Context, projectIDs []uuid.UUID, after uuid.UUID, size int) ([]AttachmentCount, error) {
	if size != 1000 {
		return nil, errors.New("attachment catalog page size must be 1000")
	}
	ids, err := attachmentIDs(projectIDs)
	if err != nil {
		return nil, err
	}
	out := []AttachmentCount{}
	for batch := range slices.Chunk(ids, 1000) {
		rows, err := r.rows.ProjectAttachmentCounts(ctx, batch, after, size)
		if err != nil {
			return nil, err
		}
		if err := validAttachmentCounts(rows, after, size); err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}
	slices.SortFunc(out, func(a, b AttachmentCount) int { return a.IssueID.Compare(b.IssueID) })
	// A row belongs to one project only; duplicates across disjoint batches
	// indicate a broken port, not an opportunity to choose one count silently.
	if err := validAttachmentCounts(out, after, len(out)); err != nil {
		return nil, err
	}
	if len(out) > size {
		out = out[:size]
	}
	return out, nil
}
