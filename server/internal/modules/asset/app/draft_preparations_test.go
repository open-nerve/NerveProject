package app

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type preparationHostFunc func(context.Context, shared.Actor, HostRequest) (Host, error)

func (f preparationHostFunc) Inspect(ctx context.Context, actor shared.Actor, q HostRequest) (Host, error) {
	return f(ctx, actor, q)
}

type preparationRowsFunc func(context.Context, DraftPreparationQuery) ([]domain.Asset, error)

func (f preparationRowsFunc) ListDraftPreparations(ctx context.Context, q DraftPreparationQuery) ([]domain.Asset, error) {
	return f(ctx, q)
}

type preparationTxFunc func(context.Context, func(context.Context) error) error

func (f preparationTxFunc) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return f(ctx, fn)
}

func TestDraftPreparationsPage(t *testing.T) {
	actor := shared.Actor{UserID: attachmentReadID(1), SessionID: attachmentReadID(2)}
	draft := attachmentReadID(3)
	host := Host{Scope: domain.Scope{Kind: domain.ScopeDraft, ID: draft}, WorkspaceID: attachmentReadID(4), DraftID: draft}
	for _, count := range []int{0, 1, 50, 51, 101} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			ctx := shared.WithActor(t.Context(), actor)
			snapshot, cancel := context.WithCancel(ctx)
			defer cancel()
			size := 50
			var limit *int
			if count == 101 {
				size = 100
				limit = &size
			}
			inspected := false
			deps := ListDraftPreparationsDeps{
				ReadTx: preparationTxFunc(func(got context.Context, fn func(context.Context) error) error {
					if got != ctx {
						t.Fatal("lost actor context")
					}
					return fn(snapshot)
				}),
				Hosts: preparationHostFunc(func(got context.Context, who shared.Actor, q HostRequest) (Host, error) {
					if got != snapshot || who != actor || q.Scope != host.Scope || q.Intent != ReadAsset {
						t.Fatalf("wrong authorization: %+v %+v", who, q)
					}
					inspected = true
					return host, nil
				}),
				Rows: preparationRowsFunc(func(got context.Context, q DraftPreparationQuery) ([]domain.Asset, error) {
					if got != snapshot || !inspected || q.ActorID != actor.UserID || q.Host != host || q.After != uuid.Nil() || q.Limit != size+1 {
						t.Fatalf("query lost authorized snapshot/bounds: %+v", q)
					}
					rows := make([]domain.Asset, count)
					for i := range rows {
						rows[i] = domain.Asset{ID: attachmentReadID(10 + i), Name: fmt.Sprint(i)}
					}
					return rows, nil
				}),
			}
			page, err := NewListDraftPreparations(deps).Execute(ctx, draft, limit, nil)
			want := min(count, size)
			if err != nil || page.Data == nil || len(page.Data) != want {
				t.Fatalf("page=%+v err=%v want=%d", page, err, want)
			}
			if (page.NextCursor != nil) != (count > size) {
				t.Fatalf("wrong lookahead: %+v", page)
			}
			for i, row := range page.Data {
				if row.ID != attachmentReadID(10+i) || row.Name != fmt.Sprint(i) {
					t.Fatal("data changed", row)
				}
			}
			if page.NextCursor != nil {
				position, err := domain.DecodeDraftPreparationCursor(draft, page.NextCursor)
				if err != nil || position != attachmentReadID(10+size-1) {
					t.Fatal("cursor used lookahead instead of last visible row", position, err)
				}
				deps.Rows = preparationRowsFunc(func(_ context.Context, q DraftPreparationQuery) ([]domain.Asset, error) {
					if q.After != position {
						t.Fatal("cursor position not forwarded", q)
					}
					return []domain.Asset{{ID: attachmentReadID(10 + size)}}, nil
				})
				next, err := NewListDraftPreparations(deps).Execute(ctx, draft, limit, page.NextCursor)
				if err != nil || len(next.Data) != 1 || next.NextCursor != nil {
					t.Fatalf("next=%+v err=%v", next, err)
				}
			}
		})
	}
}

func TestDraftPreparationsNilDraft(t *testing.T) {
	actor := shared.Actor{UserID: attachmentReadID(1)}
	for _, tc := range []struct {
		name string
		ctx  context.Context
		want error
	}{
		{"authenticated", shared.WithActor(t.Context(), actor), domain.ErrAssetNotFound},
		{"unauthenticated", t.Context(), shared.Unauthenticated()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := NewListDraftPreparations(ListDraftPreparationsDeps{ReadTx: preparationTxFunc(func(context.Context, func(context.Context) error) error {
				t.Fatal("nil draft opened a transaction before rejection")
				return nil
			})})
			page, err := u.Execute(tc.ctx, uuid.Nil(), nil, nil)
			if !errors.Is(err, tc.want) || len(page.Data) != 0 || page.NextCursor != nil {
				t.Fatalf("page=%+v err=%v want=%v", page, err, tc.want)
			}
		})
	}
}

