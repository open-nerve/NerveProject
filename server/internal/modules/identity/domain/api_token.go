package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// PATPrefix starts every personal access token (M2 design 3.4).
const PATPrefix = "nrv_pat_"

// PAT is the random part of a personal access token: 32 bytes. The server
// stores only Hash.
type PAT [32]byte

// patTextLen is the length of a token as the client holds it: the prefix
// and 43 characters of unpadded base64url, 51.
var patTextLen = len(PATPrefix) + base64.RawURLEncoding.EncodedLen(len(PAT{}))

// String is the token the client holds.
func (p PAT) String() string {
	return PATPrefix + base64.RawURLEncoding.EncodeToString(p[:])
}

// Hash is what api_tokens.token_hash holds: the SHA-256 of the whole token
// (M2 design 3.4).
func (p PAT) Hash() []byte {
	sum := sha256.Sum256([]byte(p.String()))
	return sum[:]
}

// ParsePAT reads a token that String wrote, without looking anything up. It
// accepts only that spelling: the prefix, then 43 characters of unpadded
// base64url whose unused last bits are zero, nothing around them.
func ParsePAT(s string) (PAT, bool) {
	if len(s) != patTextLen || !strings.HasPrefix(s, PATPrefix) {
		return PAT{}, false
	}
	// A decoder skips \r and \n: with them inside, fewer than 32 bytes come out.
	raw, err := base64.RawURLEncoding.Strict().DecodeString(s[len(PATPrefix):])
	if err != nil || len(raw) != len(PAT{}) {
		return PAT{}, false
	}
	return PAT(raw), true
}

// APIToken is a personal access token as the API lists it: never the token
// itself.
type APIToken struct {
	ID          uuid.UUID
	Label       string
	Description string
	ExpiredAt   *time.Time // nil: never expires
	LastUsed    *time.Time // nil: never used
	CreatedAt   time.Time
}

// APITokenSpec is what the caller asks for when creating a token. A nil
// Label asks for a generated one.
type APITokenSpec struct {
	Label       *string
	Description string
	ExpiredAt   *time.Time
}

// maxLabelLength is api_tokens.label's varchar(255), in characters.
const maxLabelLength = 255

// CheckAPIToken checks spec at now (M2 design 4.4, 4.6): a label of 1–255
// characters, a label and a description without NUL, which the database
// cannot store, and an expiry after now. Every problem is reported at once,
// as one 422 validation_failed.
func CheckAPIToken(spec APITokenSpec, now time.Time) error {
	var fields []shared.FieldError
	if spec.Label != nil {
		if f := checkText("label", *spec.Label, true, maxLabelLength); f != nil {
			fields = append(fields, *f)
		}
	}
	if f := checkText("description", spec.Description, false, 0); f != nil {
		fields = append(fields, *f)
	}
	if spec.ExpiredAt != nil && !spec.ExpiredAt.After(now) {
		fields = append(fields, shared.FieldError{Field: "expired_at", Code: shared.FieldMustBeFuture, Message: "must be in the future"})
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}

// checkText checks a text field: empty when it must not be, longer than
// maxLen characters (no bound when 0), or holding a NUL, which Postgres
// text cannot store.
func checkText(field, s string, nonEmpty bool, maxLen int) *shared.FieldError {
	switch {
	case nonEmpty && s == "":
		return &shared.FieldError{Field: field, Code: shared.FieldTooShort, Message: "must not be empty"}
	case maxLen > 0 && utf8.RuneCountInString(s) > maxLen:
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxLen)}
	case strings.ContainsRune(s, 0):
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "must not contain a NUL character"}
	}
	return nil
}

// Page sizes of a list (api/common.yaml Limit).
const (
	DefaultPageSize = 50
	MaxPageSize     = 100
)

// PageSize returns the page size that limit asks for: DefaultPageSize when
// it is nil. Outside 1–MaxPageSize it is 422 validation_failed on limit
// (M2 design 3.11); a limit that is no integer never gets here, the
// parameter binding answers 400.
func PageSize(limit *int) (int, error) {
	switch {
	case limit == nil:
		return DefaultPageSize, nil
	case *limit < 1 || *limit > MaxPageSize:
		return 0, shared.Invalid(shared.FieldError{Field: "limit", Code: shared.FieldOutOfRange, Message: "must be between 1 and 100"})
	}
	return *limit, nil
}

// errCursorShape is a token list cursor that is not [created_at, id].
var errCursorShape = errors.New("the cursor is not [created_at, id]")

// APITokenCursor is the payload of the token list's cursor (M2 design
// 3.12): the last row's created_at and id, the list's order.
type APITokenCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// MarshalJSON writes the cursor as [created_at, id].
func (c APITokenCursor) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]string{c.CreatedAt.Format(time.RFC3339Nano), c.ID.String()})
}

// UnmarshalJSON reads what MarshalJSON wrote: an array of exactly an RFC
// 3339 time and a uuid.
func (c *APITokenCursor) UnmarshalJSON(b []byte) error {
	var parts []string
	if err := json.Unmarshal(b, &parts); err != nil {
		return err
	}
	if len(parts) != 2 {
		return errCursorShape
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return err
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return err
	}
	*c = APITokenCursor{CreatedAt: at, ID: id}
	return nil
}
