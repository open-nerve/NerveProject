package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeReader holds projects by id, logs each read and fails with err.
type fakeReader struct {
	log      *callLog
	projects map[uuid.UUID]domain.Project
	err      error
}

func (f *fakeReader) GetProject(ctx context.Context, id, userID uuid.UUID) (domain.Project, bool, error) {
	f.log.add(ctx, "GetProject %s for %s", id, userID)
	p, found := f.projects[id]
	return p, found && f.err == nil, f.err
}

// newGet is GetProject over fakes sharing one log, and web, acme's
// project, which alice, acme's member, sees, and dave does not.
func newGet() (*app.GetProject, *fakeReader, *fakeAuthorizer, domain.Project) {
	log := &callLog{}
	web := domain.Project{ID: uuid.NewV7(), WorkspaceID: acme.ID, Name: "Web", MemberRole: ptr(shared.RoleAdmin)}
	reader := &fakeReader{log: log, projects: map[uuid.UUID]domain.Project{web.ID: web}}
	auth := &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{{alice, acme.ID}: {WorkspaceRole: shared.RoleMember}}}
	return app.NewGetProject(reader, auth), reader, auth, web
}

// GetProject reads the project as the caller sees it, outside any
// transaction, then decides project.read on it, in its workspace: the
// answer is the project as read.
func TestGetProject(t *testing.T) {
	uc, reader, _, web := newGet()

	got, err := uc.Execute(as(alice), web.ID)

	want := []string{fmt.Sprintf("GetProject %s for %s outside tx", web.ID, alice),
		fmt.Sprintf("Authorize %s project.read on %s/%s outside tx", alice, acme.ID, web.ID)}
	if err != nil || got.ID != web.ID || got.MemberRole == nil || *got.MemberRole != shared.RoleAdmin || !slices.Equal(reader.log.calls, want) {
		t.Errorf("Execute() = %+v, %v, calls %q; want the project as read, calls %q", got, err, reader.log.calls, want)
	}
}

// A project not there and one the caller does not see answer the same
// project.not_found; the Authorizer's other refusals and every failure
// come back as themselves; without a caller nothing is read.
func TestGetProjectRefuses(t *testing.T) {
	failure := errors.New("connection reset")
	tests := []struct {
		name     string
		ctx      context.Context
		another  bool  // asks for another id than web's
		readErr  error // the read's failure
		alicesTo error // the Authorizer's answer to alice
		want     error
		calls    int
	}{
		{"no project", as(alice), true, nil, nil, domain.ErrNotFound, 1},
		{"not seen", as(dave), false, nil, nil, domain.ErrNotFound, 2},
		{"forbidden", as(alice), false, nil, shared.Forbidden(), shared.Forbidden(), 2},
		{"the decision failing", as(alice), false, nil, failure, failure, 2},
		{"the read failing", as(alice), false, failure, nil, failure, 1},
		{"no caller", context.Background(), false, nil, nil, shared.Unauthenticated(), 0},
	}
	for _, tt := range tests {
		uc, reader, auth, web := newGet()
		reader.err = tt.readErr
		if tt.alicesTo != nil {
			auth.errs = map[grantKey]error{{alice, acme.ID}: tt.alicesTo}
		}
		id := web.ID
		if tt.another {
			id = uuid.NewV7()
		}
		got, err := uc.Execute(tt.ctx, id)
		if !errors.Is(err, tt.want) || got.ID != (uuid.UUID{}) || len(reader.log.calls) != tt.calls {
			t.Errorf("%s: Execute() = %+v, %v, calls %q; want %v after %d calls", tt.name, got, err, reader.log.calls, tt.want, tt.calls)
		}
	}
}
