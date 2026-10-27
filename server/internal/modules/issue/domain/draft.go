package domain

import (
	"encoding/json"
	"strings"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Draft stores a personal, unfinished create payload without issue semantics.
type Draft struct {
	ID, WorkspaceID, CreatedByID uuid.UUID
	ProjectID, UpdatedByID       *uuid.UUID
	Payload                      json.RawMessage
	CreatedAt, UpdatedAt         time.Time
}

// NewDraft gives an optional project and the complete initial payload.
type NewDraft struct {
	ProjectID *uuid.UUID
	Payload   json.RawMessage
}

// DraftPatch distinguishes omitted fields from a cleared project or replaced payload.
type DraftPatch struct {
	SetProject bool
	ProjectID  *uuid.UUID
	SetPayload bool
	Payload    json.RawMessage
}

// CheckDraftPayload checks only the three free strings for PostgreSQL JSONB
// representability. The HTTP boundary owns structure; publication owns create
// semantics. The caller's Unicode, whitespace and HTML remain untouched.
func CheckDraftPayload(raw json.RawMessage) error {
	var payload struct {
		Name            string `json:"name"`
		DescriptionHTML string `json:"description_html"`
		Priority        string `json:"priority"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	var fields []shared.FieldError
	for _, s := range []struct{ path, value string }{
		{"payload.name", payload.Name},
		{"payload.description_html", payload.DescriptionHTML},
		{"payload.priority", payload.Priority},
	} {
		if strings.ContainsRune(s.value, '\x00') {
			fields = append(fields, shared.FieldError{Field: s.path, Code: shared.FieldInvalidFormat, Message: "must not contain a NUL character"})
		}
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}
