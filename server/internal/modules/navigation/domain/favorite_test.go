package domain_test

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The four kinds of a favorite of a target, and its folder: a project, a cycle, a module, a view of either kind
// (P3c spec 2.3, R4).
var targetKinds = []domain.FavoriteKind{domain.FavoriteProject, domain.FavoriteCycle, domain.FavoriteModule, domain.FavoriteView}

// A creation's fields by kind (spec 2.4, 2.5): a favorite of a target is its target's id, with or without a parent,
// and no name; a folder is its name, without its surrounding white space, and no target nor parent (one level, D42).
// Each comes back as the use case takes it.
func TestANewFavoritesFields(t *testing.T) {
	target, parent := uuid.NewV7(), uuid.NewV7()
	for _, kind := range targetKinds {
		for _, in := range []*uuid.UUID{nil, &parent} {
			n := domain.NewFavorite{Kind: kind, EntityID: &target, ParentID: in}
			if got, problems := domain.CheckNewFavorite(n); problems != nil || !reflect.DeepEqual(got, n) {
				t.Fatalf("%s in %v: %+v, %v; want it as given", kind, in, got, problems)
			}
		}
	}
	got, problems := domain.CheckNewFavorite(domain.NewFavorite{Kind: domain.FavoriteFolder, Name: ptr(" \tWork\n")})
	if want := (domain.NewFavorite{Kind: domain.FavoriteFolder, Name: ptr("Work")}); problems != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("folder: %+v, %v; want %+v", got, problems, want)
	}
}

// Every problem of a creation that needs no read, in the body's order (spec 2.4, 2.7): a kind none of the five is
// that alone, its fields' rules being a kind's; a favorite of a target lacks no entity_id and gives no name, valid or
// not; a folder gives no entity_id and no parent_id, and its name is required, 1 to 255 characters once trimmed,
// without NUL.
func TestANewFavoritesProblems(t *testing.T) {
	target, parent := uuid.NewV7(), uuid.NewV7()
	long := strings.Repeat("夹", 255)
	if got, problems := domain.CheckNewFavorite(domain.NewFavorite{Kind: domain.FavoriteFolder, Name: ptr(" " + long + "\n")}); problems != nil || *got.Name != long {
		t.Fatalf("255 characters: %q, %v", *got.Name, problems)
	}
	for _, tt := range []struct {
		name string
		n    domain.NewFavorite
		want []string
	}{
		{"an issue", domain.NewFavorite{Kind: "issue", EntityID: &target, Name: ptr(""), ParentID: &parent}, []string{"entity_type invalid_format"}},
		{"a page", domain.NewFavorite{Kind: "page", EntityID: &target}, []string{"entity_type invalid_format"}},
		{"an empty kind", domain.NewFavorite{Kind: ""}, []string{"entity_type invalid_format"}},
		{"a kind in another case", domain.NewFavorite{Kind: "Folder", Name: ptr("Work")}, []string{"entity_type invalid_format"}},
		{"a project without its target", domain.NewFavorite{Kind: domain.FavoriteProject, ParentID: &parent}, []string{"entity_id required"}},
		{"a cycle with a name", domain.NewFavorite{Kind: domain.FavoriteCycle, EntityID: &target, Name: ptr("Mine")}, []string{"name not_allowed"}},
		{"a view with an empty name", domain.NewFavorite{Kind: domain.FavoriteView, EntityID: &target, Name: ptr("")}, []string{"name not_allowed"}},
		{"a module without its target, with a name", domain.NewFavorite{Kind: domain.FavoriteModule, Name: ptr("a\x00b")},
			[]string{"entity_id required", "name not_allowed"}},
		{"a folder with a target", domain.NewFavorite{Kind: domain.FavoriteFolder, EntityID: &target, Name: ptr("Work")}, []string{"entity_id not_allowed"}},
		{"a folder without a name", domain.NewFavorite{Kind: domain.FavoriteFolder}, []string{"name required"}},
		{"a folder in a folder", domain.NewFavorite{Kind: domain.FavoriteFolder, Name: ptr("Work"), ParentID: &parent}, []string{"parent_id not_allowed"}},
		{"an empty folder name", domain.NewFavorite{Kind: domain.FavoriteFolder, Name: ptr("")}, []string{"name too_short"}},
		{"a blank folder name", domain.NewFavorite{Kind: domain.FavoriteFolder, Name: ptr(" \t\n　")}, []string{"name too_short"}},
		{"a folder name of 256 characters", domain.NewFavorite{Kind: domain.FavoriteFolder, Name: ptr(long + "夹")}, []string{"name too_long"}},
		{"a folder name with NUL", domain.NewFavorite{Kind: domain.FavoriteFolder, Name: ptr("Wo\x00rk")}, []string{"name invalid_format"}},
		{"every problem of a folder", domain.NewFavorite{Kind: domain.FavoriteFolder, EntityID: &target, Name: ptr(" "), ParentID: &parent},
			[]string{"entity_id not_allowed", "name too_short", "parent_id not_allowed"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, problems := domain.CheckNewFavorite(tt.n)
			if fields := fields(problems); !reflect.DeepEqual(fields, tt.want) || !reflect.DeepEqual(got, domain.NewFavorite{}) {
				t.Fatalf("%+v, %v; want nothing and %v", got, fields, tt.want)
			}
		})
	}
}

