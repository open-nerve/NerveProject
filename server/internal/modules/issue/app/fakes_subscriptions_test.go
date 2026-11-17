package app_test

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// ShareIssue runs sharing, when set, as the lock waits, before it answers:
// what commits meanwhile; then logs the lock and answers the work item id
// as the fake then has it (read), as its row has it; found is false when
// it has none.
func (f *fakeIssues) ShareIssue(ctx context.Context, id uuid.UUID) (app.SharedIssue, bool, error) {
	if f.sharing != nil {
		f.sharing()
	}
	if err := f.log.add(ctx, "ShareIssue", id); err != nil {
		return app.SharedIssue{}, false, err
	}
	i, found := f.read(id)
	if !found {
		return app.SharedIssue{}, false, nil
	}
	return app.SharedIssue{ID: i.ID, WorkspaceID: i.WorkspaceID, ProjectID: i.ProjectID, CreatedByID: i.CreatedByID}, true, nil
}

// IsSubscribed answers whether account is among the fake's subscribers of
// its work item, of id f.issue's: none of an id the fake has not. Told no
// subscribers (nil), it answers none and logs nothing, as before M7 (the
// call lists of M4's tests); told any, an empty list too, it logs the read
// first.
func (f *fakeIssues) IsSubscribed(ctx context.Context, id, account uuid.UUID) (bool, error) {
	if f.subscribers == nil {
		return false, nil
	}
	if err := f.log.add(ctx, "IsSubscribed", id, account); err != nil {
		return false, err
	}
	return f.issue != nil && id == f.issue.ID && slices.Contains(f.subscribers, account), nil
}

// fakeSubscriptions is the subscribers' store of subscribeIssue and
// unsubscribeIssue: each write logged with the accounts it names, its rows
// kept in order, inserted and removed; and the accounts that subscribe,
// subscribed, which an inserted row of one adds once and a removal takes
// away, as the store's statements do.
type fakeSubscriptions struct {
	log        *callLog
	inserted   []app.LinkRows
	removed    []app.SubscriberRemoval
	subscribed map[uuid.UUID]bool
}

func (f *fakeSubscriptions) InsertSubscribers(ctx context.Context, r app.LinkRows) error {
	if err := f.log.add(ctx, "InsertSubscribers", r.Targets); err != nil {
		return err
	}
	f.inserted = append(f.inserted, r)
	for _, a := range r.Targets {
		f.subscribed[a] = true
	}
	return nil
}

func (f *fakeSubscriptions) RemoveSubscriber(ctx context.Context, r app.SubscriberRemoval) error {
	if err := f.log.add(ctx, "RemoveSubscriber", r.SubscriberID); err != nil {
		return err
	}
	f.removed = append(f.removed, r)
	delete(f.subscribed, r.SubscriberID)
	return nil
}
