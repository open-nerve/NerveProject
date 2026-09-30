package app

import (
	"context"
	"errors"
	"slices"
	"strings"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateWorkspaceInvitations invites addresses to a workspace: POST
// /api/v0/workspaces/{slug}/invitations.
type CreateWorkspaceInvitations struct {
	d      CreateInvitationsDeps
	tokens invitationTokens
}

// CreateInvitationsDeps are the use case's ports.
type CreateInvitationsDeps struct {
	Caller      CallerLock
	Invitations InvitationCreator
	Profiles    MemberProfiles
	Auth        shared.Authorizer
	Tx          shared.TxManager
	Clock       Clock
	MAC         InvitationMAC
}

// NewCreateWorkspaceInvitations returns the use case.
func NewCreateWorkspaceInvitations(d CreateInvitationsDeps) *CreateWorkspaceInvitations {
	return &CreateWorkspaceInvitations{d: d, tokens: invitationTokens{mac: d.MAC}}
}

// Execute checks the batch, then, in one transaction, in the order of M3
// design 3.6: the caller's account row, his credential checked under the
// lock (CallerLock), so that a password reset committed first leaves no
// invitation (3.8); the workspace FOR SHARE; the decision on
// workspace_invitation.create; the addresses of active members and of
// undeleted invitations refused; then the rows inserted in the order of
// their addresses, so that two batches of overlapping addresses under the
// shared lock wait for each other in one direction and never deadlock (3.6
// convention 5). All of the batch or none of it. The rows are new, so the
// clock is read once, before the transaction (P2 spec 2.6 is for rows that
// exist): the credential is checked at the request's time, as M2's token
// creation does. The answer is the batch in the request's order, each
// invitation with its token.
func (u *CreateWorkspaceInvitations) Execute(ctx context.Context, slug string, batch []domain.NewInvitation) ([]domain.InvitationWithToken, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	batch, err = domain.CheckInvitations(batch)
	if err != nil {
		return nil, err
	}
	now := u.d.Clock.Now()
	rows := make([]InvitationRow, len(batch))
	for i, inv := range batch {
		rows[i] = InvitationRow{ID: uuid.NewV7(), Email: inv.Email, Role: inv.Role, CreatedBy: actor.UserID, Now: now}
	}
	var stored []domain.Invitation
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := u.d.Caller.LockCaller(ctx, actor, now); err != nil {
			return err
		}
		id, _, err := lockAndDecide(ctx, u.d.Invitations.ShareWorkspaceBySlug, u.d.Auth, actor, slug, domain.ActionInvitationCreate)
		if err != nil {
			return err
		}
		if err := u.checkAddresses(ctx, id, batch); err != nil {
			return err
		}
		for i := range rows {
			rows[i].WorkspaceID = id
		}
		stored, err = u.d.Invitations.CreateInvitations(ctx, byAddress(rows))
		var taken *DuplicateInvitation
		if errors.As(err, &taken) {
			// A concurrent batch committed the address after the check: the
			// same answer as the check's (3.8).
			return shared.Invalid(domain.InvitedAddress(slices.IndexFunc(batch, func(inv domain.NewInvitation) bool { return inv.Email == taken.Email })))
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]domain.Invitation, len(stored))
	for _, inv := range stored {
		byID[inv.ID] = inv
	}
	out := make([]domain.Invitation, len(rows))
	for i, r := range rows {
		out[i] = byID[r.ID]
	}
	return u.tokens.withTokens(out), nil
}

// checkAddresses refuses, as one 422, each address of the batch that an
// active member of the workspace has (not_allowed: an invitation never
// changes an active membership) or an undeleted invitation of it has,
// pending or declined (duplicate) (M3 design 3.8). The members' addresses
// come through MemberProfiles, without a lock: Accounts is only ever a
// transaction's first lock (3.6 convention 1).
func (u *CreateWorkspaceInvitations) checkAddresses(ctx context.Context, workspaceID uuid.UUID, batch []domain.NewInvitation) error {
	members, err := u.d.Invitations.ListMembers(ctx, workspaceID)
	if err != nil {
		return err
	}
	var active []uuid.UUID
	for _, m := range members {
		if m.IsActive {
			active = append(active, m.MemberID)
		}
	}
	profiles, err := u.d.Profiles.PublicProfiles(ctx, active)
	if err != nil {
		return err
	}
	memberAddresses := map[string]bool{}
	for _, p := range profiles {
		memberAddresses[p.Email] = true
	}
	invitations, err := u.d.Invitations.ListInvitations(ctx, workspaceID)
	if err != nil {
		return err
	}
	invited := map[string]bool{}
	for _, inv := range invitations {
		invited[inv.Email] = true
	}
	var fields []shared.FieldError
	for i, inv := range batch {
		switch {
		case memberAddresses[inv.Email]:
			fields = append(fields, domain.MemberAddress(i))
		case invited[inv.Email]:
			fields = append(fields, domain.InvitedAddress(i))
		}
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}

// byAddress is rows in the order of their addresses, bytewise: the one
// order every batch inserts in (M3 design 3.6 convention 5).
func byAddress(rows []InvitationRow) []InvitationRow {
	return slices.SortedFunc(slices.Values(rows), func(a, b InvitationRow) int { return strings.Compare(a.Email, b.Email) })
}
