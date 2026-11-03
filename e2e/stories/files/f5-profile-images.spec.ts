import { bearer, login } from "../../fixtures/auth";
import { fileWorld } from "../../fixtures/file-world";
import { expectImageBytes, imageRows, imageSHA } from "../../fixtures/profile-images";
import { filePixel, storedFile } from "../../fixtures/storage";
import { expect, test } from "../../fixtures/test";

test("F5 (API): personal images bind together, survive reauthentication, replace and clear atomically while foreign preparations stay hidden", async ({
  api,
  db,
  baseURL,
}, info) => {
  if (!baseURL) throw new Error("missing server origin");
  const w = await fileWorld(api, info);
  const upload = (purpose: "USER_AVATAR" | "USER_COVER", token = w.owner.token) =>
    storedFile(api, token, purpose, { kind: "user" }, filePixel, "image/png", "profile.png");
  const avatar = await upload("USER_AVATAR");
  const cover = await upload("USER_COVER");
  // A new session is still the same applicant; ownership never depends on a login/token ID.
  const session = await login(api, w.owner.email);
  const first = await api.PATCH("/api/v0/me", {
    headers: bearer(session.access_token),
    body: { avatar_asset_id: avatar.id, cover_image_asset_id: cover.id },
  });
  expect(first.response.status, JSON.stringify(first.error)).toBe(200);
  if (!first.data) throw new Error("profile binding returned no user");
  expect(first.data).toMatchObject({ avatar_asset_id: avatar.id, cover_image_asset_id: cover.id });
  await expectImageBytes(baseURL, first.data.avatar_url, first.data.avatar_url_expires_at);
  await expectImageBytes(baseURL, first.data.cover_image_url, first.data.cover_image_url_expires_at);
  expect(
    await db.query<{ valid: boolean }>(
      `SELECT u.avatar_asset_id=$2 AND u.cover_image_asset_id=$3
        AND a.bound_at=c.bound_at AND a.bound_at=u.updated_at AND a.updated_at=u.updated_at AND c.updated_at=u.updated_at
        AND a.created_by_id=u.id AND c.created_by_id=u.id AND a.user_id=u.id AND c.user_id=u.id
        AND a.scope_kind='user' AND c.scope_kind='user'
        AND a.storage_metadata->'final'->>'sha256'=$4 AND c.storage_metadata->'final'->>'sha256'=$4 AS valid
        FROM users u JOIN file_assets a ON a.id=u.avatar_asset_id JOIN file_assets c ON c.id=u.cover_image_asset_id WHERE u.id=$1`,
      [w.owner.id, avatar.id, cover.id, imageSHA]
    )
  ).toEqual([{ valid: true }]);
  const beforeRead = await imageRows(db, [avatar.id, cover.id]);
  const fresh = await api.GET("/api/v0/me", { headers: bearer(w.owner.token) });
  expect(fresh.response.status).toBe(200);
  expect(fresh.data).toMatchObject({ avatar_asset_id: avatar.id, cover_image_asset_id: cover.id });
  if (!fresh.data) throw new Error("profile read returned no user");
  await expectImageBytes(baseURL, fresh.data.avatar_url, fresh.data.avatar_url_expires_at);
  expect(await imageRows(db, [avatar.id, cover.id])).toEqual(beforeRead);

  const next = await upload("USER_AVATAR");
  const foreign = await upload("USER_COVER", w.other.token);
  const snapshot = () =>
    db.query(
      "SELECT to_jsonb(u) AS owner, (SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM file_assets a WHERE id=ANY($2::uuid[])) AS assets FROM users u WHERE id=$1",
      [w.owner.id, [avatar.id, cover.id, next.id, foreign.id]]
    );
  const beforeRefusal = await snapshot();
  const refused = await api.PATCH("/api/v0/me", {
    headers: bearer(w.owner.token),
    body: { avatar_asset_id: next.id, cover_image_asset_id: foreign.id },
  });
  expect(refused.response.status).toBe(404);
  expect(refused.error?.code).toBe("asset.not_found");
  expect(await snapshot()).toEqual(beforeRefusal);

  const replaced = await api.PATCH("/api/v0/me", {
    headers: bearer(w.owner.token),
    body: { avatar_asset_id: next.id },
  });
  expect(replaced.response.status).toBe(200);
  if (!replaced.data) throw new Error("replacement returned no user");
  expect(replaced.data.cover_image_asset_id).toBe(cover.id);
  await expectImageBytes(baseURL, replaced.data.avatar_url, replaced.data.avatar_url_expires_at);
  expect(await imageRows(db, [avatar.id])).toMatchObject([{ row: { is_deleted: true, deleted_by_id: w.owner.id } }]);
  expect(await imageRows(db, [next.id])).toMatchObject([{ row: { bound_at: expect.any(String), is_deleted: false } }]);
  const cleared = await api.PATCH("/api/v0/me", {
    headers: bearer(w.owner.token),
    body: { avatar_asset_id: null, cover_image_asset_id: null },
  });
  expect(cleared.response.status).toBe(200);
  expect(cleared.data).toMatchObject({
    avatar_asset_id: null,
    avatar_url: null,
    avatar_url_expires_at: null,
    cover_image_asset_id: null,
    cover_image_url: null,
    cover_image_url_expires_at: null,
  });
  expect(await imageRows(db, [cover.id, next.id])).toEqual([
    { row: expect.objectContaining({ is_deleted: true, deleted_by_id: w.owner.id }) },
    { row: expect.objectContaining({ is_deleted: true, deleted_by_id: w.owner.id }) },
  ]);
});
