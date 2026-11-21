package postgresadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
)

// ProjectViews lists the project's undeleted views, the newest first, then
// by id descending; only ownedBy's when it is not nil (views.sql,
// app.ViewReader).
func (s *Store) ProjectViews(ctx context.Context, project uuid.UUID, ownedBy *uuid.UUID) ([]domain.View, error) {
	rows, err := s.queries(ctx).ListProjectViews(ctx, gen.ListProjectViewsParams{ProjectID: project, OwnedBy: ownedBy})
	if err != nil {
		return nil, fmt.Errorf("list the views of project %s: %w", project, err)
	}
	return listed(rows, func(r gen.ListProjectViewsRow) gen.GetViewRow { return gen.GetViewRow(r) })
}

// WorkspaceViews lists the workspace's own undeleted views, those without a
// project, the newest first, then by id descending; only ownedBy's when it
// is not nil (views.sql, app.ViewReader).
func (s *Store) WorkspaceViews(ctx context.Context, workspace uuid.UUID, ownedBy *uuid.UUID) ([]domain.View, error) {
	rows, err := s.queries(ctx).ListWorkspaceViews(ctx, gen.ListWorkspaceViewsParams{WorkspaceID: workspace, OwnedBy: ownedBy})
	if err != nil {
		return nil, fmt.Errorf("list the views of workspace %s: %w", workspace, err)
	}
	return listed(rows, func(r gen.ListWorkspaceViewsRow) gen.GetViewRow { return gen.GetViewRow(r) })
}

// View reads the undeleted view id without a lock; found is false when
// there is none (app.ViewReader).
func (s *Store) View(ctx context.Context, id uuid.UUID) (domain.View, bool, error) {
	r, err := s.queries(ctx).GetView(ctx, id)
	return found(r, err, "read", id)
}

// LockView reads the undeleted view id locked FOR UPDATE; found is false
// when there is none (app.ViewLocker).
func (s *Store) LockView(ctx context.Context, id uuid.UUID) (domain.View, bool, error) {
	r, err := s.queries(ctx).LockView(ctx, id)
	return found(gen.GetViewRow(r), err, "lock", id)
}

// ViewTargets reads the undeleted views among ids, of either kind and any
// workspace, by id, without a lock (views.sql, app.ViewTargets); a deleted
// view and an id of none are not among them.
func (s *Store) ViewTargets(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.View, error) {
	rows, err := s.queries(ctx).ViewTargets(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read %d views for favorites: %w", len(ids), err)
	}
	vs, err := listed(rows, func(r gen.ViewTargetsRow) gen.GetViewRow { return gen.GetViewRow(r) })
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]domain.View, len(vs))
	for _, v := range vs {
		out[v.ID] = v
	}
	return out, nil
}

// ShareView locks the undeleted view id FOR SHARE until the transaction ctx
// carries ends (views.sql, app.ViewTargets); a deleted view and an id of
// none take no lock.
func (s *Store) ShareView(ctx context.Context, id uuid.UUID) error {
	if err := s.queries(ctx).ShareView(ctx, id); err != nil {
		return fmt.Errorf("share view %s: %w", id, err)
	}
	return nil
}

// InsertView stores the new view v, made by by at its CreatedAt
// (app.ViewWriter).
func (s *Store) InsertView(ctx context.Context, v domain.View, by uuid.UUID) error {
	c, err := columnsOf(v.Content)
	if err != nil {
		return fmt.Errorf("create view %s: %w", v.ID, err)
	}
	err = s.queries(ctx).InsertView(ctx, gen.InsertViewParams{ID: v.ID, WorkspaceID: v.WorkspaceID, ProjectID: v.ProjectID, Name: v.Name,
		Description: v.Description, LogoProps: c.logo, Filters: c.filters, DisplayFilters: c.displayFilters, DisplayProperties: c.displayProperties,
		OwnedByID: v.OwnedByID, By: by, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt})
	if err != nil {
		return fmt.Errorf("create view %s: %w", v.ID, err)
	}
	return nil
}

// UpdateView replaces the content of the undeleted view v.ID with v's,
// last changed by by at v.UpdatedAt (app.ViewWriter).
func (s *Store) UpdateView(ctx context.Context, v domain.View, by uuid.UUID) error {
	c, err := columnsOf(v.Content)
	if err != nil {
		return fmt.Errorf("change view %s: %w", v.ID, err)
	}
	err = s.queries(ctx).UpdateView(ctx, gen.UpdateViewParams{ID: v.ID, Name: v.Name, Description: v.Description, LogoProps: c.logo,
		Filters: c.filters, DisplayFilters: c.displayFilters, DisplayProperties: c.displayProperties, UpdatedAt: v.UpdatedAt, By: by})
	if err != nil {
		return fmt.Errorf("change view %s: %w", v.ID, err)
	}
	return nil
}

// DeleteView soft-deletes the undeleted view id, deleted and last changed
// by by at at (app.ViewWriter).
func (s *Store) DeleteView(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error {
	if err := s.queries(ctx).DeleteView(ctx, gen.DeleteViewParams{ID: id, At: at, By: by}); err != nil {
		return fmt.Errorf("delete view %s: %w", id, err)
	}
	return nil
}

// listed are the views rows hold, as GetViewRow, in their order; never nil.
func listed[R any](rows []R, row func(R) gen.GetViewRow) ([]domain.View, error) {
	out := make([]domain.View, 0, len(rows))
	for _, r := range rows {
		v, err := viewOf(row(r))
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// found is the view a read of id gave, found when there was one; err is
// the read's.
func found(r gen.GetViewRow, err error, what string, id uuid.UUID) (domain.View, bool, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.View{}, false, nil
	case err != nil:
		return domain.View{}, false, fmt.Errorf("%s view %s: %w", what, id, err)
	}
	v, err := viewOf(r)
	if err != nil {
		return domain.View{}, false, err
	}
	return v, true, nil
}

// viewOf is the view a row of views.sql's reads holds, its four jsonb
// columns decoded.
func viewOf(r gen.GetViewRow) (domain.View, error) {
	v := domain.View{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, OwnedByID: r.OwnedByID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		Content: domain.Content{Name: r.Name, Description: r.Description}}
	for _, column := range []struct {
		name string
		data []byte
		into any
	}{
		{"logo_props", r.LogoProps, &v.LogoProps}, {"filters", r.Filters, &v.Filters},
		{"display_filters", r.DisplayFilters, &v.DisplayFilters}, {"display_properties", r.DisplayProperties, &v.DisplayProperties},
	} {
		if err := json.Unmarshal(column.data, column.into); err != nil {
			return domain.View{}, fmt.Errorf("read the %s of view %s: %w", column.name, r.ID, err)
		}
	}
	return v, nil
}

// contentColumns are a view's content as its four jsonb columns.
type contentColumns struct{ logo, filters, displayFilters, displayProperties []byte }

// columnsOf is c's jsonb columns, as the column CHECKs take them: a nil
// filter omitted, an empty one kept; the display filters with every key.
func columnsOf(c domain.Content) (contentColumns, error) {
	logo, logoErr := json.Marshal(c.LogoProps)
	filters, filtersErr := json.Marshal(c.Filters)
	displayFilters, displayErr := json.Marshal(c.DisplayFilters)
	displayProperties, propertiesErr := json.Marshal(c.DisplayProperties)
	if err := errors.Join(logoErr, filtersErr, displayErr, propertiesErr); err != nil {
		return contentColumns{}, err
	}
	return contentColumns{logo, filters, displayFilters, displayProperties}, nil
}
