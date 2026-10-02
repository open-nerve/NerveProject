package domain

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Member is an active membership of a project, as the members' list shows
// it (M3 design 5.2): its id, the project, the member's account, his role
// in the project and when he became its member.
type Member struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	MemberID  uuid.UUID
	Role      shared.Role
	CreatedAt time.Time
}
