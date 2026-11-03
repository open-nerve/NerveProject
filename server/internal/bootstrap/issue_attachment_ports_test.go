package bootstrap

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
)

func attachmentCounterOn(pool *pgxpool.Pool) issueAttachments {
	reads := asset.NewAttachmentReads(asset.Provide(pool))
	return issueAttachments{reader: reads, counter: reads, catalog: reads}
}
