package bootstrap

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestImageReferenceRetention(t *testing.T) {
	t.Run("four_ages_and_all_reverse_references", func(t *testing.T) {
		w := newImageRetentionWorld(t)
		blocked := w.group(t, "blockedimages", true)
		old := w.group(t, "oldimages", true)
		young := w.group(t, "youngimages", true)
		live := w.group(t, "liveimages", false)
		w.stop()
		w.age(t, blocked, 61, false)
		w.age(t, old, 61, true)
		w.age(t, young, 59, true)
		beforeLive, beforeYoung := imageGroupSnapshot(t, live), imageGroupSnapshot(t, young)
		beforeAudit := old.audit(t)
		fault := imageDeleteFault{assetStore: w.objects, keys: make(map[assetdomain.ObjectKey]bool)}
		for _, a := range blocked.images {
			fault.keys[a.Metadata.Final.Key] = true
		}
		if failures := cleanImageObjects(t, w, fault); failures != 4 {
			t.Fatal("expected four exact final-object failures", failures)
		}
		count, err := drainPurgeModule(forAssetPurge(imageReferencePurger(w.pool)), t.Context(), old.cutoff, 1)
		if err != nil || count != 4 {
			t.Fatal("eligible neighbour did not progress past file failures", count, err)
		}
		if !reflect.DeepEqual(old.references(t), make([]*uuid.UUID, 4)) || !slices.Equal(old.audit(t), beforeAudit) {
			t.Fatal("four references not detached atomically or owner audit changed")
		}
		assertImageRows(t, old, false)
		assertImageRows(t, blocked, true)
		for i, ref := range blocked.references(t) {
			if ref == nil || *ref != blocked.images[i].ID {
				t.Fatal("failed storage deletion lost a reverse reference", i)
			}
			a, err := assetpg.New(w.pool).FindAsset(t.Context(), blocked.images[i].ID)
			if err != nil || a.Metadata.Final == nil || a.Metadata.Final.Key != blocked.images[i].Metadata.Final.Key {
				t.Fatal("failed object lost its recovery row", a, err)
			}
			if _, err := w.objects.Inspect(t.Context(), a.Metadata.Final.Key); err != nil {
				t.Fatal("the deliberately unavailable object disappeared", err)
			}
		}
		if result := purgeImageTree(t, w); result.Blocked < 2 {
			t.Fatal("file recovery did not protect both parents", result)
		}
		assertImageParents(t, old, false)
		assertImageParents(t, blocked, true)
		if imageGroupSnapshot(t, live) != beforeLive || imageGroupSnapshot(t, young) != beforeYoung {
			t.Fatal("live or 59-day neighbour changed")
		}
		if failures := cleanImageObjects(t, w, w.objects); failures != 0 {
			t.Fatal("storage recovery still failed", failures)
		}
		purgeImageTree(t, w)
		assertImageRows(t, blocked, false)
		assertImageParents(t, blocked, false)
		var avatar, cover *uuid.UUID
		if err := w.pool.QueryRow(t.Context(), "SELECT avatar_asset_id,cover_image_asset_id FROM users WHERE id=$1", blocked.user).Scan(&avatar, &cover); err != nil || avatar != nil || cover != nil {
			t.Fatal("recovered user images still referenced", avatar, cover, err)
		}
		if imageGroupSnapshot(t, live) != beforeLive || imageGroupSnapshot(t, young) != beforeYoung {
			t.Fatal("recovery changed retained neighbours")
		}
		t.Log("16 API-bound images: 61d empty removed, 59d empty and live unchanged; four unavailable final keys protect references/parents and converge after recovery")
	})
	t.Run("old_parents_cannot_cascade_younger_images", func(t *testing.T) {
		w := newImageRetentionWorld(t)
		g := w.group(t, "youngparents", true)
		w.stop()
		w.age(t, g, 59, true)
		// The parent tombstone is older than the already-bound asset. Only the
		// physical purge runs: recovery's separate deletion-fact reconciliation
		// is not a license for a parent CASCADE to bypass the asset cutoff.
		for _, table := range []string{"workspaces", "projects"} {
			if _, err := w.pool.Exec(t.Context(), "UPDATE "+table+" SET deleted_at=$2 WHERE id=$1", g.tree.parent(table), w.now.Add(-61*24*time.Hour)); err != nil {
				t.Fatal(err)
			}
		}
		before := imageGroupSnapshot(t, g)
		if result := purgeImageTree(t, w); result.Blocked < 2 {
			t.Fatal("young asset failed to block old parent purge", result)
		}
		assertImageRows(t, g, true)
		assertImageParents(t, g, true)
		if imageGroupSnapshot(t, g) != before {
			t.Fatal("old parent purge changed young assets or references")
		}
	})
}

func imageGroupSnapshot(t *testing.T, g imageReferenceWorld) string {
	t.Helper()
	var ids []uuid.UUID
	for _, a := range g.images {
		ids = append(ids, a.ID)
	}
	var rows string
	if err := g.pool.QueryRow(t.Context(), "SELECT coalesce(jsonb_agg(to_jsonb(a) ORDER BY id),'[]'::jsonb)::text FROM file_assets a WHERE id=ANY($1::uuid[])", ids).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	owner, err := json.Marshal(struct {
		Audit []string
		Refs  []*uuid.UUID
	}{g.audit(t), g.references(t)})
	if err != nil {
		t.Fatal(err)
	}
	return string(owner) + rows
}

func assertImageRows(t *testing.T, g imageReferenceWorld, want bool) {
	t.Helper()
	for _, a := range g.images {
		var present bool
		if err := g.pool.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM file_assets WHERE id=$1)", a.ID).Scan(&present); err != nil || present != want {
			t.Fatal("image presence", a.Purpose, present, want, err)
		}
	}
}

func assertImageParents(t *testing.T, g imageReferenceWorld, want bool) {
	t.Helper()
	for _, table := range []string{"workspaces", "projects"} {
		var present bool
		if err := g.pool.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM "+table+" WHERE id=$1)", g.tree.parent(table)).Scan(&present); err != nil || present != want {
			t.Fatal("parent presence", table, present, want, err)
		}
	}
	if !want {
		for _, table := range []string{"issues", "states"} {
			var count int
			if err := g.pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE project_id=$1", g.tree.project).Scan(&count); err != nil || count != 0 {
				t.Fatal("child chain survived", table, count, err)
			}
		}
	}
}
