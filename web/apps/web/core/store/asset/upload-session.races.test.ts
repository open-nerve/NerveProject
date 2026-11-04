/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { reaction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { assetTab, avatarTarget, imageFile } from "./fake-assets";

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

function responseGate() {
  let resolve!: (response: Response) => void;
  const promise = new Promise<Response>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

it.each(["preparing", "uploading", "confirming", "ready"])(
  "retirement from an observable %s transition cannot advance the old chain",
  async (phase) => {
    const tab = await fixture();
    const upload = tab.app.asset.createUpload(avatarTarget);
    upload.select(imageFile());
    close.push(
      reaction(
        () => upload.phase,
        (value) => {
          if (value === phase) upload.dispose();
        }
      )
    );
    let bound = 0;
    const result = track(
      upload.start().then((asset) => {
        if (asset) ++bound;
        return asset;
      })
    );
    if (phase === "confirming" || phase === "ready")
      (await tab.sent(result)).answer(new Response(null, { status: 204 }));
    await until(() => result.settled, "retired transition");
    expect(result.value).toBeUndefined();
    expect(result.error).toBeUndefined();
    expect(bound).toBe(0);
    expect(upload.phase).toBe("disposed");
    if (phase !== "ready") expect(tab.nerve.calls.filter((call) => call.path.endsWith("/complete"))).toHaveLength(0);
    if (phase === "uploading" || phase === "preparing") expect(tab.puts).toHaveLength(0);
  }
);

it("a cancelled transfer's late failure cannot clear the replacement's ready asset or progress", async () => {
  const tab = await fixture();
  const upload = tab.app.asset.createUpload(avatarTarget);
  upload.select(imageFile("before.png"));
  const old = track(upload.start());
  const oldPut = await tab.sent(old);
  upload.select(imageFile("after.png"));
  const fresh = track(upload.start());
  (await tab.sent(fresh, 2)).answer(new Response(null, { status: 204 }));
  await until(() => fresh.settled, "replacement ready");
  oldPut.fail();
  await until(() => old.settled, "retired rejection");
  expect(old.value).toBeUndefined();
  expect(old.error).toBeUndefined();
  expect(fresh.value?.name).toBe("after.png");
  expect(upload.readyAsset).toEqual(fresh.value);
  expect(upload.phase).toBe("ready");
  expect(upload.error).toBeUndefined();
});

it("a non-uploaded confirmation cannot fabricate a ready asset", async () => {
  const tab = await fixture();
  tab.nerve.respond = (call) =>
    call.path.endsWith("/complete") ? call.answer(json(200, tab.assets.get("asset-1"))) : tab.answer(call);
  const upload = tab.app.asset.createUpload(avatarTarget);
  upload.select(imageFile());
  const result = track(upload.start());
  (await tab.sent(result)).answer(new Response(null, { status: 204 }));
  await until(() => result.settled, "invalid confirmation");
  expect(result.error).toBeDefined();
  expect(upload.readyAsset).toBeUndefined();
  expect(upload.phase).toBe("failed");
  expect(upload.recovery).toBe("confirm");
});

it.each(["prepare", "complete"])(
  "cancellation after the %s response reached the client suppresses its delayed delivery",
  async (stage) => {
    const tab = await fixture();
    const gate = responseGate();
    let held: Response | undefined;
    tab.api.use({
      onResponse({ request, response }) {
        if (
          new URL(request.url).pathname === (stage === "prepare" ? "/api/v0/assets" : "/api/v0/assets/asset-1/complete")
        ) {
          held = response;
          return gate.promise;
        }
        return response;
      },
    });
    const upload = tab.app.asset.createUpload(avatarTarget);
    upload.select(imageFile());
    const result = track(upload.start());
    if (stage === "complete") (await tab.sent(result)).answer(new Response(null, { status: 204 }));
    await until(() => held !== undefined || result.settled, "response at client boundary");
    expect(held).toBeDefined();
    upload.cancel();
    gate.resolve(held!);
    await until(() => result.settled, "late successful JSON");
    expect(result.value).toBeUndefined();
    expect(result.error).toBeUndefined();
    expect(upload.phase).toBe("cancelled");
    expect(upload.readyAsset).toBeUndefined();
    expect(tab.puts).toHaveLength(stage === "prepare" ? 0 : 1);
  }
);

it("old prepare failure delivered after replacement cannot fail the new upload", async () => {
  const tab = await fixture();
  const gate = responseGate();
  let held: Response | undefined;
  tab.api.use({
    onResponse({ request, response }) {
      if (!held && new URL(request.url).pathname === "/api/v0/assets") {
        held = response;
        return gate.promise;
      }
      return response;
    },
  });
  const upload = tab.app.asset.createUpload(avatarTarget);
  upload.select(imageFile("before.png"));
  const old = track(upload.start());
  await until(() => held !== undefined || old.settled, "old prepared response");
  expect(held).toBeDefined();
  upload.select(imageFile("after.png"));
  const fresh = track(upload.start());
  (await tab.sent(fresh)).answer(new Response(null, { status: 204 }));
  await until(() => fresh.settled, "new upload complete");
  gate.resolve(json(503, { status: 503, code: "server_busy", title: "Busy" }));
  await until(() => old.settled, "old refusal");
  expect(old.error).toBeUndefined();
  expect(old.value).toBeUndefined();
  expect(upload.phase).toBe("ready");
  expect(upload.readyAsset?.id).toBe("asset-2");
  expect(upload.error).toBeUndefined();
});

it("retirement from the failed transition suppresses feedback to the old owner", async () => {
  const tab = await fixture();
  const upload = tab.app.asset.createUpload(avatarTarget);
  upload.select(imageFile());
  close.push(
    reaction(
      () => upload.phase,
      (phase) => {
        if (phase === "failed") upload.dispose();
      }
    )
  );
  const result = track(upload.start());
  (await tab.sent(result)).fail();
  await until(() => result.settled, "retired failed upload");
  expect(result.value).toBeUndefined();
  expect(result.error).toBeUndefined();
  expect(upload.phase).toBe("disposed");
});
