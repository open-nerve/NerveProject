/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { heldChange } from "@/lib/auth/fake-tab";
import { loadProjects } from "@/store/project/fake-projects";
import { attachmentOf, contentAssetOf, contentAssetTab } from "./fake-assets";
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
async function fixture(kind: "issue" | "comment" | "draft" = "draft") {
  const tab = await contentAssetTab();
  close.push(() => tab.app.dispose());
  const view = tab.app.asset.openContent(tab.hosts[kind]);
  const receipts: unknown[] = [];
  const resource = tab.app.asset.attachments(view, (receipt) => receipts.push(receipt));
  const path =
    kind === "issue"
      ? `/api/v0/issues/${tab.issue.id}/attachments`
      : kind === "comment"
        ? `/api/v0/issue-comments/${tab.comment.id}/attachments`
        : `/api/v0/drafts/${tab.draft.id}/attachments`;
  const load = (data = [attachmentOf("a")], next_cursor: string | null = null) =>
    answered(tab.nerve, () => resource.load(), ["GET", path], { data, next_cursor }, "attachment page");
  return { ...tab, view, resource, receipts, path, load };
}
it.each(["issue", "comment", "draft"] as const)(
  "reads %s native attachment pages without signed URLs",
  async (kind) => {
    const tab = await fixture(kind);
    await tab.load();
    expect(tab.resource.rows).toEqual([attachmentOf("a")]);
    expect(tab.resource.nextCursor).toBeNull();
    expect(tab.nerve.calls[0]?.queryEntries).toEqual([["limit", "50"]]);
  }
);
it("keeps server UUID order across 51 files, coalesces one cursor and never calls a page a total", async () => {
  const tab = await fixture();
  const first = Array.from({ length: 50 }, (_, i) => ({
    ...attachmentOf(`id-${i.toString().padStart(2, "0")}`),
    name: `${50 - i}.txt`,
    created_at: new Date(1000 * (50 - i)).toISOString(),
  }));
  await tab.load(first, "page-2");
  expect(tab.resource.nextCursor).toBe("page-2");
  const more = tab.resource.more();
  const twice = tab.resource.more();
  await until(() => tab.nerve.to(tab.path).length === 2, "next page");
  expect(tab.nerve.to(tab.path)).toHaveLength(2);
  tab.nerve.to(tab.path)[1]!.answer(json(200, { data: [first[49], attachmentOf("id-50")], next_cursor: null }));
  await Promise.all([more, twice]);
  expect(tab.resource.rows.map((row) => row.id)).toEqual([...first.map((row) => row.id), "id-50"]);
  expect(tab.resource.nextCursor).toBeNull();
  await tab.resource.more();
  expect(tab.nerve.to(tab.path)).toHaveLength(2);
});
it("a new first page wins over an already received old page and its cursor", async () => {
  const tab = await fixture();
  await tab.load([attachmentOf("old")], "page-2");
  const gate = heldChange<Response>();
  let held: Response | undefined;
  tab.api.use({
    onResponse({ request, response }) {
      if (new URL(request.url).searchParams.get("cursor")) {
        held = response;
        return gate.sent;
      }
      return response;
    },
  });
  const more = track(tab.resource.more());
  await until(() => tab.nerve.to(tab.path).length === 2, "page sent");
  tab.nerve.to(tab.path)[1]!.answer(json(200, { data: [attachmentOf("stale")], next_cursor: "stale-cursor" }));
  await until(() => !!held, "page at boundary");
  await tab.load([attachmentOf("fresh")]);
  gate.answer(held!);
  await until(() => more.settled, "superseded page");
  expect(tab.resource.rows.map((row) => row.id)).toEqual(["fresh"]);
  expect(tab.resource.nextCursor).toBeNull();
  expect(more.error).toBeUndefined();
});
it("repeated cursors stop with a readable error and explicit first-page retry", async () => {
  const tab = await fixture();
  await tab.load([attachmentOf("a")], "loop");
  const result = await answered(
    tab.nerve,
    () => tab.resource.more(),
    ["GET", tab.path],
    { data: [attachmentOf("b")], next_cursor: "loop" },
    "repeated cursor"
  );
  expect(result.error).toBeInstanceOf(Error);
  expect(tab.resource.error).toBeInstanceOf(Error);
  expect(tab.resource.rows.map((row) => row.id)).toEqual(["a"]);
  const retry = await answered(
    tab.nerve,
    () => tab.resource.retryRead(),
    ["GET", tab.path],
    { data: [attachmentOf("c")], next_cursor: null },
    "retry first page"
  );
  expect(retry.error).toBeUndefined();
  expect(tab.resource.rows.map((row) => row.id)).toEqual(["c"]);
});
it("confirmed deletion survives an older first-page answer; refresh errors never replay DELETE", async () => {
  const tab = await fixture();
  await tab.load();
  const gate = heldChange<Response>();
  let held: Response | undefined;
  let hold = true;
  tab.api.use({
    onResponse({ request, response }) {
      if (hold && new URL(request.url).pathname === tab.path) {
        hold = false;
        held = response;
        return gate.sent;
      }
      return response;
    },
  });
  const old = track(tab.resource.load());
  await until(() => tab.nerve.to(tab.path).length === 2, "old page");
  tab.nerve.to(tab.path)[1]!.answer(json(200, { data: [attachmentOf("a")], next_cursor: null }));
  await until(() => !!held, "old read boundary");
  const command = track(tab.resource.remove("a"));
  const double = track(tab.resource.remove("a"));
  await until(() => tab.nerve.to("/api/v0/assets/a").length === 1, "delete");
  tab.nerve.to("/api/v0/assets/a")[0]!.answer(noContent());
  await until(() => tab.nerve.to(tab.path).length === 3, "confirmed refresh");
  expect(tab.receipts).toEqual([{ kind: "deleted", id: "a" }]);
  tab.nerve.to(tab.path)[2]!.answer(problem(503, "server_busy"));
  gate.answer(held!);
  await until(() => command.settled && double.settled && old.settled, "confirmation settled");
  expect(command.error).toBeUndefined();
  expect(double.error).toBeUndefined();
  expect(tab.resource.rows).toEqual([]);
  expect(tab.resource.error).toMatchObject({ status: 503 });
  await answered(
    tab.nerve,
    () => tab.resource.retryRead(),
    ["GET", tab.path],
    { data: [], next_cursor: null },
    "read only retry"
  );
  expect(tab.nerve.to("/api/v0/assets/a")).toHaveLength(1);
});
it("a refused delete keeps its row, has no receipt and does not start a refresh", async () => {
  const tab = await fixture();
  await tab.load();
  const result = track(tab.resource.remove("a"));
  await until(() => tab.nerve.to("/api/v0/assets/a").length === 1, "refused delete");
  tab.nerve.to("/api/v0/assets/a")[0]!.answer(problem(403, "forbidden"));
  await until(() => result.settled, "refusal");
  expect(result.error).toMatchObject({ status: 403 });
  expect(tab.resource.rows).toHaveLength(1);
  expect(tab.receipts).toEqual([]);
  expect(tab.nerve.to(tab.path)).toHaveLength(1);
});
it("publishes the successful issue attachment receipt before authoritative pending retires the view", async () => {
  const tab = await fixture("issue");
  await tab.load();
  tab.nerve.respond = (call) => {
    if (call.path.endsWith("/comments")) call.answer(json(200, { data: [tab.comment], next_cursor: null }));
  };
  const asset = contentAssetOf("a", tab.view.host, "attachment");
  let atReceipt: boolean | undefined;
  const resource = tab.app.asset.attachments(tab.view, () => {
    atReceipt = tab.view.current;
  });
  const work = track(resource.afterUpload(asset));
  await until(() => tab.nerve.to(`/api/v0/issues/${tab.issue.id}`).length > 0 || work.settled, "owner refresh");
  expect(atReceipt).toBe(true);
  expect(tab.view.current).toBe(false);
  tab.nerve
    .to(`/api/v0/issues/${tab.issue.id}`)
    .at(-1)!
    .answer(json(200, { ...tab.issue, attachment_count: 71 }));
  await until(() => work.settled, "confirmed owner");
  expect(work.error).toBeUndefined();
  expect(tab.app.issue.issues.getIssueDetailById(tab.issue.id)?.attachment_count).toBe(71);
  expect(tab.nerve.calls.filter((call) => call.method !== "GET")).toHaveLength(0);
});
it.each(["comment", "draft"] as const)("%s confirmation only rereads its own attachments", async (kind) => {
  const tab = await fixture(kind);
  const asset = contentAssetOf("a", tab.view.host, "attachment");
  const result = await answered(
    tab.nerve,
    () => tab.resource.afterUpload(asset),
    ["GET", tab.path],
    { data: [attachmentOf("a")], next_cursor: null },
    "bound attachment"
  );
  expect(result.error).toBeUndefined();
  expect(tab.receipts).toEqual([{ kind: "uploaded", id: "a", asset }]);
  expect(tab.nerve.calls).toHaveLength(1);
});
it("retiring a view cancels pagination and permanently withdraws its rows", async () => {
  const tab = await fixture("issue");
  await tab.load([attachmentOf("a")], "later");
  const more = track(tab.resource.more());
  await until(() => tab.nerve.to(tab.path).length === 2, "pending page");
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  expect(tab.resource.rows).toEqual([]);
  expect(tab.nerve.to(tab.path)[1]!.aborted()).toBe(true);
  await until(() => more.settled, "retired read");
  expect(more.error).toBeUndefined();
  expect((await settle(tab.resource.load(), "retired load")).error).toBeDefined();
});
it("a retired pagination cannot poison a later complete refresh that legitimately reuses an opaque cursor", async () => {
  const tab = await fixture();
  await tab.load([attachmentOf("a")], "reused");
  const gate = heldChange<Response>();
  let held: Response | undefined;
  tab.api.use({
    onResponse({ request, response }) {
      if (new URL(request.url).searchParams.get("cursor") === "reused") {
        held = response;
        return gate.sent;
      }
      return response;
    },
  });
  const old = track(tab.resource.more());
  await until(() => tab.nerve.to(tab.path).length === 2, "old pagination");
  tab.nerve.to(tab.path)[1]!.answer(json(200, { data: [attachmentOf("b")], next_cursor: null }));
  await until(() => !!held, "old page held");
  await tab.load([attachmentOf("new-a")], "fresh");
  gate.answer(held!);
  await until(() => old.settled, "old page ignored");
  const next = await answered(
    tab.nerve,
    () => tab.resource.more(),
    ["GET", tab.path],
    { data: [attachmentOf("new-b")], next_cursor: "reused" },
    "fresh pagination"
  );
  expect(next.error).toBeUndefined();
  expect(tab.resource.rows.map((row) => row.id)).toEqual(["new-a", "new-b"]);
  expect(tab.resource.nextCursor).toBe("reused");
});
it("a short page with a server cursor is still incomplete", async () => {
  const tab = await fixture();
  await tab.load([attachmentOf("only-visible")], "next-visible");
  expect(tab.resource.rows).toHaveLength(1);
  expect(tab.resource.nextCursor).toBe("next-visible");
});
