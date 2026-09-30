package bootstrap

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The invitations on the wired app and a real database: what the matrix
// and the modules' tests cannot see.

// invitationLink is an invitation's id and its link's token.
type invitationLink struct {
	id    uuid.UUID
	token string
}

// invite has admin, with the bearer token, invite email to the workspace
// slug as a member, and returns the invitation's link.
func invite(t *testing.T, contract *apitest.Contract, base, admin, slug, email string) invitationLink {
	t.Helper()
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/"+slug+"/invitations", admin,
		`{"invitations":[{"email":"`+email+`","role":15}]}`)
	var list struct {
		Data []struct {
			ID    uuid.UUID `json:"id"`
			Token string    `json:"token"`
		} `json:"data"`
	}
	if status != http.StatusCreated {
		t.Fatalf("inviting %s = %d %s", email, status, body)
	}
	decodeAnswer(t, body, &list)
	return invitationLink{list.Data[0].ID, list.Data[0].Token}
}

// answerInvitation has the bearer accept or decline the invitation of
// link, wanting status.
func answerInvitation(t *testing.T, contract *apitest.Contract, base, bearer, answer string, link invitationLink, status int) {
	t.Helper()
	got, body := call(t, contract, http.MethodPost, base+"/api/v0/workspace-invitations/"+link.id.String()+"/"+answer, bearer,
		`{"token":"`+link.token+`"}`)
	if got != status {
		t.Fatalf("%s %s = %d %s, want %d", answer, link.id, got, body, status)
	}
}

// The server never stores an invitation's token: it computes it again from
// the invitation's id whenever it answers or checks one (M3 design 3.8,
// 8.1). After each step that does, the rows of workspace_member_invites,
// one for each invitation, hold neither token in any form a column could
// keep it (expectNoTokenStored).
func TestTheInvitationTokenIsNeverStored(t *testing.T) {
	url := pgtest.NewDatabase(t)
	base, pool := startApp(t, testConfig(t, url, false), migrations.FS()), openPool(t, url)
	contract := apitest.Load(t)
	admin := registerAccount(t, contract, base, "admin@example.com").AccessToken
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", admin, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/invitations", admin,
		`{"invitations":[{"email":"carol@example.com","role":15},{"email":"dave@example.com","role":5}]}`)
	if status != http.StatusCreated {
		t.Fatalf("inviting carol and dave = %d %s", status, body)
	}
	var list struct {
		Data []struct {
			ID    uuid.UUID `json:"id"`
			Token string    `json:"token"`
		} `json:"data"`
	}
	decodeAnswer(t, body, &list)
	if len(list.Data) != 2 {
		t.Fatalf("inviting carol and dave answers %d invitations, want 2: %s", len(list.Data), body)
	}
	links := make([]invitationLink, len(list.Data))
	for i, inv := range list.Data {
		links[i] = invitationLink{inv.ID, inv.Token}
	}
	expectNoTokenStored(t, pool, links)

	if status, body := call(t, contract, http.MethodPatch, base+"/api/v0/workspace-invitations/"+links[0].id.String(), admin, `{"role":20}`); status != http.StatusOK {
		t.Fatalf("changing carol's role = %d %s", status, body)
	}
	expectNoTokenStored(t, pool, links)

	carol := registerAccount(t, contract, base, "carol@example.com").AccessToken
	answerInvitation(t, contract, base, carol, "accept", links[0], http.StatusOK)
	dave := registerAccount(t, contract, base, "dave@example.com").AccessToken
	answerInvitation(t, contract, base, dave, "decline", links[1], http.StatusNoContent)
	expectNoTokenStored(t, pool, links)
}

// expectNoTokenStored fails unless workspace_member_invites has one row for
// each of links, and no row, as row_to_json writes it, holds a link's token
// whole, its 22 characters after nrv_inv_, or its tag in hex (a bytea
// column) or in standard base64 (a text column of the other alphabet).
func expectNoTokenStored(t *testing.T, pool *pgxpool.Pool, links []invitationLink) {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT row_to_json(i)::text FROM workspace_member_invites i")
	if err != nil {
		t.Fatal(err)
	}
	stored, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != len(links) {
		t.Fatalf("workspace_member_invites has %d rows, want %d: %q", len(stored), len(links), stored)
	}
	for _, l := range links {
		tag, ok := workspacedomain.ParseToken(l.token)
		if !ok {
			t.Fatalf("the token %q of %s does not parse", l.token, l.id)
		}
		for _, form := range []string{l.token, strings.TrimPrefix(l.token, "nrv_inv_"), hex.EncodeToString(tag[:]), base64.RawStdEncoding.EncodeToString(tag[:])} {
			for _, row := range stored {
				if strings.Contains(row, form) {
					t.Errorf("a row holds the token of %s as %q: %s", l.id, form, row)
				}
			}
		}
	}
}

