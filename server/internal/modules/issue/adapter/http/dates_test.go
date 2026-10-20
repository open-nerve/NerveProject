package httpadapter_test

import (
	"encoding/json"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
)

// A date passes the body check exactly when the generated types decode it
// (M4 design 6.1): bodyshape reads a date by a rule of its own (checkDate),
// since the platform may not import shared, whose Date the generated
// IssueCreate and IssueUpdate decode. Each sample is a day, one that is no
// day, or one in another layout, with a time or a zone, with white space,
// or written with an escape; each goes as a new work item's start date and
// as a change's.
func TestDatesPassTheBodyCheckAsTheyDecode(t *testing.T) {
	shapes := gen.BodyShapes()
	for raw, want := range map[string]bool{
		`"2026-10-19"`: true, `"0001-01-01"`: true, `"9999-12-31"`: true, `"2024-02-29"`: true, `"2026-10-1\u0039"`: true,
		`"2026-02-29"`: false, `"2026-04-31"`: false, `"2026-00-10"`: false, `"2026-13-01"`: false, `"0000-12-31"`: false,
		`"2026-1-19"`: false, `"2026-10-9"`: false, `"26-10-19"`: false, `"2026/10/19"`: false, `"2026-10-19T00:00:00Z"`: false,
		`"2026-10-19+08:00"`: false, `" 2026-10-19"`: false, `"2026-10-19\n"`: false, `"+2026-10-19"`: false, `"10000-01-01"`: false,
		`"２０２６-10-19"`: false, `""`: false,
	} {
		for _, tt := range []struct {
			route, body string
			into        any
		}{
			{"POST /api/v0/projects/{project_id}/issues", `{"name":"A","start_date":` + raw + `}`, new(gen.IssueCreate)},
			{"PATCH /api/v0/issues/{issue_id}", `{"start_date":` + raw + `}`, new(gen.IssueUpdate)},
		} {
			checked, decoded := shapes.Check(tt.route, []byte(tt.body)), json.Unmarshal([]byte(tt.body), tt.into)
			if (checked == nil) != want || (decoded == nil) != want {
				t.Errorf("%s %s: the body check %v, the decoder %v; want both to accept it: %v", tt.route, tt.body, checked, decoded, want)
			}
		}
	}
}
