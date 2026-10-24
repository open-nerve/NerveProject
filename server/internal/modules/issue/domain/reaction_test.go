package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestCheckReaction(t *testing.T) {
	for _, s := range []string{"128077", "10084-65039", "0", "9999999", strings.Repeat("1234567-", 15) + "1234567"} {
		if err := domain.CheckReaction(s); err != nil {
			t.Fatal(s, err)
		}
	}
	for _, s := range []string{"", "-1", "12345678", strings.Repeat("1-", 16) + "1", "1--2", "1-", "❤️", strings.Repeat("1234567-", 16) + "1"} {
		var e *shared.Error
		if err := domain.CheckReaction(s); !errors.As(err, &e) || len(e.Fields) != 1 || e.Fields[0].Field != "reaction" {
			t.Fatalf("accepted invalid reaction %q: %v", s, err)
		}
	}
}
