/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A workspace's pages' preferences (M7/P6b spec 2.4, R2, R3, R5): one filter store, by page id, over two services. A
// default view's page reads nothing of nerve: its display is the browser's, by account and page, its filters this
// entry's, gone when the page leaves. A saved view's page is its draft of the view, as a project's view's page's is
// (P6a), seeded by one read of the view and never sent; the draft holds the view's words, the contract's 11, so that a
// change of the page keeps every one of them, its projects among them. The stores and HTTP are real.
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { EIssueFilterType } from "@nerve/constants";
import { json } from "@/lib/auth/fake-nerve";
import { toFilterExpression } from "@/lib/issue-filters";
import { track, until } from "@/lib/auth/fake-time";
import { workspaceViewOf } from "@/store/views/fake-views";
import { issueStoreTab, requestedIssue } from "../fake-issues";
import { workspacePageId } from "./page-id";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
  vi.useFakeTimers();
  tab = await issueStoreTab();
  binding.app = tab.app;
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

const A = "019a0000-0000-7000-8000-000000000001";
const B = "019a0000-0000-7000-8000-000000000002";
const S = "019a0000-0000-7000-8000-0000000000a1";
const filter = () => tab.app.issue.globalIssuesFilter;
const ASSIGNED = workspacePageId("ws", "assigned");
const high = { and: [{ priority__in: "high" }] };
/** Every word the contract gives a workspace's view, as a view made through the API holds them. */
const eleven = {
  state_id: [S],
  state_group: ["started"],
  priority: ["urgent"],
  label_id: [A],
  assignee_id: ["me"],
  created_by_id: [B],
  mention_id: ["me"],
  subscriber_id: [B],
  start_date: ["2026-10-01.."],
  target_date: ["none"],
  project_id: [A, B],
};
const saved = (fields: Partial<View> = {}): View => workspaceViewOf(tab.workspace, "Mine", "actor", fields);
async function seeded(view: View) {
  const read = await requestedIssue(tab.nerve, () => filter().fetchFilters("ws", view.id), [
    "GET",
    `/api/v0/views/${view.id}`,
  ]);
  read.call.answer(json(200, view));
  await read.promise;
}

it("reads a default view's page from the browser and memory alone: no filters, the spreadsheet's display", async () => {
  await filter().fetchFilters("ws", ASSIGNED);
  expect(tab.nerve.calls).toEqual([]);
  expect(filter().getIssueFilters(ASSIGNED)).toMatchObject({
    richFilters: {},
    displayFilters: { layout: "spreadsheet", group_by: null, sub_group_by: null, order_by: "-created_at" },
    displayProperties: { key: true, cycle: true, modules: true },
  });
});

it("keeps a default view's filters for one entry, and its display after it", async () => {
  await filter().fetchFilters("ws", ASSIGNED);
  await filter().updateFilterExpression("ws", ASSIGNED, high);
  await filter().updateFilters("ws", ASSIGNED, EIssueFilterType.DISPLAY_FILTERS, { order_by: "priority" });
  expect(filter().getIssueFilters(ASSIGNED)?.richFilters).toEqual(high);
  filter().discard(ASSIGNED);
  expect(filter().getIssueFilters(ASSIGNED), "the page left").toBeUndefined();
  await filter().fetchFilters("ws", ASSIGNED);
  expect(filter().getIssueFilters(ASSIGNED)?.richFilters, "the next entry's").toEqual({});
  expect(filter().getIssueFilters(ASSIGNED)?.displayFilters?.order_by).toBe("priority");
  expect(tab.nerve.calls).toEqual([]);
});

it("keeps the same default view's display apart in two workspaces", async () => {
  const other = workspacePageId("beta", "assigned");
  await filter().fetchFilters("ws", ASSIGNED);
  await filter().fetchFilters("beta", other);
  await filter().updateFilters("ws", ASSIGNED, EIssueFilterType.DISPLAY_FILTERS, { order_by: "priority" });
  expect(filter().getIssueFilters(ASSIGNED)?.displayFilters?.order_by).toBe("priority");
  expect(filter().getIssueFilters(other)?.displayFilters?.order_by).toBe("-created_at");
});

it("seeds a saved view's page from one read of the view, its projects among the filters shown", async () => {
  const view = saved({ filters: { priority: ["urgent"], project_id: [A] } });
  await seeded(view);
  expect(filter().getIssueFilters(view.id)?.richFilters).toEqual({
    and: [{ priority__in: "urgent" }, { project_id__in: A }],
  });
});

it("keeps every word of a view made through the API when its page changes a filter, sending nothing", async () => {
  const view = saved({ filters: eleven });
  await seeded(view);
  tab.nerve.calls.length = 0;
  expect(filter().getIssueFilters(view.id)?.richFilters, "shown").toEqual(toFilterExpression(eleven));
  // the controls give the page its whole expression back, the priority changed
  await filter().updateFilterExpression("ws", view.id, toFilterExpression({ ...eleven, priority: ["high"] }));
  expect(filter().confirmed(view.id)?.filters).toEqual({ ...eleven, priority: ["high"] });
  expect(tab.nerve.calls).toEqual([]);
});

it("holds no cycle or module of a saved view, which a workspace's view never has", async () => {
  const view = saved({ filters: { priority: ["urgent"], cycle_id: [A], module_id: [B] } });
  await seeded(view);
  expect(filter().confirmed(view.id)?.filters).toEqual({ priority: ["urgent"] });
});

it("takes nerve's answer to Update view as a saved view's draft, and ends it as the page leaves", async () => {
  const view = saved({ filters: { priority: ["urgent"] } });
  await seeded(view);
  filter().adopt(view.id, { ...view, filters: { priority: ["high"] } });
  tab.nerve.calls.length = 0;
  const fetched = track(filter().fetchFilters("ws", view.id));
  await until(() => fetched.settled || tab.nerve.calls.length > 0, "the page's filters");
  expect(tab.nerve.calls, "the draft held, the view is not read").toEqual([]);
  expect(filter().getIssueFilters(view.id)?.richFilters).toEqual(high);
  filter().discard(view.id);
  expect(filter().getIssueFilters(view.id)).toBeUndefined();
  await seeded(view);
  expect(filter().getIssueFilters(view.id)?.richFilters).toEqual({ and: [{ priority__in: "urgent" }] });
});
