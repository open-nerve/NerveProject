package bootstrap

import (
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// changeModule has the account name PATCH the module id with body, checks the answer is want, and returns the module
// it answers.
func (w issueWorld) changeModule(t *testing.T, name string, id uuid.UUID, body string, want int) moduleAnswer {
	t.Helper()
	var m moduleAnswer
	decodeAnswer(t, w.send(t, name, http.MethodPatch, "/api/v0/modules/"+id.String(), body, want), &m)
	return m
}

// refusedModuleChange has the account name PATCH the module id with body and checks it is refused with status and the
// refusal (refusalOf).
func (w issueWorld) refusedModuleChange(t *testing.T, name string, id uuid.UUID, body string, status int, refusal string) {
	t.Helper()
	if got := refusalOf(t, []byte(w.send(t, name, http.MethodPatch, "/api/v0/modules/"+id.String(), body, status)), false); got != refusal {
		t.Errorf("PATCH %s as %s with %s = %s, want %s", id, name, body, got, refusal)
	}
}

// memberRow is a row of a module's members: its account, deletion, creation and last change.
type memberRow struct {
	account            uuid.UUID
	deleted            *time.Time
	created, updated   time.Time
	createdBy, changer uuid.UUID
}

// memberRowsOf are the module id's member rows, the deleted ones too, by creation, then id.
func (w issueWorld) memberRowsOf(t *testing.T, id uuid.UUID) []memberRow {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT member_id, deleted_at, created_at, updated_at, created_by_id, updated_by_id
		FROM module_members WHERE module_id = $1 ORDER BY created_at, id`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []memberRow
	for rows.Next() {
		var r memberRow
		if err := rows.Scan(&r.account, &r.deleted, &r.created, &r.updated, &r.createdBy, &r.changer); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// sortedAccounts are ids in id order.
func sortedAccounts(ids ...uuid.UUID) []uuid.UUID {
	return slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare)
}

// The app changes Web's modules as the contract says (M6 P2 spec R4, R5, R8): bob, its member, changes alice's Auth,
// whose lead and member wanda has left Web since: the fields given, the members a whole set — bob's own row
// soft-deleted at the change's moment by him, alice and wanda kept as they are, wanda not looked at — the change his;
// the dates given alone held to the module's; a lead set anew held to Web's roles, and cleared; members added held to
// them, those kept not; a change of nothing writes nothing; a name another module has, in any case, archived too, is
// refused, a deleted one's is free, and so is its own in another case.
func TestUpdateModuleThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.turnModulesOn(t, w.web)
	alice, bob, wanda := w.ids["alice"], w.ids["bob"], w.ids["wanda"]
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	auth := w.createModule(t, "alice", fmt.Sprintf(`{"name":"Auth","start_date":"2026-11-02","target_date":"2026-11-13","lead_id":%q,`+
		`"member_ids":[%q,%q,%q]}`, wanda, alice, wanda, bob), http.StatusCreated)
	membership := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id = $1 AND member_id = $2 AND is_active AND deleted_at IS NULL",
		w.web, wanda)[0]
	w.send(t, "alice", http.MethodDelete, "/api/v0/project-members/"+membership.String(), "", http.StatusNoContent)

	got := w.changeModule(t, "bob", auth.ID, fmt.Sprintf(`{"name":" Auth 2 ","description":"Sign in","status":"in-progress","sort_order":-1,`+
		`"member_ids":[%q,%q]}`, wanda, alice), http.StatusOK)
	if got.Name != "Auth 2" || got.Description != "Sign in" || got.Status != "in-progress" || got.SortOrder != -1 || got.StartDate == nil ||
		*got.StartDate != "2026-11-02" || got.TargetDate == nil || *got.TargetDate != "2026-11-13" || got.LeadID == nil || *got.LeadID != wanda ||
		!slices.Equal(got.MemberIDs, sortedAccounts(alice, wanda)) || *got.CreatedByID != alice || *got.UpdatedByID != bob ||
		!got.UpdatedAt.After(got.CreatedAt) {
		t.Fatalf("bob's change: %+v", got)
	}
	if read := w.getModule(t, "bob", auth.ID); !reflect.DeepEqual(read, got) {
		t.Fatalf("read %+v, answered %+v", read, got)
	}
	created := w.memberRowsOf(t, auth.ID)
	var accounts []uuid.UUID
	for _, r := range created {
		accounts = append(accounts, r.account)
		removed := r.deleted != nil && r.deleted.Equal(got.UpdatedAt) && r.updated.Equal(got.UpdatedAt) && r.changer == bob
		kept := r.deleted == nil && r.updated.Equal(auth.CreatedAt) && r.changer == alice
		if r.account == bob && !removed || r.account != bob && !kept {
			t.Errorf("Auth's row of %s: %+v; want bob's removed at the change, the others as created", r.account, r)
		}
	}
	if !slices.Equal(sortedAccounts(accounts...), sortedAccounts(alice, bob, wanda)) {
		t.Fatalf("Auth's rows %+v; want alice's, bob's and wanda's, bob's soft-deleted", created)
	}

	// R4: the dates given alone, held to the module's; the target cleared, then the start alone is free.
	for _, tt := range []struct{ body, refusal string }{
		{`{"start_date":"2026-11-14"}`, "validation_failed start_date not_allowed"},
		{`{"target_date":"2026-11-01"}`, "validation_failed target_date not_allowed"},
	} {
		w.refusedModuleChange(t, "bob", auth.ID, tt.body, http.StatusUnprocessableEntity, tt.refusal)
	}
	if m := w.changeModule(t, "bob", auth.ID, `{"target_date":null}`, http.StatusOK); m.TargetDate != nil || *m.StartDate != "2026-11-02" {
		t.Fatalf("the target cleared: %+v", m)
	}
	if m := w.changeModule(t, "bob", auth.ID, `{"start_date":"2026-11-20"}`, http.StatusOK); *m.StartDate != "2026-11-20" {
		t.Fatalf("the start alone: %+v", m)
	}

	// R5: a lead set anew — bob may; wanda, who left, walt, Web's guest, and olga, never in it, may not; null clears it.
	if m := w.changeModule(t, "alice", auth.ID, fmt.Sprintf(`{"lead_id":%q}`, bob), http.StatusOK); m.LeadID == nil || *m.LeadID != bob {
		t.Fatalf("bob as the lead: %+v", m)
	}
	for _, name := range []string{"wanda", "walt", "olga"} {
		w.refusedModuleChange(t, "alice", auth.ID, fmt.Sprintf(`{"lead_id":%q}`, w.ids[name]), http.StatusUnprocessableEntity,
			"validation_failed lead_id not_allowed")
	}
	if m := w.changeModule(t, "alice", auth.ID, `{"lead_id":null}`, http.StatusOK); m.LeadID != nil {
		t.Fatalf("the lead cleared: %+v", m)
	}
	// Members added are held to Web's roles, wanda kept is not; then wanda removed and bob back, in a row of his own.
	w.refusedModuleChange(t, "alice", auth.ID, fmt.Sprintf(`{"member_ids":[%q,%q,%q]}`, wanda, alice, w.ids["walt"]), http.StatusUnprocessableEntity,
		"validation_failed member_ids[2] not_allowed")
	w.refusedModuleChange(t, "alice", auth.ID, fmt.Sprintf(`{"member_ids":[%q]}`, w.ids["olga"]), http.StatusUnprocessableEntity,
		"validation_failed member_ids[0] not_allowed")
	back := w.changeModule(t, "alice", auth.ID, fmt.Sprintf(`{"member_ids":[%q,%q]}`, alice, bob), http.StatusOK)
	if !slices.Equal(back.MemberIDs, sortedAccounts(alice, bob)) {
		t.Fatalf("bob back: %+v", back)
	}
	rows := w.memberRowsOf(t, auth.ID)
	if n := len(rows); n != 4 || rows[3].account != bob || rows[3].deleted != nil || !rows[3].created.Equal(back.UpdatedAt) || rows[3].createdBy != alice {
		t.Fatalf("Auth's rows %+v; want bob's new row at alice's change", rows)
	}
	for _, r := range rows[:3] {
		if r.account == wanda && (r.deleted == nil || !r.deleted.Equal(back.UpdatedAt) || r.changer != alice) {
			t.Errorf("wanda's row %+v; want it removed at alice's change", r)
		}
	}

	// A change of nothing writes nothing: none given, the members in another order, the values as they are.
	for _, body := range []string{`{}`, fmt.Sprintf(`{"member_ids":[%q,%q]}`, bob, alice), `{"name":"Auth 2","lead_id":null,"target_date":null}`} {
		if m := w.changeModule(t, "bob", auth.ID, body, http.StatusOK); !m.UpdatedAt.Equal(back.UpdatedAt) || *m.UpdatedByID != alice {
			t.Errorf("a change of nothing (%s) wrote: %+v", body, m)
		}
	}
	if n := len(w.memberRowsOf(t, auth.ID)); n != 4 {
		t.Errorf("Auth's member rows = %d after changes of nothing; want 4", n)
	}

	// R8: another module's name, in any case, an archived one's too, is taken; a deleted one's is free, and its own in
	// another case.
	w.createModule(t, "alice", `{"name":"Billing"}`, http.StatusCreated)
	old := w.createModule(t, "alice", `{"name":"Old"}`, http.StatusCreated)
	gone := w.createModule(t, "alice", `{"name":"Gone"}`, http.StatusCreated)
	for id, set := range map[uuid.UUID]string{old.ID: "archived_at", gone.ID: "deleted_at"} {
		if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE modules SET "+set+" = now() WHERE id = $1", id); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"billing", " BILLING ", "old"} {
		w.refusedModuleChange(t, "bob", auth.ID, `{"name":"`+name+`"}`, http.StatusConflict, "planning.module_name_taken")
	}
	for _, name := range []string{"AUTH 2", "gone"} {
		if m := w.changeModule(t, "bob", auth.ID, `{"name":"`+name+`"}`, http.StatusOK); m.Name != name {
			t.Errorf("renamed %s: %+v", name, m)
		}
	}
}

// The refusals in their order (M6 P2 spec R2): values first (422), even on a module that is not there or by a caller
// who does not see it; then the module (404); then the role (403): a member of the workspace who is none of Web's, its
// guest, and an admin of the workspace who sees only his own work items in it; then an archived project, then modules
// turned off, then an archived module (409), for a change of nothing too and before a lead the lock would refuse; a
// deleted module is not found. Nothing refused is written.
func TestUpdateModuleRefusals(t *testing.T) {
	w := newIssueWorld(t)
	w.turnModulesOn(t, w.web)
	auth := w.createModule(t, "alice", `{"name":"Auth"}`, http.StatusCreated).ID
	nowhere, olga := uuid.NewV7(), w.ids["olga"].String()
	w.refusedModuleChange(t, "olga", auth, `{"name":" "}`, http.StatusUnprocessableEntity, "validation_failed name too_short")
	w.refusedModuleChange(t, "alice", nowhere, `{"member_ids":["`+olga+`","`+olga+`"]}`, http.StatusUnprocessableEntity,
		"validation_failed member_ids[1] duplicate")
	w.refusedModuleChange(t, "olga", auth, `{"name":"A"}`, http.StatusNotFound, "planning.module_not_found")
	w.refusedModuleChange(t, "alice", nowhere, `{"name":"A"}`, http.StatusNotFound, "planning.module_not_found")
	w.refusedModuleChange(t, "walt", auth, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	w.adminGuest(t)
	w.refusedModuleChange(t, "walt", auth, `{"name":"A","lead_id":"`+olga+`"}`, http.StatusForbidden, "forbidden")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	w.refusedModuleChange(t, "bob", auth, `{"name":"A"}`, http.StatusConflict, "project.archived")
	w.refusedModuleChange(t, "walt", auth, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/unarchive", "", http.StatusOK)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"module_view":false}`, http.StatusOK)
	w.refusedModuleChange(t, "bob", auth, `{"name":"A"}`, http.StatusConflict, "planning.modules_disabled")
	w.turnModulesOn(t, w.web)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE modules SET archived_at = now() WHERE id = $1", auth); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{}`, `{"sort_order":1}`, `{"lead_id":"` + olga + `"}`} {
		w.refusedModuleChange(t, "bob", auth, body, http.StatusConflict, "planning.archived")
	}
	if m := w.getModule(t, "bob", auth); m.Name != "Auth" || m.SortOrder != 65535 || m.LeadID != nil || !m.UpdatedAt.Equal(m.CreatedAt) {
		t.Fatalf("a refused change wrote: %+v", m)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE modules SET archived_at = NULL, deleted_at = now() WHERE id = $1", auth); err != nil {
		t.Fatal(err)
	}
	w.refusedModuleChange(t, "alice", auth, `{"name":"A"}`, http.StatusNotFound, "planning.module_not_found")
}
