package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type attachmentAccessFunc func(context.Context, shared.Actor, app.AssetRequest) (app.AssetHost, error)

func (f attachmentAccessFunc) Inspect(ctx context.Context, a shared.Actor, q app.AssetRequest) (app.AssetHost, error) {
	return f(ctx, a, q)
}

type attachmentRowsFunc func(context.Context, app.AttachmentRead) ([]app.Attachment, error)

func (f attachmentRowsFunc) ListAttachments(ctx context.Context, q app.AttachmentRead) ([]app.Attachment, error) {
	return f(ctx, q)
}

func TestAttachmentListsOrderingAndErrors(t *testing.T) {
	for _, kind := range []string{"issue", "comment", "draft"} {
		t.Run(kind, func(t *testing.T) {
			log := &callLog{}
			webSix := uuid.NewV7()
			var inside context.Context
			h := app.AssetHost{WorkspaceID: acme, ProjectID: web, IssueID: webSix}
			size := 1
			ids := []uuid.UUID{uuid.NewV7(), uuid.NewV7()}
			slices.SortFunc(ids, uuid.UUID.Compare)
			calls := 0
			access := attachmentAccessFunc(func(ctx context.Context, a shared.Actor, q app.AssetRequest) (app.AssetHost, error) {
				if ctx.Value(inTxKey{}) != true || a.UserID != alice || q.ID != webSix || q.Intent != app.AssetRead {
					t.Fatalf("owner lookup escaped snapshot/actor: %+v", q)
				}
				expected := map[string]app.AssetHostKind{"issue": app.AssetIssue, "comment": app.AssetComment, "draft": app.AssetDraft}[kind]
				if q.Kind != expected {
					t.Fatal("wrong owner kind", q)
				}
				inside = ctx
				log.calls = append(log.calls, "owner")
				return h, nil
			})
			reader := attachmentRowsFunc(func(ctx context.Context, q app.AttachmentRead) ([]app.Attachment, error) {
				calls++
				if ctx != inside || q.Host != app.ContentHost(h) || q.Limit != 2 || q.After != uuid.Nil() {
					t.Fatal("read lost authorized host/context or lookahead", q)
				}
				log.calls = append(log.calls, "assets")
				return []app.Attachment{{ID: ids[0]}, {ID: ids[1]}}, nil
			})
			u := app.NewGetAttachments(kind, app.GetAttachmentsDeps{Access: access, Rows: reader, ReadTx: fakeTx{log}})
			p, err := u.Execute(as(alice), webSix, &size, nil)
			if err != nil || len(p.Data) != 1 || p.Data[0].ID != ids[0] || calls != 1 || !slices.Equal(log.calls, []string{"Begin", "owner", "assets"}) {
				t.Fatal(p, err, log.calls)
			}
			after, err := domain.DecodeAttachmentCursor(p.NextCursor, kind, webSix)
			if err != nil || after != ids[0] {
				t.Fatal(after, err)
			}
			missing := errors.New("hidden owner")
			refused := app.NewGetAttachments(kind, app.GetAttachmentsDeps{Access: attachmentAccessFunc(func(context.Context, shared.Actor, app.AssetRequest) (app.AssetHost, error) {
				return app.AssetHost{}, missing
			}), Rows: reader, ReadTx: fakeTx{log}})
			if _, err := refused.Execute(as(alice), webSix, nil, nil); !errors.Is(err, missing) || calls != 1 {
				t.Fatal("hidden owner reached asset port", err, calls)
			}
			fault := errors.New("attachment query failed")
			for _, mode := range []string{"database", "duplicate", "reverse", "over limit", "zero"} {
				failing := attachmentRowsFunc(func(context.Context, app.AttachmentRead) ([]app.Attachment, error) {
					switch mode {
					case "database":
						return nil, fault
					case "duplicate":
						return []app.Attachment{{ID: ids[0]}, {ID: ids[0]}}, nil
					case "reverse":
						return []app.Attachment{{ID: ids[1]}, {ID: ids[0]}}, nil
					case "over limit":
						return []app.Attachment{{ID: ids[0]}, {ID: ids[1]}, {ID: uuid.NewV7()}}, nil
					default:
						return []app.Attachment{{}}, nil
					}
				})
				u := app.NewGetAttachments(kind, app.GetAttachmentsDeps{Access: access, Rows: failing, ReadTx: fakeTx{log}})
				if p, err := u.Execute(as(alice), webSix, &size, nil); err == nil || len(p.Data) != 0 || (mode == "database" && !errors.Is(err, fault)) {
					t.Fatal(mode, p, err)
				}
			}
			log.calls = nil
			cursor := "invalid"
			if _, err := u.Execute(as(alice), webSix, nil, &cursor); err == nil || len(log.calls) != 0 {
				t.Fatal("invalid cursor reached snapshot")
			}
			if _, err := u.Execute(t.Context(), webSix, nil, nil); err == nil || len(log.calls) != 0 {
				t.Fatal("anonymous reached snapshot")
			}
			for _, limit := range []int{0, 101} {
				if _, err := u.Execute(as(alice), webSix, &limit, nil); err == nil || len(log.calls) != 0 {
					t.Fatal("invalid limit reached snapshot")
				}
			}
		})
	}
}
