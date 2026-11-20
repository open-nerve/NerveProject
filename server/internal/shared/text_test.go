package shared_test

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestCheckText(t *testing.T) {
	if f := shared.CheckText("description", "多行\n说明"); f != nil {
		t.Errorf("CheckText of a text without NUL = %+v; want none", f)
	}
	want := shared.FieldError{Field: "description", Code: "invalid_format", Message: "must not contain a NUL character"}
	if f := shared.CheckText("description", "a\x00b"); f == nil || *f != want {
		t.Errorf("CheckText of a text with NUL = %+v; want %+v", f, want)
	}
}
