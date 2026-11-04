/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { contentAssetOf, contentAssetTab } from "./fake-assets";
import { json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
let tab: Awaited<ReturnType<typeof contentAssetTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await contentAssetTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const asset = (id: string) => ({ ...contentAssetOf(id, tab.hosts.new_comment, "attachment"), is_bound: false });
function resource(ids: string[] = []) {
  const selected = vi.fn();
  const view = tab.app.asset.openContent(tab.hosts.new_comment);
  return { view, selected, files: tab.app.asset.commentPreparations(view, ids, selected) };
}
it("keeps selected preparations separate from image UUIDs and caps the complete POST at 100", () => {
  const { files, selected } = resource();
  for (let n = 0; n < 100; n++) files.add(asset(`file-${n}`));
  expect(files.payload()).toHaveLength(100);
  expect(() => files.add(asset("file-101"))).toThrow("100");
  expect(() => files.add({ ...asset("image"), purpose: "COMMENT_DESCRIPTION" })).toThrow();
  expect(selected).toHaveBeenCalledTimes(100);
  expect(tab.nerve.calls).toHaveLength(0);
});
it("new views reauthorize every saved ID, but never replay a transfer or POST", async () => {
  const { files } = resource(["saved-a", "saved-b"]);
  expect(() => files.payload()).toThrow();
  const reading = track(files.load());
  await until(() => tab.nerve.calls.length === 2, "metadata reads");
  for (const call of tab.nerve.calls) call.answer(json(200, asset(call.path.split("/").at(-1)!)));
  await until(() => reading.settled, "metadata complete");
  expect(reading.error).toBeUndefined();
  expect(files.payload()).toEqual(["saved-a", "saved-b"]);
  expect(tab.nerve.calls.map((c) => c.method)).toEqual(["GET", "GET"]);
});
it.each(["bound", "other-host", "image", "pending"] as const)(
  "rejects a restored %s asset until the user removes that selection",
  async (kind) => {
    const { files, selected } = resource(["saved"]);
    const reading = track(files.load());
    await until(() => tab.nerve.calls.length === 1, "asset read");
    const wrong =
      kind === "bound"
        ? { is_bound: true }
        : kind === "other-host"
          ? { scope: { kind: "new_comment", id: "other" } }
          : kind === "image"
            ? { purpose: "COMMENT_DESCRIPTION" }
            : { state: "pending" };
    tab.nerve.calls[0]!.answer(json(200, { ...asset("saved"), ...wrong }));
    await until(() => reading.settled, "refused selection");
    expect(reading.error).toBeDefined();
    expect(files.ready).toBe(false);
    expect(files.rows).toEqual([]);
    files.remove("saved");
    expect(files.payload()).toEqual([]);
    expect(selected).toHaveBeenCalledExactlyOnceWith([]);
    expect(tab.nerve.calls.some((c) => c.method === "DELETE")).toBe(false);
  }
);
it("a stale metadata answer cannot reinsert an explicitly removed selection", async () => {
  const { files } = resource(["saved"]);
  const reading = track(files.load());
  await until(() => tab.nerve.calls.length === 1, "read");
  files.remove("saved");
  tab.nerve.calls[0]!.answer(json(200, asset("saved")));
  await until(() => reading.settled, "old read");
  expect(files.rows).toEqual([]);
  expect(files.payload()).toEqual([]);
});
it("a rejected recheck withdraws previously held metadata and allows an explicit read retry", async () => {
  const { files } = resource();
  files.add(asset("saved"));
  const reading = track(files.load());
  await until(() => tab.nerve.calls.length === 1, "recheck");
  tab.nerve.calls[0]!.answer(problem(403, "forbidden"));
  await until(() => reading.settled, "rejected");
  expect(files.rows).toEqual([]);
  expect(files.unresolved).toEqual(["saved"]);
  const retry = track(files.load());
  await until(() => tab.nerve.calls.length === 2, "retry");
  tab.nerve.calls[1]!.answer(json(200, asset("saved")));
  await until(() => retry.settled, "retry done");
  expect(files.ready).toBe(true);
});
it("form resources stay independent and a retired view cannot publish its late metadata", async () => {
  const one = resource(["saved"]);
  const two = resource();
  two.files.add(asset("other"));
  const reading = track(one.files.load());
  await until(() => tab.nerve.calls.length === 1, "old read");
  one.view.dispose();
  tab.nerve.calls[0]!.answer(json(200, asset("saved")));
  await until(() => reading.settled, "ended");
  expect(one.files.rows).toEqual([]);
  expect(one.selected).not.toHaveBeenCalled();
  expect(two.files.payload()).toEqual(["other"]);
});
