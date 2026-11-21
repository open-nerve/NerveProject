package app_test

import (
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// errForbidden stands, in a table, for the Authorizer's 403, which is a new value each time.
var errForbidden = errors.New("forbidden")

// favor is a creation of a favorite of kind naming entity.
func favor(kind domain.FavoriteKind, entity uuid.UUID) domain.NewFavorite {
	return domain.NewFavorite{Kind: kind, EntityID: &entity}
}

// notAllowed is the problem of field, a target or a parent she may not have (spec 2.5, R7): the one value of each.
var (
	targetNotAllowed = shared.FieldError{Field: "entity_id", Code: shared.FieldNotAllowed, Message: "is no target you may favor in this workspace"}
	parentNotAllowed = shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed, Message: "is none of your folders in this workspace"}
)

// The pre-read, the locks and the reads under them of each kind, in order (spec 2.7, 2.8): the workspace's slug and
// favorite.write's decision on it without a lock; the target read without a lock, which names its project and its
// view; then, in the write's transaction, the workspace's FOR SHARE and the decision again, the target's project's
// FOR SHARE, a view's FOR SHARE, the parent's FOR SHARE, the target and its project read again under them with the
// projects she reads, and, after the last lock, the moment, the greatest sequence and the insertion.
var kindCalls = []struct {
	name             string
	n                domain.NewFavorite
	project          *uuid.UUID
	answer           string
	logo             shared.LogoProps
	check, locks, at []string
}{
	{"a project", favor(domain.FavoriteProject, web), ptr(web), "Web", icon,
		[]string{"project targets web"}, []string{"share project web"}, []string{"project targets web", "readable alice acme"}},
	{"a cycle", favor(domain.FavoriteCycle, sprint), ptr(web), "Sprint 1", shared.LogoProps{},
		[]string{"cycles sprint", "project targets web"}, []string{"share project web"}, []string{"cycles sprint", "project targets web", "readable alice acme"}},
	{"a module", favor(domain.FavoriteModule, docs), ptr(web), "Docs", shared.LogoProps{},
		[]string{"modules docs", "project targets web"}, []string{"share project web"}, []string{"modules docs", "project targets web", "readable alice acme"}},
	{"a project's view", favor(domain.FavoriteView, theirs), ptr(web), "Theirs", icon,
		[]string{"view targets theirs", "project targets web"}, []string{"share project web", "share view theirs"},
		[]string{"view targets theirs", "project targets web", "readable alice acme"}},
	{"a workspace's view", favor(domain.FavoriteView, board), nil, "Board", icon,
		[]string{"view targets board"}, []string{"share view board"}, []string{"view targets board"}},
	{"a folder", domain.NewFavorite{Kind: domain.FavoriteFolder, Name: ptr("  Pinned ")}, nil, "Pinned", shared.LogoProps{}, nil, nil, nil},
}

// callsOf are a creation's calls: kc's check, locks and reads, with parent's lock when it is not "".
func callsOf(kc []string, locks, at []string, kind domain.FavoriteKind, parent string) []string {
	calls := append([]string{"slug acme", "authorize alice favorite.write acme/-"}, kc...)
	calls = append(calls, "begin write", "share workspace acme", "authorize alice favorite.write acme/-")
	calls = append(calls, locks...)
	if parent != "" {
		calls = append(calls, "share folder "+parent)
	}
	calls = append(calls, at...)
	return append(calls, "now", "greatest alice acme", "insert "+string(kind)+" by alice", "end write")
}

