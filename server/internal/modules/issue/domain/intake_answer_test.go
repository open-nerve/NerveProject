package domain

import (
	"reflect"
	"testing"
	"time"
	"uuid"
)

// An intake row as a detail answers it at a moment (spec 2.8): its
// effective status; till when it is snoozed only while it is, so none once
// the snooze has run out, at the moment too; the work item it duplicates
// only while it is a duplicate, as stored, whose sight by the reader the
// use case decides. The moment is asked once, and only of a snooze, the
// one status whose answer it changes. The row itself is not changed.
func TestIntakeFactsAsOf(t *testing.T) {
	later, earlier := intakeNow.Add(time.Hour), intakeNow.Add(-time.Microsecond)
	target := uuid.NewV7()
	for _, tt := range []struct {
		name  string
		f     IntakeFacts
		want  IntakeFacts
		reads int
	}{
		{"pending", IntakeFacts{Status: IntakePending}, IntakeFacts{Status: IntakePending}, 0},
		{"snoozed till later", IntakeFacts{Status: IntakeSnoozed, SnoozedTill: &later}, IntakeFacts{Status: IntakeSnoozed, SnoozedTill: &later}, 1},
		{"snoozed till the moment", IntakeFacts{Status: IntakeSnoozed, SnoozedTill: &intakeNow}, IntakeFacts{Status: IntakePending}, 1},
		{"snoozed till before it", IntakeFacts{Status: IntakeSnoozed, SnoozedTill: &earlier}, IntakeFacts{Status: IntakePending}, 1},
		{"a duplicate", IntakeFacts{Status: IntakeDuplicate, DuplicateToID: &target}, IntakeFacts{Status: IntakeDuplicate, DuplicateToID: &target}, 0},
		{"accepted", IntakeFacts{Status: IntakeAccepted}, IntakeFacts{Status: IntakeAccepted}, 0},
		{"declined", IntakeFacts{Status: IntakeDeclined}, IntakeFacts{Status: IntakeDeclined}, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stored, reads := tt.f, 0
			now := func() time.Time { reads++; return intakeNow }
			if got := tt.f.AsOf(now); !reflect.DeepEqual(got, tt.want) || reads != tt.reads {
				t.Errorf("AsOf(%v) = %+v after %d reads of the moment, want %+v after %d", intakeNow, got, reads, tt.want, tt.reads)
			}
			if !reflect.DeepEqual(tt.f, stored) {
				t.Errorf("AsOf changed the row to %+v", tt.f)
			}
		})
	}
}
