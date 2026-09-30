package app

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaceMembers lists a workspace's members: GET
// /api/v0/workspaces/{slug}/members.
type ListWorkspaceMembers struct {
	members  MemberLister
	profiles MemberProfiles
	auth     shared.Authorizer
}

// NewListWorkspaceMembers returns the use case.
func NewListWorkspaceMembers(members MemberLister, profiles MemberProfiles, auth shared.Authorizer) *ListWorkspaceMembers {
	return &ListWorkspaceMembers{members: members, profiles: profiles, auth: auth}
}

// Execute returns every undeleted membership of the workspace, ended ones
// too, each with its member's public profile, deactivated accounts' too (M3
// design 5.1). The addresses are there for a caller whose role sees them
// (domain.SeesEmails), the caller's own included. A read opens no
// transaction and decides directly (M3 design 3.4).
func (u *ListWorkspaceMembers) Execute(ctx context.Context, slug string) ([]domain.Member, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	w, err := u.members.WorkspaceBySlug(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return nil, domain.ErrNotFound
	case err != nil:
		return nil, err
	}
	grant, err := u.auth.Authorize(ctx, actor, domain.ActionMemberList, shared.Target{WorkspaceID: w.ID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return nil, domain.ErrNotFound
	case err != nil:
		return nil, err
	}
	memberships, err := u.members.ListMembers(ctx, w.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(memberships))
	for i, m := range memberships {
		ids[i] = m.MemberID
	}
	profiles, err := u.profiles.PublicProfiles(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]PublicProfile, len(profiles))
	for _, p := range profiles {
		byID[p.ID] = p
	}
	seesEmails := domain.SeesEmails(grant.WorkspaceRole)
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
	user := domain.MemberUser{ID: p.ID, DisplayName: p.DisplayName, FirstName: p.FirstName, LastName: p.LastName}
	if seesEmails {
		user.Email = &p.Email
	}
	return user
}
