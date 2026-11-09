package app

import (
	"context"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// MemberRows are new rows of a module's members (M6 P2 spec 2.2): IDs[i] is
// the row of the account Accounts[i], under the module, written at the
// moment At by the account By.
type MemberRows struct {
	WorkspaceID, ProjectID, ModuleID uuid.UUID
	IDs, Accounts                    []uuid.UUID
	By                               uuid.UUID
	At                               time.Time
}

// ModuleCreator is the modules' store as a creation uses it, in the
// transaction ctx carries: the lowest sort order of the project's undeleted
// modules, archived ones too, nil when it has none; the new module, whose
// name another undeleted module of the project has, in any case, is
// domain.ErrModuleNameTaken (M6 P2 spec R8); and its members' rows.
type ModuleCreator interface {
	LowestModuleSortOrder(ctx context.Context, project uuid.UUID) (*float64, error)
	CreateModule(ctx context.Context, m domain.Module) error
	InsertModuleMembers(ctx context.Context, r MemberRows) error
}

// CreateModuleDeps are what CreateModule needs.
type CreateModuleDeps struct {
	Locks   Locks
	Roles   RoleReader
	Modules ModuleCreator
	Clock   Clock
}

// CreateModule creates a module in a project (createModule).
type CreateModule struct{ d CreateModuleDeps }

// NewCreateModule returns the use case over d.
func NewCreateModule(d CreateModuleDeps) *CreateModule { return &CreateModule{d: d} }

// Execute checks in's values, takes the project's lock and decision
// (Locks), holds the lead and every member in names to the project's roles
// as they are under the lock (M6 P2 spec R2, R5), and writes the module
// first among the project's, its members in account order, each a row of
// its own: all created and last changed by the caller at the moment read
// after the lock. The name is the store's to refuse (ModuleCreator).
func (u *CreateModule) Execute(ctx context.Context, projectID uuid.UUID, in domain.NewModule) (domain.Module, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Module{}, err
	}
	if in, err = domain.CheckNewModule(in); err != nil {
		return domain.Module{}, err
	}
	var out domain.Module
	err = u.d.Locks.run(ctx, actor, write{named: named{project: projectID}, action: domain.ActionModuleWrite}, func(ctx context.Context, h held) error {
		if accounts := accountsOf(in.LeadID, in.MemberIDs); len(accounts) > 0 {
			roles, err := u.d.Roles.Roles(ctx, projectID, accounts)
			if err != nil {
				return err
			}
			if err := in.CheckRoles(roles); err != nil {
				return err
			}
		}
		lowest, err := u.d.Modules.LowestModuleSortOrder(ctx, projectID)
		if err != nil {
			return err
		}
		now := u.d.Clock.Now()
		by := actor.UserID
		m := domain.Module{
			ID: uuid.NewV7(), ProjectID: projectID, WorkspaceID: h.project.WorkspaceID, Name: in.Name, Description: in.Description,
			StartDate: in.StartDate, TargetDate: in.TargetDate, Status: *in.Status, LeadID: in.LeadID,
			MemberIDs: slices.SortedFunc(slices.Values(in.MemberIDs), uuid.UUID.Compare), SortOrder: domain.NextSortOrder(lowest),
			CreatedAt: now, UpdatedAt: now, CreatedByID: &by, UpdatedByID: &by,
		}
		if err := u.d.Modules.CreateModule(ctx, m); err != nil {
			return err
		}
		if len(m.MemberIDs) > 0 {
			if err := u.d.Modules.InsertModuleMembers(ctx, memberRows(m, m.MemberIDs, by, now)); err != nil {
				return err
			}
		}
		out = m
		return nil
	})
	return out, err
}

// accountsOf are the accounts a module's values name: its lead, when it has
// one, and its members.
func accountsOf(lead *uuid.UUID, members []uuid.UUID) []uuid.UUID {
	if lead == nil {
		return members
	}
	return append([]uuid.UUID{*lead}, members...)
}

// memberRows are the new rows of the accounts under the module m, each with
// an id of its own, in their order, by the account by at at.
func memberRows(m domain.Module, accounts []uuid.UUID, by uuid.UUID, at time.Time) MemberRows {
	r := MemberRows{WorkspaceID: m.WorkspaceID, ProjectID: m.ProjectID, ModuleID: m.ID, Accounts: accounts, By: by, At: at}
	for range accounts {
		r.IDs = append(r.IDs, uuid.NewV7())
	}
	return r
}
