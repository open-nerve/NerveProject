package shared_test

import (
	"encoding/base64"
	"errors"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// page stands for a list's payload.
type page struct {
	After string `json:"after"`
}

func b64(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

func TestCursorRoundTrip(t *testing.T) {
	cursor, err := shared.EncodeCursor(page{After: "x?y"})
	if err != nil {
		t.Fatal(err)
	}
	// The envelope is the JSON {"v":1,"p":…} in unpadded base64url.
	if want := b64(`{"v":1,"p":{"after":"x?y"}}`); cursor != want {
		t.Errorf("EncodeCursor() = %q, want %q", cursor, want)
	}

	var got page
	if err := shared.DecodeCursor(cursor, &got); err != nil || got.After != "x?y" {
		t.Errorf("DecodeCursor() = %+v, %v; want the payload back", got, err)
	}
}

// Anything but a cursor that EncodeCursor wrote is 400 bad_request on the
// cursor parameter.
func TestDecodeCursorRejects(t *testing.T) {
	valid := `{"v":1,"p":{"after":"x"}}`
	tests := []struct{ name, cursor string }{
		{"empty", ""},
		{"not base64", "not a cursor!"},
		{"padded", base64.URLEncoding.EncodeToString([]byte(valid + " "))},
		{"standard alphabet", base64.RawStdEncoding.EncodeToString([]byte(`{"v":1,"p":{"after":"??>"}}`))},
		{"not JSON", b64(`{"v":1,`)},
		{"not an object", b64(`[1,{"after":"x"}]`)},
		{"an unknown member", b64(`{"v":1,"p":{"after":"x"},"x":0}`)},
		{"a second value after it", b64(valid + `{}`)},
		{"another version", b64(`{"v":2,"p":{"after":"x"}}`)},
		{"no version", b64(`{"p":{"after":"x"}}`)},
		{"no payload", b64(`{"v":1}`)},
		{"a null payload", b64(`{"v":1,"p":null}`)},
		{"a payload of another list", b64(`{"v":1,"p":{"after":5}}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got page
			err := shared.DecodeCursor(tt.cursor, &got)

			var se *shared.Error
			want := []shared.FieldError{{Field: "cursor", Code: shared.FieldInvalidFormat, Message: "is not a cursor of this list"}}
			if !errors.As(err, &se) || se.ProblemStatus() != 400 || se.Code != shared.CodeBadRequest || !slices.Equal(se.Fields, want) {
				t.Errorf("DecodeCursor(%q) = %v, want 400 bad_request on cursor", tt.cursor, err)
			}
		})
	}
	// The same bytes with the version it expects decode: each case above
	// breaks one thing.
	var got page
	if err := shared.DecodeCursor(b64(valid), &got); err != nil {
		t.Errorf("DecodeCursor(valid) = %v", err)
	}
}
