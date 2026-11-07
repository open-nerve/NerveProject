package domain

import (
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ModuleStatus is a module's status: stored, set by its writers, not read
// from its dates (M6 design 3.2).
type ModuleStatus string

// The statuses modules_status_check allows.
const (
	ModuleBacklog    ModuleStatus = "backlog"
	ModulePlanned    ModuleStatus = "planned"
	ModuleInProgress ModuleStatus = "in-progress"
	ModulePaused     ModuleStatus = "paused"
	ModuleCompleted  ModuleStatus = "completed"
	ModuleCancelled  ModuleStatus = "cancelled"
)

// moduleStatuses are the statuses, as a request spells them.
var moduleStatuses = []ModuleStatus{ModuleBacklog, ModulePlanned, ModuleInProgress, ModulePaused, ModuleCompleted, ModuleCancelled}

// Module is a module of a project as the module stores it (M6 P2 spec 2.2):
// each of its dates a day or none, the start not after the target when it
// has both, which modules_dates_check holds too; the accounts of its
// undeleted members' rows in id order.
type Module struct {
	ID, ProjectID, WorkspaceID uuid.UUID
	Name, Description          string
	StartDate, TargetDate      *shared.Date
	Status                     ModuleStatus
	LeadID                     *uuid.UUID
	MemberIDs                  []uuid.UUID
	SortOrder                  float64
	ArchivedAt                 *time.Time
	CreatedAt, UpdatedAt       time.Time
	CreatedByID, UpdatedByID   *uuid.UUID
}

// MaxModuleMembers is the most accounts a request makes a module's members
// (M6 P2 spec R5), as a work item's assignees.
const MaxModuleMembers = 100

// NewModule is a module as a creation gives it (createModule's body): a date
// nil for none, Status nil when it is not given, LeadID nil for no lead.
type NewModule struct {
	Name, Description     string
	StartDate, TargetDate *shared.Date
	Status                *ModuleStatus
	LeadID                *uuid.UUID
	MemberIDs             []uuid.UUID
}

// CheckNewModule is n with its name cleaned and its status planned when it
// gives none, so never nil once checked; or every problem of its values, as
// validation_failed, in the body's order: checked before anything is read
// (M6 P2 spec R2). The name (CleanName); a description with NUL; the start
// date after the target date, each of which may be given alone (at
// target_date); a status given that is not one of the six as they are
// spelt; more than MaxModuleMembers members, or one named twice. Whether its
// lead and members may be is decided under the lock (CheckRoles).
func CheckNewModule(n NewModule) (NewModule, error) {
	name, problem := CleanName(n.Name)
	n.Name = name
	if n.Status == nil {
		planned := ModulePlanned
		n.Status = &planned
	}
	found := []*shared.FieldError{problem, checkDescription(n.Description), checkModuleDates(n.StartDate, n.TargetDate), checkModuleStatus(*n.Status)}
	found = append(found, checkMemberIDs(n.MemberIDs)...)
	if err := invalid(found...); err != nil {
		return NewModule{}, err
	}
	return n, nil
}

// CheckRoles refuses n's lead and each of its members who may not be
// (CanJoin), by roles, the active project roles of the accounts n names, as
// validation_failed, each at its place: lead_id, member_ids[i] (M6 P2 spec
// R5). A creation makes every one of them anew, so every one is held to it.
func (n NewModule) CheckRoles(roles map[uuid.UUID]shared.Role) error {
	return invalid(refused(n.LeadID, n.MemberIDs, func(uuid.UUID) bool { return true }, roles)...)
}

// joinRoles are the project roles of a module's lead and members: not its
// guests (M6 P2 spec R5).
var joinRoles = []shared.Role{shared.RoleAdmin, shared.RoleMember}

// CanJoin reports whether an active member of a module's project, of project
// role role, may be its lead or one of its members (M6 P2 spec R5, as a work
// item's assignee): an admin or a member, not a guest. An account who is no
// active member of the project has no role, the zero Role, and may not. The
// set is named, not a bound.
func CanJoin(role shared.Role) bool {
	return slices.Contains(joinRoles, role)
}

// refused are the problems of the lead lead, when it is not nil, and of each
// of members that fresh reports, at its index, who may not be by roles
// (CanJoin).
func refused(lead *uuid.UUID, members []uuid.UUID, fresh func(uuid.UUID) bool, roles map[uuid.UUID]shared.Role) []*shared.FieldError {
	const message = "must be an active member of the project who is not its guest"
	var found []*shared.FieldError
	if lead != nil && !CanJoin(roles[*lead]) {
		found = append(found, &shared.FieldError{Field: "lead_id", Code: shared.FieldNotAllowed, Message: message})
	}
	for i, id := range members {
		if fresh(id) && !CanJoin(roles[id]) {
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("member_ids[%d]", i), Code: shared.FieldNotAllowed, Message: message})
		}
	}
	return found
}

// checkModuleStatus refuses a status that is not one of the six as they are
// spelt: in another case, with _ for -, or empty.
func checkModuleStatus(s ModuleStatus) *shared.FieldError {
	if !slices.Contains(moduleStatuses, s) {
		return &shared.FieldError{Field: "status", Code: shared.FieldInvalidFormat,
			Message: "must be one of backlog, planned, in-progress, paused, completed, cancelled"}
	}
	return nil
}

// checkModuleDates refuses a start date after the target date, at
// target_date; a date missing orders nothing, and the same day for both is
// in order.
func checkModuleDates(start, target *shared.Date) *shared.FieldError {
	if outOfOrder(start, target) {
		return targetBeforeStart()
	}
	return nil
}

// outOfOrder reports whether start comes after target, either nil for none.
func outOfOrder(start, target *shared.Date) bool {
	return start != nil && target != nil && start.Compare(*target) > 0
}

// targetBeforeStart is the problem of a target date before the start date.
func targetBeforeStart() *shared.FieldError {
	return &shared.FieldError{Field: "target_date", Code: shared.FieldNotAllowed, Message: "must not be before the start date"}
}

// checkMemberIDs refuses more than MaxModuleMembers members, as its one
// problem: no account of a list too long is looked at; otherwise each
// account named before in the list, at its index.
func checkMemberIDs(ids []uuid.UUID) []*shared.FieldError {
	if len(ids) > MaxModuleMembers {
		return []*shared.FieldError{{Field: "member_ids", Code: shared.FieldTooLong, Message: fmt.Sprintf("must name at most %d", MaxModuleMembers)}}
	}
	var found []*shared.FieldError
	named := make(map[uuid.UUID]bool, len(ids))
	for i, id := range ids {
		if named[id] {
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("member_ids[%d]", i), Code: shared.FieldDuplicate, Message: "is listed before"})
		}
		named[id] = true
	}
	return found
}

// invalid is the 422 of the problems found, in order, or nil when there is
// none.
func invalid(found ...*shared.FieldError) error {
	var fields []shared.FieldError
	for _, f := range found {
		if f != nil {
			fields = append(fields, *f)
		}
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}
