package postgresadapter_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// The insert must already carry the final cover: an extra UPDATE cannot
// substitute for the owner's atomic create contract.
func TestProjectImageFields(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	actor := newAccount(t, pool, "project-images@example.com")
	workspace := newWorkspace(t, pool, "images")
	asset := uuid.NewV7()
	exec(t, pool, `INSERT INTO file_assets(id,user_id,workspace_id,entity_type,scope_kind,size,attributes,storage_metadata)
 VALUES($1,$2,$3,'PROJECT_COVER','new_project',1,'{"name":"cover.png","content_type":"image/png"}',
 '{"schema_version":1,"origin":"copy","staging":null,"final":null,"candidates":[]}')`, asset, actor, workspace)
	builtin := "cover-29"
	id := uuid.NewV7()
	if err := s.CreateProject(ctx, app.ProjectRow{ID: id, WorkspaceID: workspace, Name: "Images", Identifier: "IMG", Network: domain.NetworkPublic,
		Timezone: "UTC", CreatedBy: actor, Now: now, CoverImageAssetID: &asset}); err != nil {
		t.Fatal(err)
	}
	check := func(t *testing.T, wantA *uuid.UUID, wantB *string) {
		t.Helper()
		got, found, err := s.GetProject(ctx, id, actor)
		if err != nil || !found {
			t.Fatalf("get: %t %v", found, err)
		}
		list, err := s.ListProjects(ctx, workspace, actor, domain.Visibility{All: true}, false)
		if err != nil || len(list) != 1 {
			t.Fatalf("list=%v %v", list, err)
		}
		for _, p := range []domain.Project{got, list[0]} {
			if !reflect.DeepEqual(p.CoverImageAssetID, wantA) || !reflect.DeepEqual(p.CoverBuiltinID, wantB) {
				t.Fatalf("cover=%v/%v want=%v/%v", p.CoverImageAssetID, p.CoverBuiltinID, wantA, wantB)
			}
		}
		var a *uuid.UUID
		var b *string
		if err := pool.QueryRow(ctx, "SELECT cover_image_asset_id,cover_builtin_id FROM projects WHERE id=$1", id).Scan(&a, &b); err != nil || !reflect.DeepEqual(a, wantA) || !reflect.DeepEqual(b, wantB) {
			t.Fatalf("database cover=%v/%v %v", a, b, err)
		}
	}
	check(t, &asset, nil)
	for _, tc := range []struct {
		name string
		p    domain.ProjectPatch
		a    *uuid.UUID
		b    *string
	}{
		{"omit", domain.ProjectPatch{}, &asset, nil},
		{"choose builtin", domain.ProjectPatch{SetCoverBuiltin: true, CoverBuiltinID: &builtin}, nil, &builtin},
		{"clear absent asset", domain.ProjectPatch{SetCoverImage: true}, nil, &builtin},
		{"choose asset", domain.ProjectPatch{SetCoverImage: true, CoverImageAssetID: &asset}, &asset, nil},
		{"clear", domain.ProjectPatch{SetCoverImage: true}, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			patch, err := domain.CheckProjectPatch(tc.p)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.UpdateProject(ctx, id, patch, actor, now); err != nil {
				t.Fatal(err)
			}
			check(t, tc.a, tc.b)
		})
	}
	boom := errors.New("rollback cover")
	err := postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
		if err := s.UpdateProject(txctx, id, domain.ProjectPatch{SetCoverBuiltin: true, CoverBuiltinID: &builtin}, actor, now); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatal(err)
	}
	check(t, nil, nil)
}
