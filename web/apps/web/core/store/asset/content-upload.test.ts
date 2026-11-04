/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { track, until } from "@/lib/auth/fake-time";
import { contentAssetTab, imageFile } from "./fake-assets";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
const close: (() => void)[] = [];
beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  close.splice(0).forEach((dispose) => dispose());
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function fixture() {
  const tab = await contentAssetTab();
  close.push(() => tab.app.dispose());
  return tab;
}

const mappings = [
  { kind: "issue", use: "attachment", purpose: "ISSUE_ATTACHMENT", scope: { kind: "issue", id: "issue-1" } },
  { kind: "issue", use: "description", purpose: "ISSUE_DESCRIPTION", scope: { kind: "issue", id: "issue-1" } },
  {
    kind: "new_comment",
    use: "attachment",
    purpose: "COMMENT_ATTACHMENT",
    scope: { kind: "new_comment", id: "issue-1" },
  },
  {
    kind: "new_comment",
    use: "description",
    purpose: "COMMENT_DESCRIPTION",
    scope: { kind: "new_comment", id: "issue-1" },
  },
  { kind: "comment", use: "attachment", purpose: "COMMENT_ATTACHMENT", scope: { kind: "comment", id: "comment-1" } },
  { kind: "comment", use: "description", purpose: "COMMENT_DESCRIPTION", scope: { kind: "comment", id: "comment-1" } },
  { kind: "draft", use: "attachment", purpose: "DRAFT_ISSUE_ATTACHMENT", scope: { kind: "draft", id: "draft-1" } },
  { kind: "draft", use: "description", purpose: "DRAFT_ISSUE_DESCRIPTION", scope: { kind: "draft", id: "draft-1" } },
] as const;

it.each(mappings)(
  "$kind $use sends exactly one original-file chain with its native purpose",
  async ({ kind, use, purpose, scope }) => {
    const tab = await fixture();
    const view = tab.app.asset.openContent(tab.hosts[kind]);
    const upload = tab.app.asset.createContentUpload(view, use);
    const file = imageFile();
    upload.select(file);
    expect(upload.phase).toBe("selected");
    expect(Boolean(upload.previewURL)).toBe(use === "description");
    const first = upload.start();
    expect(upload.start()).toBe(first);
    const result = track(first);
    const put = await tab.sent(result);
    expect(put.file).toBe(file);
    expect(tab.nerve.calls[0]?.body).toEqual({
      purpose,
      scope,
      name: "picture.png",
      size: 4,
      content_type: "image/png",
    });
    put.answer(new Response(null, { status: 204 }));
    await until(() => result.settled, "content upload");
    expect(result.error).toBeUndefined();
    expect(result.value).toMatchObject({ id: "asset-1", state: "uploaded", purpose, scope });
    expect(upload.readyAsset).toEqual(result.value);
    expect(tab.nerve.calls.map(({ method, path }) => [method, path])).toEqual([
      ["POST", "/api/v0/assets"],
      ["POST", "/api/v0/assets/asset-1/complete"],
    ]);
  }
);

it.each(["", "application/pdf", "application/vnd.example+json", "text/plain"])(
  "accepts attachment MIME %s with no preview",
  async (type) => {
    const tab = await fixture();
    const upload = tab.app.asset.createContentUpload(tab.app.asset.openContent(tab.hosts.issue), "attachment");
    upload.select(imageFile("资料.bin", 10, type));
    expect(upload.phase).toBe("selected");
    expect(upload.previewURL).toBeUndefined();
    const result = track(upload.start());
    (await tab.sent(result)).answer(new Response(null, { status: 204 }));
    await until(() => result.settled, "attachment type");
    expect(result.error).toBeUndefined();
    expect(tab.nerve.calls[0]?.body).toMatchObject({ content_type: type || "application/octet-stream", size: 10 });
  }
);

it.each([
  { name: "x.txt", type: "text/plain;charset=utf-8", size: 4, kind: "type" },
  { name: "x.txt", type: "text/*", size: 4, kind: "type" },
  { name: "x.txt", type: "broken", size: 4, kind: "type" },
  { name: "", type: "text/plain", size: 4, kind: "name" },
  { name: "a/b", type: "text/plain", size: 4, kind: "name" },
  { name: "a\\b", type: "text/plain", size: 4, kind: "name" },
  { name: "a\u0085b", type: "text/plain", size: 4, kind: "name" },
  { name: "a\n", type: "text/plain", size: 4, kind: "name" },
  { name: "💡".repeat(256), type: "text/plain", size: 4, kind: "name" },
  { name: "x.txt", type: "text/plain", size: 0, kind: "size" },
  { name: "x.txt", type: "text/plain", size: 11, kind: "size" },
])(
  "rejects invalid content selection $kind without preparing or allocating a preview",
  async ({ name, type, size, kind }) => {
    const tab = await fixture();
    const preview = vi.spyOn(URL, "createObjectURL");
    const upload = tab.app.asset.createContentUpload(tab.app.asset.openContent(tab.hosts.issue), "attachment");
    upload.select(imageFile(name, size, type));
    expect(upload.phase).toBe("failed");
    expect(upload.error).toMatchObject({ kind });
    await expect(upload.start()).rejects.toBeDefined();
    expect(tab.nerve.calls).toHaveLength(0);
    expect(preview).not.toHaveBeenCalled();
  }
);

it("counts Unicode filename characters rather than UTF-16 units", async () => {
  const tab = await fixture();
  const upload = tab.app.asset.createContentUpload(tab.app.asset.openContent(tab.hosts.issue), "attachment");
  upload.select(imageFile("💡".repeat(255), 4, "text/plain"));
  expect(upload.phase).toBe("selected");
});

it.each(["", "image/svg+xml", "image/jpg", "application/pdf"])(
  "rejects non-raster description MIME %s",
  async (type) => {
    const tab = await fixture();
    const upload = tab.app.asset.createContentUpload(tab.app.asset.openContent(tab.hosts.draft), "description");
    upload.select(imageFile("picture", 4, type));
    expect(upload.phase).toBe("failed");
    expect(upload.error).toMatchObject({ kind: "type" });
    expect(upload.previewURL).toBeUndefined();
    expect(tab.nerve.calls).toHaveLength(0);
  }
);

it("fails closed when the live size limit disappears", async () => {
  const tab = await fixture();
  const upload = tab.app.asset.createContentUpload(tab.app.asset.openContent(tab.hosts.issue), "attachment");
  runInAction(() => {
    tab.app.instance.config = undefined;
  });
  upload.select(imageFile());
  expect(upload.error).toMatchObject({ kind: "unavailable" });
  expect(tab.nerve.calls).toHaveLength(0);
});

it("replaces and releases only description previews", async () => {
  const tab = await fixture();
  const revoke = vi.spyOn(URL, "revokeObjectURL");
  const view = tab.app.asset.openContent(tab.hosts.issue);
  const upload = tab.app.asset.createContentUpload(view, "description");
  upload.select(imageFile());
  const first = upload.previewURL;
  upload.select(imageFile("second.png"));
  const second = upload.previewURL;
  expect(first).toMatch(/^blob:/);
  expect(second).not.toBe(first);
  expect(revoke).toHaveBeenCalledWith(first);
  view.dispose();
  expect(revoke).toHaveBeenCalledWith(second);
  expect(upload.previewURL).toBeUndefined();
});
