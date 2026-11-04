import { createHash } from "node:crypto";
import { readFile as readDiskFile } from "node:fs/promises";
import path from "node:path";

import type { components } from "@nerve/api-client";
import { expect, type Locator, type Request, type TestInfo } from "@playwright/test";

import { bearer, login, signInContext } from "./auth";
import { bodiesSentTo } from "./settings-pages";
import { watchPage } from "./browser";
import type { PageFileStoryWorld } from "./file-story";
import { fileWorld } from "./file-world";
import { completeFile, filePixel, readFile } from "./storage";

type Prepared = components["schemas"]["PreparedAsset"];
type NewAsset = components["schemas"]["NewAsset"];

/** An independently encoded red pixel, so a same-name replacement has genuinely different bytes. */
export const redPixel = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP4z8DwHwAFAAH/iZk9HQAAAABJRU5ErkJggg==",
  "base64"
);

/** Reuse API furnishing and session initialization; only the browser performs the image writes. */
export async function profileImagePage(world: PageFileStoryWorld, info: TestInfo, onboarded = true) {
  const { api, page, baseURL, db } = world;
  const owners = await fileWorld(api, info);
  if (onboarded) {
    const marked = await api.PATCH("/api/v0/me/profile", {
      headers: bearer(owners.owner.token),
      body: { is_onboarded: true },
    });
    expect(marked.response.status).toBe(200);
  }
  await signInContext(page.context(), baseURL, await login(api, owners.owner.email));
  // This is a deliberate test cookie, not authentication. Its presence on a real control fetch proves the probe works.
  await page
    .context()
    .addCookies([
      { name: "nerve_image_cookie_probe", value: "present", url: baseURL, httpOnly: true, sameSite: "Lax" },
    ]);
  const watch = await watchPage(page);
  const requests: Request[] = [];
  page.on("request", (request) => requests.push(request));
  const calls = (method: string, endpoint: string) =>
    requests.filter((request) => request.method() === method && new URL(request.url()).pathname === endpoint);
  const preparedBodies = await bodiesSentTo(page, "POST", "/api/v0/assets");
  const completeBodies = await bodiesSentTo(page, "POST", (pathname) =>
    /^\/api\/v0\/assets\/[^/]+\/complete$/.test(pathname)
  );
  const writes = new Map<string, unknown[]>();
  const register = async (method: string, endpoint: string) =>
    writes.set(`${method} ${endpoint}`, await bodiesSentTo(page, method, endpoint));
  await register("PATCH", "/api/v0/me");
  await register("PATCH", `/api/v0/workspaces/${owners.workspace.slug}`);
  await register("POST", `/api/v0/workspaces/${owners.workspace.slug}/projects`);
  const bodies = (method: string, endpoint: string) => {
    const recorded = writes.get(`${method} ${endpoint}`);
    if (!recorded) throw new Error(`Unobserved image owner ${method} ${endpoint}`);
    return recorded;
  };
  const probe = async () => {
    const sent = page.waitForRequest((request) => request.url() === `${baseURL}/api/v0/instance?image-cookie-probe=1`);
    expect(
      await page.evaluate(
        async () => (await fetch("/api/v0/instance?image-cookie-probe=1", { credentials: "include" })).status
      )
    ).toBe(200);
    const headers = world.wire
      ? world.wire.requests.find((request) => request.method === "GET" && request.path === "/api/v0/instance")?.headers
      : await (await sent).allHeaders();
    expect(headers?.cookie).toContain("nerve_image_cookie_probe=present");
  };
  const select = async (bytes = filePixel) => {
    await page
      .getByLabel("Choose or replace image", { exact: true })
      .setInputFiles({ name: "same-name.png", mimeType: "image/png", buffer: bytes });
    await expectDecoded(page.getByAltText("Selected image preview", { exact: true }));
  };
  const transfer = async (
    ordinal: number,
    purpose: NewAsset["purpose"],
    scope: NewAsset["scope"],
    bytes = filePixel
  ) => {
    const preparedRequests = calls("POST", "/api/v0/assets");
    expect(preparedRequests).toHaveLength(ordinal + 1);
    const prepare = preparedRequests[ordinal];
    if (!prepare) throw new Error("Missing preparation");
    expect(preparedBodies[ordinal]).toEqual({
      purpose,
      scope,
      name: "same-name.png",
      content_type: "image/png",
      size: bytes.length,
    });
    const preparedResponse = await prepare.response();
    expect(preparedResponse?.status()).toBe(201);
    if (!preparedResponse) throw new Error("Missing preparation response");
    const prepared: Prepared = await preparedResponse.json();
    // Inspect the original response: a fixture must never turn the local grant into an absolute URL first.
    if (world.bucket) expect(new URL(prepared.upload.url).origin).toBe(world.bucket.connection.endpoint);
    else expect(prepared.upload.url).toMatch(/^\/static\/local\//);
    const puts = requests.filter((request) => request.method() === "PUT");
    expect(puts).toHaveLength(ordinal + 1);
    const put = puts[ordinal];
    if (!put) throw new Error("Missing raw PUT");
    expect(put.url()).toBe(new URL(prepared.upload.url, baseURL).href);
    const received = world.wire?.requests.filter((request) => request.method === "PUT")[ordinal];
    const headers = received?.headers ?? (await put.allHeaders());
    expect(headers.authorization).toBeUndefined();
    expect(headers.cookie).toBeUndefined();
    expect(headers.referer).toBeUndefined();
    expect(headers["content-type"]).toBe("image/png");
    if (world.wire) {
      expect(received).toBeDefined();
      expect(received?.headers["content-length"]).toBe(String(bytes.length));
    } else expect((await put.sizes()).requestBodySize).toBe(bytes.length);
    expect(received?.body ?? put.postDataBuffer()).toEqual(bytes);
    expect((await put.response())?.status()).toBe(world.uploadStatus);
    const completions = calls("POST", `/api/v0/assets/${prepared.asset.id}/complete`);
    expect(completions).toHaveLength(1);
    const complete = completions[0];
    if (!complete) throw new Error("Missing confirmation");
    expect(completeBodies[ordinal]).toEqual({});
    expect((await complete.response())?.status()).toBe(200);
    expect(requests.indexOf(prepare)).toBeLessThan(requests.indexOf(put));
    expect(requests.indexOf(put)).toBeLessThan(requests.indexOf(complete));
    return prepared.asset.id;
  };
  const persisted = async (id: string, bytes = filePixel) => {
    const sha = createHash("sha256").update(bytes).digest("hex");
    const [row] = await db.query<{
      id: string;
      key: string;
      sha: string;
      uploaded: boolean;
      bound: boolean;
      deleted: boolean;
    }>(
      `SELECT id, storage_metadata->'final'->>'key' AS key, storage_metadata->'final'->>'sha256' AS sha,
      is_uploaded AS uploaded, bound_at IS NOT NULL AS bound, is_deleted AS deleted FROM file_assets WHERE id=$1`,
      [id]
    );
    expect(row).toMatchObject({ id, sha, uploaded: true, bound: true, deleted: false });
    if (!row) throw new Error("Missing bound asset row");
    expect(await readFile(api, owners.owner.token, id)).toEqual(bytes);
    if (world.bucket)
      expect(await world.bucket.inspect(row.key)).toMatchObject({
        sha256: sha,
        size: bytes.length,
        content_type: "image/png",
      });
    else expect(await readDiskFile(path.join(world.nerve.storageRoot, "objects", row.key, "data"))).toEqual(bytes);
    const before = await db.query("SELECT to_jsonb(a) AS row FROM file_assets a WHERE id=$1", [id]);
    expect(await completeFile(api, owners.owner.token, id)).toMatchObject({ id, is_bound: true });
    expect(await db.query("SELECT to_jsonb(a) AS row FROM file_assets a WHERE id=$1", [id])).toEqual(before);
    await info.attach(`image-${id}`, {
      body: JSON.stringify({
        ...row,
        size: bytes.length,
        storage: world.bucket ? "s3" : "local",
        repeatedCompleteUnchanged: true,
      }),
      contentType: "application/json",
    });
  };
  const finish = async () => {
    expect(watch.oldApiRequests).toEqual([]);
    expect(watch.pageErrors).toEqual([]);
    expect(watch.cspViolations).toEqual([]);
    expect(watch.apiFailures).toEqual([]);
    await info.attach("image-page-requests", {
      body: JSON.stringify(
        requests.map((request) => ({ method: request.method(), path: new URL(request.url()).pathname }))
      ),
      contentType: "application/json",
    });
  };
  return { ...owners, watch, calls, bodies, observeOwner: register, select, transfer, persisted, probe, finish };
}

export async function expectDecoded(image: Locator) {
  await expect(image).toBeVisible();
  await expect
    .poll(
      () =>
        image.evaluate(
          (element) => element instanceof HTMLImageElement && element.complete && element.naturalWidth > 0
        ),
      { message: "the browser decodes the actual image bytes" }
    )
    .toBe(true);
}
