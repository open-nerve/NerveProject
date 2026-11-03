package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaceMembers lists a workspace's members: GET
// /api/v0/workspaces/{slug}/members.
type ListMemberRows interface {
	MemberLister
	ImageWorkspaceSharer
}
type ListMembersDeps struct {
	Members  ListMemberRows
	Profiles MemberProfiles
	Auth     shared.Authorizer
	Caller   CallerLock
	Images   MemberAvatarReader
	Tx       shared.TxManager
	Clock    Clock
}
type ListWorkspaceMembers struct{ d ListMembersDeps }

func NewListWorkspaceMembers(d ListMembersDeps) *ListWorkspaceMembers {
	return &ListWorkspaceMembers{d: d}
}

// Execute returns every undeleted membership of the workspace, ended ones
// too, each with its member's public profile, deactivated accounts' too (M3
// design 5.1). The addresses are there for a caller whose role sees them
// (domain.SeesEmails), the caller's own included. Caller and workspace locks
// protect authorization until the avatar capabilities have been signed.
func (u *ListWorkspaceMembers) Execute(ctx context.Context, slug string) ([]domain.Member, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	w, err := u.d.Members.WorkspaceBySlug(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return nil, domain.ErrNotFound
	case err != nil:
		return nil, err
	}
	var out []domain.Member
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := u.d.Caller.LockCaller(ctx, actor, u.d.Clock.Now()); err != nil {
			return err
		}
		locked, err := u.d.Members.ShareImageWorkspaces(ctx, []uuid.UUID{w.ID})
		if err != nil {
			return err
		}
		if !slices.Contains(locked, w.ID) {
			return domain.ErrNotFound
		}
		grant, err := decide(ctx, u.d.Auth, actor, domain.ActionMemberList, w.ID, domain.ErrNotFound)
		if err != nil {
			return err
		}
		members, err := u.list(ctx, w.ID, grant.WorkspaceRole)
		if err != nil {
			return err
		}
		out, err = withMemberAvatars(ctx, u.d.Images, members)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (u *ListWorkspaceMembers) list(ctx context.Context, workspaceID uuid.UUID, role shared.Role) ([]domain.Member, error) {
	memberships, err := u.d.Members.ListMembers(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(memberships))
	for i, m := range memberships {
		ids[i] = m.MemberID
	}
	profiles, err := u.d.Profiles.PublicProfiles(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]PublicProfile, len(profiles))
	for _, p := range profiles {
		byID[p.ID] = p
	}
	seesEmails := domain.SeesEmails(role)
	out := make([]domain.Member, len(memberships))
	for i, m := range memberships {
		p, ok := byID[m.MemberID]
		if !ok {
			// The foreign key keeps every member's account: its absence is a bug.
			return nil, fmt.Errorf("workspace member %s: no account %s", m.ID, m.MemberID)
		}
		out[i] = domain.Member{Membership: m, User: memberUser(p, seesEmails)}
	}
	return out, nil
}

// memberUser is p as a caller sees it: the address only when seesEmails.
func memberUser(p PublicProfile, seesEmails bool) domain.MemberUser {
	user := domain.MemberUser{AvatarAssetID: p.AvatarAssetID, ID: p.ID, DisplayName: p.DisplayName, FirstName: p.FirstName, LastName: p.LastName}
	if seesEmails {
		user.Email = &p.Email
	}
	return user
}
