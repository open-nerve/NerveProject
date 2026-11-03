package bootstrap

import (
	"context"
	"testing"
	"time"
	"uuid"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestAttachmentEventsRequireTransactionAndMatchingHost(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	file := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", attachmentScope("issue", item), "x.txt", "text/plain", []byte("abc"))
	s := issueapp.NewAttachmentChanges(issuepg.New(w.pool))
	by := w.ids["bob"]
	for _, name := range []string{"without transaction", "workspace", "project", "issue"} {
		t.Run(name, func(t *testing.T) {
			e := shared.Event{ID: uuid.NewV7(), Type: "asset.attachment_added", WorkspaceID: w.acme, ProjectID: w.web, ActorID: &by, OccurredAt: time.Now().UTC(), Entity: shared.EntityRef{Kind: "asset", ID: file.Asset.ID}, Changes: []shared.Change{{Field: "issue", NewID: &item}}}
			switch name {
			case "workspace":
				e.WorkspaceID = uuid.NewV7()
			case "project":
				e.ProjectID = uuid.NewV7()
			case "issue":
				other := uuid.NewV7()
				e.Changes[0].NewID = &other
			}
			var err error
			if name == "without transaction" {
				err = s.Handle(t.Context(), e)
			} else {
				err = postgres.NewTxManager(w.pool, time.Second).WithinTx(t.Context(), func(ctx context.Context) error {
					tx, _ := postgres.Tx(ctx)
					if _, err := tx.Exec(ctx, "SELECT id FROM projects WHERE id=$1 FOR NO KEY UPDATE", w.web); err != nil {
						return err
					}
					return s.Handle(ctx, e)
				})
			}
			if err == nil || attachmentActivityCount(t, w, item) != 1 {
				t.Fatal("invalid consumer context/host wrote an activity", err)
			}
		})
	}
}
