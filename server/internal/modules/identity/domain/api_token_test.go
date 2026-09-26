package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// now is the time the rules are checked at.
var now = time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)

// samplePAT has every byte different, so a shifted or truncated copy
// differs, and its text holds both "-" and "_", which only base64url writes.
func samplePAT() PAT {
	var p PAT
	for i := range p {
		p[i] = byte(255 - i*7)
	}
	return p
}

func TestPATRoundTrip(t *testing.T) {
	p := samplePAT()
	s := p.String()

	// The secret-scanning pattern of M2 design 8.6 matches it.
	if !regexp.MustCompile(`^nrv_pat_[A-Za-z0-9_-]{43}$`).MatchString(s) {
		t.Errorf("String() = %q, want nrv_pat_ and 43 base64url characters", s)
	}
	if got, ok := ParsePAT(s); !ok || got != p {
		t.Errorf("ParsePAT(String()) = %x, %v; want the token back", got, ok)
	}
	// The whole token is hashed, prefix included, not only the secret.
	if want := sha256.Sum256([]byte(s)); !bytes.Equal(p.Hash(), want[:]) {
		t.Errorf("Hash() = %x, want SHA-256 of %q", p.Hash(), s)
	}
}

// Only the spelling String writes is a token: nothing is looked up for
// anything else.
func TestParsePATRejects(t *testing.T) {
	s := samplePAT().String()
	last := s[len(s)-1]
	// The last character carries 4 bits of the secret and 2 unused ones:
	// flipping its lowest bit keeps the secret and breaks the strict form.
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	unusedBit := string(alphabet[strings.IndexByte(alphabet, last)^1])
	tests := []struct{ name, token string }{
		{"empty", ""},
		{"no prefix", s[len(PATPrefix):]},
		{"a refresh token's prefix", RefreshTokenPrefix + s[len(PATPrefix):]},
		{"upper-case prefix", strings.ToUpper(PATPrefix) + s[len(PATPrefix):]},
		{"one character short", s[:len(s)-1]},
		{"one character more", s + "A"},
		{"padded", s[:len(s)-1] + "="},
		{"unused bits set", s[:len(s)-1] + unusedBit},
		{"a newline inside", s[:20] + "\n" + s[21:]},
		{"a newline added", s + "\n"},
		{"the standard alphabet", s[:20] + "+" + s[21:]},
		{"blank around it", " " + s[:len(s)-1]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := ParsePAT(tt.token); ok {
				t.Errorf("ParsePAT(%q) = %x, true; want false", tt.token, got)
			}
		})
	}
}

func TestCheckAPITokenAcceptsAValidSpec(t *testing.T) {
	label, max := "deploy bot", strings.Repeat("界", 255)
	later := now.Add(time.Nanosecond)
	for _, spec := range []APITokenSpec{
		{},
		{Label: &label, Description: "ci", ExpiredAt: &later},
		{Label: &max},
	} {
		if err := CheckAPIToken(spec, now); err != nil {
			t.Errorf("CheckAPIToken(%+v) = %v, want nil", spec, err)
		}
	}
}

// Every problem is reported at once, as one 422.
func TestCheckAPITokenReportsEveryField(t *testing.T) {
	empty, long, nul := "", strings.Repeat("界", 256), "a\x00b"
	past := now.Add(-time.Second)
	tests := []struct {
		name string
		spec APITokenSpec
		want []shared.FieldError
	}{
		{"empty label", APITokenSpec{Label: &empty}, []shared.FieldError{{Field: "label", Code: "too_short", Message: "must not be empty"}}},
		{"long label", APITokenSpec{Label: &long}, []shared.FieldError{{Field: "label", Code: "too_long", Message: "must be at most 255 characters"}}},
		{"NUL in the label", APITokenSpec{Label: &nul}, []shared.FieldError{{Field: "label", Code: "invalid_format", Message: "must not contain a NUL character"}}},
		{"NUL in the description", APITokenSpec{Description: nul}, []shared.FieldError{{Field: "description", Code: "invalid_format", Message: "must not contain a NUL character"}}},
		{"expiry now", APITokenSpec{ExpiredAt: &now}, []shared.FieldError{{Field: "expired_at", Code: "must_be_future", Message: "must be in the future"}}},
		{"all at once", APITokenSpec{Label: &empty, Description: nul, ExpiredAt: &past}, []shared.FieldError{
			{Field: "label", Code: "too_short", Message: "must not be empty"},
			{Field: "description", Code: "invalid_format", Message: "must not contain a NUL character"},
			{Field: "expired_at", Code: "must_be_future", Message: "must be in the future"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckAPIToken(tt.spec, now)

			var se *shared.Error
			if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
				t.Errorf("CheckAPIToken() = %+v, want validation_failed with %+v", err, tt.want)
			}
		})
	}
}

