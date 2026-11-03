package postgresadapter_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestWorkspaceImageFields(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	actor := newAccount(t, pool, "workspace-image@example.com")
	w := newWorkspace(t, s, "Images", "images", actor)
	id := uuid.NewV7()
	exec(t, pool, `INSERT INTO file_assets(id,user_id,workspace_id,entity_type,scope_kind,size,attributes,storage_metadata)
 VALUES($1,$2,$3,'WORKSPACE_LOGO','workspace',1,'{"name":"logo.png","content_type":"image/png"}',
 '{"schema_version":1,"origin":"copy","staging":null,"final":null,"candidates":[]}')`, id, actor, w.ID)
	check := func(t *testing.T, want *uuid.UUID) {
		t.Helper()
		bySlug, err := s.WorkspaceBySlug(ctx, w.Slug)
		if err != nil {
			t.Fatal(err)
		}
		byID, err := s.WorkspaceByID(ctx, w.ID)
		if err != nil {
			t.Fatal(err)
		}
		list, err := s.ListWorkspaces(ctx, actor)
		if err != nil || len(list) != 1 {
			t.Fatalf("list=%v %v", list, err)
		}
		for _, got := range []domain.Workspace{bySlug, byID, list[0]} {
			if !reflect.DeepEqual(got.LogoAssetID, want) {
				t.Fatalf("logo=%v want=%v", got.LogoAssetID, want)
			}
		}
		var saved *uuid.UUID
		if err := pool.QueryRow(ctx, "SELECT logo_asset_id FROM workspaces WHERE id=$1", w.ID).Scan(&saved); err != nil || !reflect.DeepEqual(saved, want) {
			t.Fatalf("database logo=%v %v", saved, err)
		}
	}
	for _, tc := range []struct {
		name string
		p    domain.WorkspacePatch
		want *uuid.UUID
	}{
		{"set", domain.WorkspacePatch{SetLogo: true, LogoAssetID: &id}, &id},
		{"omit", domain.WorkspacePatch{}, &id},
		{"clear", domain.WorkspacePatch{SetLogo: true}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.UpdateWorkspace(ctx, w.ID, tc.p, actor, now)
			if err != nil || !reflect.DeepEqual(got.LogoAssetID, tc.want) {
				t.Fatalf("update=%+v %v", got, err)
			}
			check(t, tc.want)
		})
	}
	boom := errors.New("rollback logo")
	err := postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
		if _, err := s.UpdateWorkspace(txctx, w.ID, domain.WorkspacePatch{SetLogo: true, LogoAssetID: &id}, actor, now); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatal(err)
	}
	check(t, nil)
}
