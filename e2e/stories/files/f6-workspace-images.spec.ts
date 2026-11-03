import { accept, changeRole, invite, inviteAndAccept, listMembers } from "../../fixtures/api";
import { bearer, newAccount } from "../../fixtures/auth";
import { fileWorld } from "../../fixtures/file-world";
import { expectImageBytes, imageRows, imageSHA } from "../../fixtures/profile-images";
import { filePixel, storedFile } from "../../fixtures/storage";
import { expect, test } from "../../fixtures/test";

test("F6 (API): workspace logos follow current admin authority and members and invitation previews receive only their authorized images", async ({
  api,
  db,
  baseURL,
}, info) => {
  if (!baseURL) throw new Error("missing server origin");
  const w = await fileWorld(api, info);
  const slug = w.workspace.slug;
  await inviteAndAccept(api, w.owner.token, slug, w.other, 20);
  const logo = await storedFile(
    api,
    w.owner.token,
    "WORKSPACE_LOGO",
    { kind: "workspace", id: w.workspace.id },
    filePixel,
    "image/png"
  );
  const avatar = await storedFile(api, w.owner.token, "USER_AVATAR", { kind: "user" }, filePixel, "image/png");
  expect(
    (await api.PATCH("/api/v0/me", { headers: bearer(w.owner.token), body: { avatar_asset_id: avatar.id } })).response
      .status
  ).toBe(200);
  const bound = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    headers: bearer(w.owner.token),
    body: { logo_asset_id: logo.id },
  });
  expect(bound.response.status).toBe(200);
  if (!bound.data) throw new Error("logo update returned no workspace");
  await expectImageBytes(baseURL, bound.data.logo_url, bound.data.logo_url_expires_at);
  expect(
    await db.query<{ valid: boolean }>(
      `SELECT w.logo_asset_id=$2 AND a.workspace_id=w.id AND a.scope_kind='workspace'
      AND a.bound_at=w.updated_at AND a.updated_at=w.updated_at AND a.created_by_id=$3
      AND a.storage_metadata->'final'->>'sha256'=$4 AS valid
      FROM workspaces w JOIN file_assets a ON a.id=w.logo_asset_id WHERE w.id=$1`,
      [w.workspace.id, logo.id, w.owner.id, imageSHA]
    )
  ).toEqual([{ valid: true }]);
  await changeRole(api, w.other.token, slug, w.owner.id, 15);
  const before = await imageRows(db, [logo.id]);
  const kept = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    headers: bearer(w.other.token),
    body: { logo_asset_id: logo.id, name: "New admin" },
  });
  expect(kept.response.status).toBe(200);
  if (!kept.data) throw new Error("current reference was rejected for the new admin");
  await expectImageBytes(baseURL, kept.data.logo_url, kept.data.logo_url_expires_at);
  expect(await imageRows(db, [logo.id])).toEqual(before);
  const snapshot = () =>
    db.query(
      "SELECT to_jsonb(w) AS owner,(SELECT to_jsonb(a) FROM file_assets a WHERE id=$2) AS asset FROM workspaces w WHERE id=$1",
      [w.workspace.id, logo.id]
    );
  const beforeRefusal = await snapshot();
  const refused = await api.PATCH("/api/v0/workspaces/{slug}", {
    params: { path: { slug } },
    headers: bearer(w.owner.token),
    body: { logo_asset_id: null },
  });
  expect(refused.response.status).toBe(403);
  expect(refused.error?.code).toBe("forbidden");
  expect(await snapshot()).toEqual(beforeRefusal);
  const member = (await listMembers(api, w.other.token, slug)).find((m) => m.member.id === w.owner.id)?.member;
  if (!member) throw new Error("missing public member profile");
  await expectImageBytes(baseURL, member.avatar_url, member.avatar_url_expires_at);
  const invited = await newAccount(api, info, "image-invitee");
  const [link] = await invite(api, w.other.token, slug, [{ email: invited.email, role: 15 }]);
  if (!link) throw new Error("invitation missing");
  const preview = await api.GET("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: link.id }, query: { token: link.token } },
  });
  expect(preview.response.status).toBe(200);
  if (!preview.data) throw new Error("invitation preview missing");
  await expectImageBytes(baseURL, preview.data.workspace_logo_url, preview.data.workspace_logo_url_expires_at);
  const accepted = await accept(api, invited.token, link);
  await expectImageBytes(baseURL, accepted.logo_url, accepted.logo_url_expires_at);
  expect(
    (
      await api.GET("/api/v0/workspace-invitations/{invitation_id}", {
        params: { path: { invitation_id: link.id }, query: { token: link.token } },
      })
    ).response.status
  ).toBe(404);
  expect(await imageRows(db, [logo.id])).toEqual(before);
});