func TestPageSize(t *testing.T) {
	for _, tt := range []struct {
		limit *int
		want  int
	}{{nil, 50}, {ptr(1), 1}, {ptr(100), 100}} {
		if got, err := PageSize(tt.limit); err != nil || got != tt.want {
			t.Errorf("PageSize(%v) = %d, %v; want %d", tt.limit, got, err, tt.want)
		}
	}
	for _, limit := range []int{0, 101, -1} {
		_, err := PageSize(&limit)

		var se *shared.Error
		want := []shared.FieldError{{Field: "limit", Code: "out_of_range", Message: "must be between 1 and 100"}}
		if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, want) {
			t.Errorf("PageSize(%d) = %v, want 422 limit out_of_range", limit, err)
		}
	}
}

func ptr[T any](v T) *T { return &v }

// DecodeCursor accepts only what EncodeCursor writes, so the time MarshalJSON
// writes must read back to the same instant and the same spelling, with or
// without a fraction, in UTC or at another offset.
func TestAPITokenCursorRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{"microseconds", time.Date(2026, 9, 25, 10, 0, 0, 123456000, time.UTC),
			`["2026-09-25T10:00:00.123456Z","0199a2b4-0000-7000-8000-000000000001"]`},
		{"a whole second", time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC),
			`["2026-09-25T10:00:00Z","0199a2b4-0000-7000-8000-000000000001"]`},
		{"an offset other than UTC", time.Date(2026, 9, 25, 15, 30, 0, 123456000, time.FixedZone("", 5*3600+30*60)),
			`["2026-09-25T15:30:00.123456+05:30","0199a2b4-0000-7000-8000-000000000001"]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := APITokenCursor{CreatedAt: tt.at, ID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")}

			b, err := json.Marshal(c)
			if err != nil || string(b) != tt.want {
				t.Errorf("json.Marshal() = %s, %v; want %s", b, err, tt.want)
			}
			cursor, err := shared.EncodeCursor(c)
			if err != nil {
				t.Fatal(err)
			}
			var got APITokenCursor
			if err := shared.DecodeCursor(cursor, &got); err != nil || !got.CreatedAt.Equal(c.CreatedAt) || got.ID != c.ID {
				t.Errorf("DecodeCursor() = %+v, %v; want %+v", got, err, c)
			}
		})
	}
}

func TestAPITokenCursorRejects(t *testing.T) {
	for _, payload := range []string{
		`null`,
		`{"created_at":"2026-09-25T10:00:00Z","id":"0199a2b4-0000-7000-8000-000000000001"}`,
		`[]`,
		`["2026-09-25T10:00:00Z"]`,
		`["2026-09-25T10:00:00Z","0199a2b4-0000-7000-8000-000000000001","x"]`,
		`[1,"0199a2b4-0000-7000-8000-000000000001"]`,
		`["2026-09-25 10:00:00","0199a2b4-0000-7000-8000-000000000001"]`,
		`["2026-09-25T10:00:00Z","not-a-uuid"]`,
	} {
		var c APITokenCursor
		if err := json.Unmarshal([]byte(payload), &c); err == nil {
			t.Errorf("json.Unmarshal(%s) = %+v, want an error", payload, c)
		}
	}
}

// UnmarshalJSON reads other spellings of the same time and id than
// MarshalJSON writes; DecodeCursor refuses them, as it accepts only what
// EncodeCursor writes.
func TestAPITokenCursorRefusesOtherSpellings(t *testing.T) {
	const at, id = "2026-09-25T10:00:00.123456", "0199a2b4-0000-7000-8000-000000000001"
	for _, payload := range []string{
		`["` + at + `+00:00","` + id + `"]`,
		`["` + at + `0Z","` + id + `"]`,
		`["` + at + `Z","` + strings.ToUpper(id) + `"]`,
		`["` + at + `Z","{` + id + `}"]`,
		`["` + at + `Z", "` + id + `"]`,
	} {
		var read APITokenCursor
		if err := json.Unmarshal([]byte(payload), &read); err != nil {
			t.Fatalf("json.Unmarshal(%s) = %v; the case needs a payload UnmarshalJSON reads", payload, err)
		}

		var got APITokenCursor
		cursor := base64.RawURLEncoding.EncodeToString([]byte(`{"v":1,"p":` + payload + `}`))
		if err := shared.DecodeCursor(cursor, &got); !errors.Is(err, shared.InvalidCursor()) {
			t.Errorf("DecodeCursor(%s) = %v, want InvalidCursor", payload, err)
		}
	}
}