// A favorite of each kind is created, 201 (spec 2.4, 2.5): her own, of the workspace, its project the target's (the
// project itself for a project's, none for a workspace's view's and a folder), its target's id, a folder's name
// without its surrounding white space, her first sequence 65535, made and last changed by her at the one moment read
// after the last lock, and answered with its target's name and icon. Each takes its locks in spec 2.8's order: the
// workspace, the target's project, a view, the parent, and reads its target again under them.
func TestCreateAFavoriteOfEachKind(t *testing.T) {
	for _, kc := range kindCalls {
		for _, inFolder := range []bool{false, true} {
			if inFolder && kc.n.Kind == domain.FavoriteFolder {
				continue
			}
			t.Run(kc.name+map[bool]string{true: " in a folder"}[inFolder], func(t *testing.T) {
				s := newShelf()
				n, parent := kc.n, ""
				if inFolder {
					folder := favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Work", nil, 1)
					s.stored, s.names[folder.ID] = append(s.stored, folder), "work"
					n.ParentID, parent = &folder.ID, "work"
				}
				got, created, err := s.createFavorite().Execute(as(t), "acme", n)
				if err != nil || !created {
					t.Fatalf("created %v, %v", created, err)
				}
				f := got.Favorite
				want := domain.Favorite{ID: f.ID, WorkspaceID: acme, ProjectID: kc.project, UserID: alice, Kind: n.Kind, EntityID: n.EntityID,
					ParentID: n.ParentID, Sequence: 65535, CreatedAt: moment, UpdatedAt: moment}
				if n.Kind == domain.FavoriteFolder {
					want.Name = "Pinned"
				}
				if inFolder {
					want.Sequence = 10001 // past the folder's 1
				}
				if f.ID == (uuid.UUID{}) || !reflect.DeepEqual(f, want) {
					t.Errorf("stored %+v\nwant %+v", f, want)
				}
				if got.Name != kc.answer || !reflect.DeepEqual(got.LogoProps, kc.logo) {
					t.Errorf("answered %q, %+v; want %q, %+v", got.Name, got.LogoProps, kc.answer, kc.logo)
				}
				if len(s.inserted) != 1 || s.inserted[0].favorite != f || s.inserted[0].by != alice {
					t.Errorf("inserted %+v; want the one answered, by alice", s.inserted)
				}
				wantCalls(t, s.world, callsOf(kc.check, kc.locks, kc.at, n.Kind, parent)...)
			})
		}
	}
}

// Her new favorite's sequence is one step past the greatest of her undeleted favorites in the workspace (spec 2.5):
// another's, another workspace's and a deleted one do not count.
func TestCreateAFavoriteAfterHerGreatest(t *testing.T) {
	s := newShelf()
	hers, gone := favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "A", nil, 70000.5), favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "B", nil, 9e9)
	elsewhere := favored(uuid.NewV7(), beta, domain.FavoriteFolder, nil, "C", nil, 9e9)
	bobs := favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "D", nil, 9e9)
	bobs.UserID = bob
	s.stored, s.deleted[gone.ID] = []domain.Favorite{hers, gone, elsewhere, bobs}, true
	got, _, err := s.createFavorite().Execute(as(t), "acme", favor(domain.FavoriteProject, web))
	if err != nil || got.Favorite.Sequence != 80000.5 {
		t.Fatalf("sequence %v, %v; want 80000.5", got.Favorite.Sequence, err)
	}
}

