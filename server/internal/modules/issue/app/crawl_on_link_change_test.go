package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type linkCrawlCall func(context.Context, uuid.UUID, string) error

func (f linkCrawlCall) Enqueue(ctx context.Context, id uuid.UUID, url string) error {
	return f(ctx, id, url)
}

// Wrong event identity, field selection, validation or error propagation must
// fail before a malformed event can enqueue work or silently lose it.
func TestCrawlOnLinkChange(t *testing.T) {
	old, url, title := "http://169.254.169.254/old", "http://169.254.169.254/New?x=%2F", "Named"
	for _, tc := range []struct {
		name, kind string
		changes    []shared.Change
		jobs       int
		invalid    bool
	}{
		{"created", "issue.link_created", []shared.Change{{Field: "title", New: &title}, {Field: "url", Old: &old, New: &url}}, 1, false},
		{"updated", "issue.link_updated", []shared.Change{{Field: "url", Old: &old, New: &url}, {Field: "title", New: &title}}, 1, false},
		{"title only", "issue.link_updated", []shared.Change{{Field: "title", New: &title}}, 0, false},
		{"updated empty", "issue.link_updated", nil, 0, false},
		{"created missing", "issue.link_created", []shared.Change{{Field: "title", New: &title}}, 0, true},
		{"created duplicate", "issue.link_created", []shared.Change{{Field: "url", New: &url}, {Field: "url", New: &url}}, 0, true},
		{"updated duplicate", "issue.link_updated", []shared.Change{{Field: "url", New: &url}, {Field: "url", New: &old}}, 0, true},
		{"created nil", "issue.link_created", []shared.Change{{Field: "url"}}, 0, true},
		{"updated nil", "issue.link_updated", []shared.Change{{Field: "url", Old: &old}}, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			id := uuid.NewV7()
			n := 0
			s := NewCrawlOnLinkChange(linkCrawlCall(func(got context.Context, linkID uuid.UUID, gotURL string) error {
				n++
				if got != ctx || linkID != id || gotURL != url {
					t.Errorf("enqueue lost caller context/entity/new URL: %v %s %q", got == ctx, linkID, gotURL)
				}
				return nil
			}))
			if s.Name != "crawl_on_link_change" || !reflect.DeepEqual(s.Types, []string{"issue.link_created", "issue.link_updated"}) {
				t.Fatalf("subscriber=%+v", s)
			}
			err := s.Handle(ctx, shared.Event{ID: uuid.NewV7(), Type: tc.kind, Entity: shared.EntityRef{Kind: "issue_link", ID: id}, Changes: tc.changes})
			if (err != nil) != tc.invalid || n != tc.jobs {
				t.Fatalf("error=%v jobs=%d; want invalid=%t jobs=%d", err, n, tc.invalid, tc.jobs)
			}
		})
	}
	t.Run("enqueue error", func(t *testing.T) {
		failure := errors.New("insert failed")
		s := NewCrawlOnLinkChange(linkCrawlCall(func(context.Context, uuid.UUID, string) error { return failure }))
		err := s.Handle(t.Context(), shared.Event{Type: "issue.link_created", Entity: shared.EntityRef{ID: uuid.NewV7()}, Changes: []shared.Change{{Field: "url", New: &url}}})
		if !errors.Is(err, failure) {
			t.Fatalf("enqueue error=%v; want original failure", err)
		}
	})
}