// A change's fields by the favorite's kind (spec 2.4, 2.7): a favorite of a target moves into a folder or out of it,
// null, and takes any finite sequence; a folder takes a sequence and a name, trimmed, and null as its parent, the none
// it has. {} is a change with nothing to check.
func TestAFavoritePatchsFields(t *testing.T) {
	folder := uuid.NewV7()
	for _, tt := range []struct {
		name string
		kind domain.FavoriteKind
		p    domain.FavoritePatch
	}{
		{"{}", domain.FavoriteProject, domain.FavoritePatch{}},
		{"{} on a folder", domain.FavoriteFolder, domain.FavoritePatch{}},
		{"into a folder", domain.FavoriteCycle, domain.FavoritePatch{SetParentID: true, ParentID: &folder, Sequence: ptr(-1e300)}},
		{"out of its folder", domain.FavoriteView, domain.FavoritePatch{SetParentID: true, Sequence: ptr(0.5)}},
		{"a folder's sequence and null parent", domain.FavoriteFolder, domain.FavoritePatch{SetParentID: true, Sequence: ptr(math.MaxFloat64)}},
	} {
		if got, problems := domain.CheckFavoritePatch(tt.p, tt.kind); problems != nil || !reflect.DeepEqual(got, tt.p) {
			t.Fatalf("%s: %+v, %v; want it as given", tt.name, got, problems)
		}
	}
	got, problems := domain.CheckFavoritePatch(domain.FavoritePatch{Name: ptr("  Home ")}, domain.FavoriteFolder)
	if problems != nil || *got.Name != "Home" {
		t.Fatalf("a folder's name: %+v, %v; want Home", got, problems)
	}
}

// Every problem of a change by the favorite's kind, in the body's order (spec 2.4, 2.7): a folder given a parent, a
// sequence that is no finite number, a name given to a favorite of a target, valid or not, and a folder's name by the
// creation's rules.
func TestAFavoritePatchsProblems(t *testing.T) {
	folder := uuid.NewV7()
	for _, tt := range []struct {
		name string
		kind domain.FavoriteKind
		p    domain.FavoritePatch
		want []string
	}{
		{"a folder into a folder", domain.FavoriteFolder, domain.FavoritePatch{SetParentID: true, ParentID: &folder}, []string{"parent_id not_allowed"}},
		{"NaN", domain.FavoriteModule, domain.FavoritePatch{Sequence: ptr(math.NaN())}, []string{"sequence invalid_format"}},
		{"infinity", domain.FavoriteFolder, domain.FavoritePatch{Sequence: ptr(math.Inf(1))}, []string{"sequence invalid_format"}},
		{"minus infinity", domain.FavoriteProject, domain.FavoritePatch{Sequence: ptr(math.Inf(-1))}, []string{"sequence invalid_format"}},
		{"a project renamed", domain.FavoriteProject, domain.FavoritePatch{Name: ptr("Mine")}, []string{"name not_allowed"}},
		{"a view renamed blank", domain.FavoriteView, domain.FavoritePatch{Name: ptr(" ")}, []string{"name not_allowed"}},
		{"a folder renamed blank", domain.FavoriteFolder, domain.FavoritePatch{Name: ptr(" \t")}, []string{"name too_short"}},
		{"a folder renamed long", domain.FavoriteFolder, domain.FavoritePatch{Name: ptr(strings.Repeat("x", 256))}, []string{"name too_long"}},
		{"a folder renamed with NUL", domain.FavoriteFolder, domain.FavoritePatch{Name: ptr("\x00")}, []string{"name invalid_format"}},
		{"every problem of a folder", domain.FavoriteFolder, domain.FavoritePatch{SetParentID: true, ParentID: &folder, Sequence: ptr(math.NaN()), Name: ptr("")},
			[]string{"parent_id not_allowed", "sequence invalid_format", "name too_short"}},
		{"every problem of a cycle", domain.FavoriteCycle, domain.FavoritePatch{SetParentID: true, ParentID: &folder, Sequence: ptr(math.Inf(1)), Name: ptr("Mine")},
			[]string{"sequence invalid_format", "name not_allowed"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, problems := domain.CheckFavoritePatch(tt.p, tt.kind)
			if fields := fields(problems); !reflect.DeepEqual(fields, tt.want) || !reflect.DeepEqual(got, domain.FavoritePatch{}) {
				t.Fatalf("%+v, %v; want nothing and %v", got, fields, tt.want)
			}
		})
	}
}

