package httpadapter_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/bodyshape"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Omitting Check would accept missing names, unknown keys and inner nulls.
// Stored JSONB bytes and direct POST must receive the same structural refusal.
func TestCreatePayloadDecoder(t *testing.T) {
	const pattern = "POST /api/v0/projects/{project_id}/issues"
	if _, ok := gen.BodyShapes().Roots[pattern]; !ok {
		t.Fatal("create body root missing")
	}
	decoder := httpadapter.IssueCreatePayloadDecoder{}
	cases := []string{`{}`, `{"name":"A","start_date":"2026-02-30"}`, `{"name":"A","unknown":1}`, `{"name":"A","description_html":null}`, `{"name":"A","assignee_ids":[null]}`, `{"name":"A","label_ids":{}}`, `{"name":"A","state_id":null}`}
	many := map[string]any{"name": "A"}
	for i := 0; i < 20; i++ {
		many[fmt.Sprintf("unknown%02d", i)] = i
	}
	raw, err := json.Marshal(many)
	if err != nil {
		t.Fatal(err)
	}
	cases = append(cases, string(raw))
	for _, body := range cases {
		t.Run(body, func(t *testing.T) {
			// JSONB canonicalization is represented by one marshaled object, then the
			// exact same bytes are passed to both consumers (no raw-order guarantee).
			var object map[string]any
			if err := json.Unmarshal([]byte(body), &object); err != nil {
				t.Fatal(err)
			}
			canonical, err := json.Marshal(object)
			if err != nil {
				t.Fatal(err)
			}
			_, err = decoder.DecodeCreatePayload(canonical)
			var shape *bodyshape.Error
			if !errors.As(err, &shape) {
				t.Fatalf("decoder=%v, want structured 400", err)
			}
			create := &fakeCreate{}
			res, answer := do(t, newServer(t, httpadapter.UseCases{CreateIssue: create}), request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/issues", "alice", string(canonical)))
			var p struct {
				Status int
				Code   string
				Errors []struct{ Field, Code string }
			}
			if err := json.Unmarshal([]byte(answer), &p); err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != shape.ProblemStatus() || p.Status != 400 || p.Code != shape.ProblemCode() || len(p.Errors) != len(shape.Fields) || len(create.calls) != 0 {
				t.Fatalf("POST=%d %s, decoder=%+v", res.StatusCode, answer, shape)
			}
			for i, f := range shape.Fields {
				if p.Errors[i].Field != f.Field || p.Errors[i].Code != f.Code {
					t.Fatalf("POST fields=%+v, decoder=%+v", p.Errors, shape.Fields)
				}
			}
			if body == `{}` && (len(shape.Fields) != 1 || shape.Fields[0].Field != "name" || shape.Fields[0].Code != "required") {
				t.Fatalf("missing name=%+v", shape.Fields)
			}
			if len(object) > 16 && len(shape.Fields) != 16 {
				t.Fatalf("field cap=%d, want 16", len(shape.Fields))
			}
		})
	}
	for _, body := range []string{`{"name":""}`, `{"name":"A","priority":"wrong"}`, `{"name":"A","priority":""}`} {
		t.Run(body, func(t *testing.T) {
			in, err := decoder.DecodeCreatePayload([]byte(body))
			if err != nil {
				t.Fatal(err)
			}
			_, err = domain.CheckNewIssue(in)
			var invalid *shared.Error
			if !errors.As(err, &invalid) || invalid.ProblemStatus() != 422 {
				t.Fatalf("domain=%v, want 422", err)
			}
			create := &fakeCreate{err: err}
			res, _ := do(t, newServer(t, httpadapter.UseCases{CreateIssue: create}), request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/issues", "alice", body))
			if res.StatusCode != 422 || !reflect.DeepEqual(create.got, []domain.NewIssue{in}) {
				t.Fatalf("POST=%d input=%+v, want domain input=%+v", res.StatusCode, create.got, in)
			}
		})
	}
	// The same mapper retains omitted lists versus explicit empty lists.
	for _, body := range []string{`{"name":"A"}`, `{"name":"A","assignee_ids":[],"label_ids":[]}`, `{"name":"A","description_html":"<p>Hi</p>","priority":"high","start_date":"2026-10-12","state_id":"0199a2b4-0000-7000-8000-0000000000c1"}`} {
		in, err := decoder.DecodeCreatePayload([]byte(body))
		if err != nil {
			t.Fatal(err)
		}
		create := &fakeCreate{answer: webEight}
		do(t, newServer(t, httpadapter.UseCases{CreateIssue: create}), request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/issues", "alice", body))
		if !reflect.DeepEqual(create.got, []domain.NewIssue{in}) {
			t.Fatalf("decoder=%+v POST=%+v", in, create.got)
		}
	}
}
