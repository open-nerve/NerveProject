package domain_test

import (
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestCommentAssetsInput(t *testing.T) {
	ids := make([]uuid.UUID, 100)
	for n := range ids {
		ids[n] = uuid.NewV7()
	}
	for _, list := range [][]uuid.UUID{nil, {}, ids} {
		if err := domain.CheckNewComment(domain.NewComment{HTML: "x", AttachmentIDs: list}); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		ids         []uuid.UUID
		field, code string
	}{
		{append(slices.Clone(ids), uuid.NewV7()), "attachment_ids", shared.FieldTooLong},
		{[]uuid.UUID{ids[1], ids[0], ids[1]}, "attachment_ids[2]", shared.FieldDuplicate},
	} {
		err := domain.CheckNewComment(domain.NewComment{HTML: "x", AttachmentIDs: tc.ids})
		var problem *shared.Error
		if !errors.As(err, &problem) || len(problem.Fields) != 1 || problem.Fields[0].Field != tc.field || problem.Fields[0].Code != tc.code {
			t.Fatal("unstable semantic rejection", err)
		}
	}
}
