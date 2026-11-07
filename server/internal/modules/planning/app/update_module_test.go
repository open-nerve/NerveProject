package app

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// removal is a call of RemoveModuleMembers: the module, the accounts whose rows it soft-deletes, when and by whom.
type removal struct {
	module   uuid.UUID
	accounts []uuid.UUID
	at       time.Time
	by       uuid.UUID
}

// The lock world's module store, for a change.

func (w *lockWorld) UpdateModule(ctx context.Context, m domain.Module) error {
	w.log(ctx, "UpdateModule")
	if w.updateRefusal != nil {
		return w.updateRefusal
	}
	w.updatedModules = append(w.updatedModules, m)
	return nil
}

func (w *lockWorld) RemoveModuleMembers(ctx context.Context, module uuid.UUID, accounts []uuid.UUID, at time.Time, by uuid.UUID) error {
	w.log(ctx, "RemoveModuleMembers")
	w.removals = append(w.removals, removal{module, accounts, at, by})
	return nil
}

// The people of a change: the module's lead and members — kept, an active member; left, who has left the project,
// whom Roles does not answer — and the project's admin, member and guest, who are none of them yet.
var (
	changeKept, changeLeft              = uuid.NewV7(), uuid.NewV7()
	changeAdmin, changeNew, changeGuest = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
)

// changeWorld is a world whose module is Auth, from the 2nd to the 13th of November 2026, planned, led by left, with
// kept and left as its members, at sort order 5, made by its creator an hour before; now is the 7th, noon.
func changeWorld(t *testing.T) *lockWorld {
	w := newModuleWorld()
	made := time.Date(2026, 11, 7, 11, 0, 0, 0, time.UTC)
	w.modules[lockModule] = domain.Module{ID: lockModule, ProjectID: lockProject, WorkspaceID: lockWorkspace, Name: "Auth", Description: "Sign in",
		StartDate: date(t, "2026-11-02"), TargetDate: date(t, "2026-11-13"), Status: domain.ModulePlanned, LeadID: &changeLeft,
		MemberIDs: slices.SortedFunc(slices.Values([]uuid.UUID{changeKept, changeLeft}), uuid.UUID.Compare), SortOrder: 5,
		CreatedAt: made, UpdatedAt: made, CreatedByID: &lockModuleCreator, UpdatedByID: &lockModuleCreator}
	w.roles = map[uuid.UUID]shared.Role{changeKept: shared.RoleMember, changeAdmin: shared.RoleAdmin, changeNew: shared.RoleMember,
		changeGuest: shared.RoleGuest}
	w.now = time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	return w
}

func (w *lockWorld) changeModule(p domain.ModulePatch) (domain.Module, error) {
	uc := NewUpdateModule(UpdateModuleDeps{Locks: w.locks(), Roles: w, Modules: w, Clock: w})
	return uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockModule, p)
}

