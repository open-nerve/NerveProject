package bootstrap

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The whole-program tests (M2 design 3.6, 3.11): the wired app against the
// contract, operation by operation, so an operation added later is covered
// without a new test.

// 1. The union of the modules' public operations is the contract's
// security: [] operations.
func TestPublicOperationsAreTheContractsPublicOperations(t *testing.T) {
	contract := apitest.Load(t)
	a := buildApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})

	var want []string
	for _, op := range contract.Operations() {
		if op.Public {
			want = append(want, op.Pattern())
		}
	}
	if got := slices.Sorted(slices.Values(a.publicOperations)); !slices.Equal(got, want) {
		t.Errorf("the modules declare the public operations %q; the contract has security: [] on %q", got, want)
	}
}

// 2. The routes registered under /api/v0 are the contract's operations: a
// route the contract does not describe fails here.
func TestAPIRoutesAreTheContractsOperations(t *testing.T) {
	contract := apitest.Load(t)
	a := buildApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})

	var got []string
	for _, pattern := range a.router.Patterns() {
		path := pattern
		if _, rest, ok := strings.Cut(pattern, " "); ok {
			path = rest
		}
		if strings.HasPrefix(path, "/api/v0/") {
			got = append(got, pattern)
		}
	}
	slices.Sort(got)
	var want []string
	for _, op := range contract.Operations() {
		want = append(want, op.Pattern())
	}
	if !slices.Equal(got, want) {
		t.Errorf("routes under /api/v0 = %q, want the contract's operations %q", got, want)
	}
}

// 3. Every operation that is not public answers 401 without a token.
func TestOperationsThatNeedATokenAnswer401WithoutOne(t *testing.T) {
	contract := apitest.Load(t)
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})

	for _, op := range contract.Operations() {
		if op.Public {
			continue
		}
		t.Run(op.Pattern(), func(t *testing.T) {
			req := newRequest(t, op.Method, base+op.Target(), "", nil)
			res, body := send(t, req)

			contract.CheckResponse(t, req, res)
			if res.StatusCode != http.StatusUnauthorized || problemCode(t, body) != "unauthorized" ||
				res.Header.Get("WWW-Authenticate") != "Bearer" {
				t.Errorf("answer = %d %s WWW-Authenticate %q, want 401 unauthorized Bearer",
					res.StatusCode, body, res.Header.Get("WWW-Authenticate"))
			}
		})
	}
}

// 4. Every operation with a JSON body answers a body that breaks its
// structure with 400 and every broken field, and lets null through where the
// schema allows it. Operations that need a token get a valid one, so the
// body check, not the authentication, answers.
func TestBodiesThatBreakTheStructureAnswer400(t *testing.T) {
	contract := apitest.Load(t)
	base := startApp(t, testConfig(t, pgtest.NewDatabase(t), false), migrations.FS())
	token := registerAccount(t, contract, base, "body-cases@example.com").AccessToken

	for _, op := range contract.Operations() {
		if !op.HasJSONBody() {
			continue
		}
		for _, c := range op.BodyCases() {
			t.Run(op.Pattern()+"/"+c.Name, func(t *testing.T) {
				auth := token
				if op.Public {
					auth = ""
				}
				req := newRequest(t, op.Method, base+op.Target(), auth, c.Body)
				res, body := send(t, req)

				contract.CheckResponse(t, req, res)
				if c.Accepted {
					if res.StatusCode == http.StatusBadRequest {
						t.Errorf("%s = 400 %s, want it accepted", c.Body, body)
					}
					return
				}
				var p struct {
					Code   string                 `json:"code"`
					Errors []apitest.FieldProblem `json:"errors"`
				}
				if err := json.Unmarshal(body, &p); err != nil {
					t.Fatalf("decode %s: %v", body, err)
				}
				if res.StatusCode != http.StatusBadRequest || p.Code != "bad_request" || !slices.Equal(p.Errors, c.Fields) {
					t.Errorf("%s = %d %s, want 400 bad_request with %v", c.Body, res.StatusCode, body, c.Fields)
				}
			})
		}
	}
}

// Every operation's default response, the problem, declares the two headers
// a problem may carry: Retry-After and WWW-Authenticate. Each module declares
// its own Problem response (spec P2 3 item 13), so a module that leaves one
// out fails here.
func TestEveryProblemResponseDeclaresItsHeaders(t *testing.T) {
	contract := apitest.Load(t)

	for _, op := range contract.Operations() {
		for _, header := range []string{"Retry-After", "WWW-Authenticate"} {
			if !slices.Contains(op.ProblemHeaders, header) {
				t.Errorf("%s: the default response declares %q, want %s among them", op.Pattern(), op.ProblemHeaders, header)
			}
		}
	}
}

// authTokens is the AuthTokens answer.
type authTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// registerAccount signs up email through the app and returns its tokens.
func registerAccount(t *testing.T, contract *apitest.Contract, base, email string) authTokens {
	t.Helper()
	req := newRequest(t, http.MethodPost, base+"/api/v0/auth/register", "",
		[]byte(`{"email":"`+email+`","password":"Tr0ub4dor&3"}`))
	contract.CheckRequest(t, req)
	res, body := send(t, req)
	contract.CheckResponse(t, req, res)
	var tokens authTokens
	if res.StatusCode != http.StatusCreated || json.Unmarshal(body, &tokens) != nil {
		t.Fatalf("register %s = %d %s, want 201 with tokens", email, res.StatusCode, body)
	}
	return tokens
}

// newRequest builds a request with an optional bearer token and JSON body.
func newRequest(t *testing.T, method, url, token string, body []byte) *http.Request {
	t.Helper()
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

// send sends req and returns the response with its body read.
func send(t *testing.T, req *http.Request) (*http.Response, []byte) {
	t.Helper()
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	res.Body = io.NopCloser(bytes.NewReader(body))
	return res, body
}

func problemCode(t *testing.T, body []byte) string {
	t.Helper()
	var p httpserver.Problem
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("decode problem %s: %v", body, err)
	}
	return p.Code
}
