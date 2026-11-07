package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The lock world's module link store: an update keeps the stored metadata unless the address changes, as the
// statement does.

func (w *lockWorld) InsertModuleLink(ctx context.Context, l domain.ModuleLink) error {
	w.log(ctx, "InsertModuleLink")
	if w.linkRefusal != nil {
		return w.linkRefusal
	}
	w.insertedLinks = append(w.insertedLinks, l)
	return nil
}

func (w *lockWorld) UpdateModuleLink(ctx context.Context, l domain.ModuleLink, by uuid.UUID) (domain.ModuleLink, error) {
	w.log(ctx, "UpdateModuleLink by "+by.String())
	if w.linkRefusal != nil {
		return domain.ModuleLink{}, w.linkRefusal
	}
	if stored := w.links[l.ID]; stored.URL == l.URL {
		l.Metadata = stored.Metadata
	} else {
		l.Metadata = shared.LinkMetadata{}
	}
	w.updatedLinks = append(w.updatedLinks, l)
	return l, nil
}

func (w *lockWorld) DeleteModuleLink(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error {
	w.log(ctx, "DeleteModuleLink")
	w.deletedLinks = append(w.deletedLinks, fmt.Sprint(id, " ", at.Format(time.RFC3339), " ", by))
	return nil
}

func (w *lockWorld) ListModuleLinks(ctx context.Context, module uuid.UUID) ([]domain.ModuleLink, error) {
	w.log(ctx, "ListModuleLinks")
	var out []domain.ModuleLink
	for _, l := range w.links {
		if l.ModuleID == module {
			out = append(out, l)
		}
	}
	return out, nil
}

func (w *lockWorld) moduleLinks() ModuleLinkDeps {
	return ModuleLinkDeps{Locks: w.locks(), Links: w, Clock: w}
}

func asCreator() context.Context {
	return shared.WithActor(context.Background(), shared.Actor{UserID: lockCreator})
}

// archiveModule archives the lock world's module.
func (w *lockWorld) archiveModule() {
	m, at := w.modules[lockModule], time.Now()
	m.ArchivedAt = &at
	w.modules[lockModule] = m
}

// A link is added to the module under the project's lock, after the decision: its address normalized, its title as
// given, its metadata empty, in the module's project and workspace, created and last changed by the caller at the
// moment read after the lock.
func TestCreateModuleLink(t *testing.T) {
	w := newModuleWorld()
	w.now = time.Date(2026, 11, 7, 1, 0, 0, 0, time.UTC)
	title := ""
	l, err := NewCreateModuleLink(w.moduleLinks()).Execute(asCreator(), lockModule, domain.NewModuleLink{URL: " nerve.dev/spec ", Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	if l.ID == (uuid.UUID{}) || l.ModuleID != lockModule || l.ProjectID != lockProject || l.WorkspaceID != lockWorkspace || l.URL != "http://nerve.dev/spec" ||
		l.Title == nil || *l.Title != "" || l.Metadata == nil || len(l.Metadata) != 0 || l.CreatedByID == nil || *l.CreatedByID != lockCreator ||
		!l.CreatedAt.Equal(w.now) || !l.UpdatedAt.Equal(w.now) || len(w.insertedLinks) != 1 || w.insertedLinks[0].ID != l.ID {
		t.Fatalf("created %+v, stored %+v", l, w.insertedLinks)
	}
	want := []string{"GetModule outside tx", "Facts outside tx", "Authorize module.write outside tx", "Enter 1 outside tx", "Begin", "Lock",
		"GetModule", "Authorize module.write", "Now", "InsertModuleLink", "End", "Leave"}
	if !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
}

// A link's values are checked before anything is read; an archived module, which the lock reads again, takes none
// (planning.archived), after the decision; an address the module has is the store's refusal (domain.ErrModuleLinkExists).
func TestCreateModuleLinkRefusals(t *testing.T) {
	w := newModuleWorld()
	_, err := NewCreateModuleLink(w.moduleLinks()).Execute(asCreator(), lockModule, domain.NewModuleLink{URL: "ftp://nerve.dev"})
	var e *shared.Error
	if !errors.As(err, &e) || e.Code != shared.CodeValidationFailed || len(w.calls) != 0 {
		t.Fatalf("a bad address = %v, calls %q; want validation_failed before anything", err, w.calls)
	}
	w = newModuleWorld()
	w.onLock = w.archiveModule
	if _, err := NewCreateModuleLink(w.moduleLinks()).Execute(asCreator(), lockModule, domain.NewModuleLink{URL: "nerve.dev"}); !errors.Is(err, domain.ErrArchived) ||
		len(w.insertedLinks) != 0 || !slices.Contains(w.calls, "Authorize module.write") {
		t.Fatalf("an archived module = %v, stored %+v, calls %q; want planning.archived under the lock", err, w.insertedLinks, w.calls)
	}
	w = newModuleWorld()
	w.linkRefusal = domain.ErrModuleLinkExists
	if _, err := NewCreateModuleLink(w.moduleLinks()).Execute(asCreator(), lockModule, domain.NewModuleLink{URL: "nerve.dev"}); !errors.Is(err, domain.ErrModuleLinkExists) {
		t.Fatalf("a taken address = %v; want planning.module_link_exists", err)
	}
}

// A change of a link under the project's lock writes what changes, at the moment read after the lock, by the caller: the
// title alone, the address kept and its metadata too; a new address, its metadata emptied by the store; the title
// cleared. A change of nothing, the patch empty or giving the link's values, writes nothing and answers the link as it
// is. An archived module refuses any, one of nothing too.
func TestUpdateModuleLink(t *testing.T) {
	title, edited, favicon := "Spec", "Edited", "https://nerve.dev/favicon.ico"
	url, other := "http://nerve.dev/spec", "https://nerve.dev/other"
	for _, tt := range []struct {
		name      string
		patch     domain.ModuleLinkPatch
		url       string
		title     *string
		metadata  bool
		writes    bool
		archived  bool
		refusedAs error
	}{
		{"the title alone", domain.ModuleLinkPatch{SetTitle: true, Title: &edited}, url, &edited, true, true, false, nil},
		{"the address", domain.ModuleLinkPatch{URL: &other}, other, &title, false, true, false, nil},
		{"the address normalized", domain.ModuleLinkPatch{URL: ptrTo(" nerve.dev/other ")}, "http://nerve.dev/other", &title, false, true, false, nil},
		{"the title cleared", domain.ModuleLinkPatch{SetTitle: true}, url, nil, true, true, false, nil},
		{"nothing", domain.ModuleLinkPatch{}, url, &title, true, false, false, nil},
		{"each as it is", domain.ModuleLinkPatch{URL: &url, SetTitle: true, Title: ptrTo("Spec")}, url, &title, true, false, false, nil},
		{"an archived module's", domain.ModuleLinkPatch{SetTitle: true, Title: &edited}, "", nil, false, false, true, domain.ErrArchived},
		{"an archived module's nothing", domain.ModuleLinkPatch{}, "", nil, false, false, true, domain.ErrArchived},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newModuleWorld()
			w.now = time.Date(2026, 11, 7, 1, 0, 0, 0, time.UTC)
			l := w.links[lockLink]
			l.URL, l.Title, l.Metadata = url, &title, shared.LinkMetadata{"favicon": &favicon}
			w.links[lockLink] = l
			if tt.archived {
				w.onLock = w.archiveModule
			}
			got, err := NewUpdateModuleLink(w.moduleLinks()).Execute(asCreator(), lockLink, tt.patch)
			if tt.refusedAs != nil {
				if !errors.Is(err, tt.refusedAs) || len(w.updatedLinks) != 0 {
					t.Fatalf("got %v, stored %+v; want %v", err, w.updatedLinks, tt.refusedAs)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.ID != lockLink || got.URL != tt.url || (got.Title == nil) != (tt.title == nil) || got.Title != nil && *got.Title != *tt.title ||
				(len(got.Metadata) == 1) != tt.metadata {
				t.Fatalf("answered %+v; want %s %v, metadata kept %t", got, tt.url, tt.title, tt.metadata)
			}
			if !tt.writes {
				if len(w.updatedLinks) != 0 || slices.Contains(w.calls, "Now") {
					t.Fatalf("stored %+v, calls %q; want no write", w.updatedLinks, w.calls)
				}
				return
			}
			want := []string{"GetModuleLink outside tx", "GetModule outside tx", "Facts outside tx", "Authorize module.write outside tx",
				"Enter 1 outside tx", "Begin", "Lock", "GetModuleLink", "GetModule", "Authorize module.write", "Now",
				"UpdateModuleLink by " + lockCreator.String(), "End", "Leave"}
			if !slices.Equal(w.calls, want) || len(w.updatedLinks) != 1 || !w.updatedLinks[0].UpdatedAt.Equal(w.now) {
				t.Fatalf("calls\n%q\nwant\n%q\nstored %+v", w.calls, want, w.updatedLinks)
			}
		})
	}
	// The values given are checked before anything is read; an address the module has is the store's refusal.
	w := newModuleWorld()
	if _, err := NewUpdateModuleLink(w.moduleLinks()).Execute(asCreator(), lockLink, domain.ModuleLinkPatch{URL: ptrTo("http://u:p@nerve.dev")}); err == nil ||
		len(w.calls) != 0 {
		t.Fatalf("a bad address = %v, calls %q; want validation_failed before anything", err, w.calls)
	}
	w = newModuleWorld()
	w.linkRefusal = domain.ErrModuleLinkExists
	if _, err := NewUpdateModuleLink(w.moduleLinks()).Execute(asCreator(), lockLink, domain.ModuleLinkPatch{URL: &other}); !errors.Is(err, domain.ErrModuleLinkExists) {
		t.Fatalf("a taken address = %v; want planning.module_link_exists", err)
	}
}

// A link is deleted under the project's lock at the moment read after it, by the caller; an archived module's is not.
func TestDeleteModuleLink(t *testing.T) {
	w := newModuleWorld()
	w.now = time.Date(2026, 11, 7, 1, 0, 0, 0, time.UTC)
	if err := NewDeleteModuleLink(w.moduleLinks()).Execute(asCreator(), lockLink); err != nil {
		t.Fatal(err)
	}
	if want := []string{fmt.Sprint(lockLink, " 2026-11-07T01:00:00Z ", lockCreator)}; !slices.Equal(w.deletedLinks, want) {
		t.Fatalf("deleted %q; want %q", w.deletedLinks, want)
	}
	if lock, deleted := slices.Index(w.calls, "Lock"), slices.Index(w.calls, "DeleteModuleLink"); lock < 0 || deleted < lock {
		t.Fatalf("calls %q", w.calls)
	}
	w = newModuleWorld()
	w.onLock = w.archiveModule
	if err := NewDeleteModuleLink(w.moduleLinks()).Execute(asCreator(), lockLink); !errors.Is(err, domain.ErrArchived) || len(w.deletedLinks) != 0 {
		t.Fatalf("an archived module's link = %v, deleted %q; want planning.archived", err, w.deletedLinks)
	}
}

// A module's links are read in its read's snapshot (module.read) once the decision is taken; an archived module's too.
func TestListModuleLinks(t *testing.T) {
	w := newModuleWorld()
	w.archiveModule()
	got, err := NewListModuleLinks(ListModuleLinksDeps{Reads: w.reads(), Links: w}).Execute(asCreator(), lockModule)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != lockLink {
		t.Fatalf("listed %+v", got)
	}
	if want := []string{"Begin", "GetModule", "Facts", "Authorize module.read", "ListModuleLinks", "End"}; !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	w = newModuleWorld()
	delete(w.modules, lockModule)
	if _, err := NewListModuleLinks(ListModuleLinksDeps{Reads: w.reads(), Links: w}).Execute(asCreator(), lockModule); !errors.Is(err, domain.ErrModuleNotFound) ||
		slices.Contains(w.calls, "ListModuleLinks") {
		t.Fatalf("no module = %v, calls %q", err, w.calls)
	}
}