// A change takes the lock path on the module (Locks, module.write); under the lock, the roles of those it makes anew
// — the lead it sets, the members it adds — in one read; then the clock, then the writes: the module with the fields
// it gives, last changed by the caller at that moment, its creation kept; the rows of the members it removes
// soft-deleted, and rows for those it adds, each an id of its own, at that moment by the caller. A member it keeps,
// who has left the project, is not looked at nor written. The answer is the module as stored, its members in id order.
func TestUpdateModule(t *testing.T) {
	w := changeWorld(t)
	got, err := w.changeModule(domain.ModulePatch{Name: str(" Auth 2 "), Description: str(""), SetStartDate: true, StartDate: date(t, "2026-11-01"),
		SetTargetDate: true, TargetDate: date(t, "2026-11-20"), Status: ptrTo(domain.ModuleInProgress), SetLeadID: true, LeadID: &changeAdmin,
		MemberIDs: &[]uuid.UUID{changeNew, changeLeft, changeAdmin}, SortOrder: ptrTo(-2.5)})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"GetModule outside tx", "Facts outside tx", "Authorize module.write outside tx", "Enter 1 outside tx", "Begin", "Lock",
		"GetModule", "Authorize module.write", "Roles", "Now", "UpdateModule", "RemoveModuleMembers", "InsertModuleMembers", "End", "Leave"}
	if !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	added := slices.SortedFunc(slices.Values([]uuid.UUID{changeNew, changeAdmin}), uuid.UUID.Compare)
	if len(w.asked) != 1 || !slices.Equal(w.asked[0], append([]uuid.UUID{changeAdmin}, added...)) {
		t.Fatalf("roles asked of %v; want the new lead, then the added members, once", w.asked)
	}
	old := changeWorld(t).modules[lockModule]
	stored := old
	stored.Name, stored.Description, stored.StartDate, stored.TargetDate = "Auth 2", "", date(t, "2026-11-01"), date(t, "2026-11-20")
	stored.Status, stored.LeadID, stored.SortOrder = domain.ModuleInProgress, &changeAdmin, -2.5
	stored.MemberIDs = slices.SortedFunc(slices.Values([]uuid.UUID{changeNew, changeLeft, changeAdmin}), uuid.UUID.Compare)
	stored.UpdatedAt, stored.UpdatedByID = w.now, &lockEditor
	if len(w.updatedModules) != 1 || !reflect.DeepEqual(w.updatedModules[0], stored) || !reflect.DeepEqual(got, stored) {
		t.Fatalf("stored %+v\nanswered %+v\nwant %+v", w.updatedModules, got, stored)
	}
	if want := []removal{{lockModule, []uuid.UUID{changeKept}, w.now, lockEditor}}; !reflect.DeepEqual(w.removals, want) {
		t.Fatalf("removals %+v; want kept's row at the change's moment, by the caller", w.removals)
	}
	if len(w.memberRows) != 1 {
		t.Fatalf("member rows %+v", w.memberRows)
	}
	r := w.memberRows[0]
	if r.WorkspaceID != lockWorkspace || r.ProjectID != lockProject || r.ModuleID != lockModule || !slices.Equal(r.Accounts, added) ||
		len(r.IDs) != 2 || r.IDs[0] == r.IDs[1] || slices.Contains(r.IDs, uuid.UUID{}) || r.By != lockEditor || !r.At.Equal(w.now) {
		t.Fatalf("member rows %+v; want the added members in id order at the change's moment, by the caller", r)
	}
}

// The lead set to another is read, the lead cleared or kept is not; a change that adds and removes no member writes
// no member row; one that removes alone, or adds alone, writes that alone.
func TestUpdateModuleLeadAndMembers(t *testing.T) {
	for _, tt := range []struct {
		name   string
		p      domain.ModulePatch
		asked  [][]uuid.UUID
		calls  []string // the writes, after Now
		lead   *uuid.UUID
		member []uuid.UUID
	}{
		{"the lead cleared", domain.ModulePatch{SetLeadID: true}, nil, []string{"UpdateModule"}, nil, []uuid.UUID{changeKept, changeLeft}},
		{"another lead", domain.ModulePatch{SetLeadID: true, LeadID: &changeNew}, [][]uuid.UUID{{changeNew}}, []string{"UpdateModule"}, &changeNew,
			[]uuid.UUID{changeKept, changeLeft}},
		{"the lead kept, a member removed", domain.ModulePatch{SetLeadID: true, LeadID: &changeLeft, MemberIDs: &[]uuid.UUID{changeLeft}}, nil,
			[]string{"UpdateModule", "RemoveModuleMembers"}, &changeLeft, []uuid.UUID{changeLeft}},
		{"a member added", domain.ModulePatch{MemberIDs: &[]uuid.UUID{changeLeft, changeNew, changeKept}}, [][]uuid.UUID{{changeNew}},
			[]string{"UpdateModule", "InsertModuleMembers"}, &changeLeft, []uuid.UUID{changeKept, changeLeft, changeNew}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := changeWorld(t)
			got, err := w.changeModule(tt.p)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(w.asked, tt.asked) {
				t.Errorf("roles asked of %v; want %v", w.asked, tt.asked)
			}
			now := slices.Index(w.calls, "Now")
			if writes := w.calls[now+1 : len(w.calls)-2]; now < 0 || !slices.Equal(writes, tt.calls) {
				t.Errorf("calls %q; want the writes %q", w.calls, tt.calls)
			}
			want := slices.SortedFunc(slices.Values(tt.member), uuid.UUID.Compare)
			if !sameLead(got.LeadID, tt.lead) || !slices.Equal(got.MemberIDs, want) {
				t.Errorf("answered %+v; want the lead %v and the members %v", got, tt.lead, want)
			}
		})
	}
}

func sameLead(a, b *uuid.UUID) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

