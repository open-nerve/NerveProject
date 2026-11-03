package app

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"uuid"
)

type attachmentReadRows struct {
	count   func(context.Context, []uuid.UUID) ([]AttachmentCount, error)
	catalog func(context.Context, []uuid.UUID, uuid.UUID, int) ([]AttachmentCount, error)
}

func (*attachmentReadRows) ListAttachments(context.Context, AttachmentRead) ([]Attachment, error) {
	panic("unexpected list")
}
func (f *attachmentReadRows) AttachmentCounts(ctx context.Context, ids []uuid.UUID) ([]AttachmentCount, error) {
	return f.count(ctx, ids)
}
func (f *attachmentReadRows) ProjectAttachmentCounts(ctx context.Context, ids []uuid.UUID, after uuid.UUID, limit int) ([]AttachmentCount, error) {
	return f.catalog(ctx, ids, after, limit)
}
func attachmentReadID(n int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("0199a2b4-0000-7000-8000-%012d", n))
}

func TestAttachmentCounts(t *testing.T) {
	ids := make([]uuid.UUID, 1001)
	for n := range ids {
		ids[n] = attachmentReadID(n + 1)
	}
	original := slices.Clone(ids)
	request := append(slices.Clone(ids), ids[0])
	slices.Reverse(request)
	calls := 0
	f := &attachmentReadRows{count: func(ctx context.Context, batch []uuid.UUID) ([]AttachmentCount, error) {
		if ctx != t.Context() {
			t.Fatal("lost caller snapshot context")
		}
		calls++
		if len(batch) == 0 || len(batch) > 1000 {
			t.Fatalf("unbounded/empty batch: %d", len(batch))
		}
		var rows []AttachmentCount
		for _, id := range batch {
			if id != ids[2] {
				rows = append(rows, AttachmentCount{IssueID: id, Count: 7})
			}
		}
		return rows, nil
	}}
	r := NewAttachmentReads(f)
	if rows, err := r.CountIssueAttachments(t.Context(), nil); err != nil || len(rows) != 0 || calls != 0 {
		t.Fatalf("empty query: %v %v %d", rows, err, calls)
	}
	got, err := r.CountIssueAttachments(t.Context(), request)
	if err != nil || len(got) != 1001 || calls != 2 {
		t.Fatalf("batch result: len=%d calls=%d err=%v", len(got), calls, err)
	}
	for n, row := range got {
		want := int64(7)
		if n == 2 {
			want = 0
		}
		if row.IssueID != ids[n] || row.Count != want {
			t.Fatalf("row[%d]=%+v", n, row)
		}
	}
	if !slices.Equal(original, ids) || request[1] != ids[1000] {
		t.Fatal("modified caller IDs")
	}
	sentinel := errors.New("count database failed")
	for _, name := range []string{"database", "foreign", "duplicate", "negative", "zero ID", "zero count", "unsorted"} {
		t.Run(name, func(t *testing.T) {
			f.count = func(context.Context, []uuid.UUID) ([]AttachmentCount, error) {
				switch name {
				case "database":
					return nil, sentinel
				case "foreign":
					return []AttachmentCount{{attachmentReadID(9999), 1}}, nil
				case "duplicate":
					return []AttachmentCount{{ids[0], 1}, {ids[0], 2}}, nil
				case "negative":
					return []AttachmentCount{{ids[0], -1}}, nil
				case "zero ID":
					return []AttachmentCount{{uuid.Nil(), 1}}, nil
				case "zero count":
					return []AttachmentCount{{ids[0], 0}}, nil
				default:
					return []AttachmentCount{{ids[1], 1}, {ids[0], 1}}, nil
				}
			}
			got, err := r.CountIssueAttachments(t.Context(), ids[:2])
			if err == nil || len(got) != 0 {
				t.Fatalf("bad port became successful result: %v %v", got, err)
			}
			if name == "database" && !errors.Is(err, sentinel) {
				t.Fatal("lost database cause", err)
			}
		})
	}
	f.count = func(context.Context, []uuid.UUID) ([]AttachmentCount, error) {
		t.Fatal("zero ID reached SQL")
		return nil, nil
	}
	if _, err := r.CountIssueAttachments(t.Context(), []uuid.UUID{uuid.Nil()}); err == nil {
		t.Fatal("zero ID accepted")
	}
}

