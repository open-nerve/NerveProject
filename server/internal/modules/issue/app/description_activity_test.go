package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

type descriptionActivities struct {
	latest                       ActivityRecord
	found                        bool
	readErr, insertErr, mergeErr error
	issue                        uuid.UUID
	inserted                     []ActivityRow
	merged                       []DescriptionActivityMerge
}

func (f *descriptionActivities) LatestActivity(_ context.Context, issue uuid.UUID) (ActivityRecord, bool, error) {
	f.issue = issue
	return f.latest, f.found, f.readErr
}
func (f *descriptionActivities) InsertActivities(_ context.Context, rows []ActivityRow) error {
	f.inserted = append(f.inserted, rows...)
	return f.insertErr
}
func (f *descriptionActivities) MergeDescriptionActivity(_ context.Context, m DescriptionActivityMerge) error {
	f.merged = append(f.merged, m)
	return f.mergeErr
}

func TestDescriptionActivityMerge(t *testing.T) {
	actor, other := uuid.NewV7(), uuid.NewV7()
	description, name := "description", "name"
	s := stamp{workspace: uuid.NewV7(), project: uuid.NewV7(), issue: uuid.NewV7(), by: actor, at: time.Now().UTC()}
	change := domain.Changed[string]{Old: "<p>before</p>", New: "<p>after</p>"}
	for _, tt := range []struct {
		name         string
		field        *string
		actor        *uuid.UUID
		found, merge bool
	}{
		{"latest own description", &description, &actor, true, true},
		{"intervening name interrupts older own description", &name, &actor, true, false},
		{"intervening other actor interrupts older own description", &description, &other, true, false},
		{"system activity", &description, nil, true, false},
		{"creation", nil, &actor, true, false},
		{"no activity", nil, nil, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &descriptionActivities{found: tt.found, latest: ActivityRecord{ID: uuid.NewV7(), ActorID: tt.actor, Activity: domain.Activity{Field: tt.field}}}
			if err := writeDescriptionActivity(context.Background(), f, s, change); err != nil {
				t.Fatal(err)
			}
			if f.issue != s.issue {
				t.Fatalf("read issue %s, want %s", f.issue, s.issue)
			}
			if tt.merge {
				want := []DescriptionActivityMerge{{ID: f.latest.ID, By: actor, NewHTML: change.New, At: s.at}}
				if !reflect.DeepEqual(f.merged, want) || len(f.inserted) != 0 {
					t.Fatalf("merge = %+v, inserted = %+v; want %+v only", f.merged, f.inserted, want)
				}
			} else {
				if len(f.merged) != 0 || len(f.inserted) != 1 {
					t.Fatalf("merge = %+v, inserted = %+v; want one new description", f.merged, f.inserted)
				}
				got := f.inserted[0]
				if got.ID == (uuid.UUID{}) {
					t.Fatal("new activity must have an id")
				}
				old, next := change.Old, change.New
				want := ActivityRow{ID: got.ID, WorkspaceID: s.workspace, ProjectID: s.project, IssueID: s.issue, ActorID: &s.by, At: s.at,
					Activity: domain.Activity{Field: &description, Verb: domain.VerbUpdated, OldValue: &old, NewValue: &next, Comment: "updated the description to"}}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("new activity = %+v, want %+v", got, want)
				}
			}
		})
	}
	t.Run("port errors stay unchanged", func(t *testing.T) {
		sentinel := errors.New("activity write failed")
		for _, f := range []*descriptionActivities{
			{readErr: sentinel}, {insertErr: sentinel},
			{found: true, latest: ActivityRecord{ActorID: &actor, Activity: domain.Activity{Field: &description}}, mergeErr: sentinel},
		} {
			if err := writeDescriptionActivity(context.Background(), f, s, change); err != sentinel {
				t.Fatalf("error = %v, want original %v", err, sentinel)
			}
			if f.readErr != nil && (len(f.inserted) != 0 || len(f.merged) != 0) {
				t.Fatal("read error wrote activity")
			}
		}
	})
}