// What the body tells is refused first, before any read (spec 2.7): every problem of it in one 422, also on a slug of
// no workspace, a workspace she does not see, and as a guest. Then a slug of none, and a workspace she does not see,
// are workspace.not_found, the one value, before any lock; a guest is refused 403. Under the locks, a workspace deleted
// while the lock waited is workspace.not_found, she removed from it meanwhile the same, and she made a guest meanwhile
// 403; none writes or reads the clock.
func TestCreateAFavoriteRefusals(t *testing.T) {
	bad := domain.NewFavorite{Kind: domain.FavoriteFolder, EntityID: ptr(web), Name: ptr(" ")}
	underLock := func(err error) func(shared.Action, shared.Target, bool) (shared.Grant, error) {
		return func(_ shared.Action, _ shared.Target, inTx bool) (shared.Grant, error) {
			if inTx {
				return shared.Grant{}, err
			}
			return member, nil
		}
	}
	check, locked := []string{"slug acme", "authorize alice favorite.write acme/-", "project targets web"},
		[]string{"begin write", "share workspace acme", "authorize alice favorite.write acme/-", "end write"}
	for _, tt := range []struct {
		name   string
		slug   string
		n      domain.NewFavorite
		decide func(shared.Action, shared.Target, bool) (shared.Grant, error)
		gone   bool
		want   error
		calls  []string
	}{
		{"a bad body on a slug of none", "nowhere", bad, nil, false, nil, nil},
		{"a bad body on a workspace she does not see", "acme", bad, refuse(shared.ErrNotVisible), false, nil, nil},
		{"a bad body as a guest", "acme", bad, refuse(shared.Forbidden()), false, nil, nil},
		{"a slug of none", "nowhere", favor(domain.FavoriteProject, web), nil, false, domain.ErrWorkspaceNotFound, []string{"slug nowhere"}},
		{"a workspace she does not see", "acme", favor(domain.FavoriteProject, web), refuse(shared.ErrNotVisible), false, domain.ErrWorkspaceNotFound,
			[]string{"slug acme", "authorize alice favorite.write acme/-"}},
		{"a guest", "acme", favor(domain.FavoriteProject, web), refuse(shared.Forbidden()), false, errForbidden,
			[]string{"slug acme", "authorize alice favorite.write acme/-"}},
		{"a workspace deleted while the lock waited", "acme", favor(domain.FavoriteProject, web), nil, true, domain.ErrWorkspaceNotFound,
			append(append([]string{}, check...), "begin write", "share workspace acme", "end write")},
		{"removed from the workspace meanwhile", "acme", favor(domain.FavoriteProject, web), underLock(shared.ErrNotVisible), false, domain.ErrWorkspaceNotFound,
			append(append([]string{}, check...), locked...)},
		{"made a guest meanwhile", "acme", favor(domain.FavoriteProject, web), underLock(shared.Forbidden()), false, errForbidden,
			append(append([]string{}, check...), locked...)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := newShelf()
			if tt.decide != nil {
				s.decide = tt.decide
			}
			s.workspaceGone = map[uuid.UUID]bool{acme: tt.gone}
			_, _, err := s.createFavorite().Execute(as(t), tt.slug, tt.n)
			switch tt.want {
			case nil:
				fields := fieldsOf(t, err)
				if len(fields) != 2 || fields[0].Field != "entity_id" || fields[1].Field != "name" {
					t.Errorf("fields %+v; want entity_id and name", fields)
				}
			case errForbidden:
				wantForbidden(t, err)
			default:
				wantSame(t, err, tt.want)
			}
			wantCalls(t, s.world, tt.calls...)
			wroteNothing(t, s.world)
			if len(s.inserted) != 0 {
				t.Errorf("inserted %+v", s.inserted)
			}
		})
	}
}

// Under the locks, a target she may not favor is one 422, entity_id not_allowed, the one value whatever the reason
// (spec 2.5, R7): one she does not see (the table's rows), one that is not there, another workspace's, an archived
// one. A target she sees is created, whatever its project's toggles. Another workspace's target takes no lock of its
// project or its view.
func TestCreateAFavoriteOfATargetSheMayNot(t *testing.T) {
	rows := append([]struct {
		name   string
		change func(s *shelf)
		kind   domain.FavoriteKind
		target uuid.UUID
		listed bool
	}{
		{"a project that is not there", nil, domain.FavoriteProject, uuid.NewV7(), false},
		{"a cycle that is not there", nil, domain.FavoriteCycle, uuid.NewV7(), false},
		{"a view that is not there", nil, domain.FavoriteView, uuid.NewV7(), false},
		{"another workspace's project", func(s *shelf) { s.grants[acme] = append(s.grants[acme], app.ProjectGrant{ProjectID: site}) },
			domain.FavoriteProject, site, false},
		{"another workspace's cycle", func(s *shelf) {
			s.cycles[sprint] = app.PlanningTarget{WorkspaceID: beta, ProjectID: site, Name: "Elsewhere"}
			s.grants[acme] = append(s.grants[acme], app.ProjectGrant{ProjectID: site})
		}, domain.FavoriteCycle, sprint, false},
		{"another workspace's view", nil, domain.FavoriteView, elsewhere, false},
	}, visibleRows...)
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			s := newShelf()
			if row.change != nil {
				row.change(s)
			}
			_, created, err := s.createFavorite().Execute(as(t), "acme", favor(row.kind, row.target))
			if row.listed {
				if err != nil || !created {
					t.Fatalf("created %v, %v; want 201", created, err)
				}
				return
			}
			if fields := fieldsOf(t, err); !reflect.DeepEqual(fields, []shared.FieldError{targetNotAllowed}) {
				t.Errorf("fields %+v; want %+v", fields, targetNotAllowed)
			}
			if len(s.inserted) != 0 {
				t.Errorf("inserted %+v", s.inserted)
			}
			for _, call := range s.calls {
				if call == "now" || call == "share project site" || call == "share view elsewhere" {
					t.Errorf("called %q", call)
				}
			}
		})
	}
}

