package app_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestAssetHostAccessMatrix(t *testing.T) {
	for _, newProject := range []bool{false, true} {
		for _, intent := range []app.AssetIntent{app.AssetRead, app.AssetPrepare, app.AssetRemove} {
			_, reader, auth, p := newGet()
			a := app.NewAssetAccess(reader, auth)
			id := p.ID
			if newProject {
				id = acme.ID
			}
			q := app.AssetRequest{ID: id, NewProject: newProject, Intent: intent}
			actor := shared.Actor{UserID: alice}
			h, err := a.Inspect(t.Context(), actor, q)
			if err != nil || h.WorkspaceID != acme.ID || (!newProject && h.ProjectID != p.ID) {
				t.Fatalf("host %+v %v", h, err)
			}
			if err := a.Check(t.Context(), actor, q, h); err != nil {
				t.Fatal(err)
			}
			action := "project.update"
			if intent == app.AssetRead {
				action = "project.read"
			}
			if newProject {
				action = "project.create"
			}
			last := reader.log.calls[len(reader.log.calls)-1]
			if !strings.Contains(last, " "+action+" on ") {
				t.Fatalf("wrong action %s", last)
			}
			auth.errs = map[grantKey]error{{alice, acme.ID}: shared.ErrNotVisible}
			want := domain.ErrNotFound
			if newProject {
				want = domain.ErrWorkspaceNotFound
			}
			if err := a.Check(t.Context(), actor, q, h); !errors.Is(err, want) {
				t.Fatalf("revocation=%v", err)
			}
		}
	}
	_, reader, auth, p := newGet()
	a := app.NewAssetAccess(reader, auth)
	actor := shared.Actor{UserID: alice}
	q := app.AssetRequest{ID: p.ID, Intent: app.AssetPrepare}
	h, err := a.Inspect(t.Context(), actor, q)
	if err != nil {
		t.Fatal(err)
	}
	p.ArchivedAt = &now
	reader.projects[p.ID] = p
	if _, err := a.Inspect(t.Context(), actor, q); err != nil {
		t.Fatalf("archive rejected in precheck: %v", err)
	}
	if err := a.Check(t.Context(), actor, q, h); !errors.Is(err, domain.ErrArchived) {
		t.Fatalf("archive check=%v", err)
	}
	auth.errs = map[grantKey]error{{alice, acme.ID}: shared.Forbidden()}
	if err := a.Check(t.Context(), actor, q, h); !errors.Is(err, shared.Forbidden()) {
		t.Fatalf("archive before permission: %v", err)
	}
}
