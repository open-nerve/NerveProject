package app

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestAssetPublicationNeverReadoptsRecoveryKey(t *testing.T) {
	now := time.Now().UTC()
	id, actor := uuid.NewV7(), uuid.NewV7()
	key, _ := domain.NewObjectKey('o', id, uuid.NewV7())
	candidate := domain.Candidate{ObjectRecord: domain.ObjectRecord{Key: key, Phase: domain.ObjectActive, SafeAfter: now.Add(time.Minute)}, LeaseUntil: now.Add(30 * time.Second)}
	for _, phase := range []string{"removed", "reclaiming"} {
		t.Run(phase, func(t *testing.T) {
			row := domain.Asset{ID: id, UpdatedAt: now.Add(-time.Hour), Metadata: domain.StorageMetadata{SchemaVersion: 1, Origin: domain.OriginCopy, Candidates: []domain.Candidate{}}}
			if phase == "reclaiming" {
				c := candidate
				c.Phase = domain.ObjectReclaiming
				row.Metadata.Candidates = append(row.Metadata.Candidates, c)
			}
			before := row
			before.Metadata.Candidates = slices.Clone(row.Metadata.Candidates)
			err := publishCandidate(&row, candidate, verifiedObject{}, now, now, actor)
			if !errors.Is(err, domain.ErrAssetScopeChanged) || !reflect.DeepEqual(before, row) {
				t.Fatalf("cleanup state was reversed: %+v %v", row, err)
			}
		})
	}
}
