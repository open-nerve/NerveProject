package app_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// issueRefRows is the store IssueRefs reads through, as a fake: it answers
// those of its rows whose ids it is asked, in id order, as the store does,
// and records each set of ids it is asked; err fails each read.
type issueRefRows struct {
	rows  []app.IssueRefRow
	err   error
	asked [][]uuid.UUID
}

func (f *issueRefRows) IssueRefRows(_ context.Context, ids []uuid.UUID) ([]app.IssueRefRow, error) {
	f.asked = append(f.asked, ids)
	if f.err != nil {
		return nil, f.err
	}
	var out []app.IssueRefRow
	for _, r := range f.rows {
		if slices.Contains(ids, r.ID) {
			out = append(out, r)
		}
	}
	return out, nil
}

// IssueRefs names the work items another names as a reader sees them now
// (M7 P1c spec 2.6, R4), by the grants he is given: of web's, which gail
// reads as a guest who sees only her own, hers but not alice's, nor one
// whose creator is not known; every one of web's to bob, its member; none
// of ops', which neither reads, nor of another workspace's project, which
// is among no grant. The store's row, archived or not, is the answer as it
// read it, and the ids it is asked are the reader's, in one read.
func TestIssueRefsAnswerWhatTheReaderSees(t *testing.T) {
	gail, alice, bob := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	web, ops, far := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	row := func(project uuid.UUID, sequence int, name string, creator *uuid.UUID) app.IssueRefRow {
		return app.IssueRefRow{IssueRef: app.IssueRef{ID: uuid.NewV7(), ProjectID: project, SequenceID: sequence, Name: name}, CreatedByID: creator}
	}
	hers, alices, unknown := row(web, 1, "Hers", &gail), row(web, 2, "Alice's", &alice), row(web, 3, "Unknown", nil)
	opsHers, farHers := row(ops, 1, "Ops'", &gail), row(far, 1, "Far", &gail)
	rows := &issueRefRows{rows: []app.IssueRefRow{hers, alices, unknown, opsHers, farHers}}
	refs := app.NewIssueRefs(rows)
	ids := []uuid.UUID{hers.ID, alices.ID, unknown.ID, opsHers.ID, farHers.ID, uuid.NewV7()}
	for _, tt := range []struct {
		name   string
		reader uuid.UUID
		grants []app.ProjectGrant
		want   []app.IssueRefRow
	}{
		{"gail, web's guest who sees only her own", gail, []app.ProjectGrant{{ProjectID: web, OwnOnly: true}}, []app.IssueRefRow{hers}},
		{"bob, web's member", bob, []app.ProjectGrant{{ProjectID: web}}, []app.IssueRefRow{hers, alices, unknown}},
		{"alice, who sees only her own in web", alice, []app.ProjectGrant{{ProjectID: web, OwnOnly: true}}, []app.IssueRefRow{alices}},
		{"gail, reading none", gail, nil, nil},
	} {
		rows.asked = nil
		got, err := refs.IssueRefs(context.Background(), tt.reader, tt.grants, ids)
		want := map[uuid.UUID]app.IssueRef{}
		for _, r := range tt.want {
			want[r.ID] = r.IssueRef
		}
		if err != nil || len(got) != len(want) || !maps.Equal(got, want) {
			t.Errorf("%s: IssueRefs() = %+v, %v; want %+v", tt.name, got, err, want)
		}
		if len(rows.asked) != 1 || !slices.Equal(rows.asked[0], ids) {
			t.Errorf("%s: the store was asked %v; want the ids once", tt.name, rows.asked)
		}
	}
}

// No ids are no read and no answer; a failed read is its error, and no
// answer.
func TestIssueRefsReadNothingForNoIDsAndFailAsTheStore(t *testing.T) {
	rows := &issueRefRows{}
	refs := app.NewIssueRefs(rows)
	gail, web := uuid.NewV7(), uuid.NewV7()
	grants := []app.ProjectGrant{{ProjectID: web}}
	for _, ids := range [][]uuid.UUID{nil, {}} {
		if got, err := refs.IssueRefs(context.Background(), gail, grants, ids); err != nil || len(got) != 0 {
			t.Errorf("IssueRefs(%v) = %+v, %v; want none", ids, got, err)
		}
	}
	if len(rows.asked) != 0 {
		t.Errorf("the store was asked %v; want no read for no ids", rows.asked)
	}
	failure := errors.New("connection reset")
	rows.err = failure
	if got, err := refs.IssueRefs(context.Background(), gail, grants, []uuid.UUID{uuid.NewV7()}); !errors.Is(err, failure) || got != nil {
		t.Errorf("IssueRefs() = %+v, %v; want %v and none", got, err, failure)
	}
}
