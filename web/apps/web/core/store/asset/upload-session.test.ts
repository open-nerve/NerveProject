/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { runInAction } from "mobx";
import { problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { assetTab, avatarTarget, imageFile } from "./fake-assets";
import type { ProfileUploadTarget } from "./upload-policy";

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
  const tab = await assetTab();
  close.push(() => tab.app.dispose());
  return tab;
}

it.each(["avatar", "cover", "workspace", "project", "new_project"])(
  "%s sends one exact scoped chain for two starts",
  async (kind) => {
    const tab = await fixture();
    const target: ProfileUploadTarget =
      kind === "avatar"
        ? avatarTarget
        : kind === "cover"
          ? { purpose: "USER_COVER", scope: { kind: "user" } }
          : kind === "workspace"
            ? { purpose: "WORKSPACE_LOGO", scope: { kind: "workspace", id: tab.workspace.id } }
            : {
                purpose: "PROJECT_COVER",
                scope:
                  kind === "project"
                    ? { kind: "project", id: tab.project.id }
                    : { kind: "new_project", id: tab.workspace.id },
              };
    const upload = tab.app.asset.createUpload(target);
    const file = imageFile();
    upload.select(file);
    expect(upload.phase).toBe("selected");
    expect(upload.previewURL).toMatch(/^blob:/);
    const first = upload.start();
    expect(upload.start()).toBe(first);
    const result = track(first);
    const put = await tab.sent(result);
    expect(upload.phase).toBe("uploading");
    expect(upload.busy).toBe(true);
    expect(put.file).toBe(file);
    expect(tab.nerve.calls[0]?.body).toEqual({ ...target, name: file.name, size: 4, content_type: "image/png" });
    put.answer(new Response(null, { status: 204 }));
    await until(() => result.settled, "uploaded asset");
    expect(result.error).toBeUndefined();
    expect(result.value).toMatchObject({ id: "asset-1", state: "uploaded", is_bound: false });
    expect(upload.readyAsset).toEqual(result.value);
    expect(upload.phase).toBe("ready");
    expect(upload.busy).toBe(false);
    expect(tab.nerve.calls.map((call) => call.path)).toEqual(["/api/v0/assets", "/api/v0/assets/asset-1/complete"]);
  }
);

it.each([
  { bytes: 0, type: "image/png" },
  { bytes: 11, type: "image/png" },
  { bytes: 4, type: "" },
  { bytes: 4, type: "image/svg+xml" },
])("rejects invalid selection without any request: $bytes $type", async ({ bytes, type }) => {
  const { app, nerve, puts } = await fixture();
  const upload = app.asset.createUpload(avatarTarget);
  upload.select(imageFile("picture.png", bytes, type));
  expect(upload.phase).toBe("failed");
  expect(upload.error).toBeDefined();
  expect(upload.previewURL).toBeUndefined();
  await expect(upload.start()).rejects.toBeDefined();
  expect(nerve.calls).toHaveLength(0);
  expect(puts).toHaveLength(0);
});

it.each(["image/jpeg", "image/png", "image/webp", "image/gif"])("accepts %s at the instance limit", async (type) => {
  const { app } = await fixture();
  const upload = app.asset.createUpload(avatarTarget);
  upload.select(imageFile("picture", 10, type));
  expect(upload.phase).toBe("selected");
  expect(upload.error).toBeUndefined();
});

it("fails closed while the instance size limit is unknown", async () => {
  const { app, nerve } = await fixture();
  runInAction(() => {
    app.instance.config = undefined;
  });
  const upload = app.asset.createUpload(avatarTarget);
  upload.select(imageFile());
  expect(upload.phase).toBe("failed");
  expect(nerve.calls).toHaveLength(0);
});

it("prepare refusal retains the selection and explicit start obtains a new grant", async () => {
  const tab = await fixture();
  tab.nerve.respond = (call) => call.answer(problem(503, "server_busy"));
  const upload = tab.app.asset.createUpload(avatarTarget);
  upload.select(imageFile());
  const preview = upload.previewURL;
  const first = track(upload.start());
  await until(() => first.settled, "prepare rejection");
  expect(first.error).toMatchObject({ problem: { code: "server_busy" } });
  expect(upload.phase).toBe("failed");
  expect(upload.previewURL).toBe(preview);
  tab.nerve.respond = tab.answer;
  const retry = track(upload.start());
  (await tab.sent(retry)).answer(new Response(null, { status: 204 }));
  await until(() => retry.settled, "retry preparation");
  expect(retry.value?.state).toBe("uploaded");
  expect(tab.nerve.to("/api/v0/assets")).toHaveLength(2);
});

