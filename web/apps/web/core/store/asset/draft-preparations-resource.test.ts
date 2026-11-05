/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
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
  const view = tab.app.asset.openContent(tab.hosts.draft);
  const resource = tab.app.asset.draftPreparations(view);
  const asset = { ...contentAssetOf("a", view.host, "description"), state: "pending" as const, is_bound: false };
  const path = `/api/v0/drafts/${tab.draft.id}/asset-preparations`;
  const load = (data = [asset], next_cursor: string | null = null) =>
    answered(tab.nerve, () => resource.load(), ["GET", path], { data, next_cursor }, "preparations");
  return { ...tab, view, resource, asset, path, load };
}
it("opens private null-project preparations only on request and follows the native cursor", async () => {
  const tab = await fixture();
  expect(tab.nerve.calls).toHaveLength(0);
  await tab.load(undefined, "second");
  const result = await answered(
    tab.nerve,
    () => tab.resource.more(),
    ["GET", tab.path],
    { data: [{ ...tab.asset, id: "b", state: "uploaded", purpose: "DRAFT_ISSUE_ATTACHMENT" }], next_cursor: null },
    "preparation page"
  );
  expect(result.error).toBeUndefined();
  expect(tab.resource.rows.map((row) => row.id)).toEqual(["a", "b"]);
  expect(tab.resource.nextCursor).toBeNull();
});
it.each(["bound", "scope", "purpose"])("refuses an invalid %s preparation page", async (kind) => {
  const tab = await fixture();
  const asset = {
    ...tab.asset,
    ...(kind === "bound"
      ? { is_bound: true }
      : kind === "scope"
        ? { scope: { kind: "draft" as const, id: "other" } }
        : { purpose: "USER_AVATAR" as const }),
  };
  const result = await tab.load([asset]);
  expect(result.error).toBeDefined();
  expect(tab.resource.rows).toEqual([]);
});
it("explicit discard confirms once and refresh failure only offers a read retry", async () => {
  const tab = await fixture();
  await tab.load();
  const first = track(tab.resource.remove("a"));
  const second = track(tab.resource.remove("a"));
  await until(() => tab.nerve.to("/api/v0/assets/a").length === 1, "one discard");
  tab.nerve.to("/api/v0/assets/a")[0]!.answer(noContent());
  await until(() => tab.nerve.to(tab.path).length === 2, "refresh");
  tab.nerve.to(tab.path)[1]!.answer(problem(503, "server_busy"));
  await until(() => first.settled && second.settled, "discard confirmed");
  expect(first.error).toBeUndefined();
  expect(tab.resource.rows).toEqual([]);
  expect(tab.resource.error).toMatchObject({ status: 503 });
  await answered(
    tab.nerve,
    () => tab.resource.retryRead(),
    ["GET", tab.path],
    { data: [], next_cursor: null },
    "retry"
  );
  expect(tab.nerve.to("/api/v0/assets/a")).toHaveLength(1);
});
it("unknown discard result only verifies reads; absence from preparations is not proof of deletion", async () => {
  const tab = await fixture();
  await tab.load();
  const remove = track(tab.resource.remove("a"));
  await until(() => tab.nerve.to("/api/v0/assets/a").length === 1, "discard");
  tab.nerve.to("/api/v0/assets/a")[0]!.fail();
  await until(() => remove.settled, "unknown result");
  expect(remove.error).toBeDefined();
  expect(tab.resource.uncertainIDs).toEqual(["a"]);
  const retry = track(tab.resource.retryRead());
  await until(() => tab.nerve.to(tab.path).length === 2, "verify list");
  tab.nerve.to(tab.path)[1]!.answer(json(200, { data: [], next_cursor: null }));
  await until(() => tab.nerve.to("/api/v0/assets/a").length === 2, "verify asset");
  tab.nerve.to("/api/v0/assets/a")[1]!.answer(json(200, { ...tab.asset, state: "uploaded", is_bound: true }));
  await until(() => retry.settled, "verify complete");
  expect(retry.error).toBeUndefined();
  expect(tab.resource.uncertainIDs).toEqual(["a"]);
  expect(tab.nerve.to("/api/v0/assets/a").map((call) => call.method)).toEqual(["DELETE", "GET"]);
});
it("a now-bound description refusal keeps the recovery row and original error", async () => {
  const tab = await fixture();
  await tab.load();
  const remove = track(tab.resource.remove("a"));
  await until(() => tab.nerve.to("/api/v0/assets/a").length === 1, "discard");
  tab.nerve.to("/api/v0/assets/a")[0]!.answer(problem(409, "asset.bound"));
  await until(() => remove.settled, "bound refusal");
  expect(remove.error).toMatchObject({ problem: { code: "asset.bound" } });
  expect(tab.resource.rows).toHaveLength(1);
  expect(tab.resource.uncertainIDs).toEqual([]);
});
