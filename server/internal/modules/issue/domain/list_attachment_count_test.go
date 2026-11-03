package domain_test

import (
	"encoding/json"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func TestListAttachmentCountCursor(t *testing.T) {
	for _, order := range []domain.ListOrder{"attachment_count", "-attachment_count"} {
		for _, key := range []string{"0", "1", "2", "9007199254740992", "9007199254740993", "9223372036854775807"} {
			c := domain.ListCursor{Order: order, Key: json.RawMessage(key), ID: uuid.MustParse("00000000-0000-0000-0000-000000000001")}
			encoded, err := domain.EncodeListCursor(c)
			if err != nil {
				t.Fatal(order, key, err)
			}
			decoded, err := domain.DecodeListCursor(encoded, order)
			if err != nil || string(decoded.Key) != key || decoded.ID != c.ID {
				t.Fatalf("exact count cursor %s/%s: %+v %v", order, key, decoded, err)
			}
		}
		for _, key := range []string{"-1", "1.5", "null", "9223372036854775808", `"2"`} {
			if _, err := domain.EncodeListCursor(domain.ListCursor{Order: order, Key: json.RawMessage(key), ID: uuid.NewV7()}); err == nil {
				t.Fatalf("accepted invalid count %s", key)
			}
		}
	}
}
