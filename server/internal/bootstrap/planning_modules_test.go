package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// moduleAnswer is a module as the API answers it.
type moduleAnswer struct {
	ID          uuid.UUID   `json:"id"`
	WorkspaceID uuid.UUID   `json:"workspace_id"`
	ProjectID   uuid.UUID   `json:"project_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	StartDate   *string     `json:"start_date"`
	TargetDate  *string     `json:"target_date"`
	Status      string      `json:"status"`
	LeadID      *uuid.UUID  `json:"lead_id"`
	MemberIDs   []uuid.UUID `json:"member_ids"`
	SortOrder   float64     `json:"sort_order"`
	ArchivedAt  *time.Time  `json:"archived_at"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	CreatedByID *uuid.UUID  `json:"created_by_id"`
	UpdatedByID *uuid.UUID  `json:"updated_by_id"`
}

// turnModulesOn has alice turn project's modules on through the API (M6 P2
// spec R7: they are off by default).
func (w issueWorld) turnModulesOn(t *testing.T, project uuid.UUID) {
	t.Helper()
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+project.String(), `{"module_view":true}`, http.StatusOK)
}

// createModule has the account name create a module in Web with body
// through the API, checks the answer is want, and returns the module.
func (w issueWorld) createModule(t *testing.T, name, body string, want int) moduleAnswer {
	t.Helper()
	var m moduleAnswer
	decodeAnswer(t, w.send(t, name, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/modules", body, want), &m)
	return m
}

// refusedModule has the account name create a module in project with body
// and checks it is refused with status and the refusal (refusalOf: the
// code, then each field's).
func (w issueWorld) refusedModule(t *testing.T, name string, project uuid.UUID, body string, status int, refusal string) {
	t.Helper()
	answer := w.send(t, name, http.MethodPost, "/api/v0/projects/"+project.String()+"/modules", body, status)
	if got := refusalOf(t, []byte(answer), false); got != refusal {
		t.Errorf("POST modules as %s with %s = %s, want %s", name, body, answer, refusal)
	}
}

// webModules counts Web's rows of modules, the deleted ones too.
func (w issueWorld) webModules(t *testing.T) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM modules WHERE project_id = $1", w.web).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// The app creates a module as the contract says: bob, Web's member, creates
// one named alone, its name trimmed, planned, without dates, lead or
// member, his both writers, both stamps one moment, first at 65535; alice
// one with every value, its members answered in account order, each a row
// of its own at the module's moment and by her, and one with a start alone,
// each first (10000 below the lowest). A name another module of Web has, in
// any case, archived too, is refused; a deleted one's is free, and no
// longer counts for the order, which an archived one does.
func TestCreateModuleThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.turnModulesOn(t, w.web)
	alice, bob := w.ids["alice"], w.ids["bob"]
	before := time.Now()
	auth := w.createModule(t, "bob", `{"name":"  Auth  "}`, http.StatusCreated)
	after := time.Now()
	if auth.Name != "Auth" || auth.Description != "" || auth.StartDate != nil || auth.TargetDate != nil || auth.Status != "planned" ||
		auth.LeadID != nil || auth.MemberIDs == nil || len(auth.MemberIDs) != 0 || auth.WorkspaceID != w.acme || auth.ProjectID != w.web ||
		auth.SortOrder != 65535 || auth.ArchivedAt != nil || auth.CreatedByID == nil || *auth.CreatedByID != bob || auth.UpdatedByID == nil ||
		*auth.UpdatedByID != bob || !auth.CreatedAt.Equal(auth.UpdatedAt) || auth.CreatedAt.Before(before.Add(-time.Second)) ||
		auth.CreatedAt.After(after.Add(time.Second)) {
		t.Fatalf("Auth = %+v, want bob's at %s–%s", auth, before, after)
	}
	// The members given in descending order, answered ascending.
	ascending := []uuid.UUID{alice, bob}
	slices.SortFunc(ascending, uuid.UUID.Compare)
	billing := w.createModule(t, "alice", fmt.Sprintf(`{"name":"Billing","description":"Pay","start_date":"2026-11-02","target_date":"2026-11-13",`+
		`"status":"in-progress","lead_id":%q,"member_ids":[%q,%q]}`, bob, ascending[1], ascending[0]), http.StatusCreated)
	if billing.Name != "Billing" || billing.Description != "Pay" || billing.StartDate == nil || *billing.StartDate != "2026-11-02" ||
		billing.TargetDate == nil || *billing.TargetDate != "2026-11-13" || billing.Status != "in-progress" || billing.LeadID == nil ||
		*billing.LeadID != bob || !slices.Equal(billing.MemberIDs, ascending) || billing.SortOrder != 55535 ||
		billing.CreatedByID == nil || *billing.CreatedByID != alice || !billing.CreatedAt.Equal(billing.UpdatedAt) {
		t.Fatalf("Billing = %+v", billing)
	}
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT member_id FROM module_members WHERE module_id = $1 AND workspace_id = $2 AND project_id = $3
		AND created_at = $4 AND updated_at = $4 AND created_by_id = $5 AND updated_by_id = $5 AND deleted_at IS NULL ORDER BY id`,
		billing.ID, w.acme, w.web, billing.CreatedAt, alice)
	if err != nil {
		t.Fatal(err)
	}
	var stamped []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		stamped = append(stamped, id)
	}
	if rows.Err() != nil || !slices.Equal(stamped, ascending) {
		t.Fatalf("Billing's member rows at its moment, by alice, in id order: %v, %v; want %v", stamped, rows.Err(), ascending)
	}
	kickoff := w.createModule(t, "alice", `{"name":"Kickoff","start_date":"2026-11-02"}`, http.StatusCreated)
	if kickoff.StartDate == nil || *kickoff.StartDate != "2026-11-02" || kickoff.TargetDate != nil || kickoff.SortOrder != 45535 {
		t.Fatalf("Kickoff = %+v, want its start alone at 45535", kickoff)
	}
	for _, name := range []string{"auth", " BILLING ", "KickOff"} {
		w.refusedModule(t, "alice", w.web, `{"name":"`+name+`"}`, http.StatusConflict, "planning.module_name_taken")
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE modules SET archived_at = now() WHERE id = $1", kickoff.ID); err != nil {
		t.Fatal(err)
	}
	w.refusedModule(t, "bob", w.web, `{"name":"kickoff"}`, http.StatusConflict, "planning.module_name_taken")
	if docs := w.createModule(t, "bob", `{"name":"Docs"}`, http.StatusCreated); docs.SortOrder != 35535 {
		t.Errorf("Docs' sort order = %v, want 35535: the archived Kickoff counts", docs.SortOrder)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE modules SET deleted_at = now() WHERE name IN ('Kickoff', 'Docs')"); err != nil {
		t.Fatal(err)
	}
	if again := w.createModule(t, "bob", `{"name":"kickoff"}`, http.StatusCreated); again.SortOrder != 45535 {
		t.Errorf("the new kickoff's sort order = %v, want 45535: the deleted no longer count", again.SortOrder)
	}
	if n := w.webModules(t); n != 5 {
		t.Fatalf("modules of Web = %d; want 5", n)
	}
}

// The refusals in their order (M6 P2 spec R2): values first (422), even for
// a caller who does not see the project or a project that is not there;
// then the project (404); then the role, a guest's and an admin of the
// workspace's who sees only his own work items in it (403), who learns
// nothing of the lead he names; then an archived project (409), before
// modules turned off (409).
func TestCreateModuleRefusals(t *testing.T) {
	w := newIssueWorld(t)
	nowhere, olga := uuid.NewV7(), w.ids["olga"].String()
	w.refusedModule(t, "olga", w.web, `{"name":"  "}`, http.StatusUnprocessableEntity, "validation_failed name too_short")
	w.refusedModule(t, "alice", nowhere, `{"name":"A","start_date":"2026-11-20","target_date":"2026-11-10"}`, http.StatusUnprocessableEntity,
		"validation_failed target_date not_allowed")
	w.refusedModule(t, "alice", w.web, `{"name":"A","member_ids":["`+olga+`","`+olga+`"]}`, http.StatusUnprocessableEntity,
		"validation_failed member_ids[1] duplicate")
	w.refusedModule(t, "olga", w.web, `{"name":"A"}`, http.StatusNotFound, "project.not_found")
	w.refusedModule(t, "alice", nowhere, `{"name":"A"}`, http.StatusNotFound, "project.not_found")
	// Modules still off: each refusal before it holds.
	w.refusedModule(t, "walt", w.web, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	w.refusedModule(t, "alice", w.web, `{"name":"A"}`, http.StatusConflict, "planning.modules_disabled")
	w.adminGuest(t)
	w.refusedModule(t, "walt", w.web, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	w.refusedModule(t, "alice", w.web, `{"name":"A"}`, http.StatusConflict, "project.archived")
	w.refusedModule(t, "walt", w.web, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/unarchive", "", http.StatusOK)
	w.turnModulesOn(t, w.web)
	w.refusedModule(t, "walt", w.web, `{"name":"A","lead_id":"`+olga+`"}`, http.StatusForbidden, "forbidden")
	w.createModule(t, "bob", `{"name":"A"}`, http.StatusCreated)
	if n := w.webModules(t); n != 1 {
		t.Fatalf("modules of Web = %d; want bob's one", n)
	}
}

// The lead and the members are held to Web's roles under the lock (M6 P2
// spec R5): walt, its guest, olga, never in it, and wanda, who left it, are
// refused at their places, together or alone; alice, its admin, and bob,
// its member, may. Nothing refused is written.
func TestCreateModuleLeadAndMembers(t *testing.T) {
	w := newIssueWorld(t)
	w.turnModulesOn(t, w.web)
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	membership := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id = $1 AND member_id = $2 AND is_active AND deleted_at IS NULL",
		w.web, w.ids["wanda"])[0]
	w.send(t, "alice", http.MethodDelete, "/api/v0/project-members/"+membership.String(), "", http.StatusNoContent)
	id := func(name string) string { return w.ids[name].String() }
	for _, tt := range []struct{ body, refusal string }{
		{fmt.Sprintf(`{"name":"A","lead_id":%q,"member_ids":[%q,%q,%q,%q]}`, id("walt"), id("alice"), id("olga"), id("wanda"), id("bob")),
			"validation_failed lead_id not_allowed; member_ids[1] not_allowed; member_ids[2] not_allowed"},
		{fmt.Sprintf(`{"name":"A","lead_id":%q}`, id("olga")), "validation_failed lead_id not_allowed"},
		{fmt.Sprintf(`{"name":"A","lead_id":%q,"member_ids":[%q,%q]}`, id("bob"), id("bob"), id("wanda")), "validation_failed member_ids[1] not_allowed"},
		{fmt.Sprintf(`{"name":"A","member_ids":[%q]}`, id("walt")), "validation_failed member_ids[0] not_allowed"},
	} {
		w.refusedModule(t, "alice", w.web, tt.body, http.StatusUnprocessableEntity, tt.refusal)
	}
	if n := w.webModules(t); n != 0 {
		t.Fatalf("modules of Web = %d; want none", n)
	}
	m := w.createModule(t, "bob", fmt.Sprintf(`{"name":"A","lead_id":%q,"member_ids":[%q,%q]}`, id("alice"), id("bob"), id("alice")),
		http.StatusCreated)
	members := []uuid.UUID{w.ids["alice"], w.ids["bob"]}
	slices.SortFunc(members, uuid.UUID.Compare)
	if m.LeadID == nil || *m.LeadID != w.ids["alice"] || !slices.Equal(m.MemberIDs, members) {
		t.Fatalf("A = %+v", m)
	}
}

// createsModule is name's creation of a module in Web with body.
func createsModule(name, body string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newPlanningRoute(t, w.pool, tx), name, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/modules", body)
	}
}

// Two creations of modules of one name, in two cases, the second waiting on
// Web's row while the first holds it: the first is created, the second's
// insert breaks the name's unique index once the first has committed and is
// refused, and neither deadlocks (no 40P01, which would answer 500).
func TestCreateModuleInterleaving(t *testing.T) {
	inBothOrdersAt(t, "projects", func(t *testing.T, w issueWorld) { w.turnModulesOn(t, w.web) }, "alice", createsModule("alice", `{"name":"Auth"}`),
		"bob", createsModule("bob", `{"name":"AUTH"}`), func(t *testing.T, w issueWorld, aliceFirst bool, alice, bob exchange) {
			first, second := alice, bob
			if !aliceFirst {
				first, second = bob, alice
			}
			w.answered(t, first, http.StatusCreated)
			w.refused(t, second, http.StatusConflict, "planning.module_name_taken")
			if n := w.webModules(t); n != 1 {
				t.Fatalf("modules of Web = %d; want the first's one", n)
			}
		})
}
