package bootstrap

import (
	"fmt"
	"testing"
	"uuid"
)

// Even distinct project names cannot consume the same new_project preparation
// twice. The loser must leave no project, membership, preference or default state.
func TestProjectCoverCreateRace(t *testing.T) {
	for _, firstName := range []string{"Alpha", "Beta"} {
		t.Run(firstName, func(t *testing.T) {
			w := newObservedIssueWorld(t)
			image := uploadedAssetHTTP(t, w, "alice", "PROJECT_COVER", assetScopeBody("new_project", w.acme), "new.png", "image/png", assetTestPNG(t)).Asset.ID
			secondName := "Beta"
			if firstName == secondName {
				secondName = "Alpha"
			}
			path := "/api/v0/workspaces/acme/projects"
			body := func(name string) []byte {
				return []byte(fmt.Sprintf(`{"name":%q,"identifier":%q,"cover_image_asset_id":%q}`, name, name, image))
			}
			first := newRequest(t, "POST", w.base+path, w.tokens["alice"], body(firstName))
			second := newRequest(t, "POST", w.base+path, w.tokens["alice"], body(secondName))
			firstProbe, secondProbe := w.observe(first), w.observe(second)
			held := holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", image)
			defer requestRelease(t, held)
			won := sendInBackground(first)
			pid := requestBlocked(t, w.pool, firstProbe, held.Conn().PgConn().PID(), "file_assets")
			lost := sendInBackground(second)
			requestBlocked(t, w.pool, secondProbe, pid, "users")
			requestRelease(t, held)
			got := imageRaceAnswer(t, w, first, won, 201)
			refused := imageRaceAnswer(t, w, second, lost, 404)
			if problemCode(t, refused.body) != "asset.not_found" {
				t.Fatal(string(refused.body))
			}
			checkImageReply(t, w, string(got.body), "cover_image", image, assetTestPNG(t))
			var project struct{ ID uuid.UUID }
			decodeAnswer(t, string(got.body), &project)
			var complete bool
			if err := w.pool.QueryRow(t.Context(), `SELECT
 (SELECT count(*) FROM projects WHERE name=ANY($2::text[]))=1
 AND (SELECT count(*) FROM project_members WHERE project_id=$1)=1
 AND (SELECT count(*) FROM project_user_properties WHERE project_id=$1)=1
 AND (SELECT count(*) FROM states WHERE project_id=$1)=6
 AND (SELECT project_id=$1 AND scope_kind='project' AND bound_at IS NOT NULL FROM file_assets WHERE id=$3)`, project.ID, []string{"Alpha", "Beta"}, image).Scan(&complete); err != nil || !complete {
				t.Fatal("concurrent loser left a partial graph", complete, err)
			}
		})
	}
}
