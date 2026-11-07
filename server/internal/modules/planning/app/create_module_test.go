package app

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The lock world's module store, for the creation.

func (w *lockWorld) LowestModuleSortOrder(ctx context.Context, project uuid.UUID) (*float64, error) {
	w.log(ctx, "LowestModuleSortOrder")
	var lowest *float64
	for _, m := range w.modules {
		if m.ProjectID == project && (lowest == nil || m.SortOrder < *lowest) {
			v := m.SortOrder
			lowest = &v
		}
	}
	return lowest, nil
}

func (w *lockWorld) CreateModule(ctx context.Context, m domain.Module) error {
	w.log(ctx, "CreateModule")
	if w.createRefusal != nil {
		return w.createRefusal
	}
	w.createdModules = append(w.createdModules, m)
	return nil
}

func (w *lockWorld) InsertModuleMembers(ctx context.Context, r MemberRows) error {
	w.log(ctx, "InsertModuleMembers")
	w.memberRows = append(w.memberRows, r)
	return nil
}

func (w *lockWorld) createModule(in domain.NewModule) (domain.Module, error) {
	uc := NewCreateModule(CreateModuleDeps{Locks: w.locks(), Roles: w, Modules: w, Clock: w})
	return uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockCreator}), lockProject, in)
}

// A module named alone is created first among the project's modules, its name without surrounding white space, planned,
// without dates, lead or member, created and last changed by the caller at the moment read after the project's lock;
// with no module yet, at 65535. Nobody is named, so no role is read and no member row written.
func TestCreateModuleAlone(t *testing.T) {
	w := newModuleWorld()
	w.modules = map[uuid.UUID]domain.Module{}
	w.now = time.Date(2026, 11, 7, 1, 0, 0, 0, time.UTC)
	m, err := w.createModule(domain.NewModule{Name: "  Auth ", Description: "Sign in"})
	if err != nil {
		t.Fatal(err)
	}
	if m.ID == (uuid.UUID{}) || m.ProjectID != lockProject || m.WorkspaceID != lockWorkspace || m.Name != "Auth" || m.Description != "Sign in" ||
		m.Status != domain.ModulePlanned || m.StartDate != nil || m.TargetDate != nil || m.LeadID != nil || len(m.MemberIDs) != 0 ||
		m.SortOrder != 65535 || m.ArchivedAt != nil {
		t.Fatalf("created %+v", m)
	}
	for _, id := range []*uuid.UUID{m.CreatedByID, m.UpdatedByID} {
		if id == nil || *id != lockCreator {
			t.Fatalf("creator/writer %v; want the caller", id)
		}
	}
	if !m.CreatedAt.Equal(w.now) || !m.UpdatedAt.Equal(w.now) || len(w.createdModules) != 1 || w.createdModules[0].ID != m.ID {
		t.Fatalf("stored %+v at %s/%s", w.createdModules, m.CreatedAt, m.UpdatedAt)
	}
	want := []string{"Facts outside tx", "Authorize module.write outside tx", "Enter 1 outside tx", "Begin", "Lock", "Authorize module.write",
		"LowestModuleSortOrder", "Now", "CreateModule", "End", "Leave"}
	if !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
}

// A module with every value: its lead and its members held to the project's roles read after the lock and the decision
// under it, all of them in one read; written before the project's lowest module, its members in account order, each
// member's row of its own, at the module's moment and by its writer.
func TestCreateModuleWithLeadAndMembers(t *testing.T) {
	w := newModuleWorld()
	w.modules[lockModule] = domain.Module{ID: lockModule, ProjectID: lockProject, SortOrder: 100}
	w.now = time.Date(2026, 11, 7, 1, 0, 0, 0, time.UTC)
	lead, a, b := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	w.roles = map[uuid.UUID]shared.Role{lead: shared.RoleAdmin, a: shared.RoleMember, b: shared.RoleAdmin}
	status := domain.ModuleInProgress
	in := domain.NewModule{Name: "Billing", StartDate: date(t, "2026-11-02"), TargetDate: date(t, "2026-11-13"), Status: &status, LeadID: &lead,
		MemberIDs: []uuid.UUID{b, lead, a}}
	m, err := w.createModule(in)
	if err != nil {
		t.Fatal(err)
	}
	if m.SortOrder != -9900 || m.Status != domain.ModuleInProgress || m.StartDate.String() != "2026-11-02" || m.TargetDate.String() != "2026-11-13" ||
		m.LeadID == nil || *m.LeadID != lead || !slices.Equal(m.MemberIDs, []uuid.UUID{lead, a, b}) {
		t.Fatalf("created %+v; want before 100, its members in account order", m)
	}
	if !slices.Equal(in.MemberIDs, []uuid.UUID{b, lead, a}) {
		t.Fatalf("the request's members were reordered: %v", in.MemberIDs)
	}
	if len(w.asked) != 1 || !slices.Equal(w.asked[0], []uuid.UUID{lead, b, lead, a}) {
		t.Fatalf("roles asked of %v; want the lead and the members, once", w.asked)
	}
	if len(w.memberRows) != 1 {
		t.Fatalf("member rows %+v", w.memberRows)
	}
	r := w.memberRows[0]
	if r.WorkspaceID != lockWorkspace || r.ProjectID != lockProject || r.ModuleID != m.ID || !slices.Equal(r.Accounts, m.MemberIDs) ||
		len(r.IDs) != 3 || r.By != lockCreator || !r.At.Equal(m.CreatedAt) {
		t.Fatalf("member rows %+v; want the module's members at its moment, by its writer", r)
	}
	if slices.Contains(r.IDs, uuid.UUID{}) || r.IDs[0] == r.IDs[1] || r.IDs[1] == r.IDs[2] || r.IDs[0] == r.IDs[2] {
		t.Fatalf("member row ids %v; want one of each", r.IDs)
	}
	want := []string{"Facts outside tx", "Authorize module.write outside tx", "Enter 1 outside tx", "Begin", "Lock", "Authorize module.write",
		"Roles", "LowestModuleSortOrder", "Now", "CreateModule", "InsertModuleMembers", "End", "Leave"}
	if !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
}

