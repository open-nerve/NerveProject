package httpadapter_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
)

type brokenPublishBody struct{}

func (brokenPublishBody) Read([]byte) (int, error) { return 0, errGone }
func (brokenPublishBody) Close() error             { return nil }

// oneBytePublishBody exposes over-reading as an observable I/O boundary failure.
type oneBytePublishBody struct {
	t     *testing.T
	reads int
}

func (b *oneBytePublishBody) Read(p []byte) (int, error) {
	b.reads++
	if b.reads > 1 {
		b.t.Error("publish read past first byte")
		return 0, io.EOF
	}
	if len(p) != 1 {
		b.t.Errorf("read buffer=%d want one byte", len(p))
	}
	p[0] = 'x'
	return 1, nil
}
func (*oneBytePublishBody) Close() error { return nil }

// Catches trusting ContentLength, accepting whitespace, reading past byte one,
// or putting the guard outside authentication. Refused bodies never reach app.
func TestDraftPublishBody(t *testing.T) {
	for _, tc := range []struct {
		name, body, token      string
		chunked, broken, probe bool
		want                   int
	}{
		{name: "empty", token: "alice", want: 201},
		{name: "unknown empty", token: "alice", chunked: true, want: 201},
		{name: "space", body: " ", token: "alice", want: 400},
		{name: "json", body: "{}", token: "alice", want: 400},
		{name: "chunked", body: "x", token: "alice", chunked: true, want: 400},
		{name: "binary", body: string([]byte{0xff}), token: "alice", chunked: true, want: 400},
		{name: "first byte only", token: "alice", probe: true, want: 400},
		{name: "read error", token: "alice", broken: true, want: 400},
		{name: "anonymous first", body: "{}", want: 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uc := &fakeGet{answer: webEight}
			h := newServer(t, httpadapter.UseCases{PublishDraft: uc})
			req := request(http.MethodPost, "/api/v0/drafts/"+webID.String()+"/publish", tc.token, tc.body)
			if tc.chunked {
				req.Body = io.NopCloser(strings.NewReader(tc.body))
				req.ContentLength = -1
				req.TransferEncoding = []string{"chunked"}
			}
			if tc.broken {
				req.Body = brokenPublishBody{}
				req.ContentLength = -1
			}
			if tc.probe {
				req.Body = &oneBytePublishBody{t: t}
				req.ContentLength = -1
			}
			res, raw := do(t, h, req)
			if res.StatusCode != tc.want {
				t.Fatalf("status=%d %s want %d", res.StatusCode, raw, tc.want)
			}
			calls := 0
			if tc.want == 201 {
				calls = 1
			}
			if len(uc.calls) != calls {
				t.Fatalf("usecase calls=%v want %d", uc.calls, calls)
			}
		})
	}
}
