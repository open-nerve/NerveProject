/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueActivity } from "@nerve/api-client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { E_SORT_ORDER } from "@nerve/constants";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { loadProjects } from "@/store/project/fake-projects";
import { issueActivityOf, issueCommentOf, issueStoreTab, requestedIssue } from "../fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const store = () => tab.app.issue.issueDetail.activity;
const route = "/api/v0/issues/x/activities";
const read = (incremental = false) => store().fetchActivities("ws", "A", "x", incremental);
const row = issueActivityOf("z");
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
async function loaded(data: IssueActivity[] = [row]) {
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  sent.call.answer(json(200, { data, next_cursor: null }));
  await sent.promise;
}
it.each([1, 2])("reloadsTheLastRequestedPage (%s pages)", async (pages) => {
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  sent.call.answer(json(200, { data: [row], next_cursor: pages === 2 ? "last-request" : null }));
  if (pages === 2) {
    await until(() => tab.nerve.to(route).length === 2 || sent.result.settled, "last activity page");
    expect(tab.nerve.to(route)).toHaveLength(2);
    expect(store().getActivitiesByIssueId("x")).toBeUndefined();
    tab.nerve.to(route)[1]!.answer(json(200, { data: [issueActivityOf("m")], next_cursor: null }));
  }
  await sent.promise;
  const again = await requestedIssue(tab.nerve, () => read(true), ["GET", route]);
  expect(again.call.queryEntries).toEqual(pages === 2 ? [["cursor", "last-request"]] : []);
  again.call.answer(json(200, { data: [issueActivityOf("a")], next_cursor: "new-last" }));
  await until(() => tab.nerve.to(route).length === pages + 2 || again.result.settled, "new last page");
  expect(tab.nerve.to(route)).toHaveLength(pages + 2);
  tab.nerve
    .to(route)
    .at(-1)!
    .answer(json(200, { data: [], next_cursor: null }));
  await again.promise;
  expect(store().getActivitiesByIssueId("x")).toEqual(pages === 2 ? ["a", "m", "z"] : ["a", "z"]);
  const third = await requestedIssue(tab.nerve, () => read(true), ["GET", route]);
  expect(third.call.queryEntries).toEqual([["cursor", "new-last"]]);
  third.call.answer(json(200, { data: [], next_cursor: null }));
  await third.promise;
});
it("repositionsMergedDescriptionActivity", async () => {
  const first = { ...row, id: "old-description", field: "description", created_at: "2026-10-30T00:00:00.000000001Z" };
  const second = { ...row, id: "middle", created_at: "2026-10-30T00:00:00.000000002Z" };
  await loaded([first, second]);
  const newer = { ...first, new_value: "latest HTML", created_at: "2026-10-30T00:00:00.000000009Z" };
  const sent = await requestedIssue(tab.nerve, () => read(true), ["GET", route]);
  sent.call.answer(json(200, { data: [newer], next_cursor: null }));
  await sent.promise;
  expect(store().getActivitiesByIssueId("x")).toEqual(["middle", "old-description"]);
  expect(store().getActivityById("x", first.id)).toEqual(newer);
});
it("reordersMergedActivityAcrossPages without losing nanoseconds or colliding comment ids", async () => {
  const at = (id: string, nanosecond: number): IssueActivity => ({
    ...row,
    id,
    created_at: `2026-10-30T00:00:00.00000000${nanosecond}Z`,
  });
  const description = { ...at("description", 2), field: "description", new_value: "First" };
  const pages = [
    [at("z", 1), description],
    [at("collision", 3), at("b", 4)],
    [at("a", 5), at("tail", 6)],
  ];
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  await pages.reduce(async (previous, data, index) => {
    await previous;
    await until(() => tab.nerve.to(route).length === index + 1 || sent.result.settled, "next activity page");
    expect(tab.nerve.to(route)).toHaveLength(index + 1);
    expect(store().getActivitiesByIssueId("x")).toBeUndefined();
    const call = tab.nerve.to(route)[index]!;
    expect(call.queryEntries).toEqual(index === 0 ? [] : [["cursor", `page-${index + 1}`]]);
    call.answer(json(200, { data, next_cursor: index === 2 ? null : `page-${index + 2}` }));
  }, Promise.resolve());
  await sent.promise;
  const comments = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.issueDetail.comment.fetchComments("ws", "A", "x"),
    ["GET", "/api/v0/issues/x/comments"]
  );
  comments.call.answer(
    json(200, {
      data: [
        { ...issueCommentOf("collision"), created_at: "2026-10-30T00:00:00.000000003Z" },
        { ...issueCommentOf("between"), created_at: "2026-10-30T00:00:00.000000007Z" },
      ],
      next_cursor: null,
    })
  );
  await comments.promise;
  const identities = () =>
    store()
      .getActivityAndCommentsByIssueId("x", E_SORT_ORDER.ASC)!
      .map((entry) => `${entry.kind}:${entry.id}`);
  expect(identities()).toEqual([
    "activity:z",
    "activity:description",
    "activity:collision",
    "comment:collision",
    "activity:b",
    "activity:a",
    "activity:tail",
    "comment:between",
  ]);
  const again = await requestedIssue(tab.nerve, () => read(true), ["GET", route]);
  expect(again.call.queryEntries).toEqual([["cursor", "page-3"]]);
  again.call.answer(json(200, { data: pages[2], next_cursor: "new-last" }));
  await until(() => tab.nerve.to(route).length === 5 || again.result.settled, "incremental activity tail");
  expect(tab.nerve.to(route)).toHaveLength(5);
  const moved = { ...description, created_at: "2026-10-30T00:00:00.000000009Z", new_value: "Final" };
  tab.nerve.to(route)[4]!.answer(json(200, { data: [at("after", 8), moved], next_cursor: null }));
  await again.promise;
  expect(identities()).toEqual([
    "activity:z",
    "activity:collision",
    "comment:collision",
    "activity:b",
    "activity:a",
    "activity:tail",
    "comment:between",
    "activity:after",
    "activity:description",
  ]);
  expect(store().getActivityById("x", "description")).toEqual(moved);
});
it.each([false, true])("keepsTheLastCompleteValueAfterPageTwoFails (previous=%s)", async (previous) => {
  if (previous) await loaded();
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  sent.call.answer(json(200, { data: [issueActivityOf("partial")], next_cursor: "second" }));
  const count = previous ? 3 : 2;
  await until(() => tab.nerve.to(route).length === count || sent.result.settled, "second page");
  expect(tab.nerve.to(route)).toHaveLength(count);
  tab.nerve.to(route).at(-1)!.answer(problem(500, "unavailable"));
  await expect(sent.promise).rejects.toMatchObject({ status: 500 });
  expect(store().getError("x")).toMatchObject({ status: 500 });
  expect(store().getActivitiesByIssueId("x")).toEqual(previous ? [row.id] : undefined);
  const again = await requestedIssue(tab.nerve, () => read(true), ["GET", route]);
  expect(again.call.queryEntries).toEqual([]);
  again.call.answer(json(200, { data: [], next_cursor: null }));
  await again.promise;
});
it.each(["retire", "permission"])("doesNotContinuePagingAfterRetirement (%s)", async (boundary) => {
  await loaded();
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  if (boundary === "retire") tab.app.issue.retire();
  else {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  }
  sent.call.answer(json(200, { data: [row], next_cursor: "forbidden-next" }));
  await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(tab.nerve.to(route)).toHaveLength(2);
  expect(store().getActivitiesByIssueId("x")).toBeUndefined();
});
it.each(["issue", "project"])("refuses an activity outside its %s", async (field) => {
  const sent = await requestedIssue(tab.nerve, read, ["GET", route]);
  const wrong = { ...row, ...(field === "issue" ? { issue_id: "other" } : { project_id: "other" }) };
  sent.call.answer(json(200, { data: [wrong], next_cursor: null }));
  await expect(sent.promise).rejects.toBeInstanceOf(SessionChangedError);
  expect(store().getActivitiesByIssueId("x")).toBeUndefined();
});
it("ordersNanosecondsBeforeIds and distinguishes comments from activities with the same id", async () => {
  const later = { ...row, id: "a", created_at: "2026-10-30T00:00:00.000000009Z" };
  const earlier = { ...row, id: "z", created_at: "2026-10-30T00:00:00.000000001Z" };
  await loaded([later, earlier]);
  const sent = await requestedIssue(tab.nerve, () => tab.app.issue.issueDetail.comment.fetchComments("ws", "A", "x"), [
    "GET",
    "/api/v0/issues/x/comments",
  ]);
  sent.call.answer(json(200, { data: [{ ...issueCommentOf("a"), created_at: later.created_at }], next_cursor: null }));
  await sent.promise;
  const rows = store().getActivityAndCommentsByIssueId("x", E_SORT_ORDER.ASC)!;
  expect(rows.map((entry) => [entry.kind, entry.id])).toEqual([
    ["activity", "z"],
    ["activity", "a"],
    ["comment", "a"],
  ]);
  expect(new Set(rows.map((entry) => `${entry.kind}:${entry.id}`)).size).toBe(3);
  expect(store().getActivityAndCommentsByIssueId("x", E_SORT_ORDER.DESC)).toEqual(rows.toReversed());
});
it("a complete re-open replaces old rows, and scoped lookup never leaks to a different issue", async () => {
  await loaded();
  await loaded([issueActivityOf("new")]);
  expect(store().getActivitiesByIssueId("x")).toEqual(["new"]);
  expect(store().getActivityById("other", "new")).toBeUndefined();
});
