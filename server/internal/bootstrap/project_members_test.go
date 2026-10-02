package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// membership is an account's undeleted membership of a project as stored.
type membership struct {
	id          uuid.UUID
	role        shared.Role
	active      bool
	by          uuid.UUID // updated_by_id
	at, created time.Time // updated_at, created_at
}

func membershipOf(t *testing.T, pool *pgxpool.Pool, project, user uuid.UUID) membership {
	t.Helper()
	var m membership
	if err := pool.QueryRow(context.Background(), `SELECT id, role, is_active, updated_by_id, updated_at, created_at FROM project_members
		WHERE project_id = $1 AND member_id = $2 AND deleted_at IS NULL`, project, user).Scan(&m.id, &m.role, &m.active, &m.by, &m.at,
		&m.created); err != nil {
		t.Fatal(err)
	}
	return m
}

// A restored membership gives no more than it had, nor more than a new one
// would, when its member joins; when an admin adds him, it takes the role
// the admin asks for (M3 design 3.5, 3.6 convention 6, 9.1's table), on
// the wired app. bob, acme's member, joins alice's public project Web: a
// new membership as a member, by him at the time of his request, his
// display settings at 65535. Joining again, an active member, he is left
// as he is: nothing is written. Then, each time, his membership is ended
// with a role, as P5's removal will end it (SQL stands in), and he joins
// again, or alice adds him: the same row is active again with the role of
// 9.1's row, by the caller at the time of that request, still made when it
// was. His workspace role is changed through the API before the last row.
func TestARestoredMembershipGivesNoMoreThanItHad(t *testing.T) {
	contract, base, pool, alice, aliceID, web, _ := twoProjects(t)
	bob := registerAccount(t, contract, base, "bob@example.com").AccessToken
	bobID := accountID(t, contract, base, bob)
	acme := inWorkspaceOf(t, pool, web, bobID, aliceID, shared.RoleMember)
	join := func(want shared.Role) {
		t.Helper()
		status, body := call(t, contract, http.MethodPost, base+"/api/v0/projects/"+web.String()+"/join", bob, "")
		var p struct {
			MemberRole *shared.Role `json:"member_role"`
			SortOrder  float64      `json:"sort_order"`
		}
		decodeAnswer(t, body, &p)
		if status != http.StatusOK || p.MemberRole == nil || *p.MemberRole != want || p.SortOrder != 65535 {
			t.Fatalf("bob's joining = %d %s; want 200, his role %d, at 65535", status, body, want)
		}
	}

	before := time.Now().Truncate(time.Microsecond)
	join(shared.RoleMember)
	first := membershipOf(t, pool, web, bobID)
	var prefs struct {
		sortOrder float64
		by        uuid.UUID
		at        time.Time
	}
	if err := pool.QueryRow(context.Background(), `SELECT sort_order, created_by_id, created_at FROM project_user_properties
		WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL`, web, bobID).Scan(&prefs.sortOrder, &prefs.by, &prefs.at); err != nil {
		t.Fatal(err)
	}
	if first.role != shared.RoleMember || !first.active || first.by != bobID || first.at != first.created || first.at.Before(before) ||
		first.at.After(time.Now()) || prefs.sortOrder != 65535 || prefs.by != bobID || prefs.at != first.at {
		t.Fatalf("bob's new membership %+v, display settings %+v; want him an active member and at 65535, both by him within the request",
			first, prefs)
	}
	// stored is bob's membership of Web and his display settings there, every
	// column of each row: a second row of either fails the statement.
	stored := func() string {
		t.Helper()
		var rows string
		if err := pool.QueryRow(context.Background(), `SELECT (SELECT m::text FROM project_members m WHERE m.project_id = $1 AND m.member_id = $2)
			|| ' | ' || (SELECT u::text FROM project_user_properties u WHERE u.project_id = $1 AND u.user_id = $2)`, web, bobID).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		return rows
	}
	was := stored()
	join(shared.RoleMember)
	if got := stored(); got != was {
		t.Fatalf("bob's rows after he joins again, an active member:\n%s\nwant them as they were:\n%s", got, was)
	}

	for _, tt := range []struct {
		name      string
		was       shared.Role // the ended membership's role
		workspace shared.Role // his workspace role
		add       bool        // alice adds him as a guest, rather than he joins
		want      shared.Role
	}{
		{"a guest before joins", shared.RoleGuest, shared.RoleMember, false, shared.RoleGuest},
		{"an admin before joins", shared.RoleAdmin, shared.RoleMember, false, shared.RoleMember},
		{"an admin before is added as a guest", shared.RoleAdmin, shared.RoleMember, true, shared.RoleGuest},
		{"a member before, now a workspace admin, joins", shared.RoleMember, shared.RoleAdmin, false, shared.RoleMember},
	} {
		if tag, err := pool.Exec(context.Background(), "UPDATE project_members SET role = $3, is_active = false WHERE project_id = $1 AND member_id = $2",
			web, bobID, tt.was); err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("%s: ending bob's membership = %v, %v", tt.name, tag, err)
		}
		if tt.workspace != shared.RoleMember {
			var id uuid.UUID
			if err := pool.QueryRow(context.Background(), "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", acme,
				bobID).Scan(&id); err != nil {
				t.Fatal(err)
			}
			if status, body := call(t, contract, http.MethodPatch, base+"/api/v0/workspace-members/"+id.String(), alice,
				fmt.Sprintf(`{"role":%d}`, tt.workspace)); status != http.StatusOK {
				t.Fatalf("%s: bob's workspace role = %d %s", tt.name, status, body)
			}
		}
		before := time.Now().Truncate(time.Microsecond)
		by := bobID
		if tt.add {
			by = aliceID
			if status, body := call(t, contract, http.MethodPost, base+"/api/v0/projects/"+web.String()+"/members", alice,
				`{"members":[{"member_id":"`+bobID.String()+`","role":5}]}`); status != http.StatusCreated {
				t.Fatalf("%s: alice's adding = %d %s", tt.name, status, body)
			}
		} else {
			join(tt.want)
		}
		got := membershipOf(t, pool, web, bobID)
		if got.id != first.id || got.role != tt.want || !got.active || got.by != by || got.at.Before(before) || got.at.After(time.Now()) ||
			got.created != first.created {
			t.Errorf("%s: bob's membership %+v; want %s restored as %d, by %s within the request", tt.name, got, first.id, tt.want, by)
		}
	}
}
