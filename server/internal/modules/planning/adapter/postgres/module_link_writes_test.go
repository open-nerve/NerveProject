package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// linksWorld is a world with two modules of Web, Auth and Ops', and the store.
type linksWorld struct {
	world
	s           *postgresadapter.Store
	pool        *pgxpool.Pool
	auth, other uuid.UUID
}

func newLinksWorld(t *testing.T) linksWorld {
	t.Helper()
	s, pool := newStore(t)
	w := linksWorld{world: newWorld(t, pool), s: s, pool: pool, auth: uuid.NewV7(), other: uuid.NewV7()}
	exec(t, pool, `INSERT INTO modules (id, workspace_id, project_id, name) VALUES ($1, $3, $4, 'Auth'), ($2, $3, $4, 'Other')`, w.auth, w.other,
		w.workspace, w.web)
	return w
}

// link is a new link of module with url, at at, by the world's account.
func (w linksWorld) link(module uuid.UUID, url string, at time.Time) domain.ModuleLink {
	return domain.ModuleLink{ID: uuid.NewV7(), WorkspaceID: w.workspace, ProjectID: w.web, ModuleID: module, URL: url, Metadata: shared.LinkMetadata{},
		CreatedByID: &w.by, CreatedAt: at, UpdatedAt: at}
}

// A link is written with its metadata empty and listed with its module's other undeleted links, the latest created
// first, then by id descending; not another module's, nor a deleted one. An address another undeleted link of the
// module has, as it is spelt, is domain.ErrModuleLinkExists; the same address in another case or in another module, and
// a deleted link's, are free.
func TestInsertAndListModuleLinks(t *testing.T) {
	w := newLinksWorld(t)
	ctx := context.Background()
	at := now.Truncate(time.Microsecond)
	title := ""
	first := w.link(w.auth, "https://nerve.dev/spec", at.Add(-time.Hour))
	first.Title = &title
	same := sortedIDs(2)
	second, third := w.link(w.auth, "https://nerve.dev/a", at), w.link(w.auth, "https://nerve.dev/b", at)
	second.ID, third.ID = same[0], same[1]
	for _, l := range []domain.ModuleLink{first, second, third, w.link(w.other, "https://nerve.dev/spec", at)} {
		if err := w.s.InsertModuleLink(ctx, l); err != nil {
			t.Fatal(err)
		}
	}
	got, err := w.s.ListModuleLinks(ctx, w.auth)
	if err != nil || jsonOf(t, got) != jsonOf(t, []domain.ModuleLink{third, second, first}) {
		t.Fatalf("ListModuleLinks = %s, %v\nwant %s", jsonOf(t, got), err, jsonOf(t, []domain.ModuleLink{third, second, first}))
	}
	var updatedBy *uuid.UUID
	if err := w.pool.QueryRow(ctx, "SELECT updated_by_id FROM module_links WHERE id = $1", first.ID).Scan(&updatedBy); err != nil || updatedBy == nil ||
		*updatedBy != w.by {
		t.Fatalf("the writer = %v, %v; want the creator", updatedBy, err)
	}
	if err := w.s.InsertModuleLink(ctx, w.link(w.auth, "https://nerve.dev/spec", at)); !errors.Is(err, domain.ErrModuleLinkExists) {
		t.Fatalf("an address the module has: %v; want planning.module_link_exists", err)
	}
	if err := w.s.InsertModuleLink(ctx, w.link(w.auth, "https://nerve.dev/SPEC", at)); err != nil {
		t.Fatalf("the address in another case: %v", err)
	}
	exec(t, w.pool, "UPDATE module_links SET deleted_at = now() WHERE id = $1", first.ID)
	if err := w.s.InsertModuleLink(ctx, w.link(w.auth, "https://nerve.dev/spec", at)); err != nil {
		t.Fatalf("a deleted link's address: %v", err)
	}
	if got, err := w.s.ListModuleLinks(ctx, w.auth); err != nil || len(got) != 4 || got[3].ID != second.ID {
		t.Fatalf("after the deletion: %s, %v; want the deleted one left out", jsonOf(t, got), err)
	}
	if got, err := w.s.ListModuleLinks(ctx, uuid.NewV7()); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("no module's links = %v, %v; want an empty list", got, err)
	}
}

