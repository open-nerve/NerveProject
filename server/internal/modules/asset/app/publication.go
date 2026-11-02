package app

import (
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

// Mutates only the selected candidate on the freshly locked row. Both copy and
// upload confirmation share this transition; no parent or database I/O occurs.
func publishCandidate(row *domain.Asset, candidate domain.Candidate, verified verifiedObject, now, at time.Time, by uuid.UUID) error {
	index := slices.IndexFunc(row.Metadata.Candidates, func(c domain.Candidate) bool { return c.Key == candidate.Key })
	if index < 0 || row.Metadata.Candidates[index].Phase != domain.ObjectActive || !row.Metadata.Candidates[index].LeaseUntil.Equal(candidate.LeaseUntil) || !now.Before(candidate.LeaseUntil) {
		return domain.ErrAssetScopeChanged
	}
	final := row.Metadata.Candidates[index].ObjectRecord
	final.ETag, final.SHA256 = verified.info.ETag, verified.proof.SHA256
	row.Metadata.Final = &final
	row.Metadata.Candidates = slices.Delete(row.Metadata.Candidates, index, index+1)
	row.Uploaded = true
	row.Width, row.Height = verified.proof.Width, verified.proof.Height
	row.UpdatedAt, row.UpdatedByID = at, &by
	return nil
}