// While sign-up is off, a registration goes on only with the link of a
// pending invitation to its address, normalized (M3 design 3.8): on a
// second app on the same database and key, with sign-up off. Every other
// case answers the one 403 identity.signup_disabled, byte for byte, whatever
// is wrong (8.2): no invitation, a token not the invitation's, an id no
// invitation has, a deleted invitation, an accepted one (its address has an
// account, and that is not what it says), a declined one, another address.
// Registering does not accept the invitation.
func TestRegisteringWithAnInvitationWhileSignupIsOff(t *testing.T) {
	url, keyFile := pgtest.NewDatabase(t), writeFile(t, testKeyPEM)
	configured := func(signup bool) config.Config {
		cfg := testConfig(t, url, false)
		cfg.Auth.JWT.PrivateKeyFile, cfg.Auth.SignupEnabled = keyFile, signup
		return cfg
	}
	contract := apitest.Load(t)
	open := startApp(t, configured(true), migrations.FS())
	admin := registerAccount(t, contract, open, "admin@example.com").AccessToken
	if status, body := call(t, contract, http.MethodPost, open+"/api/v0/workspaces", admin, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	links := map[string]invitationLink{}
	for _, name := range []string{"carol", "dave", "erin", "frank", "gina"} {
		links[name] = invite(t, contract, open, admin, "acme", name+"@example.com")
	}
	dave := registerAccount(t, contract, open, "dave@example.com").AccessToken
	answerInvitation(t, contract, open, dave, "decline", links["dave"], http.StatusNoContent)
	erin := registerAccount(t, contract, open, "erin@example.com").AccessToken
	answerInvitation(t, contract, open, erin, "accept", links["erin"], http.StatusOK)
	if status, body := call(t, contract, http.MethodDelete, open+"/api/v0/workspace-invitations/"+links["frank"].id.String(), admin, ""); status != http.StatusNoContent {
		t.Fatalf("deleting frank's invitation = %d %s", status, body)
	}
	closed := startApp(t, configured(false), migrations.FS())
	register := func(email, invitation string) wholeAnswer {
		t.Helper()
		req := newRequest(t, http.MethodPost, closed+"/api/v0/auth/register", "",
			[]byte(`{"email":"`+email+`","password":"Tr0ub4dor&3"`+invitation+`}`))
		contract.CheckRequest(t, req)
		res, body := send(t, req)
		contract.CheckResponse(t, req, res)
		return wholeAnswerOf(res, body)
	}
	naming := func(l invitationLink) string {
		return `,"invitation":{"id":"` + l.id.String() + `","token":"` + l.token + `"}`
	}
	nobodys := uuid.NewV7()

	refused := register("zoe@example.com", "")
	if refused.status != http.StatusForbidden || problemCode(t, []byte(refused.body)) != "identity.signup_disabled" {
		t.Fatalf("registering without an invitation = %+v, want 403 identity.signup_disabled", refused)
	}
	for _, tt := range []struct{ name, email, invitation string }{
		{"a token not the invitation's", "carol@example.com", naming(invitationLink{links["carol"].id, links["gina"].token})},
		{"an id no invitation has", "carol@example.com", naming(invitationLink{nobodys, invitationToken(t, nobodys)})},
		{"a deleted invitation", "frank@example.com", naming(links["frank"])},
		{"an accepted invitation", "erin@example.com", naming(links["erin"])},
		{"a declined invitation", "dave@example.com", naming(links["dave"])},
		{"another address", "zoe@example.com", naming(links["gina"])},
	} {
		if got := register(tt.email, tt.invitation); got != refused {
			t.Errorf("registering with %s = %+v, want what one without an invitation gets: %+v", tt.name, got, refused)
		}
	}
	if got := register(" Carol@Example.com ", naming(links["carol"])); got.status != http.StatusCreated {
		t.Errorf("registering carol with her invitation = %+v, want 201", got)
	}
	carol := links["carol"]
	if status, body := call(t, contract, http.MethodGet, closed+"/api/v0/workspace-invitations/"+carol.id.String()+"?token="+carol.token, "", ""); status != http.StatusOK ||
		strings.Contains(body, `"declined":true`) {
		t.Errorf("carol's invitation after she registered = %d %s, want it pending", status, body)
	}
}

// The link's token never reaches the logs, at any level (M3 design 8.1).
// The access line of each request has the path of the public view and no
// query, whether the token is right, wrong or missing, and so has the
// error line of a request the server fails (its table renamed away: 500).
// The recover middleware's panic line writes the path as both do; no
// request here panics.
func TestTheInvitationLinksTokenIsNotLogged(t *testing.T) {
	var logs lockedBuffer
	url := pgtest.NewDatabase(t)
	base := startAppLogging(t, testConfig(t, url, false), migrations.FS(),
		slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	contract := apitest.Load(t)
	admin := registerAccount(t, contract, base, "admin@example.com").AccessToken
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", admin, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	link := invite(t, contract, base, admin, "acme", "carol@example.com")
	token, path := link.token, "/api/v0/workspace-invitations/"+link.id.String()

	for _, tt := range []struct {
		query string
		want  int
	}{{"?token=" + token, http.StatusOK}, {"?token=" + strings.ToUpper(token), http.StatusNotFound}, {"", http.StatusBadRequest}} {
		req := newRequest(t, http.MethodGet, base+path+tt.query, "", nil)
		if res, body := send(t, req); res.StatusCode != tt.want {
			t.Errorf("GET %s = %d %s, want %d", path+tt.query, res.StatusCode, body, tt.want)
		}
	}
	if _, err := openPool(t, url).Exec(context.Background(), "ALTER TABLE workspace_member_invites RENAME TO held"); err != nil {
		t.Fatal(err)
	}
	if res, body := send(t, newRequest(t, http.MethodGet, base+path+"?token="+token, "", nil)); res.StatusCode != http.StatusInternalServerError {
		t.Errorf("GET %s with the right token and no table = %d %s, want 500", path, res.StatusCode, body)
	}

	got := logs.String()
	access, failed := 0, 0
	for _, line := range strings.Split(got, "\n") {
		switch {
		case !strings.Contains(line, " path="+path+" "):
		case strings.Contains(line, `msg="http request"`):
			access++
		case strings.Contains(line, "level=ERROR"):
			failed++
		}
	}
	if access != 4 || failed != 1 {
		t.Fatalf("the logs have %d access lines and %d error lines of the link, want 4 and 1:\n%s", access, failed, got)
	}
	for _, leak := range []string{"token=", token, token[len("nrv_inv_"):], "nrv_inv_"} {
		if strings.Contains(got, leak) {
			t.Errorf("the logs hold %q:\n%s", leak, got)
		}
	}
}
