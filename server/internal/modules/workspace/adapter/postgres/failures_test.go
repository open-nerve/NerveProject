package postgresadapter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// A read that fails answers its error, never a plausible answer: not "not a
// member", which the Authorizer would turn into workspace.not_found; not
// "free", "none" or app.ErrNotFound. Each read runs on a cancelled context
// against a workspace alice administers, so that the right answer is none
// of the zero values.
func TestAFailedReadIsAnErrorNotAnAnswer(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	failed := func(err error) bool { return errors.Is(err, context.Canceled) }

	if role, ok, err := s.ActiveRole(cancelled, w.ID, alice); !failed(err) || ok || role != 0 {
		t.Errorf("ActiveRole() = %d, %v, %v; want context.Canceled, not a member", role, ok, err)
	}
	if taken, err := s.SlugTaken(cancelled, "acme"); !failed(err) || taken {
		t.Errorf("SlugTaken() = %v, %v; want context.Canceled", taken, err)
	}
	if list, err := s.ListWorkspaces(cancelled, alice); !failed(err) || list != nil {
		t.Errorf("ListWorkspaces() = %v, %v; want context.Canceled, no list", list, err)
	}
	if got, err := s.WorkspaceBySlug(cancelled, "acme"); !failed(err) || errors.Is(err, app.ErrNotFound) || !sameWorkspace(got, domain.Workspace{}) {
		t.Errorf("WorkspaceBySlug() = %+v, %v; want context.Canceled, not app.ErrNotFound", got, err)
	}
}

// A write that fails answers its error, never nil, which a use case would
// take for done, and never a row. Each write runs on a cancelled context
// against a workspace alice administers.
func TestAFailedWriteIsAnError(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	failed := func(err error) bool { return errors.Is(err, context.Canceled) }

	if got, err := s.UpdateWorkspace(cancelled, w.ID, domain.WorkspacePatch{Name: ptr("Renamed")}, alice, now); !failed(err) ||
		!sameWorkspace(got, domain.Workspace{}) {
		t.Errorf("UpdateWorkspace() = %+v, %v; want context.Canceled", got, err)
	}
}
