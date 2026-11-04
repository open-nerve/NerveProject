// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, StrictMode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { EditorImages, ImageOperation, ImageTransferPhase } from "@nerve/editor";
import type { RootStore } from "@/store/root.store";
import type { ContentView } from "@/store/asset/content-view";
import { StoreContext } from "@/lib/store-context";
import { json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { contentAssetOf, contentAssetTab, imageFile } from "@/store/asset/fake-assets";
import { pickerDOM } from "@/components/assets/fake-image-picker";
import { useEditorAssets } from "./use-editor-assets";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let tab: Awaited<ReturnType<typeof fixture>>;
const fixture = async () => pickerDOM(await contentAssetTab());
let images: EditorImages | undefined;
function Host({ view }: { view: ContentView | undefined }) {
  images = useEditorAssets(view);
  return null;
}
const render = (view: ContentView | undefined) =>
  act(async () =>
    tab.renderer.render(
      <StrictMode>
        <StoreContext.Provider value={tab.app}>
          <Host view={view} />
        </StoreContext.Provider>
      </StrictMode>
    )
  );
const operation = () => {
  const controller = new AbortController();
  const phases: ImageTransferPhase[] = [];
  return {
    controller,
    phases,
    options: { signal: controller.signal, onPhase: (phase) => phases.push(phase) } satisfies ImageOperation,
  };
};
const start = async () => {
  const view = tab.app.asset.openContent(tab.hosts.issue);
  await render(view);
  expect(images).toBeDefined();
  return view;
};
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  tab = await fixture();
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
it("provides no capability without a current canonical owner", async () => {
  await render(undefined);
  expect(images).toBeUndefined();
  await render(tab.app.asset.openContent({ ...tab.hosts.issue, issueID: "unread" }));
  expect(images).toBeUndefined();
  expect(tab.nerve.calls).toHaveLength(0);
});
it("two nodes share reads but dispose independently, withdraw immediately and reject old handle events", async () => {
  const view = await start();
  const asset = contentAssetOf("a", view.host, "description");
  let serial = 0;
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/assets/a") call.answer(json(200, asset));
    else if (call.path.endsWith("/content"))
      call.answer(
        json(200, {
          url: `/assets/storage/local/a?grant=${++serial}`,
          expires_at: new Date(Date.now() + 1000).toISOString(),
        })
      );
  };
  const first = images!.read("a"),
    second = images!.read("a");
  await until(() => second.getSnapshot().phase === "ready", "content");
  expect(first.getSnapshot()).toBe(first.getSnapshot());
  expect(tab.nerve.to("/api/v0/assets/a/content")).toHaveLength(1);
  first.dispose();
  first.failed();
  expect(second.getSnapshot().phase).toBe("ready");
  view.dispose();
  expect(second.getSnapshot().phase).toBe("unavailable");
  expect(images!.signal.aborted).toBe(true);
  await start();
  const fresh = images!.read("a");
  await until(() => fresh.getSnapshot().phase === "ready", "new view");
  second.failed();
  expect(fresh.getSnapshot().phase).toBe("ready");
  fresh.dispose();
  expect(vi.getTimerCount()).toBe(0);
});
it("limits transfers to two and cancels a queued node before prepare", async () => {
  await start();
  const a = operation(),
    b = operation(),
    c = operation();
  const one = track(images!.write!.upload("one", imageFile(), a.options));
  const two = track(images!.write!.upload("two", imageFile(), b.options));
  const three = track(images!.write!.upload("three", imageFile(), c.options));
  await until(() => tab.puts.length === 2, "two transfers");
  expect(c.phases).toEqual(["queued"]);
  expect(tab.nerve.to("/api/v0/assets")).toHaveLength(2);
  c.controller.abort();
  await until(() => three.settled, "queued cancelled");
  expect(three.error).toBeDefined();
  tab.puts.forEach((p) => p.answer(new Response(null, { status: 200 })));
  await until(() => one.settled && two.settled, "two confirmed");
  expect(one.value).toBe("asset-1");
  expect(two.value).toBe("asset-2");
  expect(tab.nerve.to("/api/v0/assets")).toHaveLength(2);
  expect(a.phases).toEqual(["queued", "preparing", "uploading", "confirming"]);
});
it("a third node begins after one transfer finishes and success leaves no preview or live session", async () => {
  await start();
  const results = ["one", "two", "three"].map((id) =>
    track(images!.write!.upload(id, imageFile(), operation().options))
  );
  await until(() => tab.puts.length === 2, "two");
  tab.puts[0]!.answer(new Response(null, { status: 200 }));
  await until(() => tab.puts.length === 3, "queued starts");
  tab.puts.slice(1).forEach((p) => p.answer(new Response(null, { status: 200 })));
  await until(() => results.every((r) => r.settled), "complete");
  expect(results.map((r) => r.value)).toEqual(["asset-1", "asset-2", "asset-3"]);
  expect(tab.revoke).toHaveBeenCalledTimes(3);
});
it("replacing a node retires its previous file and ignores late completion", async () => {
  await start();
  const old = track(images!.write!.upload("one", imageFile("old.png"), operation().options));
  await until(() => tab.puts.length === 1, "old PUT");
  const fresh = track(images!.write!.upload("one", imageFile("new.png"), operation().options));
  await until(() => tab.puts.length === 2, "new PUT");
  expect(tab.puts[0]!.signal?.aborted).toBe(true);
  tab.puts[0]!.answer(new Response(null, { status: 200 }));
  tab.puts[1]!.answer(new Response(null, { status: 200 }));
  await until(() => old.settled && fresh.settled, "finished");
  expect(old.error).toBeDefined();
  expect(fresh.value).toBe("asset-2");
  expect(tab.nerve.to("/api/v0/assets/asset-1/complete")).toHaveLength(0);
});
it("explicit retry of an uncertain PUT confirms the same preparation without PUT or new prepare", async () => {
  await start();
  const first = track(images!.write!.upload("one", imageFile(), operation().options));
  await until(() => tab.puts.length === 1, "PUT");
  tab.puts[0]!.fail();
  await until(() => first.settled, "uncertain");
  expect(first.error).toBeDefined();
  const next = track(images!.write!.retryUpload("one", operation().options));
  await until(() => next.settled, "confirmed");
  expect(next.value).toBe("asset-1");
  expect(tab.puts).toHaveLength(1);
  expect(tab.nerve.to("/api/v0/assets")).toHaveLength(1);
});
it.each([403, 404, 502, 0])(
  "copy refusal/unknown %i never retries itself or falls back to the source ID",
  async (status) => {
    await start();
    tab.nerve.respond = undefined;
    const copy = track(images!.write!.duplicate("one", "source", operation().options));
    await until(() => tab.nerve.to("/api/v0/assets/source/duplicate").length === 1, "copy");
    const call = tab.nerve.to("/api/v0/assets/source/duplicate")[0]!;
    expect(call.body).toEqual({ purpose: "ISSUE_DESCRIPTION", scope: { kind: "issue", id: tab.issue.id } });
    if (status) call.answer(status === 502 ? new Response("gateway", { status }) : problem(status, "forbidden"));
    else call.fail();
    await until(() => copy.settled, "copy failure");
    expect(copy.error).toBeDefined();
    expect(copy.value).toBeUndefined();
    await vi.advanceTimersByTimeAsync(2000);
    window.dispatchEvent(new Event("online"));
    expect(tab.nerve.to("/api/v0/assets/source/duplicate")).toHaveLength(1);
  }
);
it("duplicate returns only the validated target ID and view retirement blocks a late answer", async () => {
  const view = await start();
  tab.nerve.respond = undefined;
  const first = track(images!.write!.duplicate("one", "source", operation().options));
  await until(() => tab.nerve.calls.length === 1, "duplicate");
  tab.nerve.calls[0]!.answer(json(201, contentAssetOf("target", view.host, "description")));
  await until(() => first.settled, "target");
  expect(first.value).toBe("target");
  const late = track(images!.write!.duplicate("two", "source", operation().options));
  await until(() => tab.nerve.calls.length === 2, "late copy");
  await render(undefined);
  expect(tab.nerve.calls[1]!.aborted()).toBe(true);
  tab.nerve.calls[1]!.answer(json(201, contentAssetOf("late", view.host, "description")));
  await until(() => late.settled, "retired");
  expect(late.value).toBeUndefined();
  expect(late.error).toBeDefined();
});
it("synchronous phase observers may cancel, with no prepare and no retained work", async () => {
  await start();
  const ctrl = new AbortController();
  const result = track(images!.write!.upload("one", imageFile(), { signal: ctrl.signal, onPhase: () => ctrl.abort() }));
  await until(() => result.settled, "cancelled synchronously");
  expect(result.error).toBeDefined();
  expect(tab.nerve.calls).toHaveLength(0);
});
it("an archived issue exposes reads without writes and an attachment is never rendered as a description image", async () => {
  await tab.readIssue({ ...tab.issue, archived_at: new Date().toISOString() });
  const view = await start();
  expect(images!.write).toBeUndefined();
  tab.nerve.respond = (call) =>
    call.answer(
      json(
        200,
        call.path.endsWith("/content")
          ? { url: "/assets/storage/local/file", expires_at: new Date(Date.now() + 1000).toISOString() }
          : contentAssetOf("ordinary", view.host, "attachment")
      )
    );
  const read = images!.read("ordinary");
  await until(() => read.getSnapshot().phase === "error", "not an image");
  expect(read.getSnapshot()).toEqual({ phase: "error", message: "errors.unknown" });
});
it("root retirement aborts uploads, queued work and reads, without publishing a late ID", async () => {
  await start();
  const held = images!;
  const uploads = ["a", "b", "c"].map((id) => track(held.write!.upload(id, imageFile(), operation().options)));
  await until(() => tab.puts.length === 2, "two PUTs");
  const read = held.read("pending-read");
  await until(() => tab.nerve.to("/api/v0/assets/pending-read").length === 1, "read");
  tab.app.dispose();
  expect(held.signal.aborted).toBe(true);
  expect(read.getSnapshot().phase).toBe("unavailable");
  expect(tab.puts.every((p) => p.signal?.aborted)).toBe(true);
  tab.puts.forEach((p) => p.answer(new Response(null, { status: 200 })));
  await until(() => uploads.every((r) => r.settled), "all cancelled");
  expect(uploads.every((r) => r.error && r.value === undefined)).toBe(true);
  expect(tab.nerve.to("/api/v0/assets")).toHaveLength(2);
  expect(vi.getTimerCount()).toBe(0);
});
it("a node download uses raw bytes and its own cancellation, without cancelling another node read", async () => {
  const view = await start();
  const asset = contentAssetOf("a", view.host, "description");
  tab.nerve.respond = (call) =>
    call.answer(
      json(
        200,
        call.path.endsWith("/content")
          ? { url: "/assets/storage/local/a?private=1", expires_at: new Date(Date.now() + 1000).toISOString() }
          : asset
      )
    );
  const read = images!.read("a");
  await until(() => read.getSnapshot().phase === "ready", "read");
  const controller = new AbortController();
  const result = track(images!.download("a", controller.signal));
  await until(() => tab.puts.length === 1, "raw GET");
  expect(tab.puts[0]!.file).toBeUndefined();
  controller.abort();
  await until(() => result.settled, "cancelled download");
  expect(result.error).toBeDefined();
  expect(tab.puts[0]!.signal?.aborted).toBe(true);
  expect(read.getSnapshot().phase).toBe("ready");
  expect(document.querySelector("a[download]")).toBeNull();
});
it("the last image lease removes wake listeners even while the content owner remains current", async () => {
  await start();
  const removed = vi.spyOn(window, "removeEventListener");
  const documentRemoved = vi.spyOn(document, "removeEventListener");
  const first = images!.read("a"),
    second = images!.read("a");
  await until(() => tab.nerve.to("/api/v0/assets/a").length === 1, "pending read");
  first.dispose();
  expect(removed.mock.calls.some((c) => c[0] === "online")).toBe(false);
  second.dispose();
  expect(removed.mock.calls.some((c) => c[0] === "online")).toBe(true);
  expect(removed.mock.calls.some((c) => c[0] === "focus")).toBe(true);
  expect(documentRemoved.mock.calls.some((c) => c[0] === "visibilitychange")).toBe(true);
  expect(images!.signal.aborted).toBe(false);
  expect(tab.nerve.to("/api/v0/assets/a")[0]!.aborted()).toBe(true);
});
