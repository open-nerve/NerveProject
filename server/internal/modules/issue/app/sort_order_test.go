package app_test

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func TestIssueSortOrderPatch(t *testing.T) {
	for _, state := range []bool{false, true} {
		t.Run(map[bool]string{false: "sort alone", true: "state and sort"}[state], func(t *testing.T) {
			uc, w, id := newUpdate()
			zero := 0.0
			p := domain.IssuePatch{SortOrder: &zero}
			if state {
				p.StateID = &done.ID
			}
			got, err := uc.Execute(as(alice), id, p)
			if err != nil || got.SortOrder != 0 || w.issues.changed == nil || w.issues.changed.SortOrder != 0 || got.UpdatedByID == nil || *got.UpdatedByID != alice || !got.UpdatedAt.Equal(clockNow) {
				t.Fatalf("sort write: %+v, %v, fields %+v", got, err, w.issues.changed)
			}
			if state {
				if got.StateID != done.ID || got.CompletedAt == nil || !got.CompletedAt.Equal(clockNow) {
					t.Fatal("state lost", got)
				}
				wantActivity(t, w, id, domain.UpdateActivity(domain.Update{State: &domain.Changed[domain.Named]{Old: domain.Named{ID: backlog.ID, Name: "Backlog"}, New: domain.Named{ID: done.ID, Name: "Done"}}}))
			} else {
				wantActivity(t, w, id, nil)
			}
		})
	}
	t.Run("locked value and omission", func(t *testing.T) {
		uc, w, id := newUpdate()
		w.project.waited = func() { w.issues.issue.SortOrder = 17.25 }
		got, err := uc.Execute(as(alice), id, domain.IssuePatch{Name: name("renamed")})
		if err != nil || got.SortOrder != 17.25 || w.issues.changed.SortOrder != 17.25 {
			t.Fatal("lost locked sort", got, err, w.issues.changed)
		}
	})
	t.Run("same value takes locks but writes nothing", func(t *testing.T) {
		uc, w, id := newUpdate()
		v := 65535.0
		_, err := uc.Execute(as(alice), id, domain.IssuePatch{SortOrder: &v})
		if err != nil || w.issues.changed != nil || !slices.Equal(w.log.calls, append(locked(id), counted(id))) {
			t.Fatal("same value wrote/read clock", err, w.log.calls)
		}
	})
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		uc, w, id := newUpdate()
		_, err := uc.Execute(as(alice), id, domain.IssuePatch{SortOrder: &value})
		if err == nil || len(w.log.calls) != 0 {
			t.Fatal("invalid value reached ports", value, err, w.log.calls)
		}
	}
	for _, project := range []bool{false, true} {
		uc, w, id := newUpdate()
		if project {
			w.project.locked.Archived = true
		} else {
			w.issues.archivedAt = &earlier
		}
		v := 65535.0
		_, err := uc.Execute(as(alice), id, domain.IssuePatch{SortOrder: &v})
		want := domain.ErrIssueArchived
		if project {
			want = domain.ErrProjectArchived
		}
		unchanged(t, w, err, want, locked(id))
	}
	// The finite validator cannot turn a checked patch into a different value.
	uc, w, id := newUpdate()
	v := math.MaxFloat64
	got, err := uc.Execute(as(alice), id, domain.IssuePatch{SortOrder: &v})
	if err != nil || !reflect.DeepEqual(got.SortOrder, v) || w.issues.changed.SortOrder != v {
		t.Fatal("finite boundary lost", got, err)
	}
}
