package app_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeCallerLock logs each lock with the caller's credential and the time,
// and answers err.
type fakeCallerLock struct {
	log *callLog
	err error
}

func (f *fakeCallerLock) LockCaller(ctx context.Context, actor shared.Actor, now time.Time) error {
	f.log.add(ctx, "LockCaller %s session %s token %s at %s", actor.UserID, actor.SessionID, actor.APITokenID, now.Format(time.RFC3339Nano))
	return f.err
}

// fakeMAC tags a message with the first 16 bytes of HMAC-SHA256 under a key
// of its own, as identity's does, and logs each Verify with the invitation
// id its message names. Two fakeMACs of other keys tag differently.
type fakeMAC struct {
	log *callLog
	key string
}

func (m fakeMAC) Tag(message []byte) [16]byte {
	h := hmac.New(sha256.New, []byte(m.key))
	h.Write(message)
	return [16]byte(h.Sum(nil)[:16])
}

func (m fakeMAC) Verify(message []byte, tag [16]byte) bool {
	id := uuid.UUID(message[len(message)-16:])
	m.log.calls = append(m.log.calls, "Verify "+id.String())
	want := m.Tag(message)
	return hmac.Equal(want[:], tag[:])
}

// tokenOf is the token of the invitation id under mac.
func tokenOf(mac fakeMAC, id uuid.UUID) string {
	return domain.FormatToken(mac.Tag(domain.InvitationMessage(id)))
}

// fakeInvitations is the invitations' repositories over fakeWorkspaces,
// which answers the workspaces and their locks: it logs every call with its
// arguments, answers from the undeleted invitations it holds, and fails a
// call with the error set for it, wrapped as the store wraps it.
type fakeInvitations struct {
	*fakeWorkspaces
	invitations []domain.Invitation
	listErr     error  // for ListInvitations
	createErr   error  // for CreateInvitations
	taken       string // an address CreateInvitations finds taken, as the unique key would
	readErr     error  // for InvitationByID and LockInvitation
	writeErr    error  // for UpdateInvitationRole and DeleteInvitation
	// failing fails the answers' steps by name: MemberOf, RestoreMember,
	// AcceptInvitation, DeclineInvitation, WorkspaceByID.
	failing map[string]error
}

// step logs a step of an answer to an invitation and fails it with the
// error set for it, wrapped as the store wraps it.
func (f *fakeInvitations) step(ctx context.Context, name, format string, args ...any) error {
	f.log.add(ctx, name+" "+format, args...)
	if err := f.failing[name]; err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (f *fakeInvitations) MemberOf(ctx context.Context, workspaceID, userID uuid.UUID) (domain.Membership, bool, error) {
	if err := f.step(ctx, "MemberOf", "%s %s", workspaceID, userID); err != nil {
		return domain.Membership{}, false, err
	}
	i := slices.IndexFunc(f.memberships[workspaceID], func(m domain.Membership) bool { return m.MemberID == userID })
	if i < 0 {
		return domain.Membership{}, false, nil
	}
	return f.memberships[workspaceID][i], true, nil
}

func (f *fakeInvitations) RestoreMember(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) error {
	return f.step(ctx, "RestoreMember", "%s as %d by %s at %s", id, role, by, now.Format(time.RFC3339Nano))
}

// AcceptInvitation drops the invitation it holds: an accepted one is
// deleted.
func (f *fakeInvitations) AcceptInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := f.step(ctx, "AcceptInvitation", "%s by %s at %s", id, by, now.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	f.invitations = slices.DeleteFunc(f.invitations, func(inv domain.Invitation) bool { return inv.ID == id })
	return nil
}

func (f *fakeInvitations) DeclineInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	return f.step(ctx, "DeclineInvitation", "%s by %s at %s", id, by, now.Format(time.RFC3339Nano))
}

// WorkspaceByID answers the workspace it holds, as stored.
func (f *fakeInvitations) WorkspaceByID(ctx context.Context, id uuid.UUID) (domain.Workspace, error) {
	if err := f.step(ctx, "WorkspaceByID", "%s", id); err != nil {
		return domain.Workspace{}, err
	}
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.ID == id })
	if i < 0 {
		return domain.Workspace{}, fmt.Errorf("read workspace %s: no such row", id)
	}
	return f.workspaces[i], nil
}

func (f *fakeInvitations) InvitationByID(ctx context.Context, id uuid.UUID) (domain.Invitation, error) {
	f.log.add(ctx, "InvitationByID %s", id)
	return f.invitation(id)
}

func (f *fakeInvitations) LockInvitation(ctx context.Context, id uuid.UUID) (domain.Invitation, error) {
	f.log.add(ctx, "LockInvitation %s", id)
	return f.invitation(id)
}

