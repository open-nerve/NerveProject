package bootstrap

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

// TestFilesCleanupDrill is the M5 acceptance drill (P8 T10). One workspace holds
// every content purpose, both workspace images, an unbound preparation, a failed
// copy, an unfinished upload and a slow upload over 1 MiB and 15 s. Beside it is
// an untouched neighbour. The drill deletes the workspace and takes the business
// clock past its 7- and 60-day marks on the production composition and its
// River jobs, failing one object deletion on the way. Properties that phase tests
// prove are not proven again: the P8 stage record names the tests rerun with it.
func TestFilesCleanupDrill(t *testing.T) {
	if os.Getenv("NERVE_FILES_DRILL") != "1" {
		t.Skip("NERVE_FILES_DRILL=1 required: the drill is acceptance evidence, not a default test")
	}
	for _, driver := range []string{"local", "s3"} {
		t.Run(driver, func(t *testing.T) { runFilesCleanupDrill(t, driver) })
	}
}

// drillTarget is workspace A. Its project cover is the last file made, so every
// sweep reaches it last.
type drillTarget struct {
	tree                                        contentAssetTree
	slowIssue                                   uuid.UUID
	logo, preparation, source, copy, unfinished uuid.UUID
	slow, cover                                 uuid.UUID
	slowSHA                                     string
	slowTook                                    time.Duration
}

