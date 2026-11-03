package bootstrap

import (
	"fmt"
	"net/url"
	"testing"
	"uuid"
)

func TestInvitationImagePreview(t *testing.T) {
	w, _ := newAssetCompleteWorld(t, func(*assetComponents) {})
	data := assetTestPNG(t)
	logo := uploadedAssetHTTP(t, w, "alice", "WORKSPACE_LOGO", fmt.Sprintf(`{"kind":"workspace","id":%q}`, w.acme), "logo.png", "image/png", data).Asset.ID
	w.send(t, "alice", "PATCH", "/api/v0/workspaces/acme", fmt.Sprintf(`{"logo_asset_id":%q}`, logo), 200)
	link := invite(t, w.contract, w.base, w.tokens["alice"], "acme", "olga@example.com")
	path := "/api/v0/workspace-invitations/" + link.id.String() + "?token=" + url.QueryEscape(link.token)
	bad := newRequest(t, "GET", w.base+"/api/v0/workspace-invitations/"+link.id.String()+"?token=wrong", "", nil)
	probe := w.observe(bad)
	res, answer := send(t, bad)
	w.contract.CheckResponse(t, bad, res)
	probe.mu.Lock()
	queries := probe.sequence
	probe.mu.Unlock()
	if res.StatusCode != 404 || queries != 0 {
		t.Fatal("bad token accessed database", res.StatusCode, queries, string(answer))
	}
	body := w.send(t, "", "GET", path, "", 200)
	checkSignedImageReply(t, w, body, "workspace_logo", data)
	checkSignedImageReply(t, w, assetAPICall(t, w, "", "GET", path+"&asset_id="+uuid.New().String(), "", 200), "workspace_logo", data)
	// A declined link still previews; it cannot be accepted.
	answerInvitation(t, w.contract, w.base, w.tokens["olga"], "decline", link, 204)
	checkSignedImageReply(t, w, w.send(t, "", "GET", path, "", 200), "workspace_logo", data)
	w.send(t, "alice", "DELETE", "/api/v0/workspace-invitations/"+link.id.String(), "", 204)
	w.send(t, "", "GET", path, "", 404)
	link = invite(t, w.contract, w.base, w.tokens["alice"], "acme", "olga@example.com")
	joined := w.send(t, "olga", "POST", "/api/v0/workspace-invitations/"+link.id.String()+"/accept", fmt.Sprintf(`{"token":%q}`, link.token), 200)
	checkImageReply(t, w, joined, "logo", logo, data)
	w.send(t, "", "GET", "/api/v0/workspace-invitations/"+link.id.String()+"?token="+url.QueryEscape(link.token), "", 404)
	last := invite(t, w.contract, w.base, w.tokens["alice"], "acme", "new@example.com")
	w.send(t, "alice", "DELETE", "/api/v0/workspaces/acme", "", 204)
	w.send(t, "", "GET", "/api/v0/workspace-invitations/"+last.id.String()+"?token="+url.QueryEscape(last.token), "", 404)
}
