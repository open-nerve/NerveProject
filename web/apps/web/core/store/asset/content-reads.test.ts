/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// @vitest-environment jsdom
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { heldChange } from "@/lib/auth/fake-tab";
import { contentAssetOf, contentAssetTab } from "./fake-assets";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
const close: (() => void)[] = [];
beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  close.splice(0).forEach((done) => done());
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function fixture() {
  const tab = await contentAssetTab();
  close.push(() => tab.app.dispose());
  const view = tab.app.asset.openContent(tab.hosts.issue);
  const reads = tab.app.asset.content(view);
  const asset = contentAssetOf("a", view.host, "description");
  let serial = 0;
  const content = () => ({
    url: `/assets/storage/local/a?grant=${++serial}`,
    expires_at: new Date(Date.now() + 1000).toISOString(),
  });
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/assets/a") call.answer(json(200, asset));
    else if (call.path === "/api/v0/assets/a/content") call.answer(json(200, content()));
  };
  return { ...tab, view, reads, asset, content };
}
it("coalesces metadata and capability in a view, validates first, and reuses only before expiry", async () => {
  const tab = await fixture();
  const signal = new AbortController().signal;
  const first = await settle(
    Promise.all([tab.reads.read("a", signal), tab.reads.read("a", signal)]),
    "coalesced content"
  );
  expect(first.error).toBeUndefined();
  expect(first.value?.[0]).toEqual(first.value?.[1]);
  expect(first.value?.[0].asset).toEqual(tab.asset);
  expect(tab.nerve.calls.map((call) => call.path)).toEqual(["/api/v0/assets/a", "/api/v0/assets/a/content"]);
  expect((await tab.reads.read("a", signal)).content.url).toBe(first.value?.[0].content.url);
  await vi.advanceTimersByTimeAsync(1001);
  const next = await settle(tab.reads.read("a", signal), "expired read");
  expect(next.value?.content.url).not.toBe(first.value?.[0].content.url);
  expect(tab.nerve.to("/api/v0/assets/a/content")).toHaveLength(2);
});
it.each([{ scope: { kind: "issue", id: "other" } }, { purpose: "USER_AVATAR" }, { state: "pending" }, { id: "other" }])(
  "rejects mismatched or unready metadata before requesting any capability: %j",
  async (patch) => {
    const tab = await fixture();
    tab.nerve.respond = (call) => call.answer(json(200, { ...tab.asset, ...patch }));
    const result = await settle(tab.reads.read("a", new AbortController().signal), "invalid asset");
    expect(result.error).toBeDefined();
    expect(tab.nerve.to("/api/v0/assets/a/content")).toHaveLength(0);
  }
);
it("different views of the same host never share a read authorization", async () => {
  const tab = await fixture();
  const otherView = tab.app.asset.openContent(tab.hosts.issue);
  const other = tab.app.asset.content(otherView);
  const result = await settle(
    Promise.all([tab.reads.read("a", new AbortController().signal), other.read("a", new AbortController().signal)]),
    "independent views"
  );
  expect(result.value?.[0].content.url).not.toBe(result.value?.[1].content.url);
  expect(tab.nerve.to("/api/v0/assets/a/content")).toHaveLength(2);
  tab.view.dispose();
  expect((await settle(other.read("a", new AbortController().signal), "other still current")).error).toBeUndefined();
});
it("multiple nodes share a stable snapshot; the last release cancels its pending request", async () => {
  const tab = await fixture();
  tab.nerve.respond = undefined;
  const first = tab.reads.acquire("a");
  const second = tab.reads.acquire("a");
  expect(first.getSnapshot()).toBe(first.getSnapshot());
  await until(() => tab.nerve.calls.length === 1, "shared asset read");
  first.dispose();
  expect(tab.nerve.calls[0]?.aborted()).toBe(false);
  second.dispose();
  expect(tab.nerve.calls[0]?.aborted()).toBe(true);
  expect(first.getSnapshot().status).toBe("unavailable");
  expect(vi.getTimerCount()).toBe(0);
});
it("one aborted reader does not cancel another reader and keeps original refusal", async () => {
  const tab = await fixture();
  tab.nerve.respond = undefined;
  const one = new AbortController();
  const first = track(tab.reads.read("a", one.signal));
  const second = track(tab.reads.read("a", new AbortController().signal));
  await until(() => tab.nerve.calls.length === 1, "one read");
  one.abort();
  await until(() => first.settled, "caller cancellation");
  expect(first.error).toBeDefined();
  expect(tab.nerve.calls[0]!.aborted()).toBe(false);
  tab.nerve.calls[0]!.answer(problem(403, "forbidden"));
  await until(() => second.settled, "original refusal");
  expect(second.error).toMatchObject({ status: 403 });
});
it("expiration, focus, online and visibility share one bounded renewal", async () => {
  const tab = await fixture();
  const lease = tab.reads.acquire("a");
  await until(() => lease.getSnapshot().status === "ready", "first ready");
  expect(vi.getTimerCount()).toBe(1);
  const before = lease.getSnapshot();
  window.dispatchEvent(new Event("focus"));
  window.dispatchEvent(new Event("online"));
  document.dispatchEvent(new Event("visibilitychange"));
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.to("/api/v0/assets/a/content")).toHaveLength(1);
  await vi.advanceTimersByTimeAsync(1001);
  window.dispatchEvent(new Event("focus"));
  window.dispatchEvent(new Event("online"));
  document.dispatchEvent(new Event("visibilitychange"));
  await until(() => lease.getSnapshot().status === "ready" && lease.getSnapshot() !== before, "renewed once");
  expect(tab.nerve.to("/api/v0/assets/a/content")).toHaveLength(2);
  lease.dispose();
  expect(vi.getTimerCount()).toBe(0);
});
it("errors allow one automatic renewal, only actual current load restores the budget, and old events do nothing", async () => {
  const tab = await fixture();
  const one = tab.reads.acquire("a");
  const two = tab.reads.acquire("a");
  await until(() => one.getSnapshot().status === "ready", "ready");
  const first = one.getSnapshot();
  one.failed(first.stamp);
  two.failed(first.stamp);
  await until(
    () => tab.nerve.to("/api/v0/assets/a/content").length === 2 && one.getSnapshot().status === "ready",
    "one retry"
  );
  const second = one.getSnapshot();
  one.loaded(first.stamp);
  one.failed(second.stamp);
  two.failed(second.stamp);
  await vi.advanceTimersByTimeAsync(20);
  expect(one.getSnapshot().status).toBe("unavailable");
  expect(tab.nerve.to("/api/v0/assets/a/content")).toHaveLength(2);
  await settle(one.retry(), "explicit retry");
  const third = one.getSnapshot();
  expect(third.status).toBe("ready");
  one.failed(third.stamp);
  await vi.advanceTimersByTimeAsync(20);
  expect(tab.nerve.to("/api/v0/assets/a/content")).toHaveLength(3);
  await settle(one.retry(), "second explicit retry");
  const fourth = one.getSnapshot();
  one.loaded(fourth.stamp);
  one.failed(fourth.stamp);
  await until(
    () => tab.nerve.to("/api/v0/assets/a/content").length === 5 && one.getSnapshot().status === "ready",
    "real load restored allowance"
  );
  one.dispose();
  two.dispose();
});
it("expiry rejection stops automatic renewal until an explicit retry", async () => {
  const tab = await fixture();
  const lease = tab.reads.acquire("a");
  await until(() => lease.getSnapshot().status === "ready", "ready");
  tab.nerve.respond = (call) => call.answer(problem(503, "server_busy"));
  await vi.advanceTimersByTimeAsync(1001);
  await until(() => lease.getSnapshot().status === "unavailable", "failed expiry");
  expect(lease.getSnapshot().error).toMatchObject({ status: 503 });
  const count = tab.nerve.calls.length;
  await vi.advanceTimersByTimeAsync(60_000);
  // A server answered: neither a later focus nor a reconnect renews it; a renewal would start its read a task later.
  window.dispatchEvent(new Event("focus"));
  window.dispatchEvent(new Event("online"));
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.calls).toHaveLength(count);
  lease.dispose();
  expect(vi.getTimerCount()).toBe(0);
});
it("a renewal the network never carried is tried again by each later wake until the grant is renewed", async () => {
  const tab = await fixture();
  const lease = tab.reads.acquire("a");
  await until(() => lease.getSnapshot().status === "ready", "ready");
  const respond = tab.nerve.respond!;
  // The device wakes after the grant has expired: its timer fires before the network is back.
  tab.nerve.respond = (call) => call.fail();
  await vi.advanceTimersByTimeAsync(1001);
  await until(() => lease.getSnapshot().status === "unavailable", "renewal failed offline");
  expect(lease.getSnapshot().error).toBeInstanceOf(TypeError);
  const offline = tab.nerve.calls.length;
  // Still offline: one focus is one more attempt, and nothing is scheduled between wakes.
  window.dispatchEvent(new Event("focus"));
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.calls).toHaveLength(offline + 1);
  await vi.advanceTimersByTimeAsync(60_000);
  expect(tab.nerve.calls).toHaveLength(offline + 1);
  // The network is back: the wake that reports it renews the grant without a click.
  tab.nerve.respond = respond;
  window.dispatchEvent(new Event("online"));
  await until(() => lease.getSnapshot().status === "ready", "renewed after reconnect");
  expect(tab.nerve.to("/api/v0/assets/a/content")).toHaveLength(2);
  lease.dispose();
  expect(vi.getTimerCount()).toBe(0);
});
it("late resolved capability cannot publish after view retirement", async () => {
  const tab = await fixture();
  const gate = heldChange<Response>();
  let held: Response | undefined;
  tab.api.use({
    onResponse({ request, response }) {
      if (new URL(request.url).pathname.endsWith("/content")) {
        held = response;
        return gate.sent;
      }
      return response;
    },
  });
  const lease = tab.reads.acquire("a");
  const snapshots: string[] = [];
  lease.subscribe(() => snapshots.push(lease.getSnapshot().status));
  await until(() => !!held, "held content");
  tab.view.dispose();
  gate.answer(held!);
  await vi.advanceTimersByTimeAsync(20);
  expect(lease.getSnapshot().status).toBe("unavailable");
  expect(snapshots).not.toContain("ready");
  expect(vi.getTimerCount()).toBe(0);
});

it("a synchronous image error on ready starts one fresh authorization instead of reusing the completed promise", async () => {
  const tab = await fixture();
  const lease = tab.reads.acquire("a");
  let failed = false;
  const unsubscribe = lease.subscribe(() => {
    const snapshot = lease.getSnapshot();
    if (snapshot.status === "ready" && !failed) {
      failed = true;
      lease.failed(snapshot.stamp);
    }
  });
  await until(() => failed, "synchronous failure");
  await vi.advanceTimersByTimeAsync(20);
  expect(tab.nerve.to("/api/v0/assets/a/content")).toHaveLength(2);
  expect(lease.getSnapshot().status).toBe("ready");
  unsubscribe();
  lease.dispose();
});
