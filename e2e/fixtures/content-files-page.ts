/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createHash } from "node:crypto";
import { readFile as readDiskFile } from "node:fs/promises";
import path from "node:path";

import type { components } from "@nerve/api-client";
import { expect, type Locator, type Page, type Request, type TestInfo } from "@playwright/test";

import { addProjectMembers, inviteAndAccept } from "./api";
import { bearer, login, signInContext } from "./auth";
import { watchPage } from "./browser";
import type { PageFileStoryWorld } from "./file-story";
import { fileWorld } from "./file-world";
import { bodiesSentTo } from "./settings-pages";
import { filePixel, readFile } from "./storage";

type Prepared = components["schemas"]["PreparedAsset"];
type NewAsset = components["schemas"]["NewAsset"];
export type ContentFile = { name: string; type: string; bytes: Buffer };

const sha256 = (bytes: Uint8Array) => createHash("sha256").update(bytes).digest("hex");
export const textFile = (name: string, body: string): ContentFile => ({
  name,
  type: "text/plain",
  bytes: Buffer.from(body),
});
export const pngFile = (name: string, bytes: Buffer = filePixel): ContentFile => ({ name, type: "image/png", bytes });

/** The native chooser of region's attachment picker, then its one upload action. */
export async function chooseAttachment(page: Page, region: Locator, file: ContentFile) {
  const chooser = page.waitForEvent("filechooser");
  await region.getByLabel("Choose or replace file", { exact: true }).click();
  await (await chooser).setFiles({ name: file.name, mimeType: file.type, buffer: file.bytes });
  await region.getByRole("button", { name: "Upload file", exact: true }).click();
}

type Carried = { name: string; type: string; base64: string };
const carried = (files: readonly ContentFile[]): Carried[] =>
  files.map(({ name, type, bytes }) => ({ name, type, base64: bytes.toString("base64") }));

/**
 * A native DataTransfer drop of these exact bytes near the top left of target, as a person's drag would end there;
 * resolves with whether a handler took it (prevented the browser's default).
 */
export async function dropFiles(target: Locator, files: readonly ContentFile[]): Promise<boolean> {
  return target.evaluate((element, list: Carried[]) => {
    const transfer = new DataTransfer();
    for (const { name, type, base64 } of list)
      transfer.items.add(new File([Uint8Array.from(atob(base64), (c) => c.charCodeAt(0))], name, { type }));
    const box = element.getBoundingClientRect();
    const drop = new DragEvent("drop", {
      bubbles: true,
      cancelable: true,
      dataTransfer: transfer,
      clientX: box.left + Math.min(box.width / 2, 40),
      clientY: box.top + Math.min(box.height / 2, 12),
    });
    element.dispatchEvent(drop);
    return drop.defaultPrevented;
  }, carried(files));
}

/** A paste of exactly these clipboard entries and files; Firefox discards ClipboardEvent constructor data. */
export async function pasteInto(
  target: Locator,
  entries: Record<string, string>,
  files: readonly ContentFile[] = []
): Promise<boolean> {
  return target.evaluate(
    (element, { strings, list }: { strings: Record<string, string>; list: Carried[] }) => {
      const data = new DataTransfer();
      for (const [type, value] of Object.entries(strings)) data.setData(type, value);
      for (const { name, type, base64 } of list)
        data.items.add(new File([Uint8Array.from(atob(base64), (c) => c.charCodeAt(0))], name, { type }));
      const paste = new ClipboardEvent("paste", { bubbles: true, cancelable: true });
      Object.defineProperty(paste, "clipboardData", { value: data });
      element.dispatchEvent(paste);
      return paste.defaultPrevented;
    },
    { strings: entries, list: carried(files) }
  );
}

/** What the editor's own copy handler writes for its current selection: the strings a clipboard would carry. */
export async function copyFrom(target: Locator): Promise<Record<string, string>> {
  return target.evaluate((element) => {
    const data = new DataTransfer();
    const copy = new ClipboardEvent("copy", { bubbles: true, cancelable: true });
    Object.defineProperty(copy, "clipboardData", { value: data });
    element.dispatchEvent(copy);
    return Object.fromEntries([...data.types].map((type) => [type, data.getData(type)]));
  });
}

/** The browser's own download of the control holds exactly the stored bytes. */
export async function downloadedExactly(page: Page, control: Locator, file: ContentFile) {
  const started = page.waitForEvent("download");
  await control.click();
  const download = await started;
  expect(download.suggestedFilename()).toBe(file.name);
  const saved = await download.path();
  expect(sha256(await readDiskFile(saved))).toBe(sha256(file.bytes));
}

const pathOf = (request: Request) => new URL(request.url()).pathname;

/**
 * Hosts and roles come from the actual owner endpoints; only the browser writes the files under test. The page is the
 * owner's, or as "member" the other account's, a member of the workspace and of the project whose role the owner can
 * change.
 */