// A change writes the address, the title and the last change, and answers the link as written: a title alone keeps
// the metadata the link has now; a new address empties it. Another link's address of the module is
// domain.ErrModuleLinkExists, the link's own is not.
func TestUpdateModuleLink(t *testing.T) {
	w := newLinksWorld(t)
	ctx := context.Background()
	at := now.Truncate(time.Microsecond)
	l, taken := w.link(w.auth, "https://nerve.dev/spec", at.Add(-time.Hour)), w.link(w.auth, "https://nerve.dev/taken", at.Add(-time.Hour))
	for _, link := range []domain.ModuleLink{l, taken} {
		if err := w.s.InsertModuleLink(ctx, link); err != nil {
			t.Fatal(err)
		}
	}
	// A crawl wrote the metadata after the change read the link.
	exec(t, w.pool, `UPDATE module_links SET metadata = '{"title":"Spec","error":null}' WHERE id = $1`, l.ID)
	editor := account(t, w.pool, "ed")
	title, crawled := "Edited", "Spec"
	l.Title, l.UpdatedAt = &title, at
	got, err := w.s.UpdateModuleLink(ctx, l, editor)
	want := l
	want.Metadata = shared.LinkMetadata{"title": &crawled, "error": nil}
	if err != nil || jsonOf(t, got) != jsonOf(t, want) {
		t.Fatalf("a title alone = %s, %v\nwant %s", jsonOf(t, got), err, jsonOf(t, want))
	}
	var by *uuid.UUID
	if err := w.pool.QueryRow(ctx, "SELECT updated_by_id FROM module_links WHERE id = $1 AND updated_at = $2", l.ID, at).Scan(&by); err != nil ||
		by == nil || *by != editor {
		t.Fatalf("the last change = %v, %v; want the editor's at the change's moment", by, err)
	}
	l.URL = "https://nerve.dev/other"
	if got, err := w.s.UpdateModuleLink(ctx, l, editor); err != nil || got.URL != l.URL || len(got.Metadata) != 0 || *got.Title != "Edited" {
		t.Fatalf("a new address = %s, %v; want its metadata emptied", jsonOf(t, got), err)
	}
	l.URL = taken.URL
	if _, err := w.s.UpdateModuleLink(ctx, l, editor); !errors.Is(err, domain.ErrModuleLinkExists) {
		t.Fatalf("another link's address: %v; want planning.module_link_exists", err)
	}
	if got, err := w.s.UpdateModuleLink(ctx, taken, editor); err != nil || got.URL != taken.URL {
		t.Fatalf("the link's own address: %s, %v", jsonOf(t, got), err)
	}
}

// A deletion and the link's last change are one moment, the deleter's; a link deleted already keeps its deletion.
func TestDeleteModuleLink(t *testing.T) {
	w := newLinksWorld(t)
	ctx := context.Background()
	at := now.Truncate(time.Microsecond)
	l := w.link(w.auth, "https://nerve.dev/spec", at.Add(-time.Hour))
	if err := w.s.InsertModuleLink(ctx, l); err != nil {
		t.Fatal(err)
	}
	deleter := account(t, w.pool, "del")
	if err := w.s.DeleteModuleLink(ctx, l.ID, at, deleter); err != nil {
		t.Fatal(err)
	}
	if err := w.s.DeleteModuleLink(ctx, l.ID, at.Add(time.Hour), w.by); err != nil {
		t.Fatal(err)
	}
	var stamped bool
	if err := w.pool.QueryRow(ctx, "SELECT deleted_at = $2 AND updated_at = $2 AND updated_by_id = $3 FROM module_links WHERE id = $1", l.ID, at,
		deleter).Scan(&stamped); err != nil || !stamped {
		t.Fatalf("the deletion = %t, %v; want the first deleter's moment", stamped, err)
	}
	if _, found, err := w.s.GetModuleLink(ctx, l.ID); err != nil || found {
		t.Fatalf("a deleted link read: %t, %v", found, err)
	}
}
