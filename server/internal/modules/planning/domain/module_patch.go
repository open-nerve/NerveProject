package domain

import (
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ModulePatch is a change of a module (updateModule's body): each field nil
// when it is not given, and keeps its value. A date and the lead may be
// cleared: SetStartDate, SetTargetDate and SetLeadID tell whether the field
// is given, StartDate, TargetDate and LeadID its value, nil for none.
// MemberIDs, given, is the module's new set of members; given empty, none.
type ModulePatch struct {
	Name, Description           *string
	SetStartDate, SetTargetDate bool
	StartDate, TargetDate       *shared.Date
	Status                      *ModuleStatus
	SetLeadID                   bool
	LeadID                      *uuid.UUID
	MemberIDs                   *[]uuid.UUID
	SortOrder                   *float64
}

// CheckModulePatch is p with its name cleaned, or every problem of the
// values it gives that needs no read, as validation_failed, in the body's
// order (M6 P2 spec R2): the name (CleanName), a description with NUL, a
// status that is not one of the six, more than MaxModuleMembers members or
// one named twice, a sort order that is no finite number. The dates it
// gives, and the lead and the members it makes anew, are held to the module
// under the lock (Apply).
func CheckModulePatch(p ModulePatch) (ModulePatch, error) {
	var found []*shared.FieldError
	if p.Name != nil {
		name, problem := CleanName(*p.Name)
		p.Name = &name
		found = append(found, problem)
	}
	if p.Description != nil {
		found = append(found, checkDescription(*p.Description))
	}
	if p.Status != nil {
		found = append(found, checkModuleStatus(*p.Status))
	}
	if p.MemberIDs != nil {
		found = append(found, checkMemberIDs(*p.MemberIDs)...)
	}
	found = append(found, checkSortOrder(p.SortOrder))
	if err := invalid(found...); err != nil {
		return ModulePatch{}, err
	}
	return p, nil
}

// Apply is m as p, checked (CheckModulePatch), leaves it, and whether p
// changes anything: a field given as it is changes nothing, and neither do
// member_ids that name m's members in any order (M6 P2 spec R4); the members
// it leaves are in id order. Under the lock, with roles, the active project
// roles of the accounts p names, every problem is refused at once as
// validation_failed, each at its place in p: the dates p gives, each else
// m's, with the start after the target (R4: at start_date when p gives the
// start date alone, else at target_date); the lead p sets anew, and each
// member it adds, who may not be (CanJoin, R5). A lead or a member m keeps is
// not looked at, even one who has left the project since.
func (p ModulePatch) Apply(m Module, roles map[uuid.UUID]shared.Role) (Module, bool, error) {
	next := m
	if p.Name != nil {
		next.Name = *p.Name
	}
	if p.Description != nil {
		next.Description = *p.Description
	}
	if p.SetStartDate {
		next.StartDate = p.StartDate
	}
	if p.SetTargetDate {
		next.TargetDate = p.TargetDate
	}
	if p.Status != nil {
		next.Status = *p.Status
	}
	if p.SetLeadID {
		next.LeadID = p.LeadID
	}
	if p.SortOrder != nil {
		next.SortOrder = *p.SortOrder
	}
	var members []uuid.UUID
	var diff MemberDiff
	if p.MemberIDs != nil {
		members = *p.MemberIDs
		diff = DiffMembers(m.MemberIDs, members)
	}
	if diff.Added != nil || diff.Removed != nil {
		next.MemberIDs = slices.SortedFunc(slices.Values(members), uuid.UUID.Compare)
	}
	var lead *uuid.UUID
	if !sameAccount(next.LeadID, m.LeadID) {
		lead = next.LeadID
	}
	found := []*shared.FieldError{p.datesProblem(next)}
	found = append(found, refused(lead, members, func(id uuid.UUID) bool { return slices.Contains(diff.Added, id) }, roles)...)
	if err := invalid(found...); err != nil {
		return Module{}, false, err
	}
	changed := next.Name != m.Name || next.Description != m.Description || !sameDay(next.StartDate, m.StartDate) ||
		!sameDay(next.TargetDate, m.TargetDate) || next.Status != m.Status || !sameAccount(next.LeadID, m.LeadID) ||
		next.SortOrder != m.SortOrder || diff.Added != nil || diff.Removed != nil
	return next, changed, nil
}

// Newcomers are the accounts p makes anew on m, whose project roles Apply
// holds them to (M6 P2 spec R5), so the only ones whose roles a change
// reads: the lead it sets, when it is another than m's, then each member it
// adds, in id order. A lead it keeps or clears, and a member it keeps or
// removes, is none; nil when there is no newcomer.
func (p ModulePatch) Newcomers(m Module) []uuid.UUID {
	var ids []uuid.UUID
	if p.SetLeadID && p.LeadID != nil && !sameAccount(p.LeadID, m.LeadID) {
		ids = append(ids, *p.LeadID)
	}
	if p.MemberIDs != nil {
		ids = append(ids, DiffMembers(m.MemberIDs, *p.MemberIDs).Added...)
	}
	return ids
}

// datesProblem is the problem of next's dates, those p gives merged with the
// module's, when the start comes after the target: at start_date when p gives
// the start date alone, since the target is the module's; else at
// target_date.
func (p ModulePatch) datesProblem(next Module) *shared.FieldError {
	switch {
	case !outOfOrder(next.StartDate, next.TargetDate):
		return nil
	case p.SetStartDate && !p.SetTargetDate:
		return &shared.FieldError{Field: "start_date", Code: shared.FieldNotAllowed, Message: "must not be after the target date"}
	}
	return targetBeforeStart()
}

// MemberDiff is how a module's members become a change's (M6 P2 spec R5):
// the accounts it adds and those it removes, each in id order, nil for none.
// A member both have is in neither: its row is left as it is.
type MemberDiff struct {
	Added, Removed []uuid.UUID
}

// DiffMembers is the change of from, a module's members, into to, a change's
// member_ids, each in any order: the accounts of to that from has not, and
// those of from that to has not.
func DiffMembers(from, to []uuid.UUID) MemberDiff {
	var d MemberDiff
	for _, id := range slices.SortedFunc(slices.Values(to), uuid.UUID.Compare) {
		if !slices.Contains(from, id) {
			d.Added = append(d.Added, id)
		}
	}
	for _, id := range slices.SortedFunc(slices.Values(from), uuid.UUID.Compare) {
		if !slices.Contains(to, id) {
			d.Removed = append(d.Removed, id)
		}
	}
	return d
}

// sameAccount reports whether a and b, either nil for none, are both none or
// the same account.
func sameAccount(a, b *uuid.UUID) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}
