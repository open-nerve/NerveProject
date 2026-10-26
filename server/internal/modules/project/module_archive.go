package project

import (
	"context"
	"uuid"
)

// ArchivePolicies offers undeleted, unarchived project IDs with archive_in
// in [1,12], in ID order. Consumers must lock each candidate and re-read
// ArchiveIn before applying its policy.
type ArchivePolicies interface {
	ArchiveProjects(context.Context) ([]uuid.UUID, error)
}
