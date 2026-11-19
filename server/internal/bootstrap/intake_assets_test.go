package bootstrap

import (
	"bytes"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fileAssetRows is every row of file_assets, in id order, as JSON.
func fileAssetRows(t *testing.T, w issueWorld) string {
	t.Helper()
	var rows string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(a) ORDER BY id), '[]'::jsonb)::text FROM file_assets a").Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// assetRefused has the account name send method path with body to the
// asset API, checks it is refused with status and code, and that no row
// of file_assets changed.
func assetRefused(t *testing.T, w issueWorld, name, method, path, body string, status int, code string) {
	t.Helper()
	before := fileAssetRows(t, w)
	if got := refusalOf(t, []byte(assetAPICall(t, w, name, method, path, body, status)), false); got != code {
		t.Errorf("%s %s as %s = %s, want %s", method, path, name, got, code)
	}
	if fileAssetRows(t, w) != before {
		t.Errorf("%s %s as %s, refused, changed file_assets", method, path, name)
	}
}

// attachedTo checks the account name lists item's attachments as want, in
// any order (getIssueAttachments).
func attachedTo(t *testing.T, w issueWorld, name string, item uuid.UUID, want ...uuid.UUID) {
	t.Helper()
	var got []uuid.UUID
	for _, row := range attachmentListRead(t, w, name, "/api/v0/issues/"+item.String()+"/attachments", http.StatusOK).Data {
		got = append(got, row.ID)
	}
	slices.SortFunc(got, uuid.UUID.Compare)
	want = slices.Clone(want)
	slices.SortFunc(want, uuid.UUID.Compare)
	if !slices.Equal(got, want) {
		t.Errorf("%s lists %s's attachments %v, want %v", name, item, got, want)
	}
}

