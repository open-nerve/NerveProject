package bootstrap

import (
	"context"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

type identityPurgeRows interface {
	Page(context.Context, time.Time, int, identity.PurgeCursor) (identity.PurgeProgress, error)
}
type identityPurge struct{ rows identityPurgeRows }

func forIdentityPurge(rows identityPurgeRows) Purger { return identityPurge{rows} }
func (p identityPurge) Page(ctx context.Context, cutoff time.Time, batch int, cursor purgePageCursor) (purgePageProgress, error) {
	r, err := p.rows.Page(ctx, cutoff, batch, identity.PurgeCursor{Table: cursor.Table, After: cursor.After})
	return purgePageProgress{Cursor: purgePageCursor{Table: r.Cursor.Table, After: r.Cursor.After}, Scanned: r.Scanned, Deleted: r.Deleted, Blocked: r.Blocked, Done: r.Done}, err
}

type issuePurgeRows interface {
	Page(context.Context, time.Time, int, issue.PurgeCursor) (issue.PurgeProgress, error)
}
type issuePurge struct{ rows issuePurgeRows }

func forIssuePurge(rows issuePurgeRows) Purger { return issuePurge{rows} }
func (p issuePurge) Page(ctx context.Context, cutoff time.Time, batch int, cursor purgePageCursor) (purgePageProgress, error) {
	r, err := p.rows.Page(ctx, cutoff, batch, issue.PurgeCursor{Table: cursor.Table, After: cursor.After})
	return purgePageProgress{Cursor: purgePageCursor{Table: r.Cursor.Table, After: r.Cursor.After}, Scanned: r.Scanned, Deleted: r.Deleted, Blocked: r.Blocked, Done: r.Done}, err
}

type projectPurgeRows interface {
	Page(context.Context, time.Time, int, project.PurgeCursor) (project.PurgeProgress, error)
}
type projectPurge struct{ rows projectPurgeRows }

func forProjectPurge(rows projectPurgeRows) Purger { return projectPurge{rows} }
func (p projectPurge) Page(ctx context.Context, cutoff time.Time, batch int, cursor purgePageCursor) (purgePageProgress, error) {
	r, err := p.rows.Page(ctx, cutoff, batch, project.PurgeCursor{Table: cursor.Table, After: cursor.After})
	return purgePageProgress{Cursor: purgePageCursor{Table: r.Cursor.Table, After: r.Cursor.After}, Scanned: r.Scanned, Deleted: r.Deleted, Blocked: r.Blocked, Done: r.Done}, err
}

type workspacePurgeRows interface {
	Page(context.Context, time.Time, int, workspace.PurgeCursor) (workspace.PurgeProgress, error)
}
type workspacePurge struct{ rows workspacePurgeRows }

func forWorkspacePurge(rows workspacePurgeRows) Purger { return workspacePurge{rows} }
func (p workspacePurge) Page(ctx context.Context, cutoff time.Time, batch int, cursor purgePageCursor) (purgePageProgress, error) {
	r, err := p.rows.Page(ctx, cutoff, batch, workspace.PurgeCursor{Table: cursor.Table, After: cursor.After})
	return purgePageProgress{Cursor: purgePageCursor{Table: r.Cursor.Table, After: r.Cursor.After}, Scanned: r.Scanned, Deleted: r.Deleted, Blocked: r.Blocked, Done: r.Done}, err
}
