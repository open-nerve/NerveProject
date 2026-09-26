package shared

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
)

// cursorVersion is the version of the cursor envelope, v.
const cursorVersion = 1

// cursorEnvelope is what a page cursor encodes (M2 design 3.12): the
// envelope's version and the list's own payload.
type cursorEnvelope struct {
	V int             `json:"v"`
	P json.RawMessage `json:"p"`
}

// EncodeCursor returns the page cursor of payload: the JSON {"v":1,"p":…}
// in unpadded base64url. Each list defines its payload by its own sort key,
// ending in the id, so pages neither repeat nor skip a row (M2 design 3.12).
func EncodeCursor(payload any) (string, error) {
	p, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(cursorEnvelope{V: cursorVersion, P: p})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(out), nil
}

// DecodeCursor reads a cursor that EncodeCursor wrote into payload, whose
// JSON decoding judges the payload. Anything else is InvalidCursor: not
// unpadded base64url, not the envelope and nothing more, another version, no
// payload, or a payload that does not decode.
func DecodeCursor(cursor string, payload any) error {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return InvalidCursor()
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var env cursorEnvelope
	if err := dec.Decode(&env); err != nil || !errors.Is(dec.Decode(new(json.RawMessage)), io.EOF) {
		return InvalidCursor()
	}
	if env.V != cursorVersion || len(env.P) == 0 || string(env.P) == "null" {
		return InvalidCursor()
	}
	if err := json.Unmarshal(env.P, payload); err != nil {
		return InvalidCursor()
	}
	return nil
}

// InvalidCursor reports a cursor that is not one the list issued: 400
// bad_request on the cursor parameter (M2 design 3.11, 3.12).
func InvalidCursor() *Error {
	return &Error{
		Kind: KindBadRequest, Code: CodeBadRequest, Detail: "The cursor is not one this list issued.",
		Fields: []FieldError{{Field: "cursor", Code: FieldInvalidFormat, Message: "is not a cursor of this list"}},
	}
}
