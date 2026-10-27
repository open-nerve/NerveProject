package riveradapter

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"
)

func TestRowDeletionsEnqueue(t *testing.T) {
	ctx := t.Context()
	at := time.Date(2026, 10, 28, 1, 2, 3, 123456000, time.UTC)
	id, by := uuid.NewV7(), uuid.NewV7()
	failure := errors.New("insert failed")
	for _, scope := range []string{"project", "workspace_drafts"} {
		calls := 0
		q := NewRowDeletions(crawlInsertCall(func(got context.Context, args river.JobArgs, opts *river.InsertOpts) error {
			calls++
			if got != ctx || args.Kind() != "issue.delete_rows" {
				t.Fatal("context/kind")
			}
			raw, err := json.Marshal(args)
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"scope": scope, "id": id.String(), "at": at.Format(time.RFC3339Nano), "by": by.String()}
			if !reflect.DeepEqual(payload, want) {
				t.Fatalf("payload=%v", payload)
			}
			if opts == nil || !reflect.DeepEqual(*opts, river.InsertOpts{Queue: "default"}) {
				t.Fatalf("opts=%+v", opts)
			}
			return failure
		}))
		for range 2 {
			if err := q.Enqueue(ctx, scope, id, at, by); err != failure {
				t.Fatal(err)
			}
		}
		if calls != 2 {
			t.Fatal(calls)
		}
	}
}
