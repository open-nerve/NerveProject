import { bearer } from "../../fixtures/auth";
import { fileWorld } from "../../fixtures/file-world";
import { expectImageBytes, imageRows, imageSHA } from "../../fixtures/profile-images";
import { filePixel, storedFile } from "../../fixtures/storage";
import { expect, test } from "../../fixtures/test";

test("F7 (API): one project POST binds its final cover and complete graph, refused creation leaves nothing, and stable presets replace custom covers", async ({
  api,
  db,
  baseURL,
}, info) => {
  if (!baseURL) throw new Error("missing server origin");
  const w = await fileWorld(api, info);
  const slug = w.workspace.slug;
  const prepared = await storedFile(
    api,
    w.owner.token,
    "PROJECT_COVER",
    { kind: "new_project", id: w.workspace.id },
    filePixel,
    "image/png"
  );
  const made = await api.POST("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug } },
    headers: bearer(w.owner.token),
    body: { name: "Created with cover", identifier: "COVER", cover_image_asset_id: prepared.id },
  });
  expect(made.response.status, JSON.stringify(made.error)).toBe(201);
  if (!made.data) throw new Error("created project missing");
  const project = made.data;
  expect(project).toMatchObject({ cover_image_asset_id: prepared.id, cover_builtin_id: null });
  await expectImageBytes(baseURL, project.cover_image_url, project.cover_image_url_expires_at);
  expect(
    await db.query<{ valid: boolean }>(
      `SELECT p.cover_image_asset_id=$2 AND a.project_id=p.id AND a.workspace_id=p.workspace_id AND a.scope_kind='project'
      AND a.bound_at=p.created_at AND a.updated_at=p.created_at AND a.storage_metadata->'final'->>'sha256'=$3
      AND (SELECT count(*) FROM project_members m WHERE m.project_id=p.id AND m.created_at=p.created_at)=1
      AND (SELECT count(*) FROM project_user_properties m WHERE m.project_id=p.id AND m.created_at=p.created_at)=1
      AND (SELECT count(*) FROM states s WHERE s.project_id=p.id AND s.created_at=p.created_at)=6 AS valid
      FROM projects p JOIN file_assets a ON a.id=p.cover_image_asset_id WHERE p.id=$1`,
      [project.id, prepared.id, imageSHA]
    )
  ).toEqual([{ valid: true }]);
  const snapshot = () =>
    db.query(
      `SELECT (SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM projects p WHERE workspace_id=$1) AS projects,
    (SELECT jsonb_agg(to_jsonb(a) ORDER BY id) FROM file_assets a WHERE workspace_id=$1) AS assets`,
      [w.workspace.id]
    );
  const before = await snapshot();
  const refused = await api.POST("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug } },
    headers: bearer(w.owner.token),
    body: { name: "Cannot reuse", identifier: "REUSED", cover_image_asset_id: prepared.id },
  });
  expect(refused.response.status).toBe(404);
  expect(refused.error?.code).toBe("asset.not_found");
  expect(await snapshot()).toEqual(before);
  const preset = await api.PATCH("/api/v0/projects/{project_id}", {
    params: { path: { project_id: project.id } },
    headers: bearer(w.owner.token),
    body: { cover_builtin_id: "cover-29" },
  });
  expect(preset.response.status).toBe(200);
  expect(preset.data).toMatchObject({
    cover_builtin_id: "cover-29",
    cover_image_asset_id: null,
    cover_image_url: null,
    cover_image_url_expires_at: null,
  });
  expect(await imageRows(db, [prepared.id])).toMatchObject([{ row: { is_deleted: true, deleted_by_id: w.owner.id } }]);
  const next = await storedFile(
    api,
    w.owner.token,
    "PROJECT_COVER",
    { kind: "project", id: project.id },
    filePixel,
    "image/png"
  );
  const again = await api.PATCH("/api/v0/projects/{project_id}", {
    params: { path: { project_id: project.id } },
    headers: bearer(w.owner.token),
    body: { cover_image_asset_id: next.id },
  });
  expect(again.response.status).toBe(200);
  if (!again.data) throw new Error("custom replacement missing");
  expect(again.data.cover_builtin_id).toBeNull();
  await expectImageBytes(baseURL, again.data.cover_image_url, again.data.cover_image_url_expires_at);
  const beforeInvalid = await snapshot();
  const invalid = await api.PATCH("/api/v0/projects/{project_id}", {
    params: { path: { project_id: project.id } },
    headers: bearer(w.owner.token),
    body: { cover_builtin_id: "cover-30" },
  });
  expect(invalid.response.status).toBe(422);
  expect(await snapshot()).toEqual(beforeInvalid);
});
