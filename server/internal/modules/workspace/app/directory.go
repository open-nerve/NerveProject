package app

import "uuid"

// DirectoryEntry is a workspace as the project module finds it by its slug
// through workspace.Provide's WorkspaceDirectory (M3 design 6.5): its id,
// and its time zone, a new project's unless the caller gives one (3.19).
// bootstrap converts it into project's value.
type DirectoryEntry struct {
	ID       uuid.UUID
	Timezone string
}
