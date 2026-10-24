package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestCommentLengthUsesUnicodeCharacters(t *testing.T) {
	if err := domain.CheckCommentHTML(strings.Repeat("界", 500000)); err != nil {
		t.Fatal(err)
	}
	var se *shared.Error
	if err := domain.CheckCommentHTML(strings.Repeat("界", 500001)); !errors.As(err, &se) || se.Fields[0].Field != "comment_html" || se.Fields[0].Code != shared.FieldTooLong {
		t.Fatal(err)
	}
}