// H11's images and attachments of a work item in the intake through the
// app (M7 P2c spec 2.2, 2.5, 4). In Web, its intake turned on, walt, its
// guest whom it shows only his own work items, submits Pending, Declined
// and Accepted. On Pending he uploads a description image and binds it
// through updateIntakeIssue, adds an attachment, and deletes the one
// alice, Web's admin, added. bob, Web's member, who submitted none of
// them, is refused a preparation and a deletion on Pending, 403, writing
// nothing, and reads its attachments and their content. Once alice
// declines Declined, each of the four writes is refused 409
// issue.intake_closed, alice's too, writing nothing; its attachment still
// reads. Once she accepts Accepted, issue.update decides: walt is
// refused, bob writes. On a work item walt submits and alice snoozes
// through the issue module on a clock two hours back (the app's clock
// passed its snooze an hour ago, and the app reads it pending), walt
// writes: its stored status alone is judged. With Web's intake turned
// off, last, a write on Pending or Declined is refused 409
// issue.intake_disabled, bob's 403 first; Accepted is written; the reads
// are unchanged; a comment on Declined and a new one on Pending take
// bob's attachment and image. The subtests are one story told in order,
// not to be run alone: the later ones read the image and attachment the
// first one makes, and the last one writes on Accepted once the third
// has accepted it.
func TestIntakeItemAssetsThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	w.intakeOn(t)
	pending, declined, accepted := w.submit(t, "walt", `{"name":"Pending"}`).ID, w.submit(t, "walt", `{"name":"Declined"}`).ID,
		w.submit(t, "walt", `{"name":"Accepted"}`).ID
	// No run the app starts with may change file_assets while a refusal
	// compares it, nor queue among the deletion's locks.
	waitForStartupRuns(t, w.pool)
	notes := []byte("walt's notes")
	prepare := func(item uuid.UUID) string {
		return assetPrepareBody("ISSUE_ATTACHMENT", attachmentScope("issue", item), "notes.txt", "text/plain", int64(len(notes)))
	}
	attach := func(t *testing.T, name string, item uuid.UUID) uuid.UUID {
		t.Helper()
		a := uploadedAssetHTTP(t, w, name, "ISSUE_ATTACHMENT", attachmentScope("issue", item), "notes.txt", "text/plain", notes)
		if !a.Asset.IsBound {
			t.Fatalf("%s's attachment on %s completed unbound", name, item)
		}
		return a.Asset.ID
	}
	var image, attachment uuid.UUID

	t.Run("its guest submitter's image and attachment", func(t *testing.T) {
		image = uploadedAssetHTTP(t, w, "walt", "ISSUE_DESCRIPTION", attachmentScope("issue", pending), "body.png", "image/png", assetTestPNG(t)).Asset.ID
		var changed struct {
			UpdatedAt time.Time `json:"updated_at"`
		}
		decodeAnswer(t, w.send(t, "walt", http.MethodPatch, intakeItemPath(pending), fmt.Sprintf(`{"description_html":%q}`, issueImageHTML(image)),
			http.StatusOK), &changed)
		bound, err := assetpg.New(w.pool).FindAsset(t.Context(), image)
		if err != nil || bound.BoundAt == nil || !bound.BoundAt.Equal(changed.UpdatedAt) || bound.UpdatedByID == nil || *bound.UpdatedByID != w.ids["walt"] {
			t.Fatalf("the image is not bound by walt's change at its moment: %+v %v", bound, err)
		}
		attachment = attach(t, "walt", pending)
		alices := attach(t, "alice", pending)
		assetAPICall(t, w, "walt", http.MethodDelete, "/api/v0/assets/"+alices.String(), "", http.StatusNoContent)
		attachedTo(t, w, "walt", pending, attachment)
	})
	t.Run("a member who did not submit it", func(t *testing.T) {
		assetRefused(t, w, "bob", http.MethodPost, "/api/v0/assets", prepare(pending), http.StatusForbidden, "forbidden")
		assetRefused(t, w, "bob", http.MethodDelete, "/api/v0/assets/"+attachment.String(), "", http.StatusForbidden, "forbidden")
		attachedTo(t, w, "bob", pending, attachment)
		if _, data := readAssetHTTP(t, w, "bob", attachment); !bytes.Equal(data, notes) {
			t.Errorf("bob reads walt's attachment as %q", data)
		}
		readAssetHTTP(t, w, "bob", image)
		assetAPICall(t, w, "bob", http.MethodGet, "/api/v0/assets/"+image.String(), "", http.StatusOK)
	})
	t.Run("declined, closed to every write", func(t *testing.T) {
		waiting := prepareAssetHTTP(t, w, "walt", "ISSUE_ATTACHMENT", attachmentScope("issue", declined), "notes.txt", "text/plain", len(notes))
		putAssetHTTP(t, w, waiting, notes, http.StatusNoContent)
		spare := prepareAssetHTTP(t, w, "walt", "ISSUE_DESCRIPTION", attachmentScope("issue", declined), "spare.png", "image/png", 8).Asset.ID
		kept := attach(t, "walt", declined)
		gone := prepareAssetHTTP(t, w, "walt", "ISSUE_ATTACHMENT", attachmentScope("issue", declined), "gone.txt", "text/plain", len(notes)).Asset.ID
		assetAPICall(t, w, "walt", http.MethodDelete, "/api/v0/assets/"+gone.String(), "", http.StatusNoContent)
		w.triage(t, "alice", declined, `{"status":"declined"}`, http.StatusOK)
		for _, name := range []string{"walt", "alice"} {
			assetRefused(t, w, name, http.MethodPost, "/api/v0/assets", prepare(declined), http.StatusConflict, "issue.intake_closed")
		}
		assetRefused(t, w, "walt", http.MethodPost, completeAssetPath(waiting.Asset.ID), `{}`, http.StatusConflict, "issue.intake_closed")
		assetRefused(t, w, "walt", http.MethodDelete, "/api/v0/assets/"+spare.String(), "", http.StatusConflict, "issue.intake_closed")
		assetRefused(t, w, "alice", http.MethodDelete, "/api/v0/assets/"+kept.String(), "", http.StatusConflict, "issue.intake_closed")
		// A repeated deletion and a repeated confirmation are writes too.
		assetRefused(t, w, "walt", http.MethodDelete, "/api/v0/assets/"+gone.String(), "", http.StatusConflict, "issue.intake_closed")
		assetRefused(t, w, "walt", http.MethodPost, completeAssetPath(kept), `{}`, http.StatusConflict, "issue.intake_closed")
		assetRefused(t, w, "walt", http.MethodPost, duplicateAssetPath(attachment), `{"purpose":"ISSUE_ATTACHMENT","scope":`+attachmentScope("issue", declined)+`}`,
			http.StatusConflict, "issue.intake_closed")
		attachedTo(t, w, "bob", declined, kept)
		readAssetHTTP(t, w, "walt", kept)
	})
	t.Run("accepted, an ordinary work item", func(t *testing.T) {
		w.triage(t, "alice", accepted, `{"status":"accepted"}`, http.StatusOK)
		assetRefused(t, w, "walt", http.MethodPost, "/api/v0/assets", prepare(accepted), http.StatusForbidden, "forbidden")
		attach(t, "bob", accepted)
	})
	t.Run("a snooze whose time has come", func(t *testing.T) {
		till := time.Now().Add(-time.Hour).UTC().Truncate(time.Microsecond)
		route := newIssueRouteWithClock(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), clockAt(till.Add(-time.Hour)))
		var s submission
		decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, intakePath(w.web), w.ids["walt"], `{"name":"Snoozed"}`, http.StatusCreated), &s)
		route.answerWithin(t, w.contract, http.MethodPost, triagePath(s.ID), w.ids["alice"], snoozedTill(till), http.StatusOK)
		intakeIs(t, "walt's read of Snoozed", w.send(t, "walt", http.MethodGet, "/api/v0/issues/"+s.ID.String(), "", http.StatusOK),
			&answeredIntake{Status: "pending"})
		var status string
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT status FROM intake_issues WHERE issue_id = $1", s.ID).Scan(&status); err != nil || status != "snoozed" {
			t.Fatalf("Snoozed's row is stored %s, %v; want snoozed", status, err)
		}
		attach(t, "walt", s.ID)
	})
	t.Run("its intake turned off", func(t *testing.T) {
		w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"intake_view":false}`, http.StatusOK)
		for _, name := range []string{"walt", "alice"} {
			assetRefused(t, w, name, http.MethodPost, "/api/v0/assets", prepare(pending), http.StatusConflict, "issue.intake_disabled")
		}
		assetRefused(t, w, "walt", http.MethodDelete, "/api/v0/assets/"+attachment.String(), "", http.StatusConflict, "issue.intake_disabled")
		assetRefused(t, w, "walt", http.MethodPost, "/api/v0/assets", prepare(declined), http.StatusConflict, "issue.intake_disabled")
		assetRefused(t, w, "bob", http.MethodPost, "/api/v0/assets", prepare(pending), http.StatusForbidden, "forbidden")
		attach(t, "bob", accepted)
		for _, name := range []string{"walt", "bob"} {
			attachedTo(t, w, name, pending, attachment)
			readAssetHTTP(t, w, name, attachment)
			readAssetHTTP(t, w, name, image)
		}
		commentAsset(t, w, "bob", "comment", w.newComment(t, "bob", declined).ID, false)
		commentAsset(t, w, "bob", "new_comment", pending, true)
	})
}