it.each(["put", "complete"])(
  "uncertain %s retries only complete for the same prepared asset, even after grant expiry",
  async (stage) => {
    const tab = await fixture();
    const upload = tab.app.asset.createUpload(avatarTarget);
    upload.select(imageFile());
    tab.nerve.respond = (call) =>
      stage === "complete" && call.path.endsWith("/complete") ? call.fail() : tab.answer(call);
    const initial = track(upload.start());
    const put = await tab.sent(initial);
    if (stage === "put") put.fail();
    else put.answer(new Response(null, { status: 204 }));
    await until(() => initial.settled, "uncertain upload");
    expect(initial.error).toBeDefined();
    expect(upload.phase).toBe("failed");
    await vi.advanceTimersByTimeAsync(120_000);
    tab.nerve.respond = tab.answer;
    const retry = track(upload.retryComplete());
    await until(() => retry.settled, "old ID complete");
    expect(retry.value).toMatchObject({ id: "asset-1", state: "uploaded" });
    expect(tab.nerve.to("/api/v0/assets")).toHaveLength(1);
    expect(tab.puts).toHaveLength(1);
  }
);

it("object_missing requires explicit restart with a new asset/key, never a second PUT to the old key", async () => {
  const tab = await fixture();
  tab.nerve.respond = (call) =>
    call.path.endsWith("/complete") ? call.answer(problem(409, "asset.object_missing")) : tab.answer(call);
  const upload = tab.app.asset.createUpload(avatarTarget);
  upload.select(imageFile());
  const first = track(upload.start());
  (await tab.sent(first)).fail();
  await until(() => first.settled, "PUT uncertainty");
  await expect(upload.retryComplete()).rejects.toMatchObject({ problem: { code: "asset.object_missing" } });
  await expect(upload.start()).rejects.toBeDefined();
  await expect(upload.retryComplete()).rejects.toBeDefined();
  tab.nerve.respond = tab.answer;
  const restart = track(upload.restart());
  (await tab.sent(restart, 2)).answer(new Response(null, { status: 204 }));
  await until(() => restart.settled, "fresh upload");
  expect(restart.value?.id).toBe("asset-2");
  expect(tab.nerve.to("/api/v0/assets")).toHaveLength(2);
});

it.each(["select", "cancel", "dispose", "root"])(
  "%s terminates the old PUT and releases its preview; late success cannot complete",
  async (ending) => {
    const tab = await fixture();
    const revoke = vi.spyOn(URL, "revokeObjectURL");
    const upload = tab.app.asset.createUpload(avatarTarget);
    upload.select(imageFile());
    const oldPreview = upload.previewURL;
    const old = track(upload.start());
    const put = await tab.sent(old);
    if (ending === "select") upload.select(imageFile("new.png"));
    else if (ending === "cancel") upload.cancel();
    else if (ending === "dispose") {
      upload.dispose();
      upload.dispose();
    } else {
      tab.app.dispose();
      tab.app.dispose();
    }
    expect(put.signal?.aborted).toBe(true);
    expect(revoke.mock.calls.filter(([url]) => url === oldPreview)).toHaveLength(1);
    const phase = upload.phase;
    put.answer(new Response(null, { status: 204 }));
    await until(() => old.settled, "retired upload");
    expect(old.value).toBeUndefined();
    expect(old.error).toBeUndefined();
    expect(upload.readyAsset).toBeUndefined();
    expect(upload.phase).toBe(phase);
    expect(tab.nerve.calls.map((call) => call.path)).toEqual(["/api/v0/assets"]);
    if (ending === "select") expect(upload.previewURL).not.toBe(oldPreview);
    else expect(upload.previewURL).toBeUndefined();
    expect(vi.getTimerCount()).toBe(0);
  }
);

it.each(["prepare", "complete"])(
  "dispose during %s ignores late success and cannot start owner work",
  async (stage) => {
    const tab = await fixture();
    tab.nerve.respond = (call) => {
      if (stage === "complete" && call.path === "/api/v0/assets") tab.answer(call);
    };
    const upload = tab.app.asset.createUpload(avatarTarget);
    upload.select(imageFile());
    let bound = 0;
    const result = track(
      upload.start().then((asset) => {
        if (asset) ++bound;
        return asset;
      })
    );
    if (stage === "complete") (await tab.sent(result)).answer(new Response(null, { status: 204 }));
    await until(() => tab.nerve.calls.length === (stage === "prepare" ? 1 : 2) || result.settled, "parked API request");
    const call = tab.nerve.calls.at(-1)!;
    expect(call).toBeDefined();
    upload.dispose();
    expect(call.aborted()).toBe(true);
    tab.answer(call);
    await until(() => result.settled, "disposed API answer");
    expect(bound).toBe(0);
    expect(result.value).toBeUndefined();
    expect(upload.phase).toBe("disposed");
  }
);
