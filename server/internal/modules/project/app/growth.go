package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// growth is one account made a member of a project, by addProjectMembers
// or joinProject, under the project's FOR NO KEY UPDATE and his workspace
// membership's FOR SHARE (M3 design 3.6 conventions 3 and 6).
type growth struct {
	workspaceID, projectID, user uuid.UUID
	// ended is his ended membership of the project, nil when he has none.
	ended     *Membership
	role      shared.Role
	sortOrder float64 // the project's place in his sidebar, if he has none there
	by        uuid.UUID
	now       time.Time
}

// apply writes g: his ended membership restored with g's role, or a new
// one; then his display settings in the project at g's place, unless he
// has undeleted ones there, which a restored membership keeps.
func (g growth) apply(ctx context.Context, p MemberGrower) error {
	if g.ended != nil {
		if err := p.RestoreMember(ctx, g.ended.ID, g.role, g.by, g.now); err != nil {
			return err
		}
	} else if err := p.CreateMember(ctx, MemberRow{ID: uuid.NewV7(), WorkspaceID: g.workspaceID, ProjectID: g.projectID, MemberID: g.user,
		Role: g.role, CreatedBy: g.by, Now: g.now}); err != nil {
		return err
	}
	return p.EnsurePreferences(ctx, PreferencesRow{ID: uuid.NewV7(), WorkspaceID: g.workspaceID, ProjectID: g.projectID, UserID: g.user,
		SortOrder: g.sortOrder, CreatedBy: g.by, Now: g.now})
}

// endedOf is the ended membership of memberships' account user, nil when
// he has none or his is active.
func endedOf(memberships map[uuid.UUID]Membership, user uuid.UUID) *Membership {
	if m, ok := memberships[user]; ok && !m.Active {
		return &m
	}
	return nil
}