func (d *filesDrill) target(t *testing.T) drillTarget {
	t.Helper()
	base, status := d.raw()
	png := assetTestPNG(t)
	g := drillTarget{tree: newContentAssetTreeAt(t, d.issueWorld, base, status, d.web, "acme")}
	g.logo = uploadedAssetHTTPAt(t, d.issueWorld, base, status, "alice", "WORKSPACE_LOGO", assetScopeBody("workspace", d.acme), "logo.png", "image/png", png).Asset.ID
	d.send(t, "alice", "PATCH", "/api/v0/workspaces/acme", fmt.Sprintf(`{"logo_asset_id":%q}`, g.logo), 200)
	g.preparation = uploadedAssetHTTPAt(t, d.issueWorld, base, status, "bob", "DRAFT_ISSUE_DESCRIPTION", assetScopeBody("draft", g.tree.draft), "prepared.png", "image/png", png).Asset.ID
	if page := preparationsHTTP(t, d.issueWorld, "bob", g.tree.draft, ""); len(page.Data) != 1 || page.Data[0].ID != g.preparation {
		t.Fatalf("the unbound image is not the draft's only preparation: %+v", page)
	}
	// Copying bytes that are no image into an image purpose fails on the
	// source's image check, after the target row reserved its candidate: the
	// record, whose object was never written, is left to recovery.
	g.source = uploadedAssetHTTPAt(t, d.issueWorld, base, status, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", g.tree.issue), "fake.png", "image/png", []byte("looks like no PNG")).Asset.ID
	failed := assetAPICall(t, d.issueWorld, "bob", "POST", duplicateAssetPath(g.source), fmt.Sprintf(`{"purpose":"ISSUE_DESCRIPTION","scope":%s,"name":"copied.png"}`, assetScopeBody("issue", g.tree.issue)), 422)
	copies := queryIDs(t, d.pool, "SELECT id FROM file_assets WHERE workspace_id=$1 AND entity_type='ISSUE_DESCRIPTION' AND NOT is_uploaded", d.acme)
	if problemCode(t, []byte(failed)) != "asset.content_invalid" || len(copies) != 1 {
		t.Fatalf("failed copy %s left %v", failed, copies)
	}
	g.copy = copies[0]
	g.unfinished = prepareAssetHTTP(t, d.issueWorld, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", g.tree.issue), "pending.txt", "text/plain", 3).Asset.ID
	g.slowIssue = d.newWorkItemBy(t, "bob", d.web, `{"name":"Slow upload"}`)
	g.slow, g.slowSHA, g.slowTook = d.slowUpload(t, g.slowIssue)
	g.cover = uploadedAssetHTTPAt(t, d.issueWorld, base, status, "alice", "PROJECT_COVER", assetScopeBody("project", d.web), "cover.png", "image/png", png).Asset.ID
	d.send(t, "alice", "PATCH", "/api/v0/projects/"+d.web.String(), fmt.Sprintf(`{"cover_image_asset_id":%q}`, g.cover), 200)
	return g
}

func (d *filesDrill) neighbour(t *testing.T) drillNeighbour {
	t.Helper()
	base, status := d.raw()
	var other struct{ ID uuid.UUID }
	decodeAnswer(t, d.send(t, "bob", "POST", "/api/v0/workspaces", `{"name":"Neighbour","slug":"neighbour"}`, 201), &other)
	nw := d.issueWorld
	nw.acme = other.ID
	nw.web = createdProject(t, d.contract, d.base, d.tokens["bob"], "neighbour", "Keep", "KEEP")
	n := drillNeighbour{tree: newContentAssetTreeAt(t, nw, base, status, nw.web, "neighbour")}
	n.avatar = uploadedAssetHTTPAt(t, d.issueWorld, base, status, "alice", "USER_AVATAR", `{"kind":"user"}`, "avatar.png", "image/png", assetTestPNG(t)).Asset.ID
	d.send(t, "alice", "PATCH", "/api/v0/me", fmt.Sprintf(`{"avatar_asset_id":%q}`, n.avatar), 200)
	return n
}

func runFilesCleanupDrill(t *testing.T, driver string) {
	d := newFilesDrill(t, driver)
	neighbour := d.neighbour(t)
	g := d.target(t)
	evidence := map[string]any{"driver": driver, "slow_upload": map[string]any{"bytes": 3 << 20, "took": g.slowTook.String(), "sha256": g.slowSHA}}

	// Every completed upload still has its staging copy and the failed copy its
	// candidate record; the first restart, two runtime hours on, reclaims both.
	staging := map[assetdomain.ObjectKey]string{}
	candidates := 0
	for _, row := range d.workspaceAssets(t, d.acme) {
		if row.Metadata.Staging != nil && row.Uploaded {
			staging[row.Metadata.Staging.Key] = ""
			if objectDigest(t, d.objects, row.Metadata.Staging.Key) == "" {
				t.Fatal("staging copy missing before the first restart", row.ID)
			}
		}
		for _, candidate := range row.Metadata.Candidates {
			candidates++
			if row.ID != g.copy || objectDigest(t, d.objects, candidate.Key) != "" {
				t.Fatalf("unexpected candidate %s of file %s", candidate.Key, row.ID)
			}
		}
	}
	if candidates != 1 {
		t.Fatalf("the failed copy holds %d candidates, want 1", candidates)
	}
	d.restart(t, 0).clean(t, "first restart")
	d.expectBytes(t, "first restart", staging)
	start := d.workspaceAssets(t, d.acme)
	finals := map[assetdomain.ObjectKey]string{}
	for id, row := range start {
		unfinished, copy := id == g.unfinished, id == g.copy
		if row.DeletedAt != nil || len(row.Metadata.Candidates) != 0 || (row.Metadata.Staging != nil) != unfinished || (row.Metadata.Final == nil) != (unfinished || copy) {
			t.Fatalf("after the first restart file %s keeps unexpected objects: %+v", id, row)
		}
		if row.Metadata.Final != nil {
			finals[row.Metadata.Final.Key] = row.Metadata.Final.SHA256
		}
	}
	if len(start) != 13 || finals[start[g.slow].Metadata.Final.Key] != g.slowSHA {
		t.Fatalf("workspace A holds %d files, want 13, or the slow upload's digest changed", len(start))
	}
	d.expectBytes(t, "first restart", finals)
	keep := d.state(t, neighbour)
	unchanged := func(step string) {
		t.Helper()
		if got := d.state(t, neighbour); !reflect.DeepEqual(got, keep) {
			t.Fatalf("%s changed the neighbour:\n got %+v\nwant %+v", step, got, keep)
		}
	}
	hosts := map[string][]uuid.UUID{"workspaces": {d.acme}, "projects": {d.web}, "issues": {g.tree.issue, g.slowIssue}, "issue_comments": {g.tree.comment}, "draft_issues": {g.tree.draft}}
	present := func(step string, want map[string]int) {
		t.Helper()
		if got := d.hosts(t, hosts); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: hosts %v, want %v", step, got, want)
		}
	}
	references := func(step string, logo, cover *uuid.UUID) {
		t.Helper()
		var gotLogo, gotCover *uuid.UUID
		if err := d.pool.QueryRow(t.Context(), "SELECT (SELECT logo_asset_id FROM workspaces WHERE id=$1),(SELECT cover_image_asset_id FROM projects WHERE id=$2)", d.acme, d.web).Scan(&gotLogo, &gotCover); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotLogo, logo) || !reflect.DeepEqual(gotCover, cover) {
			t.Fatalf("%s: logo %v cover %v, want %v %v", step, gotLogo, gotCover, logo, cover)
		}
	}
	all := map[string]int{"workspaces": 1, "projects": 1, "issues": 2, "issue_comments": 1, "draft_issues": 1}
	references("first restart", &g.logo, &g.cover)

	// Deleting A ends every new authorization with the request.
	d.send(t, "alice", "DELETE", "/api/v0/workspaces/acme", "", 204)
	assetAPICall(t, d.issueWorld, "bob", "GET", "/api/v0/assets/"+g.slow.String()+"/content", "", 404)
	assetAPICall(t, d.issueWorld, "bob", "POST", duplicateAssetPath(g.tree.rows[0].ID), fmt.Sprintf(`{"purpose":"ISSUE_ATTACHMENT","scope":%s}`, assetScopeBody("issue", g.slowIssue)), 404)
	assetAPICall(t, d.issueWorld, "bob", "POST", "/api/v0/assets", assetPrepareBody("ISSUE_ATTACHMENT", assetScopeBody("issue", g.tree.issue), "late.txt", "text/plain", 3), 404)
	preparationRefusal(t, d.issueWorld, "bob", g.tree.draft)
	waitWorkspaceDeletion(t, d.pool, d.acme)
	var at time.Time
	if err := d.pool.QueryRow(t.Context(), "SELECT deleted_at FROM workspaces WHERE id=$1", d.acme).Scan(&at); err != nil {
		t.Fatal(err)
	}
	stamp := d.workspaceAssets(t, d.acme)[g.cover].DeletedAt
	if stamp == nil || !stamp.Equal(at) {
		t.Fatalf("deletion stamp %v, want the workspace's %s", stamp, at)
	}
	alice := d.ids["alice"]
	want := map[uuid.UUID]assetdomain.Asset{}
	for id, row := range start {
		row.DeletedAt, row.DeletedByID, row.UpdatedAt, row.UpdatedByID = stamp, &alice, *stamp, &alice
		want[id] = row
	}
	d.expectRows(t, "deleted", d.acme, want)
	d.expectBytes(t, "deleted", finals)
	for table, ids := range hosts {
		var stamped int
		if err := d.pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE id=ANY($1::uuid[]) AND deleted_at=$2", ids, at).Scan(&stamped); err != nil || stamped != len(ids) {
			t.Fatalf("%s: %d of %v carry the workspace's deletion time (%v)", table, stamped, ids, err)
		}
	}
	references("deleted", &g.logo, &g.cover)
	unchanged("deleting A")

	// Past seven days no tombstone is stamped again; deletion has made the
	// unfinished upload's never-written staging record reclaimable.
	d.restart(t, 8*24*time.Hour).clean(t, "day 8")
	row := want[g.unfinished]
	row.Metadata.Staging = nil
	want[g.unfinished] = row
	d.expectRows(t, "day 8", d.acme, want)
	d.expectBytes(t, "day 8", finals)
	present("day 8", all)
	references("day 8", &g.logo, &g.cover)
	unchanged("day 8")

	// Past sixty days the sweep deletes the final objects; the cover's deletion
	// fails once. The purge then removes every tombstone without objects and its
	// hosts, and skips the cover, its project and the workspace.
	cover := want[g.cover]
	day61 := d.restart(t, 53*24*time.Hour, cover.Metadata.Final.Key)
	sweep, purge := day61.cleanup.sweep(t), day61.purge.purged(t)
	if day61.cleanup.Attempt != 2 || len(day61.cleanup.Errors) != 1 || !strings.Contains(day61.cleanup.Errors[0], errImageDelete.Error()) || !sweep.Complete || sweep.Failed != 1 {
		t.Fatalf("day 61: cleanup attempt=%d errors=%q output=%+v", day61.cleanup.Attempt, day61.cleanup.Errors, sweep)
	}
	if day61.purge.Attempt != 1 || len(day61.purge.Errors) != 0 || !purge.Complete || purge.Blocked == 0 {
		t.Fatalf("day 61: purge attempt=%d errors=%q output=%+v", day61.purge.Attempt, day61.purge.Errors, purge)
	}
	final := *cover.Metadata.Final
	final.Phase = assetdomain.ObjectReclaiming
	cover.Metadata.Final = &final
	d.expectRows(t, "day 61", d.acme, map[uuid.UUID]assetdomain.Asset{g.cover: cover})
	gone := map[assetdomain.ObjectKey]string{}
	for key := range finals {
		gone[key] = ""
	}
	gone[final.Key] = finals[final.Key]
	d.expectBytes(t, "day 61", gone)
	present("day 61", map[string]int{"workspaces": 1, "projects": 1, "issues": 0, "issue_comments": 0, "draft_issues": 0})
	references("day 61", nil, &g.cover)
	unchanged("day 61")

	// The next cycle retries the reclaiming record and the purge finishes A.
	d.restart(t, 0).clean(t, "next cycle")
	d.expectRows(t, "next cycle", d.acme, nil)
	gone[final.Key] = ""
	d.expectBytes(t, "next cycle", gone)
	present("next cycle", map[string]int{"workspaces": 0, "projects": 0, "issues": 0, "issue_comments": 0, "draft_issues": 0})
	unchanged("next cycle")

	runs, longest := d.runs(t)
	evidence["pages"] = d.pages.bounded(t, d.cfg.Files.CleanupBatch, d.cfg.Cleanup.PurgeBatch)
	evidence["job_runs"] = map[string]any{"count": runs, "longest": longest.String(), "budget": d.cfg.Files.CleanupBudget.String()}
	evidence["day61"] = map[string]any{"cleanup": sweep, "cleanup_errors": day61.cleanup.Errors, "purge": purge}
	evidence["files"] = map[string]int{"workspace_a": len(start), "neighbour": len(keep.Bytes), "objects_reclaimed": len(staging) + len(finals)}
	report, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("files cleanup drill: %s", report)
}
