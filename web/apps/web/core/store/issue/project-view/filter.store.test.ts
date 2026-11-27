/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A view's page's preferences are its draft of the view (P6a spec 2.5, 2.6, R2; re-96 I2): the page-preferences base
// reads them from the draft, which one read of the view seeds (a GET of the view, its answer the view store's too);
// the page's changes stay in the draft, never sent; the page's leaving ends the draft and the page's filters, and a
// seed then out makes neither. The stores and HTTP are real; the test holds its own draft service, to adopt nerve's
// answer as Update view does.
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { ProjectIssueFilters, View } from "@nerve/api-client";
import { json } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { viewOf } from "@/store/views/fake-views";
import { issueStoreTab, requestedIssue } from "../fake-issues";
import { ProjectViewIssuesFilter } from "./filter.store";
import { ViewDraftService } from "../helpers/view-draft.service";

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

const urgent = { and: [{ priority__in: "urgent" }] };
const high = { and: [{ priority__in: "high" }] };
/** The view of project A as nerve holds it: urgent work items. */
const saved = (): View => viewOf(tab.project, "Urgent", "u-me", { filters: { priority: ["urgent"] } });
/** nerve's answer to Update view: high work items on a board grouped by priority. */
const answer = (view: View): View => ({
  ...view,
  filters: { priority: ["high"] },
  display_filters: { ...view.display_filters, layout: "kanban", group_by: "priority" },
});
const readOf = (view: View): [string, string] => ["GET", `/api/v0/views/${view.id}`];
/** The view page's preferences over a draft service of the test's own. */
function page() {
  const drafts = new ViewDraftService<ProjectIssueFilters>(tab.app.views.fetchView, (filters) => filters);
  return { drafts, filter: new ProjectViewIssuesFilter(tab.app.issue, drafts) };
}
/** The page reads its preferences: the view is read and nerve answers it with view. */
async function seeded(filter: ProjectViewIssuesFilter, view: View) {
  const read = await requestedIssue(tab.nerve, () => filter.fetchFilters("ws", view.id), readOf(view));
  read.call.answer(json(200, view));
  await read.promise;
}

it("seeds the page's filters from one read of the view, the view store's too; with the draft held, reads nothing", async () => {
  const { drafts, filter } = page();
  const view = saved();
  await seeded(filter, view);
  expect(filter.getIssueFilters(view.id)?.richFilters).toEqual(urgent);
  expect(tab.app.views.getViewById(view.id)).toEqual(view);
  // "Update view" confirmed: the draft is nerve's answer, which the page's filters are read again from
  drafts.adopt(view.id, answer(view));
  tab.nerve.calls.length = 0;
  const fetched = track(filter.fetchFilters("ws", view.id));
  await until(() => fetched.settled || tab.nerve.calls.length > 0, "the page's filters");
  expect(tab.nerve.calls, "the draft held, the view is not read").toEqual([]);
  expect(filter.getIssueFilters(view.id)).toMatchObject({
    richFilters: high,
    displayFilters: { layout: "kanban", group_by: "priority" },
  });
});

it("keeps the page's changes in its draft, sent nowhere", async () => {
  const { filter } = page();
  const view = saved();
  await seeded(filter, view);
  tab.nerve.calls.length = 0;
  await filter.updateFilterExpression("ws", view.id, high);
  expect(filter.getIssueFilters(view.id)?.richFilters).toEqual(high);
  expect(filter.getIssueFilters(view.id)?.displayFilters?.layout).toBe("list");
  expect(tab.nerve.calls).toEqual([]);
});

it("ends the page's filters with its draft: entering again shows nothing of the old draft until the read answers", async () => {
  const { filter } = page();
  const view = saved();
  await seeded(filter, view);
  await filter.updateFilterExpression("ws", view.id, high);
  filter.discard(view.id);
  expect(filter.getIssueFilters(view.id), "the page left").toBeUndefined();
  const read = await requestedIssue(tab.nerve, () => filter.fetchFilters("ws", view.id), readOf(view));
  expect(filter.getIssueFilters(view.id), "while the view is read again").toBeUndefined();
  read.call.answer(json(200, view));
  await read.promise;
  expect(filter.getIssueFilters(view.id)?.richFilters).toEqual(urgent);
});

it("takes a seed answered after the page left as the view store's alone: no draft, no filters; entering again reads once", async () => {
  const { filter } = page();
  const view = saved();
  const late = await requestedIssue(tab.nerve, () => filter.fetchFilters("ws", view.id), readOf(view));
  filter.discard(view.id);
  late.call.answer(json(200, view));
  await expect(late.promise).rejects.toThrow(view.id);
  expect(filter.getIssueFilters(view.id)).toBeUndefined();
  expect(tab.app.views.getViewById(view.id), "the answer is the view store's").toEqual(view);
  const next = { ...view, filters: { priority: ["high"] } };
  const read = await requestedIssue(tab.nerve, () => filter.fetchFilters("ws", view.id), readOf(view));
  read.call.answer(json(200, next));
  await read.promise;
  expect(filter.getIssueFilters(view.id)?.richFilters).toEqual(high);
  expect(tab.nerve.to(`/api/v0/views/${view.id}`)).toHaveLength(2);
});

it("takes no answer to Update view once the page left: entering again reads the view, once", async () => {
  const { drafts, filter } = page();
  const view = saved();
  await seeded(filter, view);
  filter.discard(view.id);
  drafts.adopt(view.id, answer(view));
  await seeded(filter, view);
  expect(filter.getIssueFilters(view.id)?.richFilters).toEqual(urgent);
  expect(tab.nerve.to(`/api/v0/views/${view.id}`)).toHaveLength(2);
});