// A lead or members the project's roles under the lock refuse — a guest, no active member (one never in it, or one who
// left, whom Roles does not answer) — are refused together, each at its place, the lead alone too, a member alone too;
// nothing is written.
func TestCreateModuleRefusesWhoMayNotJoin(t *testing.T) {
	guest, gone, member := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for _, tt := range []struct {
		name string
		in   domain.NewModule
		want []shared.FieldError
	}{
		{"all", domain.NewModule{Name: "A", LeadID: &guest, MemberIDs: []uuid.UUID{member, gone, guest}}, []shared.FieldError{
			{Field: "lead_id", Code: shared.FieldNotAllowed}, {Field: "member_ids[1]", Code: shared.FieldNotAllowed},
			{Field: "member_ids[2]", Code: shared.FieldNotAllowed}}},
		{"the lead alone", domain.NewModule{Name: "A", LeadID: &gone, MemberIDs: []uuid.UUID{member}},
			[]shared.FieldError{{Field: "lead_id", Code: shared.FieldNotAllowed}}},
		{"a member alone", domain.NewModule{Name: "A", LeadID: &member, MemberIDs: []uuid.UUID{member, guest}},
			[]shared.FieldError{{Field: "member_ids[1]", Code: shared.FieldNotAllowed}}},
		{"a member without a lead", domain.NewModule{Name: "A", MemberIDs: []uuid.UUID{gone}},
			[]shared.FieldError{{Field: "member_ids[0]", Code: shared.FieldNotAllowed}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newModuleWorld()
			w.roles = map[uuid.UUID]shared.Role{guest: shared.RoleGuest, member: shared.RoleMember}
			_, err := w.createModule(tt.in)
			var e *shared.Error
			if !errors.As(err, &e) || e.Code != shared.CodeValidationFailed || len(e.Fields) != len(tt.want) {
				t.Fatalf("got %v; want validation_failed on %v", err, tt.want)
			}
			for i, f := range e.Fields {
				if f.Field != tt.want[i].Field || f.Code != tt.want[i].Code {
					t.Fatalf("problem %d = %+v; want %+v", i, f, tt.want[i])
				}
			}
			if len(w.createdModules) != 0 || len(w.memberRows) != 0 || slices.Contains(w.calls, "CreateModule") {
				t.Fatalf("written %+v %+v, calls %q", w.createdModules, w.memberRows, w.calls)
			}
			// The decision under the lock, not the precheck's ("… outside tx").
			if roles, decided := slices.Index(w.calls, "Roles"), slices.Index(w.calls, "Authorize module.write"); decided < 0 || roles < decided {
				t.Fatalf("calls %q: want the roles read under the lock, after its decision", w.calls)
			}
		})
	}
}

// The values are checked before anything is read: a bad name, a status not of the six and a member named twice answer
// validation_failed with every problem, and the project is not looked at.
func TestCreateModuleChecksValuesFirst(t *testing.T) {
	w := newModuleWorld()
	delete(w.facts, lockProject) // a read would answer project.not_found
	done, member := domain.ModuleStatus("done"), uuid.NewV7()
	_, err := w.createModule(domain.NewModule{Name: " ", Status: &done, MemberIDs: []uuid.UUID{member, member}})
	var e *shared.Error
	if !errors.As(err, &e) || e.Code != shared.CodeValidationFailed || len(e.Fields) != 3 {
		t.Fatalf("got %v; want validation_failed on name, status and member_ids[1]", err)
	}
	if len(w.calls) != 0 {
		t.Fatalf("calls %q; want none", w.calls)
	}
}

// A name the store refuses (domain.ErrModuleNameTaken, its unique index's) is the creation's answer, and no member row
// is written after it.
func TestCreateModuleNameTaken(t *testing.T) {
	w := newModuleWorld()
	w.createRefusal = domain.ErrModuleNameTaken
	member := uuid.NewV7()
	w.roles = map[uuid.UUID]shared.Role{member: shared.RoleMember}
	if _, err := w.createModule(domain.NewModule{Name: "Auth", MemberIDs: []uuid.UUID{member}}); !errors.Is(err, domain.ErrModuleNameTaken) {
		t.Fatalf("got %v; want planning.module_name_taken", err)
	}
	if len(w.memberRows) != 0 || w.calls[len(w.calls)-1] != "Leave" {
		t.Fatalf("member rows %+v, calls %q", w.memberRows, w.calls)
	}
}