// The parent is one of her own undeleted folders in the workspace (spec 2.5), as the store's lock finds it (the fake
// mirrors its rule): another's folder, her favorite of a target, her deleted folder, her folder of another workspace
// and an id of none are each one 422, parent_id not_allowed; with a target she may not favor too, the one 422 carries
// both.
func TestCreateAFavoriteInAParentSheMayNot(t *testing.T) {
	bobs := favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Bob's", nil, 1)
	bobs.UserID = bob
	item, gone := favored(uuid.NewV7(), acme, domain.FavoriteProject, ptr(ops), "", nil, 2), favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Gone", nil, 3)
	betas := favored(uuid.NewV7(), beta, domain.FavoriteFolder, nil, "Beta's", nil, 4)
	for _, tt := range []struct {
		name   string
		parent uuid.UUID
		target uuid.UUID
		want   []shared.FieldError
	}{
		{"another's folder", bobs.ID, web, []shared.FieldError{parentNotAllowed}},
		{"her favorite of a target", item.ID, web, []shared.FieldError{parentNotAllowed}},
		{"her deleted folder", gone.ID, web, []shared.FieldError{parentNotAllowed}},
		{"her folder of another workspace", betas.ID, web, []shared.FieldError{parentNotAllowed}},
		{"an id of none", uuid.NewV7(), web, []shared.FieldError{parentNotAllowed}},
		{"a target she may not and another's folder", bobs.ID, site, []shared.FieldError{targetNotAllowed, parentNotAllowed}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := newShelf()
			s.stored, s.deleted[gone.ID] = []domain.Favorite{bobs, item, gone, betas}, true
			n := favor(domain.FavoriteProject, tt.target)
			n.ParentID = &tt.parent
			_, _, err := s.createFavorite().Execute(as(t), "acme", n)
			if fields := fieldsOf(t, err); !reflect.DeepEqual(fields, tt.want) {
				t.Errorf("fields %+v; want %+v", fields, tt.want)
			}
			if len(s.inserted) != 0 {
				t.Errorf("inserted %+v", s.inserted)
			}
		})
	}
}

// A favorite of a target she has an undeleted favorite of is answered as it is, 200, not created (spec 2.5): its id,
// its folder and its sequence kept though the body names none, and nothing stored. A target she may no longer favor is
// the 422 first.
func TestCreateAFavoriteSheHas(t *testing.T) {
	s := newShelf()
	folder := favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Work", nil, 1)
	has := favored(uuid.NewV7(), acme, domain.FavoriteProject, ptr(web), "", &folder.ID, 5)
	s.stored = []domain.Favorite{folder, has}
	got, created, err := s.createFavorite().Execute(as(t), "acme", favor(domain.FavoriteProject, web))
	if err != nil || created {
		t.Fatalf("created %v, %v; want the one she has", created, err)
	}
	if want := (app.NamedFavorite{Favorite: has, Name: "Web", LogoProps: icon}); !reflect.DeepEqual(got, want) {
		t.Errorf("answered %+v\nwant %+v", got, want)
	}
	if len(s.inserted) != 0 || len(s.stored) != 2 {
		t.Errorf("inserted %+v; want nothing", s.inserted)
	}
	archive(s.projectTargets, web)
	if _, _, err := s.createFavorite().Execute(as(t), "acme", favor(domain.FavoriteProject, web)); !reflect.DeepEqual(fieldsOf(t, err), []shared.FieldError{targetNotAllowed}) {
		t.Errorf("of an archived project she has: %v; want the 422", err)
	}
}