func TestAttachmentCountsCatalog(t *testing.T) {
	projects := make([]uuid.UUID, 1001)
	for n := range projects {
		projects[n] = attachmentReadID(n + 1)
	}
	saved := slices.Clone(projects)
	// Project batches have interleaved issue IDs. Appending their pages would
	// skip the second batch's smaller issue IDs after the first global cursor.
	calls := 0
	f := &attachmentReadRows{catalog: func(ctx context.Context, pids []uuid.UUID, after uuid.UUID, limit int) ([]AttachmentCount, error) {
		if ctx != t.Context() || len(pids) == 0 || len(pids) > 1000 || limit != 1000 {
			t.Fatalf("catalog SQL bounds/context: %d %d", len(pids), limit)
		}
		calls++
		rows := []AttachmentCount{}
		offset := 1
		if pids[0] == projects[0] {
			offset = 2
		}
		for n := offset; n <= 2002; n += 2 {
			id := attachmentReadID(10000 + n)
			if id.Compare(after) > 0 {
				rows = append(rows, AttachmentCount{id, int64(n)})
			}
			if len(rows) == limit {
				break
			}
		}
		return rows, nil
	}}
	r := NewAttachmentReads(f)
	after := uuid.Nil()
	got := []AttachmentCount{}
	for page := 0; page < 4; page++ {
		rows, err := r.ProjectAttachmentCounts(t.Context(), append(slices.Clone(projects), projects[0]), after, 1000)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, rows...)
		if len(rows) < 1000 {
			break
		}
		after = rows[len(rows)-1].IssueID
	}
	if len(got) != 2002 || calls != 6 {
		t.Fatalf("lost global pages: %d rows/%d SQL", len(got), calls)
	}
	for n, row := range got {
		if row.IssueID != attachmentReadID(10001+n) || row.Count != int64(n+1) {
			t.Fatalf("merged[%d]=%+v", n, row)
		}
	}
	if !reflect.DeepEqual(projects, saved) {
		t.Fatal("mutated projects")
	}
	f.catalog = func(context.Context, []uuid.UUID, uuid.UUID, int) ([]AttachmentCount, error) {
		t.Fatal("empty/invalid scope reached SQL")
		return nil, nil
	}
	if got, err := r.ProjectAttachmentCounts(t.Context(), nil, uuid.Nil(), 1000); err != nil || len(got) != 0 {
		t.Fatal(got, err)
	}
	for _, limit := range []int{0, 1, 999, 1001} {
		if _, err := r.ProjectAttachmentCounts(t.Context(), projects, uuid.Nil(), limit); err == nil {
			t.Fatalf("invalid page size accepted: %d", limit)
		}
	}
}

func TestAttachmentCountsCatalogRejectsBrokenPages(t *testing.T) {
	ids := make([]uuid.UUID, 1001)
	for n := range ids {
		ids[n] = attachmentReadID(n + 1)
	}
	after := attachmentReadID(2000)
	a, b := attachmentReadID(2001), attachmentReadID(2002)
	fault := errors.New("second project batch failed")
	for _, mode := range []string{"database", "duplicate", "negative", "zero ID", "zero count", "unsorted", "before cursor", "too many", "duplicate across batches"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			f := &attachmentReadRows{catalog: func(context.Context, []uuid.UUID, uuid.UUID, int) ([]AttachmentCount, error) {
				calls++
				if calls == 1 {
					return []AttachmentCount{{a, 1}}, nil
				}
				switch mode {
				case "database":
					return nil, fault
				case "duplicate":
					return []AttachmentCount{{b, 1}, {b, 2}}, nil
				case "negative":
					return []AttachmentCount{{b, -1}}, nil
				case "zero ID":
					return []AttachmentCount{{uuid.Nil(), 1}}, nil
				case "zero count":
					return []AttachmentCount{{b, 0}}, nil
				case "unsorted":
					return []AttachmentCount{{b, 1}, {a, 1}}, nil
				case "before cursor":
					return []AttachmentCount{{after, 1}}, nil
				case "too many":
					rows := make([]AttachmentCount, 1001)
					for n := range rows {
						rows[n] = AttachmentCount{attachmentReadID(3000 + n), 1}
					}
					return rows, nil
				default:
					return []AttachmentCount{{a, 2}}, nil
				}
			}}
			got, err := NewAttachmentReads(f).ProjectAttachmentCounts(t.Context(), ids, after, 1000)
			if err == nil || len(got) != 0 || calls != 2 || (mode == "database" && !errors.Is(err, fault)) {
				t.Fatal("broken later batch became partial success", got, err, calls)
			}
		})
	}
}
