package project

import "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"

// EventTypes declares the archive and deletion events; the request publisher
// is wired separately.
func EventTypes() []string { return []string{domain.EventArchived, "project.deleted"} }
