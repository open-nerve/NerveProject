package bootstrap

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every wrapper delegates to the real port. The barrier is after the named
// directory read, or after real empty Rows and before Listing's fallback.
type listSnapshotProbe struct {
	pool             *pgxpool.Pool
	mu               sync.Mutex
	pid              int32
	steps            []string
	reads            []issueapp.ListRead
	stage            string
	entered, release chan struct{}
	once, released   sync.Once
}

func (p *listSnapshotProbe) after(ctx context.Context, step string) error {
	var isolation, readOnly string
	var pid int32
	if err := postgres.DB(ctx, p.pool).QueryRow(ctx, "SELECT pg_backend_pid(),current_setting('transaction_isolation'),current_setting('transaction_read_only')").Scan(&pid, &isolation, &readOnly); err != nil {
		return err
	}
	if isolation != "repeatable read" || readOnly != "on" {
		return fmt.Errorf("%s escaped read snapshot: %s/%s", step, isolation, readOnly)
	}
	p.mu.Lock()
	if p.pid != 0 && p.pid != pid {
		p.mu.Unlock()
		return fmt.Errorf("%s changed snapshot connection", step)
	}
	p.pid = pid
	p.steps = append(p.steps, step)
	p.mu.Unlock()
	if step == p.stage {
		p.once.Do(func() { close(p.entered) })
		select {
		case <-p.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (p *listSnapshotProbe) resume() { p.released.Do(func() { close(p.release) }) }

func listObserved[T any](ctx context.Context, p *listSnapshotProbe, step string, read func() (T, error)) (T, error) {
	v, err := read()
	if err == nil {
		err = p.after(ctx, step)
	}
	return v, err
}

type snapshotListProjects struct {
	issueProjects
	p *listSnapshotProbe
}

func (d snapshotListProjects) Find(ctx context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	wid, found, err := d.issueProjects.Find(ctx, id)
	if err == nil {
		err = d.p.after(ctx, "source")
	}
	return wid, found, err
}
func (d snapshotListProjects) Visible(ctx context.Context, wid, actor uuid.UUID) ([]issueapp.VisibleProject, error) {
	return listObserved(ctx, d.p, "visible", func() ([]issueapp.VisibleProject, error) { return d.issueProjects.Visible(ctx, wid, actor) })
}
func (d snapshotListProjects) ListStates(ctx context.Context, ids []uuid.UUID) ([]issueapp.StateRef, error) {
	return listObserved(ctx, d.p, "states", func() ([]issueapp.StateRef, error) { return d.issueProjects.ListStates(ctx, ids) })
}
func (d snapshotListProjects) ListLabels(ctx context.Context, ids []uuid.UUID) ([]issueapp.LabelRef, error) {
	return listObserved(ctx, d.p, "labels", func() ([]issueapp.LabelRef, error) { return d.issueProjects.ListLabels(ctx, ids) })
}

type snapshotListAuth struct {
	shared.Authorizer
	p *listSnapshotProbe
}

func (a snapshotListAuth) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, target shared.Target) (shared.Grant, error) {
	return listObserved(ctx, a.p, "auth", func() (shared.Grant, error) { return a.Authorizer.Authorize(ctx, actor, action, target) })
}

type snapshotListWorkspace struct {
	issueWorkspaces
	p *listSnapshotProbe
}

func (d snapshotListWorkspace) WorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error) {
	id, found, err := d.issueWorkspaces.WorkspaceBySlug(ctx, slug)
	if err == nil {
		err = d.p.after(ctx, "workspace")
	}
	return id, found, err
}

type snapshotListMembers struct {
	issueMembers
	p *listSnapshotProbe
}

func (d snapshotListMembers) MemberExists(ctx context.Context, wid, id uuid.UUID) (bool, error) {
	return listObserved(ctx, d.p, "member", func() (bool, error) { return d.issueMembers.MemberExists(ctx, wid, id) })
}

type snapshotListProfiles struct {
	issueProfiles
	p *listSnapshotProbe
}

func (d snapshotListProfiles) DisplayNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	return listObserved(ctx, d.p, "names", func() (map[uuid.UUID]string, error) { return d.issueProfiles.DisplayNames(ctx, ids) })
}