// A folder whose name, in any case, another of her undeleted folders in the workspace has is 409, from the store; one
// another has, or of another workspace, is hers to create.
func TestCreateAFolderWhoseNameIsTaken(t *testing.T) {
	s := newShelf()
	pinned, elsewhere := favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Pinned", nil, 1), favored(uuid.NewV7(), beta, domain.FavoriteFolder, nil, "Work", nil, 2)
	bobs := favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Work", nil, 3)
	bobs.UserID = bob
	s.stored = []domain.Favorite{pinned, elsewhere, bobs}
	_, _, err := s.createFavorite().Execute(as(t), "acme", domain.NewFavorite{Kind: domain.FavoriteFolder, Name: ptr(" PINNED ")})
	wantSame(t, err, domain.ErrFavoriteFolderNameTaken)
	if _, created, err := s.createFavorite().Execute(as(t), "acme", domain.NewFavorite{Kind: domain.FavoriteFolder, Name: ptr("work")}); err != nil || !created {
		t.Fatalf("work: created %v, %v", created, err)
	}
}

// What the decision under the locks reads is final (spec 2.7, 2.8): another writer's change committed while the
// check ran and the lock waited is what it decides on, each way.
func TestCreateAFavoriteDecidesUnderItsLocks(t *testing.T) {
	folder := favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Work", nil, 1)
	for _, tt := range []struct {
		name      string
		before    func(s *shelf)
		meanwhile func(s *shelf)
		n         domain.NewFavorite
		want      []shared.FieldError
	}{
		{"its project archived", nil, func(s *shelf) { archive(s.projectTargets, web) }, favor(domain.FavoriteProject, web), []shared.FieldError{targetNotAllowed}},
		{"its project unarchived", func(s *shelf) { archive(s.projectTargets, web) }, func(s *shelf) { s.projectTargets[web] = app.ProjectTarget{WorkspaceID: acme, Name: "Web"} },
			favor(domain.FavoriteProject, web), nil},
		{"the cycle archived", nil, func(s *shelf) { archive(s.cycles, sprint) }, favor(domain.FavoriteCycle, sprint), []shared.FieldError{targetNotAllowed}},
		{"the module deleted", nil, func(s *shelf) { delete(s.modules, docs) }, favor(domain.FavoriteModule, docs), []shared.FieldError{targetNotAllowed}},
		{"she left the project", nil, func(s *shelf) { s.grants[acme] = s.grants[acme][1:] }, favor(domain.FavoriteCycle, sprint), []shared.FieldError{targetNotAllowed}},
		{"she joined the project", func(s *shelf) { s.grants[acme] = s.grants[acme][1:] }, func(s *shelf) { s.grants[acme] = append(s.grants[acme], app.ProjectGrant{ProjectID: web}) },
			favor(domain.FavoriteCycle, sprint), nil},
		{"she was made a guest who sees her own", nil, ownOnly, favor(domain.FavoriteView, theirs), []shared.FieldError{targetNotAllowed}},
		{"the view deleted", nil, func(s *shelf) { delete(s.viewTargets, board) }, favor(domain.FavoriteView, board), []shared.FieldError{targetNotAllowed}},
		{"the folder deleted", nil, func(s *shelf) { s.deleted[folder.ID] = true }, domain.NewFavorite{Kind: domain.FavoriteProject, EntityID: ptr(web), ParentID: &folder.ID},
			[]shared.FieldError{parentNotAllowed}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := newShelf()
			s.stored = []domain.Favorite{folder}
			if tt.before != nil {
				tt.before(s)
			}
			s.meanwhile = tt.meanwhile
			_, created, err := s.createFavorite().Execute(as(t), "acme", tt.n)
			if tt.want == nil {
				if err != nil || !created {
					t.Fatalf("created %v, %v; want 201", created, err)
				}
				return
			}
			if fields := fieldsOf(t, err); !reflect.DeepEqual(fields, tt.want) {
				t.Errorf("fields %+v; want %+v", fields, tt.want)
			}
		})
	}
}
