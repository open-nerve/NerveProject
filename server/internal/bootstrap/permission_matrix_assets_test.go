package bootstrap

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// Every column owns an explicit pending preparation, before any bytes arrive.
// Visible read/status conflicts and forbidden writes distinguish authorization
// from upload state. Real successful byte chains are covered by F1/F3 and the
// complete/duplicate HTTP suites; this matrix does not fake published objects.
func assetMatrixRows() []matrixRow {
	notFound := cell{404, "asset.not_found"}
	pending := cell{409, "asset.not_uploaded"}
	missing := cell{409, "asset.object_missing"}
	cells := func(admin, member, guest cell) map[caller]cell {
		return workspaceCells(admin, member, guest, notFound, notFound, notFound)
	}
	body := func(c caller, s seeded) string {
		return fmt.Sprintf(`{"purpose":"WORKSPACE_LOGO","scope":{"kind":"workspace","id":%q}}`, s.workspace(workspaceOf(c)))
	}
	own := func(method, suffix, requestBody string) func(caller, seeded) (string, string, string) {
		return func(c caller, s seeded) (string, string, string) {
			return method, "/api/v0/assets/" + s.asset(c).String() + suffix, requestBody
		}
	}
	return []matrixRow{
		{op: "listDraftAssetPreparations", request: func(c caller, s seeded) (string, string, string) {
			return "GET", preparationsPath(s.drafts[workspaceOf(c)+"/"+string(c)]), ""
		}, cells: cells(cellOK, cellOK, notFound), check: func(t *testing.T, _ caller, _ seeded, answer string) {
			var page preparationPageHTTP
			decodeAnswer(t, answer, &page)
			if page.Data == nil || len(page.Data) != 0 || page.NextCursor != nil {
				t.Fatal("empty private preparation list differs from its contract", answer)
			}
		}},
		{op: "createAsset", write: true, request: func(c caller, s seeded) (string, string, string) {
			return "POST", "/api/v0/assets", assetPrepareBody("WORKSPACE_LOGO", assetScopeBody("workspace", s.workspace(workspaceOf(c))), "logo.png", "image/png", 3)
		}, cells: cells(cellCreated, cellForbidden, cellForbidden)},
		{op: "getAsset", request: own("GET", "", ""), cells: cells(cellOK, cellOK, cellOK), check: func(t *testing.T, c caller, s seeded, answer string) {
			var got preparedAssetAnswer
			decodeAnswer(t, answer, &got.Asset)
			if got.Asset.ID != s.asset(c) || got.Asset.Scope.Kind != "workspace" || got.Asset.Scope.ID == nil || *got.Asset.Scope.ID != s.workspace(workspaceOf(c)) || got.Asset.State != "pending" || got.Asset.IsBound {
				t.Fatal("metadata did not describe the column's private pending preparation", answer)
			}
		}},
		{op: "resolveAssetContent", variant: "pending before transfer", request: own("GET", "/content", ""), cells: cells(pending, pending, pending)},
		{op: "completeAsset", variant: "staging bytes absent", write: true, request: own("POST", "/complete", `{}`), cells: cells(missing, cellForbidden, cellForbidden)},
		{op: "duplicateAsset", variant: "pending source", write: true, request: func(c caller, s seeded) (string, string, string) {
			return "POST", duplicateAssetPath(s.asset(c)), body(c, s)
		}, cells: cells(pending, cellForbidden, cellForbidden)},
		{op: "deleteAsset", write: true, request: own("DELETE", "", ""), cells: cells(cellNoContent, cellForbidden, cellForbidden)},
	}
}

func (s seeded) asset(c caller) uuid.UUID {
	id, ok := s.assets[workspaceOf(c)+"/"+string(c)]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no private asset in %s for %s is seeded", workspaceOf(c), c)
	}
	return id
}

func seedMatrixAssets(t *testing.T, pool *pgxpool.Pool, s seeded, at time.Time) {
	t.Helper()
	expires := at.Add(15 * time.Minute)
	for _, c := range workspaceColumns {
		id, by, workspace := s.asset(c), s.account(c), s.workspace(workspaceOf(c))
		key, err := domain.NewObjectKey('u', id, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		row := domain.Asset{ID: id, Purpose: domain.WorkspaceLogo, Name: "logo.png", Size: 3, ContentType: "image/png",
			Placement: domain.Placement{Scope: domain.Scope{Kind: domain.ScopeWorkspace, ID: workspace}, UserID: by, WorkspaceID: workspace},
			CreatedAt: at, UpdatedAt: at, CreatedByID: &by, UpdatedByID: &by, UploadExpiresAt: &expires,
			Metadata: domain.StorageMetadata{SchemaVersion: 1, Origin: domain.OriginUpload, Candidates: []domain.Candidate{},
				Staging: &domain.ObjectRecord{Key: key, Phase: domain.ObjectActive, SafeAfter: expires.Add(time.Minute), Size: 3, ContentType: "image/png"}},
		}
		if _, err := assetpg.New(pool).CreateAsset(t.Context(), row); err != nil {
			t.Fatal(err)
		}
	}
}

// An asset path is a checked private target, never a new global {id} exemption.
func TestAssetMatrixCoverageGuards(t *testing.T) {
	s := newSeeded().in(t)
	var row matrixRow
	for _, candidate := range assetMatrixRows() {
		if candidate.op == "getAsset" {
			row = candidate
		}
	}
	ops := []apitest.Operation{{ID: "getAsset", Tags: []string{"asset"}, Method: "GET", Path: "/api/v0/assets/{id}"}}
	if got := matrixViolations(ops, matrixExemptions{}, []matrixRow{row}, s, nil); len(got) != 0 {
		t.Fatal(got)
	}
	if got := matrixViolations(ops, matrixExemptions{}, nil, s, nil); !slices.Equal(got, []string{"operation getAsset, tagged [asset], has no row"}) {
		t.Fatal(got)
	}
	correct := row.request
	row.request = func(c caller, s seeded) (string, string, string) {
		if c == callerDeleted {
			return "GET", "/api/v0/assets/" + s.asset(callerAdmin).String(), ""
		}
		return correct(c, s)
	}
	if got := matrixViolations(ops, matrixExemptions{}, []matrixRow{row}, s, nil); len(got) != 1 || !strings.Contains(got[0], "is not its column's private asset in gone") {
		t.Fatal("wrong workspace/private owner was accepted", got)
	}
	row.request = correct
	delete(row.cells, callerGuest)
	if got := matrixViolations(ops, matrixExemptions{}, []matrixRow{row}, s, nil); !slices.Equal(got, []string{"row getAsset has no cell for guest"}) {
		t.Fatal(got)
	}
}
