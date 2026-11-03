package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type imageCandidateFunc func(context.Context, time.Time, uuid.UUID, int) ([]uuid.UUID, error)

func (f imageCandidateFunc) PurgeAssetCandidates(ctx context.Context, at time.Time, after uuid.UUID, n int) ([]uuid.UUID, error) {
	return f(ctx, at, after, n)
}

func TestImageReferencePurgeKeepsChangedReference(t *testing.T) {
	w := newImageReferenceWorld(t)
	rows := assetpg.New(w.pool)
	next := w.images[0]
	next.ID = uuid.NewV7()
	next.DeletedAt = nil
	next.DeletedByID = nil
	key, err := assetdomain.NewObjectKey('o', next.ID, uuid.NewV7())
	if err != nil {
		t.Fatal(err)
	}
	next.Metadata.Final = &assetdomain.ObjectRecord{Key: key, Phase: assetdomain.ObjectActive, SafeAfter: w.now, Size: 3, ContentType: "image/png", ETag: "tag", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	if _, err := rows.CreateAsset(t.Context(), next); err != nil {
		t.Fatal(err)
	}
	candidates := imageCandidateFunc(func(ctx context.Context, cutoff time.Time, after uuid.UUID, n int) ([]uuid.UUID, error) {
		ids, err := rows.PurgeAssetCandidates(ctx, cutoff, after, n)
		if err != nil {
			return nil, err
		}
		// Another transaction commits a replacement after this exact candidate set.
		_, err = w.pool.Exec(ctx, "UPDATE users SET avatar_asset_id=$1 WHERE id=$2", next.ID, w.user)
		return ids, err
	})
	p := assetapp.NewPurger(assetapp.PurgeDeps{Candidates: candidates, Deleter: rows, References: providedImageReferences(w.pool), Tx: postgres.NewTxManager(w.pool, time.Second)})
	page, err := p.Page(t.Context(), w.cutoff, 100, assetapp.PurgeCursor{})
	if err != nil || page.Deleted != 4 {
		t.Fatalf("purge=%+v %v", page, err)
	}
	refs := w.references(t)
	if refs[0] == nil || *refs[0] != next.ID || refs[1] != nil || refs[2] != nil || refs[3] != nil {
		t.Fatal("candidate snapshot cleared changed reference", refs)
	}
}

func TestImageReferencePurgeZeroDeleteAdvances(t *testing.T) {
	w := newImageReferenceWorld(t)
	rows := assetpg.New(w.pool)
	first := true
	candidates := imageCandidateFunc(func(ctx context.Context, cutoff time.Time, after uuid.UUID, n int) ([]uuid.UUID, error) {
		ids, err := rows.PurgeAssetCandidates(ctx, cutoff, after, n)
		if err != nil {
			return nil, err
		}
		if first {
			first = false
			page, err := imageReferencePurger(w.pool).Page(ctx, cutoff, n, assetapp.PurgeCursor{})
			if err != nil {
				return nil, err
			}
			if page.Deleted != 4 {
				return nil, fmt.Errorf("competing purge deleted %d", page.Deleted)
			}
		}
		return ids, nil
	})
	p := assetapp.NewPurger(assetapp.PurgeDeps{Candidates: candidates, Deleter: rows, References: providedImageReferences(w.pool), Tx: postgres.NewTxManager(w.pool, time.Second)})
	page, err := p.Page(t.Context(), w.cutoff, 100, assetapp.PurgeCursor{})
	if err != nil || page.Done || page.Scanned != 4 || page.Deleted != 0 || page.Cursor.After != w.images[3].ID {
		t.Fatalf("zero delete stalled=%+v %v", page, err)
	}
	done, err := p.Page(t.Context(), w.cutoff, 100, page.Cursor)
	if err != nil || !done.Done || done.Cursor != page.Cursor {
		t.Fatalf("resume=%+v %v", done, err)
	}
}

func TestImageReferencePurgeLockOrder(t *testing.T) {
	for _, target := range []string{"users", "workspaces", "projects", "file_assets"} {
		t.Run(target, func(t *testing.T) {
			w := newImageReferenceWorld(t)
			other := seedImageReferenceWorld(t, w.pool)
			ids := map[string][]uuid.UUID{"users": {w.user, other.user}, "workspaces": {w.tree.workspace, other.tree.workspace}, "projects": {w.tree.project, other.tree.project}}
			for _, a := range append(slices.Clone(w.images), other.images...) {
				ids["file_assets"] = append(ids["file_assets"], a.ID)
			}
			for _, v := range ids {
				slices.SortFunc(v, uuid.UUID.Compare)
			}
			last := ids[target][len(ids[target])-1]
			holder := holding(t, w.pool, "SELECT id FROM "+target+" WHERE id=$1 FOR UPDATE", last)
			probe := requestTx(w.pool)
			rows := assetpg.New(w.pool)
			p := assetapp.NewPurger(assetapp.PurgeDeps{Candidates: rows, Deleter: rows, References: providedImageReferences(w.pool), Tx: probe})
			collect := requestBackground(t, func() error {
				page, err := p.Page(t.Context(), w.cutoff, 100, assetapp.PurgeCursor{})
				if err == nil && (page.Scanned != 8 || page.Deleted != 8) {
					return fmt.Errorf("incomplete page %+v", page)
				}
				return err
			}, func() { requestRelease(t, holder) })
			requestBlocked(t, w.pool, probe, holder.Conn().PgConn().PID(), target)
			order := []string{"users", "workspaces", "projects", "file_assets"}
			boundary := slices.Index(order, target)
			for i, table := range order {
				if got := pgtest.RowWaits(t, w.pool, table, ids[table][0], "FOR UPDATE"); got != (i <= boundary) {
					t.Errorf("lock order %s while waiting for %s: locked=%t", table, target, got)
				}
			}
			if target != "file_assets" && pgtest.RowWaits(t, w.pool, target, ids[target][0], "FOR KEY SHARE") {
				t.Error("owner lock stronger than NO KEY UPDATE")
			}
			if target == "users" {
				// A normal account-first writer can take its own asset while
				// purge waits for that account. Asset-first purge creates a
				// real cycle; bound it with this fixture's local lock timeout.
				if _, err := holder.Exec(t.Context(), "SET LOCAL lock_timeout='100ms'"); err != nil {
					t.Fatal(err)
				}
				if _, err := holder.Exec(t.Context(), "SELECT id FROM file_assets WHERE id=$1 FOR UPDATE", other.images[0].ID); err != nil {
					t.Errorf("account-first writer blocked by asset-first purge: %v", err)
				}
			}
			requestRelease(t, holder)
			collect()
		})
	}
}