func TestDraftPreparationsRejectsBeforeReading(t *testing.T) {
	actor := shared.Actor{UserID: attachmentReadID(1)}
	draft := attachmentReadID(3)
	ctx := shared.WithActor(t.Context(), actor)
	bad := "bad"
	cursor, err := domain.EncodeDraftPreparationCursor(attachmentReadID(8), attachmentReadID(9))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		ctx    context.Context
		limit  *int
		cursor *string
		want   error
	}{
		{"no actor", t.Context(), nil, nil, shared.Unauthenticated()},
		{"zero limit", ctx, new(int), nil, shared.Invalid()},
		{"large limit", ctx, new(101), nil, shared.Invalid()},
		{"malformed cursor", ctx, nil, &bad, shared.InvalidCursor()},
		{"other draft cursor", ctx, nil, &cursor, shared.InvalidCursor()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := NewListDraftPreparations(ListDraftPreparationsDeps{ReadTx: preparationTxFunc(func(context.Context, func(context.Context) error) error {
				t.Fatal("invalid input opened transaction")
				return nil
			})})
			page, err := u.Execute(tc.ctx, draft, tc.limit, tc.cursor)
			if !errors.Is(err, tc.want) || len(page.Data) != 0 || page.NextCursor != nil {
				t.Fatalf("page=%+v err=%v want=%v", page, err, tc.want)
			}
		})
	}
}

func TestDraftPreparationsFailures(t *testing.T) {
	actor := shared.Actor{UserID: attachmentReadID(1)}
	draft := attachmentReadID(3)
	ctx := shared.WithActor(t.Context(), actor)
	fault := errors.New("read storage unavailable")
	for _, tc := range []struct {
		name string
		err  error
		want error
	}{
		{"forbidden", shared.Forbidden(), domain.ErrAssetNotFound},
		{"wrapped forbidden", fmt.Errorf("owner: %w", shared.Forbidden()), domain.ErrAssetNotFound},
		{"not found", domain.ErrAssetNotFound, domain.ErrAssetNotFound},
		{"unauthorized", shared.Unauthenticated(), shared.Unauthenticated()},
		{"canceled", context.Canceled, context.Canceled},
		{"database", fault, fault},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := NewListDraftPreparations(ListDraftPreparationsDeps{
				ReadTx: preparationTxFunc(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }),
				Hosts:  preparationHostFunc(func(context.Context, shared.Actor, HostRequest) (Host, error) { return Host{}, tc.err }),
				Rows: preparationRowsFunc(func(context.Context, DraftPreparationQuery) ([]domain.Asset, error) {
					t.Fatal("refused owner exposed rows")
					return nil, nil
				}),
			})
			page, err := u.Execute(ctx, draft, nil, nil)
			if !errors.Is(err, tc.want) || len(page.Data) != 0 {
				t.Fatalf("page=%+v err=%v want=%v", page, err, tc.want)
			}
		})
	}
	for _, at := range []string{"rows", "commit"} {
		t.Run(at, func(t *testing.T) {
			u := NewListDraftPreparations(ListDraftPreparationsDeps{
				ReadTx: preparationTxFunc(func(ctx context.Context, fn func(context.Context) error) error {
					if err := fn(ctx); err != nil {
						return err
					}
					return fault
				}),
				Hosts: preparationHostFunc(func(context.Context, shared.Actor, HostRequest) (Host, error) {
					return Host{Scope: domain.Scope{Kind: domain.ScopeDraft, ID: draft}, WorkspaceID: attachmentReadID(4), DraftID: draft}, nil
				}),
				Rows: preparationRowsFunc(func(context.Context, DraftPreparationQuery) ([]domain.Asset, error) {
					if at == "rows" {
						return nil, fault
					}
					return []domain.Asset{{ID: attachmentReadID(10)}}, nil
				}),
			})
			page, err := u.Execute(ctx, draft, nil, nil)
			if !errors.Is(err, fault) || len(page.Data) != 0 || page.NextCursor != nil {
				t.Fatalf("partial success: %+v %v", page, err)
			}
		})
	}
}

func TestDraftPreparationsHostIdentity(t *testing.T) {
	draft := attachmentReadID(3)
	ctx := shared.WithActor(t.Context(), shared.Actor{UserID: attachmentReadID(1)})
	for _, tc := range []struct {
		name string
		host Host
	}{
		{"scope", Host{Scope: domain.Scope{Kind: domain.ScopeIssue, ID: draft}, DraftID: draft}},
		{"scope id", Host{Scope: domain.Scope{Kind: domain.ScopeDraft, ID: attachmentReadID(4)}, DraftID: draft}},
		{"draft id", Host{Scope: domain.Scope{Kind: domain.ScopeDraft, ID: draft}, DraftID: attachmentReadID(4)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := NewListDraftPreparations(ListDraftPreparationsDeps{
				ReadTx: preparationTxFunc(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }),
				Hosts:  preparationHostFunc(func(context.Context, shared.Actor, HostRequest) (Host, error) { return tc.host, nil }),
				Rows: preparationRowsFunc(func(context.Context, DraftPreparationQuery) ([]domain.Asset, error) {
					t.Fatal("changed owner identity reached the asset query")
					return nil, nil
				}),
			})
			if _, err := u.Execute(ctx, draft, nil, nil); !errors.Is(err, domain.ErrAssetScopeChanged) {
				t.Fatalf("mismatched owner: %v", err)
			}
		})
	}
}
