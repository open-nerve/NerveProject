package domain_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Pending metadata is an object; absent keys, nulls and string values remain
// distinct across the JSON boundary used by storage and the response.
func TestLinkMetadata(t *testing.T) {
	for _, raw := range []string{`{}`, `{"favicon":null,"favicon_url":null,"title":"Title","url":"http://example.com"}`, `{"error":"blocked","url":"http://example.com"}`} {
		var got domain.LinkMetadata
		if err := json.Unmarshal([]byte(raw), &got); err != nil || got == nil {
			t.Fatalf("metadata decode = %#v, %v", got, err)
		}
		encoded, err := json.Marshal(got)
		if err != nil || string(encoded) != raw {
			t.Fatalf("metadata JSON = %s, %v; want %s", encoded, err, raw)
		}
		var again domain.LinkMetadata
		if err := json.Unmarshal(encoded, &again); err != nil || !reflect.DeepEqual(got, again) {
			t.Fatalf("metadata round trip = %#v, %v", again, err)
		}
	}
	for _, raw := range []string{`{"title":17}`, `{"favicon":false}`, `{"url":[]}`, `{"error":{}}`} {
		var got domain.LinkMetadata
		if err := json.Unmarshal([]byte(raw), &got); err == nil {
			t.Fatalf("accepted metadata value %s", raw)
		}
	}
}
