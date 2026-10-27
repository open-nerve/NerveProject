package app_test

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

type draftProjects struct {
	*fakeProject
	rows   map[uuid.UUID]uuid.UUID
	shared []uuid.UUID
	waited func()
}

func (p *draftProjects) Find(ctx context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	err := p.log.add(ctx, "Find", id)
	w, ok := p.rows[id]
	return w, ok, err
}
func (p *draftProjects) UndeletedIDs(ctx context.Context, w uuid.UUID) ([]uuid.UUID, error) {
	if err := p.log.add(ctx, "UndeletedIDs", w); err != nil {
		return nil, err
	}
	out := []uuid.UUID{}
	for id, wid := range p.rows {
		if wid == w {
			out = append(out, id)
		}
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return a.Compare(b) })
	return out, nil
}
func (p *draftProjects) ShareMany(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	p.shared = slices.Clone(ids)
	if p.waited != nil {
		p.waited()
	}
	if err := p.log.add(ctx, "ShareMany", ids); err != nil {
		return nil, err
	}
	out := map[uuid.UUID]uuid.UUID{}
	for _, id := range ids {
		if w, ok := p.rows[id]; ok {
			out[id] = w
		}
	}
	return out, nil
}

type draftWorkspace struct {
	log    *callLog
	found  bool
	waited func()
}

func (w draftWorkspace) WorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error) {
	return acme, w.found, w.log.add(ctx, "Workspace", slug)
}
func (w draftWorkspace) ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (bool, error) {
	if w.waited != nil {
		w.waited()
	}
	return w.found, w.log.add(ctx, "ShareWorkspace", id)
}

type draftStore struct {
	log       *callLog
	row       domain.Draft
	found     bool
	readCount int
	reading   func(int)
	waited    func()
	written   app.DraftWrite
	deleted   app.DraftDeletion
	rows      []domain.Draft
	total     int64
	list      app.DraftListRead
}

func (s *draftStore) GetDraft(ctx context.Context, id uuid.UUID) (domain.Draft, bool, error) {
	s.readCount++
	if s.reading != nil {
		s.reading(s.readCount)
	}
	err := s.log.add(ctx, "GetDraft", id)
	return s.row, s.found && id == s.row.ID, err
}
func (s *draftStore) LockDraft(ctx context.Context, id uuid.UUID) (domain.Draft, bool, error) {
	if s.waited != nil {
		s.waited()
	}
	err := s.log.add(ctx, "LockDraft", id)
	return s.row, s.found && id == s.row.ID, err
}
func (s *draftStore) InsertDraft(ctx context.Context, w app.DraftWrite) (domain.Draft, error) {
	s.written = w
	err := s.log.add(ctx, "InsertDraft")
	return domain.Draft{ID: w.ID, WorkspaceID: w.WorkspaceID, ProjectID: w.ProjectID, Payload: w.Payload, CreatedByID: w.By, UpdatedByID: &w.By, CreatedAt: w.At, UpdatedAt: w.At}, err
}
func (s *draftStore) UpdateDraft(ctx context.Context, w app.DraftWrite) (domain.Draft, error) {
	s.written = w
	err := s.log.add(ctx, "UpdateDraft")
	r := s.row
	r.ProjectID = w.ProjectID
	r.Payload = w.Payload
	r.UpdatedAt = w.At
	r.UpdatedByID = &w.By
	return r, err
}
func (s *draftStore) DeleteDraft(ctx context.Context, d app.DraftDeletion) error {
	s.deleted = d
	return s.log.add(ctx, "DeleteDraft")
}
func (s *draftStore) ListDrafts(ctx context.Context, r app.DraftListRead) ([]domain.Draft, int64, error) {
	s.list = r
	err := s.log.add(ctx, "ListDrafts")
	return s.rows, s.total, err
}

type draftWorld struct {
	log  *callLog
	auth *fakeAuthorizer
	p    *draftProjects
	w    draftWorkspace
	s    *draftStore
}

func newDraftWorld() *draftWorld {
	log := &callLog{}
	return &draftWorld{log: log, auth: &fakeAuthorizer{log: log}, p: &draftProjects{fakeProject: &fakeProject{log: log}, rows: map[uuid.UUID]uuid.UUID{web: acme}}, w: draftWorkspace{log: log, found: true}, s: &draftStore{log: log, found: true, row: domain.Draft{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: &web, CreatedByID: alice, Payload: []byte(`{"name":"unfinished"}`), CreatedAt: earlier, UpdatedAt: earlier}}}
}
func (w *draftWorld) locks() app.DraftLocks {
	return app.NewDraftLocks(app.NewLocks(w.p, nil, w.auth, fakeGate{log: w.log}, fakeTx{log: w.log}), w.w, w.p, w.s)
}
func (w *draftWorld) update() *app.UpdateDraft {
	return app.NewUpdateDraft(app.UpdateDraftDeps{Locks: w.locks(), Drafts: w.s, Clock: clockAt{at: clockNow, log: w.log}})
}
func (w *draftWorld) create() *app.CreateDraft {
	return app.NewCreateDraft(app.CreateDraftDeps{Locks: w.locks(), Workspaces: w.w, Drafts: w.s, Clock: clockAt{at: clockNow, log: w.log}})
}
func (w *draftWorld) delete() *app.DeleteDraft {
	return app.NewDeleteDraft(app.DeleteDraftDeps{Locks: w.locks(), Drafts: w.s, Clock: clockAt{at: clockNow, log: w.log}})
}
