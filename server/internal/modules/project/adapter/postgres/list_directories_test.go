package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

func TestListDirectories(t *testing.T) {
	s, pool := newStore(t)
	owner := newAccount(t, pool, "owner@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, arch, other := newProject(t, s, acme, "Web", "WEB", owner), newProject(t, s, acme, "Archived", "ARC", owner), newProject(t, s, acme, "Other", "OTH", owner)
	groups := map[string]string{"Backlog": "backlog", "Todo": "unstarted", "In Progress": "started", "Done": "completed", "Cancelled": "cancelled", "Triage": "triage"}
	wantStates := make([]app.IssueState, 0)
	for _, pid := range []uuid.UUID{web, arch, other} {
		for name, id := range seedDefaultStates(t, s, acme, pid, owner) {
			if pid != other && name != "Triage" {
				wantStates = append(wantStates, app.IssueState{ID: id, Name: name, Group: groups[name], Default: name == "Backlog"})
			}
		}
	}
	slices.SortFunc(wantStates, func(a, b app.IssueState) int { return a.ID.Compare(b.ID) })
	// No issue uses these directories. An archived project still needs its
	// complete catalogs when an explicit project list grants access to it.
	bug := seedLabel(t, s, acme, web, nil, "Unused Bug", owner)
	old := seedLabel(t, s, acme, web, nil, "Deleted", owner)
	feature := seedLabel(t, s, acme, arch, nil, "Unused Feature", owner)
	seedLabel(t, s, acme, other, nil, "Out of scope", owner)
	exec(t, pool, "UPDATE labels SET deleted_at=$2 WHERE id=$1", old, now)
	exec(t, pool, `INSERT INTO states (id,workspace_id,project_id,name,color,"group",deleted_at) VALUES ($1,$2,$3,'Deleted','#000','started',$4)`, uuid.NewV7(), acme, web, now)
	exec(t, pool, "UPDATE projects SET archived_at=$2 WHERE id=$1", arch, now)
	pids := []uuid.UUID{arch, web, web}
	got, err := s.IssueListStates(context.Background(), pids)
	if err != nil || !slices.Equal(got, wantStates) {
		t.Errorf("complete list states: got %+v, %v; want %+v", got, err, wantStates)
	}
	labels, err := s.IssueListLabels(context.Background(), pids)
	wantLabels := []app.IssueLabel{{ID: bug, Name: "Unused Bug"}, {ID: feature, Name: "Unused Feature"}}
	if err != nil || !slices.Equal(labels, wantLabels) {
		t.Errorf("complete list labels: got %+v, %v; want %+v", labels, err, wantLabels)
	}
	writeStates, err := s.IssueStates(context.Background(), web)
	if err != nil || !slices.ContainsFunc(writeStates, func(s app.IssueState) bool { return s.Group == "triage" }) {
		t.Errorf("write states must keep triage: %+v, %v", writeStates, err)
	}
	for _, empty := range [][]uuid.UUID{nil, {}, {uuid.NewV7()}} {
		states, err := s.IssueListStates(context.Background(), empty)
		if err != nil || states == nil || len(states) != 0 {
			t.Errorf("empty state scope: %+v, %v", states, err)
		}
		labels, err := s.IssueListLabels(context.Background(), empty)
		if err != nil || labels == nil || len(labels) != 0 {
			t.Errorf("empty label scope: %+v, %v", labels, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.IssueListStates(ctx, pids); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("cancelled states: %+v, %v", got, err)
	}
	if got, err := s.IssueListLabels(ctx, pids); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("cancelled labels: %+v, %v", got, err)
	}
}
