import { randomUUID } from "node:crypto";

import {
  addProjectMembers,
  createProject,
  inviteAndAccept,
  projectMemberWrites,
  projectMembershipOf,
} from "../../fixtures/api";
import { bearer } from "../../fixtures/auth";
import { fileWorld } from "../../fixtures/file-world";
import { createIssue } from "../../fixtures/issues";
import { fileContent, filePixel, readFile, storedFile } from "../../fixtures/storage";
import { expect, test } from "../../fixtures/test";

test("F13 (API): preparation ownership and current host rights govern every operation; source denial precedes target denial, and issued capabilities keep their stated expiry", async ({
  api,
  db,
}, info) => {
  const w = await fileWorld(api, info);
  await inviteAndAccept(api, w.owner.token, w.workspace.slug, w.other, 15);
  await addProjectMembers(api, w.owner.token, w.project.id, [{ member_id: w.other.id, role: 15 }]);
  const ownIssue = await createIssue(api, w.other.token, w.project.id, { name: "Member's file" });
  const shared = await storedFile(api, w.other.token, "ISSUE_ATTACHMENT", { kind: "issue", id: ownIssue.id });
  expect(shared.is_bound).toBe(true);
  expect(await readFile(api, w.owner.token, shared.id)).toEqual(Buffer.from("real attachment bytes"));
  // An uploaded but never referenced image remains a private preparation.
  const asset = await storedFile(
    api,
    w.other.token,
    "ISSUE_DESCRIPTION",
    { kind: "issue", id: ownIssue.id },
    filePixel,
    "image/png",
    "unreferenced.png"
  );
  expect(asset.is_bound).toBe(false);
  const headers = bearer(w.owner.token);
  const get = (id: string) => api.GET("/api/v0/assets/{id}", { params: { path: { id } }, headers });
  const hidden = await get(asset.id);
  const absent = await get(randomUUID());
  expect(hidden.response.status).toBe(404);
  expect(hidden.error).toEqual(absent.error);
  // Even the workspace/project admin cannot claim another applicant's preparation.
  const before = await db.query("SELECT count(*)::int AS n FROM file_assets");
  const deniedCopy = await api.POST("/api/v0/assets/{id}/duplicate", {
    params: { path: { id: asset.id } },
    headers,
    body: { purpose: "USER_AVATAR", scope: { kind: "user" } },
  });
  expect(deniedCopy.response.status).toBe(404);
  expect(deniedCopy.error?.code).toBe("asset.not_found");
  expect(
    (await api.DELETE("/api/v0/assets/{id}", { params: { path: { id: asset.id } }, headers })).response.status
  ).toBe(404);
  expect(
    (await api.POST("/api/v0/assets/{id}/complete", { params: { path: { id: asset.id } }, body: {}, headers })).response
      .status
  ).toBe(404);
  expect(
    (await api.GET("/api/v0/assets/{id}/content", { params: { path: { id: asset.id } }, headers })).response.status
  ).toBe(404);
  expect(await db.query("SELECT count(*)::int AS n FROM file_assets")).toEqual(before);
  const issued = await fileContent(api, w.other.token, asset.id);
  const membership = await projectMembershipOf(api, w.owner.token, w.project.id, w.other.id);
  expect(await projectMemberWrites(api, w.project.id).change(w.owner.token, membership, 5)).toEqual({ status: 200 });
  const memberHeaders = bearer(w.other.token);
  expect(
    (await api.GET("/api/v0/assets/{id}", { params: { path: { id: asset.id } }, headers: memberHeaders })).response
      .status
  ).toBe(200);
  expect(
    (
      await api.POST("/api/v0/assets/{id}/complete", {
        params: { path: { id: asset.id } },
        body: {},
        headers: memberHeaders,
      })
    ).response.status
  ).toBe(403);
  expect(await projectMemberWrites(api, w.project.id).leave(w.other.token)).toEqual({ status: 204 });
  expect(
    (await api.GET("/api/v0/assets/{id}/content", { params: { path: { id: asset.id } }, headers: memberHeaders }))
      .response.status
  ).toBe(404);
  const target = await createProject(api, w.owner.token, w.workspace.slug, {
    name: "Forbidden target",
    identifier: "TARGET",
  });
  await addProjectMembers(api, w.owner.token, target.id, [{ member_id: w.other.id, role: 15 }]);
  const targetAlone = await api.POST("/api/v0/assets", {
    headers: memberHeaders,
    body: {
      purpose: "PROJECT_COVER",
      scope: { kind: "project", id: target.id },
      name: "cover.png",
      size: 1,
      content_type: "image/png",
    },
  });
  expect(targetAlone.response.status).toBe(403);
  const bothDenied = await api.POST("/api/v0/assets/{id}/duplicate", {
    params: { path: { id: asset.id } },
    headers: memberHeaders,
    body: { purpose: "PROJECT_COVER", scope: { kind: "project", id: target.id } },
  });
  expect(bothDenied.response.status).toBe(404);
  expect(bothDenied.error?.code).toBe("asset.not_found");
  const retained = await fetch(issued.url, { credentials: "omit" });
  expect(retained.status).toBe(200);
  expect(Buffer.from(await retained.arrayBuffer())).toEqual(filePixel);
  expect(Date.parse(issued.expires_at)).toBeGreaterThan(Date.now());
});