// A change of nothing — none given, each as it is, the members in another order — reads no role, nor the clock, and
// writes nothing; its answer is the module as the lock read it.
func TestUpdateModuleChangesNothing(t *testing.T) {
	for _, p := range []domain.ModulePatch{{}, {Name: str("Auth"), Description: str("Sign in"), SetStartDate: true, StartDate: nil,
		SetTargetDate: true, TargetDate: nil, Status: ptrTo(domain.ModulePlanned), SetLeadID: true, LeadID: &changeLeft,
		MemberIDs: &[]uuid.UUID{changeLeft, changeKept}, SortOrder: ptrTo(5.0)}} {
		w := changeWorld(t)
		if p.SetStartDate {
			p.StartDate, p.TargetDate = date(t, "2026-11-02"), date(t, "2026-11-13")
		}
		got, err := w.changeModule(p)
		if err != nil || !reflect.DeepEqual(got, w.modules[lockModule]) {
			t.Fatalf("%+v: %+v, %v; want the module as read", p, got, err)
		}
		for _, call := range []string{"Roles", "Now", "UpdateModule", "RemoveModuleMembers", "InsertModuleMembers"} {
			if slices.Contains(w.calls, call) {
				t.Fatalf("%+v: calls %q; want no %s", p, w.calls, call)
			}
		}
	}
}

// The refusals in their order (M6 P2 spec R2): a value's problem before anything is read; an archived module, any
// change, none too, after the decision; then, after the roles under the lock, every problem of the merged dates and
// of the newcomers at once (R4, R5); then the name the store refuses, after which no member row is written. Nothing
// refused is written.
func TestUpdateModuleRefusals(t *testing.T) {
	archived := func(w *lockWorld) {
		m := w.modules[lockModule]
		at := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
		m.ArchivedAt = &at
		w.modules[lockModule] = m
	}
	for _, tt := range []struct {
		name   string
		change func(w *lockWorld)
		p      domain.ModulePatch
		want   string   // the error's code
		fields []string // a 422's fields, as "field code"
	}{
		{"a blank name, archived", archived, domain.ModulePatch{Name: str(" "), MemberIDs: &[]uuid.UUID{changeNew, changeNew}}, "validation_failed",
			[]string{"name too_short", "member_ids[1] duplicate"}},
		{"archived, nothing given", archived, domain.ModulePatch{}, "planning.archived", nil},
		{"archived, a guest added", archived, domain.ModulePatch{MemberIDs: &[]uuid.UUID{changeGuest}}, "planning.archived", nil},
		{"the start alone after the target", nil, domain.ModulePatch{SetStartDate: true, StartDate: date(t, "2026-11-14")}, "validation_failed",
			[]string{"start_date not_allowed"}},
		{"every problem under the lock", nil, domain.ModulePatch{SetTargetDate: true, TargetDate: date(t, "2026-11-01"), SetLeadID: true,
			LeadID: &changeGuest, MemberIDs: &[]uuid.UUID{changeKept, changeLeft, uuid.NewV7(), changeAdmin}}, "validation_failed",
			[]string{"target_date not_allowed", "lead_id not_allowed", "member_ids[2] not_allowed"}},
		{"a name taken", func(w *lockWorld) { w.updateRefusal = domain.ErrModuleNameTaken },
			domain.ModulePatch{Name: str("Billing"), MemberIDs: &[]uuid.UUID{changeNew}}, "planning.module_name_taken", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := changeWorld(t)
			if tt.change != nil {
				tt.change(w)
			}
			_, err := w.changeModule(tt.p)
			var e *shared.Error
			if !errors.As(err, &e) || e.Code != tt.want {
				t.Fatalf("got %v; want %s", err, tt.want)
			}
			var fields []string
			for _, f := range e.Fields {
				fields = append(fields, f.Field+" "+string(f.Code))
			}
			if !slices.Equal(fields, tt.fields) {
				t.Fatalf("fields %q; want %q", fields, tt.fields)
			}
			if len(w.updatedModules) != 0 || len(w.removals) != 0 || len(w.memberRows) != 0 {
				t.Fatalf("written %+v %+v %+v", w.updatedModules, w.removals, w.memberRows)
			}
			switch tt.want {
			case shared.CodeValidationFailed:
				if decided := slices.Index(w.calls, "Authorize module.write"); tt.name == "a blank name, archived" && len(w.calls) != 0 ||
					tt.name != "a blank name, archived" && decided < 0 {
					t.Fatalf("calls %q", w.calls)
				}
			case "planning.archived":
				if slices.Contains(w.calls, "Roles") || !slices.Contains(w.calls, "Authorize module.write") {
					t.Fatalf("calls %q; want the refusal after the decision, before any role", w.calls)
				}
			}
		})
	}
}
