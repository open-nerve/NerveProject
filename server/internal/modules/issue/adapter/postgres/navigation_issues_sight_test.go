package postgresadapter_test

import (
	"context"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// What a caller sees now of the work items a recent visit names, NavigationIssues over the store (M7 P3d spec 2.5, R3):
// those undeleted, archived or not, in a project among his grants, which another workspace's project never is, that
// the guest rule does not hide from him by his grant there, each with whether it is in its project's intake. gail is a
// guest of web: without guest_view_all_features her grant is OwnOnly, and she sees her own alone, not alice's nor one
// whose creator is not known; with it she sees every one of web's. A work item in triage is seen, InTriage, pending or
// declined; an accepted one is seen and no longer is; a deleted one, ops' when ops is not among the grants, and far's,
// another workspace's, never are.
func TestNavigationIssuesAnswerWhatTheCallerSees(t *testing.T) {
	s, pool := newStore(t)
	gail, alice := newAccount(t, pool, "gail@corp.com"), newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "sight")
	web, ops, far := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS"), newProject(t, pool, newWorkspace(t, pool, "far"), "FAR")
	hers, alices, unknown, shelved, pending, accepted, declined, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(),
		uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	opsHers, farHers := uuid.NewV7(), uuid.NewV7()
	creators := map[uuid.UUID]uuid.UUID{hers: gail, alices: alice, unknown: alice, shelved: alice, pending: alice, accepted: alice, declined: gail,
		gone: gail}
	for n, i := range []uuid.UUID{hers, alices, unknown, shelved, pending, accepted, declined, gone} {
		newIssue(t, pool, web, i, n+1, creators[i])
	}
	newIssue(t, pool, ops, opsHers, 1, gail)
	newIssue(t, pool, far, farHers, 1, gail)
	exec(t, pool, "UPDATE issues SET created_by_id = NULL WHERE id = $1", unknown)
	exec(t, pool, "UPDATE issues SET archived_at = now() WHERE id = $1", shelved)
	exec(t, pool, "UPDATE issues SET deleted_at = now() WHERE id = $1", gone)
	newIntakeRow(t, pool, web, pending, "pending", nil, nil, alice, false)
	newIntakeRow(t, pool, web, accepted, "accepted", nil, nil, alice, false)
	newIntakeRow(t, pool, web, declined, "declined", nil, nil, gail, false)
	answer := func(id uuid.UUID, p project, sequence int, inTriage bool) app.NavigationIssue {
		return app.NavigationIssue{ID: id, ProjectID: p.id, SequenceID: sequence, Name: "Work", StateID: p.state, Priority: "none",
			AssigneeIDs: []uuid.UUID{}, InTriage: inTriage}
	}
	webs := []app.NavigationIssue{answer(hers, web, 1, false), answer(alices, web, 2, false), answer(unknown, web, 3, false),
		answer(shelved, web, 4, false), answer(pending, web, 5, true), answer(accepted, web, 6, false), answer(declined, web, 7, true)}
	ids := []uuid.UUID{hers, alices, unknown, shelved, pending, accepted, declined, gone, opsHers, farHers, uuid.NewV7()}
	issues := app.NewNavigationIssues(s)
	for _, tt := range []struct {
		name   string
		grants []app.ProjectGrant
		want   []app.NavigationIssue
	}{
		{"gail, web's guest without guest_view_all_features", []app.ProjectGrant{{ProjectID: web.id, OwnOnly: true}}, []app.NavigationIssue{webs[0], webs[6]}},
		{"gail, web's guest with guest_view_all_features", []app.ProjectGrant{{ProjectID: web.id}}, webs},
		{"gail, web's and ops' member", []app.ProjectGrant{{ProjectID: web.id}, {ProjectID: ops.id}}, append(webs[:7:7], answer(opsHers, ops, 1, false))},
		{"gail, reading none", nil, nil},
	} {
		got, err := issues.NavigationIssues(context.Background(), gail, tt.grants, ids)
		want := map[uuid.UUID]app.NavigationIssue{}
		for _, i := range tt.want {
			want[i.ID] = i
		}
		if err != nil || len(got) != len(want) || (len(want) > 0 && !reflect.DeepEqual(got, want)) {
			t.Errorf("%s: NavigationIssues() =\n %+v, %v\nwant\n %+v", tt.name, got, err, want)
		}
	}
}