type snapshotListRows struct {
	*issuepg.Store
	p *listSnapshotProbe
}

func (s snapshotListRows) AssigneeIDs(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	return listObserved(ctx, s.p, "assignees", func() ([]uuid.UUID, error) { return s.Store.AssigneeIDs(ctx, ids) })
}
func (s snapshotListRows) GetIssues(ctx context.Context, ids []uuid.UUID) ([]domain.Issue, error) {
	return listObserved(ctx, s.p, "parents", func() ([]domain.Issue, error) { return s.Store.GetIssues(ctx, ids) })
}
func (s snapshotListRows) Flat(ctx context.Context, in issueapp.ListRead) ([]issueapp.ListRecord, error) {
	return s.read(ctx, in, s.Store.Flat)
}
func (s snapshotListRows) Groups(ctx context.Context, in issueapp.ListRead) ([]issueapp.ListRecord, error) {
	return s.read(ctx, in, s.Store.Groups)
}
func (s snapshotListRows) Subgroups(ctx context.Context, in issueapp.ListRead) ([]issueapp.ListRecord, error) {
	return s.read(ctx, in, s.Store.Subgroups)
}
func (s snapshotListRows) read(ctx context.Context, in issueapp.ListRead, fetch func(context.Context, issueapp.ListRead) ([]issueapp.ListRecord, error)) ([]issueapp.ListRecord, error) {
	s.p.mu.Lock()
	if len(s.p.reads) == 1 && s.p.stage == "empty" {
		want := s.p.reads[0]
		want.Query.Cursor, want.Query.Limit = nil, 1 // App limit; adapter asks SQL for two rows.
		if !reflect.DeepEqual(in, want) {
			s.p.mu.Unlock()
			return nil, fmt.Errorf("fallback changed original filters/directories/selection: %+v", in)
		}
	}
	s.p.reads = append(s.p.reads, in)
	s.p.mu.Unlock()
	rows, err := fetch(ctx, in)
	if err != nil {
		return nil, err
	}
	step := "rows"
	if in.Query.Cursor != nil && len(rows) == 0 {
		step = "empty"
	}
	return rows, s.p.after(ctx, step)
}

func snapshotListUseCases(w issueWorld, p *listSnapshotProbe) (*issueapp.ListIssues, *issueapp.ListMemberIssues) {
	projects, workspaces := project.Provide(w.pool), workspace.Provide(w.pool)
	dirs := snapshotListProjects{issueProjects{projects: projects.IssueProjects}, p}
	store := snapshotListRows{issuepg.New(w.pool), p}
	auth := snapshotListAuth{authorizerOn(w.pool), p}
	listing := issueapp.NewListing(issueapp.ListingDeps{Rows: store, States: dirs, Labels: dirs, Assignees: store,
		Profiles: snapshotListProfiles{issueProfiles{profiles: identity.Provide(w.pool).PublicProfiles}, p}, Issues: store})
	tx := postgres.NewTxManager(w.pool, 3*time.Second, postgres.ReadOnlySnapshot())
	return issueapp.NewListIssues(issueapp.ListIssuesDeps{ReadTx: tx, Locks: issueapp.NewLocks(dirs, store, auth, nil, nil), Listing: listing}),
		issueapp.NewListMemberIssues(issueapp.ListMemberIssuesDeps{ReadTx: tx, Authorizer: auth, Projects: dirs, Listing: listing,
			Workspaces: snapshotListWorkspace{issueWorkspaces{directory: workspaces.WorkspaceDirectory}, p},
			Members:    snapshotListMembers{issueMembers{members: workspaces.WorkspaceMemberDirectory}, p}})
}
