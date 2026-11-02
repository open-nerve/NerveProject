import type { components } from "@nerve/api-client";
import { expect } from "@playwright/test";

import type { Api } from "./api";
import { bearer } from "./auth";
import type { Database } from "./db";

type NewAsset = components["schemas"]["NewAsset"];
export type Asset = components["schemas"]["Asset"];
type PreparedAsset = components["schemas"]["PreparedAsset"];
type Upload = components["schemas"]["Upload"];
type Content = components["schemas"]["Content"];
type DuplicateAsset = components["schemas"]["DuplicateAsset"];

/** One real PNG pixel; the fixture makes no claim about full pixel decoding by the server. */
export const filePixel = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAC0lEQVR4nGNgAAIAAAUAAXpeqz8AAAAASUVORK5CYII=",
  "base64"
);

export async function prepareFile(api: Api, token: string, body: NewAsset): Promise<PreparedAsset> {
  const { data, error, response } = await api.POST("/api/v0/assets", { body, headers: bearer(token) });
  expect(response.status, JSON.stringify(error)).toBe(201);
  if (!data) throw new Error("asset preparation returned no asset");
  // Resolve a local relative capability against the actual API response origin.
  return { ...data, upload: { ...data.upload, url: new URL(data.upload.url, response.url).href } };
}

/** Raw capabilities use an independent transport with no business bearer or browser cookies. */
export function sendFile(upload: Upload, bytes: Uint8Array): Promise<Response> {
  return fetch(upload.url, {
    method: upload.method,
    headers: upload.headers,
    credentials: "omit",
    body: new Uint8Array(bytes),
    signal: AbortSignal.timeout(15_000),
  });
}

export async function uploadFile(upload: Upload, bytes: Uint8Array): Promise<void> {
  const response = await sendFile(upload, bytes);
  expect(response.status, await response.text()).toBe(204);
}

export async function completeFile(api: Api, token: string, id: string): Promise<Asset> {
  const { data, error, response } = await api.POST("/api/v0/assets/{id}/complete", {
    params: { path: { id } },
    body: {},
    headers: bearer(token),
  });
  expect(response.status, JSON.stringify(error)).toBe(200);
  if (!data) throw new Error("complete returned no asset");
  return data;
}

export async function fileContent(api: Api, token: string, id: string): Promise<Content> {
  const { data, error, response } = await api.GET("/api/v0/assets/{id}/content", {
    params: { path: { id } },
    headers: bearer(token),
  });
  expect(response.status, JSON.stringify(error)).toBe(200);
  if (!data) throw new Error("content returned no capability");
  return { ...data, url: new URL(data.url, response.url).href };
}

export async function readFile(api: Api, token: string, id: string): Promise<Buffer> {
  const content = await fileContent(api, token, id);
  const response = await fetch(content.url, { credentials: "omit", signal: AbortSignal.timeout(15_000) });
  expect(response.status).toBe(200);
  return Buffer.from(await response.arrayBuffer());
}

export async function duplicateFile(api: Api, token: string, id: string, body: DuplicateAsset): Promise<Asset> {
  const { data, error, response } = await api.POST("/api/v0/assets/{id}/duplicate", {
    params: { path: { id } },
    body,
    headers: bearer(token),
  });
  expect(response.status, JSON.stringify(error)).toBe(201);
  if (!data) throw new Error("duplicate returned no asset");
  return data;
}

/** API preparation, actual PUT, confirmation and byte equality, without a binding fixture. */
export async function storedFile(
  api: Api,
  token: string,
  purpose: Asset["purpose"],
  scope: Asset["scope"],
  bytes: Buffer = Buffer.from("real attachment bytes"),
  contentType = "text/plain",
  name = "fixture.txt"
): Promise<Asset> {
  const prepared = await prepareFile(api, token, {
    purpose,
    scope,
    name,
    size: bytes.length,
    content_type: contentType,
  });
  await uploadFile(prepared.upload, bytes);
  const asset = await completeFile(api, token, prepared.asset.id);
  expect(asset.state).toBe("uploaded");
  expect(asset.is_bound).toBe(false);
  expect(await readFile(api, token, asset.id)).toEqual(bytes);
  return asset;
}

/** Both catalog edges below an issue have an independently uploaded object. */
export async function furnishIssueFiles(api: Api, token: string, issueId: string, commentId: string): Promise<void> {
  await storedFile(api, token, "ISSUE_ATTACHMENT", { kind: "issue", id: issueId });
  await storedFile(api, token, "COMMENT_ATTACHMENT", { kind: "comment", id: commentId });
}

/** Wait on persisted real deadlines, never fast-forward a runtime safety window. */
export async function waitForFileSafety(db: Database, ids: string[]): Promise<void> {
  expect(ids.length).toBeGreaterThan(0);
  const [row] = await db.query<{ deadline: string | null }>(
    `SELECT (extract(epoch FROM max((r.value->>'safe_after')::timestamptz))*1000)::text AS deadline
       FROM file_assets a CROSS JOIN LATERAL jsonb_array_elements(
         jsonb_build_array(a.storage_metadata->'staging', a.storage_metadata->'final') || (a.storage_metadata->'candidates')) r
       WHERE a.id=ANY($1::uuid[])`,
    [ids]
  );
  if (!row?.deadline) throw new Error("the safety fixture has no persisted object deadline");
  const deadline = Math.ceil(Number(row.deadline));
  await expect
    .poll(() => Date.now(), {
      message: "actual persisted upload/lease safety has elapsed",
      timeout: Math.max(1000, deadline - Date.now() + 10_000),
      intervals: [100, 250, 1000],
    })
    .toBeGreaterThan(deadline);
}
