package postgresadapter

import (
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestAssetConfiguredRetentionBoundary(t *testing.T) {
	for _, days := range []int{1, 90} {
		for _, delta := range []time.Duration{-time.Microsecond, 0, time.Microsecond} {
			t.Run(fmt.Sprintf("%dd/boundary%v", days, delta), func(t *testing.T) {
				w := newAssetWorld(t)
				objects := w.local(t)
				a := w.asset(domain.ScopeIssue)
				retention := time.Duration(days) * 24 * time.Hour
				at := w.now.Add(-retention).Add(delta)
				a.DeletedAt, a.Uploaded = &at, true
				record := w.object(t, objects, a, 'o')
				a.Metadata.Final = &record
				if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
					t.Fatal(err)
				}
				deps := w.cleanupDeps(objects, nil, w.now)
				deps.PurgeAfter = retention
				result, err := app.NewCleanup(deps).Execute(t.Context(), app.SweepCursor{})
				if err != nil || !result.Complete {
					t.Fatalf("cleanup=%+v error=%v", result, err)
				}
				_, err = objects.Inspect(t.Context(), record.Key)
				removed := delta < 0
				if (err != nil && !errors.Is(err, domain.ErrObjectNotFound)) || errors.Is(err, domain.ErrObjectNotFound) != removed {
					t.Fatalf("final removed=%t want=%t: %v", errors.Is(err, domain.ErrObjectNotFound), removed, err)
				}
				page, err := w.store.purgeUnreferencedTestAssets(t.Context(), w.now.Add(-retention), uuid.Nil(), 2)
				if err != nil || (page.Deleted == 1) != removed {
					t.Fatalf("row cutoff differs from object cutoff: %+v %v", page, err)
				}
			})
		}
	}
}
