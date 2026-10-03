package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The ports of the writes that change one project membership (M3 design
// 3.5, 3.7): updateProjectMember and removeProjectMember, which name it by
// its id, and leaveProject, which names the project and the caller.

// ProjectMembership is a project membership as MemberByID reads it: its id,
// its workspace and project, its member's account, his role, and whether
// it is active.
type ProjectMembership struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	MemberID    uuid.UUID
	Role        shared.Role
	Active      bool
}

// MemberFinder reads a project membership by its id: what a write on it
// reads first, for the workspace and the project it locks, and again under
// their locks (Locks).
type MemberFinder interface {
	// MemberByID is the undeleted membership id, active or ended; found is
	// false when there is none.
	MemberByID(ctx context.Context, id uuid.UUID) (m ProjectMembership, found bool, err error)
}

// MemberRoleChanger is updateProjectMember's repository. It runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type MemberRoleChanger interface {
	// UpdateMemberRole gives the active membership id role, by the account
	// by at now, and returns it as stored. An ended or deleted one is an
	// error, and is not written.
	UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Member, error)
}

// MemberEnder is removeProjectMember's repository. It runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type MemberEnder interface {
	// EndMember ends userID's active membership of projectID, by the account
	// by at now; the row stays. Anything but exactly one row ended is an
	// error.
	EndMember(ctx context.Context, projectID, userID, by uuid.UUID, now time.Time) error
}

// MemberLeaver is leaveProject's repository. It runs in the transaction ctx
// carries, under the project's FOR NO KEY UPDATE.
type MemberLeaver interface {
	MemberEnder
	// HasOtherAdmin reports whether projectID has an active admin other than
	// userID.
	HasOtherAdmin(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
}
