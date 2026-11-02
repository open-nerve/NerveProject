package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

func TestPurgeCursorRecordOutputFailureIsNotCompletion(t *testing.T) {
	lost := errors.New("database fault")
	for _, fail := range []bool{false, true} {
		calls := 0
		rows := purgeFunc(func(context.Context, time.Time, int, purgePageCursor) (purgePageProgress, error) {
			calls++
			r := purgePageProgress{Cursor: purgePageCursor{"issues", uuid.New()}, Scanned: 1, Deleted: 1, Done: true}
			if fail {
				return r, lost
			}
			return r, nil
		})
		uc := newPurgeDeleted([]purgeModule{{"issue", rows}}, clockAt(time.Now()), time.Now, 60*24*time.Hour, 1, time.Second)
		err := (&purgeWorker{uc: uc}).Work(t.Context(), &river.Job[purgeArgs]{JobRow: &rivertype.JobRow{Metadata: []byte(`{}`)}})
		if err == nil || !strings.Contains(err.Error(), "RecordOutput") || errors.Is(err, lost) != fail || calls != 1 {
			t.Fatalf("recording failure=%v calls=%d", err, calls)
		}
	}
}

func TestPurgeCursorBadOutputDoesNotCallOwner(t *testing.T) {
	uc := newPurgeDeleted([]purgeModule{{"issue", purgeFunc(func(context.Context, time.Time, int, purgePageCursor) (purgePageProgress, error) {
		t.Fatal("owner called for bad output")
		return purgePageProgress{}, nil
	})}}, clockAt(time.Now()), time.Now, 60*24*time.Hour, 1, time.Second)
	for _, raw := range []string{`{"output":null}`, `{"output":{"cursor":{"version":2}}}`, `{"output":{"cursor":{"version":1,"module":"gone","table":"issues","after":"00000000-0000-0000-0000-000000000000","cutoff":"2026-01-01T00:00:00Z"},"scanned":0,"deleted":0,"blocked":0,"complete":false}}`} {
		err := (&purgeWorker{uc: uc}).Work(t.Context(), &river.Job[purgeArgs]{JobRow: &rivertype.JobRow{Metadata: []byte(raw)}})
		if err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}