// A change replaces each field it gives and leaves the others (spec 2.4). What it changes is told after its check: a
// field given as it is changes nothing, nor does {}, and the use case then answers the favorite as it is, without a
// write. A name differing in case alone, a sequence or a parent of another value, and leaving a folder are changes.
func TestAFavoritePatchApplies(t *testing.T) {
	folder, other := uuid.NewV7(), uuid.NewV7()
	at := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	item := domain.Favorite{ID: uuid.NewV7(), WorkspaceID: uuid.NewV7(), ProjectID: ptr(uuid.NewV7()), UserID: uuid.NewV7(), Kind: domain.FavoriteModule,
		EntityID: ptr(uuid.NewV7()), ParentID: ptr(folder), Sequence: 75535, CreatedAt: at, UpdatedAt: at}
	loose := item
	loose.ParentID = nil
	top := domain.Favorite{ID: uuid.NewV7(), WorkspaceID: item.WorkspaceID, UserID: item.UserID, Kind: domain.FavoriteFolder, Name: "Work",
		Sequence: 65535, CreatedAt: at, UpdatedAt: at}
	for _, tt := range []struct {
		name    string
		of      domain.Favorite
		patch   domain.FavoritePatch
		want    func(*domain.Favorite)
		changed bool
	}{
		{"{}", item, domain.FavoritePatch{}, func(*domain.Favorite) {}, false},
		{"every field as it is", item, domain.FavoritePatch{SetParentID: true, ParentID: ptr(folder), Sequence: ptr(75535.0)}, func(*domain.Favorite) {}, false},
		{"a folder's every field as it is, once checked", top, domain.FavoritePatch{SetParentID: true, Sequence: ptr(65535.0), Name: ptr(" Work\t")},
			func(*domain.Favorite) {}, false},
		{"another folder", item, domain.FavoritePatch{SetParentID: true, ParentID: ptr(other)}, func(f *domain.Favorite) { f.ParentID = ptr(other) }, true},
		{"out of its folder", item, domain.FavoritePatch{SetParentID: true}, func(f *domain.Favorite) { f.ParentID = nil }, true},
		{"into a folder", loose, domain.FavoritePatch{SetParentID: true, ParentID: ptr(other)}, func(f *domain.Favorite) { f.ParentID = ptr(other) }, true},
		{"null at the top", loose, domain.FavoritePatch{SetParentID: true}, func(*domain.Favorite) {}, false},
		{"another sequence", item, domain.FavoritePatch{Sequence: ptr(75535.5)}, func(f *domain.Favorite) { f.Sequence = 75535.5 }, true},
		{"renamed in case alone", top, domain.FavoritePatch{Name: ptr("WORK")}, func(f *domain.Favorite) { f.Name = "WORK" }, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p, problems := domain.CheckFavoritePatch(tt.patch, tt.of.Kind)
			if problems != nil {
				t.Fatal(problems)
			}
			before := tt.of
			want := tt.of
			tt.want(&want)
			got, changed := p.Apply(tt.of)
			if !reflect.DeepEqual(got, want) || changed != tt.changed {
				t.Fatalf("Apply = %+v, %v; want %+v, %v", got, changed, want, tt.changed)
			}
			if !reflect.DeepEqual(tt.of, before) {
				t.Fatalf("Apply changed the favorite it was given: %+v", tt.of)
			}
		})
	}
}

// A new favorite's sequence (spec 2.5, D33, PE-7): 10000 past the greatest of one's undeleted favorites in the
// workspace, any finite number a change has left, or 65535 for one's first; so the newest is listed first. It is
// finite for every finite greatest, as the column's CHECK holds.
func TestANewFavoritesSequence(t *testing.T) {
	for _, tt := range []struct {
		greatest *float64
		want     float64
	}{
		{nil, 65535},
		{ptr(65535.0), 75535},
		{ptr(-20000.0), -10000},
		{ptr(0.25), 10000.25},
		{ptr(math.MaxFloat64), math.MaxFloat64},
	} {
		if got := domain.SequenceAfter(tt.greatest); got != tt.want || math.IsInf(got, 0) {
			t.Errorf("SequenceAfter(%v) = %v; want %v", tt.greatest, got, tt.want)
		}
	}
}

// The favorites' two codes (spec 2.4): every 404 on /favorites/{favorite_id} is navigation.favorite_not_found, and a
// folder's name that another of one's folders in the workspace has, in any case, is 409
// navigation.favorite_folder_name_taken.
func TestTheFavoritesErrors(t *testing.T) {
	for _, tt := range []struct {
		err    *shared.Error
		status int
		code   string
	}{
		{domain.ErrFavoriteNotFound, 404, "navigation.favorite_not_found"},
		{domain.ErrFavoriteFolderNameTaken, 409, "navigation.favorite_folder_name_taken"},
	} {
		if tt.err.ProblemStatus() != tt.status || tt.err.ProblemCode() != tt.code || tt.err.Detail == "" {
			t.Errorf("%+v; want %d %s with a detail", tt.err, tt.status, tt.code)
		}
	}
}
