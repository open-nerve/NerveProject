/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A list page's preferences as nerve last confirmed them (P6a spec 2.7, R8): what Save as and Update view keep of the
// page's display, in nerve's words, not the controls' (group_by "state_id", order_by "state_name"); a copy, which its
// reader may change without changing what the page holds. The stores and HTTP are real.
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssueFilterType } from "@nerve/constants";
import { json } from "@/lib/auth/fake-nerve";
import { preferencesOf } from "@/store/project/fake-projects";
import { issueStoreTab, requestedIssue } from "../fake-issues";

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

const path = "/api/v0/me/projects/A/preferences";
/** The project page's preferences as nerve gives them: grouped by state, ordered by its name. */
const read = preferencesOf({
  filters: { priority: ["high"] },
  display_filters: { ...preferencesOf().display_filters, group_by: "state_id", order_by: "state_name" },
});

it("gives nothing of a page nerve has not given yet, then a copy of what nerve confirmed, in nerve's words", async () => {
  const store = tab.app.issue.projectIssuesFilter;
  expect(store.confirmed("A")).toBeUndefined();
  const loading = await requestedIssue(tab.nerve, () => store.fetchFilters("ws", "A"), ["GET", path]);
  loading.call.answer(json(200, read));
  await loading.promise;
  expect(store.confirmed("A")).toEqual(read);
  // the controls' words are the page's filters', not the confirmed preferences'
  expect(store.getIssueFilters("A")?.displayFilters).toMatchObject({ group_by: "state", order_by: "state__name" });
  // a change of the copy is the reader's alone
  const copy = store.confirmed("A")!;
  copy.display_filters.layout = "kanban";
  copy.display_properties.assignee = false;
  copy.filters.priority!.push("low");
  expect(store.confirmed("A")).toEqual(read);
  // a change nerve confirms is what the page then holds
  const display_filters = { ...read.display_filters, layout: "spreadsheet" };
  const patch = await requestedIssue(
    tab.nerve,
    () => store.updateFilters("ws", "A", EIssueFilterType.DISPLAY_FILTERS, { layout: "spreadsheet" }),
    ["PATCH", path]
  );
  patch.call.answer(json(200, { ...read, display_filters }));
  await patch.promise;
  expect(store.confirmed("A")).toEqual({ ...read, display_filters });
});
