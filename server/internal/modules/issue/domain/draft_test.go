package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Removing a NUL guard must fail at its exact payload path; applying create
// validation would incorrectly reject these otherwise storable drafts.
func TestDraftPayloadStorageBoundary(t *testing.T) {
	labels := `"0199a2b4-0000-7000-8000-000000000001"`
	for _, raw := range []string{
		`{}`, `{"name":""}`, `{"name":"` + strings.Repeat("字", 256) + `"}`,
		`{"priority":"later"}`, `{"label_ids":[` + strings.Repeat(labels+",", 100) + labels + `]}`,
		`{"state_id":"0199a2b4-0000-7000-8000-000000000099"}`,
		`{"name":"  中文🙂  ","description_html":"<script>你好</script><p> &amp; </p>","priority":"later"}`,
	} {
		payload := json.RawMessage(raw)
		before := bytes.Clone(payload)
		if err := CheckDraftPayload(payload); err != nil {
			t.Errorf("CheckDraftPayload(%s) = %v", raw, err)
		}
		if !bytes.Equal(payload, before) {
			t.Errorf("payload changed: %s", payload)
		}
	}
	for _, field := range []string{"name", "description_html", "priority"} {
		t.Run(field, func(t *testing.T) {
			err := CheckDraftPayload(json.RawMessage(`{"` + field + `":"a\u0000b"}`))
			var problem *shared.Error
			if !errors.As(err, &problem) || problem.ProblemStatus() != 422 || problem.Code != "validation_failed" {
				t.Fatalf("error = %v; want 422 validation_failed", err)
			}
			if len(problem.Fields) != 1 || problem.Fields[0].Field != "payload."+field || problem.Fields[0].Code != "invalid_format" {
				t.Fatalf("fields = %+v; want only payload.%s invalid_format", problem.Fields, field)
			}
		})
	}
	err := CheckDraftPayload(json.RawMessage(`{"name":"\u0000","description_html":"\u0000","priority":"\u0000"}`))
	var problem *shared.Error
	if !errors.As(err, &problem) || len(problem.Fields) != 3 {
		t.Fatalf("combined NUL fields = %v", err)
	}
}
