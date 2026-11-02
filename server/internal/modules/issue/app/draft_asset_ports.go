package app

import "context"

type DraftAssets interface {
	DeleteDraftAssets(context.Context, DraftDeletion) error
}
