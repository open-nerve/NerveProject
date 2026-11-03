package app

import (
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func assertImageField(t *testing.T, err error, field string) {
	t.Helper()
	var se *shared.Error
	if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || len(se.Fields) != 1 || se.Fields[0].Field != field || se.Fields[0].Code != shared.FieldNotAllowed {
		t.Fatalf("field refusal=%v want=%s", err, field)
	}
}
func TestImageBindingsErrorsAndRevalidation(t *testing.T) {
	t.Run("lock error", func(t *testing.T) {
		u, f, req, _ := imageFixture(t)
		boom := errors.New("database")
		f.lockErr = boom
		if _, err := u.LockImageBindings(t.Context(), req); !errors.Is(err, boom) {
			t.Fatal(err)
		}
	})
	t.Run("apply rechecks missing candidate", func(t *testing.T) {
		u, f, req, _ := imageFixture(t)
		at, err := u.LockImageBindings(t.Context(), req)
		if err != nil {
			t.Fatal(err)
		}
		delete(f.rows, *req.Changes[1].NextID)
		if err := u.ApplyImageBindings(t.Context(), req, at); !errors.Is(err, domain.ErrAssetNotFound) {
			t.Fatal(err)
		}
		if len(f.writes) != 0 {
			t.Fatal("invalid batch partially applied")
		}
	})
	t.Run("writer error", func(t *testing.T) {
		u, f, req, _ := imageFixture(t)
		at, err := u.LockImageBindings(t.Context(), req)
		if err != nil {
			t.Fatal(err)
		}
		boom := errors.New("write")
		f.writeErr = boom
		if err := u.ApplyImageBindings(t.Context(), req, at); !errors.Is(err, boom) {
			t.Fatal(err)
		}
	})
}
func TestImageBindingsNewProject(t *testing.T) {
	u, f, req, _ := imageFixture(t)
	req.Host.WorkspaceID, req.Host.ProjectID, req.Host.NewProject = accessID(30), accessID(40), true
	req.Changes = req.Changes[:1]
	req.Changes[0].CurrentID = nil
	req.Changes[0].Purpose = domain.ProjectCover
	req.Changes[0].Field = "cover_image_asset_id"
	a := f.rows[*req.Changes[0].NextID]
	a.Purpose = domain.ProjectCover
	a.Placement.Scope = domain.Scope{Kind: domain.ScopeNewProject, ID: req.Host.WorkspaceID}
	a.Placement.WorkspaceID = req.Host.WorkspaceID
	f.rows[a.ID] = a
	at, err := u.LockImageBindings(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if f.rows[a.ID].Placement.ProjectID != uuid.Nil() {
		t.Fatal("validation migrated before project insert")
	}
	if err := u.ApplyImageBindings(t.Context(), req, at); err != nil {
		t.Fatal(err)
	}
	got := f.rows[a.ID]
	if got.Placement.Scope != (domain.Scope{Kind: domain.ScopeProject, ID: req.Host.ProjectID}) || got.Placement.ProjectID != req.Host.ProjectID || got.Placement.WorkspaceID != req.Host.WorkspaceID || got.Placement.UserID != req.Host.ApplicantID || got.BoundAt == nil {
		t.Fatalf("wrong migration: %+v", got)
	}
}
