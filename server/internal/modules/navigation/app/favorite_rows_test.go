package app_test

import (
	"math"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// alices are the stored favorites of the changes' and the deletions' tests, each named for the log by its field's
// name: alice's folders work ("Work") and home ("Home"); in work, her favorites cyc of the cycle sprint and mod of the
// module docs; at the top, proj of the project web, view of bob's view theirs and wsview of acme's own view board; her
// deleted folder gone ("Gone"), her folder betas of beta ("Library") and bob's folder bobs ("Office"). cyc's id is
// less than work's: a favorite made before the folder that holds it (re-91 I2).
type alices struct {
	work, home, gone, betas, bobs, cyc, mod, proj, view, wsview domain.Favorite
}

// stocked is newShelf with alices stored.
func stocked() (*shelf, alices) {
	s := newShelf()
	var a alices
	a.cyc = favored(uuid.NewV7(), acme, domain.FavoriteCycle, ptr(sprint), "", nil, 5)
	a.work = favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Work", nil, 100)
	a.cyc.ParentID = &a.work.ID
	a.mod = favored(uuid.NewV7(), acme, domain.FavoriteModule, ptr(docs), "", &a.work.ID, 4)
	a.home = favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Home", nil, 90)
	a.gone = favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Gone", nil, 80)
	a.betas = favored(uuid.NewV7(), beta, domain.FavoriteFolder, nil, "Library", nil, 70)
	a.bobs = favored(uuid.NewV7(), acme, domain.FavoriteFolder, nil, "Office", nil, 60)
	a.bobs.UserID = bob
	a.proj = favored(uuid.NewV7(), acme, domain.FavoriteProject, ptr(web), "", nil, 7)
	a.view = favored(uuid.NewV7(), acme, domain.FavoriteView, ptr(theirs), "", nil, 6)
	a.wsview = favored(uuid.NewV7(), acme, domain.FavoriteView, ptr(board), "", nil, 3)
	for name, f := range map[string]domain.Favorite{"work": a.work, "home": a.home, "gone": a.gone, "betas": a.betas, "bobs": a.bobs, "cyc": a.cyc,
		"mod": a.mod, "proj": a.proj, "view": a.view, "wsview": a.wsview} {
		s.names[f.ID] = name
		s.stored = append(s.stored, f)
	}
	s.deleted[a.gone.ID] = true
	s.names[nowhere] = "nowhere"
	return s, a
}

// idOf is the id of the favorite named name.
func (s *shelf) idOf(name string) uuid.UUID {
	for id, n := range s.names {
		if n == name {
			return id
		}
	}
	return uuid.UUID{}
}

// storedOf is the stored favorite id as it is now.
func (s *shelf) storedOf(id uuid.UUID) domain.Favorite {
	for _, f := range s.stored {
		if f.ID == id {
			return f
		}
	}
	return domain.Favorite{}
}

// targetReads are the reads of each of alices' targets, before the locks and again under them (Targets.sight): the
// target, its project, and the projects she reads; none of a folder's.
var targetReads = map[string][]string{
	"cyc":    {"cycles sprint", "project targets web", "readable alice acme"},
	"mod":    {"modules docs", "project targets web", "readable alice acme"},
	"proj":   {"project targets web", "readable alice acme"},
	"view":   {"view targets theirs", "project targets web", "readable alice acme"},
	"wsview": {"view targets board"},
}

// targetLocks are the locks each of alices' targets names (spec 2.8): its project's FOR SHARE, and a view's.
var targetLocks = map[string][]string{
	"cyc": {"share project web"}, "mod": {"share project web"}, "proj": {"share project web"},
	"view": {"share project web", "share view theirs"}, "wsview": {"share view board"},
}

// rowCalls are the calls of a change or a deletion of alice's favorite name (spec 2.7, 2.8): her favorite read without
// a lock, favorite.write's decision on its workspace and its target's reads; then, in the write's transaction, the
// workspace's FOR SHARE and the decision again, its target's locks, the locks before its own (before: a parent's),
// its own FOR UPDATE, those after it (after: a folder's favorites'), its target's reads again, and then.
func rowCalls(name string, before, after []string, then ...string) []string {
	calls := append([]string{"favorite " + name, "authorize alice favorite.write acme/-"}, targetReads[name]...)
	calls = append(calls, "begin write", "share workspace acme", "authorize alice favorite.write acme/-")
	calls = append(calls, targetLocks[name]...)
	calls = append(calls, before...)
	calls = append(calls, "lock favorite "+name)
	calls = append(calls, after...)
	calls = append(calls, targetReads[name]...)
	calls = append(calls, then...)
	return append(calls, "end write")
}

// checked are the calls of a change of alice's favorite name refused before its locks: its read, the decision and its
// target's reads.
func checked(name string) []string {
	return append([]string{"favorite " + name, "authorize alice favorite.write acme/-"}, targetReads[name]...)
}

// badItem is a change no favorite of a target may have: a sequence that is no finite number and a name.
var badItem = domain.FavoritePatch{Sequence: ptr(math.NaN()), Name: ptr("Mine")}

// answerOf is alice's favorite name, as f, answered with its target's name and icon now, a folder with its own.
func answerOf(name string, f domain.Favorite) app.NamedFavorite {
	switch name {
	case "proj":
		return app.NamedFavorite{Favorite: f, Name: "Web", LogoProps: icon}
	case "cyc":
		return app.NamedFavorite{Favorite: f, Name: "Sprint 1"}
	case "mod":
		return app.NamedFavorite{Favorite: f, Name: "Docs"}
	case "view":
		return app.NamedFavorite{Favorite: f, Name: "Theirs", LogoProps: icon}
	case "wsview":
		return app.NamedFavorite{Favorite: f, Name: "Board", LogoProps: icon}
	}
	return app.NamedFavorite{Favorite: f, Name: f.Name}
}

// underLock is a decision that allows her as a member before the locks and refuses with err under them.
func underLock(err error) func(shared.Action, shared.Target, bool) (shared.Grant, error) {
	return func(_ shared.Action, _ shared.Target, inTx bool) (shared.Grant, error) {
		if inTx {
			return shared.Grant{}, err
		}
		return member, nil
	}
}

// meanwhiles are what another writer, or her grants, changed while a change's or a deletion's check ran and its lock
// waited, each refusing it under its locks (spec 2.7): every 404 navigation.favorite_not_found (want), the one value,
// and a role refused 403 (errForbidden).
var meanwhiles = []struct {
	name   string
	row    string
	change func(s *shelf, a alices)
	want   error
}{
	{"her favorite deleted meanwhile", "proj", func(s *shelf, a alices) { s.deleted[a.proj.ID] = true }, domain.ErrFavoriteNotFound},
	{"her folder deleted meanwhile", "work", func(s *shelf, a alices) { s.deleted[a.work.ID] = true }, domain.ErrFavoriteNotFound},
	{"its workspace deleted meanwhile", "work", func(s *shelf, _ alices) { s.workspaceGone = map[uuid.UUID]bool{acme: true} }, domain.ErrFavoriteNotFound},
	{"she was removed from the workspace meanwhile", "work", func(s *shelf, _ alices) { s.decide = underLock(shared.ErrNotVisible) },
		domain.ErrFavoriteNotFound},
	{"she was made a guest meanwhile", "work", func(s *shelf, _ alices) { s.decide = underLock(shared.Forbidden()) }, errForbidden},
	{"its project archived meanwhile", "proj", func(s *shelf, _ alices) { archive(s.projectTargets, web) }, domain.ErrFavoriteNotFound},
	{"its cycle archived meanwhile", "cyc", func(s *shelf, _ alices) { archive(s.cycles, sprint) }, domain.ErrFavoriteNotFound},
	{"she left its project meanwhile", "mod", func(s *shelf, _ alices) { s.grants[acme] = s.grants[acme][1:] }, domain.ErrFavoriteNotFound},
	{"she was made a guest who sees her own meanwhile", "view", func(s *shelf, _ alices) { ownOnly(s) }, domain.ErrFavoriteNotFound},
	{"its workspace's view deleted meanwhile", "wsview", func(s *shelf, _ alices) { delete(s.viewTargets, board) }, domain.ErrFavoriteNotFound},
}

// refusals are the favorites alice may not change or delete, each refused before its locks (spec 2.7): every 404
// navigation.favorite_not_found, the one value, whatever she may not have (want), and a role refused 403
// (errForbidden); calls are its reads.
var refusals = []struct {
	name   string
	row    func(a alices) uuid.UUID
	change func(s *shelf)
	want   error
	calls  []string
}{
	{"an id of none", func(alices) uuid.UUID { return nowhere }, nil, domain.ErrFavoriteNotFound, []string{"favorite nowhere"}},
	{"her deleted folder", func(a alices) uuid.UUID { return a.gone.ID }, nil, domain.ErrFavoriteNotFound, []string{"favorite gone"}},
	{"another's folder", func(a alices) uuid.UUID { return a.bobs.ID }, nil, domain.ErrFavoriteNotFound, []string{"favorite bobs"}},
	{"a favorite of a workspace she no longer sees", func(a alices) uuid.UUID { return a.proj.ID },
		func(s *shelf) { s.decide = refuse(shared.ErrNotVisible) }, domain.ErrFavoriteNotFound, []string{"favorite proj", "authorize alice favorite.write acme/-"}},
	{"a guest's folder", func(a alices) uuid.UUID { return a.work.ID }, func(s *shelf) { s.decide = refuse(shared.Forbidden()) }, errForbidden,
		[]string{"favorite work", "authorize alice favorite.write acme/-"}},
	{"a favorite of an archived project", func(a alices) uuid.UUID { return a.proj.ID }, func(s *shelf) { archive(s.projectTargets, web) },
		domain.ErrFavoriteNotFound, checked("proj")},
	{"a favorite of a project she left", func(a alices) uuid.UUID { return a.proj.ID }, func(s *shelf) { s.grants[acme] = s.grants[acme][1:] },
		domain.ErrFavoriteNotFound, checked("proj")},
	{"a favorite of an archived cycle", func(a alices) uuid.UUID { return a.cyc.ID }, func(s *shelf) { archive(s.cycles, sprint) },
		domain.ErrFavoriteNotFound, checked("cyc")},
	{"a favorite of a deleted module", func(a alices) uuid.UUID { return a.mod.ID }, func(s *shelf) { delete(s.modules, docs) },
		domain.ErrFavoriteNotFound, []string{"favorite mod", "authorize alice favorite.write acme/-", "modules docs"}},
	{"a favorite of another's view while she sees her own alone", func(a alices) uuid.UUID { return a.view.ID }, ownOnly,
		domain.ErrFavoriteNotFound, checked("view")},
	{"a favorite of a deleted workspace's view", func(a alices) uuid.UUID { return a.wsview.ID }, func(s *shelf) { delete(s.viewTargets, board) },
		domain.ErrFavoriteNotFound, checked("wsview")},
}

// nowhere is an id of no favorite, named for the log.
var nowhere = uuid.NewV7()

// wantRefused fails t unless err is want: the one value of a 404, or the Authorizer's 403.
func wantRefused(t *testing.T, err, want error) {
	t.Helper()
	if want == errForbidden {
		wantForbidden(t, err)
		return
	}
	wantSame(t, err, want)
}
