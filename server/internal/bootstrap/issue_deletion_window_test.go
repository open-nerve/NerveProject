package bootstrap

import (
	"strings"
	"testing"
)

type deletionWindowRequest struct {
	name, actor, method, path, body string
	want                            int
	absentPath, absentBody          string
	check                           func(*testing.T, string)
}

func (r deletionWindowRequest) answer(t *testing.T, w issueWorld) wholeAnswer {
	t.Helper()
	req := newRequest(t, r.method, w.base+r.path, w.tokens[r.actor], nil)
	if r.body != "" {
		req = newRequest(t, r.method, w.base+r.path, w.tokens[r.actor], []byte(r.body))
	}
	w.contract.CheckRequest(t, req)
	res, raw := send(t, req)
	w.contract.CheckResponse(t, req, res)
	if res.StatusCode != r.want {
		t.Fatalf("%s: %s %s as %s = %d %s want %d", r.name, r.method, r.path, r.actor, res.StatusCode, raw, r.want)
	}
	if r.check != nil {
		r.check(t, string(raw))
	}
	return wholeAnswerOf(res, raw)
}

func (r deletionWindowRequest) checked(t *testing.T, w issueWorld) wholeAnswer {
	t.Helper()
	got := r.answer(t, w)
	if r.absentPath != "" {
		missing := r
		missing.path, missing.body, missing.check = r.absentPath, r.absentBody, nil
		if absent := missing.answer(t, w); got != absent {
			t.Fatalf("%s differs from unknown resource:\n%+v\n%+v", r.name, got, absent)
		}
	}
	return got
}

func (f deletionWindowFixture) requests(t *testing.T, workspace bool) []deletionWindowRequest {
	t.Helper()
	absent := f.x.absent(workspace)
	var requests []deletionWindowRequest
	for _, row := range deletionWindowCases() {
		path, body := row.request(f.x)
		if !pathOf(row.pattern, strings.Split(path, "?")[0]) {
			t.Fatal("wrong operation target", row.op, path)
		}
		r := deletionWindowRequest{name: row.op, actor: "alice", method: row.method, path: path, body: body, want: 404}
		if !workspace {
			switch row.op {
			case "searchIssues", "listMemberIssues", "listDrafts":
				r.want = 200
			case "createDraft":
				r.want = 422
			}
		}
		if r.want != 200 {
			r.absentPath, r.absentBody = row.request(absent)
		}
		if row.op == "createDraft" && !workspace {
			r.check = windowField("project_id not_allowed")
		}
		if !workspace {
			switch row.op {
			case "searchIssues":
				r.check = windowIDs(f.kept[2], f.kept[1], f.kept[0])
			case "listMemberIssues":
				r.check = windowPageIDs(3, f.kept[2], f.kept[1], f.kept[0])
			case "listDrafts":
				r.check = windowPageIDs(2, f.noProject, f.qDraft)
			}
		}
		requests = append(requests, r)
	}
	return requests
}

// A missing visibility check must fail while rows still exist, and a worker's
// changes cannot alter any caller-visible byte after the deletion commits.
func TestIssueDeletionWindowAnswersAreStable(t *testing.T) {
	for _, workspace := range []bool{false, true} {
		t.Run(map[bool]string{false: "project", true: "workspace"}[workspace], func(t *testing.T) {
			f := newDeletionWindowFixture(t)
			requests := f.requests(t, workspace)
			requests = append(requests, f.pageRequests(t, workspace)...)
			requests = append(requests, f.boundaryRequests(t, workspace)...)
			complete := f.suspend(t, workspace)
			before := make([]wholeAnswer, len(requests))
			sampled := make([]bool, len(requests))
			for i, r := range requests {
				sampled[i] = t.Run("pending/"+r.name, func(t *testing.T) { before[i] = r.checked(t, f.w) })
			}
			if !workspace {
				f.unassignedWrites(t)
			}
			complete()
			for i, r := range requests {
				t.Run("completed/"+r.name, func(t *testing.T) {
					if got := r.checked(t, f.w); sampled[i] && got != before[i] {
						t.Fatalf("answer changed after worker completion:\n%+v\n%+v", got, before[i])
					}
				})
			}
			if !workspace {
				f.unassignedWrites(t)
			}
		})
	}
}

func windowField(want string) func(*testing.T, string) {
	return func(t *testing.T, body string) {
		t.Helper()
		if got := refusalOf(t, []byte(body), false); got != "validation_failed "+want {
			t.Fatalf("fields=%s want %s", got, want)
		}
	}
}