export async function contentFilesPage(world: PageFileStoryWorld, info: TestInfo, as: "owner" | "member" = "owner") {
  const { api, page, baseURL, db } = world;
  const files = await fileWorld(api, info);
  if (as === "member") {
    await inviteAndAccept(api, files.owner.token, files.workspace.slug, files.other, 15);
    await addProjectMembers(api, files.owner.token, files.project.id, [{ member_id: files.other.id, role: 15 }]);
  }
  const user = as === "owner" ? files.owner : files.other;
  const onboarded = await api.PATCH("/api/v0/me/profile", {
    headers: bearer(user.token),
    body: { is_onboarded: true },
  });
  expect(onboarded.response.status).toBe(200);
  await signInContext(page.context(), baseURL, await login(api, user.email));
  const watch = await watchPage(page);
  const requests: Request[] = [];
  page.on("request", (request) => requests.push(request));
  let inFlight = 0;
  let lastActivity = Date.now();
  const touch = (request: Request, change: number) => {
    if (!pathOf(request).startsWith("/api/v0/")) return;
    inFlight += change;
    lastActivity = Date.now();
  };
  page.on("request", (request) => touch(request, 1));
  page.on("requestfinished", (request) => touch(request, -1));
  page.on("requestfailed", (request) => touch(request, -1));
  /**
   * Waits until no API request has been in flight for quietMs. A write's canonical re-read re-mounts the body, which
   * reads its images again; a story that leaves the page before then would abort those reads.
   */
  const quiet = (quietMs = 500) =>
    expect
      .poll(() => inFlight === 0 && Date.now() - lastActivity >= quietMs, { timeout: 15_000, intervals: [100] })
      .toBe(true);
  const calls = (method: string, endpoint: string | RegExp) =>
    requests.filter(
      (request) =>
        request.method() === method &&
        (typeof endpoint === "string" ? pathOf(request) === endpoint : endpoint.test(pathOf(request)))
    );
  const prepared = await bodiesSentTo(page, "POST", "/api/v0/assets");
  /** The ordinal-th browser transfer: prepare, then one raw PUT of exactly these bytes, then one confirmation. */
  const transfer = async (
    ordinal: number,
    purpose: NewAsset["purpose"],
    scope: NewAsset["scope"],
    file: ContentFile
  ) => {
    await expect.poll(() => calls("POST", "/api/v0/assets").length).toBeGreaterThan(ordinal);
    const prepare = calls("POST", "/api/v0/assets")[ordinal]!;
    expect(prepared[ordinal]).toEqual({
      purpose,
      scope,
      name: file.name,
      content_type: file.type,
      size: file.bytes.length,
    });
    const answer = await prepare.response();
    expect(answer?.status()).toBe(201);
    const grant: Prepared = await answer!.json();
    if (world.bucket) expect(new URL(grant.upload.url).origin).toBe(world.bucket.connection.endpoint);
    else expect(grant.upload.url).toMatch(/^\/static\/local\//);
    await expect.poll(() => requests.filter((request) => request.method() === "PUT").length).toBeGreaterThan(ordinal);
    const put = requests.filter((request) => request.method() === "PUT")[ordinal]!;
    expect(put.url()).toBe(new URL(grant.upload.url, baseURL).href);
    // The receiving side records a body once it has ended; WebKit's debug API omits the File bytes themselves.
    const wirePuts = () => world.wire?.requests.filter((request) => request.method === "PUT") ?? [];
    if (world.wire) await expect.poll(() => wirePuts().length).toBeGreaterThan(ordinal);
    const received = wirePuts()[ordinal];
    const headers = received?.headers ?? (await put.allHeaders());
    expect(headers.authorization).toBeUndefined();
    expect(headers.cookie).toBeUndefined();
    expect(headers.referer).toBeUndefined();
    expect(headers["content-type"]).toBe(file.type);
    if (received) expect(received.headers["content-length"]).toBe(String(file.bytes.length));
    expect(received?.body ?? put.postDataBuffer()).toEqual(file.bytes);
    expect((await put.response())?.status()).toBe(world.uploadStatus);
    await expect.poll(() => calls("POST", `/api/v0/assets/${grant.asset.id}/complete`).length).toBe(1);
    const complete = calls("POST", `/api/v0/assets/${grant.asset.id}/complete`)[0]!;
    expect((await complete.response())?.status()).toBe(200);
    expect(requests.indexOf(prepare)).toBeLessThan(requests.indexOf(put));
    expect(requests.indexOf(put)).toBeLessThan(requests.indexOf(complete));
    return grant.asset.id;
  };
  /** The stored row, the stored object and an authorized read all hold the same bytes. */
  const persisted = async (id: string, file: ContentFile, bound = true) => {
    const [row] = await db.query<{ key: string; sha: string; uploaded: boolean; bound: boolean; deleted: boolean }>(
      `SELECT storage_metadata->'final'->>'key' AS key, storage_metadata->'final'->>'sha256' AS sha,
      is_uploaded AS uploaded, bound_at IS NOT NULL AS bound, is_deleted AS deleted FROM file_assets WHERE id=$1`,
      [id]
    );
    expect(row).toMatchObject({ sha: sha256(file.bytes), uploaded: true, bound, deleted: false });
    expect(await readFile(api, files.owner.token, id)).toEqual(file.bytes);
    if (world.bucket)
      expect(await world.bucket.inspect(row!.key)).toMatchObject({
        sha256: sha256(file.bytes),
        size: file.bytes.length,
        content_type: file.type,
      });
    else
      expect(await readDiskFile(path.join(world.nerve.storageRoot, "objects", row!.key, "data"))).toEqual(file.bytes);
    await info.attach(`file-${id}`, {
      body: JSON.stringify({ id, ...row, size: file.bytes.length, storage: world.bucket ? "s3" : "local" }),
      contentType: "application/json",
    });
  };
  const downloaded = (control: Locator, file: ContentFile) => downloadedExactly(page, control, file);
  /** expected names the failures a story caused on purpose; every other API failure fails the story. */
  const finish = async (expected: (failure: string) => boolean = () => false) => {
    expect(watch.oldApiRequests).toEqual([]);
    expect(watch.pageErrors).toEqual([]);
    expect(watch.cspViolations).toEqual([]);
    expect(watch.apiFailures.filter((failure) => !expected(failure))).toEqual([]);
    await info.attach("content-page-requests", {
      body: JSON.stringify(requests.map((request) => `${request.method()} ${pathOf(request)}`)),
      contentType: "application/json",
    });
  };
  return { ...files, watch, requests, calls, quiet, transfer, persisted, downloaded, finish };
}
