/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Draft, DraftUpdate } from "@nerve/api-client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { draftOf, issueDetailOf, issuePageOf, issueStoreTab, requestedIssue } from "../fake-issues";
import { IssueListBinding } from "../helpers/issue-list-binding";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const store = () => tab.app.issue.workspaceDraftIssues;
const path = "/api/v0/workspaces/ws/drafts";
const row = draftOf("d");
const changed = { ...row, project_id: null, payload: { name: "saved" }, updated_at: "2026-10-30T00:01:00Z" };
const read = () => store().fetchIssues("ws");
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
async function loaded(rows: Draft[] = [row], cursor: string | null = null, total = rows.length) {
  const sent = await requestedIssue(tab.nerve, read, ["GET", path]);
  sent.call.answer(json(200, { data: rows, next_cursor: cursor, total }));
  await sent.promise;
}
function answerRefresh(rows: Draft[], total = rows.length) {
  tab.nerve.respond = (call) => {
    if (call.path === path && call.method === "GET") call.answer(json(200, { data: rows, next_cursor: null, total }));
  };
}
it("keeps workspace pagination and loaders independent and uses server total", async () => {
  const other = workspaceOf("other");
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, other]);
  const a = await requestedIssue(tab.nerve, read, ["GET", path]);
  const b = await requestedIssue(tab.nerve, () => store().fetchIssues("other"), [
    "GET",
    "/api/v0/workspaces/other/drafts",
  ]);
  expect(store().getLoader("ws")).toBe("init-loader");
  expect(store().getLoader("other")).toBe("init-loader");
  b.call.answer(json(200, { data: [draftOf("b", other.id)], total: 100, next_cursor: "other-cursor" }));
  await b.promise;
  expect(store().getLoader("other")).toBeUndefined();
  expect(store().getLoader("ws")).toBe("init-loader");
  a.call.answer(json(200, { data: [row], total: 9, next_cursor: "opaque+/=" }));
  await a.promise;
  const next = await requestedIssue(tab.nerve, () => store().fetchIssues("ws", true), ["GET", path]);
  expect(next.call.queryEntries).toEqual([["cursor", "opaque+/="]]);
  next.call.answer(json(200, { data: [draftOf("z")], total: 10, next_cursor: null }));
  await next.promise;
  expect(
    store()
      .getDrafts("ws")
      ?.map((item) => item.id)
  ).toEqual(["z", "d"]);
  expect(store().getTotal("ws")).toBe(10);
  expect(store().getTotal("other")).toBe(100);
});
it("a read refusal retains the last page and ends only its own loader", async () => {
  await loaded([row], "next", 90);
  const sent = await requestedIssue(tab.nerve, () => store().fetchIssues("ws", true), ["GET", path]);
  sent.call.answer(problem(500, "unavailable"));
  await expect(sent.promise).rejects.toMatchObject({ status: 500 });
  expect(store().getDrafts("ws")).toEqual([row]);
  expect(store().getTotal("ws")).toBe(90);
  expect(store().getLoader("ws")).toBeUndefined();
  expect(store().getError("ws")).toMatchObject({ status: 500 });
});
it.each([5, 15, 20] as const)("only workspace members/admins access personal drafts (%s)", async (role) => {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role }]);
  tab.nerve.calls.length = 0;
  if (role !== 5) {
    await loaded();
    expect(store().getDrafts("ws")).toEqual([row]);
    return;
  }
  const result = track(read());
  await until(() => result.settled || tab.nerve.calls.length > 0, "guest refusal");
  expect(tab.nerve.calls).toHaveLength(0);
  expect(result.error).toBeInstanceOf(SessionChangedError);
});
it.each(["permission", "retire"])("old drafts remain hidden after %s", async (boundary) => {
  await loaded();
  const sent = await requestedIssue(tab.nerve, read, ["GET", path]);
  if (boundary === "retire") tab.app.issue.retire();
  else {
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }]);
    expect(store().getDrafts("ws")).toBeUndefined();
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace]);
  }
  sent.call.answer(json(200, { data: [row], next_cursor: null, total: 1 }));
  await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(store().getDrafts("ws")).toBeUndefined();
});
it.each([403, 422])("save refusal %s leaves payload and project unchanged", async (status) => {
  await loaded();
  const sent = await requestedIssue(
    tab.nerve,
    () => store().updateIssue("ws", "d", { project_id: null, payload: { name: "" } }),
    ["PATCH", "/api/v0/drafts/d"]
  );
  expect(sent.call.body).toEqual({ project_id: null, payload: { name: "" } });
  expect(store().getIssueById("d")).toEqual(row);
  sent.call.answer(problem(status, "original.code"));
  await expect(sent.promise).rejects.toMatchObject({ status });
  expect(store().getIssueById("d")).toEqual(row);
  expect(store().getLoader("ws")).toBeUndefined();
});
it("confirmed save is replayed over an older list without flattening or merging replaced payload", async () => {
  await loaded();
  const old = await requestedIssue(tab.nerve, read, ["GET", path]);
  const sent = await requestedIssue(
    tab.nerve,
    () => store().updateIssue("ws", "d", { project_id: null, payload: changed.payload }),
    ["PATCH", "/api/v0/drafts/d"]
  );
  sent.call.answer(json(200, changed));
  expect(await sent.promise).toEqual(changed);
  old.call.answer(json(200, { data: [row], next_cursor: null, total: 8 }));
  await old.promise;
  expect(store().getIssueById("d")).toEqual(changed);
  expect(store().getTotal("ws")).toBe(8);
});
it("create preserves omitted versus cleared project and never invents a total before a list", async () => {
  for (const body of [{ payload: { name: "" } }, { project_id: null, payload: {} }]) {
    // Each creation must settle before the next command is exercised.
    // oxlint-disable-next-line no-await-in-loop
    const sent = await requestedIssue(tab.nerve, () => store().createIssue("ws", body), ["POST", path]);
    expect(sent.call.body).toEqual(body);
    sent.call.answer(json(201, { ...row, project_id: null, payload: body.payload }));
    // oxlint-disable-next-line no-await-in-loop
    await sent.promise;
    expect(store().getTotal("ws")).toBeUndefined();
  }
});
it("confirmed deletion cannot be undone by a late list; total comes from the following GET", async () => {
  await loaded([row], "more", 80);
  const old = await requestedIssue(tab.nerve, read, ["GET", path]);
  const sent = await requestedIssue(tab.nerve, () => store().deleteIssue("ws", "d"), ["DELETE", "/api/v0/drafts/d"]);
  expect(store().getIssueById("d")).toEqual(row);
  sent.call.answer(noContent());
  await until(() => tab.nerve.to(path).length === 3, "authoritative draft count");
  expect(store().getIssueById("d")).toBeUndefined();
  expect(store().getTotal("ws")).toBeUndefined();
  tab.nerve.to(path)[2]!.answer(json(200, { data: [], next_cursor: "more", total: 79 }));
  await sent.promise;
  old.call.answer(json(200, { data: [row], next_cursor: null, total: 80 }));
  await old.promise.catch(() => {});
  expect(store().getIssueById("d")).toBeUndefined();
  expect(store().getTotal("ws")).toBe(79);
});
it("delete refusal leaves the draft and an acknowledged delete is never retried after its count read fails", async () => {
  await loaded();
  let sent = await requestedIssue(tab.nerve, () => store().deleteIssue("ws", "d"), ["DELETE", "/api/v0/drafts/d"]);
  sent.call.answer(problem(403, "forbidden"));
  await expect(sent.promise).rejects.toMatchObject({ status: 403 });
  expect(store().getIssueById("d")).toEqual(row);
  sent = await requestedIssue(tab.nerve, () => store().deleteIssue("ws", "d"), ["DELETE", "/api/v0/drafts/d"]);
  sent.call.answer(noContent());
  await until(() => tab.nerve.to(path).length === 2, "delete follow-up");
  tab.nerve.to(path)[1]!.answer(problem(500, "unavailable"));
  await sent.promise;
  expect(store().getIssueById("d")).toBeUndefined();
  expect(store().getError("ws")).toMatchObject({ status: 500 });
  answerRefresh([]);
  await store().retry("ws");
  expect(tab.nerve.calls.filter((call) => call.method === "DELETE")).toHaveLength(2);
  expect(store().getError("ws")).toBeUndefined();
});
it.each([403, 409, 422])("patchesBeforePublishingWithoutABody; publish %s retains the saved draft", async (status) => {
  await loaded();
  const input: DraftUpdate = { project_id: null, payload: { name: "saved" } };
  const sent = await requestedIssue(tab.nerve, () => store().moveIssue("ws", "d", input), [
    "PATCH",
    "/api/v0/drafts/d",
  ]);
  expect(tab.nerve.to("/api/v0/drafts/d/publish")).toHaveLength(0);
  sent.call.answer(json(200, changed));
  await until(() => tab.nerve.to("/api/v0/drafts/d/publish").length === 1 || sent.result.settled, "publish after save");
  const publish = tab.nerve.to("/api/v0/drafts/d/publish")[0]!;
  expect(publish).toBeDefined();
  expect(publish.body).toBeUndefined();
  publish.answer(problem(status, status === 409 ? "draft_changed" : "forbidden"));
  await expect(sent.promise).rejects.toMatchObject({ status });
  expect(store().getIssueById("d")).toEqual(changed);
});
it("PATCH refusal prevents publish", async () => {
  await loaded();
  const sent = await requestedIssue(tab.nerve, () => store().moveIssue("ws", "d", { payload: {} }), [
    "PATCH",
    "/api/v0/drafts/d",
  ]);
  sent.call.answer(problem(422, "validation_failed"));
  await expect(sent.promise).rejects.toMatchObject({ status: 422 });
  expect(tab.nerve.to("/api/v0/drafts/d/publish")).toHaveLength(0);
});
it("a save waits for the whole PATCH/publish operation and uses the last confirmed payload", async () => {
  await loaded();
  const sent = await requestedIssue(tab.nerve, () => store().moveIssue("ws", "d", { payload: changed.payload }), [
    "PATCH",
    "/api/v0/drafts/d",
  ]);
  const save = store().updateIssue("ws", "d", (held) => ({ payload: { ...held.payload, description_html: "second" } }));
  const result = track(save);
  sent.call.answer(json(200, changed));
  await until(() => tab.nerve.to("/api/v0/drafts/d/publish").length === 1, "publish");
  expect(tab.nerve.to("/api/v0/drafts/d")).toHaveLength(1);
  tab.nerve.to("/api/v0/drafts/d/publish")[0]!.answer(problem(409, "draft_changed"));
  await expect(sent.promise).rejects.toMatchObject({ status: 409 });
  await until(() => tab.nerve.to("/api/v0/drafts/d").length === 2 || result.settled, "queued save");
  expect(tab.nerve.to("/api/v0/drafts/d")[1]!.body).toEqual({ payload: { name: "saved", description_html: "second" } });
  tab.nerve
    .to("/api/v0/drafts/d")[1]!
    .answer(json(200, { ...changed, payload: { name: "saved", description_html: "second" } }));
  await save;
});
it.each([false, true])(
  "publishing confirms canonical plus every held list once, and makes no project refresh for number width (read failure: %s)",
  async (readFails) => {
    await loaded();
    const root = tab.app.issue;
    const bindings = [
      new IssueListBinding(root, {
        scope: "project",
        workspaceSlug: "ws",
        workspaceID: "id-ws",
        projectID: "A",
        query: { order_by: "sort_order" },
      }),
      new IssueListBinding(root, {
        scope: "member",
        workspaceSlug: "ws",
        workspaceID: "id-ws",
        userID: "actor",
        query: { relation: "created", order_by: "sort_order" },
      }),
    ];
    for (const binding of bindings) {
      const endpoint =
        binding.scope.scope === "project" ? "/api/v0/projects/A/issues" : "/api/v0/workspaces/ws/members/actor/issues";
      // Independent bindings are established in fixed order to identify their held counter sources.
      // oxlint-disable-next-line no-await-in-loop
      const sent = await requestedIssue(tab.nerve, () => binding.fetchFirst(), ["GET", endpoint]);
      sent.call.answer(json(200, issuePageOf([])));
      // oxlint-disable-next-line no-await-in-loop
      await sent.promise;
    }
    const stale = [];
    for (const binding of bindings) {
      // requestedIssue identifies one arrival at a time; both reads remain outstanding until after publish.
      // oxlint-disable-next-line no-await-in-loop
      const late = await requestedIssue(tab.nerve, () => binding.fetchFirst(), [
        "GET",
        binding.scope.scope === "project" ? "/api/v0/projects/A/issues" : "/api/v0/workspaces/ws/members/actor/issues",
      ]);
      stale.push(late);
    }
    const issue = issueDetailOf("published");
    answerRefresh([]);
    if (readFails)
      tab.nerve.respond = (call) => {
        if (call.path === path) call.answer(problem(500, "unavailable"));
      };
    const sent = await requestedIssue(tab.nerve, () => store().moveIssue("ws", "d", { payload: row.payload }), [
      "PATCH",
      "/api/v0/drafts/d",
    ]);
    sent.call.answer(json(200, row));
    await until(() => tab.nerve.to("/api/v0/drafts/d/publish").length === 1 || sent.result.settled, "publish");
    tab.nerve.to("/api/v0/drafts/d/publish")[0]!.answer(json(201, issue));
    await sent.promise;
    expect(root.issues.getIssueDetailById(issue.id)).toEqual(issue);
    expect(store().getIssueById("d")).toBeUndefined();
    for (const late of stale) late.call.answer(json(200, issuePageOf([])));
    await until(
      () => tab.nerve.to("/api/v0/issues/published").length === 1,
      "one authoritative verification after stale lists"
    );
    tab.nerve.to("/api/v0/issues/published")[0]!.answer(json(200, issue));
    await Promise.all(stale.map((late) => late.promise));
    expect(tab.nerve.to("/api/v0/issues/published")).toHaveLength(1);
    if (readFails) {
      expect(store().getError("ws")).toMatchObject({ status: 500 });
      answerRefresh([]);
      await store().retry("ws");
      expect(store().getError("ws")).toBeUndefined();
    }
    expect(tab.nerve.to("/api/v0/drafts/d/publish")).toHaveLength(1);
    expect(tab.nerve.to("/api/v0/drafts/d")).toHaveLength(1);
    for (const binding of bindings) {
      expect(binding.group()?.rows.map((item) => item.id)).toEqual([issue.id]);
      expect(binding.value?.total).toBe(1);
    }
    expect(
      tab.nerve.calls.filter((call) => /^\/api\/v0\/(workspaces\/ws\/projects|projects\/A)$/.test(call.path))
    ).toHaveLength(0);
  }
);

it("the first publish request is a PATCH and no POST is sent before its confirmation", async () => {
  await loaded();
  tab.nerve.calls.length = 0;
  const sent = track(store().moveIssue("ws", "d", { payload: { name: "latest" } }));
  await until(() => tab.nerve.calls.length > 0 || sent.settled, "first publish step");
  expect(tab.nerve.calls[0]).toMatchObject({ method: "PATCH", path: "/api/v0/drafts/d" });
  expect(tab.nerve.to("/api/v0/drafts/d/publish")).toHaveLength(0);
  tab.nerve.calls[0]!.answer(problem(422, "validation_failed"));
  await until(() => sent.settled, "PATCH refusal");
  expect(sent.error).toMatchObject({ status: 422 });
  expect(tab.nerve.to("/api/v0/drafts/d/publish")).toHaveLength(0);
});
