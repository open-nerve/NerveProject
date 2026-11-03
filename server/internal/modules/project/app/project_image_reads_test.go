package app_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type imageReadRows struct {
	rows   []domain.Project
	fail   string
	shared []uuid.UUID
}

func (f *imageReadRows) read(ctx context.Context) ([]domain.Project, error) {
	stage := "pre-read"
	if ctx.Value(inTxKey{}) == true {
		stage = "reread"
	}
	if f.fail == stage {
		return nil, errDisk
	}
	return f.rows, nil
}
func (f *imageReadRows) GetProject(ctx context.Context, id, _ uuid.UUID) (domain.Project, bool, error) {
	rows, err := f.read(ctx)
	for _, row := range rows {
		if row.ID == id {
			return row, true, err
		}
	}
	return domain.Project{}, false, err
}
func (f *imageReadRows) ListProjects(ctx context.Context, _, _ uuid.UUID, _ domain.Visibility, _ bool) ([]domain.Project, error) {
	return f.read(ctx)
}
func (f *imageReadRows) ShareImageProjects(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	if ctx.Value(inTxKey{}) != true {
		return nil, errors.New("share outside transaction")
	}
	f.shared = slices.Clone(ids)
	if f.fail == "share" {
		return nil, errDisk
	}
	out := make(map[uuid.UUID]uuid.UUID)
	for _, p := range f.rows {
		if slices.Contains(ids, p.ID) {
			out[p.ID] = p.WorkspaceID
		}
	}
	return out, nil
}

type imageReadGate struct {
	before        func()
	err           error
	entered, left bool
	ids           []uuid.UUID
}

func (g *imageReadGate) Enter(ctx context.Context, ids ...uuid.UUID) (func(), error) {
	if ctx.Value(inTxKey{}) == true {
		return nil, errors.New("gate entered inside transaction")
	}
	g.entered, g.ids = true, slices.Clone(ids)
	if g.err != nil {
		return nil, g.err
	}
	if g.before != nil {
		g.before()
	}
	return func() { g.left = true }, nil
}

type imageReadSigner struct {
	noProjectImages
	err error
}

func (s imageReadSigner) ReadProjectImages(ctx context.Context, refs []app.ProjectImageReference) (map[uuid.UUID]app.ImageURL, error) {
	if _, err := s.noProjectImages.ReadProjectImages(ctx, refs); err != nil {
		return nil, err
	}
	return nil, s.err
}

type imageReadCaller struct {
	imageCaller
	err error
}

func (c imageReadCaller) LockCaller(ctx context.Context, a shared.Actor, at time.Time) error {
	if err := c.imageCaller.LockCaller(ctx, a, at); err != nil {
		return err
	}
	return c.err
}

// Dropping the candidate intersection returns an unheld new owner; returning the
// pre-read instead keeps a revoked one. Reordering output by lock ID breaks UI order.
func TestProjectImageReadCandidates(t *testing.T) {
	low := domain.Project{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), WorkspaceID: acme.ID, Name: "Z"}
	high := domain.Project{ID: uuid.MustParse("00000000-0000-4000-8000-000000000003"), WorkspaceID: acme.ID, Name: "B"}
	added := domain.Project{ID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), WorkspaceID: acme.ID, Name: "A"}
	for _, tc := range []struct {
		name       string
		next, want []domain.Project
	}{
		{"order", []domain.Project{high, low}, []domain.Project{high, low}},
		{"added and revoked", []domain.Project{added, low}, []domain.Project{low}},
		{"all revoked", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, ws, _, auth := newList()
			rows := &imageReadRows{rows: []domain.Project{high, low}}
			gate := &imageReadGate{before: func() { rows.rows = tc.next }}
			locks := app.NewImageReadLocks(imageCaller{ws.log}, ws, rows, clockAt{at: clockNow})
			uc := app.NewListProjects(app.ListProjectsDeps{Workspaces: ws, Projects: rows, Auth: auth, Locks: locks, Gate: gate, Images: noProjectImages{}, Tx: &fakeTx{}})
			got, err := uc.Execute(as(alice), "acme", false)
			if err != nil || !reflect.DeepEqual(got, tc.want) || !slices.Equal(rows.shared, []uuid.UUID{low.ID, high.ID}) || !slices.Equal(gate.ids, rows.shared) || !gate.left {
				t.Fatal("candidate/order boundary", got, rows.shared, gate, err)
			}
		})
	}
}

// A slug that now names another workspace cannot cause signing under the old lock.
func TestProjectImageReadSlugIdentity(t *testing.T) {
	_, ws, rows, auth := newList()
	gate := &imageReadGate{before: func() {
		reused := uuid.New()
		ws.workspaces["acme"] = app.Workspace{ID: reused}
		auth.grants[grantKey{alice, reused}] = shared.Grant{WorkspaceRole: shared.RoleAdmin}
	}}
	locks := app.NewImageReadLocks(imageCaller{ws.log}, &fakeWorkspaces{log: ws.log}, rows, clockAt{at: clockNow})
	uc := app.NewListProjects(app.ListProjectsDeps{Workspaces: ws, Projects: rows, Auth: auth, Locks: locks, Gate: gate, Images: noProjectImages{}, Tx: &fakeTx{}})
	got, err := uc.Execute(as(alice), "acme", false)
	if !errors.Is(err, domain.ErrWorkspaceNotFound) || got != nil || !gate.left {
		t.Fatal(got, err, gate)
	}
}

func TestProjectImageReadErrors(t *testing.T) {
	for _, list := range []bool{false, true} {
		for _, stage := range []string{"pre-read", "gate", "caller", "workspace", "share", "reread", "decision", "signer", "commit"} {
			t.Run(fmt.Sprintf("list %v %s", list, stage), func(t *testing.T) {
				_, ws, _, auth := newList()
				rows := &imageReadRows{rows: []domain.Project{{ID: webID, WorkspaceID: acme.ID}}}
				gate, tx := &imageReadGate{}, &fakeTx{}
				caller, signer := imageReadCaller{imageCaller: imageCaller{ws.log}}, imageReadSigner{}
				switch stage {
				case "pre-read", "share", "reread":
					rows.fail = stage
				case "gate":
					gate.err = errDisk
				case "caller":
					caller.err = errDisk
				case "workspace":
					gate.before = func() { ws.err = errDisk }
				case "decision":
					gate.before = func() { auth.errs = map[grantKey]error{{alice, acme.ID}: errDisk} }
				case "signer":
					signer.err = errDisk
				case "commit":
					tx.commitErr = errDisk
				}
				locks := app.NewImageReadLocks(caller, ws, rows, clockAt{at: clockNow})
				var err error
				if list {
					uc := app.NewListProjects(app.ListProjectsDeps{Workspaces: ws, Projects: rows, Auth: auth, Locks: locks, Gate: gate, Images: signer, Tx: tx})
					var got []domain.Project
					got, err = uc.Execute(as(alice), "acme", false)
					if got != nil {
						t.Fatal("partial list after failure", got)
					}
				} else {
					uc := app.NewGetProject(app.GetProjectDeps{Projects: rows, Auth: auth, Locks: locks, Gate: gate, Images: signer, Tx: tx})
					var got domain.Project
					got, err = uc.Execute(as(alice), webID)
					if got.ID != uuid.Nil() {
						t.Fatal("partial project after failure", got)
					}
				}
				if err != errDisk || (gate.entered && stage != "gate" && !gate.left) {
					t.Fatal("lost raw failure or gate", err, gate)
				}
			})
		}
	}
}
