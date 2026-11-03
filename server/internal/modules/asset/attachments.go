package asset

import "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"

type Attachment = app.Attachment
type AttachmentRead = app.AttachmentRead
type AttachmentCount = app.AttachmentCount
type AttachmentReader = app.AttachmentReader
type AttachmentCounter = app.AttachmentCounter
type AttachmentCountCatalog = app.AttachmentCountCatalog

func NewAttachmentReads(p Provided) *app.AttachmentReads { return app.NewAttachmentReads(p.store) }
