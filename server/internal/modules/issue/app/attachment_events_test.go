package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type attachmentChangeCall func(context.Context, ActivityRow) error

func (f attachmentChangeCall) RecordAttachmentChange(ctx context.Context, row ActivityRow) error {
	return f(ctx, row)
}

func attachmentEventFixture() shared.Event {
	by, id := uuid.NewV7(), uuid.NewV7()
	return shared.Event{ID: uuid.NewV7(), Type: "asset.attachment_added", OccurredAt: time.Date(2026, 11, 3, 12, 0, 0, 123000000, time.UTC), WorkspaceID: uuid.NewV7(), ProjectID: uuid.NewV7(), ActorID: &by, Entity: shared.EntityRef{Kind: "asset", ID: uuid.NewV7()}, Changes: []shared.Change{{Field: "issue", NewID: &id}}}
}

func TestAttachmentEventsPayload(t *testing.T) {
	for _, added := range []bool{true, false} {
		e := attachmentEventFixture()
		id := *e.Changes[0].NewID
		if !added {
			e.Type = "asset.attachment_deleted"
			e.Changes[0].OldID, e.Changes[0].NewID = e.Changes[0].NewID, nil
		}
		failure := errors.New("activity write rejected")
		calls := 0
		ctx := t.Context()
		s := NewAttachmentChanges(attachmentChangeCall(func(got context.Context, row ActivityRow) error {
			calls++
			if got != ctx || row.ID == uuid.Nil() || row.WorkspaceID != e.WorkspaceID || row.ProjectID != e.ProjectID || row.IssueID != id || row.ActorID == nil || *row.ActorID != *e.ActorID || !row.At.Equal(e.OccurredAt) {
				t.Errorf("lost context/stamp/host: %+v", row)
			}
			a := row.Activity
			if a.Field == nil || *a.Field != "attachment" || a.OldValue != nil || a.NewValue != nil {
				t.Errorf("private or wrong activity: %+v", a)
			}
			if added && (a.Verb != domain.VerbCreated || a.OldIdentifier != nil || a.NewIdentifier == nil || *a.NewIdentifier != e.Entity.ID) {
				t.Errorf("wrong added identity: %+v", a)
			}
			if !added && (a.Verb != domain.VerbDeleted || a.NewIdentifier != nil || a.OldIdentifier == nil || *a.OldIdentifier != e.Entity.ID) {
				t.Errorf("wrong deleted identity: %+v", a)
			}
			return failure
		}))
		if err := s.Handle(ctx, e); !errors.Is(err, failure) || calls != 1 {
			t.Fatalf("delivery=%d %v", calls, err)
		}
	}
}

func TestAttachmentEventsRejectMalformed(t *testing.T) {
	for _, tc := range []struct {
		name  string
		alter func(*shared.Event)
	}{
		{"unknown type", func(e *shared.Event) { e.Type = "asset.other" }},
		{"missing event", func(e *shared.Event) { e.ID = uuid.Nil() }},
		{"wrong entity", func(e *shared.Event) { e.Entity.Kind = "issue" }},
		{"zero entity", func(e *shared.Event) { e.Entity.ID = uuid.Nil() }},
		{"zero workspace", func(e *shared.Event) { e.WorkspaceID = uuid.Nil() }},
		{"zero project", func(e *shared.Event) { e.ProjectID = uuid.Nil() }},
		{"missing actor", func(e *shared.Event) { e.ActorID = nil }},
		{"zero actor", func(e *shared.Event) { id := uuid.Nil(); e.ActorID = &id }},
		{"zero time", func(e *shared.Event) { e.OccurredAt = time.Time{} }},
		{"missing change", func(e *shared.Event) { e.Changes = nil }},
		{"extra change", func(e *shared.Event) { e.Changes = append(e.Changes, e.Changes[0]) }},
		{"wrong field", func(e *shared.Event) { e.Changes[0].Field = "filename" }},
		{"private old value", func(e *shared.Event) { s := "private"; e.Changes[0].Old = &s }},
		{"private new value", func(e *shared.Event) { s := "private"; e.Changes[0].New = &s }},
		{"added missing issue", func(e *shared.Event) { e.Changes[0].NewID = nil }},
		{"added zero issue", func(e *shared.Event) { id := uuid.Nil(); e.Changes[0].NewID = &id }},
		{"added old slot", func(e *shared.Event) { e.Changes[0].OldID = e.Changes[0].NewID }},
		{"deleted new slot", func(e *shared.Event) { e.Type = "asset.attachment_deleted" }},
		{"deleted missing issue", func(e *shared.Event) { e.Type = "asset.attachment_deleted"; e.Changes[0].NewID = nil }},
		{"deleted zero issue", func(e *shared.Event) {
			e.Type = "asset.attachment_deleted"
			id := uuid.Nil()
			e.Changes[0].NewID = nil
			e.Changes[0].OldID = &id
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := attachmentEventFixture()
			tc.alter(&e)
			called := false
			s := NewAttachmentChanges(attachmentChangeCall(func(context.Context, ActivityRow) error { called = true; return nil }))
			if err := s.Handle(t.Context(), e); err == nil || called {
				t.Fatalf("malformed event reached storage: %v %t", err, called)
			}
		})
	}
	if !reflect.DeepEqual(NewAttachmentChanges(nil).Types, []string{"asset.attachment_added", "asset.attachment_deleted"}) {
		t.Fatal("subscriber omitted an event")
	}
}