// invitation answers the invitation id it holds, app.ErrNotFound when it
// holds none, and readErr wrapped as the store wraps it.
func (f *fakeInvitations) invitation(id uuid.UUID) (domain.Invitation, error) {
	if f.readErr != nil {
		return domain.Invitation{}, fmt.Errorf("read workspace invitation: %w", f.readErr)
	}
	i := slices.IndexFunc(f.invitations, func(inv domain.Invitation) bool { return inv.ID == id })
	if i < 0 {
		return domain.Invitation{}, app.ErrNotFound
	}
	return f.invitations[i], nil
}

// InvitationPreview answers the invitation it holds with its workspace's
// name and slug; app.ErrNotFound when it holds neither.
func (f *fakeInvitations) InvitationPreview(ctx context.Context, id uuid.UUID) (domain.InvitationPreview, error) {
	f.log.add(ctx, "InvitationPreview %s", id)
	inv, err := f.invitation(id)
	if err != nil {
		return domain.InvitationPreview{}, err
	}
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.ID == inv.WorkspaceID })
	if i < 0 {
		return domain.InvitationPreview{}, app.ErrNotFound
	}
	return domain.InvitationPreview{ID: inv.ID, Role: inv.Role, Declined: inv.Responded(), WorkspaceName: f.workspaces[i].Name,
		WorkspaceSlug: f.workspaces[i].Slug}, nil
}

// ShareWorkspace answers as LockWorkspace does.
func (f *fakeInvitations) ShareWorkspace(ctx context.Context, id uuid.UUID) error {
	f.log.add(ctx, "ShareWorkspace %s", id)
	return f.lockByID(id)
}

// UpdateInvitationRole sets the role of the invitation it holds and answers
// it as stored.
func (f *fakeInvitations) UpdateInvitationRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Invitation, error) {
	f.log.add(ctx, "UpdateInvitationRole %s to %d by %s at %s", id, role, by, now.Format(time.RFC3339Nano))
	if f.writeErr != nil {
		return domain.Invitation{}, fmt.Errorf("update workspace invitation: %w", f.writeErr)
	}
	i := slices.IndexFunc(f.invitations, func(inv domain.Invitation) bool { return inv.ID == id })
	if i < 0 {
		return domain.Invitation{}, fmt.Errorf("update workspace invitation %s: no such row", id)
	}
	f.invitations[i].Role = role
	return f.invitations[i], nil
}

// DeleteInvitation drops the invitation it holds.
func (f *fakeInvitations) DeleteInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DeleteInvitation %s by %s at %s", id, by, now.Format(time.RFC3339Nano))
	if f.writeErr != nil {
		return fmt.Errorf("delete workspace invitation: %w", f.writeErr)
	}
	f.invitations = slices.DeleteFunc(f.invitations, func(inv domain.Invitation) bool { return inv.ID == id })
	return nil
}

func (f *fakeInvitations) ListInvitations(ctx context.Context, workspaceID uuid.UUID) ([]domain.Invitation, error) {
	f.log.add(ctx, "ListInvitations %s", workspaceID)
	if f.listErr != nil {
		return nil, fmt.Errorf("list workspace invitations: %w", f.listErr)
	}
	var out []domain.Invitation
	for _, inv := range f.invitations {
		if inv.WorkspaceID == workspaceID {
			out = append(out, inv)
		}
	}
	return out, nil
}

// CreateInvitations logs the rows' addresses in the order given, and their
// workspace, inviter and time; it stores them and answers them as stored,
// at the stored time, unless an address is taken.
func (f *fakeInvitations) CreateInvitations(ctx context.Context, rows []app.InvitationRow) ([]domain.Invitation, error) {
	var emails []string
	for _, r := range rows {
		emails = append(emails, fmt.Sprintf("%s as %d", r.Email, r.Role))
	}
	f.log.add(ctx, "CreateInvitations %s in %s by %s at %s", strings.Join(emails, ", "), rows[0].WorkspaceID, rows[0].CreatedBy,
		rows[0].Now.Format(time.RFC3339Nano))
	if f.createErr != nil {
		return nil, fmt.Errorf("create workspace invitation: %w", f.createErr)
	}
	var out []domain.Invitation
	for _, r := range rows {
		if r.Email == f.taken {
			return nil, &app.DuplicateInvitation{Email: r.Email}
		}
		by := r.CreatedBy
		out = append(out, domain.Invitation{ID: r.ID, WorkspaceID: r.WorkspaceID, Email: r.Email, Role: r.Role, CreatedAt: stored(r.Now),
			CreatedByID: &by})
	}
	f.invitations = append(f.invitations, out...)
	return out, nil
}
