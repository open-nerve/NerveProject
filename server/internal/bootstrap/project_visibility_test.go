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
