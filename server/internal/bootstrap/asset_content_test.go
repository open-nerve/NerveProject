package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
)

type contentAdapterProbe struct {
	ctx     context.Context
	content asset.ContentBindRequest
	draft   asset.DraftAssetRequest
	at      time.Time
	issueID uuid.UUID
	err     error
}

func (f *contentAdapterProbe) LockContent(ctx context.Context, q asset.ContentBindRequest) (time.Time, error) {
	f.ctx, f.content = ctx, q
	return f.at, f.err
}
func (f *contentAdapterProbe) BindContent(ctx context.Context, q asset.ContentBindRequest, at time.Time) error {
	f.ctx, f.content, f.at = ctx, q, at
	return f.err
}
func (f *contentAdapterProbe) LockDraftAssets(ctx context.Context, q asset.DraftAssetRequest) (time.Time, error) {
	f.ctx, f.draft = ctx, q
	return f.at, f.err
}
func (f *contentAdapterProbe) MoveDraftAssets(ctx context.Context, q asset.DraftAssetRequest, id uuid.UUID, at time.Time) error {
	f.ctx, f.draft, f.issueID, f.at = ctx, q, id, at
	return f.err
}

func TestContentAssetAdapters(t *testing.T) {
	ctx := t.Context()
	at := time.Date(2026, 11, 3, 0, 0, 1, 0, time.UTC)
	lost := errors.New("asset owner failed")
	cq := issue.ContentBindRequest{ActorID: uuid.NewV7(), Host: issue.ContentHost{
		WorkspaceID: uuid.NewV7(), ProjectID: uuid.NewV7(), IssueID: uuid.NewV7(), CommentID: uuid.NewV7(), DraftID: uuid.NewV7(),
	}, ImageIDs: []uuid.UUID{uuid.NewV7(), uuid.NewV7()}, AttachmentIDs: []uuid.UUID{uuid.NewV7()}, NewComment: true}
	want := asset.ContentBindRequest{ActorID: cq.ActorID, Host: asset.ContentHost(cq.Host), ImageIDs: cq.ImageIDs, AttachmentIDs: cq.AttachmentIDs, NewComment: true}
	for _, failure := range []error{nil, lost} {
		f := &contentAdapterProbe{at: at, err: failure}
		p := issueContentBindings{bindings: f}
		got, err := p.LockContent(ctx, cq)
		if !errors.Is(err, failure) || !got.Equal(at) || f.ctx != ctx || !reflect.DeepEqual(f.content, want) {
			t.Fatalf("lock adapter lost values/transaction/error: %+v %v", f, err)
		}
		later := at.Add(time.Second)
		if err := p.BindContent(ctx, cq, later); !errors.Is(err, failure) || f.ctx != ctx || !f.at.Equal(later) || !reflect.DeepEqual(f.content, want) {
			t.Fatalf("bind adapter lost values/transaction/error: %+v %v", f, err)
		}
	}
	for _, withProject := range []bool{false, true} {
		for _, failure := range []error{nil, lost} {
			q := issue.DraftAssetRequest{ActorID: cq.ActorID, WorkspaceID: cq.Host.WorkspaceID, DraftID: cq.Host.DraftID, ImageIDs: cq.ImageIDs, Publish: withProject}
			if withProject {
				q.OldProjectID, q.NewProjectID = &cq.Host.ProjectID, &cq.Host.ProjectID
			}
			want := asset.DraftAssetRequest(q)
			f := &contentAdapterProbe{at: at, err: failure}
			p := issueDraftAssetTransfer{assets: f}
			got, err := p.LockDraftAssets(ctx, q)
			if !errors.Is(err, failure) || !got.Equal(at) || f.ctx != ctx || !reflect.DeepEqual(f.draft, want) {
				t.Fatalf("draft lock adapter lost values/transaction/error: %+v %v", f, err)
			}
			later := at.Add(time.Second)
			if err := p.MoveDraftAssets(ctx, q, cq.Host.IssueID, later); !errors.Is(err, failure) || f.ctx != ctx || f.issueID != cq.Host.IssueID || !f.at.Equal(later) || !reflect.DeepEqual(f.draft, want) {
				t.Fatalf("draft move adapter lost values/transaction/error: %+v %v", f, err)
			}
		}
	}
}
