package bootstrap

import (
	"maps"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The list and the decision agree (M3 design 3.4, 9.3): for every account
// of the matrix, of each kind of 9.2, acme's projects that listProjects
// lists, archived or not, are exactly those that getProject lets him read
// in the same state, partingStates' among them: a guest's ended (endings)
// and deleted memberships, a member's deleted one, an active one without
// display settings. The list applies the visibility in its query
// (domain.Visibility), the decision in access's rule: here the two cannot
// part. Neither side is empty for every account, nor full.
func TestListingProjectsIsReadingEach(t *testing.T) {
	d := prepareMatrix(t)
	contract := apitest.Load(t)
	base := startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), nil), migrations.FS())
	s := d.seeded.in(t)
	projects := map[string]bool{"acme/public": false, "acme/private": false, "acme/archived": true} // key: archived
	var sizes []int
	for _, account := range matrixAccounts {
		token := d.tokens[account]
		for _, archived := range []bool{false, true} {
			status, body := call(t, contract, http.MethodGet, base+"/api/v0/workspaces/acme/projects?archived="+strconv.FormatBool(archived), token, "")
			var list struct {
				Data []struct {
					ID uuid.UUID `json:"id"`
				} `json:"data"`
			}
			if status == http.StatusOK {
				decodeAnswer(t, body, &list)
			}
			var listed, read []uuid.UUID
			for _, p := range list.Data {
				listed = append(listed, p.ID)
			}
			for _, key := range slices.Sorted(maps.Keys(projects)) {
				id := s.project(key)
				if status, _ := call(t, contract, http.MethodGet, base+"/api/v0/projects/"+id.String(), token, ""); status == http.StatusOK &&
					projects[key] == archived {
					read = append(read, id)
				}
			}
			slices.SortFunc(listed, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
			slices.SortFunc(read, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
			if !slices.Equal(listed, read) {
				t.Errorf("%s, archived %v: lists %v, reads %v", account, archived, listed, read)
			}
			sizes = append(sizes, len(read))
		}
	}
	if slices.Min(sizes) != 0 || slices.Max(sizes) != 2 {
		t.Errorf("the accounts read %v projects; want some none and some both of acme's", sizes)
	}
}

// The workspace's list of states and each project's agree (M3 design 3.4,
// 9.3): for every account of the matrix, the states listWorkspaceStates
// lists of acme are exactly those listStates lists of each of acme's
// projects, by project in the order of their ids: the projects whose states
// state.list lets him list, partingStates' memberships among them, the
// archived one's none. The workspace's list reads the membership in its
// query, each project's list in access's rule: here the two cannot part.
// Some accounts list no state, some both unarchived projects'.
func TestListingWorkspaceStatesIsListingEachProjects(t *testing.T) {
	d := prepareMatrix(t)
	contract := apitest.Load(t)
	base := startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), nil), migrations.FS())
	s := d.seeded.in(t)
	var sizes []int
	for _, account := range matrixAccounts {
		token := d.tokens[account]
		listed := stateIDs(t, contract, base+"/api/v0/workspaces/acme/states", token)
		var each []uuid.UUID
		for _, key := range byProjectID(s, "acme/public", "acme/private", "acme/archived") {
			each = append(each, stateIDs(t, contract, base+"/api/v0/projects/"+s.project(key).String()+"/states", token)...)
		}
		if !slices.Equal(listed, each) {
			t.Errorf("%s: acme's list %v, each project's %v", account, listed, each)
		}
		sizes = append(sizes, len(each))
	}
	if slices.Min(sizes) != 0 || slices.Max(sizes) != 2*len(listedStates) {
		t.Errorf("the accounts list %v states; want some none and some both unarchived projects' %d", sizes, 2*len(listedStates))
	}
}

// Each workspace's list of states holds its own projects' alone (M3 design
// 3.17, 9.3), on memberWorld, whose bob and carol are members of acme's Web
// and Ops and of beta's Lab: for each of its accounts, acme's list is
// exactly what listStates lists of Web and of Ops, by project id, and
// beta's what it lists of Lab. A list that took in the projects of another
// workspace would hold Lab's in acme's, Web's and Ops's in beta's; Web has
// a deleted state, Dropped, which bob created and deleted, and which
// neither list holds. The matrix's accounts are members of one live
// workspace's projects alone, and no live project of it has a deleted
// state.
func TestListingWorkspaceStatesKeepsToItsWorkspace(t *testing.T) {
	w := newMemberWorld(t)
	status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+w.web.String()+"/states", w.tokens["bob"],
		`{"name":"Dropped","color":"#000000","group":"backlog"}`)
	if status != http.StatusCreated {
		t.Fatalf("bob's creating Dropped = %d %s", status, body)
	}
	var dropped struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &dropped)
	if status, body := call(t, w.contract, http.MethodDelete, w.base+"/api/v0/states/"+dropped.ID.String(), w.tokens["bob"],
		""); status != http.StatusNoContent {
		t.Fatalf("bob's deleting Dropped = %d %s", status, body)
	}
	acme := slices.SortedFunc(slices.Values([]uuid.UUID{w.web, w.ops}), func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	for _, p := range []uuid.UUID{w.web, w.ops, w.lab} {
		if len(stateIDs(t, w.contract, w.base+"/api/v0/projects/"+p.String()+"/states", w.tokens["bob"])) == 0 {
			t.Fatalf("bob lists no state of %s; the test needs him to list each project's", p)
		}
	}
	for _, name := range []string{"alice", "bob", "carol", "dave", "erin", "gina"} {
		for _, ws := range []struct {
			slug     string
			projects []uuid.UUID
		}{{"acme", acme}, {"beta", []uuid.UUID{w.lab}}} {
			listed := stateIDs(t, w.contract, w.base+"/api/v0/workspaces/"+ws.slug+"/states", w.tokens[name])
			var each []uuid.UUID
			for _, p := range ws.projects {
				each = append(each, stateIDs(t, w.contract, w.base+"/api/v0/projects/"+p.String()+"/states", w.tokens[name])...)
			}
			if !slices.Equal(listed, each) {
				t.Errorf("%s: %s's list %v, each project's %v", name, ws.slug, listed, each)
			}
		}
	}
}

// stateIDs are the ids of the states GET url answers token, in its order;
// none when it answers other than 200.
func stateIDs(t *testing.T, contract *apitest.Contract, url, token string) []uuid.UUID {
	t.Helper()
	status, body := call(t, contract, http.MethodGet, url, token, "")
	if status != http.StatusOK {
		return nil
	}
	var list struct {
		Data []struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}
	decodeAnswer(t, body, &list)
	ids := make([]uuid.UUID, len(list.Data))
	for i, st := range list.Data {
		ids[i] = st.ID
	}
	return ids
}
